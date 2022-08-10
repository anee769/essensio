package jsonrpc

import (
	"fmt"
	"log"
	"net/http"

	"github.com/anee769/essensio/common"
	"github.com/anee769/essensio/core"
)

type AddBlockArgs struct {
	Transactions []TransactionInput `json:"transactions"`
}

type TransactionInput struct {
	To    string `json:"to"`
	From  string `json:"from"`
	Value int    `json:"value"`
}

type AddBlockResult struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash"`
}

func (api *API) AddBlock(r *http.Request, args *AddBlockArgs, result *AddBlockResult) error {
	log.Println("'AddBlock' Called")

	if len(args.Transactions) == 0 {
		return fmt.Errorf("no transactions for block")
	}

	transactions := make(core.Transactions, 0, len(args.Transactions))
	for _, txn := range args.Transactions {
		newtxn := core.NewTransaction(common.Address(txn.From), common.Address(txn.To), txn.Value, api.chain)
		transactions = append(transactions, newtxn)
	}

	if err := api.chain.AddBlock(transactions); err != nil {
		return fmt.Errorf("failed to add block: %w", err)
	}

	*result = AddBlockResult{
		BlockHeight: uint64(api.chain.Height - 1),
		BlockHash:   api.chain.Head.Hex(),
	}

	return nil
}
