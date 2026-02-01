package p2p

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

const (
	// P2P protocol constants
	DefaultPort       = 38901
	MaxMessageSize    = 1024 * 1024 // 1MB
	HandshakeTimeout  = 30 * time.Second
	KeepAliveInterval = 60 * time.Second
	MaxPeers          = 1000
	DiscoveryInterval = 30 * time.Second
	MessageTTL        = 4 * time.Minute

	// Message types
	MsgTypeHandshake  = 0x01
	MsgTypePing       = 0x02
	MsgTypePong       = 0x03
	MsgTypePeerList   = 0x04
	MsgTypeBroadcast  = 0x05
	MsgTypeDirect     = 0x06
	MsgTypeV6CoinTx   = 0x07
	MsgTypeDisconnect = 0x08
	MsgTypeKeepAlive  = 0x09

	// Peer states
	PeerStateNew        = "new"
	PeerStateConnecting = "connecting"
	PeerStateConnected  = "connected"
	PeerStateIdle       = "idle"
	PeerStateBusy       = "busy"
	PeerStateOffline    = "offline"

	// Connection types
	ConnTypeOutbound = "outbound"
	ConnTypeInbound  = "inbound"

	// Routing metrics
	DefaultMetricWeight = 10
	MaxMetricWeight     = 100

	// Anti-DDoS constants
	MaxMessagesPerSecond = 100
	MaxConnectionsPerIP  = 10
	MaxPendingMessages   = 1000

	MinReputationScore = 0
	MaxReputationScore = 1000
	SpamThresholdScore = 500

	// Protocol version
	ProtocolVersion uint16 = 0x0100
)

// MessageType represents P2P message type
type MessageType uint8

// PeerState represents the state of a peer connection
type PeerState string

// ConnectionType represents the type of connection
type ConnectionType string

// Peer represents a peer in the network
type Peer struct {
	ID              PeerID
	Address         net.IP
	Port            uint16
	LastSeen        time.Time
	State           PeerState
	ConnectionType  ConnectionType
	MessageQueue    []Message
	ReputationScore int
	BytesSent       uint64
	BytesReceived   uint64
	MessageCount    uint32
	PingTime        time.Time
	mu              sync.RWMutex
}

// PeerID is the unique identifier for a peer
type PeerID string

// Message represents a P2P message
type Message struct {
	Version   uint16
	Type      MessageType
	From      net.IP
	To        net.IP
	Payload   []byte
	Timestamp uint64
	Signature []byte // Ed25519 signature
	TTL       uint8  // Time to live for message forwarding
	Hops      uint8  // Number of hops
	ID        []byte // Unique message ID
}

// PeerListMessage represents a list of known peers
type PeerListMessage struct {
	Version   uint16
	Type      MessageType
	Peers     []PeerInfo
	Signature []byte
}

// PeerInfo represents peer information for peer list
type PeerInfo struct {
	Address    net.IP
	Port       uint16
	LastSeen   time.Time
	Reputation int
}

// HandshakeMessage represents the handshake message
type HandshakeMessage struct {
	Version      uint16
	NodeType     string // "full", "light", "pruning"
	Capabilities uint64
	Signature    []byte
}

// V6CoinTxMessage represents a V6Coin transaction embedded in P2P message
type V6CoinTxMessage struct {
	TxData    []byte // Serialized V6Coin transaction
	Timestamp uint64
	Signature []byte
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	SeedPeers        []string // List of bootstrap seed peers
	MaxPeers         int
	DiscoveryEnabled bool
	EnableUPnP       bool
	BindAddress      net.IP
	BindPort         uint16
}

// RoutingTable represents the routing table for DHT
type RoutingTable struct {
	mu      sync.RWMutex
	peers   map[PeerID]*Peer
	routes  map[PeerID][]*PeerID // DHT-like routing
	buckets [256]*Peer           // Kademlia buckets (8-bit prefixes)
}

// ConnectionManager manages peer connections
type ConnectionManager struct {
	listener            net.Listener
	connections         map[string]*net.Conn
	connectionsMu       sync.RWMutex
	peers               map[string]*Peer
	incomingRateLimiter *RateLimiter
	outgoingRateLimiter *RateLimiter
	maxConnections      int
}

// RateLimiter implements rate limiting
type RateLimiter struct {
	maxMessagesPerSecond int
	messages             []time.Time
	mu                   sync.Mutex
}

