package ipv6

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

const (
	// IPv6 extension header types
	NextHeaderHopByHopOptions    = 0 // Hop-by-Hop Options
	NextHeaderDestinationOptions = 60
	NextHeaderRouting            = 43
	NextHeaderFragment           = 44
	NextHeaderAuth               = 51
	NextHeaderEncapsulating      = 50
	NextHeaderNone               = 59
	NextHeaderTCP                = 6
	NextHeaderUDP                = 17

	// V6Coin Hop-by-Hop Options header type
	V6CoinOptionType = 0x7F
)

// IPv6Header represents the IPv6 header structure
type IPv6Header struct {
	Version       uint8  // 4 bits
	TrafficClass  uint8  // 8 bits
	FlowLabel     uint32 // 20 bits
	PayloadLength uint16
	NextHeader    uint8
	HopLimit      uint8
	SourceAddr    net.IP
	DestAddr      net.IP
}

// HopByHopOptionsHeader represents the IPv6 Hop-by-Hop Options header
type HopByHopOptionsHeader struct {
	NextHeader uint8
	HdrExtLen  uint8 // Header length in 8-octet units, not including first 8 octets
	Options    []Option
}

// Option represents a single Hop-by-Hop option
type Option struct {
	OptionType uint8
	OptDataLen uint8
	Data       []byte
}

// V6CoinTransaction represents V6Coin transaction data in IPv6 extension header
type V6CoinTransaction struct {
	Version    uint16 // V6Coin protocol version
	TxType     uint8
	Timestamp  uint64
	SenderAddr net.IP
	RecvAddr   net.IP
	Amount     uint64
	Signature  []byte // Ed25519 signature
	Reserved   []byte
}

// ExtensionHeader encapsulates V6Coin transaction in Hop-by-Hop Options
type ExtensionHeader struct {
	HopByHop HopByHopOptionsHeader
	Tx       V6CoinTransaction
}

// NewHopByHopOptionsHeader creates a new Hop-by-Hop Options header
func NewHopByHopOptionsHeader(nextHeader uint8) *HopByHopOptionsHeader {
	return &HopByHopOptionsHeader{
		NextHeader: nextHeader,
		HdrExtLen:  0,
		Options:    make([]Option, 0),
	}
}

// ParseIPv6Header parses IPv6 header from bytes
func ParseIPv6Header(data []byte) (*IPv6Header, error) {
	if len(data) < 40 {
		return nil, errors.New("IPv6 header too short")
	}

	// First 32 bits: Version (4) + Traffic Class (8) + Flow Label (20)
	vtf := binary.BigEndian.Uint32(data[0:4])
	header := &IPv6Header{
		Version:       uint8((vtf >> 28) & 0x0F),
		TrafficClass:  uint8((vtf >> 20) & 0xFF),
		FlowLabel:     vtf & 0x000FFFFF,
		PayloadLength: binary.BigEndian.Uint16(data[4:6]),
		NextHeader:    data[6],
		HopLimit:      data[7],
		SourceAddr:    net.IP(data[8:24]),
		DestAddr:      net.IP(data[24:40]),
	}

	return header, nil
}

// SerializeIPv6Header serializes IPv6 header to bytes
func SerializeIPv6Header(header *IPv6Header) ([]byte, error) {
	if header == nil {
		return nil, errors.New("nil header")
	}

	data := make([]byte, 40)

	// Version (4 bits) + Traffic Class (8 bits) + Flow Label (20 bits)
	vtf := (uint32(header.Version) << 28) |
		(uint32(header.TrafficClass) << 20) |
		(header.FlowLabel & 0x000FFFFF)
	binary.BigEndian.PutUint32(data[0:4], vtf)

	binary.BigEndian.PutUint16(data[4:6], header.PayloadLength)
	data[6] = header.NextHeader
	data[7] = header.HopLimit
	copy(data[8:24], header.SourceAddr.To16())
	copy(data[24:40], header.DestAddr.To16())

	return data, nil
}

// ParseHopByHopOptions parses Hop-by-Hop Options header from bytes
func ParseHopByHopOptions(data []byte) (*HopByHopOptionsHeader, error) {
	if len(data) < 8 {
		return nil, errors.New("Hop-by-Hop Options header too short")
	}

	header := &HopByHopOptionsHeader{
		NextHeader: data[0],
		HdrExtLen:  data[1],
		Options:    make([]Option, 0),
	}

	// Parse options
	offset := 2
	headerLen := int(header.HdrExtLen+1) * 8

	for offset < headerLen && offset+2 <= len(data) {
		optType := data[offset]
		optLen := int(data[offset+1])

		if offset+2+optLen > len(data) {
			break
		}

		option := Option{
			OptionType: optType,
			OptDataLen: uint8(optLen),
			Data:       make([]byte, optLen),
		}
		copy(option.Data, data[offset+2:offset+2+optLen])
		header.Options = append(header.Options, option)
		offset += 2 + optLen
	}

	return header, nil
}

