package wallet

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/jinba225/v6coin-protocol/pkg/address"
	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

// Signature represents a transaction signature with public key
type Signature struct {
	PublicKey []byte `json:"publicKey"`
	Sign      []byte `json:"sign"`
}

// SignTransaction signs a transaction using the provided private key
func SignTransaction(tx *consensus.Transaction, privateKey []byte) (*Signature, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("invalid private key length: %d (expected 32)", len(privateKey))
	}

	message := buildMessageForSigning(tx)

	signature := ed25519.Sign(privateKey, message)

	return &Signature{
		PublicKey: ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey),
		Sign:      signature,
	}, nil
}

// VerifyTransaction verifies a transaction signature using the public key
func VerifyTransaction(tx *consensus.Transaction, signature *Signature) bool {
	if tx == nil || signature == nil {
		return false
	}
	if len(signature.PublicKey) != 32 {
		return false
	}
	if len(signature.Sign) != 64 {
		return false
	}

	message := buildMessageForSigning(tx)

	return ed25519.Verify(signature.PublicKey, message, signature.Sign)
}

// SignTransactionWithKeyPair signs a transaction using a KeyPair
func SignTransactionWithKeyPair(tx *consensus.Transaction, kp *KeyPair) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	if kp == nil {
		return fmt.Errorf("key pair cannot be nil")
	}

	hash := tx.Hash()
	signature, err := crypto.Sign(kp.PrivateKey, hash)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	tx.Signature = signature
	return nil
}

// VerifyTransactionWithPublicKey verifies a transaction using a public key
func VerifyTransactionWithPublicKey(tx *consensus.Transaction, publicKey []byte) bool {
	if tx == nil || len(publicKey) != 32 {
		return false
	}

	return tx.Verify(publicKey)
}

// buildMessageForSigning builds the message to be signed for a transaction
func buildMessageForSigning(tx *consensus.Transaction) []byte {
	return tx.Hash()
}

// CreateSignedTransaction creates and signs a transaction in one step
func CreateSignedTransaction(txType consensus.TxType, from, to net.IP, amount, fee, nonce uint64, privateKey []byte) (*consensus.Transaction, error) {
	tx := consensus.NewTransaction(txType, from, to, amount, fee, nonce)

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	return tx, nil
}

// CreateSignedTransactionWithKeyPair creates and signs a transaction using a KeyPair
func CreateSignedTransactionWithKeyPair(txType consensus.TxType, from, to net.IP, amount, fee, nonce uint64, kp *KeyPair) (*consensus.Transaction, error) {
	tx := consensus.NewTransaction(txType, from, to, amount, fee, nonce)

	if err := SignTransactionWithKeyPair(tx, kp); err != nil {
		return nil, err
	}

	return tx, nil
}

// CreateMigrationTransaction creates and signs an asset migration transaction
func CreateMigrationTransaction(fromAddr, toAddr *address.V6Address, privateKey []byte) (*consensus.Transaction, error) {
	if fromAddr == nil || toAddr == nil {
		return nil, fmt.Errorf("both from and to addresses must be provided")
	}
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("invalid private key length")
	}

	migration, err := address.CreateMigration(fromAddr, toAddr, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration: %w", err)
	}

	fromIP := fromAddr.GetFullAddress()
	toIP := toAddr.GetFullAddress()

	tx := consensus.NewTransaction(
		consensus.TxTypeMigration,
		fromIP,
		toIP,
		0,
		1000,
		uint64(migration.Timestamp),
	)

	tx.Data = []byte(fmt.Sprintf("%x", migration.Signature))

	if err := tx.Sign(privateKey); err != nil {
		return nil, fmt.Errorf("failed to sign migration transaction: %w", err)
	}

	return tx, nil
}

// SignArbitraryMessage signs an arbitrary message
func SignArbitraryMessage(message []byte, privateKey []byte) ([]byte, error) {
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("invalid private key length")
	}

	hash := crypto.HashSHA256(message)

	signature, err := crypto.Sign(privateKey, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return signature, nil
}

// VerifyArbitraryMessage verifies a signature on an arbitrary message
func VerifyArbitraryMessage(message, signature, publicKey []byte) bool {
	if len(publicKey) != 32 {
		return false
	}

	hash := crypto.HashSHA256(message)

	return crypto.Verify(publicKey, hash, signature)
}

// GetSignatureHex returns signature as hex string
func (sig *Signature) GetSignatureHex() string {
	return hex.EncodeToString(sig.Sign)
}

// GetPublicKeyHex returns public key as hex string
func (sig *Signature) GetPublicKeyHex() string {
	return hex.EncodeToString(sig.PublicKey)
}

// SignBytes signs arbitrary bytes using Ed25519
func (kp *KeyPair) SignBytes(data []byte) ([]byte, error) {
	return SignArbitraryMessage(data, kp.PrivateKey)
}

// VerifyBytes verifies a signature on arbitrary bytes
func (kp *KeyPair) VerifyBytes(data, signature []byte) bool {
	return VerifyArbitraryMessage(data, signature, kp.PublicKey)
}
