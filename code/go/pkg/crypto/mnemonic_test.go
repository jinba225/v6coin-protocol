package crypto

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ed25519"
)

func TestGenerateMnemonic(t *testing.T) {
	for _, wordCount := range []int{Words12, Words15, Words18, Words21, Words24} {
		m, err := GenerateMnemonic(wordCount)
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, wordCount, len(m.words))
		assert.NotEmpty(t, m.String())
	}
}

func TestMnemonicFromWords(t *testing.T) {
	words := []string{"abandon", "ability", "able", "about", "above", "absent",
		"absorb", "abstract", "absurd", "abuse", "access", "accident"}

	m, err := MnemonicFromWords(words)
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, words, m.words)
	assert.Equal(t, "abandon ability able about above absent absorb abstract absurd abuse access accident", m.String())
}

func TestMnemonicFromWordsInvalid(t *testing.T) {
	// Invalid word count
	_, err := MnemonicFromWords([]string{"word1", "word2"})
	assert.Error(t, err)

	// Invalid word (not in BIP-39 word list)
	words := make([]string, 12)
	for i := range words {
		words[i] = "invalidword"
	}
	_, err = MnemonicFromWords(words)
	assert.Error(t, err)
}

func TestMnemonicFromString(t *testing.T) {
	phrase := "abandon ability able about above absent absorb abstract absurd abuse access accident"
	m, err := MnemonicFromString(phrase)
	assert.NoError(t, err)
	assert.Equal(t, 12, len(m.words))
	assert.Equal(t, phrase, m.String())
}

func TestMnemonicFromStringInvalid(t *testing.T) {
	_, err := MnemonicFromString("")
	assert.Error(t, err)
}

func TestMnemonicToSeed(t *testing.T) {
	m, err := GenerateMnemonic(Words12)
	assert.NoError(t, err)

	// Test without passphrase
	seed1, err := m.ToSeed()
	assert.NoError(t, err)
	assert.NotNil(t, seed1)
	assert.Equal(t, KeyLength, len(seed1))

	// Test with passphrase
	seed2, err := m.ToSeed("my-passphrase")
	assert.NoError(t, err)
	assert.NotNil(t, seed2)

	// Seeds should be different
	assert.NotEqual(t, seed1, seed2)
}

func TestMnemonicToKeyPair(t *testing.T) {
	m, err := GenerateMnemonic(Words12)
	assert.NoError(t, err)

	kp, err := m.ToKeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, kp)
	assert.NotNil(t, kp.PrivateKey)
	assert.NotNil(t, kp.PublicKey)
	assert.Len(t, kp.PrivateKey, ed25519.PrivateKeySize)
	assert.Len(t, kp.PublicKey, ed25519.PublicKeySize)
}

func TestMnemonicDeterministic(t *testing.T) {
	// Generate two mnemonics with same seed should produce different results
	m1, err1 := GenerateMnemonic(Words12)
	m2, err2 := GenerateMnemonic(Words12)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, m1.String(), m2.String())
}

func TestDeriveAddress(t *testing.T) {
	mnemonic := "abandon ability able about above absent absorb abstract absurd abuse access accident"

	iid1, err := DeriveAddress(mnemonic, "")
	assert.NoError(t, err)
	assert.NotZero(t, iid1)

	iid2, err := DeriveAddress(mnemonic, "m/44'/0'/0'/0/0")
	assert.NoError(t, err)
	assert.NotZero(t, iid2)

	// Different derivation paths should produce different IIDs
	assert.NotEqual(t, iid1, iid2)

	// Same derivation path should produce same IID
	iid3, err := DeriveAddress(mnemonic, "m/44'/0'/0'/0/0")
	assert.NoError(t, err)
	assert.Equal(t, iid2, iid3)
}

func TestDeriveAddressWithPassphrase(t *testing.T) {
	mnemonic := "abandon ability able about above absent absorb abstract absurd abuse access accident"

	iid1, err := DeriveAddress(mnemonic, "")
	assert.NoError(t, err)

	iid2, err := DeriveAddress(mnemonic, "", "passphrase")
	assert.NoError(t, err)

	// Different passphrases should produce different IIDs
	assert.NotEqual(t, iid1, iid2)
}

func TestGenerateMasterKey(t *testing.T) {
	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	assert.NoError(t, err)

	masterKey, err := GenerateMasterKey(seed)
	assert.NoError(t, err)
	assert.NotNil(t, masterKey)
	assert.Equal(t, 64, len(masterKey))
}

func TestGenerateMasterKeyShortSeed(t *testing.T) {
	seed := make([]byte, 8)
	_, err := GenerateMasterKey(seed)
	assert.Error(t, err)
}

