package ipv6

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHopByHopOptionsHeader(t *testing.T) {
	header := NewHopByHopOptionsHeader(NextHeaderNone)
	assert.Equal(t, uint8(NextHeaderNone), header.NextHeader)
	assert.Equal(t, uint8(0), header.HdrExtLen)
	assert.NotNil(t, header.Options)
	assert.Len(t, header.Options, 0)
}

func TestSerializeIPv6Header(t *testing.T) {
	header := &IPv6Header{
		Version:       6,
		TrafficClass:  0,
		FlowLabel:     0,
		PayloadLength: 100,
		NextHeader:    NextHeaderHopByHopOptions,
		HopLimit:      64,
		SourceAddr:    net.ParseIP("2001:db8::1"),
		DestAddr:      net.ParseIP("2001:db8::2"),
	}

	data, err := SerializeIPv6Header(header)
	assert.NoError(t, err)
	assert.Len(t, data, 40)
}

func TestParseIPv6Header(t *testing.T) {
	original := &IPv6Header{
		Version:       6,
		TrafficClass:  0,
		FlowLabel:     0x12345,
		PayloadLength: 100,
		NextHeader:    NextHeaderHopByHopOptions,
		HopLimit:      64,
		SourceAddr:    net.ParseIP("2001:db8::1"),
		DestAddr:      net.ParseIP("2001:db8::2"),
	}

	data, _ := SerializeIPv6Header(original)
	parsed, err := ParseIPv6Header(data)

	assert.NoError(t, err)
	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.TrafficClass, parsed.TrafficClass)
	assert.Equal(t, original.FlowLabel, parsed.FlowLabel)
	assert.Equal(t, original.PayloadLength, parsed.PayloadLength)
	assert.Equal(t, original.NextHeader, parsed.NextHeader)
	assert.Equal(t, original.HopLimit, parsed.HopLimit)
	assert.Equal(t, original.SourceAddr.String(), parsed.SourceAddr.String())
	assert.Equal(t, original.DestAddr.String(), parsed.DestAddr.String())
}

func TestParseIPv6HeaderTooShort(t *testing.T) {
	data := make([]byte, 30)
	_, err := ParseIPv6Header(data)
	assert.Error(t, err)
}

func TestSerializeHopByHopOptions(t *testing.T) {
	header := NewHopByHopOptionsHeader(NextHeaderNone)
	header.Options = append(header.Options, Option{
		OptionType: 0x01,
		OptDataLen: 4,
		Data:       []byte{1, 2, 3, 4},
	})

	data, err := SerializeHopByHopOptions(header)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestParseHopByHopOptions(t *testing.T) {
	header := NewHopByHopOptionsHeader(NextHeaderNone)
	header.Options = append(header.Options, Option{
		OptionType: 0x01,
		OptDataLen: 4,
		Data:       []byte{1, 2, 3, 4},
	})

	data, _ := SerializeHopByHopOptions(header)
	parsed, err := ParseHopByHopOptions(data)

	assert.NoError(t, err)
	assert.Equal(t, header.NextHeader, parsed.NextHeader)
	assert.Len(t, parsed.Options, 1)
	assert.Equal(t, header.Options[0].OptionType, parsed.Options[0].OptionType)
	assert.Equal(t, header.Options[0].OptDataLen, parsed.Options[0].OptDataLen)
	assert.Equal(t, header.Options[0].Data, parsed.Options[0].Data)
}

func TestParseHopByHopOptionsTooShort(t *testing.T) {
	data := make([]byte, 5)
	_, err := ParseHopByHopOptions(data)
	assert.Error(t, err)
}

func TestSerializeV6CoinTransaction(t *testing.T) {
	tx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
		Reserved:   make([]byte, 0),
	}

	data, err := SerializeV6CoinTransaction(tx)
	assert.NoError(t, err)
	assert.Len(t, data, 83)
}

func TestParseV6CoinTransaction(t *testing.T) {
	original := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  1640000000,
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
		Reserved:   []byte{1, 2, 3},
	}

	data, _ := SerializeV6CoinTransaction(original)
	parsed, err := ParseV6CoinTransaction(data)

	assert.NoError(t, err)
	assert.Equal(t, original.Version, parsed.Version)
	assert.Equal(t, original.TxType, parsed.TxType)
	assert.Equal(t, original.Timestamp, parsed.Timestamp)
	assert.Equal(t, original.SenderAddr.String(), parsed.SenderAddr.String())
	assert.Equal(t, original.RecvAddr.String(), parsed.RecvAddr.String())
	assert.Equal(t, original.Amount, parsed.Amount)
	assert.Equal(t, original.Signature, parsed.Signature)
	assert.Equal(t, original.Reserved, parsed.Reserved)
}

func TestParseV6CoinTransactionTooShort(t *testing.T) {
	data := make([]byte, 50)
	_, err := ParseV6CoinTransaction(data)
	assert.Error(t, err)
}

func TestValidateV6CoinTransaction(t *testing.T) {
	validTx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	err := ValidateV6CoinTransaction(validTx)
	assert.NoError(t, err)
}

