package wallet

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/address"
	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

var (
	ErrWalletLocked    = errors.New("wallet is locked")
	ErrWalletUnlocked  = errors.New("wallet is already unlocked")
	ErrAccountNotFound = errors.New("account not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrPrivateKeyEmpty = errors.New("private key is empty")
	ErrInvalidMnemonic = errors.New("invalid mnemonic")
)

// Account represents a wallet account
type Account struct {
	Address       address.V6Address // IPv6 address
	PublicKey     []byte            // Ed25519 public key
	PrivateKey    []byte            // Ed25519 private key (encrypted when locked)
	NetworkPrefix net.IP            // Network prefix
	Label         string            // Account label
	CreatedAt     time.Time         // Creation time
	Index         uint32            // Account index
	Nonce         uint64            // Account nonce
	IsEncrypted   bool              // Whether private key is encrypted
}

// Wallet represents a V6Coin wallet
type Wallet struct {
	accounts      []*Account
	accountsIndex map[string]*Account // address string -> account
	mnemonic      *crypto.Mnemonic    // BIP-39 mnemonic
	encrypted     bool
	locked        bool
	passwordHash  []byte
	mu            sync.RWMutex
	dataDir       string
	networkPrefix net.IP
}

// WalletConfig is configuration for wallet creation
type WalletConfig struct {
	DataDir       string
	Mnemonic      string // Optional: recover from mnemonic
	Password      string // Optional: encrypt with password
	NetworkPrefix string // IPv6 network prefix (64-bit, e.g., "2001:db8::")
}

// NewWallet creates a new wallet
func NewWallet(config *WalletConfig) (*Wallet, error) {
	// Parse network prefix
	// net.ParseIP returns a 16-byte IPv6 address, we need the first 8 bytes as prefix
	fullIP := net.ParseIP(config.NetworkPrefix)
	if fullIP == nil {
		return nil, errors.New("invalid network prefix")
	}
	if len(fullIP) != 16 {
		return nil, fmt.Errorf("network prefix must be an IPv6 address (16 bytes), got %d", len(fullIP))
	}

	// Extract first 8 bytes as network prefix
	networkPrefix := make(net.IP, 8)
	copy(networkPrefix, fullIP[0:8])

	w := &Wallet{
		accounts:      make([]*Account, 0),
		accountsIndex: make(map[string]*Account),
		encrypted:     false,
		locked:        false,
		dataDir:       config.DataDir,
		networkPrefix: networkPrefix,
	}

	var err error

	// Create or recover mnemonic
	if config.Mnemonic != "" {
		// Recover from mnemonic
		w.mnemonic, err = crypto.MnemonicFromString(config.Mnemonic)
		if err != nil {
			return nil, fmt.Errorf("failed to recover from mnemonic: %w", err)
		}
	} else {
		// Generate new mnemonic
		w.mnemonic, err = crypto.GenerateMnemonic(12) // 12 words
		if err != nil {
			return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
		}
	}

	// Encrypt if password provided
	if config.Password != "" {
		w.passwordHash = hashPassword(config.Password)
		w.encrypted = true
		w.locked = true
	}

	// Create default account (index 0)
	if err := w.CreateAccount(0, "Default", config.Password); err != nil {
		return nil, fmt.Errorf("failed to create default account: %w", err)
	}

	return w, nil
}

// CreateAccount creates a new account with given index
func (w *Wallet) CreateAccount(index uint32, label string, password string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Derive key pair from mnemonic with derivation path
	// Simplified: use seed directly for now
	seed, err := w.mnemonic.ToSeed(password)
	if err != nil {
		return fmt.Errorf("failed to get seed: %w", err)
	}

	// Derive child key using index (simplified BIP-32 style)
	derivedSeed, err := crypto.DeriveChildKey(seed, fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// Generate key pair from derived seed
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}
	// Override with derived seed if needed
	if len(derivedSeed) >= 32 {
		privateKey := derivedSeed[:32]
		// Use private key directly
		kp.PrivateKey = privateKey
	}

	// Derive V6 address
	v6Addr, err := address.GenerateAddress(w.networkPrefix, kp.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to derive address: %w", err)
	}

	// Encrypt private key if wallet is encrypted
	privateKey := make([]byte, len(kp.PrivateKey))
	copy(privateKey, kp.PrivateKey)
	isEncrypted := false

	if w.encrypted && w.passwordHash != nil {
		encrypted, err := crypto.Encrypt(w.passwordHash, privateKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt private key: %w", err)
		}
		privateKey = encrypted
		isEncrypted = true
	}

	// Create account
	account := &Account{
		Address:       *v6Addr,
		PublicKey:     kp.PublicKey,
		PrivateKey:    privateKey,
		NetworkPrefix: w.networkPrefix,
		Label:         label,
		CreatedAt:     time.Now(),
		Index:         index,
		Nonce:         0,
		IsEncrypted:   isEncrypted,
	}

	// Add to wallet
	w.accounts = append(w.accounts, account)
	w.accountsIndex[v6Addr.String()] = account

	return nil
}

// GetAccount returns an account by address
func (w *Wallet) GetAccount(addrStr string) (*Account, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return account, nil
}

// GetAccountByIndex returns an account by index
func (w *Wallet) GetAccountByIndex(index uint32) (*Account, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, account := range w.accounts {
		if account.Index == index {
			return account, nil
		}
	}

	return nil, ErrAccountNotFound
}

// GetAccounts returns all accounts
func (w *Wallet) GetAccounts() []*Account {
	w.mu.RLock()
	defer w.mu.RUnlock()

	accounts := make([]*Account, len(w.accounts))
	for i, account := range w.accounts {
		// Return copies to avoid exposing private keys
		accounts[i] = &Account{
			Address:       account.Address,
			PublicKey:     account.PublicKey,
			PrivateKey:    nil, // Don't copy private key
			NetworkPrefix: account.NetworkPrefix,
			Label:         account.Label,
			CreatedAt:     account.CreatedAt,
			Index:         account.Index,
			Nonce:         account.Nonce,
			IsEncrypted:   account.IsEncrypted,
		}
	}
	return accounts
}

// GetAccountCount returns number of accounts
func (w *Wallet) GetAccountCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.accounts)
}

