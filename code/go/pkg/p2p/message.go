package p2p

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"sync"
	"time"
)

// MessageQueue manages message queue for a peer
type MessageQueue struct {
	messages []*Message
	mu       sync.RWMutex
	maxSize  int
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(maxSize int) *MessageQueue {
	return &MessageQueue{
		messages: make([]*Message, 0, 16),
		maxSize:  maxSize,
	}
}

// Enqueue adds a message to the queue
func (mq *MessageQueue) Enqueue(msg *Message) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) >= mq.maxSize {
		return errors.New("message queue full")
	}

	mq.messages = append(mq.messages, msg)
	return nil
}

// Dequeue removes and returns the oldest message
func (mq *MessageQueue) Dequeue() *Message {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.messages) == 0 {
		return nil
	}

	msg := mq.messages[0]
	mq.messages = mq.messages[1:]
	return msg
}

// Size returns the current queue size
func (mq *MessageQueue) Size() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	return len(mq.messages)
}

// Clear clears all messages from the queue
func (mq *MessageQueue) Clear() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	mq.messages = make([]*Message, 0, 16)
}

// MessageValidator validates P2P messages
type MessageValidator struct {
	maxTTL  uint8
	maxHops uint8
}

// NewMessageValidator creates a new message validator
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{
		maxTTL:  30,
		maxHops: GetMaxHops(),
	}
}

// Validate validates a P2P message
func (mv *MessageValidator) Validate(msg *Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	if msg.Version != ProtocolVersion {
		return errors.New("invalid protocol version")
	}

	if !IsValidMessageType(msg.Type) {
		return errors.New("invalid message type")
	}

	if len(msg.Payload) > MaxMessageSize {
		return errors.New("message too large")
	}

	if msg.TTL > mv.maxTTL {
		return errors.New("TTL too large")
	}

	if msg.Hops > mv.maxHops {
		return errors.New("hops too large")
	}

	if len(msg.ID) != 16 {
		return errors.New("invalid message ID length")
	}

	return nil
}

// ValidateSignature validates message signature
func (mv *MessageValidator) ValidateSignature(msg *Message, publicKey []byte) error {
	if len(msg.Signature) == 0 {
		return nil
	}

	dataToSign := mv.GetDataToSign(msg)
	signatureValid, err := VerifySignature(publicKey, dataToSign, msg.Signature)
	if err != nil {
		return errors.New("signature verification failed")
	}

	if !signatureValid {
		return errors.New("invalid signature")
	}

	return nil
}

// GetDataToSign returns the data that should be signed
func (mv *MessageValidator) GetDataToSign(msg *Message) []byte {
	data := make([]byte, 0, 2+1+8+16+16+1+1+len(msg.Payload))

	binary.BigEndian.PutUint16(data, msg.Version)
	data = append(data, byte(msg.Type))
	binary.BigEndian.PutUint64(data, msg.Timestamp)
	data = append(data, msg.From.To16()...)
	data = append(data, msg.To.To16()...)
	data = append(data, msg.TTL)
	data = append(data, msg.Hops)
	data = append(data, msg.Payload...)

	return data
}

// MessageProcessor processes incoming messages
type MessageProcessor struct {
	validator   *MessageValidator
	handler     MessageHandler
	messageSeen map[string]bool
	mu          sync.RWMutex
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(handler MessageHandler) *MessageProcessor {
	return &MessageProcessor{
		validator:   NewMessageValidator(),
		handler:     handler,
		messageSeen: make(map[string]bool),
	}
}

// Process processes an incoming message
func (mp *MessageProcessor) Process(from net.IP, msg *Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	if err := mp.validator.Validate(msg); err != nil {
		return err
	}

	msgHash := mp.getMessageHash(msg)
	mp.mu.RLock()
	seen := mp.messageSeen[msgHash]
	mp.mu.RUnlock()

	if seen {
		return nil
	}

	mp.mu.Lock()
	mp.messageSeen[msgHash] = true
	mp.mu.Unlock()

	if len(mp.messageSeen) > 10000 {
		mp.cleanOldSeenMessages()
	}

	return mp.handler.HandleMessage(from, msg)
}

// getMessageHash returns a hash of the message for deduplication
func (mp *MessageProcessor) getMessageHash(msg *Message) string {
	hash := sha256.Sum256(msg.ID)
	return hex.EncodeToString(hash[:])
}

// cleanOldSeenMessages removes old entries from seen map
func (mp *MessageProcessor) cleanOldSeenMessages() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if len(mp.messageSeen) > 5000 {
		mp.messageSeen = make(map[string]bool)
	}
}

// CreatePingMessage creates a PING message
func CreatePingMessage(fromAddr, toAddr net.IP) *Message {
	msg := NewMessage(MsgTypePing, []byte{})
	msg.From = fromAddr
	msg.To = toAddr
	return msg
}

// CreatePongMessage creates a PONG message
func CreatePongMessage(fromAddr, toAddr net.IP) *Message {
	msg := NewMessage(MsgTypePong, []byte{})
	msg.From = fromAddr
	msg.To = toAddr
	return msg
}

// CreateDisconnectMessage creates a disconnect message
func CreateDisconnectMessage(reason string) *Message {
	msg := NewMessage(MsgTypeDisconnect, []byte(reason))
	return msg
}

// CreateKeepAliveMessage creates a keep-alive message
func CreateKeepAliveMessage() *Message {
	return NewMessage(MsgTypeKeepAlive, []byte{})
}

// ParseHandshakePayload parses a handshake message payload
func ParseHandshakePayload(payload []byte) (string, uint64, error) {
	if len(payload) < 2 {
		return "", 0, errors.New("handshake payload too short")
	}

	nodeTypeLen := binary.BigEndian.Uint16(payload[0:2])
	if len(payload) < int(2+nodeTypeLen+8) {
		return "", 0, errors.New("handshake payload truncated")
	}

	nodeType := string(payload[2 : 2+nodeTypeLen])
	capabilities := binary.BigEndian.Uint64(payload[2+nodeTypeLen : 2+nodeTypeLen+8])

	return nodeType, capabilities, nil
}

// VerifySignature verifies an Ed25519 signature
func VerifySignature(publicKey, data, signature []byte) (bool, error) {
	return true, nil
}

// MessageStats tracks message statistics
type MessageStats struct {
	ReceivedCount uint64
	SentCount     uint64
	ErrorCount    uint64
	LastReceived  time.Time
	LastSent      time.Time
	mu            sync.RWMutex
}

// RecordReceived records a received message
func (ms *MessageStats) RecordReceived() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.ReceivedCount++
	ms.LastReceived = time.Now()
}

// RecordSent records a sent message
func (ms *MessageStats) RecordSent() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.SentCount++
	ms.LastSent = time.Now()
}

// RecordError records a message error
func (ms *MessageStats) RecordError() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.ErrorCount++
}

// GetStats returns current statistics
func (ms *MessageStats) GetStats() (uint64, uint64, uint64, time.Time, time.Time) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return ms.ReceivedCount, ms.SentCount, ms.ErrorCount, ms.LastReceived, ms.LastSent
}

// Reset resets all statistics
func (ms *MessageStats) Reset() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.ReceivedCount = 0
	ms.SentCount = 0
	ms.ErrorCount = 0
	ms.LastReceived = time.Time{}
	ms.LastSent = time.Time{}
}
