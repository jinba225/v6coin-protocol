package state

import (
	"time"
)

// StateDB is the interface for state database operations
type StateDB interface {
	// Account operations
	GetAccount(address []byte) (*Account, error)
	SetAccount(address []byte, account *Account) error

	// Convenience methods for account operations
	GetBalance(address []byte) (uint64, error)
	GetNonce(address []byte) (uint64, error)
	HasAccount(address []byte) (bool, error)

	// State operations
	CurrentRoot() []byte
	Commit() ([]byte, error)

	// Database operations
	Close() error
}

// Account represents an account in V6Coin state
type Account struct {
	Nonce       uint64    // Account nonce for transaction replay protection
	Balance     uint64    // Account balance in nano-V6
	CodeHash    []byte    // Hash of the account's smart contract code (if any)
	StorageRoot []byte    // Root of the account's storage trie
	LastUpdated time.Time // Last update timestamp
}

// NewAccount creates a new account
func NewAccount() *Account {
	return &Account{
		Nonce:       0,
		Balance:     0,
		CodeHash:    []byte{},
		StorageRoot: []byte{},
		LastUpdated: time.Now(),
	}
}

// Copy creates a copy of an account
func (a *Account) Copy() *Account {
	if a == nil {
		return nil
	}

	// Copy CodeHash and StorageRoot to avoid sharing underlying arrays
	codeHash := make([]byte, len(a.CodeHash))
	copy(codeHash, a.CodeHash)
	storageRoot := make([]byte, len(a.StorageRoot))
	copy(storageRoot, a.StorageRoot)

	return &Account{
		Nonce:       a.Nonce,
		Balance:     a.Balance,
		CodeHash:    codeHash,
		StorageRoot: storageRoot,
		LastUpdated: a.LastUpdated,
	}
}

// IsEmpty checks if an account is empty (has zero balance and no nonce)
func (a *Account) IsEmpty() bool {
	if a == nil {
		return true
	}
	return a.Nonce == 0 && a.Balance == 0 && len(a.CodeHash) == 0 && len(a.StorageRoot) == 0
}

// HasCode checks if an account has smart contract code
func (a *Account) HasCode() bool {
	if a == nil {
		return false
	}
	return len(a.CodeHash) > 0 && !isZeroHash(a.CodeHash)
}

// HasStorage checks if an account has storage
func (a *Account) HasStorage() bool {
	if a == nil {
		return false
	}
	return len(a.StorageRoot) > 0 && !isZeroHash(a.StorageRoot)
}

// isZeroHash checks if a hash is all zeros
func isZeroHash(hash []byte) bool {
	if hash == nil || len(hash) == 0 {
		return true
	}
	for _, b := range hash {
		if b != 0 {
			return false
		}
	}
	return true
}

// AddBalance adds amount to account balance
func (a *Account) AddBalance(amount uint64) {
	a.Balance += amount
	a.LastUpdated = time.Now()
}

// SubBalance subtracts amount from account balance
func (a *Account) SubBalance(amount uint64) error {
	if a.Balance < amount {
		return ErrInsufficientBalance
	}
	a.Balance -= amount
	a.LastUpdated = time.Now()
	return nil
}

// IncrementNonce increments account nonce
func (a *Account) IncrementNonce() uint64 {
	a.Nonce++
	a.LastUpdated = time.Now()
	return a.Nonce
}
