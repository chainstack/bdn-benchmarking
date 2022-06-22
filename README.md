# Public benchmarking scripts (Go)

## Objective
This package contains a command line utility which is intended for benchmarking
performance of Evm node vs Bloxroute Go gateway. It has the following functionality:
* Compare transaction streams.
* Compare block streams.
* Compare transaction send speed.
* Compare transaction propagation time.

## Usage
This command line utility has three top-level commands:
* `transactions` - compares stream of txs from gateway vs node.
* `blocks` - compares stream of blocks from gateway vs node.
* `txspeed` - compares transaction sending speed by submitting conflicting txs
with the same nonce to node and gateway (so only one tx will land on chain).
* `nodetxspeed` - compares transaction sending speed by submitting conflicting txs
with the same nonce to two nodes (so only one tx will land on chain).
* `httpnodetxspeed` - compares transaction sending speed by submitting conflicting txs with the same nonce to two nodes (so only one tx will land on chain) with http.
* `measuretxpropagationtime` - take two nodes at two different ends of the earth, becnhmark will send tx to closest node and subscribe to NewPendingTxs in other node, then measure time between getting response from the closest node and catch tx with same hash from other node

### Transactions steam
This benchmark is invoked by `transactions` command which has the following options:
```
   --gateway value            gateway websocket connection string (default: "ws://127.0.0.1:28333/ws")
   --feed-ws-endpoint value                node websocket connection string (default: "ws://127.0.0.1:8546")
   --feed-name value          specify feed name, possible values: 'newTxs', 'pendingTxs', 'transactionStatus' (default: "newTxs")
   --min-gas-price value      gas price in gigawei (default: 0)
   --addresses value          comma separated list of Evm addresses
   --exclude-tx-contents      optionally exclude tx contents (default: false)
   --interval value           length of feed sample interval in seconds (default: 60)
   --num-intervals value      number of intervals (default: 1)
   --lead-time value          seconds to wait before starting to compare feeds (default: 60)
   --trail-time value         seconds to wait after interval to receive tx on both feeds (default: 60)
   --dump value               specify info to dump, possible values: 'ALL', 'MISSING', 'ALL,MISSING'
   --exclude-duplicates       for pendingTxs only (default: true)
   --ignore-delta value       ignore tx with delta above this amount (seconds) (default: 5)
   --use-cloud-api            use cloud API (default: false)
   --verbose                  level of output (default: false)
   --exclude-from-blockchain  exclude from blockchain (default: false)
   --cloud-api-ws-uri value   specify websocket connection string for cloud API (default: "wss://api.blxrbdn.com/ws")
   --auth-header value        authorization header created with account id and password
   --use-go-gateway           use GO Gateway (default: false)
   --help, -h                 show help (default: false)
```
The following command can be used to print help related to `transactions` command:
```shell
go run cmd/evmcompare/main.go transactions -h
```
#### Example
Here is an example of using `transactions` command:
```shell
go run cmd/evmcompare/main.go transactions --gateway wss://uk.eth.blxrbdn.com/ws --auth-header <YOUR HEADER> --feed-name wss://ws-nd-816-696-544.p2pify.com/1388f61befcd2f46869e9f6a10d57547
```

### Blocks stream
This benchmark is invoked by `blocks` command which has the following options:
```
   --gateway value           gateway websocket connection string (default: "ws://127.0.0.1:28333/ws")
   --feed-ws-endpoint value               node websocket connection string (default: "ws://127.0.0.1:8546")
   --feed-name value         specify feed name, possible values: 'newBlocks', 'bdnBlocks' (default: "bdnBlocks")
   --exclude-block-contents  optionally exclude block contents (default: false)
   --interval value          length of feed sample interval in seconds (default: 60)
   --num-intervals value     number of intervals (default: 1)
   --lead-time value         seconds to wait before starting to compare feeds (default: 60)
   --trail-time value        seconds to wait after interval to receive blocks on both feeds (default: 60)
   --dump value              specify info to dump, possible values: 'ALL', 'MISSING', 'ALL,MISSING'
   --ignore-delta value      ignore blocks with delta above this amount (seconds) (default: 5)
   --use-cloud-api           use cloud API (default: false)
   --auth-header value       authorization header created with account id and password
   --cloud-api-ws-uri value  specify websocket connection string for cloud API (default: "wss://api.blxrbdn.com/ws")
   --help, -h                show help (default: false)
```
The following command can be used to print help related to `blocks` command:
```shell
go run cmd/evmcompare/main.go blocks -h
```
#### Example
Here is an example of using `blocks` command:
```shell
go run cmd/evmcompare/main.go blocks --gateway wss://uk.eth.blxrbdn.com/ws --auth-header <YOUR HEADER> --feed-name wss://ws-nd-816-696-544.p2pify.com/1388f61befcd2f46869e9f6a10d57547
```