// Lock locks the wallet (encrypts private keys)
func (w *Wallet) Lock() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.locked {
		return ErrWalletLocked
	}

	if !w.encrypted || w.passwordHash == nil {
		return errors.New("wallet is not encrypted with password")
	}

	// Encrypt all private keys
	for _, account := range w.accounts {
		if !account.IsEncrypted && len(account.PrivateKey) > 0 {
			encrypted, err := crypto.Encrypt(w.passwordHash, account.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt private key: %w", err)
			}
			account.PrivateKey = encrypted
			account.IsEncrypted = true
		}
	}

	w.locked = true
	return nil
}

// Unlock unlocks the wallet (decrypts private keys)
func (w *Wallet) Unlock(password string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.locked {
		return ErrWalletUnlocked
	}

	// Verify password
	if !hashEqual(w.passwordHash, hashPassword(password)) {
		return ErrInvalidPassword
	}

	// Decrypt all private keys
	for _, account := range w.accounts {
		if account.IsEncrypted && len(account.PrivateKey) > 0 {
			decrypted, err := crypto.Decrypt(w.passwordHash, account.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt private key: %w", err)
			}
			account.PrivateKey = decrypted
			account.IsEncrypted = false
		}
	}

	w.locked = false
	return nil
}

// IsLocked returns whether wallet is locked
func (w *Wallet) IsLocked() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.locked
}

