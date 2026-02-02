package p2p

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// DHT parameters per Kademlia specification
const (
	K     = 20 // K-bucket size
	Alpha = 3  // Concurrency parameter for lookups
)

// NodeID represents a unique node identifier (16 bytes)
type NodeID struct {
	Key  []byte // 16-byte node ID
	IP   net.IP
	Port uint16
}

// KBucket represents a Kademlia k-bucket
type KBucket struct {
	nodes    []*NodeID
	lastSeen time.Time
	mu       sync.RWMutex
}

// Kademlia implements Kademlia DHT for node discovery
type Kademlia struct {
	mu      sync.RWMutex
	myID    NodeID
	buckets [256]*KBucket // 256 k-buckets for 128-bit ID space
	routing map[string]*NodeID
}

// NewKademlia creates a new Kademlia DHT instance
func NewKademlia(myKey []byte, myIP net.IP, myPort uint16) *Kademlia {
	k := &Kademlia{
		myID: NodeID{
			Key:  myKey,
			IP:   myIP,
			Port: myPort,
		},
		buckets: [256]*KBucket{},
		routing: make(map[string]*NodeID),
	}

	// Initialize buckets
	for i := 0; i < 256; i++ {
		k.buckets[i] = &KBucket{
			nodes:    make([]*NodeID, 0, K),
			lastSeen: time.Now(),
		}
	}

	return k
}

// AddNode adds a node to the DHT
func (k *Kademlia) AddNode(nodeID NodeID) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Don't add self
	if k.isSelfNode(nodeID) {
		return nil
	}

	// Calculate bucket index
	bucketIndex := k.calculateBucketIndex(nodeID.Key)
	bucket := k.buckets[bucketIndex]

	// Check if node already exists in bucket
	for _, existing := range bucket.nodes {
		if k.nodeIDsEqual(existing, &nodeID) {
			// Update last seen time
			bucket.lastSeen = time.Now()
			return nil
		}
	}

	// Add to bucket if space available
	if len(bucket.nodes) < K {
		bucket.nodes = append(bucket.nodes, &nodeID)
		bucket.lastSeen = time.Now()
	}

	// Add to routing map
	hash := sha256.Sum256(nodeID.Key)
	k.routing[hex.EncodeToString(hash[:])] = &nodeID

	return nil
}

// FindNode finds the K closest nodes to the given key
func (k *Kademlia) FindNode(key []byte) []NodeID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	candidates := make([]nodeDistance, 0)

	// Collect candidates from all buckets
	for _, bucket := range k.buckets {
		for _, node := range bucket.nodes {
			distance := k.xorDistance(key, node.Key)
			candidates = append(candidates, nodeDistance{
				node:     node,
				distance: distance,
			})
		}
	}

	// Sort by distance
	k.sortByDistance(candidates)

	// Return top K nodes
	result := make([]NodeID, 0, K)
	for i := 0; i < len(candidates) && i < K; i++ {
		result = append(result, *candidates[i].node)
	}

	return result
}

// RemoveNode removes a node from the DHT
func (k *Kademlia) RemoveNode(key []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	bucketIndex := k.calculateBucketIndex(key)
	bucket := k.buckets[bucketIndex]

	// Find and remove node from bucket
	for i, node := range bucket.nodes {
		if k.bytesEqual(node.Key, key) {
			bucket.nodes = append(bucket.nodes[:i], bucket.nodes[i+1:]...)

			// Remove from routing map
			hash := sha256.Sum256(key)
			delete(k.routing, hex.EncodeToString(hash[:]))

			return nil
		}
	}

	return errors.New("node not found in DHT")
}

// FindClosestNodes finds the N closest nodes to the target
func (k *Kademlia) FindClosestNodes(target []byte, n int) []NodeID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	nodes := k.findClosestNodes(target, n)
	result := make([]NodeID, len(nodes))
	for i, node := range nodes {
		result[i] = *node
	}

	return result
}

// findClosestNodes finds N closest nodes (internal, holds lock)
func (k *Kademlia) findClosestNodes(target []byte, n int) []*NodeID {
	candidates := make([]nodeDistance, 0)

	// Collect candidates from all buckets
	for _, bucket := range k.buckets {
		for _, node := range bucket.nodes {
			distance := k.xorDistance(target, node.Key)
			candidates = append(candidates, nodeDistance{
				node:     node,
				distance: distance,
			})
		}
	}

	// Sort by distance
	k.sortByDistance(candidates)

	// Return top N nodes
	result := make([]*NodeID, 0, n)
	for i := 0; i < len(candidates) && i < n; i++ {
		result = append(result, candidates[i].node)
	}

	return result
}

// GetAllNodes returns all nodes in the DHT
func (k *Kademlia) GetAllNodes() []NodeID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	nodes := make([]NodeID, 0)
	for _, bucket := range k.buckets {
		for _, node := range bucket.nodes {
			nodes = append(nodes, *node)
		}
	}

	return nodes
}

// GetNodeCount returns the total number of nodes in the DHT
func (k *Kademlia) GetNodeCount() int {
	k.mu.RLock()
	defer k.mu.RUnlock()

	count := 0
	for _, bucket := range k.buckets {
		count += len(bucket.nodes)
	}

	return count
}

// GetMyNodeID returns the local node ID
func (k *Kademlia) GetMyNodeID() NodeID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	return k.myID
}

