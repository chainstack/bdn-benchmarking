package flags

import "github.com/urfave/cli/v2"

// CLI flags for evmcompare
var (
	Gateway = &cli.StringFlag{
		Name:  "gateway",
		Usage: "gateway websocket connection string",
		Value: "ws://127.0.0.1:28333/ws",
	}
	FeedWSEndpoint = &cli.StringFlag{
		Name:  "feed-ws-endpoint",
		Usage: "evm node websocket connection string",
		Value: "ws://127.0.0.1:8546",
	}
	TxFeedName = &cli.StringFlag{
		Name:  "feed-name",
		Usage: "specify feed name, possible values: 'newTxs', 'pendingTxs', 'transactionStatus'",
		Value: "newTxs",
	}
	BkFeedName = &cli.StringFlag{
		Name:  "feed-name",
		Usage: "specify feed name, possible values: 'newBlocks', 'bdnBlocks'",
		Value: "bdnBlocks",
	}
	MinGasPrice = &cli.Float64Flag{
		Name:  "min-gas-price",
		Usage: "gas price in gigawei",
	}
	Addresses = &cli.StringFlag{
		Name:  "addresses",
		Usage: "comma separated list of evm addresses",
	}
	ExcludeTxContents = &cli.BoolFlag{
		Name:  "exclude-tx-contents",
		Usage: "optionally exclude tx contents",
		Value: false,
	}
	ExcludeBkContents = &cli.BoolFlag{
		Name:  "exclude-block-contents",
		Usage: "optionally exclude block contents",
		Value: false,
	}
	Interval = &cli.IntFlag{
		Name:  "interval",
		Usage: "length of feed sample interval in seconds",
		Value: 60,
	}
	NumIntervals = &cli.IntFlag{
		Name:  "num-intervals",
		Usage: "number of intervals",
		Value: 1,
	}
	LeadTime = &cli.IntFlag{
		Name:  "lead-time",
		Usage: "seconds to wait before starting to compare feeds",
		Value: 60,
	}
	TxTrailTime = &cli.IntFlag{
		Name:  "trail-time",
		Usage: "seconds to wait after interval to receive tx on both feeds",
		Value: 60,
	}
	BkTrailTime = &cli.IntFlag{
		Name:  "trail-time",
		Usage: "seconds to wait after interval to receive blocks on both feeds",
		Value: 60,
	}
	Dump = &cli.StringFlag{
		Name:  "dump",
		Usage: "specify info to dump, possible values: 'ALL', 'MISSING', 'ALL,MISSING'",
	}
	ExcludeDuplicates = &cli.BoolFlag{
		Name:  "exclude-duplicates",
		Usage: "for pendingTxs only",
		Value: true,
	}
	TxIgnoreDelta = &cli.IntFlag{
		Name:  "ignore-delta",
		Usage: "ignore tx with delta above this amount (seconds)",
		Value: 5,
	}
	BkIgnoreDelta = &cli.IntFlag{
		Name:  "ignore-delta",
		Usage: "ignore blocks with delta above this amount (seconds)",
		Value: 5,
	}
	UseCloudAPI = &cli.BoolFlag{
		Name:  "use-cloud-api",
		Usage: "use cloud API",
		Value: false,
	}
	Verbose = &cli.BoolFlag{
		Name:  "verbose",
		Usage: "level of output",
		Value: false,
	}
	ExcludeFromBlockchain = &cli.BoolFlag{
		Name:  "exclude-from-blockchain",
		Usage: "exclude from blockchain",
		Value: false,
	}
	CloudAPIWSURI = &cli.StringFlag{
		Name:  "cloud-api-ws-uri",
		Usage: "specify websocket connection string for cloud API",
		Value: "wss://api.blxrbdn.com/ws",
	}
	AuthHeader = &cli.StringFlag{
		Name:  "auth-header",
		Usage: "authorization header created with account id and password",
	}
	UseGoGateway = &cli.BoolFlag{
		Name:  "use-go-gateway",
		Usage: "use GO Gateway",
		Value: false,
	}
	NodeWSEndpoint = &cli.StringFlag{
		Name:     "node-ws-endpoint",
		Usage:    "evm node ws endpoint. Sample Input: ws://127.0.0.1:8546",
		Required: true,
	}
	SecondNodeWSEndpoint = &cli.StringFlag{
		Name:     "second-node-ws-endpoint",
		Usage:    "evm ws endpoint. Sample Input: ws://127.0.0.1:8546",
		Required: false,
	}
	NodeEndpoint = &cli.StringFlag{
		Name:     "node-endpoint",
		Usage:    "evm node ws endpoint. Sample Input: http://127.0.0.1:8546",
		Required: true,
	}
	SecondNodeEndpoint = &cli.StringFlag{
		Name:     "second-node-endpoint",
		Usage:    "evm ws endpoint. Sample Input: http://127.0.0.1:8546",
		Required: false,
	}
	BXEndpoint = &cli.StringFlag{
		Name:  "blxr-endpoint",
		Usage: "bloXroute endpoint. Use wss://api.blxrbdn.com/ws for Cloud-API.",
		Value: "wss://api.blxrbdn.com/ws",
	}
	BXAuthHeader = &cli.StringFlag{
		Name: "blxr-auth-header",
		Usage: "bloXroute authorization header. Use base64 encoded value of " +
			"account_id:secret_hash for Cloud-API. For more information, see " +
			"https://bloxroute.com/docs/bloxroute-documentation/cloud-api/overview/",
	}
	SenderPrivateKey = &cli.StringFlag{
		Name:     "sender-private-key",
		Usage:    "Sender's private key, which starts with 0x.",
		Required: true,
	}
	ChainID = &cli.IntFlag{
		Name:  "chain-id",
		Usage: "EVM chain id",
		Value: 1,
	}
	NetworkName = &cli.StringFlag{
		Name:  "network-name",
		Usage: "One of networks name: Mainnet, BSC-Mainnet, Polygon-Mainnet",
		Value: "Mainnet",
	}
	NumTxGroups = &cli.IntFlag{
		Name:  "num-tx-groups",
		Usage: "Number of groups of transactions to submit.",
		Value: 1,
	}
	TxCount = &cli.IntFlag{
		Name:  "tx-count",
		Usage: "Number of transactions to submit.",
		Value: 1,
	}
	GasPrice = &cli.Int64Flag{
		Name:     "gas-price",
		Usage:    "Transaction gas price in Gwei.",
		Required: true,
	}
	Delay = &cli.IntFlag{
		Name:  "delay",
		Usage: "Time (sec) to sleep between two consecutive groups.",
		Value: 30,
	}
	UseBloxroute = &cli.BoolFlag{
		Name:  "use-bloxroute",
		Usage: "use BloXroute BDN",
		Value: false,
	}
	UseBlocknative = &cli.BoolFlag{
		Name:  "use-blocknative",
		Usage: "use Blocknative API",
		Value: false,
	}
	APIkey = &cli.StringFlag{
		Name:  "api-key",
		Usage: "API key for Blocknative requests",
		Value: "",
	}
)
