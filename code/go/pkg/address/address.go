package address

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

const (
	// IID generation constants
	DefaultIIDPrefix      = uint64(0)
	MaxLogicalSubnets   = 8
	MinMigrationInterval = time.Hour
	MaxMigrationInterval = 30 * 24 * time.Hour // 30 days
)

// AddressType represents the type of IPv6 address
type AddressType int

const (
	AddressTypeCore     AddressType = iota // Core private key address
	AddressTypeTemporary                         // Temporary derived address
	AddressTypeLogicalSubnet                     // Logical subnet address
)

// V6Address represents a V6Coin IPv6 address
type V6Address struct {
	NetworkPrefix net.IP // 64-bit network prefix
	IID           uint64 // 64-bit interface identifier (derived from private key)
	IsTemporary   bool   // Whether this is a temporary address
	AddressType   AddressType
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Index         int     // Index for temporary addresses or logical subnets
}

// AddressMigration represents an asset migration between addresses
type AddressMigration struct {
	FromAddress   net.IP
	ToAddress     net.IP
	Timestamp     uint64
	Signature     []byte
	PreviousIndex int // Previous address index in logical subnet
}

// LogicalSubnet represents a logical subnet for mobile devices
type LogicalSubnet struct {
	CoreAddress   V6Address
	SubnetPrefixes []net.IP // Up to 8 associated prefixes
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// GenerateIID generates 64-bit Interface Identifier from public key
func GenerateIID(publicKey []byte) uint64 {
	hash := sha256.Sum256(publicKey)
	iid := binary.BigEndian.Uint64(hash[:8])
	iid &^= 1
	return iid
}

// GenerateAddress generates a V6Coin IPv6 address from network prefix and private key
func GenerateAddress(networkPrefix net.IP, privateKey []byte) (*V6Address, error) {
	if len(networkPrefix) != 8 {
		return nil, errors.New("network prefix must be 64 bits (8 bytes)")
	}
	if len(privateKey) != 32 {
		return nil, errors.New("private key must be 32 bytes (Ed25519)")
	}

	kp, err := crypto.KeyPairFromPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %w", err)
	}

	iid := GenerateIID(kp.PublicKey)
	prefix := make(net.IP, 16)
	copy(prefix, networkPrefix)
	binary.BigEndian.PutUint64(prefix[8:], iid)

	return &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           iid,
		IsTemporary:   false,
		AddressType:   AddressTypeCore,
		CreatedAt:     time.Now(),
	}, nil
}

// GenerateTemporaryAddress generates a temporary IPv6 address from core private key
func GenerateTemporaryAddress(corePrivateKey []byte, index int) (*V6Address, error) {
	if len(corePrivateKey) != 32 {
		return nil, errors.New("core private key must be 32 bytes")
	}

	h := sha256.New()
	h.Write(corePrivateKey)
	h.Write([]byte{byte(index)})
	keyMaterial := h.Sum(nil)
	tempPrivateKey := keyMaterial[:32]
	kp, err := crypto.KeyPairFromPrivateKey(tempPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive temporary key: %w", err)
	}

	iid := GenerateIID(kp.PublicKey)

	return &V6Address{
		IID:         iid,
		IsTemporary: true,
		AddressType: AddressTypeTemporary,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Index:       index,
	}, nil
}

// GenerateTemporaryAddressWithPrefix generates a temporary address with specific network prefix
func GenerateTemporaryAddressWithPrefix(networkPrefix net.IP, corePrivateKey []byte, index int, expiry time.Duration) (*V6Address, error) {
	if len(networkPrefix) != 8 {
		return nil, errors.New("network prefix must be 64 bits (8 bytes)")
	}
	if len(corePrivateKey) != 32 {
		return nil, errors.New("core private key must be 32 bytes")
	}

	h := sha256.New()
	h.Write(corePrivateKey)
	h.Write([]byte{byte(index)})
	keyMaterial := h.Sum(nil)
	tempPrivateKey := keyMaterial[:32]
	
	kp, err := crypto.KeyPairFromPrivateKey(tempPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive temporary key: %w", err)
	}

	iid := GenerateIID(kp.PublicKey)

	prefix := make(net.IP, 16)
	copy(prefix, networkPrefix)
	binary.BigEndian.PutUint64(prefix[8:], iid)

	return &V6Address{
		NetworkPrefix: networkPrefix,
		IID:           iid,
		IsTemporary:   true,
		AddressType: AddressTypeTemporary,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(expiry),
		Index:         index,
	}, nil
}

func (addr *V6Address) GetFullAddress() net.IP {
	if addr.NetworkPrefix == nil {
		prefix := make(net.IP, 16)
		prefix[0] = 0x20
		prefix[1] = 0x01
		copy(prefix[2:8], make([]byte, 6))
		binary.BigEndian.PutUint64(prefix[8:], addr.IID)
		return prefix
	}

	fullAddr := make(net.IP, 16)
	copy(fullAddr, addr.NetworkPrefix)
	binary.BigEndian.PutUint64(fullAddr[8:], addr.IID)
	return fullAddr
}

func (addr *V6Address) IsExpired() bool {
	if !addr.IsTemporary {
		return false
	}
	return time.Now().After(addr.ExpiresAt)
}

