package blockchain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

const (
	ForkDetectionWindow = 100 // Number of blocks to look back for forks
	MaxReorgDepth       = 50  // Maximum allowed reorganization depth
	MinForkWorkDiff     = 1   // Minimum work difference to trigger reorg
)

// Fork represents a blockchain fork
type Fork struct {
	NewBlock    *consensus.Block
	OldBlock    *consensus.Block
	ForkPoint   *consensus.Block
	NewChainLen int
	OldChainLen int
	Difficulty  uint64
	DetectedAt  time.Time
}

// ForkHandler handles fork detection and resolution
type ForkHandler struct {
	mu              sync.RWMutex
	forks           []Fork
	maxForkDepth    int
	enabled         bool
	chain           *BlockChain
	reorgInProgress bool
}

// NewForkHandler creates a new fork handler
func NewForkHandler(chain *BlockChain) *ForkHandler {
	return &ForkHandler{
		forks:        make([]Fork, 0),
		maxForkDepth: MaxReorgDepth,
		enabled:      true,
		chain:        chain,
	}
}

// CheckForFork checks if adding a block would create a fork
func (fh *ForkHandler) CheckForFork(block *consensus.Block) (*Fork, error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	if !fh.enabled {
		return nil, nil
	}

	chainHead := fh.chain.GetChainHead()

	// Check if this block extends the current chain
	if stringEqual(block.Header.PrevBlockHash, chainHead.Hash()) {
		// No fork
		return nil, nil
	}

	// Check if this block's parent exists
	parent := fh.chain.GetBlock(block.Header.PrevBlockHash)
	if parent == nil {
		return nil, nil // Parent not found yet
	}

	// Find fork point
	forkPoint := fh.findForkPoint(block)
	if forkPoint == nil {
		return nil, nil // Cannot determine fork point yet
	}

	// Check if fork depth is acceptable
	forkDepth := chainHead.Header.Height - forkPoint.Header.Height
	if int(forkDepth) > fh.maxForkDepth {
		return nil, ErrInvalidChain // Fork too deep, reject
	}

	// Calculate work for both chains
	newChainWork := fh.calculateChainWork(block, forkPoint.Header.Height)
	oldChainWork := fh.calculateChainWork(chainHead, forkPoint.Header.Height)

	// Create fork record
	fork := Fork{
		NewBlock:    block,
		OldBlock:    chainHead,
		ForkPoint:   forkPoint,
		NewChainLen: int(block.Header.Height - forkPoint.Header.Height),
		OldChainLen: int(chainHead.Header.Height - forkPoint.Header.Height),
		Difficulty:  newChainWork - oldChainWork,
		DetectedAt:  time.Now(),
	}

	// Only trigger reorg if new chain has more work
	if fork.Difficulty >= MinForkWorkDiff {
		fh.forks = append(fh.forks, fork)
		return &fork, nil
	}

	// Keep the longer chain (same work, but longer)
	if fork.NewChainLen > fork.OldChainLen {
		fh.forks = append(fh.forks, fork)
		return &fork, nil
	}

	return nil, nil
}

// findForkPoint finds the common ancestor of two blocks
func (fh *ForkHandler) findForkPoint(block *consensus.Block) *consensus.Block {
	current := block
	visited := make(map[string]bool)

	// Walk back from new block
	for current != nil && current.Header.Height >= fh.chain.GetChainHead().Header.Height-uint64(fh.maxForkDepth) {
		currentHash := string(current.Hash())
		visited[currentHash] = true

		parentHash := string(current.Header.PrevBlockHash)
		current = fh.chain.GetBlock([]byte(parentHash))
	}

	// Walk back from current chain head
	chainHead := fh.chain.GetChainHead()
	current = chainHead

	for current != nil {
		currentHash := string(current.Hash())
		if visited[currentHash] {
			return current
		}

		parentHash := string(current.Header.PrevBlockHash)
		current = fh.chain.GetBlock([]byte(parentHash))
	}

	return nil
}