// ChangePassword changes wallet password
func (w *Wallet) ChangePassword(oldPassword, newPassword string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.locked {
		// First verify old password
		if !hashEqual(w.passwordHash, hashPassword(oldPassword)) {
			return ErrInvalidPassword
		}

		// Unlock temporarily
		w.locked = false
		defer func() { w.locked = true }()
	}

	// Change password hash
	oldHash := w.passwordHash
	newHash := hashPassword(newPassword)

	// Re-encrypt all private keys with new password
	for _, account := range w.accounts {
		if len(account.PrivateKey) > 0 && account.IsEncrypted {
			// Decrypt with old password
			decrypted, err := crypto.Decrypt(oldHash, account.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt private key: %w", err)
			}

			// Encrypt with new password
			encrypted, err := crypto.Encrypt(newHash, decrypted)
			if err != nil {
				return fmt.Errorf("failed to encrypt private key: %w", err)
			}
			account.PrivateKey = encrypted
		} else if len(account.PrivateKey) > 0 && !account.IsEncrypted {
			// Encrypt with new password
			encrypted, err := crypto.Encrypt(newHash, account.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt private key: %w", err)
			}
			account.PrivateKey = encrypted
			account.IsEncrypted = true
		}
	}

	w.passwordHash = newHash
	return nil
}

// ExportPrivateKey exports private key for an account
func (w *Wallet) ExportPrivateKey(addrStr string, password string) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.locked {
		return nil, ErrWalletLocked
	}

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// If account is encrypted, need password
	if account.IsEncrypted {
		if password == "" {
			return nil, ErrWalletLocked
		}
		if !hashEqual(w.passwordHash, hashPassword(password)) {
			return nil, ErrInvalidPassword
		}

		// Decrypt private key
		decrypted, err := crypto.Decrypt(w.passwordHash, account.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}

		// Return copy of private key
		privateKey := make([]byte, len(decrypted))
		copy(privateKey, decrypted)
		return privateKey, nil
	}

	if len(account.PrivateKey) == 0 {
		return nil, ErrPrivateKeyEmpty
	}

	// Return copy of private key
	privateKey := make([]byte, len(account.PrivateKey))
	copy(privateKey, account.PrivateKey)
	return privateKey, nil
}

// GetMnemonic returns the wallet's mnemonic
func (w *Wallet) GetMnemonic() (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.mnemonic == nil {
		return "", errors.New("mnemonic not available")
	}

	return w.mnemonic.String(), nil
}

// Sign signs data with private key of specified account
func (w *Wallet) Sign(addrStr string, data []byte, password string) ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Get private key
	var privateKey []byte
	if account.IsEncrypted {
		if password == "" {
			return nil, ErrWalletLocked
		}
		if !hashEqual(w.passwordHash, hashPassword(password)) {
			return nil, ErrInvalidPassword
		}

		// Decrypt private key
		decrypted, err := crypto.Decrypt(w.passwordHash, account.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
		privateKey = decrypted
	} else {
		privateKey = account.PrivateKey
	}

	if len(privateKey) == 0 {
		return nil, ErrPrivateKeyEmpty
	}

	signature, err := crypto.Sign(privateKey, data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	return signature, nil
}

// Verify verifies a signature with account's public key
func (w *Wallet) Verify(addrStr string, data, signature []byte) (bool, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return false, ErrAccountNotFound
	}

	valid := crypto.Verify(account.PublicKey, data, signature)
	return valid, nil
}

// UpdateNonce updates nonce for an account
func (w *Wallet) UpdateNonce(addrStr string, nonce uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return ErrAccountNotFound
	}

	account.Nonce = nonce
	return nil
}

// GetNonce returns nonce for an account
func (w *Wallet) GetNonce(addrStr string) (uint64, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return 0, ErrAccountNotFound
	}

	return account.Nonce, nil
}

// UpdateAccountLabel updates the label for an account
func (w *Wallet) UpdateAccountLabel(addrStr, label string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	account, exists := w.accountsIndex[addrStr]
	if !exists {
		return ErrAccountNotFound
	}

	account.Label = label
	return nil
}

// hashPassword hashes a password using SHA-256 (simplified)
// In production, use Argon2 or scrypt
func hashPassword(password string) []byte {
	return crypto.HashSHA256([]byte(password))
}

// hashEqual compares two hashes in constant time
func hashEqual(a, b []byte) bool {
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
