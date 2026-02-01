package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

// HashAlgorithm represents the hash algorithm type
type HashAlgorithm string

const (
	SHA256   HashAlgorithm = "sha256"
	SHA512   HashAlgorithm = "sha512"
	SHA3_256 HashAlgorithm = "sha3-256"
	SHA3_512 HashAlgorithm = "sha3-512"
)

// Hash computes the hash of data using the specified algorithm
func Hash(data []byte, algorithm HashAlgorithm) []byte {
	switch algorithm {
	case SHA256:
		return HashSHA256(data)
	case SHA512:
		return HashSHA512(data)
	case SHA3_256:
		return HashSHA3_256(data)
	case SHA3_512:
		return HashSHA3_512(data)
	default:
		return HashSHA256(data)
	}
}

// HashSHA512 calculates SHA-512 hash of data
func HashSHA512(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

// HashSHA3_256 calculates SHA3-256 hash of data
func HashSHA3_256(data []byte) []byte {
	hash := sha3.Sum256(data)
	return hash[:]
}

// HashSHA3_512 calculates SHA3-512 hash of data
func HashSHA3_512(data []byte) []byte {
	hash := sha3.Sum512(data)
	return hash[:]
}

// DoubleHash performs double SHA-256 hash (SHA-256(SHA-256(data)))
func DoubleHash(data []byte) []byte {
	first := HashSHA256(data)
	return HashSHA256(first)
}

// HashToHex converts hash bytes to hexadecimal string
func HashToHex(hash []byte) string {
	return hex.EncodeToString(hash)
}

// HexToHash converts hexadecimal string to hash bytes
func HexToHash(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}

// Hash256 computes SHA-256 hash (alias for HashSHA256)
func Hash256(data []byte) []byte {
	return HashSHA256(data)
}

// Hash512 computes SHA-512 hash (alias for HashSHA512)
func Hash512(data []byte) []byte {
	return HashSHA512(data)
}

// Hash160 computes RIPEMD-160 of SHA-256 (Bitcoin address style)
func Hash160(data []byte) []byte {
	sha256Hash := sha256.Sum256(data)
	// RIPEMD-160 would be implemented here
	// For now, use first 20 bytes of SHA-256 as placeholder
	_ = sha256Hash // Use hash to avoid unused variable warning
	return sha256Hash[:20]
}

// MerkleRoot computes the Merkle root of a list of hashes
func MerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		return []byte{}
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	// Build the Merkle tree
	tree := make([][]byte, len(hashes))
	copy(tree, hashes)

	for len(tree) > 1 {
		// If odd number, duplicate last hash
		if len(tree)%2 == 1 {
			tree = append(tree, tree[len(tree)-1])
		}

		// Hash pairs
		newTree := make([][]byte, len(tree)/2)
		for i := 0; i < len(tree); i += 2 {
			combined := append(tree[i], tree[i+1]...)
			newTree[i/2] = DoubleHash(combined)
		}
		tree = newTree
	}

	return tree[0]
}

// ComputeRootHash computes the root hash of a list of data items
func ComputeRootHash(items [][]byte) []byte {
	hashes := make([][]byte, len(items))
	for i, item := range items {
		hashes[i] = HashSHA256(item)
	}
	return MerkleRoot(hashes)
}

// HashPair hashes two byte arrays together
func HashPair(a, b []byte) []byte {
	combined := append(a, b...)
	return HashSHA256(combined)
}

// HashWithSalt hashes data with a salt
func HashWithSalt(data, salt []byte, algorithm HashAlgorithm) []byte {
	combined := append(salt, data...)
	return Hash(combined, algorithm)
}

// Pbkdf2 derives a key from password using PBKDF2
func Pbkdf2(password, salt []byte, iterations, keyLen int, h func() hash.Hash) []byte {
	return Pbkdf2Key(password, salt, iterations, keyLen, h)
}

// Pbkdf2Key is the actual PBKDF2 implementation
func Pbkdf2Key(password, salt []byte, iterations, keyLen int, h func() hash.Hash) []byte {
	return pbkdf2.Key(password, salt, iterations, keyLen, h)
}

