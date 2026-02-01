package p2p

import (
	"encoding/binary"
	"errors"
	"net"
	"time"
)

// GetMaxHops returns maximum allowed hops
func GetMaxHops() uint8 {
	return 64
}

// IsExpiredMessage checks if a message has expired
func IsExpiredMessage(msg *Message) bool {
	return time.Since(time.Unix(0, int64(msg.Timestamp))) > time.Duration(msg.TTL)*time.Second
}

// NewNetworkMetrics creates zero metrics
func NewNetworkMetrics() *NetworkMetrics {
	return &NetworkMetrics{
		TotalPeers:        0,
		ConnectedPeers:    0,
		ActivePeers:       0,
		MessagesPerSecond: 0,
		BandwidthUsage:    0,
		AverageLatency:    0,
		PacketLossRate:    0,
	}
}

// MessageToBytes converts message to bytes for sending
func MessageToBytes(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message cannot be nil")
	}

	// Calculate total message length
	totalLen := 2 + 1 + 16 + 16 + 8 + 16 + 1 + 1 + len(msg.Payload)
	buf := make([]byte, totalLen)

	offset := 0

	// Version (2 bytes)
	binary.BigEndian.PutUint16(buf[offset:offset+2], msg.Version)
	offset += 2

	// Type (1 byte)
	buf[offset] = byte(msg.Type)
	offset += 1

	// From (16 bytes)
	copy(buf[offset:offset+16], msg.From.To16())
	offset += 16

	// To (16 bytes)
	copy(buf[offset:offset+16], msg.To.To16())
	offset += 16

	// Timestamp (8 bytes)
	binary.BigEndian.PutUint64(buf[offset:offset+8], msg.Timestamp)
	offset += 8

	// ID (16 bytes)
	copy(buf[offset:offset+16], msg.ID)
	offset += 16

	// TTL (1 byte)
	buf[offset] = msg.TTL
	offset += 1

	// Hops (1 byte)
	buf[offset] = msg.Hops
	offset += 1

	// Payload (variable length)
	copy(buf[offset:], msg.Payload)

	return buf, nil
}

// BytesToMessage converts bytes to Message
func BytesToMessage(data []byte) (*Message, error) {
	if len(data) < 62 {
		return nil, errors.New("message data too short (min 62 bytes)")
	}

	msg := &Message{
		Version: binary.BigEndian.Uint16(data[0:2]),
	}

	offset := 2

	// Type (1 byte)
	msg.Type = MessageType(data[offset])
	offset += 1

	// From (16 bytes)
	msg.From = net.IP(data[offset : offset+16])
	offset += 16

	// To (16 bytes)
	msg.To = net.IP(data[offset : offset+16])
	offset += 16

	// Timestamp (8 bytes)
	msg.Timestamp = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// ID (16 bytes)
	msg.ID = make([]byte, 16)
	copy(msg.ID, data[offset:offset+16])
	offset += 16

	// TTL (1 byte)
	if offset < len(data) {
		msg.TTL = data[offset]
		offset += 1
	}

	// Hops (1 byte)
	if offset < len(data) {
		msg.Hops = data[offset]
		offset += 1
	}

	// Payload (remaining bytes)
	if offset < len(data) {
		msg.Payload = make([]byte, len(data)-offset)
		copy(msg.Payload, data[offset:])
	}

	return msg, nil
}
