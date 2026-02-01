package address

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

func TestGenerateIID(t *testing.T) {
	publicKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		publicKey[i] = byte(i)
	}

	iid := GenerateIID(publicKey)
	assert.NotZero(t, iid)
	assert.Equal(t, uint64(0), iid&1) // IID should have bit 0 = 0 for unicast
}

func TestGenerateAddress(t *testing.T) {
	networkPrefix := net.ParseIP("2001:db8:85a3::/32")
	privateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		privateKey[i] = byte(i)
	}

	addr, err := GenerateAddress(networkPrefix, privateKey)
	assert.NoError(t, err)
	assert.NotNil(t, addr)
	assert.NotNil(t, addr.NetworkPrefix)
	assert.NotZero(t, addr.IID)
	assert.Equal(t, networkPrefix, addr.NetworkPrefix)
	assert.False(t, addr.IsTemporary)
	assert.Equal(t, AddressTypeCore, addr.AddressType)
}

func TestGenerateAddressInvalidPrefix(t *testing.T) {
	networkPrefix := net.IPv4(192, 168, 1, 1) // Wrong length
	privateKey := make([]byte, 32)

	_, err := GenerateAddress(networkPrefix, privateKey)
	assert.Error(t, err)
}

func TestGenerateAddressInvalidKey(t *testing.T) {
	networkPrefix := net.ParseIP("2001:db8::/32")
	privateKey := make([]byte, 16) // Wrong length

	_, err := GenerateAddress(networkPrefix, privateKey)
	assert.Error(t, err)
}

func TestGetFullAddress(t *testing.T) {
	addr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}

	fullAddr := addr.GetFullAddress()
	expectedPrefix := []byte{0x20, 0x01, 0x0d, 0xb8, 0x85, 0xa3}

	assert.Equal(t, 16, len(fullAddr))
	assert.Equal(t, expectedPrefix, fullAddr[0:8])
	assert.Equal(t, []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}, fullAddr[8:16])
}

func TestGetFullAddressDefaultPrefix(t *testing.T) {
	addr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		AddressType: AddressTypeCore,
		CreatedAt:   time.Now(),
	}

	fullAddr := addr.GetFullAddress()
	assert.NotNil(t, fullAddr)
	assert.True(t, fullAddr.IsGlobalUnicast())
	assert.Equal(t, 16, len(fullAddr))
}

func TestIsExpired(t *testing.T) {
	temporaryAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: true,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Already expired
	}

	assert.True(t, temporaryAddr.IsExpired())

	coreAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		CreatedAt:   time.Now(),
	}

	assert.False(t, coreAddr.IsExpired())
}

func TestGenerateTemporaryAddress(t *testing.T) {
	corePrivateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		corePrivateKey[i] = byte(i)
	}

	addr, err := GenerateTemporaryAddress(corePrivateKey, 0)
	assert.NoError(t, err)
	assert.NotNil(t, addr)
	assert.True(t, addr.IsTemporary)
	assert.Equal(t, AddressTypeTemporary, addr.AddressType)
	assert.Equal(t, 0, addr.Index)
}

func TestGenerateTemporaryAddressInvalidKey(t *testing.T) {
	corePrivateKey := make([]byte, 16)

	_, err := GenerateTemporaryAddress(corePrivateKey, 0)
	assert.Error(t, err)
}

func TestGenerateTemporaryAddressWithPrefix(t *testing.T) {
	networkPrefix := net.ParseIP("2001:db8::/32")
	corePrivateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		corePrivateKey[i] = byte(i)
	}

	expiry := 12 * time.Hour
	addr, err := GenerateTemporaryAddressWithPrefix(networkPrefix, corePrivateKey, 1, expiry)
	assert.NoError(t, err)
	assert.NotNil(t, addr)
	assert.True(t, addr.IsTemporary)
	assert.Equal(t, networkPrefix, addr.NetworkPrefix)
	assert.Equal(t, 1, addr.Index)
}

