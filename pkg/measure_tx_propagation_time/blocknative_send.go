package measuretxpropagationtime

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"performance/internal/pkg/ws"
	"performance/pkg/cmpnodestxspeedhttp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

type blocknativeEvantResponse struct {
	Version       string             `json:"version"`
	ServerVersion string             `json:"serverVersion"`
	TimeStamp     time.Time          `json:"timeStamp"`
	ConnectionId  string             `json:"connectionId"`
	Status        string             `json:"status"`
	Raw           string             `json:"raw"`
	Event         blocknativeRequest `json:"event"`
	Reason        string             `json:"reason"`
	Hash          string             `json:"hash"`
}

func (s *MeasureTxPropagationTimeService) sendTxBlocknative(nodeEndpoint, address string, gasLimit, gasPriceWei, chainID int64, secretKey *ecdsa.PrivateKey, apiAddress, apiKey, networkName string) (string, error) {
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
		return "", err
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
		return "", err
	}

	s.txHashToFind = evmSignedTx.Hash().Hex()
	fmt.Printf("hash to find %s\n", evmSignedTx.Hash().Hex())

	rawTx, err := cmpnodestxspeedhttp.EncodeSignedTxWithout0xPrefix(evmSignedTx)
	if err != nil {
		return "", err
	}

	// websocket connection
	wsconn, err := ws.NewConnection(apiAddress, apiKey)
	if err != nil {
		return "", err
	}
	defer wsconn.Close()

	// Init request
	initReq, err := json.Marshal(NewBlocknativeInitializationRequest(apiKey, networkName))
	if err != nil {
		return "", err
	}
	resp, err := wsconn.CallCastomRequest(initReq)
	log.Debug("Init response: %s, error: %v\n", string(resp), err)

	// Transaction request
	txReq, err := json.Marshal(NewBlocknativeTransactionRequest(apiKey, networkName, rawTx))
	if err != nil {
		return "", err
	}

	resp, err = wsconn.CallCastomRequest(txReq)
	log.Debug("TX response: %s, error: %v\n", string(resp), err)

	return evmSignedTx.Hash().Hex(), err

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
