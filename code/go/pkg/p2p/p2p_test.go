package p2p

import (
	"net"
	"testing"
	"time"
)

func TestNewPeer(t *testing.T) {
	peerID := PeerID("test-peer-1")
	address := net.ParseIP("2001:db8::1")
	port := uint16(38901)

	peer := NewPeer(peerID, address, port)

	if peer.ID != peerID {
		t.Errorf("Expected peer ID %s, got %s", peerID, peer.ID)
	}

	if !peer.Address.Equal(address) {
		t.Errorf("Expected address %s, got %s", address, peer.Address)
	}

	if peer.Port != port {
		t.Errorf("Expected port %d, got %d", port, peer.Port)
	}

	if peer.State != PeerStateNew {
		t.Errorf("Expected state %s, got %s", PeerStateNew, peer.State)
	}
}

func TestNewMessage(t *testing.T) {
	msgType := MessageType(MsgTypePing)
	payload := []byte("test payload")

	msg := NewMessage(msgType, payload)

	if msg.Type != MessageType(MsgTypePing) {
		t.Errorf("Expected message type %d, got %d", MsgTypePing, msg.Type)
	}

	if string(msg.Payload) != string(payload) {
		t.Errorf("Expected payload %s, got %s", payload, msg.Payload)
	}

	if msg.Version != ProtocolVersion {
		t.Errorf("Expected version %d, got %d", ProtocolVersion, msg.Version)
	}

	if len(msg.ID) != 16 {
		t.Errorf("Expected message ID length 16, got %d", len(msg.ID))
	}

	if msg.TTL != uint8(MessageTTL/time.Second) {
		t.Errorf("Expected TTL %d, got %d", uint8(MessageTTL/time.Second), msg.TTL)
	}
}

func TestIsValidMessageType(t *testing.T) {
	tests := []struct {
		msgType MessageType
		valid   bool
	}{
		{MsgTypeHandshake, true},
		{MsgTypePing, true},
		{MsgTypePong, true},
		{MsgTypePeerList, true},
		{MsgTypeBroadcast, true},
		{MsgTypeDirect, true},
		{MsgTypeV6CoinTx, true},
		{MsgTypeDisconnect, true},
		{MsgTypeKeepAlive, true},
		{MessageType(0xFF), false},
	}

	for _, test := range tests {
		result := IsValidMessageType(test.msgType)
		if result != test.valid {
			t.Errorf("Expected MessageType %d to be %v, got %v", test.msgType, test.valid, result)
		}
	}
}

func TestRoutingTable(t *testing.T) {
	rt := NewRoutingTable()

	if rt.GetPeerCount() != 0 {
		t.Errorf("Expected empty routing table, got %d peers", rt.GetPeerCount())
	}

	// Add a peer
	peerID := PeerID("test-peer-1")
	address := net.ParseIP("2001:db8::1")
	peer := NewPeer(peerID, address, 38901)

	err := rt.AddPeer(peer)
	if err != nil {
		t.Errorf("Failed to add peer: %v", err)
	}

	if rt.GetPeerCount() != 1 {
		t.Errorf("Expected 1 peer, got %d", rt.GetPeerCount())
	}

	// Get peer
	retrievedPeer := rt.GetPeer(peerID)
	if retrievedPeer == nil {
		t.Errorf("Failed to retrieve peer")
	}

	if retrievedPeer.ID != peerID {
		t.Errorf("Expected peer ID %s, got %s", peerID, retrievedPeer.ID)
	}

	// Get all peers
	allPeers := rt.GetAllPeers()
	if len(allPeers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(allPeers))
	}

	// Remove peer
	err = rt.RemovePeer(peerID)
	if err != nil {
		t.Errorf("Failed to remove peer: %v", err)
	}

	if rt.GetPeerCount() != 0 {
		t.Errorf("Expected empty routing table, got %d peers", rt.GetPeerCount())
	}
}

