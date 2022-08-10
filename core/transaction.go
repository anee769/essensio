package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/anee769/essensio/common"
)

// Transactions is a group of Transaction objects
type Transactions []*Transaction

type Transaction struct {
	ID      common.Hash
	Inputs  []TxInput
	Outputs []TxOutput
}

type TxOutput struct {
	Value  int
	PubKey common.Address
}

type TxInput struct {
	ID  common.Hash
	Out int
	Sig common.Address
}

func CoinbaseTxn(to common.Address, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txnIn := TxInput{common.NullHash(), -1, common.Address(data)}
	txnOut := TxOutput{100, to}

	tx := Transaction{common.NullHash(), []TxInput{txnIn}, []TxOutput{txnOut}}
	tx.SetID()

	return &tx
}

func NewTransaction(from, to common.Address, amount int, chain *ChainManager) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput

	acc, validOutputs, err := chain.FindSpendableOutputs(from, amount)
	if err != nil {
		log.Panic(err)
	}

	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		for _, out := range outs {
			input := TxInput{txid, out, from}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, TxOutput{amount, to})

	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, from})
	}

	tx := Transaction{common.NullHash(), inputs, outputs}
	tx.SetID()

	return &tx
}

func (txn *Transaction) SetID() error {
	txnHash, err := txn.Serialize()
	if err != nil {
		return err
	}
	hash := sha256.Sum256(txnHash)
	txn.ID = hash
	return nil
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (in *TxInput) CanUnlock(address common.Address) bool {
	return in.Sig == address
}

func (out *TxOutput) CanBeUnlocked(address common.Address) bool {
	return out.PubKey == address
}

func (txn *Transaction) Serialize() ([]byte, error) {
	return common.GobEncode(txn)
}

// Deserialize implements the common.Serializable interface for Transaction.
// Converts the given data into Transaction and sets it the method's receiver using common.GobDecode.
func (txn *Transaction) Deserialize(data []byte) error {
	// Decode the data into a *Transaction
	object, err := common.GobDecode(data, new(Transaction))
	if err != nil {
		return err
	}

	// Cast the object into a *Transaction and
	// set it to the method receiver
	*txn = *object.(*Transaction)
	return nil
}

// Hash returns the SHA-256	hash of the Transaction's serialized representation.
func (txn *Transaction) Hash() common.Hash {
	data, err := txn.Serialize()
	if err != nil {
		return common.NullHash()
	}

	return common.Hash256(data)
}

// GenerateSummary generates a summary hash for a given set of Transactions.
// Currently, concatenates the hash of all given transactions and hashes that data to obtain the summary.
// This is a valid method of summary generation but does not for allow tamper detection or inclusivity checks
func GenerateSummary(txns Transactions) common.Hash {
	// Iterate over each transaction, obtain
	// its hash and append it into the buffer
	var buffer bytes.Buffer
	for _, txn := range txns {
		hash := txn.Hash()
		if hash == common.NullHash() {
			return hash
		}

		buffer.Write(hash.Bytes())
	}

	// Generate the hash of the buffer bytes
	txnsum := common.Hash256(buffer.Bytes())
	return txnsum
}
