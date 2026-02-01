package blockchain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/jinba225/v6coin-protocol/pkg/state"
)

var (
	ErrBlockNotFound     = errors.New("block not found")
	ErrInvalidChain      = errors.New("invalid chain")
	ErrInvalidHeight     = errors.New("invalid block height")
	ErrForkDetected      = errors.New("fork detected")
	ErrInvalidParent     = errors.New("invalid parent block")
	ErrDuplicateBlock    = errors.New("duplicate block")
	ErrStateRootMismatch = errors.New("state root mismatch")
)

// BlockChain represents the V6Coin blockchain
type BlockChain struct {
	genesisBlock *consensus.Block
	chainHead    *consensus.Block
	blocks       map[string]*consensus.Block   // hash -> block
	heightMap    map[uint64][]*consensus.Block // height -> blocks (for forks)
	chainMutex   sync.RWMutex
	stateDB      state.StateDB
	validator    Validator
	rewardDist   RewardDistributor
}

// Validator is interface for block validation
type Validator interface {
	ValidateBlock(block *consensus.Block, parent *consensus.Block) error
	ValidateTransaction(tx *consensus.Transaction) error
	IsValidBlockHash(hash []byte) bool
}

// RewardDistributor is interface for reward distribution
type RewardDistributor interface {
	DistributeBlockReward(block *consensus.Block, stateDB state.StateDB) error
}

// BlockChainConfig is configuration for blockchain
type BlockChainConfig struct {
	GenesisBlock *consensus.Block
	StateDB      state.StateDB
	Validator    Validator
	RewardDist   RewardDistributor
}

// NewBlockChain creates a new blockchain
func NewBlockChain(config *BlockChainConfig) (*BlockChain, error) {
	if config.GenesisBlock == nil {
		return nil, errors.New("genesis block required")
	}

	bc := &BlockChain{
		genesisBlock: config.GenesisBlock,
		chainHead:    config.GenesisBlock,
		blocks:       make(map[string]*consensus.Block),
		heightMap:    make(map[uint64][]*consensus.Block),
		stateDB:      config.StateDB,
		validator:    config.Validator,
		rewardDist:   config.RewardDist,
	}

	// Add genesis block
	genesisHash := string(config.GenesisBlock.Hash())
	bc.blocks[genesisHash] = config.GenesisBlock
	bc.heightMap[0] = []*consensus.Block{config.GenesisBlock}

	return bc, nil
}

// AddBlock adds a block to the blockchain
func (bc *BlockChain) AddBlock(block *consensus.Block) error {
	bc.chainMutex.Lock()
	defer bc.chainMutex.Unlock()

	blockHash := string(block.Hash())

	// Check for duplicate
	if _, exists := bc.blocks[blockHash]; exists {
		return ErrDuplicateBlock
	}

	// Get parent block
	parentHash := string(block.Header.PrevBlockHash)
	parent, exists := bc.blocks[parentHash]
	if !exists {
		return fmt.Errorf("%w: parent not found", ErrInvalidParent)
	}

	// Validate block
	if err := bc.validator.ValidateBlock(block, parent); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// Execute transactions
	executor := state.NewTransactionExecutor(bc.stateDB)
	receipts := make([]*state.Receipt, 0, len(block.Transactions))

	for _, tx := range block.Transactions {
		// Validate transaction
		if err := bc.validator.ValidateTransaction(tx); err != nil {
			return fmt.Errorf("transaction validation failed: %w", err)
		}

		// Execute transaction
		blockReward := consensus.CalculateBlockReward(block.Header.Height)
		receipt, err := executor.ExecuteTransaction(tx, blockReward)
		if err != nil {
			return fmt.Errorf("transaction execution failed: %w", err)
		}

		receipts = append(receipts, receipt)
	}

	// Check state root
	expectedStateRoot := bc.stateDB.CurrentRoot()
	if !stringEqual(block.Header.StateRoot, expectedStateRoot) {
		return ErrStateRootMismatch
	}

	// Add block to chain
	bc.blocks[blockHash] = block
	bc.heightMap[block.Header.Height] = append(bc.heightMap[block.Header.Height], block)

	// Update chain head if longer
	if block.Header.Height > bc.chainHead.Header.Height {
		// Check if this creates a fork
		if !stringEqual(block.Header.PrevBlockHash, bc.chainHead.Hash()) {
			// Fork detected - need to select longer chain
			if err := bc.selectBestChain(block); err != nil {
				// Rollback state
				bc.rollbackState(parent)
				return err
			}
		}

		bc.chainHead = block
	}

	// Distribute block reward
	if bc.rewardDist != nil {
		if err := bc.rewardDist.DistributeBlockReward(block, bc.stateDB); err != nil {
			bc.rollbackState(parent)
			return fmt.Errorf("reward distribution failed: %w", err)
		}
	}

	return nil
}

