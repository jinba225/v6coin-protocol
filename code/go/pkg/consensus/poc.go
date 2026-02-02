package consensus

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"
)

// nodeScore represents a node ID and its contribution score
type nodeScore struct {
	nodeID string
	score  float64
}

// ConnectionQuality represents quality metrics for a peer connection
type ConnectionQuality struct {
	NodeID      string
	LastSeen    time.Time
	Duration    time.Duration
	Latency     time.Duration
	UptimeRatio float64
	mu          sync.RWMutex
}

// NewConnectionQuality creates a new connection quality tracker
func NewConnectionQuality(nodeID string) *ConnectionQuality {
	return &ConnectionQuality{
		NodeID:      nodeID,
		LastSeen:    time.Now(),
		Duration:    0,
		Latency:     0,
		UptimeRatio: 1.0,
	}
}

// UpdateDuration updates the connection duration
func (cq *ConnectionQuality) UpdateDuration() {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.Duration = time.Since(cq.LastSeen)
	cq.LastSeen = time.Now()
}

// UpdateLatency updates the measured latency
func (cq *ConnectionQuality) UpdateLatency(latency time.Duration) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.Latency = latency
}

// UpdateUptimeRatio updates the uptime ratio (0.0 - 1.0)
func (cq *ConnectionQuality) UpdateUptimeRatio(ratio float64) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	cq.UptimeRatio = math.Max(0.0, math.Min(1.0, ratio))
}

// ContributionScore represents node's contribution metrics
type ContributionScore struct {
	NodeID           string
	Score            float64
	ConnectionTime   time.Duration
	ForwardCount     uint64
	PacketsForwarded uint64
	mu               sync.RWMutex
}

// NewContributionScore creates a new contribution score tracker
func NewContributionScore(nodeID string) *ContributionScore {
	return &ContributionScore{
		NodeID:           nodeID,
		Score:            0.0,
		ConnectionTime:   0,
		ForwardCount:     0,
		PacketsForwarded: 0,
	}
}

// AddForwardCount increments the forward count
func (cs *ContributionScore) AddForwardCount(count uint64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.ForwardCount += count
}

// AddPacketsForwarded increments the forwarded packets count
func (cs *ContributionScore) AddPacketsForwarded(count uint64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.PacketsForwarded += count
}

// UpdateScore calculates and updates the contribution score
func (cs *ContributionScore) UpdateScore(onlineTime time.Duration, uptimeRatio float64, latency time.Duration) float64 {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.ConnectionTime = onlineTime

	// Weighted contribution calculation:
	// Connection time (40%) + Forward count (30%) + Uptime ratio (20%) - Latency (10%)
	score := 0.0

	// Connection time weight (40%)
	// Normalize: assume 90 days (OnlineWindow) is maximum
	normalizedOnlineTime := math.Min(float64(onlineTime.Hours())/float64(OnlineWindow.Hours()), 1.0)
	score += normalizedOnlineTime * 40

	// Forward count weight (30%)
	// Normalize: assume 1e6 forwards is maximum
	normalizedForwardCount := math.Min(float64(cs.ForwardCount)/1e6, 1.0)
	score += normalizedForwardCount * 30

	// Uptime ratio weight (20%)
	score += uptimeRatio * 20

	// Latency penalty (10%) - lower is better
	// Normalize: assume 10 seconds (10000ms) is maximum
	latencyPenalty := math.Min(float64(latency.Milliseconds())/10000.0, 1.0) * 10
	score -= latencyPenalty

	// Ensure score is in range [0, 100]
	cs.Score = math.Max(0.0, math.Min(100.0, score))
	return cs.Score
}

// PoCConsensus implements Proof of Connection consensus
type PoCConsensus struct {
	mu             sync.RWMutex
	connections    map[string]*ConnectionQuality
	contributions  map[string]*ContributionScore
	validators     []string // Current validator set
	myNodeID       string
	prefixMonitor  *PrefixMonitor
	validatorCount int
	maxPerPrefix   int
}