// DHTOptions for DHT operations
type DHTOptions struct {
	Alpha   int
	K       int
	Timeout time.Duration
}

// FindPeerResult represents the result of a peer search
type FindPeerResult struct {
	Peers   []*Peer
	Closest *Peer
}

// NetworkMetrics represents network statistics
type NetworkMetrics struct {
	TotalPeers        int
	ConnectedPeers    int
	ActivePeers       int
	MessagesPerSecond float64
	BandwidthUsage    float64
	AverageLatency    time.Duration
	PacketLossRate    float64
}

// BootstrapConfig for network bootstrapping
type BootstrapConfig struct {
	SeedPeers          []string
	BootstrapInterval  time.Duration
	MaxBootstrapPeers  int
	InitialPingTimeout time.Duration
}

// MessageHandler handles different message types
type MessageHandler interface {
	HandleMessage(from net.IP, msg *Message) error
}

// V6CoinTxHandler handles V6Coin transaction messages
type V6CoinTxHandler interface {
	HandleV6CoinTransaction(from net.IP, txData []byte, signature []byte) error
}

// RoutingStrategy defines routing algorithm
type RoutingStrategy interface {
	FindClosestPeers(target []byte, count int) []*Peer
	RouteMessage(msg *Message, ttl uint8) (*Peer, error)
	IsExpired(msg *Message) bool
}

// NetworkEvent represents network events
type NetworkEvent int

const (
	EventPeerDiscovered NetworkEvent = iota
	EventPeerConnected
	EventPeerDisconnected
	EventMessageReceived
	EventMessageBroadcasted
	EventRoutingTableUpdate
	DDoSDetected
)

// EventListener is called on network events
type EventListener func(event NetworkEvent, data interface{})

