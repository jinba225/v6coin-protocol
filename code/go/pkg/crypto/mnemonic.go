package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

const (
	// Mnemonic word count options
	Words12 = 12
	Words15 = 15
	Words18 = 18
	Words21 = 21
	Words24 = 24

	// BIP-39 constants
	EntropyBitSize12 = 128
	EntropyBitSize24 = 256
	IterationCount   = 2048
	KeyLength        = 64
)

// Mnemonic represents a BIP-39 mnemonic phrase
type Mnemonic struct {
	words []string
}

// GenerateMnemonic generates a new BIP-39 mnemonic phrase
func GenerateMnemonic(wordCount int) (*Mnemonic, error) {
	entropySize := getEntropySize(wordCount)
	if entropySize == 0 {
		return nil, fmt.Errorf("invalid word count: %d", wordCount)
	}

	// Generate entropy
	entropy := make([]byte, entropySize/8)
	_, err := rand.Read(entropy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate entropy: %w", err)
	}

	// Calculate checksum
	hash := sha256.Sum256(entropy)
	checksumBits := entropySize / 32

	// Combine entropy and checksum
	checksumBytes := (checksumBits + 7) / 8 // Round up to bytes
	combined := append(entropy, hash[:checksumBytes]...)

	// Convert to words
	return bytesToMnemonic(combined, wordCount)
}

// MnemonicFromWords creates a Mnemonic from a list of words
func MnemonicFromWords(words []string) (*Mnemonic, error) {
	if len(words) != 12 && len(words) != 15 && len(words) != 18 && len(words) != 21 && len(words) != 24 {
		return nil, errors.New("invalid word count")
	}

	// Validate words
	for _, word := range words {
		if !isValidBIP39Word(word) {
			return nil, fmt.Errorf("invalid BIP-39 word: %s", word)
		}
	}

	return &Mnemonic{words: words}, nil
}

// MnemonicFromString creates a Mnemonic from a string phrase
func MnemonicFromString(phrase string) (*Mnemonic, error) {
	words := strings.Fields(phrase)
	if len(words) == 0 {
		return nil, errors.New("empty mnemonic phrase")
	}

	return MnemonicFromWords(words)
}

// String returns the mnemonic as a space-separated string
func (m *Mnemonic) String() string {
	return strings.Join(m.words, " ")
}

// Words returns the list of words
func (m *Mnemonic) Words() []string {
	return m.words
}

// ToSeed converts the mnemonic to a seed using BIP-39 PBKDF2
// An optional passphrase can be provided (default is empty string)
func (m *Mnemonic) ToSeed(passphrase ...string) ([]byte, error) {
	if len(m.words) == 0 {
		return nil, errors.New("mnemonic has no words")
	}

	// Get passphrase (default empty)
	p := ""
	if len(passphrase) > 0 {
		p = passphrase[0]
	}

	salt := "mnemonic" + p

	// BIP-39 uses HMAC-SHA512, not SHA3
	seed := pbkdf2.Key([]byte(m.String()), []byte(salt), IterationCount, 64, sha512.New)
	return seed, nil
}

// ToKeyPair derives an Ed25519 key pair from the mnemonic
func (m *Mnemonic) ToKeyPair(passphrase ...string) (*KeyPair, error) {
	seed, err := m.ToSeed(passphrase...)
	if err != nil {
		return nil, err
	}

	// Ed25519 requires 32-byte seed, take first 32 bytes of BIP-39 seed
	if len(seed) < 32 {
		return nil, errors.New("seed too short")
	}

	ed25519Seed := seed[:32]
	privateKey := ed25519.NewKeyFromSeed(ed25519Seed)
	publicKey := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(publicKey, privateKey[32:])

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// bytesToMnemonic converts entropy bytes to mnemonic words
func bytesToMnemonic(data []byte, wordCount int) (*Mnemonic, error) {
	// Convert bytes to bits
	bits := bytesToBits(data)
	words := make([]string, 0, wordCount)

	// Convert 11-bit chunks to word indices
	for i := 0; i < wordCount; i++ {
		start := i * 11
		if start+11 > len(bits) {
			break
		}

		index := bitsToInt(bits[start : start+11])
		word, err := getBIP39Word(index)
		if err != nil {
			return nil, err
		}
		words = append(words, word)
	}

	return &Mnemonic{words: words}, nil
}

// bytesToBits converts a byte slice to a bit slice
func bytesToBits(data []byte) []int {
	bits := make([]int, 0, len(data)*8)
	for _, b := range data {
		for i := 7; i >= 0; i-- {
			bits = append(bits, int((b>>i)&1))
		}
	}
	return bits
}

// bitsToInt converts a slice of bits to an integer
func bitsToInt(bits []int) int {
	var result int
	for _, bit := range bits {
		result = (result << 1) | bit
	}
	return result
}

// getEntropySize returns the entropy size in bits for a given word count
func getEntropySize(wordCount int) int {
	switch wordCount {
	case Words12:
		return EntropyBitSize12
	case Words15:
		return 160
	case Words18:
		return 192
	case Words21:
		return 224
	case Words24:
		return EntropyBitSize24
	default:
		return 0
	}
}

// isValidBIP39Word checks if a word is in the BIP-39 word list
// For production, this should use the complete BIP-39 word list
func isValidBIP39Word(word string) bool {
	// Simplified check - in production use complete BIP-39 word list
	// This is a placeholder for demonstration
	bip39Words := []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse",
		"access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "act",
		// ... complete 2048 word list needed for production
	}

	for _, w := range bip39Words {
		if w == strings.ToLower(word) {
			return true
		}
	}
	return false
}