// NewPoCConsensus creates a new PoC consensus instance
func NewPoCConsensus(nodeID string) *PoCConsensus {
	return &PoCConsensus{
		connections:    make(map[string]*ConnectionQuality),
		contributions:  make(map[string]*ContributionScore),
		validators:     make([]string, 0),
		myNodeID:       nodeID,
		prefixMonitor:  NewPrefixMonitor(),
		validatorCount: ValidatorCount,
		maxPerPrefix:   8, // Max 8 devices per prefix
	}
}

// TrackConnection tracks a new peer connection
func (poc *PoCConsensus) TrackConnection(peerID string) {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	if _, exists := poc.connections[peerID]; !exists {
		poc.connections[peerID] = NewConnectionQuality(peerID)
	}

	if _, exists := poc.contributions[peerID]; !exists {
		poc.contributions[peerID] = NewContributionScore(peerID)
	}
}

// UpdateConnectionQuality updates quality metrics for a peer
func (poc *PoCConsensus) UpdateConnectionQuality(peerID string, latency time.Duration, uptime float64) {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	quality, exists := poc.connections[peerID]
	if !exists {
		return
	}

	quality.UpdateLatency(latency)
	quality.UpdateUptimeRatio(uptime)
	quality.UpdateDuration()

	contribution, exists := poc.contributions[peerID]
	if !exists {
		contribution = NewContributionScore(peerID)
		poc.contributions[peerID] = contribution
	}

	contribution.UpdateScore(quality.Duration, uptime, latency)
}

// GetConnectionQuality returns the connection quality for a peer
func (poc *PoCConsensus) GetConnectionQuality(peerID string) *ConnectionQuality {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	if quality, exists := poc.connections[peerID]; exists {
		return quality
	}
	return nil
}

// CalculateContribution calculates the contribution score for a peer
func (poc *PoCConsensus) CalculateContribution(peerID string) float64 {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	if contribution, exists := poc.contributions[peerID]; exists {
		return contribution.Score
	}
	return 0.0
}

