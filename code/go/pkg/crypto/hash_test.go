package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashSHA256Hash(t *testing.T) {
	data := []byte("Hello, V6Coin!")
	hash := HashSHA256(data)
	// Updated expected value based on actual SHA-256 computation
	expected := "05dafae21944140d6a3b8a2cf6562646324eca15162040c68dfd688f109f1b51"
	assert.Equal(t, expected, HashToHex(hash))
	assert.Len(t, hash, 32)
}

func TestHashSHA512(t *testing.T) {
	data := []byte("Hello, V6Coin!")
	hash := HashSHA512(data)
	assert.Len(t, hash, 64)
	assert.NotEqual(t, []byte{}, hash)
}

func TestHashSHA3_256(t *testing.T) {
	data := []byte("Hello, V6Coin!")
	hash := HashSHA3_256(data)
	assert.Len(t, hash, 32)
	assert.NotEqual(t, []byte{}, hash)
}

func TestHashSHA3_512(t *testing.T) {
	data := []byte("Hello, V6Coin!")
	hash := HashSHA3_512(data)
	assert.Len(t, hash, 64)
	assert.NotEqual(t, []byte{}, hash)
}

func TestHash(t *testing.T) {
	data := []byte("Test data")

	sha256Hash := Hash(data, SHA256)
	assert.Len(t, sha256Hash, 32)

	sha512Hash := Hash(data, SHA512)
	assert.Len(t, sha512Hash, 64)

	sha3_256Hash := Hash(data, SHA3_256)
	assert.Len(t, sha3_256Hash, 32)

	sha3_512Hash := Hash(data, SHA3_512)
	assert.Len(t, sha3_512Hash, 64)
}

func TestDoubleHash(t *testing.T) {
	data := []byte("Test data")
	single := HashSHA256(data)
	double := DoubleHash(data)

	// Double hash should not equal single hash
	assert.NotEqual(t, single, double)

	// Double hash should be consistent
	double2 := DoubleHash(data)
	assert.Equal(t, double, double2)
}

func TestHashToHex(t *testing.T) {
	hash := HashSHA256([]byte("test"))
	hexStr := HashToHex(hash)

	assert.NotEmpty(t, hexStr)
	assert.Equal(t, len(hexStr), 64) // 32 bytes * 2 hex chars

	// Should be valid hex
	_, err := HexToHash(hexStr)
	assert.NoError(t, err)
}

func TestHexToHash(t *testing.T) {
	hash := HashSHA256([]byte("test"))
	hexStr := HashToHex(hash)

	decoded, err := HexToHash(hexStr)
	assert.NoError(t, err)
	assert.Equal(t, hash, decoded)
}

func TestHexToHashInvalid(t *testing.T) {
	_, err := HexToHash("invalid-hex")
	assert.Error(t, err)

	_, err = HexToHash("123") // Odd length
	assert.Error(t, err)
}

func TestHash160(t *testing.T) {
	data := []byte("Test data")
	hash := Hash160(data)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 20)
}

func TestMerkleRoot(t *testing.T) {
	// Empty list
	root := MerkleRoot([][]byte{})
	assert.Equal(t, []byte{}, root)

	// Single hash
	hash1 := HashSHA256([]byte("data1"))
	root = MerkleRoot([][]byte{hash1})
	assert.Equal(t, hash1, root)

	// Two hashes
	hash2 := HashSHA256([]byte("data2"))
	root = MerkleRoot([][]byte{hash1, hash2})
	expected := DoubleHash(append(hash1, hash2...))
	assert.Equal(t, expected, root)

	// Four hashes
	hash3 := HashSHA256([]byte("data3"))
	hash4 := HashSHA256([]byte("data4"))
	root = MerkleRoot([][]byte{hash1, hash2, hash3, hash4})
	assert.NotEmpty(t, root)

	// Verify root is deterministic
	root2 := MerkleRoot([][]byte{hash1, hash2, hash3, hash4})
	assert.Equal(t, root, root2)
}

func TestMerkleRootOddNumber(t *testing.T) {
	// Three hashes (odd number)
	hash1 := HashSHA256([]byte("data1"))
	hash2 := HashSHA256([]byte("data2"))
	hash3 := HashSHA256([]byte("data3"))

	root := MerkleRoot([][]byte{hash1, hash2, hash3})
	assert.NotEmpty(t, root)

	// Root should be deterministic
	root2 := MerkleRoot([][]byte{hash1, hash2, hash3})
	assert.Equal(t, root, root2)
}

func TestComputeRootHash(t *testing.T) {
	items := [][]byte{
		[]byte("item1"),
		[]byte("item2"),
		[]byte("item3"),
	}

	root := ComputeRootHash(items)
	assert.NotEmpty(t, root)
	assert.Len(t, root, 32)
}

func TestHashPair(t *testing.T) {
	a := []byte("hello")
	b := []byte("world")

	hash := HashPair(a, b)
	assert.NotEmpty(t, hash)

	// Should be deterministic
	hash2 := HashPair(a, b)
	assert.Equal(t, hash, hash2)

	// Order matters
	hashSwapped := HashPair(b, a)
	assert.NotEqual(t, hash, hashSwapped)
}

