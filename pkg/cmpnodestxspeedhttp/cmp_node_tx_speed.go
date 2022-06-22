package cmpnodestxspeedhttp

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"performance/internal/pkg/flags"
	"performance/internal/pkg/utils"
	"performance/internal/pkg/ws"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// TxSpeedCompareService represents a service which compares transaction sending speed time
// between Evm nodes.
type TxSpeedCompareService struct{}

// Run is an entry point to the TxSpeedCompareService.
func (s *TxSpeedCompareService) Run(c *cli.Context) error {
	var (
		gasLimit          = int64(22000)
		senderPrivateKey  = c.String(flags.SenderPrivateKey.Name)
		numTxGroups       = c.Int(flags.NumTxGroups.Name)
		gasPriceWei       = c.Int64(flags.GasPrice.Name) * params.GWei
		chainID           = c.Int(flags.ChainID.Name)
		delay             = c.Int(flags.Delay.Name)
		nodeEndpoint      = c.String(flags.NodeEndpoint.Name)
		secondNodeEnpoint = c.String(flags.SecondNodeEndpoint.Name)
	)

	secretKey, err := MakePrivateKey(senderPrivateKey)
	if err != nil {
		return err
	}

	address, err := GetSenderAddress(secretKey)
	if err != nil {
		return err
	}

	nonce, err := GetNonce(address, nodeEndpoint)
	if err != nil {
		return err
	}

	balance, err := GetBalance(address, nodeEndpoint)
	if err != nil {
		return err
	}

	if expense := int64(numTxGroups) * gasPriceWei * gasLimit; balance < uint64(expense) {
		var (
			requiredEvm = float64(expense) / params.Ether
			currentEvm  = float64(balance) / params.Ether
		)

		fmt.Printf("Sender %s does not have enough balance for %d groups of transactions.\n"+
			"Sender's balance is %f Coins,\n"+
			"while at least %f Coins is required\n",
			address,
			numTxGroups,
			currentEvm,
			requiredEvm)

		return nil
	}

	fmt.Printf("Initial check completed. Sleeping %d sec.\n", delay)
	time.Sleep(time.Duration(delay) * time.Second)

	var (
		addr         = common.HexToAddress(address)
		value        = big.NewInt(0)
		limit        = uint64(gasLimit)
		price        = big.NewInt(gasPriceWei)
		signer       = types.NewEIP155Signer(big.NewInt(int64(chainID)))
		groupNumToTx = make(map[int]map[string]string)
	)

	for i := 1; i <= numTxGroups; i++ {
		endpointToTx := make(map[string]string)

		fmt.Printf("Sending tx group %d\n", i)

		// Node 1 transaction
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
			return err
		}

		evmEncodedTx, err := EncodeSignedTx(evmSignedTx)
		if err != nil {
			return err
		}

		endpointToTx[nodeEndpoint] = evmSignedTx.Hash().Hex()

		// Node 2 transaction
		tx = types.NewTx(&types.LegacyTx{
			To:       &addr,
			Value:    value,
			Gas:      limit,
			GasPrice: price,
			Nonce:    nonce,
			Data:     []byte("0x22222222"),
		})

		secondevmSignedTx, err := types.SignTx(tx, signer, secretKey)
		if err != nil {
			return err
		}

		secondevmEncodedTx, err := EncodeSignedTx(secondevmSignedTx)
		if err != nil {
			return err
		}

		endpointToTx[secondNodeEnpoint] = secondevmSignedTx.Hash().Hex()

		nodeCh, sNodeCh := make(chan []byte), make(chan []byte)
		go evmSendTx(nodeCh, evmEncodedTx, nodeEndpoint)
		go evmSendTx(sNodeCh, secondevmEncodedTx, secondNodeEnpoint)
		nodeRes, sNodeRes := <-nodeCh, <-sNodeCh

		fmt.Printf("node response: %s\n", string(nodeRes))
		fmt.Printf("second node response: %s\n", string(sNodeRes))

		time.Sleep(5 * time.Second)

		nonce++
		groupNumToTx[i] = endpointToTx
		// Add a delay to all the groups except for the last group
		if i < numTxGroups {
			fmt.Printf("Sleeping %d sec.\n", delay)
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}

	fmt.Println("Sleeping 7 sec before checking transaction status.")
	time.Sleep(7 * time.Second)

	var (
		endpointToTxMined = make(map[string]int)
		minedTxNums       = utils.NewHashSet()
		sleepLeftMinute   = 4
	)

	for len(minedTxNums) < numTxGroups && sleepLeftMinute > 0 {
		for groupNum, txMap := range groupNumToTx {
			grpNum := strconv.Itoa(groupNum)

			// Continue for confirmed transactions
			if minedTxNums.Contains(grpNum) {
				continue
			}

			// Check transactions sent to different endpoints and find the confirmed one
			for endpoint, txHash := range txMap {
				confirmed, err := IsConfirmed(txHash, nodeEndpoint)
				if err != nil {
					log.Errorf("cannot get tx confirmation, hash: %s, endpoint: %s, error: %v",
						txHash, endpoint, err)
					continue
				}

				if confirmed {
					endpointToTxMined[endpoint]++
					minedTxNums.Add(grpNum)
					break
				}
			}
		}

		// When there is any pending transaction, maximum sleep time is 4 min
		if len(minedTxNums) < numTxGroups {
			fmt.Printf("%d transactions are pending.\n"+
				"Sleeping 1 min before checking status again.\n",
				numTxGroups-len(minedTxNums))

			time.Sleep(60 * time.Second)
			sleepLeftMinute--
		}
	}

	fmt.Printf("\n----------------------------------------------------------------\n"+
		"Sent %d groups of transactions to node and second node endpoints,\n"+
		"%d of them have been confirmed:\n"+
		"Number of confirmed transactions is %d for first node endpoint %s\n"+
		"Number of confirmed transactions is %d for second node endpoint %s\n",
		numTxGroups,
		len(minedTxNums),
		endpointToTxMined[nodeEndpoint], nodeEndpoint,
		endpointToTxMined[secondNodeEnpoint], secondNodeEnpoint)

	return nil
}