// MessageID generates unique message ID
func MessageID() []byte {
	id := make([]byte, 16)
	binary.BigEndian.PutUint64(id[0:8], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(id[8:16], uint64(time.Now().UnixNano()))
	return id
}

// IsValidMessageType checks if a message type is valid
func IsValidMessageType(msgType MessageType) bool {
	return msgType >= MsgTypeHandshake && msgType <= MsgTypeKeepAlive
}

// CalculateDistance calculates distance metric between two peers
func CalculateDistance(p1, p2 *Peer) int64 {
	if p1 == nil || p2 == nil {
		return 0
	}

	// Calculate distance using latency and reputation
	latencyDist := int64(p2.PingTime.Sub(p1.PingTime) / time.Millisecond)
	if latencyDist < 0 {
		latencyDist = -latencyDist
	}

	distance := latencyDist/10 + int64(p2.ReputationScore-p1.ReputationScore)/1000
	return distance
}

// IsPeerHealthy checks if a peer is healthy for routing
func IsPeerHealthy(peer *Peer) bool {
	if peer == nil {
		return false
	}

	// Check if peer is in connecting state
	if peer.State != PeerStateConnected && peer.State != PeerStateIdle {
		return false
	}

	// Check if peer has been seen recently
	return time.Since(peer.LastSeen) < MaxPeerAge
}

// MaxPeerAge is the maximum time before a peer is considered stale
const MaxPeerAge = 24 * time.Hour

// NewPeer creates a new peer instance
func NewPeer(id PeerID, address net.IP, port uint16) *Peer {
	return &Peer{
		ID:              id,
		Address:         address,
		Port:            port,
		LastSeen:        time.Now(),
		State:           PeerStateNew,
		ConnectionType:  ConnTypeOutbound,
		MessageQueue:    make([]Message, 0, 16),
		ReputationScore: MinReputationScore,
		mu:              sync.RWMutex{},
	}
}

// NewMessage creates a new P2P message
func NewMessage(msgType MessageType, payload []byte) *Message {
	return &Message{
		Version:   ProtocolVersion,
		Type:      msgType,
		Payload:   payload,
		Timestamp: uint64(time.Now().Unix()),
		ID:        MessageID(),
		TTL:       uint8(MessageTTL / time.Second),
		Hops:      0,
	}
}

// NewSignedMessage creates a signed message
func NewSignedMessage(msgType MessageType, fromAddr, toAddr net.IP, payload []byte, privateKey []byte) (*Message, error) {
	msg := NewMessage(msgType, payload)

	// Set source and destination
	msg.From = fromAddr
	msg.To = toAddr

	// Sign the message (sign: timestamp + fromAddr + toAddr + payload + version)
	dataToSign := make([]byte, 0, 2+16+16+8+len(payload)+2+64+8)
	offset := 0

	binary.BigEndian.PutUint16(dataToSign[offset:offset+2], msg.Version)
	offset += 2
	dataToSign[offset] = byte(msg.Type)
	offset += 1
	binary.BigEndian.PutUint64(dataToSign[offset:offset+8], msg.Timestamp)
	offset += 8
	from := fromAddr.To16()
	copy(dataToSign[offset:offset+16], from)
	offset += 16
	to := toAddr.To16()
	copy(dataToSign[offset:offset+16], to)
	offset += 16
	binary.BigEndian.PutUint16(dataToSign[offset:offset+2], uint16(len(payload)))
	offset += 2
	dataToSign[offset] = msg.TTL
	offset += 1
	dataToSign[offset] = msg.Hops
	offset += 1
	copy(dataToSign[offset:], payload)
	signature, err := crypto.Sign(privateKey, dataToSign[offset:])

	if err != nil {
		return nil, err
	}

	msg.Signature = signature
	return msg, nil
}

// NewPeerListMessage creates a peer list message
func NewPeerListMessage(peers []*Peer, privateKey []byte) (*PeerListMessage, error) {
	peerInfos := make([]PeerInfo, len(peers))

	for i, peer := range peers {
		if peer == nil {
			continue
		}

		peerInfos[i] = PeerInfo{
			Address:    peer.Address,
			Port:       peer.Port,
			LastSeen:   peer.LastSeen,
			Reputation: peer.ReputationScore,
		}
	}

	payload, err := SerializePeerList(peerInfos)
	if err != nil {
		return nil, err
	}

	msg := NewMessage(MsgTypePeerList, payload)

	// Sign the message
	signature, err := crypto.Sign(privateKey, append([]byte{byte(msg.Version >> 8), byte(msg.Version)}, msg.ID...))
	if err != nil {
		return nil, err
	}

	return &PeerListMessage{
		Version:   ProtocolVersion,
		Type:      MsgTypePeerList,
		Peers:     peerInfos,
		Signature: signature,
	}, nil
}

// SerializePeerList serializes peer list
func SerializePeerList(peers []PeerInfo) ([]byte, error) {
	// Format: count(2) + [address(16) + port(2) + reputation(4) + lastSeen(8)]
	totalLen := 2 + len(peers)*(16+2+4+8)
	buf := make([]byte, 0, totalLen)

	// Write count
	countBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(countBytes, uint16(len(peers)))
	buf = append(buf, countBytes...)

	for _, peer := range peers {
		// Address (16 bytes)
		buf = append(buf, peer.Address.To16()...)

		// Port (2 bytes)
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, peer.Port)
		buf = append(buf, portBytes...)

		// Reputation (4 bytes)
		reputationBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(reputationBytes, uint32(peer.Reputation))
		buf = append(buf, reputationBytes...)

		// LastSeen (8 bytes)
		lastSeenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lastSeenBytes, uint64(peer.LastSeen.UnixNano()))
		buf = append(buf, lastSeenBytes...)
	}

	return buf, nil
}

// DeserializePeerList deserializes peer list from bytes
func DeserializePeerList(data []byte) ([]PeerInfo, error) {
	if len(data) < 6 {
		return nil, errors.New("peer list data too short")
	}

	count := binary.BigEndian.Uint16(data[0:2])
	if int(count)*30 > len(data) {
		return nil, fmt.Errorf("peer list data truncated (count=%d, len=%d)", count, len(data))
	}

	peers := make([]PeerInfo, 0, count)
	offset := 2

	for i := 0; i < int(count); i++ {
		if offset+30 > len(data) {
			return nil, fmt.Errorf("peer list data truncated at index %d", i)
		}

		peer := PeerInfo{
			Address:    make(net.IP, 16),
			Port:       binary.BigEndian.Uint16(data[offset+16 : offset+18]),
			Reputation: int(binary.BigEndian.Uint32(data[offset+18 : offset+22])),
			LastSeen:   time.Unix(0, int64(binary.BigEndian.Uint64(data[offset+22:offset+30]))),
		}
		copy(peer.Address, data[offset:offset+16])
		offset += 30

		peers = append(peers, peer)
	}

	return peers, nil
}

