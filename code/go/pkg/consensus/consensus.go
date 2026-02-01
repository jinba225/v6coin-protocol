package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/crypto"
	"github.com/jinba225/v6coin-protocol/pkg/p2p"
)

const (
	// Consensus parameters
	BlockInterval      = 10 * time.Second
	ValidatorCount     = 100
	ConfirmationBlocks = 3
	OnlineWindow       = 90 * 24 * time.Hour // 90 days
	MaxPacketLossRate  = 0.30                // 30%
	MaxPrefixDevices   = 50
	RevenueDecayFactor = 10.0
	InitialBlockReward = 100 // V6
	HalvingYears       = 4
)

// Block represents a V6Coin block
type Block struct {
	Header        *BlockHeader
	Transactions  []*Transaction
	ValidatorSigs [][]byte // Signatures from all validator nodes
}

// BlockHeader represents block header
type BlockHeader struct {
	Version       uint16
	Height        uint64
	PrevBlockHash []byte // Hash of previous block
	MerkleRoot    []byte // Merkle root of transactions
	Timestamp     uint64
	ValidatorID   p2p.PeerID // ID of the validator node who created this block
	StateRoot     []byte     // Root of the state trie
	Signature     []byte     // Validator's signature
}

// Transaction represents a V6Coin transaction
type Transaction struct {
	Version   uint16
	Type      TxType
	From      net.IP // 128-bit IPv6 address
	To        net.IP // 128-bit IPv6 address
	Amount    uint64 // in nano-V6
	Fee       uint64 // Transaction fee
	Nonce     uint64
	Timestamp uint64
	Signature []byte // Ed25519 signature
	Data      []byte // Optional transaction data
}

// TxType represents transaction type
type TxType uint8

const (
	TxTypeOnline     TxType = 0x01 // Online transaction (embedded in IPv6 extension header)
	TxTypeOffline    TxType = 0x02 // Offline transaction (delayed settlement)
	TxTypeMigration  TxType = 0x03 // Asset migration
	TxTypeStake      TxType = 0x04 // Staking transaction
	TxTypeUnstake    TxType = 0x05 // Unstaking transaction
	TxTypeGovernance TxType = 0x06 // Governance proposal
	TxTypeVote       TxType = 0x07 // Governance voting
)

// NodeContribution represents node contribution metrics
type NodeContribution struct {
	NodeID        p2p.PeerID
	OnlineTime    time.Duration // Total online time
	LastOnline    time.Time
	PacketLoss    float64 // Packet loss rate (0.0 - 1.0)
	Forwarded     uint64  // Total forwarded bytes
	Score         float64 // Contribution score (0.0 - 1.0)
	RewardBalance uint64  // Pending reward balance
	mu            sync.RWMutex
}

// ValidatorSet represents the set of validator nodes
type ValidatorSet struct {
	Validators      []p2p.PeerID // Top N nodes by contribution score
	RoundRobinIndex int
	mu              sync.RWMutex
}

// PrefixMonitor monitors device count per IPv6 prefix
type PrefixMonitor struct {
	prefixCounts map[string]int // IPv6 /64 prefix -> device count
	mu           sync.RWMutex
}

// ConsensusEngine represents the consensus engine
type ConsensusEngine struct {
	chainHead     *Block
	pendingBlocks []*Block
	transactions  map[string]*Transaction // Hash -> Transaction
	contributions map[p2p.PeerID]*NodeContribution
	validators    *ValidatorSet
	prefixMonitor *PrefixMonitor
	currentHeight uint64
	mu            sync.RWMutex
}

// NewBlock creates a new block
func NewBlock(height uint64, prevHash []byte, txs []*Transaction) *Block {
	header := &BlockHeader{
		Version:       0x0100,
		Height:        height,
		PrevBlockHash: prevHash,
		Timestamp:     uint64(time.Now().Unix()),
		MerkleRoot:    ComputeMerkleRoot(txs),
	}

	return &Block{
		Header:       header,
		Transactions: txs,
	}
}

// ComputeMerkleRoot computes the Merkle root of transactions
func ComputeMerkleRoot(txs []*Transaction) []byte {
	if len(txs) == 0 {
		return make([]byte, 32)
	}

	hashes := make([][]byte, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}

	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}

		newHashes := make([][]byte, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			hash := sha256.Sum256(combined)
			newHashes[i/2] = hash[:]
		}
		hashes = newHashes
	}

	return hashes[0]
}