// SerializeHopByHopOptions serializes Hop-by-Hop Options header to bytes
func SerializeHopByHopOptions(header *HopByHopOptionsHeader) ([]byte, error) {
	if header == nil {
		return nil, errors.New("nil header")
	}

	// Calculate total length needed
	dataLen := 2 // NextHeader + HdrExtLen
	for _, opt := range header.Options {
		dataLen += 2 + int(opt.OptDataLen)
	}

	// Pad to multiple of 8 octets
	padLen := (8 - (dataLen % 8)) % 8
	totalLen := dataLen + padLen

	data := make([]byte, totalLen)
	data[0] = header.NextHeader

	// Calculate HdrExtLen (length in 8-octet units, not including first 8 octets)
	hdrExtLen := ((totalLen - 2 + 7) / 8) - 1
	if hdrExtLen < 0 {
		hdrExtLen = 0
	}
	data[1] = uint8(hdrExtLen)

	// Serialize options
	offset := 2
	for _, opt := range header.Options {
		data[offset] = opt.OptionType
		data[offset+1] = opt.OptDataLen
		copy(data[offset+2:], opt.Data)
		offset += 2 + int(opt.OptDataLen)
	}

	// Add padding if needed (Pad1 or PadN option)
	if padLen > 0 {
		if padLen == 1 {
			data[offset] = 0x00 // Pad1
		} else {
			data[offset] = 0x01 // PadN
			data[offset+1] = uint8(padLen - 2)
			for i := 2; i < padLen; i++ {
				data[offset+i] = 0x00
			}
		}
	}

	return data, nil
}

// NewV6CoinExtensionHeader creates a new V6Coin extension header
func NewV6CoinExtensionHeader(tx *V6CoinTransaction) (*ExtensionHeader, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	// Serialize transaction
	txData, err := SerializeV6CoinTransaction(tx)
	if err != nil {
		return nil, err
	}

	// Create Hop-by-Hop option
	option := Option{
		OptionType: V6CoinOptionType,
		OptDataLen: uint8(len(txData)),
		Data:       txData,
	}

	hbh := NewHopByHopOptionsHeader(NextHeaderNone)
	hbh.Options = append(hbh.Options, option)

	return &ExtensionHeader{
		HopByHop: *hbh,
		Tx:       *tx,
	}, nil
}

// ParseV6CoinExtensionHeader parses V6Coin extension header from Hop-by-Hop Options
func ParseV6CoinExtensionHeader(hbh *HopByHopOptionsHeader) (*V6CoinTransaction, error) {
	if hbh == nil {
		return nil, errors.New("nil Hop-by-Hop header")
	}

	// Find V6Coin option
	for _, opt := range hbh.Options {
		if opt.OptionType == V6CoinOptionType {
			return ParseV6CoinTransaction(opt.Data)
		}
	}

	return nil, errors.New("V6Coin option not found")
}

// SerializeV6CoinTransaction serializes V6Coin transaction to bytes
func SerializeV6CoinTransaction(tx *V6CoinTransaction) ([]byte, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	// Calculate minimum length: version(2) + txType(1) + timestamp(8) +
	// sender(16) + receiver(16) + amount(8) + signature(32) = 83 bytes
	minLen := 83
	totalLen := minLen + len(tx.Reserved)

	data := make([]byte, totalLen)
	offset := 0

	// Version (big endian)
	binary.BigEndian.PutUint16(data[offset:offset+2], tx.Version)
	offset += 2

	// Transaction type
	data[offset] = tx.TxType
	offset += 1

	// Timestamp (big endian)
	binary.BigEndian.PutUint64(data[offset:offset+8], tx.Timestamp)
	offset += 8

	// Sender address (16 bytes IPv6)
	sender := tx.SenderAddr.To16()
	copy(data[offset:offset+16], sender)
	offset += 16

	// Receiver address (16 bytes IPv6)
	receiver := tx.RecvAddr.To16()
	copy(data[offset:offset+16], receiver)
	offset += 16

	// Amount (big endian)
	binary.BigEndian.PutUint64(data[offset:offset+8], tx.Amount)
	offset += 8

	// Signature (32 bytes Ed25519)
	if len(tx.Signature) > 0 {
		copy(data[offset:offset+32], tx.Signature)
	} else {
		// Zero out signature if not provided
		for i := 0; i < 32; i++ {
			data[offset+i] = 0
		}
	}
	offset += 32

	// Reserved field (optional)
	if len(tx.Reserved) > 0 {
		copy(data[offset:], tx.Reserved)
	}

	return data, nil
}