// RefreshBuckets refreshes stale buckets
func (k *Kademlia) RefreshBuckets() int {
	k.mu.Lock()
	defer k.mu.Unlock()

	refreshed := 0
	now := time.Now()

	for _, bucket := range k.buckets {
		if now.Sub(bucket.lastSeen) > time.Hour {
			bucket.lastSeen = now
			refreshed++
		}
	}

	return refreshed
}

// ClearAll clears all nodes from the DHT
func (k *Kademlia) ClearAll() {
	k.mu.Lock()
	defer k.mu.Unlock()

	for i := 0; i < 256; i++ {
		k.buckets[i] = &KBucket{
			nodes:    make([]*NodeID, 0, K),
			lastSeen: time.Now(),
		}
	}

	k.routing = make(map[string]*NodeID)
}

// calculateBucketIndex calculates the bucket index for a node ID
func (k *Kademlia) calculateBucketIndex(nodeKey []byte) int {
	xor := k.xorDistance(k.myID.Key, nodeKey)

	for i := 0; i < 128; i++ {
		bit := uint(i)
		byteIndex := int(bit / 8)
		bitIndex := int(bit % 8)

		if byteIndex >= len(xor) {
			return 255
		}

		if xor[byteIndex]&(1<<(7-bitIndex)) != 0 {
			return 127 - byteIndex
		}
	}

	return 0
}

// xorDistance calculates XOR distance between two keys
func (k *Kademlia) xorDistance(key1, key2 []byte) []byte {
	result := make([]byte, 16)
	minLen := 16

	if len(key1) < minLen {
		minLen = len(key1)
	}
	if len(key2) < minLen {
		minLen = len(key2)
	}

	for i := 0; i < minLen; i++ {
		result[i] = key1[i] ^ key2[i]
	}

	return result
}

// nodeDistance represents a node with its distance to a target
type nodeDistance struct {
	node     *NodeID
	distance []byte
}

// sortByDistance sorts nodes by XOR distance (simple bubble sort)
func (k *Kademlia) sortByDistance(nodes []nodeDistance) {
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			if k.compareDistance(nodes[i].distance, nodes[j].distance) > 0 {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}

// compareDistance compares two distance byte arrays
func (k *Kademlia) compareDistance(d1, d2 []byte) int {
	for i := 0; i < len(d1) && i < len(d2); i++ {
		if d1[i] < d2[i] {
			return -1
		}
		if d1[i] > d2[i] {
			return 1
		}
	}
	return 0
}

// isSelfNode checks if the node is the local node
func (k *Kademlia) isSelfNode(nodeID NodeID) bool {
	return k.bytesEqual(k.myID.Key, nodeID.Key) &&
		k.myID.IP.Equal(nodeID.IP) &&
		k.myID.Port == nodeID.Port
}

// nodeIDsEqual compares two node IDs
func (k *Kademlia) nodeIDsEqual(n1, n2 *NodeID) bool {
	return k.bytesEqual(n1.Key, n2.Key) &&
		n1.IP.Equal(n2.IP) &&
		n1.Port == n2.Port
}

// bytesEqual compares two byte arrays
func (k *Kademlia) bytesEqual(b1, b2 []byte) bool {
	if len(b1) != len(b2) {
		return false
	}
	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			return false
		}
	}
	return true
}

// Bootstrap adds bootstrap nodes to the DHT
func (k *Kademlia) Bootstrap(bootstrapNodes []NodeID) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	for _, node := range bootstrapNodes {
		if !k.isSelfNode(node) {
			bucketIndex := k.calculateBucketIndex(node.Key)
			bucket := k.buckets[bucketIndex]

			if len(bucket.nodes) < K {
				// Check if already exists
				exists := false
				for _, existing := range bucket.nodes {
					if k.nodeIDsEqual(existing, &node) {
						exists = true
						break
					}
				}

				if !exists {
					bucket.nodes = append(bucket.nodes, &node)
					bucket.lastSeen = time.Now()

					hash := sha256.Sum256(node.Key)
					k.routing[hex.EncodeToString(hash[:])] = &node
				}
			}
		}
	}

	return nil
}

// FindNodeByAddress finds a node by IP address and port
func (k *Kademlia) FindNodeByAddress(ip net.IP, port uint16) *NodeID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	for _, bucket := range k.buckets {
		for _, node := range bucket.nodes {
			if node.IP.Equal(ip) && node.Port == port {
				return node
			}
		}
	}

	return nil
}

// UpdateNodeLastSeen updates the last seen time for a node
func (k *Kademlia) UpdateNodeLastSeen(key []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	bucketIndex := k.calculateBucketIndex(key)
	bucket := k.buckets[bucketIndex]

	for _, node := range bucket.nodes {
		if k.bytesEqual(node.Key, key) {
			bucket.lastSeen = time.Now()
			return nil
		}
	}

	return fmt.Errorf("node not found in DHT")
}

// GetRandomNodes returns random nodes from the DHT
func (k *Kademlia) GetRandomNodes(count int) []NodeID {
	k.mu.RLock()
	defer k.mu.RUnlock()

	allNodes := make([]*NodeID, 0)
	for _, bucket := range k.buckets {
		for _, node := range bucket.nodes {
			allNodes = append(allNodes, node)
		}
	}

	if len(allNodes) == 0 {
		return []NodeID{}
	}

	result := make([]NodeID, 0, count)
	for i := 0; i < count && i < len(allNodes); i++ {
		// Simple random selection
		idx := i % len(allNodes)
		result = append(result, *allNodes[idx])
	}

	return result
}