// NewV6CoinTxMessage creates a V6Coin transaction message
func NewV6CoinTxMessage(txData []byte, fromAddr, toAddr net.IP) (*V6CoinTxMessage, error) {
	return &V6CoinTxMessage{
		TxData:    txData,
		Timestamp: uint64(time.Now().Unix()),
		Signature: []byte{}, // Will be signed by caller
	}, nil
}

// SignV6CoinTx signs a V6Coin transaction message
func SignV6CoinTx(msg *V6CoinTxMessage, privateKey []byte) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	// Create payload: version + timestamp + txData
	payload := make([]byte, 0, 2+8+8+len(msg.TxData))
	offset := 0

	binary.BigEndian.PutUint16(payload[offset:offset+2], ProtocolVersion)
	offset += 2
	binary.BigEndian.PutUint64(payload[offset:offset+8], msg.Timestamp)
	offset += 8
	copy(payload[offset:], msg.TxData)

	// Sign the message
	signature, err := crypto.Sign(privateKey, payload)
	if err != nil {
		return fmt.Errorf("failed to sign V6Coin message: %w", err)
	}

	msg.Signature = signature
	return nil
}

// IsValidMessage validates a P2P message
func IsValidMessage(msg *Message) error {
	if msg == nil {
		return errors.New("message cannot be nil")
	}

	// Check version
	if msg.Version != ProtocolVersion {
		return fmt.Errorf("invalid protocol version: 0x%04X", msg.Version)
	}

	// Check message type
	if !IsValidMessageType(msg.Type) {
		return fmt.Errorf("invalid message type: %d", msg.Type)
	}

	// Check message size
	if len(msg.Payload) > MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes (max %d bytes)", len(msg.Payload), MaxMessageSize)
	}

	// Check TTL
	if msg.TTL > uint8(MessageTTL/time.Second) {
		return fmt.Errorf("TTL too large: %d (max %d)", msg.TTL, uint8(MessageTTL/time.Second))
	}

	// Check hops
	if msg.Hops > 64 {
		return fmt.Errorf("hops too large: %d (max 64)", msg.Hops)
	}

	// Check message ID
	if len(msg.ID) != 16 {
		return errors.New("invalid message ID length")
	}

	// Check signature if present
	if msg.Type != MsgTypePing && msg.Type != MsgTypePong &&
		msg.Type != MsgTypeKeepAlive && len(msg.Signature) != 64 {
		return errors.New("signature must be present for non-handshake/ping messages")
	}

	return nil
}

// CreateHandshakeMessage creates a handshake message
func CreateHandshakeMessage(nodeType string, capabilities uint64) (*HandshakeMessage, error) {
	return &HandshakeMessage{
		Version:      ProtocolVersion,
		NodeType:     nodeType,
		Capabilities: capabilities,
		Signature:    []byte{}, // Will be signed by caller
	}, nil
}

// SignHandshake signs a handshake message
func SignHandshake(msg *HandshakeMessage, privateKey []byte) error {
	if msg == nil {
		return errors.New("handshake message cannot be nil")
	}

	// Create payload: version + nodeType + capabilities
	payload := make([]byte, 0, 2+2+8)
	offset := 0

	binary.BigEndian.PutUint16(payload[offset:offset+2], msg.Version)
	offset += 2
	payload[offset] = byte(len(msg.NodeType))
	offset += 1

	binary.BigEndian.PutUint64(payload[offset:offset+8], msg.Capabilities)
	offset += 8

	signature, err := crypto.Sign(privateKey, payload)
	if err != nil {
		return fmt.Errorf("failed to sign handshake: %w", err)
	}

	msg.Signature = signature
	return nil
}

// NewRoutingTable creates a new routing table
func NewRoutingTable() *RoutingTable {
	buckets := [256]*Peer{}
	for i := 0; i < 256; i++ {
		buckets[i] = nil
	}

	return &RoutingTable{
		peers:   make(map[PeerID]*Peer),
		routes:  make(map[PeerID][]*PeerID),
		buckets: buckets,
		mu:      sync.RWMutex{},
	}
}

// GetBucket calculates the bucket index for a peer ID
func (rt *RoutingTable) GetBucket(id PeerID) int {
	// XOR first 8 bytes of peer ID to get bucket (0-255)
	hash := sha256.Sum256([]byte(id))
	return int(hash[0]) & 0xFF
}

