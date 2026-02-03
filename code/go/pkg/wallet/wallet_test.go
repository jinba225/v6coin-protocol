package wallet

import (
	"net"
	"testing"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

func TestNewWallet(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Check mnemonic
	mnemonic, err := wallet.GetMnemonic()
	if err != nil {
		t.Fatalf("failed to get mnemonic: %v", err)
	}
	if mnemonic == "" {
		t.Error("mnemonic is empty")
	}

	// Check default account was created
	accountCount := wallet.GetAccountCount()
	if accountCount != 1 {
		t.Errorf("expected 1 account, got %d", accountCount)
	}

	// Check account is locked if password provided
	if config.Password != "" {
		if !wallet.IsLocked() {
			t.Error("wallet should be locked")
		}
	}
}

func TestCreateAccount(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Create additional account
	err = wallet.CreateAccount(1, "Account 1", "")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// Check account count
	accountCount := wallet.GetAccountCount()
	if accountCount != 2 {
		t.Errorf("expected 2 accounts, got %d", accountCount)
	}

	// Get account by index
	account, err := wallet.GetAccountByIndex(1)
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}

	if account.Label != "Account 1" {
		t.Errorf("expected label 'Account 1', got '%s'", account.Label)
	}
}

func TestLockUnlock(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		Password:      "test123",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Wallet should be locked initially
	if !wallet.IsLocked() {
		t.Error("wallet should be locked")
	}

	// Unlock wallet
	err = wallet.Unlock("test123")
	if err != nil {
		t.Fatalf("failed to unlock wallet: %v", err)
	}

	if wallet.IsLocked() {
		t.Error("wallet should be unlocked")
	}

	// Lock wallet
	err = wallet.Lock()
	if err != nil {
		t.Fatalf("failed to lock wallet: %v", err)
	}

	if !wallet.IsLocked() {
		t.Error("wallet should be locked")
	}
}

func TestLockUnlockInvalidPassword(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		Password:      "test123",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Try to unlock with wrong password
	err = wallet.Unlock("wrongpassword")
	if err != ErrInvalidPassword {
		t.Errorf("expected ErrInvalidPassword, got %v", err)
	}

	// Wallet should still be locked
	if !wallet.IsLocked() {
		t.Error("wallet should still be locked")
	}
}

func TestGetAccount(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Get default account
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	addrStr := accounts[0].Address.String()

	// Get account by address
	account, err := wallet.GetAccount(addrStr)
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}

	if account.Label != "Default" {
		t.Errorf("expected label 'Default', got '%s'", account.Label)
	}
}

func TestSignVerify(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Get default account
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	addrStr := accounts[0].Address.String()
	data := []byte("test message")

	// Sign data
	signature, err := wallet.Sign(addrStr, data, "")
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	if len(signature) != 64 {
		t.Errorf("expected 64-byte signature, got %d", len(signature))
	}

	// Verify signature
	valid, err := wallet.Verify(addrStr, data, signature)
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	if !valid {
		t.Error("signature should be valid")
	}

	// Verify with wrong data
	valid, _ = wallet.Verify(addrStr, []byte("wrong data"), signature)
	if valid {
		t.Error("signature should be invalid for wrong data")
	}
}

func TestTxBuilder(t *testing.T) {
	walletConfig := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(walletConfig)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	txBuilder := NewTxBuilder(wallet)

	// Get accounts
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	fromAddrStr := accounts[0].Address.String()

	// Get another address (use a placeholder)
	toAddr := net.ParseIP("2001:db8::1:2")
	if toAddr == nil {
		t.Fatal("failed to parse to address")
	}

	txConfig := &TxConfig{
		FromAddress: fromAddrStr,
		ToAddress:   toAddr.String(),
		Amount:      1000000, // 1000 nano-V6
		Fee:         1000,
		Nonce:       0,
		Password:    "",
	}

	// Build online transaction
	tx, err := txBuilder.BuildOnlineTransaction(txConfig)
	if err != nil {
		t.Fatalf("failed to build transaction: %v", err)
	}

	// Verify transaction
	if tx.Type != consensus.TxTypeOnline {
		t.Errorf("expected TxTypeOnline, got %d", tx.Type)
	}

	if tx.Amount != 1000000 {
		t.Errorf("expected amount 1000000, got %d", tx.Amount)
	}

	if len(tx.Signature) != 64 {
		t.Errorf("expected 64-byte signature, got %d", len(tx.Signature))
	}
}