// Hash computes the hash of a transaction
func (tx *Transaction) Hash() []byte {
	data := make([]byte, 0, 2+1+16+16+8+8+8+8)
	data = binary.BigEndian.AppendUint16(data, tx.Version)
	data = append(data, byte(tx.Type))
	data = append(data, tx.From.To16()...)
	data = append(data, tx.To.To16()...)
	data = binary.BigEndian.AppendUint64(data, tx.Amount)
	data = binary.BigEndian.AppendUint64(data, tx.Fee)
	data = binary.BigEndian.AppendUint64(data, tx.Nonce)
	data = binary.BigEndian.AppendUint64(data, tx.Timestamp)

	if len(tx.Data) > 0 {
		data = append(data, tx.Data...)
	}

	hash := sha256.Sum256(data)
	return hash[:]
}

// Hash computes the hash of a block header
func (bh *BlockHeader) Hash() []byte {
	data := make([]byte, 0, 2+8+32+32+8+16+32+64)
	data = binary.BigEndian.AppendUint16(data, bh.Version)
	data = binary.BigEndian.AppendUint64(data, bh.Height)
	data = append(data, bh.PrevBlockHash...)
	data = append(data, bh.MerkleRoot...)
	data = binary.BigEndian.AppendUint64(data, bh.Timestamp)
	data = append(data, bh.ValidatorID...)
	data = append(data, bh.StateRoot...)

	if len(bh.Signature) > 0 {
		data = append(data, bh.Signature...)
	}

	hash := sha256.Sum256(data)
	return hash[:]
}

// Hash computes the hash of a block
func (b *Block) Hash() []byte {
	return b.Header.Hash()
}

// NewTransaction creates a new transaction
func NewTransaction(txType TxType, from, to net.IP, amount, fee, nonce uint64) *Transaction {
	return &Transaction{
		Version:   0x0100,
		Type:      txType,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     nonce,
		Timestamp: uint64(time.Now().Unix()),
		Signature: []byte{},
	}
}

// Sign signs a transaction with the private key
func (tx *Transaction) Sign(privateKey []byte) error {
	hash := tx.Hash()
	signature, err := crypto.Sign(privateKey, hash)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}
	tx.Signature = signature
	return nil
}

// Verify verifies the transaction signature
func (tx *Transaction) Verify(publicKey []byte) bool {
	hash := tx.Hash()
	return crypto.Verify(publicKey, hash, tx.Signature)
}

// NewNodeContribution creates a new node contribution record
func NewNodeContribution(nodeID p2p.PeerID) *NodeContribution {
	return &NodeContribution{
		NodeID:     nodeID,
		OnlineTime: 0,
		LastOnline: time.Now(),
		PacketLoss: 0.0,
		Forwarded:  0,
		Score:      0.0,
	}
}

// CalculateScore calculates the contribution score
func (nc *NodeContribution) CalculateScore() float64 {
	nc.mu.Lock()
	defer nc.mu.Unlock()

	// 1. Online time score (60% weight)
	onlineScore := math.Min(float64(nc.OnlineTime.Hours())/float64(OnlineWindow.Hours()), 1.0) * 0.6

	// 2. Packet loss score (10% weight) - inverse: lower loss = higher score
	lossScore := (1.0 - nc.PacketLoss) * 0.1

	// 3. Forwarded bytes score (30% weight) - normalized against average
	// TODO: Get average from network statistics
	forwardedScore := math.Min(float64(nc.Forwarded)/1e9, 1.0) * 0.3 // Normalize against 1GB

	nc.Score = onlineScore + lossScore + forwardedScore
	return nc.Score
}

// UpdateOnlineTime updates the node's online time
func (nc *NodeContribution) UpdateOnlineTime(duration time.Duration) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.OnlineTime += duration
	nc.LastOnline = time.Now()
	nc.CalculateScore()
}

// UpdatePacketLoss updates the packet loss rate
func (nc *NodeContribution) UpdatePacketLoss(lossRate float64) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.PacketLoss = math.Max(0.0, math.Min(1.0, lossRate))
	nc.CalculateScore()
}

// AddForwardedBytes adds to the forwarded bytes count
func (nc *NodeContribution) AddForwardedBytes(bytes uint64) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.Forwarded += bytes
	nc.CalculateScore()
}

// NewValidatorSet creates a new validator set
func NewValidatorSet() *ValidatorSet {
	return &ValidatorSet{
		Validators:      make([]p2p.PeerID, 0, ValidatorCount),
		RoundRobinIndex: 0,
		mu:              sync.RWMutex{},
	}
}

