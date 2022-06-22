package measuretxpropagationtime

type handler func() error

type message struct {
	hash  string
	bytes []byte
	err   error
}

type evmTxFeedResponse struct {
	Params struct {
		Subscription string `json:"subscription"`
		Result       string `json:"result"`
	} `json:"params"`
}