func TestValidateV6CoinTransactionInvalidVersion(t *testing.T) {
	invalidTx := &V6CoinTransaction{
		Version:    0x0000,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	err := ValidateV6CoinTransaction(invalidTx)
	assert.Error(t, err)
}

func TestValidateV6CoinTransactionInvalidTxType(t *testing.T) {
	invalidTx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0xFF,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	err := ValidateV6CoinTransaction(invalidTx)
	assert.Error(t, err)
}

func TestValidateV6CoinTransactionInvalidAddress(t *testing.T) {
	// Invalid IPv6 address - too short after To16()
	invalidTx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.IPv6loopback,   // Should work
		RecvAddr:   net.IP{0x20, 0x01}, // Truncated IPv6 address
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	validateErr := ValidateV6CoinTransaction(invalidTx)
	assert.Error(t, validateErr)
}

func TestValidateV6CoinTransactionInvalidSignature(t *testing.T) {
	invalidTx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 16),
	}

	err := ValidateV6CoinTransaction(invalidTx)
	assert.Error(t, err)
}

func TestNewV6CoinExtensionHeader(t *testing.T) {
	tx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	extHeader, err := NewV6CoinExtensionHeader(tx)
	assert.NoError(t, err)
	assert.NotNil(t, extHeader)
	assert.Equal(t, uint8(NextHeaderNone), extHeader.HopByHop.NextHeader)
	assert.Len(t, extHeader.HopByHop.Options, 1)
	assert.Equal(t, uint8(V6CoinOptionType), uint8(extHeader.HopByHop.Options[0].OptionType))
}

func TestNewV6CoinExtensionHeaderNilTx(t *testing.T) {
	_, err := NewV6CoinExtensionHeader(nil)
	assert.Error(t, err)
}

func TestParseV6CoinExtensionHeader(t *testing.T) {
	tx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	extHeader, _ := NewV6CoinExtensionHeader(tx)
	data, _ := SerializeHopByHopOptions(&extHeader.HopByHop)

	// Total length should be multiple of 8
	totalLen := len(data)
	fmt.Printf("DEBUG: Extension header length = %d\n", totalLen)
	assert.Equal(t, 0, totalLen%8)
}

func TestIPv6HeaderPadding(t *testing.T) {
	// Test that IPv6 header is exactly 40 bytes
	header := &IPv6Header{
		Version:       6,
		TrafficClass:  0,
		FlowLabel:     0,
		PayloadLength: 100,
		NextHeader:    NextHeaderHopByHopOptions,
		HopLimit:      64,
		SourceAddr:    net.ParseIP("2001:db8::1"),
		DestAddr:      net.ParseIP("2001:db8::2"),
	}

	data, err := SerializeIPv6Header(header)
	assert.NoError(t, err)
	assert.Len(t, data, 40)
}

func TestGetTotalPacketLength(t *testing.T) {
	header := &IPv6Header{
		Version:       6,
		TrafficClass:  0,
		FlowLabel:     0,
		PayloadLength: 100,
		NextHeader:    NextHeaderHopByHopOptions,
		HopLimit:      64,
		SourceAddr:    net.ParseIP("2001:db8::1"),
		DestAddr:      net.ParseIP("2001:db8::2"),
	}

	tx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}

	extHeader, _ := NewV6CoinExtensionHeader(tx)
	payload := []byte("test payload")

	totalLen := GetTotalPacketLength(header, extHeader, payload)
	assert.Equal(t, 140, totalLen) // IPv6(40) + ExtHeader(86) + Payload(14)
}

func BenchmarkSerializeIPv6Header(b *testing.B) {
	header := &IPv6Header{
		Version:       6,
		TrafficClass:  0,
		FlowLabel:     0,
		PayloadLength: 100,
		NextHeader:    NextHeaderHopByHopOptions,
		HopLimit:      64,
		SourceAddr:    net.ParseIP("2001:db8::1"),
		DestAddr:      net.ParseIP("2001:db8::2"),
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = SerializeIPv6Header(header)
	}
}

func BenchmarkSerializeV6CoinTransaction(b *testing.B) {
	tx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = SerializeV6CoinTransaction(tx)
	}
}

func BenchmarkParseIPv6Header(b *testing.B) {
	header := &IPv6Header{
		Version:       6,
		TrafficClass:  0,
		FlowLabel:     0,
		PayloadLength: 100,
		NextHeader:    NextHeaderHopByHopOptions,
		HopLimit:      64,
		SourceAddr:    net.ParseIP("2001:db8::1"),
		DestAddr:      net.ParseIP("2001:db8::2"),
	}
	data, _ := SerializeIPv6Header(header)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseIPv6Header(data)
	}
}

func BenchmarkParseV6CoinTransaction(b *testing.B) {
	tx := &V6CoinTransaction{
		Version:    0x0100,
		TxType:     0x01,
		Timestamp:  uint64(time.Now().Unix()),
		SenderAddr: net.ParseIP("2001:db8::1"),
		RecvAddr:   net.ParseIP("2001:db8::2"),
		Amount:     1000000,
		Signature:  make([]byte, 32),
	}
	data, _ := SerializeV6CoinTransaction(tx)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseV6CoinTransaction(data)
	}
}