// calculateChainWork calculates total work from block back to given height
func (fh *ForkHandler) calculateChainWork(block *consensus.Block, startHeight uint64) uint64 {
	work := uint64(0)
	current := block

	for current != nil && current.Header.Height >= startHeight {
		// Work is proportional to block reward (proof of work proxy)
		reward := consensus.CalculateBlockReward(current.Header.Height)
		work += reward

		parentHash := string(current.Header.PrevBlockHash)
		current = fh.chain.GetBlock([]byte(parentHash))
	}

	return work
}

// ResolveFork resolves a detected fork
func (fh *ForkHandler) ResolveFork(fork *Fork) error {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	if fh.reorgInProgress {
		return errors.New("reorg already in progress")
	}

	fh.reorgInProgress = true
	defer func() { fh.reorgInProgress = false }()

	// Determine which chain to keep
	if fork.Difficulty >= MinForkWorkDiff || fork.NewChainLen > fork.OldChainLen {
		// Switch to new chain
		if err := fh.switchToNewChain(fork); err != nil {
			return err
		}
	}

	// Remove fork from history
	for i, f := range fh.forks {
		if f.NewBlock == fork.NewBlock {
			fh.forks = append(fh.forks[:i], fh.forks[i+1:]...)
			break
		}
	}

	return nil
}

// switchToNewChain switches to the new chain after a fork
func (fh *ForkHandler) switchToNewChain(fork *Fork) error {
	// Get blocks to remove from old chain
	toRemove := fh.getBlocksAfter(fork.ForkPoint)

	// Rollback state for removed blocks
	for _, block := range toRemove {
		if err := fh.chain.RollbackState(block); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
	}

	// Get blocks to add from new chain
	toAdd := fh.getBlocksFromFork(fork.ForkPoint, fork.NewBlock)

	// Execute transactions for new blocks
	for _, block := range toAdd {
		if err := fh.chain.ExecuteBlock(block); err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}
	}

	// Update chain head
	fh.chain.SetChainHead(fork.NewBlock)

	return nil
}

// getBlocksAfter returns all blocks after the given fork point
func (fh *ForkHandler) getBlocksAfter(forkPoint *consensus.Block) []*consensus.Block {
	blocks := make([]*consensus.Block, 0)
	current := fh.chain.GetChainHead()

	for current != nil && !stringEqual(current.Hash(), forkPoint.Hash()) {
		blocks = append(blocks, current)
		parentHash := string(current.Header.PrevBlockHash)
		current = fh.chain.GetBlock([]byte(parentHash))
	}

	return blocks
}

// getBlocksFromFork returns blocks from fork point to new block
func (fh *ForkHandler) getBlocksFromFork(forkPoint *consensus.Block, newBlock *consensus.Block) []*consensus.Block {
	blocks := make([]*consensus.Block, 0)
	current := newBlock

	for current != nil && !stringEqual(current.Hash(), forkPoint.Hash()) {
		blocks = append([]*consensus.Block{current}, blocks...) // Prepend
		parentHash := string(current.Header.PrevBlockHash)
		current = fh.chain.GetBlock([]byte(parentHash))
	}

	return blocks
}

// GetRecentForks returns recent forks
func (fh *ForkHandler) GetRecentForks() []Fork {
	fh.mu.RLock()
	defer fh.mu.RUnlock()

	return fh.forks
}

// ClearOldForks removes forks older than specified duration
func (fh *ForkHandler) ClearOldForks(maxAge time.Duration) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	now := time.Now()
	filtered := make([]Fork, 0)

	for _, fork := range fh.forks {
		if now.Sub(fork.DetectedAt) <= maxAge {
			filtered = append(filtered, fork)
		}
	}

	fh.forks = filtered
}

// IsReorgInProgress checks if a reorganization is in progress
func (fh *ForkHandler) IsReorgInProgress() bool {
	fh.mu.RLock()
	defer fh.mu.RUnlock()

	return fh.reorgInProgress
}

// Enable enables fork detection
func (fh *ForkHandler) Enable() {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.enabled = true
}

// Disable disables fork detection
func (fh *ForkHandler) Disable() {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.enabled = false
}