func TestUint32ToBytesAndBack(t *testing.T) {
	testValues := []uint32{0, 1, 255, 256, 65535, 65536, 16777215, 16777216, 4294967295}

	for _, val := range testValues {
		bytes := Uint32ToBytes(val)
		back := BytesToUint32(bytes)
		assert.Equal(t, val, back)
	}
}

func TestBytesToBits(t *testing.T) {
	data := []byte{0x12, 0x34}
	bits := bytesToBits(data)

	assert.Equal(t, 16, len(bits))

	// 0x12 = 00010010
	assert.Equal(t, 0, bits[0])
	assert.Equal(t, 0, bits[1])
	assert.Equal(t, 0, bits[2])
	assert.Equal(t, 1, bits[3])
	assert.Equal(t, 0, bits[4])
	assert.Equal(t, 0, bits[5])
	assert.Equal(t, 1, bits[6])
	assert.Equal(t, 0, bits[7])

	// 0x34 = 00110100
	assert.Equal(t, 0, bits[8])
	assert.Equal(t, 0, bits[9])
	assert.Equal(t, 1, bits[10])
	assert.Equal(t, 1, bits[11])
	assert.Equal(t, 0, bits[12])
	assert.Equal(t, 1, bits[13])
	assert.Equal(t, 0, bits[14])
	assert.Equal(t, 0, bits[15])
}

func TestBitsToInt(t *testing.T) {
	tests := []struct {
		bits     []int
		expected int
	}{
		{[]int{0, 0, 0, 0, 0, 0, 0, 0}, 0},
		{[]int{0, 0, 0, 0, 0, 0, 0, 1}, 1},
		{[]int{0, 0, 0, 0, 1, 0, 0, 0}, 8},
		{[]int{0, 0, 0, 0, 1, 0, 0, 1}, 9},
		{[]int{1, 0, 0, 0, 0, 0, 0, 0}, 128},
		{[]int{1, 1, 1, 1, 1, 1, 1, 1}, 255},
	}

	for _, test := range tests {
		result := bitsToInt(test.bits)
		assert.Equal(t, test.expected, result)
	}
}

func TestGetEntropySize(t *testing.T) {
	tests := []struct {
		wordCount int
		expected  int
	}{
		{Words12, EntropyBitSize12},
		{Words15, 160},
		{Words18, 192},
		{Words21, 224},
		{Words24, EntropyBitSize24},
		{100, 0},
	}

	for _, test := range tests {
		result := getEntropySize(test.wordCount)
		assert.Equal(t, test.expected, result)
	}
}

func TestDeriveChildKey(t *testing.T) {
	parentKey := make([]byte, 32)
	_, err := rand.Read(parentKey)
	assert.NoError(t, err)

	// Empty path should return parent key
	child1, err := DeriveChildKey(parentKey, "")
	assert.NoError(t, err)
	assert.Equal(t, parentKey, child1)

	// Non-empty path should derive different key
	child2, err := DeriveChildKey(parentKey, "m/0")
	assert.NoError(t, err)
	assert.NotEqual(t, parentKey, child2)
}

func TestMnemonicIntegration(t *testing.T) {
	// Full integration test: Generate mnemonic -> Derive key pair -> Sign/Verify
	m, err := GenerateMnemonic(Words12)
	assert.NoError(t, err)

	kp, err := m.ToKeyPair()
	assert.NoError(t, err)

	message := []byte("Test message for V6Coin")
	signature := kp.Sign(message)

	assert.True(t, kp.Verify(message, signature))
	assert.False(t, kp.Verify([]byte("Different message"), signature))
}

func TestMnemonicKeyPairIID(t *testing.T) {
	// Test deriving IID from mnemonic-generated key pair
	m, err := GenerateMnemonic(Words12)
	assert.NoError(t, err)

	kp, err := m.ToKeyPair()
	assert.NoError(t, err)

	iid := GenerateIID(kp.PublicKey)
	assert.NotZero(t, iid)

	// Same mnemonic should generate same IID
	kp2, err := m.ToKeyPair()
	assert.NoError(t, err)

	iid2 := GenerateIID(kp2.PublicKey)
	assert.Equal(t, iid, iid2)
}

func BenchmarkMnemonicGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateMnemonic(Words12)
	}
}

func BenchmarkMnemonicToSeed(b *testing.B) {
	m, _ := GenerateMnemonic(Words12)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.ToSeed()
	}
}

func BenchmarkMnemonicToKeyPair(b *testing.B) {
	m, _ := GenerateMnemonic(Words12)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.ToKeyPair()
	}
}

func BenchmarkDeriveAddress(b *testing.B) {
	mnemonic := "abandon ability able about above absent absorb abstract absurd abuse access accident"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = DeriveAddress(mnemonic, "")
	}
}
