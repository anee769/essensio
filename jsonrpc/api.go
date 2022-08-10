package jsonrpc

import (
	"log"

	"github.com/anee769/essensio/core"
)

type API struct {
	chain *core.ChainManager
}

func NewAPI() *API {
	chain, err := core.NewChainManager()
	if err != nil {
		log.Fatalln("Failed to Start Blockchain:", err)
	}

	return &API{chain}
}

func (api *API) Stop() {
	api.chain.Stop()
}