### Transactions speed
This benchmark is invoked by `txspeed` command which has the following options:
```
   --node-ws-endpoint value    Evm node ws endpoint. Sample Input: ws://127.0.0.1:8546
   --blxr-endpoint value       bloXroute endpoint. Use wss://api.blxrbdn.com/ws for Cloud-API. (default: "wss://api.blxrbdn.com/ws")
   --blxr-auth-header value    bloXroute authorization header. Use base64 encoded value of account_id:secret_hash for Cloud-API. For more information, see https://bloxroute.com/docs/bloxroute-documentation/cloud-api/overview/
   --sender-private-key value  Sender's private key, which starts with 0x.
   --chain-id value            EVM chain id (default: 1)
   --num-tx-groups value       Number of groups of transactions to submit. (default: 1)
   --gas-price value           Transaction gas price in Gwei. (default: 0)
   --delay value               Time (sec) to sleep between two consecutive groups. (default: 30)
   --network-name value        One of networks name: Mainnet, BSC-Mainnet, Polygon-Mainnet
   --help, -h                  show help (default: false)
```
The following command can be used to print help related to `txspeed` command:
```shell
go run cmd/evmcompare/main.go txspeed -h
```
#### Example
Here is an example of using `txspeed` command:
```shell
go run cmd/evmcompare/main.go txspeed --node-ws-endpoint wss://ws-nd-612-026-052.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --chain-id 56 --sender-private-key <YOUR PRIVATE KEY> --blxr-endpoint ws://127.0.0.1:28333 --blxr-auth-header <YOUR AUTH HEADER> --gas-price 50 --num-tx-groups 10 --network-name BSC-Mainnet
```

### Transactions speed between two nodes
This benchmark is invoked by `nodetxspeed` command which has the following options:
```
   --node-ws-endpoint value    Evm node ws endpoint. Sample Input: ws://127.0.0.1:8546
   --second-node-ws-endpoint value    Second Evm node ws endpoint. Sample Input: ws://127.0.0.1:8546
   --sender-private-key value  Sender's private key, which starts with 0x.
   --chain-id value            EVM chain id (default: 1)
   --num-tx-groups value       Number of groups of transactions to submit. (default: 1)
   --gas-price value           Transaction gas price in Gwei. (default: 0)
   --delay value               Time (sec) to sleep between two consecutive groups. (default: 30)
   --help, -h                  show help (default: false)
```
The following command can be used to print help related to `nodetxspeed` command:
```shell
go run cmd/evmcompare/main.go nodetxspeed -h
```
#### Example
Here is an example of using `nodetxspeed` command:
```shell
go run cmd/evmcompare/main.go nodetxspeed --node-ws-endpoint wss://ws-nd-612-026-052.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --second-node-ws-endpoint wss://ws-nd-612-026-052.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --chain-id 56 --sender-private-key <YOUR PRIVATE KEY> --gas-price 5 --num-tx-groups 10
```

### Transactions speed between two nodes with http
This benchmark is invoked by `httpnodetxspeed` command which has the following options:
```
   --node-endpoint value    Evm node HTTP endpoint. Sample Input: http://127.0.0.1:8546
   --second-node-endpoint value    Second Evm node HTTP endpoint. Sample Input: http://127.0.0.1:8546
   --sender-private-key value  Sender's private key, which starts with 0x.
   --chain-id value            EVM chain id (default: 1)
   --num-tx-groups value       Number of groups of transactions to submit. (default: 1)
   --gas-price value           Transaction gas price in Gwei. (default: 0)
   --delay value               Time (sec) to sleep between two consecutive groups. (default: 30)
   --help, -h                  show help (default: false)
```
The following command can be used to print help related to `httpnodetxspeed` command:
```shell
go run cmd/evmcompare/main.go httpnodetxspeed -h
```
#### Example
Here is an example of using `httpnodetxspeed` command:
```shell
go run cmd/evmcompare/main.go httpnodetxspeed --node-endpoint https://nd-612-026-052.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --second-node-endpoint https://nd-612-026-052.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --chain-id 137 --sender-private-key <YOUR PRIVATE KEY> --gas-price 50 --delay 5 --num-tx-groups 10
```

### Measure tx propagation time
This benchmark is invoked by `measuretxpropagationtime` command which has the following options:
```
   --node-endpoint value    Evm node HTTP endpoint. Sample Input: http://127.0.0.1:8546
   --feed-ws-endpoint value   Evm node ws endpoint. Sample Input: ws://127.0.0.1:8546
   --sender-private-key value  Sender's private key, which starts with 0x.
   --chain-id value            EVM chain id (default: 1)
   --tx-count value       Number of transactions to submit. (default: 1)
   --gas-price value           Transaction gas price in Gwei. (default: 0)
   --delay value               Time (sec) to sleep between sending tx. (default: 0)
   --help, -h                  show help (default: false)
```
The following command can be used to print help related to `measuretxpropagationtime` command:
```shell
go run cmd/evmcompare/main.go measuretxpropagationtime -h
```
#### Example
Here is an example of using `measuretxpropagationtime` command:
```shell
go run cmd/evmcompare/main.go measuretxpropagationtime --node-endpoint https://nd-143-578-236.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --feed-ws-endpoint wss://ws-nd-816-696-544.p2pify.com/1388f61befcd2f46869e9f6a10d57547 --chain-id 137 --sender-private-key <YOUR PRIVATE KEY> --gas-price 300 --tx-count 10
```


## Installation
This package requires only Go to be installed in the system.
Dependencies should be downloaded automatically when `go run cmd/evmcompare/main.go`
is attempted for the first time.