func TestBuildOfflineTransaction(t *testing.T) {
	walletConfig := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(walletConfig)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	txBuilder := NewTxBuilder(wallet)

	// Get accounts
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	fromAddrStr := accounts[0].Address.String()

	// Get another address (use a placeholder)
	toAddr := net.ParseIP("2001:db8::1:2")
	if toAddr == nil {
		t.Fatal("failed to parse to address")
	}

	txConfig := &TxConfig{
		FromAddress: fromAddrStr,
		ToAddress:   toAddr.String(),
		Amount:      1000000,
		Fee:         1000,
		Nonce:       0,
		Password:    "",
	}

	// Build offline transaction
	tx, err := txBuilder.BuildOfflineTransaction(txConfig)
	if err != nil {
		t.Fatalf("failed to build offline transaction: %v", err)
	}

	// Verify transaction
	if tx.Type != consensus.TxTypeOffline {
		t.Errorf("expected TxTypeOffline, got %d", tx.Type)
	}
}

func TestBuildMigrationTransaction(t *testing.T) {
	walletConfig := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(walletConfig)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Create second account for migration
	err = wallet.CreateAccount(1, "Account 1", "")
	if err != nil {
		t.Fatalf("failed to create second account: %v", err)
	}

	txBuilder := NewTxBuilder(wallet)

	// Get accounts
	accounts := wallet.GetAccounts()
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	// 使用 Address.String() 格式，因为 GetAccount 使用这个格式在 accountsIndex 中查找
	fromAddrStr := accounts[0].Address.String()
	toAddrStr := accounts[1].Address.GetFullAddress().String()

	txConfig := &TxConfig{
		FromAddress: fromAddrStr,
		ToAddress:   toAddrStr,
		Amount:      1000000,
		Fee:         1000,
		Nonce:       0,
		Password:    "",
	}

	// Build migration transaction
	tx, err := txBuilder.BuildMigrationTransaction(txConfig)
	if err != nil {
		t.Fatalf("failed to build migration transaction: %v", err)
	}

	// Verify transaction
	if tx.Type != consensus.TxTypeMigration {
		t.Errorf("expected TxTypeMigration, got %d", tx.Type)
	}
}

func TestUpdateNonce(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Get default account
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	addrStr := accounts[0].Address.String()

	// Update nonce
	err = wallet.UpdateNonce(addrStr, 42)
	if err != nil {
		t.Fatalf("failed to update nonce: %v", err)
	}

	// Get nonce
	nonce, err := wallet.GetNonce(addrStr)
	if err != nil {
		t.Fatalf("failed to get nonce: %v", err)
	}

	if nonce != 42 {
		t.Errorf("expected nonce 42, got %d", nonce)
	}
}

func TestUpdateAccountLabel(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Get default account
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	addrStr := accounts[0].Address.String()

	// Update label
	err = wallet.UpdateAccountLabel(addrStr, "New Label")
	if err != nil {
		t.Fatalf("failed to update label: %v", err)
	}

	// Get account and check label
	account, err := wallet.GetAccount(addrStr)
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}

	if account.Label != "New Label" {
		t.Errorf("expected label 'New Label', got '%s'", account.Label)
	}
}

func TestExportPrivateKey(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Get default account
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	addrStr := accounts[0].Address.String()

	// Export private key (Ed25519 format is 64 bytes: seed 32 + publicKey 32)
	privateKey, err := wallet.ExportPrivateKey(addrStr, "")
	if err != nil {
		t.Fatalf("failed to export private key: %v", err)
	}

	if len(privateKey) != 64 {
		t.Errorf("expected 64-byte Ed25519 private key, got %d", len(privateKey))
	}
}

func TestExportPrivateKeyLocked(t *testing.T) {
	config := &WalletConfig{
		DataDir:       "/tmp/v6coin",
		Password:      "test123",
		NetworkPrefix: "2001:db8::",
	}

	wallet, err := NewWallet(config)
	if err != nil {
		t.Fatalf("failed to create wallet: %v", err)
	}

	// Get default account
	accounts := wallet.GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	addrStr := accounts[0].Address.String()

	// Try to export private key without password
	_, err = wallet.ExportPrivateKey(addrStr, "")
	if err != ErrWalletLocked {
		t.Errorf("expected ErrWalletLocked, got %v", err)
	}

	// Export with password
	privateKey, err := wallet.ExportPrivateKey(addrStr, "test123")
	if err != nil {
		t.Fatalf("failed to export private key: %v", err)
	}

	if len(privateKey) != 64 {
		t.Errorf("expected 64-byte Ed25519 private key, got %d", len(privateKey))
	}
}
