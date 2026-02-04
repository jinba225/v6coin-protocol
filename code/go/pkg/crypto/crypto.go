package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/ed25519"
)

// KeyPair represents a public/private key pair
type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateKeyPair generates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}, nil
}

// Sign signs a message with the private key
func (kp *KeyPair) Sign(message []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, message)
}

// Verify verifies a signature with the public key
func (kp *KeyPair) Verify(message, signature []byte) bool {
	return ed25519.Verify(kp.PublicKey, message, signature)
}

// GenerateIID generates a 64-bit Interface Identifier from public key
func GenerateIID(publicKey []byte) uint64 {
	hash := sha256.Sum256(publicKey)
	// Take first 8 bytes and convert to uint64
	return uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 |
		uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])
}

// Encrypt encrypts data using AES-256-GCM
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// HashSHA256 calculates SHA-256 hash of data
func HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// KeyPairFromPrivateKey creates a key pair from private key seed
func KeyPairFromPrivateKey(privateKey []byte) (*KeyPair, error) {
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("invalid private key length: %d (expected 32)", len(privateKey))
	}

	// 从 32 字节种子生成 Ed25519 密钥对
	// Ed25519 的私钥格式：种子(32字节) + 公钥(32字节) = 64字节
	seed := make([]byte, 32)
	copy(seed, privateKey)

	// 使用标准库方法从种子生成密钥对
	privateKeyFull := ed25519.NewKeyFromSeed(seed)
	publicKey := make(ed25519.PublicKey, 32)
	copy(publicKey, privateKeyFull[32:])

	return &KeyPair{
		PrivateKey: privateKeyFull,
		PublicKey:  publicKey,
	}, nil
}

// Sign signs a message with private key (alias for KeyPair.Sign)
func Sign(privateKey []byte, message []byte) ([]byte, error) {
	if len(privateKey) != 32 {
		return nil, fmt.Errorf("invalid private key length: %d (expected 32)", len(privateKey))
	}

	kp, err := KeyPairFromPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return kp.Sign(message), nil
}

// Verify verifies a signature with public key
func Verify(publicKey []byte, message, signature []byte) bool {
	if len(publicKey) != 32 {
		return false
	}
	if len(signature) != 64 {
		return false
	}

	return ed25519.Verify(publicKey, message, signature)
}

// IsValidSignatureFormat checks if a signature has valid Ed25519 format
func IsValidSignatureFormat(signature []byte) bool {
	// Ed25519 签名固定 64 字节
	if len(signature) != 64 {
		return false
	}

	// 基本格式检查（非空）
	for i := 0; i < 64; i++ {
		if signature[i] != 0 {
			return true
		}
	}

	// 全零签名无效
	return false
}