func TestConnectionManager(t *testing.T) {
	cm := NewConnectionManager(10)

	// Add a peer
	peerID := "test-peer-1"
	address := net.ParseIP("2001:db8::1")
	peer := &Peer{
		ID:              PeerID(peerID),
		Address:         address,
		Port:            38901,
		LastSeen:        time.Now(),
		State:           PeerStateConnected,
		ReputationScore: 500,
	}

	cm.connectionsMu.Lock()
	cm.peers[peerID] = peer
	cm.connections[peerID] = nil // Mock connection
	cm.connectionsMu.Unlock()

	// Update peer stats
	cm.UpdatePeerStats(peerID, 1024, 2048, 10)

	// Ban peer
	err := cm.BanPeer(peerID, "spamming", 100)
	if err != nil {
		t.Errorf("Failed to ban peer: %v", err)
	}

	// Get connected peers
	connected := cm.GetConnectedPeers()
	if len(connected) != 1 {
		t.Errorf("Expected 1 connected peer, got %d", len(connected))
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10)

	// Allow 10 messages per second
	for i := 0; i < 10; i++ {
		if !rl.Allow() {
			t.Errorf("Expected message %d to be allowed", i)
		}
		rl.Record()
	}

	// 11th message should be blocked
	if rl.Allow() {
		t.Errorf("Expected message 11 to be blocked")
	}

	// Wait a second and try again
	time.Sleep(time.Second)
	if !rl.Allow() {
		t.Errorf("Expected message to be allowed after waiting")
	}
}

func TestCalculateDistance(t *testing.T) {
	addr1 := net.ParseIP("2001:db8::1")
	addr2 := net.ParseIP("2001:db8::2")

	peer1 := &Peer{
		ID:       PeerID("peer-1"),
		Address:  addr1,
		PingTime: time.Now(),
	}

	peer2 := &Peer{
		ID:       PeerID("peer-2"),
		Address:  addr2,
		PingTime: time.Now().Add(100 * time.Millisecond),
	}

	distance := CalculateDistance(peer1, peer2)
	if distance < 0 {
		t.Errorf("Distance should be non-negative, got %d", distance)
	}
}

func TestIsPeerHealthy(t *testing.T) {
	peer := &Peer{
		ID:       PeerID("test-peer"),
		State:    PeerStateConnected,
		LastSeen: time.Now(),
	}

	if !IsPeerHealthy(peer) {
		t.Errorf("Expected peer to be healthy")
	}

	// Make peer stale
	peer.LastSeen = time.Now().Add(-25 * time.Hour)
	if IsPeerHealthy(peer) {
		t.Errorf("Expected stale peer to be unhealthy")
	}
}

func TestPeerSerialization(t *testing.T) {
	peers := []PeerInfo{
		{
			Address:    net.ParseIP("2001:db8::1"),
			Port:       38901,
			LastSeen:   time.Now(),
			Reputation: 500,
		},
	}

	// Serialize
	data, err := SerializePeerList(peers)
	if err != nil {
		t.Errorf("Failed to serialize peers: %v", err)
	}

	// Deserialize
	deserializedPeers, err := DeserializePeerList(data)
	if err != nil {
		t.Errorf("Failed to deserialize peers: %v", err)
	}

	if len(deserializedPeers) != len(peers) {
		t.Errorf("Expected %d peers, got %d", len(peers), len(deserializedPeers))
	}

	// Check peer data
	if !deserializedPeers[0].Address.Equal(peers[0].Address) {
		t.Errorf("Address mismatch")
	}

	if deserializedPeers[0].Port != peers[0].Port {
		t.Errorf("Port mismatch: expected %d, got %d", peers[0].Port, deserializedPeers[0].Port)
	}

	if deserializedPeers[0].Reputation != peers[0].Reputation {
		t.Errorf("Reputation mismatch: expected %d, got %d", peers[0].Reputation, deserializedPeers[0].Reputation)
	}
}

func TestMessageValidation(t *testing.T) {
	validMsg := &Message{
		Version:   ProtocolVersion,
		Type:      MsgTypePing,
		Timestamp: uint64(time.Now().Unix()),
		ID:        MessageID(),
		TTL:       60,
		Hops:      0,
		Payload:   []byte("test"),
	}

	err := IsValidMessage(validMsg)
	if err != nil {
		t.Errorf("Expected valid message, got error: %v", err)
	}

	// Test invalid version
	invalidMsg := &Message{
		Version:   0x0000,
		Type:      MsgTypePing,
		Timestamp: uint64(time.Now().Unix()),
		ID:        MessageID(),
		TTL:       60,
		Hops:      0,
		Payload:   []byte("test"),
	}

	err = IsValidMessage(invalidMsg)
	if err == nil {
		t.Errorf("Expected error for invalid version")
	}

	// Test too large TTL
	invalidMsg2 := &Message{
		Version:   ProtocolVersion,
		Type:      MsgTypePing,
		Timestamp: uint64(time.Now().Unix()),
		ID:        MessageID(),
		TTL:       255,
		Hops:      0,
		Payload:   []byte("test"),
	}

	err = IsValidMessage(invalidMsg2)
	if err == nil {
		t.Errorf("Expected error for too large TTL")
	}
}