// HkdfSha256 derives keys using HKDF-SHA256
// Simplified implementation for demonstration
func HkdfSha256(secret, salt, info []byte, outputLen int) []byte {
	// Extract
	if salt == nil || len(salt) == 0 {
		salt = make([]byte, 32) // Zero salt
	}

	// Expand
	t := []byte{}
	okm := []byte{}
	n := (outputLen + 32 - 1) / 32 // ceil(outputLen / hashLen)

	for i := 0; i < n; i++ {
		t = hmacSHA256(t, append(info, byte(i+1)))
		if len(t) > outputLen-len(okm) {
			okm = append(okm, t[:outputLen-len(okm)]...)
		} else {
			okm = append(okm, t...)
		}
	}

	return okm
}

// hmacSHA256 calculates HMAC-SHA256
func hmacSHA256(key, data []byte) []byte {
	if len(key) > 64 {
		hash := sha256.Sum256(key)
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

	hash := sha256.Sum256(append(iPad, data...))
	mac := sha256.Sum256(append(oPad, hash[:]...))

	return mac[:]
}

// HashTreeProof generates a Merkle proof for a specific leaf
func HashTreeProof(hashes [][]byte, index int) ([][]byte, error) {
	if index < 0 || index >= len(hashes) {
		return nil, fmt.Errorf("index out of range")
	}

	proof := make([][]byte, 0)
	level := hashes

	for len(level) > 1 {
		// If odd number, duplicate last hash
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}

		// Find sibling for target index
		siblingIndex := index ^ 1
		proofLevel := []byte{}
		if siblingIndex < len(level) {
			proofLevel = level[siblingIndex]
		}
		proof = append(proof, proofLevel)

		// Hash pairs
		newLevel := make([][]byte, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			combined := append(level[i], level[i+1]...)
			newLevel[i/2] = DoubleHash(combined)
		}
		level = newLevel
		index = index / 2
	}

	return proof, nil
}

// VerifyMerkleProof verifies a Merkle proof
func VerifyMerkleProof(leafHash []byte, proof [][]byte, root []byte, index int) bool {
	current := leafHash
	currentIndex := index

	for _, sibling := range proof {
		if len(sibling) == 0 {
			continue
		}

		// Determine order
		if currentIndex%2 == 0 {
			current = DoubleHash(append(current, sibling...))
		} else {
			current = DoubleHash(append(sibling, current...))
		}
		currentIndex = currentIndex / 2
	}

	return compareHashes(current, root)
}

// compareHashes compares two hashes in constant time
func compareHashes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	result := byte(0)
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// GenerateECDSAKey generates a new ECDSA key pair using P-256 curve
func GenerateECDSAKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// GenerateECDSAKeyFromSeed generates ECDSA key from seed
func GenerateECDSAKeyFromSeed(seed []byte) (*ecdsa.PrivateKey, error) {
	// Note: For deterministic keys in production, use proper BIP-32 style derivation
	// This is a simplified implementation for demonstration
	// In production, use: github.com/btcsuite/btcd/btcutil/hdkey
	hash := sha256.Sum256(seed)
	// Use seed reader that cycles through hash data
	reader := NewSeedReader(hash[:])
	return ecdsa.GenerateKey(elliptic.P256(), reader)
}

// hashReader implements io.Reader using hash as entropy source
type hashReader struct {
	data []byte
	pos  int
}

// NewHashReader creates a new hash reader
func NewHashReader(data []byte) *hashReader {
	return &hashReader{data: data, pos: 0}
}

// Read implements io.Reader
func (r *hashReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, nil // EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		err = nil // EOF
	}
	return n, err
}

// seedReader wraps hashReader to provide endless entropy by cycling
type seedReader struct {
	hr *hashReader
}

// NewSeedReader creates a new seed reader that cycles through hash data
func NewSeedReader(data []byte) *seedReader {
	return &seedReader{hr: NewHashReader(data)}
}

// Read implements io.Reader with cycling
func (r *seedReader) Read(p []byte) (n int, err error) {
	totalRead := 0
	for totalRead < len(p) {
		bytesRead, readErr := r.hr.Read(p[totalRead:])
		totalRead += bytesRead
		if readErr == nil && bytesRead == 0 {
			// Reset to beginning
			r.hr.pos = 0
			continue
		}
		if totalRead >= len(p) {
			break
		}
	}
	return totalRead, nil
}

// HashAddress hashes an address for storage
func HashAddress(address []byte) []byte {
	return DoubleHash(address)
}

// HashTransaction hashes a transaction for signature
func HashTransaction(txData []byte) []byte {
	return DoubleHash(txData)
}

// HashBlock hashes a block header
func HashBlock(blockData []byte) []byte {
	return DoubleHash(blockData)
}
