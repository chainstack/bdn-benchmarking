package measuretxpropagationtime

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"performance/internal/pkg/flags"
	"performance/internal/pkg/ws"
	"performance/pkg/cmpnodestxspeedhttp"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// MeasureTxPropagationTimeService represents a service which compares transaction feeds time difference
// between EVM node and BX gateway.
type MeasureTxPropagationTimeService struct {
	txHashToFind string

	propagatedTxs map[string]time.Duration
}

// NewMeasureTxPropagationTimeService creates and initializes MeasureTxPropagationTimeService instance.
func NewMeasureTxPropagationTimeService() *MeasureTxPropagationTimeService {
	return &MeasureTxPropagationTimeService{
		txHashToFind:  "foo",
		propagatedTxs: make(map[string]time.Duration),
	}
}

// Run is an entry point to the MeasureTxPropagationTimeService.
func (s *MeasureTxPropagationTimeService) Run(c *cli.Context) error {
	var (
		gasLimit         = int64(22000)
		senderPrivateKey = c.String(flags.SenderPrivateKey.Name)
		gasPriceWei      = c.Int64(flags.GasPrice.Name) * params.GWei
		chainID          = c.Int(flags.ChainID.Name)
		nodeEndpoint     = c.String(flags.NodeEndpoint.Name)
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secretKey, err := cmpnodestxspeedhttp.MakePrivateKey(senderPrivateKey)
	if err != nil {
		zap.L().Error("error while making private key", zap.Error(err))
		return err
	}

	address, err := cmpnodestxspeedhttp.GetSenderAddress(secretKey)
	if err != nil {
		zap.L().Error("error while getting sender address", zap.Error(err))
		return err
	}

	balance, err := cmpnodestxspeedhttp.GetBalance(address, nodeEndpoint)
	if err != nil {
		zap.L().Error("error while getting balance", zap.Error(err))
		return err
	}

	var (
		txsFeedUri = c.String(flags.FeedWSEndpoint.Name)
		txsCount   = c.Int(flags.TxCount.Name)
	)

	if expense := int64(txsCount) * gasPriceWei * gasLimit; balance < uint64(expense) {
		var (
			requiredEvm = float64(expense) / params.Ether
			currentEvm  = float64(balance) / params.Ether
		)

		fmt.Printf("Sender %s does not have enough balance for %d groups of transactions.\n"+
			"Sender's balance is %f Coins,\n"+
			"while at least %f Coins is required\n",
			address,
			txsCount,
			currentEvm,
			requiredEvm)
	}

	txFeedChan, err := s.readNewTxsFeed(ctx, txsFeedUri)
	if err != nil {
		zap.L().Error("error while reading new txs feed", zap.Error(err))
		return err
	}

	foundTxHashChan := s.findTxHash(ctx, txFeedChan)

	averagePropagationTime := time.Duration(0)
	fmt.Printf("Starting sending and waiting for tx, tx count in queue %d\n\n", txsCount)
	for i := 0; i < txsCount; i++ {
		// check before send that previous tx already not pending

		hash, err := s.sendTx(nodeEndpoint, address, gasLimit, gasPriceWei, int64(chainID), secretKey)
		if err != nil {
			zap.L().Error("Error while sendind tx", zap.Error(err))
			return err
		}
		now := time.Now()
		message := <-foundTxHashChan
		if message.err != nil {
			zap.L().Error("Error while receiving message from tx feed", zap.Error(message.err))
			return err
		}
		propagationTime := time.Since(now)
		s.propagatedTxs[hash] = propagationTime
		averagePropagationTime = averagePropagationTime + (propagationTime / time.Duration(txsCount))
		fmt.Printf("\nTx with hash %s propagated in %s\nSleeping for %s\n\n", hash, propagationTime, c.Duration(flags.Delay.Name))
		time.Sleep(c.Duration(flags.Delay.Name))
		for {
			if confirmed, err := cmpnodestxspeedhttp.IsConfirmed(s.txHashToFind, nodeEndpoint); !confirmed || err != nil {
				fmt.Printf("Waiting for the tx '%s' to be confirmed, sleeping for 5s.\n", s.txHashToFind)
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
	}

	fmt.Println("\n\nResult:")
	for hash, duration := range s.propagatedTxs {
		fmt.Printf("%s propagated in %s\n", hash, duration)
	}
	fmt.Printf("\nAverage propagation time is %s\n", averagePropagationTime)

	return nil
}

func (s *MeasureTxPropagationTimeService) findTxHash(
	ctx context.Context, txFeed <-chan *message,
) <-chan *message {
	foundTxChan := make(chan *message)
	go func() {
		defer close(foundTxChan)
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-txFeed:
				if !ok {
					continue
				}

				if data.hash == s.txHashToFind {
					fmt.Printf("found tx with hash %s\n", data.hash)
					foundTxChan <- data
				}
			}
		}
	}()

	return foundTxChan
}

func (s *MeasureTxPropagationTimeService) readNewTxsFeed(
	ctx context.Context, uri string,
) (<-chan *message, error) {
	log.Debug("Initiating connection to %s", uri)

	conn, err := ws.NewConnection(uri, "")
	if err != nil {
		return nil, fmt.Errorf("cannot establish connection to %s: %v", uri, err)
	}

	log.Debug("Connection to %s established", uri)

	sub, err := conn.SubscribeTxFeedEvm(1)
	if err != nil {
		return nil, fmt.Errorf("cannot subscribe to EVM feed: %v", err)
	}

	out := make(chan *message)

	go func() {
		defer close(out)

		defer func() {
			if err := conn.Close(); err != nil {
				log.Errorf("cannot close socket connection to %s: %v", uri, err)
			}
		}()

		defer func() {
			if err := sub.Unsubscribe(); err != nil {
				log.Errorf("cannot unsubscribe from EVM feed: %v", err)
			}
		}()
		for {
			var (
				data, err = sub.NextMessage()
				msg       = &message{
					bytes: data,
					err:   err,
				}
			)
			var feedRes evmTxFeedResponse
			err = json.Unmarshal(data, &feedRes)
			if err != nil {
				fmt.Printf("Error while unmarshaling tx feed: %v", err)
				return
			}

			msg.hash = feedRes.Params.Result

			select {
			case <-ctx.Done():
				return
			case out <- msg:
			}
		}
	}()

	return out, nil
}

func (s *MeasureTxPropagationTimeService) sendTx(nodeEndpoint, address string, gasLimit, gasPriceWei, chainID int64, secretKey *ecdsa.PrivateKey) (string, error) {
	var (
		addr   = common.HexToAddress(address)
		value  = big.NewInt(0)
		limit  = uint64(gasLimit)
		price  = big.NewInt(gasPriceWei)
		signer = types.NewEIP155Signer(big.NewInt(chainID))
	)
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

	evmEncodedTx, err := cmpnodestxspeedhttp.EncodeSignedTx(evmSignedTx)
	if err != nil {
		return "", err
	}

	req := ws.NewRequest(1, "eth_sendRawTransaction", []interface{}{
		evmEncodedTx,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	data, err := cmpnodestxspeedhttp.DoRequest(nodeEndpoint, reqBody)
	log.Debug("Send transaction response: %s, error: %v\n", string(data), err)

	return evmSignedTx.Hash().Hex(), err
}
