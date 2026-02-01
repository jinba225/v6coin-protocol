package state

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

var (
	// Execution errors
	ErrInsufficientBalance = errors.New("insufficient balance for transaction")
	ErrInvalidNonce        = errors.New("invalid account nonce")
	ErrInvalidFromAccount  = errors.New("from account does not exist")
	ErrInvalidToAccount    = errors.New("to account does not exist")
	ErrContractExecution   = errors.New("contract execution error")
	ErrInvalidValue        = errors.New("invalid value transfer")
	ErrGasLimitExceeded    = errors.New("gas limit exceeded")
)

// Receipt represents a transaction execution receipt
type Receipt struct {
	TxHash          []byte        // Hash of the transaction
	Status          ReceiptStatus // Execution status
	GasUsed         uint64        // Gas used during execution
	Logs            []LogEntry    // Contract logs
	ContractAddress []byte        // Address of created contract (if any)
}

// ReceiptStatus represents the status of a transaction
type ReceiptStatus uint8

const (
	ReceiptStatusSuccess ReceiptStatus = 0x01
	ReceiptStatusFailure ReceiptStatus = 0x02
	ReceiptStatusPending ReceiptStatus = 0x03
)

// LogEntry represents a contract log entry
type LogEntry struct {
	Address []byte   // Contract address that emitted the log
	Topics  [][]byte // Array of topics
	Data    []byte   // Log data
}

// ExecutionResult represents the result of transaction execution
type ExecutionResult struct {
	Success      bool
	GasUsed      uint64
	Logs         []LogEntry
	NewStateRoot []byte
	Error        error
}

// StateTransition represents a state transition during transaction execution
type StateTransition struct {
	DB          StateDB
	FromAccount *Account
	ToAccount   *Account
	BlockReward uint64
}

// TransactionExecutor executes transactions and updates state
type TransactionExecutor struct {
	DB          StateDB
	blockReward uint64
	mu          sync.RWMutex
}

// NewTransactionExecutor creates a new transaction executor
func NewTransactionExecutor(db StateDB) *TransactionExecutor {
	return &TransactionExecutor{
		DB:          db,
		blockReward: 0,
		mu:          sync.RWMutex{},
	}
}

// ExecuteTransaction executes a transaction and returns a receipt
func (ex *TransactionExecutor) ExecuteTransaction(tx *consensus.Transaction, blockReward uint64) (*Receipt, error) {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	ex.blockReward = blockReward

	// Create state transition
	transition := &StateTransition{
		DB:          ex.DB,
		BlockReward: blockReward,
	}

	// Get from account
	fromAddr := tx.From.To16()
	fromAccount, err := ex.DB.GetAccount(fromAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get from account: %w", err)
	}

	if fromAccount == nil {
		// Create new account
		fromAccount = NewAccount()
		fromAccount.Balance = 10000000000000 // Initial balance for new accounts
	}

	// Check nonce
	if tx.Nonce != fromAccount.Nonce {
		return nil, fmt.Errorf("invalid nonce: expected %d, got %d", fromAccount.Nonce, tx.Nonce)
	}

	// Validate transaction
	if err := ex.ValidateTransaction(tx); err != nil {
		return nil, err
	}

	// Get to account
	toAddr := tx.To.To16()
	toAccount, err := ex.DB.GetAccount(toAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get to account: %w", err)
	}

	if toAccount == nil {
		// Create new account
		toAccount = NewAccount()
	}

	transition.FromAccount = fromAccount
	transition.ToAccount = toAccount

	// Execute transaction based on type
	var result *ExecutionResult

	switch tx.Type {
	case consensus.TxTypeOnline, consensus.TxTypeOffline:
		result, err = ex.executeValueTransfer(transition, tx)
	case consensus.TxTypeMigration:
		result, err = ex.executeMigration(transition, tx)
	case consensus.TxTypeStake:
		result, err = ex.executeStake(transition, tx)
	case consensus.TxTypeUnstake:
		result, err = ex.executeUnstake(transition, tx)
	case consensus.TxTypeGovernance:
		result, err = ex.executeGovernance(transition, tx)
	case consensus.TxTypeVote:
		result, err = ex.executeVote(transition, tx)
	default:
		return nil, fmt.Errorf("unsupported transaction type: %d", tx.Type)
	}

	if err != nil {
		receipt := &Receipt{
			TxHash:          tx.Hash(),
			Status:          ReceiptStatusFailure,
			GasUsed:         0,
			Logs:            []LogEntry{},
			ContractAddress: []byte{},
		}
		return receipt, err
	}

	// Update accounts in database
	if err := ex.DB.SetAccount(fromAddr, fromAccount); err != nil {
		return nil, fmt.Errorf("failed to set from account: %w", err)
	}

	if err := ex.DB.SetAccount(toAddr, toAccount); err != nil {
		return nil, fmt.Errorf("failed to set to account: %w", err)
	}

	// Commit state changes
	_, err = ex.DB.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit state: %w", err)
	}

	// Return receipt
	receipt := &Receipt{
		TxHash:          tx.Hash(),
		Status:          ReceiptStatusSuccess,
		GasUsed:         result.GasUsed,
		Logs:            result.Logs,
		ContractAddress: []byte{},
	}

	return receipt, nil
}

