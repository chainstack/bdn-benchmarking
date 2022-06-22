package cmpfeeds

import "time"

type handler func() error

type message struct {
	hash  string
	bytes []byte
	err   error
}

type hashEntry struct {
	evmTimeReceived time.Time
	bxrTimeReceived time.Time
	hash            string
}

type evmTxFeedResponse struct {
	Params struct {
		Subscription string `json:"subscription"`
		Result       string `json:"result"`
	} `json:"params"`
}

type evmBkFeedResponse struct {
	Params struct {
		Subscription string `json:"subscription"`
		Result       struct {
			Hash string `json:"hash"`
		} `json:"result"`
	} `json:"params"`
}

type bxTxFeedResponse struct {
	Params struct {
		Result struct {
			TxHash     string `json:"txHash"`
			TxContents struct {
				GasPrice *string `json:"gasPrice"`
				To       *string `json:"to"`
			} `json:"txContents"`
		} `json:"result"`
	} `json:"params"`
}

type bxBkFeedResponse struct {
	Params struct {
		Result struct {
			Hash string `json:"hash"`
		} `json:"result"`
	} `json:"params"`
}

type evmTxContentsResponse struct {
	Result *struct {
		GasPrice string `json:"gasPrice"`
		To       string `json:"to"`
	} `json:"result"`
}

type evmBkContentsResponse struct {
	Result *struct {
		Hash string `json:"hash"`
	} `json:"result"`
}
