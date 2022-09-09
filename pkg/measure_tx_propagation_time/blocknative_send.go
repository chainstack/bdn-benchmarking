package measuretxpropagationtime

import (
	"crypto/ecdsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"performance/pkg/cmpnodestxspeedhttp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type blocknativeRequest struct {
	TimeStamp  time.Time `json:"timeStamp"`
	DappID     string    `json:"dappId"`
	Version    string    `json:"version"`
	Blockchain struct {
		System  string `json:"system"`
		Network string `json:"network"`
	} `json:"blockchain"`
	CategoryCode      string `json:"categoryCode"`
	EventCode         string `json:"eventCode"`
	SignedTransaction string `json:"signedTransaction,omitempty"`
}

type blocknativeEventResponse struct {
	Version       int64              `json:"version"`
	ServerVersion string             `json:"serverVersion"`
	TimeStamp     time.Time          `json:"timeStamp"`
	ConnectionId  string             `json:"connectionId"`
	Status        string             `json:"status"`
	Raw           string             `json:"raw"`
	Event         blocknativeRequest `json:"event"`
	Reason        string             `json:"reason"`
	Hash          string             `json:"hash,omitempty"`
}

func (s *MeasureTxPropagationTimeService) sendTxBlocknative(nodeEndpoint, address string, gasLimit, gasPriceWei, chainID int64, secretKey *ecdsa.PrivateKey, apiAddress, apiKey, networkName string) (string, time.Time, time.Time, error) {
	var (
		addr   = common.HexToAddress(address)
		value  = big.NewInt(0)
		limit  = uint64(gasLimit)
		price  = big.NewInt(gasPriceWei)
		signer = types.NewEIP155Signer(big.NewInt(chainID))
	)

	// From Blocknative documentation:
	//blockchain network, valid values for support systems are:
	//Ethereum (and EVM compatible) - main, ropsten, rinkeby, goerli, kovan, xdai, bsc-main, matic-main, fantom-main
	//Bitcoin - main, test
	if networkName == "Mainnet" {
		networkName = "main"
	} else {
		networkName = strings.ToLower(networkName)
	}
	log.Debugf("Use network '%s'\n", networkName)

	nonce, err := cmpnodestxspeedhttp.GetNonce(address, nodeEndpoint)
	if err != nil {
		return "", time.Now(), time.Now(), err
	}
	tx := types.NewTx(&types.LegacyTx{
		To:       &addr,
		Value:    value,
		Gas:      limit,
		GasPrice: price,
		Nonce:    nonce,
		Data:     []byte("0x11111111"),
	})
	evmSignedTx, err := types.SignTx(tx, signer, secretKey)
	if err != nil {
		return "", time.Now(), time.Now(), err
	}

	s.txHashToFind = evmSignedTx.Hash().Hex()
	fmt.Printf("hash to find %s\n", evmSignedTx.Hash().Hex())

	rawTx, err := cmpnodestxspeedhttp.EncodeSignedTx(evmSignedTx)
	if err != nil {
		return "", time.Now(), time.Now(), err
	}

	// websocket connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = tlsConfig
	wsconn, _, err := dialer.Dial(apiAddress, http.Header{"Authorization": []string{apiKey}})
	if err != nil {
		log.Debugf("Dial error: %s\n", err.Error())
		return "", time.Now(), time.Now(), err
	}
	defer wsconn.Close()

	// Init request
	initReq := NewBlocknativeInitializationRequest(apiKey, networkName)

	resp, status, err := BlocknativeSendEvent(wsconn, *initReq, initReq.EventCode)

	log.Debugf("Resp: %v, Status: %s Error: %s\n", resp, status, err)

	// Transaction request
	txReq := NewBlocknativeTransactionRequest(apiKey, networkName, rawTx)
	if err != nil {
		return "", time.Now(), time.Now(), err
	}
	txReqJson, err := json.Marshal(txReq)
	if err != nil {
		log.Debugf("Can't marshal json: %v\n", err)
	}
	log.Debugf("Send TX request: %s\n", string(txReqJson))

	timeStartTXreq := time.Now()

	resp, status, err = BlocknativeSendEvent(wsconn, *txReq, "tdnSubmitResponse")

	timeEndTxReq := time.Now()

	log.Debugf("TX response: %v, Status: %s, error: %v, request time:%s\n", resp, status, err, timeEndTxReq.Sub(timeStartTXreq))

	return evmSignedTx.Hash().Hex(), timeStartTXreq, timeEndTxReq, err

}

func NewBlocknativeInitializationRequest(apiKey, network string) *blocknativeRequest {
	return &blocknativeRequest{
		TimeStamp: time.Now().UTC(),
		DappID:    apiKey,
		Version:   "1",
		Blockchain: struct {
			System  string "json:\"system\""
			Network string "json:\"network\""
		}{System: "ethereum", Network: network},
		CategoryCode: "initialize",
		EventCode:    "checkDappId",
	}
}

func NewBlocknativeTransactionRequest(apiKey, network, signedTransaction string) *blocknativeRequest {
	return &blocknativeRequest{
		TimeStamp: time.Now().UTC(),
		DappID:    apiKey,
		Version:   "1",
		Blockchain: struct {
			System  string "json:\"system\""
			Network string "json:\"network\""
		}{System: "ethereum", Network: network},
		CategoryCode:      "tdn",
		EventCode:         "txSubmit",
		SignedTransaction: signedTransaction,
	}
}

func BlocknativeSendEvent(conn *websocket.Conn, event blocknativeRequest, expectResponseEventCode string) (blocknativeEventResponse, string, error) {
	var resp blocknativeEventResponse
	reqB, err := json.Marshal(event)
	if err != nil {
		return resp, "", err
	}
	log.Debugf("Request: %s\n", string(reqB))

	err = conn.WriteMessage(websocket.TextMessage, reqB)
	if err != nil {
		return resp, "", err
	}

	for {
		conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		_, respB, err := conn.ReadMessage()
		if err != nil {
			break
		}
		log.Debugf("Response: %s\n", string(respB))
		err = json.Unmarshal(respB, &resp)
		if err != nil {
			log.Debugf("This is not event response %s\n", err.Error())
			continue
		}

		if resp.Event.EventCode == expectResponseEventCode {
			log.Debugf("Event response recieved\n")
			break
		} else {
			log.Debug("Response code wrong: ", resp.Event.CategoryCode, event.CategoryCode, resp.Event.EventCode, event.EventCode)
		}
	}
	return resp, resp.Status, nil

}
