package tx

import (
	"container/heap"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

const (
	MaxPoolSize       = 10000              // Maximum number of transactions in pool
	MaxTTL            = 7 * 24 * time.Hour // Max time-to-live for transactions (7 days)
	MinFeePerByte     = 1                  // Minimum fee per byte (in nano-V6)
	BaseTxSize        = 100                // Base transaction size in bytes
	PriorityThreshold = 0.6                // Threshold for high-priority transactions
)

var (
	ErrPoolFull            = ErrInvalidTransaction("transaction pool is full")
	ErrDuplicateTx         = ErrInvalidTransaction("duplicate transaction")
	ErrTxTooOld            = ErrInvalidTransaction("transaction too old")
	ErrInvalidFee          = ErrInvalidTransaction("invalid transaction fee")
	ErrUnknownSender       = ErrInvalidTransaction("unknown sender account")
	ErrInsufficientBalance = ErrInvalidTransaction("insufficient balance")
	ErrInvalidNonce        = ErrInvalidTransaction("invalid nonce")
)

// ErrInvalidTransaction represents an invalid transaction error
type ErrInvalidTransaction string

func (e ErrInvalidTransaction) Error() string {
	return string(e)
}

// TxPoolItem represents a transaction in the pool with metadata
type TxPoolItem struct {
	Tx          *consensus.Transaction
	AddedAt     time.Time
	Priority    float64
	GasLimit    uint64
	GasPrice    uint64
	Nonce       uint64
	FromAddress string
	Size        int
	index       int // For heap interface
}

// TxPriorityQueue implements heap.Interface for prioritized transactions
type TxPriorityQueue []*TxPoolItem

func (pq TxPriorityQueue) Len() int { return len(pq) }

func (pq TxPriorityQueue) Less(i, j int) bool {
	// Higher priority first
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority > pq[j].Priority
	}
	// If same priority, higher gas price first
	if pq[i].GasPrice != pq[j].GasPrice {
		return pq[i].GasPrice > pq[j].GasPrice
	}
	// If same gas price, older transactions first
	return pq[i].AddedAt.Before(pq[j].AddedAt)
}

func (pq TxPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *TxPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TxPoolItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *TxPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// TxPool manages pending transactions
type TxPool struct {
	pending     TxPriorityQueue
	all         map[string]*TxPoolItem            // tx hash -> item
	bySender    map[string]map[uint64]*TxPoolItem // sender address -> nonce -> item
	mu          sync.RWMutex
	chainHeight uint64
	stateDB     StateDB // Interface for checking balances and nonces
}

// StateDB is interface for state operations
type StateDB interface {
	GetBalance(address []byte) (uint64, error)
	GetNonce(address []byte) (uint64, error)
	HasAccount(address []byte) (bool, error)
}

// NewTxPool creates a new transaction pool
func NewTxPool(stateDB StateDB) *TxPool {
	pq := make(TxPriorityQueue, 0)
	heap.Init(&pq)

	return &TxPool{
		pending:  pq,
		all:      make(map[string]*TxPoolItem),
		bySender: make(map[string]map[uint64]*TxPoolItem),
		stateDB:  stateDB,
	}
}

// AddTransaction adds a transaction to the pool
func (tp *TxPool) AddTransaction(tx *consensus.Transaction) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	txHash := string(tx.Hash())

	// Check for duplicate
	if _, exists := tp.all[txHash]; exists {
		return ErrDuplicateTx
	}

	// Validate transaction
	if err := tp.validateTx(tx); err != nil {
		return err
	}

	// Check pool size
	if tp.pending.Len() >= MaxPoolSize {
		// Try to evict lowest priority transaction
		if err := tp.evictLowestPriority(); err != nil {
			return ErrPoolFull
		}
	}

	// Calculate priority
	priority := tp.calculatePriority(tx)

	// Create pool item
	fromAddr := tx.From.String()
	item := &TxPoolItem{
		Tx:          tx,
		AddedAt:     time.Now(),
		Priority:    priority,
		GasLimit:    tp.calculateGasLimit(tx),
		GasPrice:    tp.calculateGasPrice(tx),
		Nonce:       tx.Nonce,
		FromAddress: fromAddr,
		Size:        tp.calculateSize(tx),
	}

	// Add to pool
	tp.all[txHash] = item
	heap.Push(&tp.pending, item)

	// Track by sender
	if tp.bySender[fromAddr] == nil {
		tp.bySender[fromAddr] = make(map[uint64]*TxPoolItem)
	}
	tp.bySender[fromAddr][tx.Nonce] = item

	return nil
}

// GetPendingTransactions returns pending transactions sorted by priority
func (tp *TxPool) GetPendingTransactions(maxCount int) []*consensus.Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	if maxCount <= 0 || maxCount > tp.pending.Len() {
		maxCount = tp.pending.Len()
	}

	txs := make([]*consensus.Transaction, 0, maxCount)
	for i := 0; i < maxCount && i < tp.pending.Len(); i++ {
		txs = append(txs, tp.pending[i].Tx)
	}

	return txs
}

// GetTransaction returns a transaction by hash
func (tp *TxPool) GetTransaction(hash []byte) *consensus.Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	item, exists := tp.all[string(hash)]
	if !exists {
		return nil
	}

	return item.Tx
}

// RemoveTransaction removes a transaction from the pool
func (tp *TxPool) RemoveTransaction(hash []byte) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	txHash := string(hash)
	item, exists := tp.all[txHash]
	if !exists {
		return
	}

	// Remove from heap
	heap.Remove(&tp.pending, item.index)

	// Remove from all map
	delete(tp.all, txHash)

	// Remove from sender map
	fromAddr := item.FromAddress
	if senderTxs, ok := tp.bySender[fromAddr]; ok {
		delete(senderTxs, item.Nonce)
		if len(senderTxs) == 0 {
			delete(tp.bySender, fromAddr)
		}
	}
}