// selectBestChain selects the best chain when fork is detected
func (bc *BlockChain) selectBestChain(newBlock *consensus.Block) error {
	// Get the two fork points
	forkPoint := bc.findForkPoint(newBlock)
	if forkPoint == nil {
		return ErrForkDetected
	}

	// Calculate total difficulty for both chains
	newChainDifficulty := bc.calculateChainDifficulty(newBlock, forkPoint.Header.Height)
	currentChainDifficulty := bc.calculateChainDifficulty(bc.chainHead, forkPoint.Header.Height)

	// Select chain with higher difficulty
	if newChainDifficulty > currentChainDifficulty {
		// Reorg to new chain
		return bc.reorgToNewChain(newBlock, forkPoint)
	}

	// Keep current chain
	return nil
}

// findForkPoint finds the common ancestor of two blocks
func (bc *BlockChain) findForkPoint(block *consensus.Block) *consensus.Block {
	current := block

	for current != nil {
		currentHash := string(current.Hash())
		if _, exists := bc.blocks[currentHash]; exists {
			// Check if this block is in the current chain
			if bc.isInCurrentChain(current) {
				return current
			}
		}

		// Move to parent
		parentHash := string(current.Header.PrevBlockHash)
		current = bc.blocks[parentHash]
	}

	return nil
}

// isInCurrentChain checks if a block is in the current main chain
func (bc *BlockChain) isInCurrentChain(block *consensus.Block) bool {
	current := bc.chainHead
	for current != nil {
		if stringEqual(current.Hash(), block.Hash()) {
			return true
		}
		parentHash := string(current.Header.PrevBlockHash)
		current = bc.blocks[parentHash]
	}
	return false
}

// calculateChainDifficulty calculates total difficulty from block back to given height
func (bc *BlockChain) calculateChainDifficulty(block *consensus.Block, startHeight uint64) uint64 {
	difficulty := uint64(0)
	current := block

	for current != nil && current.Header.Height >= startHeight {
		// Add block difficulty (for now, use block number as proxy)
		difficulty += 1
		parentHash := string(current.Header.PrevBlockHash)
		current = bc.blocks[parentHash]
	}

	return difficulty
}

// reorgToNewChain reorganizes the blockchain to the new chain
func (bc *BlockChain) reorgToNewChain(newBlock *consensus.Block, forkPoint *consensus.Block) error {
	// Get blocks to remove from old chain
	toRemove := bc.getBlocksAfter(forkPoint)

	// Rollback state for removed blocks
	for _, block := range toRemove {
		if err := bc.rollbackState(block); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
	}

	// Get blocks to add from new chain
	toAdd := bc.getBlocksFromFork(forkPoint, newBlock)

	// Execute transactions for new blocks
	for _, block := range toAdd {
		executor := state.NewTransactionExecutor(bc.stateDB)
		for _, tx := range block.Transactions {
			if _, err := executor.ExecuteTransaction(tx, 0); err != nil {
				return fmt.Errorf("execution failed: %w", err)
			}
		}
	}

	// Update chain head
	bc.chainHead = newBlock

	return nil
}

// RollbackState rolls back state changes for a block (exported)
func (bc *BlockChain) RollbackState(block *consensus.Block) error {
	return bc.rollbackState(block)
}

// rollbackState rolls back state changes for a block
func (bc *BlockChain) rollbackState(block *consensus.Block) error {
	// For now, this is a placeholder
	// In a full implementation, we would need state snapshots
	// to rollback to previous state
	return nil
}

// getBlocksAfter returns all blocks after the given fork point
func (bc *BlockChain) getBlocksAfter(forkPoint *consensus.Block) []*consensus.Block {
	blocks := make([]*consensus.Block, 0)
	current := bc.chainHead

	for current != nil && !stringEqual(current.Hash(), forkPoint.Hash()) {
		blocks = append(blocks, current)
		parentHash := string(current.Header.PrevBlockHash)
		current = bc.blocks[parentHash]
	}

	return blocks
}

// getBlocksFromFork returns blocks from fork point to new block
func (bc *BlockChain) getBlocksFromFork(forkPoint *consensus.Block, newBlock *consensus.Block) []*consensus.Block {
	blocks := make([]*consensus.Block, 0)
	current := newBlock

	for current != nil && !stringEqual(current.Hash(), forkPoint.Hash()) {
		blocks = append([]*consensus.Block{current}, blocks...) // Prepend
		parentHash := string(current.Header.PrevBlockHash)
		current = bc.blocks[parentHash]
	}

	return blocks
}