// GetTopValidators returns the top N validators by contribution score
func (poc *PoCConsensus) GetTopValidators(count int) []string {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	scores := make([]nodeScore, 0, len(poc.contributions))
	for nodeID, contribution := range poc.contributions {
		scores = append(scores, nodeScore{
			nodeID: nodeID,
			score:  contribution.Score,
		})
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Apply prefix anti-monopoly
	filtered := poc.applyPrefixLimits(scores)

	// Extract validator addresses
	validators := make([]string, 0, count)
	for i := 0; i < count && i < len(filtered); i++ {
		validators = append(validators, filtered[i].nodeID)
	}

	return validators
}

// applyPrefixLimits applies prefix anti-monopoly rules
func (poc *PoCConsensus) applyPrefixLimits(scores []nodeScore) []nodeScore {
	prefixCounts := make(map[string]int)
	result := make([]nodeScore, 0, len(scores))

	for _, node := range scores {
		// Extract prefix from nodeID (assuming nodeID is an IP address or contains prefix info)
		prefix := extractPrefix(node.nodeID)

		if prefixCounts[prefix] < poc.maxPerPrefix {
			result = append(result, node)
			prefixCounts[prefix]++
		}
	}

	return result
}

// UpdateValidators updates the validator set based on current contributions
func (poc *PoCConsensus) UpdateValidators() {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	// Get top validators
	validators := poc.GetTopValidators(poc.validatorCount)
	poc.validators = validators
}

// GetCurrentValidators returns the current validator set
func (poc *PoCConsensus) GetCurrentValidators() []string {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	return poc.validators
}

// EnforcePrefixLimit enforces the maximum number of validators per prefix
func (poc *PoCConsensus) EnforcePrefixLimit(prefix string, maxNodes int) {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	// Count validators with this prefix
	count := 0
	for _, validator := range poc.validators {
		if extractPrefix(validator) == prefix {
			count++
		}
	}

	if count > maxNodes {
		type validatorWithScore struct {
			nodeID string
			score  float64
		}

		var validatorsWithPrefix []validatorWithScore
		for _, validator := range poc.validators {
			if extractPrefix(validator) == prefix {
				score := float64(0)
				if contribution, exists := poc.contributions[validator]; exists {
					score = contribution.Score
				}
				validatorsWithPrefix = append(validatorsWithPrefix, validatorWithScore{
					nodeID: validator,
					score:  score,
				})
			}
		}

		// Sort by score ascending (lowest first)
		sort.Slice(validatorsWithPrefix, func(i, j int) bool {
			return validatorsWithPrefix[i].score < validatorsWithPrefix[j].score
		})

		toRemove := count - maxNodes
		newValidators := make([]string, 0, len(poc.validators)-toRemove)
		removedSet := make(map[string]bool)

		for i := 0; i < toRemove && i < len(validatorsWithPrefix); i++ {
			removedSet[validatorsWithPrefix[i].nodeID] = true
		}

		for _, validator := range poc.validators {
			if !removedSet[validator] {
				newValidators = append(newValidators, validator)
			}
		}

		poc.validators = newValidators
	}
}

// extractPrefix extracts the /64 prefix from an address
func extractPrefix(address string) string {
	// Simplified implementation: return first 8 characters as prefix
	// In production, this would parse the IPv6 address and extract the /64 prefix
	if len(address) >= 8 {
		return address[:8]
	}
	return address
}

// ValidateBlock validates a block using PoC consensus rules
func (poc *PoCConsensus) ValidateBlock(block *Block) error {
	if block == nil || block.Header == nil {
		return errors.New("block cannot be nil")
	}

	// Validate block signature
	if len(block.Header.Signature) == 0 {
		return errors.New("block missing validator signature")
	}

	// Validate validator
	validatorID := string(block.Header.ValidatorID)
	if !poc.IsValidValidator(validatorID) {
		return errors.New("block signed by non-validator")
	}

	// Validate block height and hash
	if block.Header.Version != 0x0100 {
		return errors.New("invalid block version")
	}

	return nil
}

// IsValidValidator checks if a node is in the current validator set
func (poc *PoCConsensus) IsValidValidator(nodeID string) bool {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	for _, validator := range poc.validators {
		if validator == nodeID {
			return true
		}
	}
	return false
}

// GetCurrentValidator returns the validator for the given block height
func (poc *PoCConsensus) GetCurrentValidator(height uint64) string {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	if len(poc.validators) == 0 {
		return ""
	}

	index := int(height) % len(poc.validators)
	return poc.validators[index]
}

// UpdateForwardCount updates the forward count for a node
func (poc *PoCConsensus) UpdateForwardCount(peerID string, count uint64) {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	if contribution, exists := poc.contributions[peerID]; exists {
		contribution.AddForwardCount(count)
	}
}

// UpdatePacketsForwarded updates the packets forwarded count for a node
func (poc *PoCConsensus) UpdatePacketsForwarded(peerID string, count uint64) {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	if contribution, exists := poc.contributions[peerID]; exists {
		contribution.AddPacketsForwarded(count)
	}
}

// GetContributionScore returns the contribution score for a node
func (poc *PoCConsensus) GetContributionScore(peerID string) float64 {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	if contribution, exists := poc.contributions[peerID]; exists {
		return contribution.Score
	}
	return 0.0
}

// GetAllContributions returns all contribution scores
func (poc *PoCConsensus) GetAllContributions() map[string]float64 {
	poc.mu.RLock()
	defer poc.mu.RUnlock()

	result := make(map[string]float64, len(poc.contributions))
	for nodeID, contribution := range poc.contributions {
		result[nodeID] = contribution.Score
	}
	return result
}

// RemovePeer removes a peer from tracking
func (poc *PoCConsensus) RemovePeer(peerID string) {
	poc.mu.Lock()
	defer poc.mu.Unlock()

	delete(poc.connections, peerID)
	delete(poc.contributions, peerID)
}

// GetPeerCount returns the number of tracked peers
func (poc *PoCConsensus) GetPeerCount() int {
	poc.mu.RLock()
	defer poc.mu.RUnlock()
	return len(poc.connections)
}