// UpdateValidators updates the validator set based on contribution scores
func (vs *ValidatorSet) UpdateValidators(contributions map[p2p.PeerID]*NodeContribution) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Sort nodes by contribution score
	type nodeScore struct {
		nodeID p2p.PeerID
		score  float64
	}

	scores := make([]nodeScore, 0, len(contributions))
	for nodeID, contribution := range contributions {
		scores = append(scores, nodeScore{
			nodeID: nodeID,
			score:  contribution.Score,
		})
	}

	// Sort by score descending
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Take top N validators
	maxValidators := min(ValidatorCount, len(scores))
	vs.Validators = make([]p2p.PeerID, maxValidators)
	for i := 0; i < maxValidators; i++ {
		vs.Validators[i] = scores[i].nodeID
	}
}

// GetCurrentValidator returns the current validator based on round-robin
func (vs *ValidatorSet) GetCurrentValidator(height uint64) p2p.PeerID {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	if len(vs.Validators) == 0 {
		return ""
	}

	index := int(height) % len(vs.Validators)
	return vs.Validators[index]
}

// IsValidator checks if a node is in the validator set
func (vs *ValidatorSet) IsValidator(nodeID p2p.PeerID) bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	for _, validatorID := range vs.Validators {
		if validatorID == nodeID {
			return true
		}
	}
	return false
}

// NewPrefixMonitor creates a new prefix monitor
func NewPrefixMonitor() *PrefixMonitor {
	return &PrefixMonitor{
		prefixCounts: make(map[string]int),
		mu:           sync.RWMutex{},
	}
}

// GetPrefix returns the /64 prefix of an IPv6 address
func GetPrefix(addr net.IP) string {
	if addr == nil || len(addr.To16()) < 8 {
		return ""
	}

	// Get first 64 bits (8 bytes)
	ipv6 := addr.To16()
	return hex.EncodeToString(ipv6[:8])
}