// executeValueTransfer executes a value transfer transaction
func (ex *TransactionExecutor) executeValueTransfer(transition *StateTransition, tx *consensus.Transaction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: 21000, // Base gas for transfer
		Logs:    []LogEntry{},
	}

	// Calculate total cost (amount + fee)
	totalCost := new(big.Int).SetUint64(tx.Amount)
	totalCost.Add(totalCost, new(big.Int).SetUint64(tx.Fee))

	// Check balance
	balance := new(big.Int).SetUint64(transition.FromAccount.Balance)
	if balance.Cmp(totalCost) < 0 {
		return nil, ErrInsufficientBalance
	}

	// Subtract from balance
	transition.FromAccount.SubBalance(totalCost.Uint64())

	// Add to balance
	transition.ToAccount.AddBalance(tx.Amount)

	result.NewStateRoot = transition.DB.CurrentRoot()
	return result, nil
}

// executeMigration executes an account migration transaction
func (ex *TransactionExecutor) executeMigration(transition *StateTransition, tx *consensus.Transaction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: 40000, // Higher gas for migration
		Logs:    []LogEntry{},
	}

	// Verify migration proof
	// TODO: Implement migration proof verification

	// Create new account at destination
	toAddr := tx.To.To16()
	newAccount := NewAccount()
	newAccount.Balance = transition.FromAccount.Balance
	newAccount.CodeHash = transition.FromAccount.CodeHash
	newAccount.StorageRoot = transition.FromAccount.StorageRoot

	// Empty source account
	transition.FromAccount.Balance = 0
	transition.FromAccount.CodeHash = []byte{}
	transition.FromAccount.StorageRoot = []byte{}

	// Set accounts
	if err := transition.DB.SetAccount(toAddr, newAccount); err != nil {
		return nil, fmt.Errorf("failed to set to account: %w", err)
	}

	result.NewStateRoot = transition.DB.CurrentRoot()
	return result, nil
}

// executeStake executes a staking transaction
func (ex *TransactionExecutor) executeStake(transition *StateTransition, tx *consensus.Transaction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: 50000, // Base gas for staking
		Logs:    []LogEntry{},
	}

	// Check balance
	balance := new(big.Int).SetUint64(transition.FromAccount.Balance)
	stakeAmount := new(big.Int).SetUint64(tx.Amount)
	if balance.Cmp(stakeAmount) < 0 {
		return nil, ErrInsufficientBalance
	}

	// Stake: subtract from balance and lock
	transition.FromAccount.SubBalance(stakeAmount.Uint64())
	// Note: Staked amount would be tracked separately

	// Add stake to total staked pool
	// TODO: Implement staking pool management

	result.NewStateRoot = transition.DB.CurrentRoot()
	return result, nil
}

// executeUnstake executes an unstaking transaction
func (ex *TransactionExecutor) executeUnstake(transition *StateTransition, tx *consensus.Transaction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: 50000, // Base gas for unstaking
		Logs:    []LogEntry{},
	}

	// Verify unstake conditions
	// TODO: Implement unstake period and eligibility check

	// Unstake: return staked amount with rewards
	toAddr := tx.To.To16()
	toAccount, err := transition.DB.GetAccount(toAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get to account: %w", err)
	}

	if toAccount == nil {
		// Create new account
		toAccount = NewAccount()
	}

	// TODO: Calculate and add staking rewards
	unstakeAmount := tx.Amount
	reward := uint64(0) // Calculate based on staking duration

	// Add unstaked amount and rewards to account
	totalAmount := new(big.Int).SetUint64(unstakeAmount)
	totalAmount.Add(totalAmount, new(big.Int).SetUint64(reward))

	toAccount.AddBalance(totalAmount.Uint64())

	// Update to account in DB
	if err := transition.DB.SetAccount(toAddr, toAccount); err != nil {
		return nil, fmt.Errorf("failed to set to account: %w", err)
	}

	result.NewStateRoot = transition.DB.CurrentRoot()
	return result, nil
}

