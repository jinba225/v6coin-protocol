package p2p

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Router manages P2P message routing
type Router struct {
	mu           sync.RWMutex
	routes       map[MessageType]MessageHandler
	peerManager  *PeerManager
	dht          *Kademlia
	broadcast    *Broadcaster
	stats        *RouterStats
	routingTable *RoutingTable
}

// RouterStats tracks routing statistics
type RouterStats struct {
	MessagesRouted    uint64
	MessagesDropped   uint64
	MessagesForwarded uint64
	mu                sync.RWMutex
}

// NewRouter creates a new P2P router
func NewRouter(peerManager *PeerManager, dht *Kademlia) *Router {
	return &Router{
		routes:       make(map[MessageType]MessageHandler),
		peerManager:  peerManager,
		dht:          dht,
		broadcast:    NewBroadcaster(),
		stats:        &RouterStats{},
		routingTable: NewRoutingTable(),
	}
}

// RegisterHandler registers a message handler for a message type
func (r *Router) RegisterHandler(msgType MessageType, handler MessageHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.routes[msgType]; exists {
		return fmt.Errorf("handler already registered for message type %d", msgType)
	}

	r.routes[msgType] = handler
	return nil
}

// UnregisterHandler removes a message handler
func (r *Router) UnregisterHandler(msgType MessageType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.routes, msgType)
}

// RouteMessage routes a message to the appropriate handler
func (r *Router) RouteMessage(from net.IP, msg *Message) error {
	r.mu.RLock()
	handler, exists := r.routes[msg.Type]
	r.mu.RUnlock()

	if !exists {
		r.stats.mu.Lock()
		r.stats.MessagesDropped++
		r.stats.mu.Unlock()

		return fmt.Errorf("no handler registered for message type %d", msg.Type)
	}

	r.stats.mu.Lock()
	r.stats.MessagesRouted++
	r.stats.mu.Unlock()

	return handler.HandleMessage(from, msg)
}

// BroadcastMessage broadcasts a message to all connected peers
func (r *Router) BroadcastMessage(msg *Message, excludeFrom net.IP) error {
	peers := r.peerManager.GetConnectedPeers()
	if len(peers) == 0 {
		return errors.New("no connected peers")
	}

	sentCount := 0
	for _, peer := range peers {
		if excludeFrom != nil && peer.Address.Equal(excludeFrom) {
			continue
		}

		if err := r.sendMessageToPeer(peer.ID, msg); err == nil {
			sentCount++
		}
	}

	r.stats.mu.Lock()
	r.stats.MessagesForwarded += uint64(sentCount)
	r.stats.mu.Unlock()

	return nil
}

// SendToPeer sends a message to a specific peer
func (r *Router) SendToPeer(peerID PeerID, msg *Message) error {
	return r.sendMessageToPeer(peerID, msg)
}

// sendMessageToPeer sends a message to a peer
func (r *Router) sendMessageToPeer(peerID PeerID, msg *Message) error {
	peer := r.peerManager.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	conn := r.peerManager.GetPeerConnection(peerID)
	if conn == nil {
		return fmt.Errorf("peer %s not connected", peerID)
	}

	return r.sendViaConnection(conn, msg)
}

