package wallet

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/address"
	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidFee          = errors.New("invalid fee")
)

// TxBuilder builds transactions
type TxBuilder struct {
	wallet      *Wallet
	chainHeight uint64
	gasPrice    uint64
	nonceCache  map[string]uint64 // address -> nonce
}

// TxConfig is configuration for transaction building
type TxConfig struct {
	FromAddress string
	ToAddress   string
	Amount      uint64 // in nano-V6
	Fee         uint64 // in nano-V6
	Data        []byte
	Type        consensus.TxType
	Nonce       uint64 // 0 to use wallet nonce
	Password    string
}

// NewTxBuilder creates a new transaction builder
func NewTxBuilder(wallet *Wallet) *TxBuilder {
	return &TxBuilder{
		wallet:     wallet,
		gasPrice:   1, // Default: 1 nano-V6 per gas unit
		nonceCache: make(map[string]uint64),
	}
}

// SetChainHeight sets current chain height for calculating timestamps
func (tb *TxBuilder) SetChainHeight(height uint64) {
	tb.chainHeight = height
}

// SetGasPrice sets gas price for transactions
func (tb *TxBuilder) SetGasPrice(price uint64) {
	tb.gasPrice = price
}

// BuildOnlineTransaction builds an online transaction
func (tb *TxBuilder) BuildOnlineTransaction(config *TxConfig) (*consensus.Transaction, error) {
	config.Type = consensus.TxTypeOnline
	return tb.buildTransaction(config)
}

// BuildOfflineTransaction builds an offline transaction
func (tb *TxBuilder) BuildOfflineTransaction(config *TxConfig) (*consensus.Transaction, error) {
	config.Type = consensus.TxTypeOffline
	return tb.buildTransaction(config)
}

// BuildStakeTransaction builds a staking transaction
func (tb *TxBuilder) BuildStakeTransaction(config *TxConfig) (*consensus.Transaction, error) {
	config.Type = consensus.TxTypeStake
	return tb.buildTransaction(config)
}

// BuildUnstakeTransaction builds an unstaking transaction
func (tb *TxBuilder) BuildUnstakeTransaction(config *TxConfig) (*consensus.Transaction, error) {
	config.Type = consensus.TxTypeUnstake
	return tb.buildTransaction(config)
}

// BuildMigrationTransaction builds an asset migration transaction
func (tb *TxBuilder) BuildMigrationTransaction(config *TxConfig) (*consensus.Transaction, error) {
	config.Type = consensus.TxTypeMigration
	return tb.buildTransaction(config)
}