// GetBlock returns a block by hash
func (bc *BlockChain) GetBlock(hash []byte) *consensus.Block {
	bc.chainMutex.RLock()
	defer bc.chainMutex.RUnlock()

	return bc.blocks[string(hash)]
}

// GetBlockByHeight returns blocks at given height (may return multiple if fork)
func (bc *BlockChain) GetBlockByHeight(height uint64) []*consensus.Block {
	bc.chainMutex.RLock()
	defer bc.chainMutex.RUnlock()

	return bc.heightMap[height]
}

// GetChainHead returns the current head of the chain
func (bc *BlockChain) GetChainHead() *consensus.Block {
	bc.chainMutex.RLock()
	defer bc.chainMutex.RUnlock()

	return bc.chainHead
}

// GetCurrentHeight returns the current block height
func (bc *BlockChain) GetCurrentHeight() uint64 {
	bc.chainMutex.RLock()
	defer bc.chainMutex.RUnlock()

	return bc.chainHead.Header.Height
}

// GetGenesisBlock returns the genesis block
func (bc *BlockChain) GetGenesisBlock() *consensus.Block {
	bc.chainMutex.RLock()
	defer bc.chainMutex.RUnlock()

	return bc.genesisBlock
}

// SetChainHead sets the chain head to the given block
func (bc *BlockChain) SetChainHead(block *consensus.Block) {
	bc.chainMutex.Lock()
	defer bc.chainMutex.Unlock()
	bc.chainHead = block
}

// ExecuteBlock executes all transactions in a block
func (bc *BlockChain) ExecuteBlock(block *consensus.Block) error {
	executor := state.NewTransactionExecutor(bc.stateDB)
	for _, tx := range block.Transactions {
		if _, err := executor.ExecuteTransaction(tx, 0); err != nil {
			return err
		}
	}
	return nil
}

// stringEqual compares two byte slices for equality
func stringEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// NewBlockValidator creates a new block validator
func NewBlockValidator() *BlockValidator {
	return &BlockValidator{}
}

// BlockValidator implements Validator interface
type BlockValidator struct{}

// ValidateBlock validates a block
func (v *BlockValidator) ValidateBlock(block *consensus.Block, parent *consensus.Block) error {
	// Check version
	if block.Header.Version != 0x0100 {
		return fmt.Errorf("invalid block version: %d", block.Header.Version)
	}

	// Check height
	if block.Header.Height != parent.Header.Height+1 {
		return fmt.Errorf("invalid block height: expected %d, got %d", parent.Header.Height+1, block.Header.Height)
	}

	// Check parent hash
	if !stringEqual(block.Header.PrevBlockHash, parent.Hash()) {
		return ErrInvalidParent
	}

	// Check timestamp (must be after parent)
	if block.Header.Timestamp <= parent.Header.Timestamp {
		return errors.New("block timestamp must be after parent")
	}

	// Check timestamp (must not be too far in the future)
	if block.Header.Timestamp > uint64(time.Now().Add(30*time.Second).Unix()) {
		return errors.New("block timestamp too far in the future")
	}

	// Check merkle root
	computedMerkleRoot := consensus.ComputeMerkleRoot(block.Transactions)
	if !stringEqual(block.Header.MerkleRoot, computedMerkleRoot) {
		return errors.New("merkle root mismatch")
	}

	// Validate all transactions
	for _, tx := range block.Transactions {
		if err := v.ValidateTransaction(tx); err != nil {
			return fmt.Errorf("invalid transaction: %w", err)
		}
	}

	return nil
}

// ValidateTransaction validates a transaction
func (v *BlockValidator) ValidateTransaction(tx *consensus.Transaction) error {
	// Check version
	if tx.Version != 0x0100 {
		return fmt.Errorf("invalid transaction version: %d", tx.Version)
	}

	// Check addresses
	if len(tx.From) != 16 {
		return errors.New("invalid from address length")
	}
	if len(tx.To) != 16 {
		return errors.New("invalid to address length")
	}

	// Check amount
	if tx.Amount == 0 {
		return errors.New("amount must be > 0")
	}

	// Check signature
	if len(tx.Signature) != 64 {
		return errors.New("invalid signature length")
	}

	return nil
}

// IsValidBlockHash checks if a block hash is valid
func (v *BlockValidator) IsValidBlockHash(hash []byte) bool {
	return len(hash) == 32
}
