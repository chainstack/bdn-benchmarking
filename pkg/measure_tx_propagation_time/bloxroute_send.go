package measuretxpropagationtime

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"performance/internal/pkg/ws"
	"performance/pkg/cmpnodestxspeedhttp"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

func (s *MeasureTxPropagationTimeService) sendTxBloxroute(nodeEndpoint, address string, gasLimit, gasPriceWei, chainID int64, secretKey *ecdsa.PrivateKey, apiAddress, apiKey, networkName string) (string, time.Time, error) {
	var (
		addr   = common.HexToAddress(address)
		value  = big.NewInt(0)
		limit  = uint64(gasLimit)
		price  = big.NewInt(gasPriceWei)
		signer = types.NewEIP155Signer(big.NewInt(chainID))
	)
	nonce, err := cmpnodestxspeedhttp.GetNonce(address, nodeEndpoint)
	if err != nil {
		return "", time.Now(), err
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
		return "", time.Now(), err
	}

	s.txHashToFind = evmSignedTx.Hash().Hex()
	fmt.Printf("hash to find %s\n", evmSignedTx.Hash().Hex())

	rawTx, err := cmpnodestxspeedhttp.EncodeSignedTxWithout0xPrefix(evmSignedTx)
	if err != nil {
		return "", time.Now(), err
	}

	// method: "blxr_tx" for bloXroute
	req := ws.NewRequest(1, "blxr_tx", []interface{}{
		map[string]interface{}{
			"transaction":        rawTx,
			"blockchain_network": networkName,
		},
	})

	wsconn, err := ws.NewConnection(apiAddress, apiKey)
	if err != nil {
		return "", time.Now(), err
	}

	go func() {
		resp, err := wsconn.Call(req)
		defer wsconn.Close()

		log.Debugf("Send transaction response: %s, error: %v\n", string(resp), err)
	}()

	return evmSignedTx.Hash().Hex(), time.Now(), err
}