func TestCreateLogicalSubnet(t *testing.T) {
	coreAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		AddressType: AddressTypeCore,
		CreatedAt:   time.Now(),
	}

	prefixes := []net.IP{
		net.ParseIP("2001:db8::/32"),
		net.ParseIP("2001:db8::/33"),
		net.ParseIP("2001:db8::/34"),
	}

	subnet, err := CreateLogicalSubnet(coreAddr, prefixes)
	assert.NoError(t, err)
	assert.NotNil(t, subnet)
	assert.Equal(t, coreAddr.IID, subnet.CoreAddress.IID)
	assert.Equal(t, len(prefixes), len(subnet.SubnetPrefixes))
}

func TestCreateLogicalSubnetNoPrefixes(t *testing.T) {
	coreAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		AddressType: AddressTypeCore,
		CreatedAt:   time.Now(),
	}

	_, err := CreateLogicalSubnet(coreAddr, []net.IP{})
	assert.Error(t, err)
}

func TestCreateLogicalSubnetTooManyPrefixes(t *testing.T) {
	coreAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		AddressType: AddressTypeCore,
		CreatedAt:   time.Now(),
	}

	prefixes := make([]net.IP, 9) // More than MaxLogicalSubnets (8)

	_, err := CreateLogicalSubnet(coreAddr, prefixes)
	assert.Error(t, err)
}

func TestGenerateSubnetAddress(t *testing.T) {
	coreAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		AddressType: AddressTypeCore,
		CreatedAt:   time.Now(),
	}

	prefixes := []net.IP{
		net.ParseIP("2001:db8::/32"),
		net.ParseIP("2001:db8::/33"),
	}

	subnet, _ := CreateLogicalSubnet(coreAddr, prefixes)

	addr, err := GenerateSubnetAddress(subnet, 1)
	assert.NoError(t, err)
	assert.NotNil(t, addr)
	assert.Equal(t, coreAddr.IID, addr.IID)
	assert.Equal(t, prefixes[1], addr.NetworkPrefix)
	assert.Equal(t, 1, addr.Index)
	assert.Equal(t, AddressTypeLogicalSubnet, addr.AddressType)
}

func TestGenerateSubnetAddressInvalidIndex(t *testing.T) {
	coreAddr := &V6Address{
		IID: 0x123456789ABCDEF0,
	}

	subnet, _ := CreateLogicalSubnet(coreAddr, []net.IP{})

	_, err := GenerateSubnetAddress(subnet, 0) // Index 0 has no prefixes
	assert.Error(t, err)
}

func TestIsInSameLogicalSubnet(t *testing.T) {
	coreAddr := &V6Address{
		IID: 0x123456789ABCDEF0,
	}

	prefixes := []net.IP{
		net.ParseIP("2001:db8::/32"),
		net.ParseIP("2001:db8::/33"),
	}

	subnet, _ := CreateLogicalSubnet(coreAddr, prefixes)

	addr1, _ := GenerateSubnetAddress(subnet, 0)
	addr2, _ := GenerateSubnetAddress(subnet, 1)

	assert.True(t, IsInSameLogicalSubnet(addr1, addr2, subnet))
	assert.Equal(t, addr1.IID, addr2.IID)
}

func TestIsInSameLogicalSubnetDifferentIID(t *testing.T) {
	prefixes := []net.IP{
		net.ParseIP("2001:db8::/32"),
	}

	coreAddr := &V6Address{
		IID: 0x123456789ABCDEF0,
	}

	subnet, _ := CreateLogicalSubnet(coreAddr, prefixes)

	addr1, _ := GenerateSubnetAddress(subnet, 0)

	differentIIDAddr := &V6Address{
		IID: 0x123456789ABCDEF1, // Different IID
	}

	assert.False(t, IsInSameLogicalSubnet(addr1, differentIIDAddr, subnet))
}

func TestCreateMigration(t *testing.T) {
	privateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		privateKey[i] = byte(i)
	}

	fromAddr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}

	toAddr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/33"),
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}

	migration, err := CreateMigration(fromAddr, toAddr, privateKey)
	assert.NoError(t, err)
	assert.NotNil(t, migration)
	assert.NotNil(t, migration.FromAddress)
	assert.NotNil(t, migration.ToAddress)
	assert.NotEmpty(t, migration.Signature)
}

