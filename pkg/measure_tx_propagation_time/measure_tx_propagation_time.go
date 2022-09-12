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
	"sync"
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

	//map[hash]map[feed-endpoint]PropagationTimes
	propagatedTxs      map[string]map[string]PropagationTimes
	propagatedTxsMutex sync.Mutex

	// map[feed-endpoint]PropagationTimes
	averageTimes      map[string]PropagationTimes
	averageTimesMutex sync.Mutex
}

type PropagationTimes struct {
	TimeDurSinceRequest  time.Duration
	TimeDurSinceResponse time.Duration
}

// NewMeasureTxPropagationTimeService creates and initializes MeasureTxPropagationTimeService instance.
func NewMeasureTxPropagationTimeService() *MeasureTxPropagationTimeService {
	return &MeasureTxPropagationTimeService{
		txHashToFind:  "foo",
		propagatedTxs: make(map[string]map[string]PropagationTimes),
		averageTimes:  make(map[string]PropagationTimes),
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

	//log.SetLevel(log.DebugLevel)

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
		txsFeedUris = c.StringSlice(flags.FeedWSEndpoint.Name)
		txsCount    = c.Int(flags.TxCount.Name)
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

	feedTXmap := make(map[string]<-chan *message)
	for _, txsFeedUri := range txsFeedUris {
		txFeedChan, err := s.readNewTxsFeed(ctx, txsFeedUri)
		if err != nil {
			zap.L().Error("error while reading new txs feed", zap.Error(err))
			return err
		}
		feedTXmap[txsFeedUri] = s.findTxHash(ctx, txFeedChan)

	}

	fmt.Printf("Starting sending and waiting for tx, tx count in queue %d\n\n", txsCount)
	for i := 0; i < txsCount; i++ {
		// check before send that previous tx already not pending

		var hash string
		var err error
		var tReqStart, tReqEnd time.Time

		log.Debug("Values:", c.Bool(flags.UseBlocknative.Name), " ", c.Bool(flags.UseBloxroute.Name), " ", c.String(flags.BXAuthHeader.Name), " ", c.String(flags.CloudAPIWSURI.Name), " ", c.String(flags.NetworkName.Name))

		switch {
		case c.Bool(flags.UseBlocknative.Name):
			fmt.Println("Use Blocknative")
			apiURI := c.String(flags.CloudAPIWSURI.Name)
			apiKey := c.String(flags.APIkey.Name)
			network := c.String(flags.NetworkName.Name)

			hash, tReqStart, tReqEnd, err = s.sendTxBlocknative(nodeEndpoint, address, gasLimit, gasPriceWei, int64(chainID), secretKey, apiURI, apiKey, network)
		case c.Bool(flags.UseBloxroute.Name):
			apiURI := c.String(flags.CloudAPIWSURI.Name)
			authHeader := c.String(flags.BXAuthHeader.Name)
			network := c.String(flags.NetworkName.Name)

			fmt.Println("Use Bloxroute")
			hash, tReqStart, tReqEnd, err = s.sendTxBloxroute(nodeEndpoint, address, gasLimit, gasPriceWei, int64(chainID), secretKey, apiURI, authHeader, network)
		default:
			log.Debug("Use Node endpoint")
			hash, tReqStart, tReqEnd, err = s.sendTx(nodeEndpoint, address, gasLimit, gasPriceWei, int64(chainID), secretKey)
		}

		if err != nil {
			zap.L().Error("Error while sendind tx", zap.Error(err))
			return err
		}

		// wait tx from all feed-ws-nodes
		feedWG := new(sync.WaitGroup)

		for feedEndpoint, ch := range feedTXmap {
			feedWG.Add(1)

			go func(feedEP string, c <-chan *message) {
				defer feedWG.Done()
				message := <-c
				if message.err != nil {
					zap.L().Error("Error while receiving message from tx feed", zap.Error(message.err))
					return
				}
				propTimes := PropagationTimes{TimeDurSinceRequest: time.Since(tReqStart), TimeDurSinceResponse: time.Since(tReqEnd)}

				s.setPropagatedTx(hash, feedEP, propTimes)

				s.addTxToAverageTimes(feedEP, propTimes, txsCount)

			}(feedEndpoint, ch)
		}

		feedWG.Wait()

		txReportStr := fmt.Sprintf("\nTx with hash %s propagated in: \n", hash)
		for feedEndpoint, timeDurs := range s.propagatedTxs[hash] {
			txReportStr += fmt.Sprintf("%s: %s[%s]\n", feedEndpoint, timeDurs.TimeDurSinceRequest, timeDurs.TimeDurSinceResponse)
		}
		txReportStr += fmt.Sprintf("\nSleeping for %s\n\n", c.Duration(flags.Delay.Name))
		fmt.Print(txReportStr)

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
	for hash, fMap := range s.propagatedTxs {
		fmt.Printf("%s propagated in:\n", hash)
		for feedNode, timeDurs := range fMap {
			fmt.Printf("%s: %s [%s]\n", feedNode, timeDurs.TimeDurSinceRequest, timeDurs.TimeDurSinceResponse)
		}
		fmt.Println()

	}

	for feedNode, avTimeDurs := range s.averageTimes {
		fmt.Printf("\nAverage propagation time for feed-node\n %s\n is %s [%s]\n", feedNode, avTimeDurs.TimeDurSinceRequest, avTimeDurs.TimeDurSinceResponse)
	}

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
	log.Debugf("Initiating connection to %s", uri)

	conn, err := ws.NewConnection(uri, "")
	if err != nil {
		return nil, fmt.Errorf("cannot establish connection to %s: %v", uri, err)
	}

	log.Debugf("Connection to %s established", uri)

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

func (s *MeasureTxPropagationTimeService) sendTx(nodeEndpoint, address string, gasLimit, gasPriceWei, chainID int64, secretKey *ecdsa.PrivateKey) (string, time.Time, time.Time, error) {
	var (
		addr   = common.HexToAddress(address)
		value  = big.NewInt(0)
		limit  = uint64(gasLimit)
		price  = big.NewInt(gasPriceWei)
		signer = types.NewEIP155Signer(big.NewInt(chainID))
	)
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

	evmEncodedTx, err := cmpnodestxspeedhttp.EncodeSignedTx(evmSignedTx)
	if err != nil {
		return "", time.Now(), time.Now(), err
	}

	req := ws.NewRequest(1, "eth_sendRawTransaction", []interface{}{
		evmEncodedTx,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", time.Now(), time.Now(), err
	}

	timeStartTXreq := time.Now()
	resp, err := cmpnodestxspeedhttp.DoRequest(nodeEndpoint, reqBody)
	timeEndTxReq := time.Now()

	log.Debugf("TX response: %v, error: %v, request time:%s\n", resp, err, timeEndTxReq.Sub(timeStartTXreq))

	return evmSignedTx.Hash().Hex(), timeStartTXreq, timeEndTxReq, err
}

func (s *MeasureTxPropagationTimeService) setPropagatedTx(hash, feedNode string, propTimeDur PropagationTimes) error {
	defer s.propagatedTxsMutex.Unlock()
	s.propagatedTxsMutex.Lock()

	feedMap, ok := s.propagatedTxs[hash]
	if ok {
		feedMap[feedNode] = propTimeDur
		s.propagatedTxs[hash] = feedMap
	} else {
		s.propagatedTxs[hash] = make(map[string]PropagationTimes)
		s.propagatedTxs[hash][feedNode] = propTimeDur
	}
	return nil
}

func (s *MeasureTxPropagationTimeService) addTxToAverageTimes(feedNode string, propTimeDur PropagationTimes, txsCount int) error {
	defer s.averageTimesMutex.Unlock()
	s.averageTimesMutex.Lock()

	averTimeDur, ok := s.averageTimes[feedNode]
	if ok {
		averTimeDur.TimeDurSinceRequest = averTimeDur.TimeDurSinceRequest + (propTimeDur.TimeDurSinceRequest / time.Duration(txsCount))
		averTimeDur.TimeDurSinceResponse = averTimeDur.TimeDurSinceResponse + (propTimeDur.TimeDurSinceResponse / time.Duration(txsCount))
		s.averageTimes[feedNode] = averTimeDur
	} else {
		s.averageTimes[feedNode] = PropagationTimes{TimeDurSinceRequest: (propTimeDur.TimeDurSinceRequest / time.Duration(txsCount)), TimeDurSinceResponse: (propTimeDur.TimeDurSinceResponse / time.Duration(txsCount))}
	}
	return nil
}