// RegisterPrefix registers a device under a prefix
func (pm *PrefixMonitor) RegisterPrefix(addr net.IP) {
	prefix := GetPrefix(addr)
	if prefix == "" {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.prefixCounts[prefix]++
}

// UnregisterPrefix unregisters a device from a prefix
func (pm *PrefixMonitor) UnregisterPrefix(addr net.IP) {
	prefix := GetPrefix(addr)
	if prefix == "" {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	count, exists := pm.prefixCounts[prefix]
	if exists {
		if count <= 1 {
			delete(pm.prefixCounts, prefix)
		} else {
			pm.prefixCounts[prefix]--
		}
	}
}

// CalculateRewardMultiplier calculates the reward multiplier based on prefix device count
func (pm *PrefixMonitor) CalculateRewardMultiplier(addr net.IP) float64 {
	prefix := GetPrefix(addr)
	if prefix == "" {
		return 1.0
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	count, exists := pm.prefixCounts[prefix]
	if !exists {
		return 1.0
	}

	// Revenue decay formula: R = R0 * e^(-n/10)
	multiplier := math.Exp(-float64(count) / RevenueDecayFactor)
	return math.Max(0.01, multiplier) // Minimum 1% reward
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine() *ConsensusEngine {
	return &ConsensusEngine{
		chainHead:     nil,
		pendingBlocks: make([]*Block, 0),
		transactions:  make(map[string]*Transaction),
		contributions: make(map[p2p.PeerID]*NodeContribution),
		validators:    NewValidatorSet(),
		prefixMonitor: NewPrefixMonitor(),
		currentHeight: 0,
		mu:            sync.RWMutex{},
	}
}

// AddTransaction adds a transaction to the pending pool
func (ce *ConsensusEngine) AddTransaction(tx *Transaction) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	// Validate transaction
	if err := ValidateTransaction(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	hash := hex.EncodeToString(tx.Hash())

	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Check for duplicates
	if _, exists := ce.transactions[hash]; exists {
		return errors.New("transaction already exists")
	}

	ce.transactions[hash] = tx
	return nil
}

// GetPendingTransactions returns all pending transactions
func (ce *ConsensusEngine) GetPendingTransactions() []*Transaction {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	txs := make([]*Transaction, 0, len(ce.transactions))
	for _, tx := range ce.transactions {
		txs = append(txs, tx)
	}
	return txs
}

// UpdateContribution updates a node's contribution
func (ce *ConsensusEngine) UpdateContribution(nodeID p2p.PeerID, updateFunc func(*NodeContribution)) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if _, exists := ce.contributions[nodeID]; !exists {
		ce.contributions[nodeID] = NewNodeContribution(nodeID)
	}

	updateFunc(ce.contributions[nodeID])

	// Update validator set periodically
	ce.validators.UpdateValidators(ce.contributions)
}

// GetCurrentBlockHeight returns the current block height
func (ce *ConsensusEngine) GetCurrentBlockHeight() uint64 {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.currentHeight
}

// GetChainHead returns the current chain head
func (ce *ConsensusEngine) GetChainHead() *Block {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.chainHead
}

// SetChainHead sets a new chain head
func (ce *ConsensusEngine) SetChainHead(block *Block) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	if block != nil && (ce.chainHead == nil || block.Header.Height > ce.chainHead.Header.Height) {
		ce.chainHead = block
		ce.currentHeight = block.Header.Height
	}
}

// CalculateBlockReward calculates the block reward
func CalculateBlockReward(height uint64) uint64 {
	// Initial reward: 100 V6 per block
	reward := uint64(InitialBlockReward * 1e9) // Convert to nano-V6

	// Halve every 4 years (assuming 10 second blocks, 4 years = ~12.6M blocks)
	halvingInterval := uint64((HalvingYears * 365 * 24 * 3600) / 10)
	halvings := height / halvingInterval

	for i := uint64(0); i < halvings; i++ {
		reward /= 2
	}

	return reward
}

// ValidateTransaction validates a transaction
func ValidateTransaction(tx *Transaction) error {
	if tx == nil {
		return errors.New("transaction cannot be nil")
	}

	// Check version
	if tx.Version != 0x0100 {
		return fmt.Errorf("invalid transaction version: %d", tx.Version)
	}

	// Check addresses
	if tx.From == nil || len(tx.From.To16()) != 16 {
		return errors.New("invalid 'from' address")
	}

	if tx.To == nil || len(tx.To.To16()) != 16 {
		return errors.New("invalid 'to' address")
	}

	// Check amount
	if tx.Amount == 0 {
		return errors.New("transaction amount cannot be zero")
	}

	// Check signature
	if len(tx.Signature) == 0 {
		return errors.New("transaction signature is missing")
	}

	if len(tx.Signature) != 64 {
		return fmt.Errorf("invalid signature length: %d (expected 64)", len(tx.Signature))
	}

	// Check timestamp (offline transactions valid for 7 days)
	if tx.Type == TxTypeOffline {
		txTime := time.Unix(int64(tx.Timestamp), 0)
		if time.Since(txTime) > 7*24*time.Hour {
			return errors.New("offline transaction expired (valid for 7 days)")
		}
	}

	return nil
}

// ValidateBlock validates a block
func ValidateBlock(block *Block, prevBlock *Block) error {
	if block == nil {
		return errors.New("block cannot be nil")
	}

	if block.Header == nil {
		return errors.New("block header cannot be nil")
	}

	// Check version
	if block.Header.Version != 0x0100 {
		return fmt.Errorf("invalid block version: %d", block.Header.Version)
	}

	// Check previous block hash
	if prevBlock != nil {
		expectedHeight := prevBlock.Header.Height + 1
		if block.Header.Height != expectedHeight {
			return fmt.Errorf("invalid block height: expected %d, got %d", expectedHeight, block.Header.Height)
		}

		prevHash := prevBlock.Hash()
		if !equalBytes(block.Header.PrevBlockHash, prevHash) {
			return fmt.Errorf("invalid previous block hash")
		}
	}

	// Validate all transactions
	for _, tx := range block.Transactions {
		if err := ValidateTransaction(tx); err != nil {
			return fmt.Errorf("invalid transaction in block: %w", err)
		}
	}

	// Check Merkle root
	computedMerkleRoot := ComputeMerkleRoot(block.Transactions)
	if !equalBytes(block.Header.MerkleRoot, computedMerkleRoot) {
		return errors.New("invalid merkle root")
	}

	return nil
}

// equalBytes compares two byte slices
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BlockHashToString converts block hash to hex string
func BlockHashToString(hash []byte) string {
	if hash == nil {
		return ""
	}
	return hex.EncodeToString(hash)
}

// TxHashToString converts transaction hash to hex string
func TxHashToString(hash []byte) string {
	if hash == nil {
		return ""
	}
	return hex.EncodeToString(hash)
}