// AddPeer adds a peer to routing table
func (rt *RoutingTable) AddPeer(peer *Peer) error {
	if peer == nil {
		return errors.New("peer cannot be nil")
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if len(rt.peers) >= MaxPeers {
		return fmt.Errorf("routing table full (max %d peers)", MaxPeers)
	}

	_, exists := rt.peers[peer.ID]
	if !exists {
		bucket := rt.GetBucket(peer.ID)
		rt.buckets[bucket] = peer
	}

	rt.peers[peer.ID] = peer
	return nil
}

// RemovePeer removes a peer from routing table
func (rt *RoutingTable) RemovePeer(id PeerID) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	delete(rt.peers, id)

	// Remove from bucket
	bucket := rt.GetBucket(id)
	if rt.buckets[bucket] != nil && rt.buckets[bucket].ID == id {
		rt.buckets[bucket] = nil
	}

	return nil
}

// FindClosestPeers finds N closest peers to a target
func (rt *RoutingTable) FindClosestPeers(target []byte, count int) []*Peer {
	if target == nil || len(target) < 16 {
		return nil
	}

	rt.mu.RLock()
	defer rt.mu.RUnlock()

	// Collect all peers and calculate distances
	type peerDist struct {
		peer     *Peer
		distance int
	}

	peerDists := make([]peerDist, 0, len(rt.peers))
	for _, peer := range rt.peers {
		if peer == nil {
			continue
		}

		// Calculate XOR distance
		distance := 0
		peerAddr := peer.Address.To16()
		for i := 0; i < 16 && i < len(target); i++ {
			distance += int(peerAddr[i] ^ target[i])
		}

		peerDists = append(peerDists, peerDist{peer: peer, distance: distance})
	}

	// Sort by distance (simple bubble sort)
	for i := 0; i < len(peerDists)-1; i++ {
		for j := i + 1; j < len(peerDists); j++ {
			if peerDists[i].distance > peerDists[j].distance {
				peerDists[i], peerDists[j] = peerDists[j], peerDists[i]
			}
		}
	}

	// Return top N peers
	result := make([]*Peer, 0, count)
	for i := 0; i < len(peerDists) && i < count; i++ {
		result = append(result, peerDists[i].peer)
	}

	return result
}

// GetPeer retrieves a peer by ID
func (rt *RoutingTable) GetPeer(id PeerID) *Peer {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return rt.peers[id]
}

// GetAllPeers returns all known peers
func (rt *RoutingTable) GetAllPeers() []*Peer {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	peers := make([]*Peer, 0, len(rt.peers))
	for _, peer := range rt.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetPeerCount returns the number of known peers
func (rt *RoutingTable) GetPeerCount() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return len(rt.peers)
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(maxConnections int) *ConnectionManager {
	return &ConnectionManager{
		connections:         make(map[string]*net.Conn),
		connectionsMu:       sync.RWMutex{},
		peers:               make(map[string]*Peer),
		incomingRateLimiter: NewRateLimiter(MaxMessagesPerSecond),
		outgoingRateLimiter: NewRateLimiter(MaxMessagesPerSecond),
		maxConnections:      maxConnections,
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxMessagesPerSecond int) *RateLimiter {
	return &RateLimiter{
		maxMessagesPerSecond: maxMessagesPerSecond,
		messages:             make([]time.Time, 0, maxMessagesPerSecond*2),
		mu:                   sync.Mutex{},
	}
}

// Allow checks if a message is allowed based on rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Remove old messages outside the time window
	windowStart := now.Add(-time.Second * 1)
	newMessages := rl.messages[:0]

	for _, msgTime := range rl.messages {
		if msgTime.After(windowStart) {
			newMessages = append(newMessages, msgTime)
		}
	}

	rl.messages = newMessages

	// Check if we're under the limit
	return len(rl.messages) < rl.maxMessagesPerSecond
}

// Record records a message timestamp
func (rl *RateLimiter) Record() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.messages = append(rl.messages, time.Now())
}

// GetMetrics calculates network metrics
func (cm *ConnectionManager) GetMetrics() *NetworkMetrics {
	cm.connectionsMu.RLock()
	defer cm.connectionsMu.RUnlock()

	peers := make([]*Peer, 0, len(cm.peers))
	for _, peer := range cm.peers {
		peers = append(peers, peer)
	}

	if len(peers) == 0 {
		return &NetworkMetrics{}
	}

	return &NetworkMetrics{
		TotalPeers:        len(peers),
		ConnectedPeers:    len(peers),
		MessagesPerSecond: 0, // TODO: Calculate from message stats
		BandwidthUsage:    0, // TODO: Calculate from network stats
		AverageLatency:    0, // TODO: Calculate from ping times
		PacketLossRate:    0, // TODO: Calculate from message delivery
	}
}