func evmSendTx(out chan<- []byte, rawTx string, address string) {
	req := ws.NewRequest(1, "eth_sendRawTransaction", []interface{}{
		rawTx,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		out <- []byte(err.Error())
		return
	}

	data, err := DoRequest(address, reqBody)
	if err != nil {
		out <- []byte(err.Error())
	} else {
		out <- data
	}
}

// NewTxSpeedCompareService creates and initializes TxSpeedCompareService instance.
func NewTxSpeedCompareService() *TxSpeedCompareService {
	return &TxSpeedCompareService{}
}

func GetSenderAddress(key *ecdsa.PrivateKey) (string, error) {
	return crypto.PubkeyToAddress(key.PublicKey).Hex(), nil
}

func DoRequest(address string, body []byte) ([]byte, error) {
	t := time.Now()
	c := http.DefaultClient
	resp, err := c.Post(address, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	resBody, err := ioutil.ReadAll(resp.Body)
	log.Debug("response time ", time.Since(t), "body ", string(body))

	return resBody, err
}

func GetNonce(address string, nodeEndpoint string) (uint64, error) {
	req := ws.NewRequest(1, "eth_getTransactionCount", []interface{}{
		address, "latest",
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	data, err := DoRequest(nodeEndpoint, reqBody)
	if err != nil {
		return 0, err
	}

	var res nodeTxCountResponse
	if err = json.Unmarshal(data, &res); err != nil {
		return 0, err
	}

	if res.Error != nil {
		return 0, fmt.Errorf("cannot get nonce: %s", res.Error.Message)
	}

	if res.Result == nil {
		return 0, fmt.Errorf("cannot get nonce: empty response result")
	}

	return parseHexNum(*res.Result)
}

func GetBalance(address string, nodeEndpoint string) (uint64, error) {
	req := ws.NewRequest(1, "eth_getBalance", []interface{}{
		address, "latest",
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	data, err := DoRequest(nodeEndpoint, reqBody)
	if err != nil {
		return 0, err
	}

	var res nodeBalanceResponse
	if err = json.Unmarshal(data, &res); err != nil {
		return 0, err
	}

	if res.Error != nil {
		return 0, fmt.Errorf("cannot get balance: %s", res.Error.Message)
	}

	if res.Result == nil {
		return 0, fmt.Errorf("cannot get balance: empty response result")
	}

	return parseHexNum(*res.Result)
}

func IsConfirmed(txHash string, nodeEndpoint string) (bool, error) {
	req := ws.NewRequest(1, "eth_getTransactionReceipt", []interface{}{
		txHash,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return false, err
	}

	data, err := DoRequest(nodeEndpoint, reqBody)
	if err != nil {
		return false, err
	}

	var res nodeReceiptResponse
	if err = json.Unmarshal(data, &res); err != nil {
		return false, err
	}

	if res.Error != nil {
		return false, fmt.Errorf(
			"cannot get receipt for transaction %s: %s", txHash, res.Error.Message)
	}

	return res.Result != nil, nil
}

func trimHexPrefix(number string) string {
	return strings.Replace(strings.ToLower(number), "0x", "", 1)
}

func parseHexNum(number string) (uint64, error) {
	return strconv.ParseUint(trimHexPrefix(number), 16, 64)
}

func MakePrivateKey(key string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(key[2:])
}

func EncodeSignedTx(signedTx *types.Transaction) (string, error) {
	var buf bytes.Buffer

	if err := signedTx.EncodeRLP(&buf); err != nil {
		return "", err
	}

	return hexutil.Encode(buf.Bytes()), nil
}