// buildTransaction builds a transaction from config
func (tb *TxBuilder) buildTransaction(config *TxConfig) (*consensus.Transaction, error) {
	// Validate config
	if err := tb.validateConfig(config); err != nil {
		return nil, err
	}

	// Get account
	account, err := tb.wallet.GetAccount(config.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Parse addresses
	fromAddr := account.Address.GetFullAddress()
	toAddr := net.ParseIP(config.ToAddress)
	if toAddr == nil || len(toAddr) != 16 {
		return nil, ErrInvalidAddress
	}

	// Determine nonce
	nonce := config.Nonce
	if nonce == 0 {
		// Use wallet nonce
		nonce, _ = tb.wallet.GetNonce(config.FromAddress)
	}

	// Calculate gas
	gasLimit := tb.calculateGasLimit(config.Type, len(config.Data))
	gasFee := gasLimit * tb.gasPrice

	// Calculate total fee
	totalFee := gasFee
	if config.Fee > 0 {
		totalFee = config.Fee
	}

	// Create transaction
	tx := &consensus.Transaction{
		Version:   0x0100,
		Type:      config.Type,
		From:      fromAddr,
		To:        toAddr,
		Amount:    config.Amount,
		Fee:       totalFee,
		Nonce:     nonce,
		Timestamp: uint64(time.Now().Unix()),
		Data:      config.Data,
	}

	// Sign transaction
	signature, err := tb.wallet.Sign(config.FromAddress, tx.Hash(), config.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	tx.Signature = signature

	// Update nonce in wallet (for next transaction)
	tb.wallet.UpdateNonce(config.FromAddress, nonce+1)

	return tx, nil
}

// BuildSignedOfflineTx builds an offline transaction with pre-signed signature
// This is useful for devices that cannot sign transactions themselves
func (tb *TxBuilder) BuildSignedOfflineTx(config *TxConfig, signature []byte) (*consensus.Transaction, error) {
	config.Type = consensus.TxTypeOffline

	// Validate config
	if err := tb.validateConfig(config); err != nil {
		return nil, err
	}

	// Get account
	account, err := tb.wallet.GetAccount(config.FromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Parse addresses
	fromAddr := account.Address.GetFullAddress()
	toAddr := net.ParseIP(config.ToAddress)
	if toAddr == nil || len(toAddr) != 16 {
		return nil, ErrInvalidAddress
	}

	// Determine nonce
	nonce := config.Nonce
	if nonce == 0 {
		nonce, _ = tb.wallet.GetNonce(config.FromAddress)
	}

	// Calculate gas
	gasLimit := tb.calculateGasLimit(config.Type, len(config.Data))
	gasFee := gasLimit * tb.gasPrice

	// Calculate total fee
	totalFee := gasFee
	if config.Fee > 0 {
		totalFee = config.Fee
	}

	// Create transaction
	tx := &consensus.Transaction{
		Version:   0x0100,
		Type:      config.Type,
		From:      fromAddr,
		To:        toAddr,
		Amount:    config.Amount,
		Fee:       totalFee,
		Nonce:     nonce,
		Timestamp: uint64(time.Now().Unix()),
		Data:      config.Data,
		Signature: signature,
	}

	// Verify signature
	valid, err := tb.wallet.Verify(config.FromAddress, tx.Hash(), signature)
	if err != nil {
		return nil, fmt.Errorf("failed to verify signature: %w", err)
	}
	if !valid {
		return nil, errors.New("invalid signature")
	}

	// Update nonce in wallet (for next transaction)
	tb.wallet.UpdateNonce(config.FromAddress, nonce+1)

	return tx, nil
}

// CacheOfflineSignature caches an offline transaction signature
func (tb *TxBuilder) CacheOfflineSignature(fromAddr string, data []byte, signature []byte) error {
	// TODO: Implement offline signature caching
	return nil
}

// GetCachedOfflineSignature retrieves a cached offline signature
func (tb *TxBuilder) GetCachedOfflineSignature(fromAddr string, data []byte) ([]byte, error) {
	// TODO: Implement offline signature caching
	return nil, errors.New("not implemented")
}

// validateConfig validates transaction configuration
func (tb *TxBuilder) validateConfig(config *TxConfig) error {
	// Check from address
	if config.FromAddress == "" {
		return errors.New("from address required")
	}

	// Check to address
	if config.ToAddress == "" {
		return errors.New("to address required")
	}

	// Check amount
	if config.Amount == 0 {
		return ErrInvalidAmount
	}

	return nil
}

// calculateGasLimit calculates gas limit for a transaction
func (tb *TxBuilder) calculateGasLimit(txType consensus.TxType, dataSize int) uint64 {
	baseGas := uint64(21000)
	dataGas := uint64(dataSize) * 2

	switch txType {
	case consensus.TxTypeOnline, consensus.TxTypeOffline:
		return baseGas + uint64(dataGas)
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

// EstimateFee estimates fee for a transaction
func (tb *TxBuilder) EstimateFee(config *TxConfig) (uint64, error) {
	gasLimit := tb.calculateGasLimit(config.Type, len(config.Data))
	return gasLimit * tb.gasPrice, nil
}

// EstimateTransactionResult estimates the result of a transaction
func (tb *TxBuilder) EstimateTransactionResult(config *TxConfig, balance uint64) (bool, uint64, error) {
	// Calculate fee
	gasLimit := tb.calculateGasLimit(config.Type, len(config.Data))
	fee := gasLimit * tb.gasPrice

	// Check if sufficient balance
	totalCost := config.Amount + fee
	if balance < totalCost {
		return false, fee, ErrInsufficientBalance
	}

	return true, fee, nil
}

// CreateTemporaryAddress creates a temporary address from a core address
func (tb *TxBuilder) CreateTemporaryAddress(coreAddrStr string, index int, expiry time.Duration) (*address.V6Address, error) {
	// Get core account
	account, err := tb.wallet.GetAccount(coreAddrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Get private key
	var privateKey []byte
	if account.IsEncrypted {
		if len(account.PrivateKey) > 0 {
			// Decrypt (simplified - would need password in real implementation)
			privateKey, err = crypto.Decrypt(tb.wallet.passwordHash, account.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt private key: %w", err)
			}
		}
	} else {
		privateKey = account.PrivateKey
	}

	// Generate temporary address
	tempAddr, err := address.GenerateTemporaryAddress(privateKey, index)
	if err != nil {
		return nil, fmt.Errorf("failed to generate temporary address: %w", err)
	}

	// Set expiry
	tempAddr.ExpiresAt = time.Now().Add(expiry)

	return tempAddr, nil
}

// CreateLogicalSubnet creates a logical subnet for a core address
func (tb *TxBuilder) CreateLogicalSubnet(coreAddrStr string, prefixes []string) (*address.LogicalSubnet, error) {
	// Get core account
	account, err := tb.wallet.GetAccount(coreAddrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Parse prefixes
	prefixIPs := make([]net.IP, len(prefixes))
	for i, prefix := range prefixes {
		prefixIPs[i] = net.ParseIP(prefix)
		if prefixIPs[i] == nil || len(prefixIPs[i]) != 8 {
			return nil, fmt.Errorf("invalid prefix at index %d", i)
		}
	}

	// Create logical subnet
	subnet, err := address.CreateLogicalSubnet(&account.Address, prefixIPs)
	if err != nil {
		return nil, fmt.Errorf("failed to create logical subnet: %w", err)
	}

	return subnet, nil
}

// CreateMigration creates an address migration
func (tb *TxBuilder) CreateMigration(fromAddrStr, toAddrStr string) (*address.AddressMigration, error) {
	// Get from account
	fromAccount, err := tb.wallet.GetAccount(fromAddrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get from account: %w", err)
	}

	// Get to account
	toAccount, err := tb.wallet.GetAccount(toAddrStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get to account: %w", err)
	}

	// Get private key
	var privateKey []byte
	if fromAccount.IsEncrypted {
		privateKey, err = crypto.Decrypt(tb.wallet.passwordHash, fromAccount.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
	} else {
		privateKey = fromAccount.PrivateKey
	}

	// Create migration
	migration, err := address.CreateMigration(&fromAccount.Address, &toAccount.Address, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration: %w", err)
	}

	return migration, nil
}

// ValidateMigration validates an address migration
func (tb *TxBuilder) ValidateMigration(migration *address.AddressMigration) error {
	// Get account
	account, err := tb.wallet.GetAccount(migration.FromAddress.String())
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Validate migration
	return address.ValidateMigration(migration, account.PublicKey)
}
