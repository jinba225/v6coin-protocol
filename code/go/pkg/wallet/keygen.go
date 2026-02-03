package wallet

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/jinba225/v6coin-protocol/pkg/address"
	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

const (
	// DefaultNetworkPrefix is the default IPv6 network prefix for V6Coin
	DefaultNetworkPrefix = "\x20\x01\x0d\xb8\x00\x00\x00\x00" // 2001:db8::/64
)

// KeyPair represents a V6Coin key pair with associated address
type KeyPair struct {
	PrivateKey ed25519.PrivateKey `json:"-"`
	PublicKey  ed25519.PublicKey  `json:"-"`
	Address    *address.V6Address `json:"address"`
}

// GenerateKeyPair generates a new Ed25519 key pair and derives the CGA address
func GenerateKeyPair(networkPrefix ...net.IP) (*KeyPair, error) {
	// Generate Ed25519 key pair using crypto package
	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Use provided prefix or default
	prefix := parseNetworkPrefix(networkPrefix...)

	// Generate V6Address from private key
	v6Addr, err := address.GenerateAddress(prefix, kp.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return &KeyPair{
		PrivateKey: kp.PrivateKey,
		PublicKey:  kp.PublicKey,
		Address:    v6Addr,
	}, nil
}

// GenerateKeyPairFromSeed generates a key pair from a seed
func GenerateKeyPairFromSeed(seed []byte, networkPrefix ...net.IP) (*KeyPair, error) {
	if len(seed) < 32 {
		return nil, fmt.Errorf("seed too short: %d bytes (minimum 32)", len(seed))
	}

	// Use first 32 bytes as Ed25519 seed
	ed25519Seed := seed[:32]
	privateKey := ed25519.NewKeyFromSeed(ed25519Seed)
	publicKey := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(publicKey, privateKey[32:])

	// Use provided prefix or default
	prefix := parseNetworkPrefix(networkPrefix...)

	// Generate V6Address
	v6Addr, err := address.GenerateAddress(prefix, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    v6Addr,
	}, nil
}

// GenerateTemporaryKeyPair generates a temporary key pair for privacy
func GenerateTemporaryKeyPair(corePrivateKey ed25519.PrivateKey, index int, networkPrefix ...net.IP) (*KeyPair, error) {
	if len(corePrivateKey) != 32 {
		return nil, fmt.Errorf("core private key must be 32 bytes")
	}

	// Use provided prefix or default
	prefix := parseNetworkPrefix(networkPrefix...)

	// Generate temporary address with prefix
	tempAddr, err := address.GenerateTemporaryAddressWithPrefix(prefix, corePrivateKey, index, 24*3600) // 24 hours
	if err != nil {
		return nil, fmt.Errorf("failed to generate temporary address: %w", err)
	}

	tempKey := deriveTemporaryKey(corePrivateKey, index)

	kp, err := crypto.KeyPairFromPrivateKey(tempKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive temporary key pair: %w", err)
	}

	return &KeyPair{
		PrivateKey: kp.PrivateKey,
		PublicKey:  kp.PublicKey,
		Address:    tempAddr,
	}, nil
}

func deriveTemporaryKey(coreKey ed25519.PrivateKey, index int) []byte {
	h := crypto.HashSHA256(append(coreKey, byte(index)))
	return h[:32]
}

// parseNetworkPrefix parses network prefix arguments
func parseNetworkPrefix(prefixes ...net.IP) net.IP {
	if len(prefixes) > 0 && prefixes[0] != nil {
		if len(prefixes[0]) >= 8 {
			return prefixes[0][:8]
		}
	}
	return net.ParseIP(DefaultNetworkPrefix)
}

// GetPublicKeyHex returns the public key as hex string
func (kp *KeyPair) GetPublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey)
}

// GetPrivateKeyHex returns the private key as hex string (WARNING: handle with care)
func (kp *KeyPair) GetPrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey)
}

// GetAddress returns the IPv6 address
func (kp *KeyPair) GetAddress() net.IP {
	return kp.Address.GetFullAddress()
}

// GetAddressString returns the IPv6 address as string
func (kp *KeyPair) GetAddressString() string {
	return kp.Address.GetFullAddress().String()
}

// GetIID returns the 64-bit Interface Identifier
func (kp *KeyPair) GetIID() uint64 {
	return kp.Address.IID
}

// IsTemporary returns true if this is a temporary key pair
func (kp *KeyPair) IsTemporary() bool {
	return kp.Address.IsTemporary
}

// GetNetworkPrefix returns the network prefix
func (kp *KeyPair) GetNetworkPrefix() net.IP {
	return kp.Address.NetworkPrefix
}

// KeyPairFromPrivateKey creates a KeyPair from a private key
func KeyPairFromPrivateKey(privateKey []byte, networkPrefix ...net.IP) (*KeyPair, error) {
	kp, err := crypto.KeyPairFromPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	prefix := parseNetworkPrefix(networkPrefix...)
	v6Addr, err := address.GenerateAddress(prefix, kp.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: kp.PrivateKey,
		PublicKey:  kp.PublicKey,
		Address:    v6Addr,
	}, nil
}

// KeyPairFromPublicKeyHex creates a KeyPair from a public key hex string (read-only)
func KeyPairFromPublicKeyHex(pubKeyHex string) (*KeyPair, error) {
	pubKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key hex: %w", err)
	}

	if len(pubKey) != 32 {
		return nil, fmt.Errorf("invalid public key length: %d (expected 32)", len(pubKey))
	}

	// For public key only, we can only extract IID
	iid := address.GenerateIID(pubKey)
	v6Addr := &address.V6Address{
		IID:         iid,
		IsTemporary: false,
		AddressType: address.AddressTypeCore,
	}

	return &KeyPair{
		PublicKey:  ed25519.PublicKey(pubKey),
		PrivateKey: nil, // No private key available
		Address:    v6Addr,
	}, nil
}
