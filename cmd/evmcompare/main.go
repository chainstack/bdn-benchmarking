package main

import (
	"os"
	"performance/internal/pkg/flags"
	"performance/pkg/cmpfeeds"
	"performance/pkg/cmpnodestxspeed"
	"performance/pkg/cmpnodestxspeedhttp"
	"performance/pkg/cmptxspeed"
	measuretxpropagationtime "performance/pkg/measure_tx_propagation_time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	app := &cli.App{
		Name:  "evmcompare",
		Usage: "compares stream of txs/blocks from gateway vs node",
		Commands: []*cli.Command{
			{
				Name:  "transactions",
				Usage: "compares stream of txs from gateway vs node",
				Flags: []cli.Flag{
					flags.Gateway,
					flags.FeedWSEndpoint,
					flags.TxFeedName,
					flags.MinGasPrice,
					flags.Addresses,
					flags.ExcludeTxContents,
					flags.Interval,
					flags.NumIntervals,
					flags.LeadTime,
					flags.TxTrailTime,
					flags.Dump,
					flags.ExcludeDuplicates,
					flags.TxIgnoreDelta,
					flags.UseCloudAPI,
					flags.Verbose,
					flags.ExcludeFromBlockchain,
					flags.CloudAPIWSURI,
					flags.AuthHeader,
					flags.UseGoGateway,
				},
				Action: cmpfeeds.NewTxFeedsCompareService().Run,
			},
			{
				Name:  "blocks",
				Usage: "compares stream of blocks from gateway vs node",
				Flags: []cli.Flag{
					flags.Gateway,
					flags.FeedWSEndpoint,
					flags.BkFeedName,
					flags.ExcludeBkContents,
					flags.Interval,
					flags.NumIntervals,
					flags.LeadTime,
					flags.BkTrailTime,
					flags.Dump,
					flags.BkIgnoreDelta,
					flags.UseCloudAPI,
					flags.AuthHeader,
					flags.CloudAPIWSURI,
				},
				Action: cmpfeeds.NewBkFeedsCompareService().Run,
			},
			{
				Name: "txspeed",
				Usage: "compares sending tx speed by submitting conflicting txs with the same nonce " +
					"to node and gateway, so only one tx will land on chain",
				Flags: []cli.Flag{
					flags.NodeWSEndpoint,
					flags.BXEndpoint,
					flags.BXAuthHeader,
					flags.SenderPrivateKey,
					flags.ChainID,
					flags.NumTxGroups,
					flags.GasPrice,
					flags.Delay,
					flags.NetworkName,
				},
				Action: cmptxspeed.NewTxSpeedCompareService().Run,
			},
			{
				Name: "nodetxspeed",
				Usage: "compares sending tx speed by submitting conflicting txs with the same nonce " +
					"to two nodes, so only one tx will land on chain",
				Flags: []cli.Flag{
					flags.NodeWSEndpoint,
					flags.SecondNodeWSEndpoint,
					flags.SenderPrivateKey,
					flags.ChainID,
					flags.NumTxGroups,
					flags.GasPrice,
					flags.Delay,
				},
				Action: cmpnodestxspeed.NewTxSpeedCompareService().Run,
			},
			{
				Name: "httpnodetxspeed",
				Usage: "compares sending tx speed by submitting conflicting txs with the same nonce " +
					"to two nodes, so only one tx will land on chain",
				Flags: []cli.Flag{
					flags.NodeEndpoint,
					flags.SecondNodeEndpoint,
					flags.SenderPrivateKey,
					flags.ChainID,
					flags.NumTxGroups,
					flags.GasPrice,
					flags.Delay,
				},
				Action: cmpnodestxspeedhttp.NewTxSpeedCompareService().Run,
			},
			{
				Name: "measuretxpropagationtime",
				Usage: "takes two nodes at two different ends of the earth, sending tx to closest node " +
					"and subscribing to NewPendingTxs in other node, then measuring time between getting response from the closest node and catching tx with same hash from other node",
				Flags: []cli.Flag{
					flags.NodeEndpoint,
					flags.FeedWSEndpoint,
					flags.SenderPrivateKey,
					flags.ChainID,
					flags.TxCount,
					flags.GasPrice,
					flags.Delay,
					flags.UseBloxroute, // usage: --use-bloxroute --cloud-api-ws-uri wss://virginia.eth.blxrbdn.com/ws --blxr-auth-header NzQ4NGJmNzkt...
					flags.BXAuthHeader,
					flags.UseBlocknative, // usage: --use-blocknative --cloud-api-ws-uri wss://api.blocknative.com/v0 --api-key ba13ea2f-00c3-13c5.....
					flags.APIkey,
					flags.CloudAPIWSURI,
					flags.NetworkName,
				},
				Action: measuretxpropagationtime.NewMeasureTxPropagationTimeService().Run,
			},
		},
	}

	log, _ := zap.NewDevelopment(zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(log)

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("fatal", zap.Error(err))
	}
}
