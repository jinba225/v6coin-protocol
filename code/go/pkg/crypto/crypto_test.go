package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, kp.PrivateKey)
	assert.NotNil(t, kp.PublicKey)
	assert.Len(t, kp.PrivateKey, ed25519.PrivateKeySize)
	assert.Len(t, kp.PublicKey, ed25519.PublicKeySize)
}

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	assert.NoError(t, err)

	message := []byte("Hello, V6Coin!")
	signature := kp.Sign(message)

	assert.True(t, kp.Verify(message, signature))
	assert.False(t, kp.Verify([]byte("Different message"), signature))
}

func TestGenerateIID(t *testing.T) {
	kp, err := GenerateKeyPair()
	assert.NoError(t, err)

	iid := GenerateIID(kp.PublicKey)
	assert.NotZero(t, iid)

	// Same public key should generate same IID
	iid2 := GenerateIID(kp.PublicKey)
	assert.Equal(t, iid, iid2)
}

func TestEncryptAndDecrypt(t *testing.T) {
	key := make([]byte, 32) // AES-256 requires 32-byte key
	plaintext := []byte("Secret message")

	ciphertext, err := Encrypt(key, plaintext)
	assert.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := Decrypt(key, ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestHashSHA256(t *testing.T) {
	data := []byte("Test data")
	hash := HashSHA256(data)
	assert.NotNil(t, hash)
	assert.Len(t, hash, 32) // SHA-256 produces 32-byte hash

	// Same data should produce same hash
	hash2 := HashSHA256(data)
	assert.Equal(t, hash, hash2)
}