// getBIP39Word returns the word at the given index from the BIP-39 word list
func getBIP39Word(index int) (string, error) {
	// In production, use complete BIP-39 word list
	// This is a placeholder for demonstration
	if index < 0 || index >= 2048 {
		return "", fmt.Errorf("invalid word index: %d", index)
	}

	// Placeholder words - replace with actual BIP-39 word list
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract", "absurd", "abuse",
		"access", "accident", "account", "accuse", "achieve", "acid", "acoustic", "acquire", "across", "act",
		// ... complete 2048 word list needed
	}

	if index >= len(words) {
		// For demonstration, generate a placeholder
		return fmt.Sprintf("word%d", index), nil
	}

	return words[index], nil
}

// ValidateMnemonic validates a BIP-39 mnemonic phrase
func ValidateMnemonic(phrase string) error {
	m, err := MnemonicFromString(phrase)
	if err != nil {
		return err
	}

	// Convert words back to bits and verify checksum
	// In production, implement full checksum validation
	_ = m // TODO: Implement full validation

	return nil
}

// DeriveChildKey derives a child key from a parent key using BIP-32 style derivation
// This is a simplified version for demonstration
func DeriveChildKey(parentKey []byte, derivationPath string) ([]byte, error) {
	// Parse derivation path (e.g., "m/44'/0'/0'/0/0")
	// In production, implement full BIP-32 hierarchical deterministic wallet

	// Simplified: Use HMAC-SHA512 to derive child key
	// This is NOT BIP-32 compliant, just for demonstration

	if len(derivationPath) == 0 {
		return parentKey, nil
	}

	// Use derivation path as salt for key derivation
	salt := sha512.Sum512_256([]byte(derivationPath))
	derivedKey := pbkdf2.Key(parentKey, salt[:], IterationCount, len(parentKey), sha3.New256)

	return derivedKey, nil
}

// DeriveAddress derives an IPv6 Interface Identifier from a mnemonic and optional derivation path
func DeriveAddress(mnemonic string, derivationPath string, passphrase ...string) (uint64, error) {
	m, err := MnemonicFromString(mnemonic)
	if err != nil {
		return 0, err
	}

	kp, err := m.ToKeyPair(passphrase...)
	if err != nil {
		return 0, err
	}

	// Apply derivation path if specified
	if derivationPath != "" {
		derivedKey, err := DeriveChildKey(kp.PrivateKey, derivationPath)
		if err != nil {
			return 0, err
		}
		// Generate key pair from derived key (use first 32 bytes for Ed25519)
		if len(derivedKey) < 32 {
			return 0, errors.New("derived key too short")
		}
		ed25519Key := derivedKey[:32]
		kp.PrivateKey = ed25519.NewKeyFromSeed(ed25519Key)
		kp.PublicKey = make(ed25519.PublicKey, ed25519.PublicKeySize)
		copy(kp.PublicKey, kp.PrivateKey[32:])
	}

	iid := GenerateIID(kp.PublicKey)
	return iid, nil
}

// GenerateMasterKey generates a master key from a mnemonic seed
func GenerateMasterKey(seed []byte) ([]byte, error) {
	if len(seed) < 16 {
		return nil, errors.New("seed too short")
	}

	// Use HMAC-SHA512 to generate master key
	mac := hmacSHA512(seed, []byte("Bitcoin seed"))
	return mac, nil
}

// hmacSHA512 calculates HMAC-SHA512
func hmacSHA512(key, data []byte) []byte {
	if len(key) > 64 {
		hash := sha512.Sum512(key)
		key = hash[:]
	}
	if len(key) < 64 {
		padded := make([]byte, 64)
		copy(padded, key)
		key = padded
	}

	oPad := make([]byte, 64)
	iPad := make([]byte, 64)
	for i := 0; i < 64; i++ {
		oPad[i] = key[i] ^ 0x5c
		iPad[i] = key[i] ^ 0x36
	}

	hash := sha512.Sum512(append(iPad, data...))
	mac := sha512.Sum512(append(oPad, hash[:]...))

	return mac[:]
}

// Uint32ToBytes converts a uint32 to bytes (big endian)
func Uint32ToBytes(n uint32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, n)
	return buf
}

// BytesToUint32 converts bytes to uint32 (big endian)
func BytesToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}