func TestCreateMigrationNoAddresses(t *testing.T) {
	privateKey := make([]byte, 32)

	_, err := CreateMigration(nil, nil, privateKey)
	assert.Error(t, err)
}

func TestCreateMigrationInvalidKey(t *testing.T) {
	privateKey := make([]byte, 16)

	fromAddr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
	}

	toAddr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/33"),
	}

	_, err := CreateMigration(fromAddr, toAddr, privateKey)
	assert.Error(t, err)
}

func TestValidateMigration(t *testing.T) {
	privateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		privateKey[i] = byte(i)
	}

	kp, _ := crypto.KeyPairFromPrivateKey(privateKey)
	publicKey := kp.PublicKey

	fromAddr := net.ParseIP("2001:db8::1234:5678::abcd")
	toAddr := net.ParseIP("2001:db8:1234:5678::abce")

	migration, _ := CreateMigration(&V6Address{
		NetworkPrefix: fromAddr[0:8],
	}, &V6Address{
		NetworkPrefix: toAddr[0:8],
	}, privateKey)

	err := ValidateMigration(migration, publicKey)
	assert.NoError(t, err)
}

func TestValidateMigrationInvalidSignature(t *testing.T) {
	publicKey := make([]byte, 32)

	fromAddr := net.ParseIP("2001:db8::1234:5678::abcd")
	toAddr := net.ParseIP("2001:db8::1234:5678::abce")

	migration := &AddressMigration{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Timestamp:   uint64(time.Now().Unix()),
		Signature:   make([]byte, 64),
	}

	err := ValidateMigration(migration, publicKey)
	assert.Error(t, err)
}

func TestValidateMigrationOldTimestamp(t *testing.T) {
	privateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		privateKey[i] = byte(i)
	}

	kp, _ := crypto.KeyPairFromPrivateKey(privateKey)
	publicKey := kp.PublicKey

	fromAddr := net.ParseIP("2001:db8::1234:5678::abcd")
	toAddr := net.ParseIP("2001:db8::1234:5678::abce")

	migration := &AddressMigration{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Timestamp:   uint64(time.Now().Add(-8 * 24 * time.Hour).Unix()), // 8 days old
		Signature:   []byte{},
	}

	err := ValidateMigration(migration, publicKey)
	assert.Error(t, err)
}

func TestValidateMigrationNilMigration(t *testing.T) {
	privateKey := make([]byte, 32)
	kp, _ := crypto.KeyPairFromPrivateKey(privateKey)

	err := ValidateMigration(nil, kp.PublicKey)
	assert.Error(t, err)
}

func TestValidateAddress(t *testing.T) {
	validAddr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
		Index:         0,
	}

	err := ValidateAddress(validAddr)
	assert.NoError(t, err)
}

func TestValidateAddressInvalidIID(t *testing.T) {
	invalidAddr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
		IID:           0x123456789ABCDEF1, // bit 0 = 1 (multicast)
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
		Index:         0,
	}

	err := ValidateAddress(invalidAddr)
	assert.Error(t, err)
}

func TestValidateAddressInvalidPrefix(t *testing.T) {
	invalidAddr := &V6Address{
		NetworkPrefix: net.IPv4(192, 168, 1, 1), // Only 4 bytes
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}

	err := ValidateAddress(invalidAddr)
	assert.Error(t, err)
}

func TestValidateExpiredTemporaryAddress(t *testing.T) {
	expiredAddr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: true,
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
		CreatedAt:   time.Now(),
	}

	err := ValidateAddress(expiredAddr)
	assert.Error(t, err)
}

func TestExtractIID(t *testing.T) {
	fullAddr := net.ParseIP("2001:db8:1234:5678::abcd:1234:5678:1234")
	expectedIID := binary.BigEndian.Uint64(fullAddr[8:16])

	iid := ExtractIID(fullAddr)
	assert.Equal(t, expectedIID, iid)
}

