package p2p

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PeerManager manages peer connections and state
type PeerManager struct {
	mu              sync.RWMutex
	peers           map[PeerID]*Peer
	connectedPeers  map[PeerID]bool
	peerConnections map[PeerID]net.Conn
	maxPeers        int
	peerCount       int
}

// NewPeerManager creates a new peer manager
func NewPeerManager(maxPeers int) *PeerManager {
	return &PeerManager{
		peers:           make(map[PeerID]*Peer),
		connectedPeers:  make(map[PeerID]bool),
		peerConnections: make(map[PeerID]net.Conn),
		maxPeers:        maxPeers,
		peerCount:       0,
	}
}

// AddPeer adds a new peer to the manager
func (pm *PeerManager) AddPeer(peer *Peer) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.peers) >= pm.maxPeers {
		return fmt.Errorf("peer manager full (max %d peers)", pm.maxPeers)
	}

	if _, exists := pm.peers[peer.ID]; exists {
		return fmt.Errorf("peer %s already exists", peer.ID)
	}

	pm.peers[peer.ID] = peer
	pm.peerCount++

	return nil
}

// RemovePeer removes a peer from the manager
func (pm *PeerManager) RemovePeer(peerID PeerID) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[peerID]; !exists {
		return fmt.Errorf("peer %s not found", peerID)
	}

	// Close connection if exists
	if conn, exists := pm.peerConnections[peerID]; exists {
		conn.Close()
		delete(pm.peerConnections, peerID)
	}

	delete(pm.connectedPeers, peerID)
	delete(pm.peers, peerID)
	pm.peerCount--

	return nil
}

// GetPeer retrieves a peer by ID
func (pm *PeerManager) GetPeer(peerID PeerID) *Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.peers[peerID]
}

// GetAllPeers returns all known peers
func (pm *PeerManager) GetAllPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*Peer, 0, len(pm.peers))
	for _, peer := range pm.peers {
		peers = append(peers, peer)
	}

	return peers
}

// GetConnectedPeers returns all connected peers
func (pm *PeerManager) GetConnectedPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*Peer, 0, len(pm.connectedPeers))
	for id := range pm.connectedPeers {
		if peer, exists := pm.peers[id]; exists {
			peers = append(peers, peer)
		}
	}

	return peers
}

// MarkPeerConnected marks a peer as connected
func (pm *PeerManager) MarkPeerConnected(peerID PeerID, conn net.Conn) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.connectedPeers[peerID] = true
	pm.peerConnections[peerID] = conn

	// Update peer state
	if peer, exists := pm.peers[peerID]; exists {
		peer.mu.Lock()
		peer.State = PeerStateConnected
		peer.LastSeen = time.Now()
		peer.mu.Unlock()
	}
}

// MarkPeerDisconnected marks a peer as disconnected
func (pm *PeerManager) MarkPeerDisconnected(peerID PeerID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.connectedPeers, peerID)

	// Close connection
	if conn, exists := pm.peerConnections[peerID]; exists {
		conn.Close()
		delete(pm.peerConnections, peerID)
	}

	// Update peer state
	if peer, exists := pm.peers[peerID]; exists {
		peer.mu.Lock()
		peer.State = PeerStateOffline
		peer.mu.Unlock()
	}
}

// GetPeerCount returns the number of known peers
func (pm *PeerManager) GetPeerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.peerCount
}

// GetConnectedPeerCount returns the number of connected peers
func (pm *PeerManager) GetConnectedPeerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.connectedPeers)
}

// UpdatePeerLastSeen updates the last seen time for a peer
func (pm *PeerManager) UpdatePeerLastSeen(peerID PeerID) {
	pm.mu.RLock()
	peer := pm.peers[peerID]
	pm.mu.RUnlock()

	if peer != nil {
		peer.mu.Lock()
		peer.LastSeen = time.Now()
		peer.mu.Unlock()
	}
}

// UpdatePeerConnection updates peer connection statistics
func (pm *PeerManager) UpdatePeerConnection(peerID PeerID, bytesSent, bytesReceived uint64, messageCount uint32) {
	pm.mu.RLock()
	peer := pm.peers[peerID]
	pm.mu.RUnlock()

	if peer != nil {
		peer.mu.Lock()
		peer.BytesSent += bytesSent
		peer.BytesReceived += bytesReceived
		peer.MessageCount += messageCount
		peer.mu.Unlock()
	}
}

// UpdatePeerReputation updates a peer's reputation score
func (pm *PeerManager) UpdatePeerReputation(peerID PeerID, delta int) {
	pm.mu.RLock()
	peer := pm.peers[peerID]
	pm.mu.RUnlock()

	if peer != nil {
		UpdateReputation(peer, delta)
	}
}

// GetHealthyPeers returns peers that are healthy for routing
func (pm *PeerManager) GetHealthyPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	healthyPeers := make([]*Peer, 0)
	for _, peer := range pm.peers {
		if IsPeerHealthy(peer) {
			healthyPeers = append(healthyPeers, peer)
		}
	}

	return healthyPeers
}

// FindPeerByAddress finds a peer by address
func (pm *PeerManager) FindPeerByAddress(address net.IP, port uint16) *Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, peer := range pm.peers {
		if peer.Address.Equal(address) && peer.Port == port {
			return peer
		}
	}

	return nil
}

// ClearStalePeers removes stale peers that haven't been seen recently
func (pm *PeerManager) ClearStalePeers(maxAge time.Duration) int {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	staleCount := 0

	for id, peer := range pm.peers {
		if now.Sub(peer.LastSeen) > maxAge {
			// Don't remove connected peers
			if _, connected := pm.connectedPeers[id]; !connected {
				delete(pm.peers, id)
				staleCount++
				pm.peerCount--
			}
		}
	}

	return staleCount
}

// GetPeerConnection returns the connection for a peer
func (pm *PeerManager) GetPeerConnection(peerID PeerID) net.Conn {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.peerConnections[peerID]
}

// HasPeer checks if a peer exists
func (pm *PeerManager) HasPeer(peerID PeerID) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, exists := pm.peers[peerID]
	return exists
}

// IsConnected checks if a peer is connected
func (pm *PeerManager) IsConnected(peerID PeerID) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.connectedPeers[peerID]
}