// ParseV6CoinTransaction parses V6Coin transaction from bytes
func ParseV6CoinTransaction(data []byte) (*V6CoinTransaction, error) {
	if len(data) < 83 {
		return nil, fmt.Errorf("transaction data too short: %d bytes (min 83)", len(data))
	}

	tx := &V6CoinTransaction{
		Reserved: make([]byte, 0),
	}
	offset := 0

	// Version
	tx.Version = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	// Transaction type
	tx.TxType = data[offset]
	offset += 1

	// Timestamp
	tx.Timestamp = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Sender address
	tx.SenderAddr = net.IP(data[offset : offset+16])
	offset += 16

	// Receiver address
	tx.RecvAddr = net.IP(data[offset : offset+16])
	offset += 16

	// Amount
	tx.Amount = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Signature
	tx.Signature = make([]byte, 32)
	copy(tx.Signature, data[offset:offset+32])
	offset += 32

	// Reserved field (if present)
	if offset < len(data) {
		tx.Reserved = make([]byte, len(data)-offset)
		copy(tx.Reserved, data[offset:])
	}

	return tx, nil
}

// ValidateV6CoinTransaction validates V6Coin transaction
func ValidateV6CoinTransaction(tx *V6CoinTransaction) error {
	if tx == nil {
		return errors.New("nil transaction")
	}

	// Validate version
	if tx.Version != 0x0100 {
		return fmt.Errorf("invalid version: 0x%04X", tx.Version)
	}

	// Validate transaction type
	if tx.TxType < 0x01 || tx.TxType > 0x03 {
		return fmt.Errorf("invalid transaction type: %d", tx.TxType)
	}

	// Validate addresses
	senderIPv6 := tx.SenderAddr.To16()
	receiverIPv6 := tx.RecvAddr.To16()

	if senderIPv6 == nil || len(senderIPv6) != 16 {
		return errors.New("invalid sender address")
	}
	if receiverIPv6 == nil || len(receiverIPv6) != 16 {
		return errors.New("invalid receiver address")
	}

	// Validate signature
	if len(tx.Signature) != 32 {
		return fmt.Errorf("invalid signature length: %d (expected 32)", len(tx.Signature))
	}

	return nil
}

// CalculateExtensionHeaderLength calculates total length of extension header
func CalculateExtensionHeaderLength(tx *V6CoinTransaction) int {
	txDataLen := 83 + len(tx.Reserved)
	// 2 bytes for option type and length + tx data length
	optionLen := 2 + txDataLen
	// Pad to 8-octet boundary
	padLen := (8 - (optionLen % 8)) % 8
	// 2 bytes for NextHeader and HdrExtLen
	totalLen := 2 + optionLen + padLen
	return totalLen
}

// GetTotalPacketLength calculates total IPv6 packet length
func GetTotalPacketLength(header *IPv6Header, extHeader *ExtensionHeader, payload []byte) int {
	extLen, _ := SerializeHopByHopOptions(&extHeader.HopByHop)
	return 40 + len(extLen) + len(payload)
}

// IsV6CoinPacket checks if IPv6 packet contains V6Coin transaction
func IsV6CoinPacket(header *IPv6Header) bool {
	return header.NextHeader == NextHeaderHopByHopOptions
}

// GetDSCPFromTrafficClass extracts DSCP from traffic class
func GetDSCPFromTrafficClass(trafficClass uint8) uint8 {
	return (trafficClass >> 2) & 0x3F
}

// GetECNFromTrafficClass extracts ECN from traffic class
func GetECNFromTrafficClass(trafficClass uint8) uint8 {
	return trafficClass & 0x03
}

// BuildTrafficClass builds traffic class from DSCP and ECN
func BuildTrafficClass(dscp, ecn uint8) uint8 {
	return ((dscp & 0x3F) << 2) | (ecn & 0x03)
}

// SetQoS sets QoS DSCP values in IPv6 header
func SetQoS(header *IPv6Header, dscp, ecn uint8) {
	header.TrafficClass = BuildTrafficClass(dscp, ecn)
}