// executeGovernance executes a governance proposal transaction
func (ex *TransactionExecutor) executeGovernance(transition *StateTransition, tx *consensus.Transaction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: 100000, // Higher gas for governance
		Logs:    []LogEntry{},
	}

	// Check governance requirements
	// TODO: Implement governance proposal requirements check

	// Validate proposal format and content
	if len(tx.Data) == 0 {
		return nil, errors.New("governance proposal data required")
	}

	// Create governance proposal record
	// TODO: Implement governance proposal storage

	result.NewStateRoot = transition.DB.CurrentRoot()
	return result, nil
}

// executeVote executes a governance vote transaction
func (ex *TransactionExecutor) executeVote(transition *StateTransition, tx *consensus.Transaction) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: 5000, // Lower gas for voting
		Logs:    []LogEntry{},
	}

	// Validate vote
	// TODO: Implement vote validation

	// Record vote for proposal
	// TODO: Implement vote recording

	result.NewStateRoot = transition.DB.CurrentRoot()
	return result, nil
}

// ApplyBlockReward applies block reward to coinbase account
func (ex *TransactionExecutor) ApplyBlockReward(coinbase []byte, reward uint64) error {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	// Get or create coinbase account
	account, err := ex.DB.GetAccount(coinbase)
	if err != nil {
		return fmt.Errorf("failed to get coinbase account: %w", err)
	}

	if account == nil {
		// Create new coinbase account
		account = NewAccount()
	}

	// Add reward to coinbase
	account.AddBalance(reward)

	// Update account in database
	if err := ex.DB.SetAccount(coinbase, account); err != nil {
		return fmt.Errorf("failed to set coinbase account: %w", err)
	}

	return nil
}

// ValidateTransaction validates a transaction before execution
func (ex *TransactionExecutor) ValidateTransaction(tx *consensus.Transaction) error {
	// Check transaction hash
	if len(tx.Hash()) == 0 {
		return errors.New("invalid transaction hash")
	}

	// Check addresses
	if len(tx.From) != 16 {
		return errors.New("invalid from address length")
	}

	if len(tx.To) != 16 {
		return errors.New("invalid to address length")
	}

	// Check amounts
	if tx.Amount == 0 {
		return ErrInvalidValue
	}

	if tx.Fee == 0 && tx.Type != consensus.TxTypeOnline {
		// Some transactions require minimum fee
		return errors.New("transaction fee too low")
	}

	// Check data size
	if len(tx.Data) > 1024*1024 {
		return errors.New("transaction data too large (max 1MB)")
	}

	return nil
}

// CalculateGas calculates gas required for transaction
func (ex *TransactionExecutor) CalculateGas(tx *consensus.Transaction) uint64 {
	baseGas := uint64(21000)

	switch tx.Type {
	case consensus.TxTypeOnline, consensus.TxTypeOffline:
		return baseGas + uint64(len(tx.Data))*2
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

// EstimateTransactionResult estimates the result of a transaction without executing it
func (ex *TransactionExecutor) EstimateTransactionResult(tx *consensus.Transaction, fromBalance uint64) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: true,
		GasUsed: ex.CalculateGas(tx),
		Logs:    []LogEntry{},
	}

	// Check if balance is sufficient
	totalCost := tx.Amount + tx.Fee
	if fromBalance < totalCost {
		return nil, ErrInsufficientBalance
	}

	result.NewStateRoot = ex.DB.CurrentRoot()
	return result, nil
}

// GetTotalFeesCollected returns total fees collected in the current block
func (ex *TransactionExecutor) GetTotalFeesCollected() uint64 {
	ex.mu.RLock()
	defer ex.mu.RUnlock()

	return ex.blockReward
}

// ResetFees resets collected fees
func (ex *TransactionExecutor) ResetFees() {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	ex.blockReward = 0
}