func TestExtractNetworkPrefix(t *testing.T) {
	fullAddr := net.ParseIP("2001:db8:1234:5678::abcd:1234:5678:1234")
	expectedPrefix := fullAddr[0:8]

	prefix := ExtractNetworkPrefix(fullAddr)
	assert.Equal(t, expectedPrefix, prefix)
}

func TestString(t *testing.T) {
	addr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: false,
		AddressType: AddressTypeCore,
		CreatedAt:   time.Now(),
		Index:       0,
	}

	str := addr.String()
	assert.Contains(t, str, "Core[0]")
	assert.NotEmpty(t, str)
}

func TestStringTemporary(t *testing.T) {
	addr := &V6Address{
		IID:         0x123456789ABCDEF0,
		IsTemporary: true,
		AddressType: AddressTypeTemporary,
		CreatedAt:   time.Now(),
		Index:       5,
	}

	str := addr.String()
	assert.Contains(t, str, "Temporary[5]")
	assert.NotEmpty(t, str)
}

func TestEquals(t *testing.T) {
	networkPrefix := net.ParseIP("2001:db8::/32")

	addr1 := &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
		Index:         0,
	}

	addr2 := &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
		Index:         0,
	}

	assert.True(t, addr1.Equals(addr2))
	assert.True(t, addr2.Equals(addr1))
}

func TestEqualsDifferent(t *testing.T) {
	networkPrefix := net.ParseIP("2001:db8::/32")

	addr1 := &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
		Index:         0,
	}

	addr2 := &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           0x123456789ABCDEF1, // Different IID
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
		Index:         0,
	}

	assert.False(t, addr1.Equals(addr2))
}

func TestEqualsTemporaryDifferent(t *testing.T) {
	networkPrefix := net.ParseIP("2001:db8::/32")

	addr1 := &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}

	addr2 := &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           0x123456789ABCDEF0,
		IsTemporary:   true, // Different
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}

	assert.False(t, addr1.Equals(addr2))
}

func TestGetMigrationData(t *testing.T) {
	fromAddr := net.ParseIP("2001:db8::1")
	toAddr := net.ParseIP("2001:db8::2")
	timestamp := uint64(1640000000)

	data := GetMigrationData(fromAddr, toAddr, timestamp)
	assert.Equal(t, 32, len(data))
	assert.Equal(t, fromAddr, data[0:32])
	assert.Equal(t, toAddr, data[32:64])
	assert.Equal(t, []byte{0x61, 0xd0, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00}, data[64:72])
}

func TestAddressRoundtrip(t *testing.T) {
	original := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
		IID:           0x123456789ABCDEF0,
		IsTemporary:   true,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Index:         3,
	}

	fullAddr := original.GetFullAddress()
	extractedIID := ExtractIID(fullAddr)
	extractedPrefix := ExtractNetworkPrefix(fullAddr)

	newAddr := &V6Address{
		NetworkPrefix: extractedPrefix,
		IID:           extractedIID,
		IsTemporary:   original.IsTemporary,
		AddressType:   original.AddressType,
		CreatedAt:     original.CreatedAt,
		ExpiresAt:     original.ExpiresAt,
		Index:         original.Index,
	}

	assert.True(t, original.Equals(newAddr))
}

func BenchmarkGenerateIID(b *testing.B) {
	publicKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		publicKey[i] = byte(i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GenerateIID(publicKey)
	}
}

func BenchmarkGenerateAddress(b *testing.B) {
	networkPrefix := net.ParseIP("2001:db8::/32")
	privateKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		privateKey[i] = byte(i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = GenerateAddress(networkPrefix, privateKey)
	}
}

func BenchmarkGetFullAddress(b *testing.B) {
	addr := &V6Address{
		NetworkPrefix: net.ParseIP("2001:db8::/32"),
		IID:           0x123456789ABCDEF0,
		IsTemporary:   false,
		CreatedAt:     time.Now(),
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = addr.GetFullAddress()
	}
}