// UpdatePeerStats updates peer statistics
func (cm *ConnectionManager) UpdatePeerStats(peerID string, bytesSent, bytesReceived, messageCount uint32) {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()

	peer := cm.peers[peerID]
	if peer == nil {
		return
	}

	peer.BytesSent += uint64(bytesSent)
	peer.BytesReceived += uint64(bytesReceived)
	peer.MessageCount += messageCount
}

// DisconnectPeer disconnects a peer
func (cm *ConnectionManager) DisconnectPeer(peerID string) error {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()

	if conn, exists := cm.connections[peerID]; exists {
		(*conn).Close()
		delete(cm.connections, peerID)
	}

	delete(cm.peers, peerID)

	return nil
}

// GetConnectedPeers returns list of connected peers
func (cm *ConnectionManager) GetConnectedPeers() []PeerID {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()

	connected := make([]PeerID, 0, len(cm.peers))
	for id := range cm.peers {
		if _, exists := cm.connections[id]; exists {
			connected = append(connected, PeerID(id))
		}
	}

	return connected
}

// BanPeer bans a peer
func (cm *ConnectionManager) BanPeer(peerID string, reason string, score int) error {
	cm.connectionsMu.Lock()
	defer cm.connectionsMu.Unlock()

	peer := cm.peers[peerID]
	if peer == nil {
		return errors.New("peer not found")
	}

	peer.ReputationScore = -score

	return nil
}

// CreateBootstrapConfig creates default bootstrap configuration
func CreateBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		SeedPeers:          []string{},
		BootstrapInterval:  DiscoveryInterval,
		MaxBootstrapPeers:  MaxPeers / 2,
		InitialPingTimeout: 5 * time.Minute,
	}
}

// NewNetworkConfig creates default network configuration
func NewNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		MaxPeers:         MaxPeers,
		DiscoveryEnabled: true,
		EnableUPnP:       true,
		BindAddress:      nil,
		BindPort:         DefaultPort,
	}
}

// IsValidPeerID validates a peer ID format
func IsValidPeerID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}

	return true
}

// AllowedPeers calculates how many new peers we can accept
func AllowedPeers(current int) int {
	if current >= MaxPeers {
		return 0
	}
	return MaxPeers - current
}

// IsSpammer checks if a peer is a spammer
func IsSpammer(peer *Peer) bool {
	if peer == nil {
		return false
	}
	return peer.ReputationScore < SpamThresholdScore
}

// UpdateReputation updates peer reputation score
func UpdateReputation(peer *Peer, delta int) {
	peer.mu.Lock()
	defer peer.mu.Unlock()

	peer.ReputationScore += delta
	if peer.ReputationScore < MinReputationScore {
		peer.ReputationScore = MinReputationScore
	} else if peer.ReputationScore > MaxReputationScore {
		peer.ReputationScore = MaxReputationScore
	}
}

// NewDHTOptions creates default DHT options
func NewDHTOptions() *DHTOptions {
	return &DHTOptions{
		Alpha:   3,  // Kademlia alpha
		K:       20, // Kademlia replication factor
		Timeout: 5 * time.Second,
	}
}

// ParsePeerID extracts peer ID from various formats
func ParsePeerID(data interface{}) (PeerID, error) {
	switch v := data.(type) {
	case string:
		return PeerID(v), nil
	case []byte:
		if len(data.([]byte)) != 16 {
			return "", errors.New("invalid peer ID byte array length")
		}
		return PeerID(data.([]byte)), nil
	default:
		return "", errors.New("unsupported peer ID type")
	}
}

// MessageToHex converts message to hex string for debugging
func MessageToHex(msg *Message) string {
	if msg == nil {
		return ""
	}

	return fmt.Sprintf("V:0x%04X T:0x%02X Len:%d", msg.Type, msg.Version, len(msg.Payload))
}

// CreateNetworkMetrics creates zero metrics
func CreateNetworkMetrics() *NetworkMetrics {
	return &NetworkMetrics{
		TotalPeers:        0,
		ConnectedPeers:    0,
		MessagesPerSecond: 0,
		BandwidthUsage:    0,
		AverageLatency:    0,
		PacketLossRate:    0,
	}
}