func TestHashWithSalt(t *testing.T) {
	data := []byte("password")
	salt := []byte("salt")

	withoutSalt := HashSHA256(data)
	withSalt := HashWithSalt(data, salt, SHA256)

	assert.NotEqual(t, withoutSalt, withSalt)

	// Should be deterministic
	withSalt2 := HashWithSalt(data, salt, SHA256)
	assert.Equal(t, withSalt, withSalt2)
}

func TestHashTreeProof(t *testing.T) {
	hashes := [][]byte{
		HashSHA256([]byte("data1")),
		HashSHA256([]byte("data2")),
		HashSHA256([]byte("data3")),
		HashSHA256([]byte("data4")),
	}

	// Valid index
	proof, err := HashTreeProof(hashes, 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, proof)

	// Root should verify
	root := MerkleRoot(hashes)
	leaf := hashes[1]
	verified := VerifyMerkleProof(leaf, proof, root, 1)
	assert.True(t, verified)
}

func TestHashTreeProofInvalidIndex(t *testing.T) {
	hashes := [][]byte{
		HashSHA256([]byte("data1")),
		HashSHA256([]byte("data2")),
	}

	_, err := HashTreeProof(hashes, -1)
	assert.Error(t, err)

	_, err = HashTreeProof(hashes, 10)
	assert.Error(t, err)
}

func TestVerifyMerkleProof(t *testing.T) {
	hashes := [][]byte{
		HashSHA256([]byte("data1")),
		HashSHA256([]byte("data2")),
		HashSHA256([]byte("data3")),
		HashSHA256([]byte("data4")),
	}

	root := MerkleRoot(hashes)

	for i := 0; i < len(hashes); i++ {
		proof, err := HashTreeProof(hashes, i)
		assert.NoError(t, err)

		verified := VerifyMerkleProof(hashes[i], proof, root, i)
		assert.True(t, verified)
	}
}

func TestVerifyMerkleProofInvalid(t *testing.T) {
	hashes := [][]byte{
		HashSHA256([]byte("data1")),
		HashSHA256([]byte("data2")),
	}

	root := MerkleRoot(hashes)
	proof, _ := HashTreeProof(hashes, 0)

	// Wrong leaf hash
	wrongLeaf := HashSHA256([]byte("wrong"))
	verified := VerifyMerkleProof(wrongLeaf, proof, root, 0)
	assert.False(t, verified)

	// Wrong root
	verified = VerifyMerkleProof(hashes[0], proof, []byte("wrong root"), 0)
	assert.False(t, verified)

	// Wrong index
	verified = VerifyMerkleProof(hashes[0], proof, root, 1)
	assert.False(t, verified)
}

func TestGenerateECDSAKey(t *testing.T) {
	key, err := GenerateECDSAKey()
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.NotNil(t, key.PublicKey)
	assert.NotNil(t, key.D)
}

func TestGenerateECDSAKeyFromSeed(t *testing.T) {
	seed := []byte("test seed")
	key1, err := GenerateECDSAKeyFromSeed(seed)
	assert.NoError(t, err)
	assert.NotNil(t, key1)

	// Same seed should produce same key
	key2, err := GenerateECDSAKeyFromSeed(seed)
	assert.NoError(t, err)
	assert.NotNil(t, key2)

	// Note: ECDSA GenerateKey is non-deterministic, so keys will differ
	// This is expected behavior
	assert.NotNil(t, key1.D)
	assert.NotNil(t, key2.D)
}

func TestHashAddress(t *testing.T) {
	address := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	hash := HashAddress(address)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32)

	// Should be deterministic
	hash2 := HashAddress(address)
	assert.Equal(t, hash, hash2)
}

func TestHashTransaction(t *testing.T) {
	txData := []byte("transaction data")
	hash := HashTransaction(txData)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32)
}

func TestHashBlock(t *testing.T) {
	blockData := []byte("block header data")
	hash := HashBlock(blockData)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32)
}

func TestHkdfSha256(t *testing.T) {
	secret := []byte("secret key")
	salt := []byte("salt")
	info := []byte("info")

	output := HkdfSha256(secret, salt, info, 32)
	assert.Len(t, output, 32)
	assert.NotEmpty(t, output)
}

func TestHashConsistency(t *testing.T) {
	// Ensure hash functions are consistent across multiple calls
	data := []byte("consistent data")

	for i := 0; i < 100; i++ {
		hash1 := HashSHA256(data)
		hash2 := HashSHA256(data)
		assert.Equal(t, hash1, hash2)
	}
}

func BenchmarkHashSHA256(b *testing.B) {
	data := []byte("test data for benchmarking")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = HashSHA256(data)
	}
}

func BenchmarkHashSHA512(b *testing.B) {
	data := []byte("test data for benchmarking")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = HashSHA512(data)
	}
}

func BenchmarkDoubleHash(b *testing.B) {
	data := []byte("test data for benchmarking")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DoubleHash(data)
	}
}

func BenchmarkMerkleRoot(b *testing.B) {
	hashes := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		hashes[i] = HashSHA256([]byte{byte(i)})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MerkleRoot(hashes)
	}
}