// RemoveByNonce removes all transactions from a sender with nonce >= given nonce
func (tp *TxPool) RemoveByNonce(fromAddress []byte, nonce uint64) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	fromAddr := string(fromAddress)
	senderTxs, ok := tp.bySender[fromAddr]
	if !ok {
		return
	}

	// Find and remove transactions with nonce >= given nonce
	toRemove := make([]*TxPoolItem, 0)
	for n, item := range senderTxs {
		if n >= nonce {
			toRemove = append(toRemove, item)
		}
	}

	// Remove each transaction
	for _, item := range toRemove {
		txHash := string(item.Tx.Hash())
		heap.Remove(&tp.pending, item.index)
		delete(tp.all, txHash)
		delete(senderTxs, item.Nonce)
	}

	if len(senderTxs) == 0 {
		delete(tp.bySender, fromAddr)
	}
}

// UpdateChainHeight updates the current chain height
func (tp *TxPool) UpdateChainHeight(height uint64) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.chainHeight = height
}

// CleanExpired removes expired transactions
func (tp *TxPool) CleanExpired() int {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for hash, item := range tp.all {
		if now.Sub(item.AddedAt) > MaxTTL {
			toRemove = append(toRemove, hash)
		}
	}

	for _, hash := range toRemove {
		item := tp.all[hash]
		heap.Remove(&tp.pending, item.index)
		delete(tp.all, hash)

		fromAddr := item.FromAddress
		if senderTxs, ok := tp.bySender[fromAddr]; ok {
			delete(senderTxs, item.Nonce)
			if len(senderTxs) == 0 {
				delete(tp.bySender, fromAddr)
			}
		}
	}

	return len(toRemove)
}

// Size returns the number of transactions in the pool
func (tp *TxPool) Size() int {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	return tp.pending.Len()
}

// validateTx validates a transaction before adding to pool
func (tp *TxPool) validateTx(tx *consensus.Transaction) error {
	// Check timestamp for offline transactions
	if tx.Type == consensus.TxTypeOffline {
		txTime := time.Unix(int64(tx.Timestamp), 0)
		if time.Since(txTime) > MaxTTL {
			return ErrTxTooOld
		}
	}

	// Check fee
	if tx.Fee < MinFeePerByte*uint64(len(tx.Data)+BaseTxSize) {
		return ErrInvalidFee
	}

	// Check sender exists and has sufficient balance
	hasAccount, err := tp.stateDB.HasAccount(tx.From)
	if err != nil {
		return err
	}
	if !hasAccount {
		return ErrUnknownSender
	}

	balance, err := tp.stateDB.GetBalance(tx.From)
	if err != nil {
		return err
	}

	totalCost := tx.Amount + tx.Fee
	if balance < totalCost {
		return ErrInsufficientBalance
	}

	// Check nonce
	currentNonce, err := tp.stateDB.GetNonce(tx.From)
	if err != nil {
		return err
	}

	if tx.Nonce < currentNonce {
		return ErrInvalidNonce
	}

	return nil
}

// calculatePriority calculates transaction priority
func (tp *TxPool) calculatePriority(tx *consensus.Transaction) float64 {
	// Priority based on fee, age, and size
	fee := float64(tx.Fee)
	size := float64(tp.calculateSize(tx))
	age := time.Since(time.Unix(int64(tx.Timestamp), 0)).Seconds() / float64(MaxTTL.Seconds())

	priority := (fee / size) * (1 + age*0.5)

	// Boost high-value transactions
	if tx.Amount > 1e12 { // > 1000 V6
		priority *= 1.5
	}

	return priority
}

// calculateGasLimit calculates gas limit for transaction
func (tp *TxPool) calculateGasLimit(tx *consensus.Transaction) uint64 {
	baseGas := uint64(21000)
	dataGas := uint64(len(tx.Data)) * 2

	switch tx.Type {
	case consensus.TxTypeOnline, consensus.TxTypeOffline:
		return baseGas + dataGas
	case consensus.TxTypeMigration:
		return 40000
	case consensus.TxTypeStake, consensus.TxTypeUnstake:
		return 50000
	case consensus.TxTypeGovernance:
		return 100000
	case consensus.TxTypeVote:
		return 5000
	default:
		return baseGas
	}
}

// calculateGasPrice calculates gas price (fee per gas unit)
func (tp *TxPool) calculateGasPrice(tx *consensus.Transaction) uint64 {
	gasLimit := tp.calculateGasLimit(tx)
	if gasLimit == 0 {
		return 0
	}
	return tx.Fee / gasLimit
}

// calculateSize calculates transaction size in bytes
func (tp *TxPool) calculateSize(tx *consensus.Transaction) int {
	size := BaseTxSize
	size += len(tx.Data)
	size += len(tx.Signature)
	return size
}

// evictLowestPriority removes the lowest priority transaction
func (tp *TxPool) evictLowestPriority() error {
	if tp.pending.Len() == 0 {
		return ErrPoolFull
	}

	item := heap.Pop(&tp.pending).(*TxPoolItem)
	txHash := string(item.Tx.Hash())

	delete(tp.all, txHash)

	fromAddr := item.FromAddress
	if senderTxs, ok := tp.bySender[fromAddr]; ok {
		delete(senderTxs, item.Nonce)
		if len(senderTxs) == 0 {
			delete(tp.bySender, fromAddr)
		}
	}

	return nil
}