func CreateLogicalSubnet(coreAddress *V6Address, prefixes []net.IP) (*LogicalSubnet, error) {
	if coreAddress == nil {
		return nil, errors.New("core address cannot be nil")
	}
	if len(prefixes) == 0 {
		return nil, errors.New("at least one prefix required")
	}
	if len(prefixes) > MaxLogicalSubnets {
		return nil, fmt.Errorf("too many prefixes (max %d)", MaxLogicalSubnets)
	}

	for i, prefix := range prefixes {
		if len(prefix) != 8 {
			return nil, fmt.Errorf("prefix %d must be 64 bits", i)
		}
	}

	return &LogicalSubnet{
		CoreAddress:   *coreAddress,
		SubnetPrefixes: prefixes,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
	}, nil
}

func GenerateSubnetAddress(subnet *LogicalSubnet, prefixIndex int) (*V6Address, error) {
	if subnet == nil {
		return nil, errors.New("subnet cannot be nil")
	}
	if prefixIndex < 0 || prefixIndex >= len(subnet.SubnetPrefixes) {
		return nil, fmt.Errorf("prefix index %d out of range [0, %d]", prefixIndex, len(subnet.SubnetPrefixes)-1)
	}

	prefix := subnet.SubnetPrefixes[prefixIndex]
	
	return &V6Address{
		NetworkPrefix: prefix,
		IID:           subnet.CoreAddress.IID,
		IsTemporary:   false,
		AddressType:   AddressTypeLogicalSubnet,
		CreatedAt:     time.Now(),
		Index:         prefixIndex,
	}, nil
}

func IsInSameLogicalSubnet(addr1, addr2 *V6Address, subnet *LogicalSubnet) bool {
	if subnet == nil {
		return false
	}
	
	return addr1.IID == addr2.IID &&
		containsPrefix(subnet.SubnetPrefixes, addr1.NetworkPrefix) &&
		containsPrefix(subnet.SubnetPrefixes, addr2.NetworkPrefix)
}

func containsPrefix(prefixes []net.IP, prefix net.IP) bool {
	for _, p := range prefixes {
		if p.Equal(prefix) {
			return true
		}
	}
	return false
}

func CreateMigration(fromAddr, toAddr *V6Address, privateKey []byte) (*AddressMigration, error) {
	if fromAddr == nil || toAddr == nil {
		return nil, errors.New("both from and to addresses must be provided")
	}
	if len(privateKey) != 32 {
		return nil, errors.New("private key must be 32 bytes")
	}

	fromIP := fromAddr.GetFullAddress()
	toIP := toAddr.GetFullAddress()

	data := GetMigrationData(fromIP, toIP, uint64(time.Now().Unix()))

	signature, err := crypto.Sign(privateKey, data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign migration: %w", err)
	}

	return &AddressMigration{
		FromAddress:   fromIP,
		ToAddress:     toIP,
		Timestamp:     uint64(time.Now().Unix()),
		Signature:     signature,
		PreviousIndex: fromAddr.Index,
	}, nil
}

func ValidateMigration(migration *AddressMigration, publicKey []byte) error {
	if migration == nil {
		return errors.New("migration cannot be nil")
	}
	if len(publicKey) != 32 {
		return errors.New("public key must be 32 bytes")
	}

	data := GetMigrationData(migration.FromAddress, migration.ToAddress, migration.Timestamp)

	if !crypto.Verify(publicKey, data, migration.Signature) {
		return errors.New("invalid migration signature")
	}

	age := time.Since(time.Unix(int64(migration.Timestamp), 0))
	if age > 7*24*time.Hour {
		return errors.New("migration timestamp too old (max 7 days)")
	}

	return nil
}

func ValidateAddress(addr *V6Address) error {
	if addr == nil {
		return errors.New("address cannot be nil")
	}

	if addr.IID&1 != 0 {
		return errors.New("IID must be unicast address (bit 0 = 0)")
	}

	if addr.NetworkPrefix != nil && len(addr.NetworkPrefix) != 8 {
		return errors.New("network prefix must be 64 bits (8 bytes)")
	}

	if addr.IsTemporary {
		if addr.CreatedAt.IsZero() {
			return errors.New("temporary address must have creation time")
		}
		if time.Now().After(addr.ExpiresAt) {
			return errors.New("temporary address has expired")
		}
	}

	return nil
}

func ExtractIID(fullAddr net.IP) uint64 {
	if len(fullAddr) != 16 {
		return 0
	}
	return binary.BigEndian.Uint64(fullAddr[8:16])
}

func ExtractNetworkPrefix(fullAddr net.IP) net.IP {
	if len(fullAddr) != 16 {
		return nil
	}
	return fullAddr[0:8]
}

func (addr *V6Address) String() string {
	fullAddr := addr.GetFullAddress()
	addrType := "Core"
	if addr.IsTemporary {
		addrType = "Temporary"
	}
	return fmt.Sprintf("%s[%d] %s", addrType, addr.Index, fullAddr.String())
}

func (addr *V6Address) Equals(other *V6Address) bool {
	if addr == nil && other == nil {
		return true
	}
	if addr == nil || other == nil {
		return false
	}
	return addr.IID == other.IID &&
		addr.NetworkPrefix.Equal(other.NetworkPrefix) &&
		addr.IsTemporary == other.IsTemporary &&
		addr.Index == other.Index
}

func GetMigrationData(fromAddr, toAddr net.IP, timestamp uint64) []byte {
	data := make([]byte, 16+16+8)
	copy(data[0:16], fromAddr)
	copy(data[16:32], toAddr)
	binary.BigEndian.PutUint64(data[32:40], timestamp)
	return data
}
