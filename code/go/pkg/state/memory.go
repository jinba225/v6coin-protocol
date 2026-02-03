package state

import (
	"sync"
)

// MemoryStateDB is an in-memory implementation of StateDB
type MemoryStateDB struct {
	accounts map[string]*Account
	mu       sync.RWMutex
	rootHash []byte
}

// NewMemoryStateDB creates a new in-memory state database
func NewMemoryStateDB() *MemoryStateDB {
	return &MemoryStateDB{
		accounts: make(map[string]*Account),
		mu:       sync.RWMutex{},
		rootHash: make([]byte, 32),
	}
}

// GetAccount retrieves an account by address
func (db *MemoryStateDB) GetAccount(address []byte) (*Account, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	account, exists := db.accounts[string(address)]
	if !exists {
		return nil, nil // Account doesn't exist, return nil (not an error)
	}

	return account.Copy(), nil
}

// SetAccount stores or updates an account
func (db *MemoryStateDB) SetAccount(address []byte, account *Account) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if account == nil {
		delete(db.accounts, string(address))
		return nil
	}

	// Store a copy of the account
	db.accounts[string(address)] = account.Copy()

	// Update root hash (simplified)
	db.updateRootHash()

	return nil
}

// GetBalance retrieves account balance
func (db *MemoryStateDB) GetBalance(address []byte) (uint64, error) {
	account, err := db.GetAccount(address)
	if err != nil {
		return 0, err
	}
	if account == nil {
		return 0, nil // New account has 0 balance
	}
	return account.Balance, nil
}

// GetNonce retrieves account nonce
func (db *MemoryStateDB) GetNonce(address []byte) (uint64, error) {
	account, err := db.GetAccount(address)
	if err != nil {
		return 0, err
	}
	if account == nil {
		return 0, nil // New account has 0 nonce
	}
	return account.Nonce, nil
}

// HasAccount checks if an account exists
func (db *MemoryStateDB) HasAccount(address []byte) (bool, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	_, exists := db.accounts[string(address)]
	return exists, nil
}

// CurrentRoot returns the current state root hash
func (db *MemoryStateDB) CurrentRoot() []byte {
	db.mu.RLock()
	defer db.mu.RUnlock()

	rootHash := make([]byte, len(db.rootHash))
	copy(rootHash, db.rootHash)
	return rootHash
}

// Commit commits state changes and returns new root hash
func (db *MemoryStateDB) Commit() ([]byte, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// In memory implementation, changes are already applied
	// Just return the current root hash
	rootHash := make([]byte, len(db.rootHash))
	copy(rootHash, db.rootHash)
	return rootHash, nil
}

// Close closes the state database
func (db *MemoryStateDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Clear accounts
	db.accounts = make(map[string]*Account)
	db.rootHash = make([]byte, 32)
	return nil
}

// updateRootHash updates the state root hash
// This is a simplified implementation - in production, this would use a Merkle Patricia Trie
func (db *MemoryStateDB) updateRootHash() {
	// Simplified: hash all account balances
	// In production, use MPT for proper state root calculation
	// For now, just keep the existing hash or generate a new one

	// Create a 32-byte root hash based on account count
	newRootHash := make([]byte, 32)
	count := len(db.accounts)
	newRootHash[0] = byte(count >> 24)
	newRootHash[1] = byte(count >> 16)
	newRootHash[2] = byte(count >> 8)
	newRootHash[3] = byte(count)
	db.rootHash = newRootHash
}

// GetAccountCount returns the number of accounts
func (db *MemoryStateDB) GetAccountCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return len(db.accounts)
}

// GetAllAccounts returns all accounts
func (db *MemoryStateDB) GetAllAccounts() map[string]*Account {
	db.mu.RLock()
	defer db.mu.RUnlock()

	accounts := make(map[string]*Account)
	for addr, acc := range db.accounts {
		accounts[addr] = acc.Copy()
	}
	return accounts
}

// Reset resets the state database
func (db *MemoryStateDB) Reset() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.accounts = make(map[string]*Account)
	db.rootHash = make([]byte, 32)
	return nil
}
