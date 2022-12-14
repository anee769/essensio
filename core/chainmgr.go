package core

import (
	"fmt"

	"github.com/anee769/essensio/common"
	"github.com/anee769/essensio/db"
)

var (
	ChainHeadKey   = []byte("state-chainhead")
	ChainHeightKey = []byte("state-chainheight")
)

// ChainManager represents a blockchain as a set of Blocks
type ChainManager struct {
	// Represents the database of blockchain data
	// This contains the state and blocks of the blockchain
	db *db.Database

	// Represents the hash of the last Block
	Head common.Hash
	// Represents the Height of the chain. Last block Height+1
	Height int64
}

// String implements the Stringer interface for BlockChain
func (chain *ChainManager) String() string {
	return fmt.Sprintf("Chain Head: %x || Chain Height: %v", chain.Head, chain.Height)
}

// AddBlock generates and appends a Block to the chain for a given string data.
// The generated block is stored in the database. Any error that occurs is returned.
func (chain *ChainManager) AddBlock(txns Transactions) error {
	// Create a new Block with the given data
	block := NewBlock(txns, chain.Head, chain.Height)

	// Serialize the Block
	blockData, err := block.Serialize()
	if err != nil {
		return fmt.Errorf("block serialize failed: %w", err)
	}

	// Add block to db
	if err := chain.db.SetEntry(block.BlockHash.Bytes(), blockData); err != nil {
		return fmt.Errorf("block store to db failed: %w", err)
	}

	// Update the chain head with the new block hash and increment chain height
	chain.Head = block.BlockHash
	chain.Height++

	// Sync the chain state into the DB
	if err := chain.syncState(); err != nil {
		return fmt.Errorf("chain state sync failed: %w", err)
	}

	return nil
}

// NewChainManager returns a new BlockChain with an initialized
// Genesis Block with the provided genesis data.
func NewChainManager() (*ChainManager, error) {
	// Create a new ChainManager object
	chain := new(ChainManager)

	// Check if the database already exists
	if db.Exists() {
		// Load blockchain state from database
		if err := chain.load(); err != nil {
			return nil, fmt.Errorf("failed to load existing blockchain: %w", err)
		}

	} else {
		// Initialize blockchain state and database
		if err := chain.init(); err != nil {
			return nil, fmt.Errorf("failed to initialize new blockchain: $%w", err)
		}
	}

	return chain, nil
}

// load restarts a ChainManager from the database.
// It updates its in-memory chain state chain information from the DB.
func (chain *ChainManager) load() (err error) {
	// Open the database
	if chain.db, err = db.Open(); err != nil {
		return err
	}

	// Get the chain head and set it
	head, err := chain.db.GetEntry(ChainHeadKey)
	if err != nil {
		return fmt.Errorf("chain head retrieve failed: %w", err)
	}

	// Get the chain height
	height, err := chain.db.GetEntry(ChainHeightKey)
	if err != nil {
		return fmt.Errorf("chain height retrieve failed: %w", err)
	}

	// Deserialize the height into an int64
	object, err := common.GobDecode(height, new(int64))
	if err != nil {
		return fmt.Errorf("error deserializing chain height: %w", err)
	}

	// Cast the object into an int64 and set it
	chain.Height = *object.(*int64)
	// Convert the head bytes into a Hash and set it
	chain.Head = common.BytesToHash(head)

	return nil
}

// init initializes a new chain in the database.
// It generates a Genesis Block and adds it to DB and updates all chain state data.
func (chain *ChainManager) init() (err error) {
	// Open the database
	if chain.db, err = db.Open(); err != nil {
		return err
	}

	fmt.Println(">>>> New Blockchain Initialization. Creating Genesis Block <<<<")

	// Create Genesis Block & serialize it
	genesisBlock := NewBlock(
		Transactions{CoinbaseTxn(common.MinerAddress(), "Genesis Block Coinbase Transaction")},
		common.NullHash(), 0)
	genesisData, err := genesisBlock.Serialize()
	if err != nil {
		return fmt.Errorf("block serialize failed: %w", err)
	}

	// Add Genesis Block to DB
	if err := chain.db.SetEntry(genesisBlock.BlockHash.Bytes(), genesisData); err != nil {
		return fmt.Errorf("genesis block store to db failed: %w", err)
	}

	// Set the chain height and head into struct
	chain.Head, chain.Height = genesisBlock.BlockHash, 1

	// Sync the chain state into the DB
	if err := chain.syncState(); err != nil {
		return fmt.Errorf("chain state sync failed: %w", err)
	}

	return nil
}

func (chain *ChainManager) Stop() {
	chain.db.Close()
}

// syncState updates the chain head and height values into the DB at keys
// specified by the ChainHeadKey and ChainHeightKey respectively.
func (chain *ChainManager) syncState() error {
	// Sync chain head into the DB
	if err := chain.db.SetEntry(ChainHeadKey, chain.Head.Bytes()); err != nil {
		return fmt.Errorf("error syncing chain head: %w", err)
	}

	// Serialize the chain height
	height, err := common.GobEncode(chain.Height)
	if err != nil {
		return fmt.Errorf("error serializing chain height: %w", err)
	}

	// Sync the encoded height into the DB
	if err := chain.db.SetEntry(ChainHeightKey, height); err != nil {
		return fmt.Errorf("error syncing chain height: %w", err)
	}

	return nil
}

func (chain *ChainManager) FindUnspentTransactions(address common.Address) (Transactions, error) {
	var unspentTxs Transactions

	spentTXOs := make(map[common.Hash][]int)

	iter := chain.NewIterator()

	for {
		block, err := iter.Next()
		if err != nil {
			return nil, err
		}

		for _, tx := range block.BlockTxns {
			txID := tx.ID

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, tx)
				}
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := in.ID
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
					}
				}
			}
		}

		if len(block.Priori) == 0 {
			break
		}
	}
	return unspentTxs, nil
}

func (chain *ChainManager) FindUTXO(address common.Address) ([]TxOutput, error) {
	var UTXOs []TxOutput
	unspentTransactions, err := chain.FindUnspentTransactions(address)
	if err != nil {
		return nil, err
	}

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs, nil
}

func (chain *ChainManager) FindSpendableOutputs(address common.Address, amount int) (int, map[common.Hash][]int, error) {
	unspentOuts := make(map[common.Hash][]int)
	unspentTxs, err := chain.FindUnspentTransactions(address)
	if err != nil {
		return -1, nil, err
	}
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := tx.ID

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOuts, nil
}