// sendViaConnection sends a message via a connection
func (r *Router) sendViaConnection(conn net.Conn, msg *Message) error {
	data, err := MessageToBytes(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// BroadcastToRandomPeers broadcasts a message to random subset of peers
func (r *Router) BroadcastToRandomPeers(msg *Message, count int, excludeFrom net.IP) error {
	peers := r.peerManager.GetConnectedPeers()
	if len(peers) == 0 {
		return errors.New("no connected peers")
	}

	if count > len(peers) {
		count = len(peers)
	}

	sentCount := 0
	for i := 0; i < count; i++ {
		peer := peers[i]
		if excludeFrom != nil && peer.Address.Equal(excludeFrom) {
			continue
		}

		if err := r.sendMessageToPeer(peer.ID, msg); err == nil {
			sentCount++
		}
	}

	r.stats.mu.Lock()
	r.stats.MessagesForwarded += uint64(sentCount)
	r.stats.mu.Unlock()

	return nil
}

// ForwardMessage forwards a message to next hops
func (r *Router) ForwardMessage(msg *Message, from net.IP) error {
	if msg.Hops >= GetMaxHops() {
		return errors.New("maximum hops exceeded")
	}

	msg.Hops++
	msg.TTL--

	if msg.TTL == 0 {
		return errors.New("message TTL expired")
	}

	targetPeers := r.dht.FindClosestNodes(msg.From.To16(), 5)
	sentCount := 0

	for _, node := range targetPeers {
		peer := r.peerManager.GetPeer(PeerID(hex.EncodeToString(node.Key)))
		if peer == nil {
			continue
		}

		if err := r.sendMessageToPeer(peer.ID, msg); err == nil {
			sentCount++
		}
	}

	r.stats.mu.Lock()
	r.stats.MessagesForwarded += uint64(sentCount)
	r.stats.mu.Unlock()

	return nil
}

// GetStats returns router statistics
func (r *Router) GetStats() (uint64, uint64, uint64) {
	r.stats.mu.RLock()
	defer r.stats.mu.RUnlock()

	return r.stats.MessagesRouted, r.stats.MessagesDropped, r.stats.MessagesForwarded
}

// ResetStats resets router statistics
func (r *Router) ResetStats() {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()

	r.stats.MessagesRouted = 0
	r.stats.MessagesDropped = 0
	r.stats.MessagesForwarded = 0
}

// BroadcastTransaction broadcasts a transaction message
func (r *Router) BroadcastTransaction(txData []byte) error {
	msg := NewMessage(MsgTypeV6CoinTx, txData)
	return r.BroadcastMessage(msg, nil)
}

// BroadcastBlock broadcasts a block message
func (r *Router) BroadcastBlock(blockData []byte) error {
	msg := NewMessage(MsgTypeBroadcast, blockData)
	return r.BroadcastMessage(msg, nil)
}

// SendPing sends a PING message to a peer
func (r *Router) SendPing(peerID PeerID) error {
	peer := r.peerManager.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	msg := CreatePingMessage(peer.Address, r.dht.GetMyNodeID().IP)
	return r.sendMessageToPeer(peerID, msg)
}

// SendPong sends a PONG message to a peer
func (r *Router) SendPong(peerID PeerID, pingMsg *Message) error {
	peer := r.peerManager.GetPeer(peerID)
	if peer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	msg := CreatePongMessage(r.dht.GetMyNodeID().IP, peer.Address)
	return r.sendMessageToPeer(peerID, msg)
}

// HandleDiscoveryMessage handles FIND_NODE message
func (r *Router) HandleDiscoveryMessage(from net.IP, targetID PeerID) ([]NodeID, error) {
	nodes := r.dht.FindNode([]byte(targetID))
	return nodes, nil
}

// HandlePeerListMessage handles PEER_LIST message
func (r *Router) HandlePeerListMessage() []*Peer {
	peers := r.peerManager.GetHealthyPeers()
	return peers
}

// Broadcaster manages efficient message broadcasting
type Broadcaster struct {
	mu           sync.RWMutex
	pendingBcast []*BroadcastTask
	maxPending   int
}

// BroadcastTask represents a pending broadcast task
type BroadcastTask struct {
	Message     *Message
	ExcludeFrom net.IP
	TargetPeers []PeerID
	CreatedAt   int64
}

// NewBroadcaster creates a new broadcaster
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		pendingBcast: make([]*BroadcastTask, 0, 100),
		maxPending:   100,
	}
}

// AddBroadcast adds a broadcast task
func (b *Broadcaster) AddBroadcast(msg *Message, excludeFrom net.IP) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pendingBcast) >= b.maxPending {
		return errors.New("broadcast queue full")
	}

	task := &BroadcastTask{
		Message:     msg,
		ExcludeFrom: excludeFrom,
		CreatedAt:   time.Now().Unix(),
	}

	b.pendingBcast = append(b.pendingBcast, task)
	return nil
}

// ProcessBroadcasts processes pending broadcasts
func (b *Broadcaster) ProcessBroadcasts(sendFunc func(PeerID, *Message) error) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pendingBcast) == 0 {
		return 0
	}

	sent := 0
	for _, task := range b.pendingBcast {
		for _, peerID := range task.TargetPeers {
			if err := sendFunc(peerID, task.Message); err == nil {
				sent++
			}
		}
	}

	b.pendingBcast = make([]*BroadcastTask, 0, 100)
	return sent
}

// GetPendingCount returns number of pending broadcasts
func (b *Broadcaster) GetPendingCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.pendingBcast)
}

// Clear clears all pending broadcasts
func (b *Broadcaster) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pendingBcast = make([]*BroadcastTask, 0, 100)
}
