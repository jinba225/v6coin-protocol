package wallet

import (
	"fmt"
	"net"

	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

const (
	Words12 = 12
	Words15 = 15
	Words18 = 18
	Words21 = 21
	Words24 = 24
)

// MnemonicWallet represents a wallet with BIP-39 mnemonic support
type MnemonicWallet struct {
	Mnemonic *crypto.Mnemonic
	Language string
	KeyPair  *KeyPair
}

// GenerateMnemonic generates a new BIP-39 mnemonic phrase
func GenerateMnemonic(wordCount int) (*MnemonicWallet, error) {
	mnemonic, err := crypto.GenerateMnemonic(wordCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return &MnemonicWallet{
		Mnemonic: mnemonic,
		Language: "en",
		KeyPair:  nil,
	}, nil
}

// MnemonicWalletFromPhrase creates a wallet from an existing mnemonic phrase
func MnemonicWalletFromPhrase(phrase string, passphrase ...string) (*MnemonicWallet, error) {
	mnemonic, err := crypto.MnemonicFromString(phrase)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic phrase: %w", err)
	}

	keyPair, err := mnemonic.ToKeyPair(passphrase...)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key pair from mnemonic: %w", err)
	}

	v6KeyPair := &KeyPair{
		PrivateKey: keyPair.PrivateKey,
		PublicKey:  keyPair.PublicKey,
		Address:    nil,
	}

	return &MnemonicWallet{
		Mnemonic: mnemonic,
		Language: "en",
		KeyPair:  v6KeyPair,
	}, nil
}

// MnemonicWalletFromPhraseWithPrefix creates a wallet from mnemonic with custom network prefix
func MnemonicWalletFromPhraseWithPrefix(phrase string, networkPrefix net.IP, passphrase ...string) (*MnemonicWallet, error) {
	mnemonic, err := crypto.MnemonicFromString(phrase)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic phrase: %w", err)
	}

	keyPair, err := mnemonic.ToKeyPair(passphrase...)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key pair from mnemonic: %w", err)
	}

	v6KeyPair, err := KeyPairFromPrivateKey(keyPair.PrivateKey, networkPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create V6Coin key pair: %w", err)
	}

	return &MnemonicWallet{
		Mnemonic: mnemonic,
		Language: "en",
		KeyPair:  v6KeyPair,
	}, nil
}

// ToSeed converts the mnemonic to a seed
func (mw *MnemonicWallet) ToSeed(passphrase ...string) ([]byte, error) {
	if mw.Mnemonic == nil {
		return nil, fmt.Errorf("mnemonic not initialized")
	}

	return mw.Mnemonic.ToSeed(passphrase...)
}

// DeriveKeyPair derives the Ed25519 key pair from the mnemonic
func (mw *MnemonicWallet) DeriveKeyPair(networkPrefix ...net.IP) error {
	if mw.Mnemonic == nil {
		return fmt.Errorf("mnemonic not initialized")
	}

	keyPair, err := mw.Mnemonic.ToKeyPair()
	if err != nil {
		return fmt.Errorf("failed to derive key pair: %w", err)
	}

	v6KeyPair, err := KeyPairFromPrivateKey(keyPair.PrivateKey, networkPrefix...)
	if err != nil {
		return fmt.Errorf("failed to create V6Coin key pair: %w", err)
	}

	mw.KeyPair = v6KeyPair
	return nil
}

// GetMnemonicPhrase returns the mnemonic phrase as a string
func (mw *MnemonicWallet) GetMnemonicPhrase() string {
	if mw.Mnemonic == nil {
		return ""
	}
	return mw.Mnemonic.String()
}

// GetMnemonicWords returns the mnemonic words as a slice
func (mw *MnemonicWallet) GetMnemonicWords() []string {
	if mw.Mnemonic == nil {
		return nil
	}
	return mw.Mnemonic.Words()
}

// ValidatePhrase validates a BIP-39 mnemonic phrase
func ValidatePhrase(phrase string) error {
	return crypto.ValidateMnemonic(phrase)
}

// GenerateNewWallet creates a new wallet with mnemonic and key pair
func GenerateNewWallet(wordCount int, networkPrefix ...net.IP) (*MnemonicWallet, error) {
	mnemonic, err := crypto.GenerateMnemonic(wordCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	keyPair, err := mnemonic.ToKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to derive key pair: %w", err)
	}

	v6KeyPair, err := KeyPairFromPrivateKey(keyPair.PrivateKey, networkPrefix...)
	if err != nil {
		return nil, fmt.Errorf("failed to create V6Coin key pair: %w", err)
	}

	return &MnemonicWallet{
		Mnemonic: mnemonic,
		Language: "en",
		KeyPair:  v6KeyPair,
	}, nil
}

// DeriveAddress derives a V6Coin address from mnemonic
func (mw *MnemonicWallet) DeriveAddress(networkPrefix ...net.IP) (net.IP, error) {
	if mw.KeyPair == nil {
		if err := mw.DeriveKeyPair(networkPrefix...); err != nil {
			return nil, err
		}
	}

	return mw.KeyPair.GetAddress(), nil
}

// DeriveAddressWithPath derives an address using a derivation path
func DeriveAddressWithPath(mnemonic, derivationPath string, passphrase ...string) (uint64, error) {
	return crypto.DeriveAddress(mnemonic, derivationPath, passphrase...)
}
