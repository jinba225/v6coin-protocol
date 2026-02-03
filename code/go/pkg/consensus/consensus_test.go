package consensus

import (
	"net"
	"testing"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/p2p"
)

func TestNewBlock(t *testing.T) {
	height := uint64(1)
	prevHash := []byte{0x01, 0x02, 0x03}
	txs := []*Transaction{
		NewTransaction(TxTypeOnline, net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::2"), 1000, 10, 1),
	}

	block := NewBlock(height, prevHash, txs)

	if block.Header.Height != height {
		t.Errorf("Expected height %d, got %d", height, block.Header.Height)
	}

	if !equalBytes(block.Header.PrevBlockHash, prevHash) {
		t.Errorf("Expected prev hash %v, got %v", prevHash, block.Header.PrevBlockHash)
	}

	if len(block.Transactions) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(block.Transactions))
	}
}

func TestComputeMerkleRoot(t *testing.T) {
	txs := []*Transaction{
		NewTransaction(TxTypeOnline, net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::2"), 1000, 10, 1),
		NewTransaction(TxTypeOnline, net.ParseIP("2001:db8::3"), net.ParseIP("2001:db8::4"), 2000, 20, 2),
	}

	root := ComputeMerkleRoot(txs)

	if len(root) != 32 {
		t.Errorf("Expected merkle root length 32, got %d", len(root))
	}

	emptyRoot := ComputeMerkleRoot([]*Transaction{})
	if len(emptyRoot) != 32 {
		t.Errorf("Expected empty merkle root length 32, got %d", len(emptyRoot))
	}
}

func TestTransactionHash(t *testing.T) {
	tx := NewTransaction(TxTypeOnline, net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::2"), 1000, 10, 1)

	hash := tx.Hash()

	if len(hash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}
}

func TestBlockHeaderHash(t *testing.T) {
	block := NewBlock(1, []byte{0x01, 0x02, 0x03}, []*Transaction{})

	hash := block.Header.Hash()

	if len(hash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}
}

func TestNewTransaction(t *testing.T) {
	from := net.ParseIP("2001:db8::1")
	to := net.ParseIP("2001:db8::2")
	amount := uint64(1000)
	fee := uint64(10)
	nonce := uint64(1)

	tx := NewTransaction(TxTypeOnline, from, to, amount, fee, nonce)

	if !tx.From.Equal(from) {
		t.Errorf("Expected from address %v, got %v", from, tx.From)
	}

	if !tx.To.Equal(to) {
		t.Errorf("Expected to address %v, got %v", to, tx.To)
	}

	if tx.Amount != amount {
		t.Errorf("Expected amount %d, got %d", amount, tx.Amount)
	}

	if tx.Fee != fee {
		t.Errorf("Expected fee %d, got %d", fee, tx.Fee)
	}

	if tx.Nonce != nonce {
		t.Errorf("Expected nonce %d, got %d", nonce, tx.Nonce)
	}

	if tx.Version != 0x0100 {
		t.Errorf("Expected version 0x0100, got %d", tx.Version)
	}
}

func TestValidateTransaction(t *testing.T) {
	tests := []struct {
		name    string
		tx      *Transaction
		wantErr bool
	}{
		{
			name:    "nil transaction",
			tx:      nil,
			wantErr: true,
		},
		{
			name: "invalid version",
			tx: &Transaction{
				Version: 0x0000,
				From:    net.ParseIP("2001:db8::1"),
				To:      net.ParseIP("2001:db8::2"),
				Amount:  1000,
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			tx: &Transaction{
				Version: 0x0100,
				From:    net.ParseIP("2001:db8::1"),
				To:      net.ParseIP("2001:db8::2"),
				Amount:  0,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateTransaction(test.tx)
			if (err != nil) != test.wantErr {
				t.Errorf("ValidateTransaction() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestValidateBlock(t *testing.T) {
	prevBlock := NewBlock(0, make([]byte, 32), []*Transaction{})
	block := NewBlock(1, prevBlock.Hash(), []*Transaction{})

	err := ValidateBlock(block, prevBlock)
	if err != nil {
		t.Errorf("Expected valid block, got error: %v", err)
	}

	err = ValidateBlock(nil, prevBlock)
	if err == nil {
		t.Errorf("Expected error for nil block")
	}

	err = ValidateBlock(&Block{Header: nil}, prevBlock)
	if err == nil {
		t.Errorf("Expected error for nil header")
	}
}

func TestNodeContribution(t *testing.T) {
	nodeID := p2p.PeerID("test-node-1")
	nc := NewNodeContribution(nodeID)

	if nc.NodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, nc.NodeID)
	}

	if nc.OnlineTime != 0 {
		t.Errorf("Expected initial online time 0, got %v", nc.OnlineTime)
	}

	if nc.Score != 0.0 {
		t.Errorf("Expected initial score 0.0, got %f", nc.Score)
	}
}

func TestCalculateContributionScore(t *testing.T) {
	nodeID := p2p.PeerID("test-node-1")
	nc := NewNodeContribution(nodeID)

	nc.UpdateOnlineTime(45 * 24 * time.Hour)
	nc.UpdatePacketLoss(0.01)
	nc.AddForwardedBytes(500 * 1024 * 1024)

	score := nc.CalculateScore()

	if score <= 0.0 || score > 1.0 {
		t.Errorf("Expected score in range [0.0, 1.0], got %f", score)
	}
}

func TestValidatorSet(t *testing.T) {
	vs := NewValidatorSet()

	if vs == nil {
		t.Errorf("Expected validator set to be created")
	}

	if len(vs.Validators) != 0 {
		t.Errorf("Expected empty validator set, got %d validators", len(vs.Validators))
	}
}

func TestGetCurrentValidator(t *testing.T) {
	vs := NewValidatorSet()
	vs.Validators = []p2p.PeerID{
		p2p.PeerID("node-A"),
		p2p.PeerID("node-B"),
		p2p.PeerID("node-C"),
	}

	validator1 := vs.GetCurrentValidator(0)
	validator2 := vs.GetCurrentValidator(1)
	validator3 := vs.GetCurrentValidator(2)

	if validator1 != p2p.PeerID("node-A") {
		t.Errorf("Expected node-A, got %s", validator1)
	}

	if validator2 != p2p.PeerID("node-B") {
		t.Errorf("Expected node-B, got %s", validator2)
	}

	if validator3 != p2p.PeerID("node-C") {
		t.Errorf("Expected node-C, got %s", validator3)
	}
}

func TestPrefixMonitor(t *testing.T) {
	pm := NewPrefixMonitor()

	if pm == nil {
		t.Errorf("Expected prefix monitor to be created")
	}

	addr1 := net.ParseIP("2001:db8::1")
	addr2 := net.ParseIP("2001:db8::2")
	addr3 := net.ParseIP("2001:db9::1")

	pm.RegisterPrefix(addr1)
	pm.RegisterPrefix(addr2)
	pm.RegisterPrefix(addr3)

	multiplier1 := pm.CalculateRewardMultiplier(addr1)
	multiplier2 := pm.CalculateRewardMultiplier(addr2)
	multiplier3 := pm.CalculateRewardMultiplier(addr3)

	if multiplier1 != multiplier2 {
		t.Errorf("Expected same multiplier for same prefix")
	}

	if multiplier3 <= multiplier1 {
		t.Errorf("Expected higher multiplier for different prefix")
	}
}

func TestCalculateBlockReward(t *testing.T) {
	initialReward := CalculateBlockReward(0)
	if initialReward != uint64(100*1000000000) {
		t.Errorf("Expected initial reward %d, got %d", 100*1000000000, initialReward)
	}

	// 计算实际的减半间隔：4年 = (4 * 365 * 24 * 3600) / 10 = 12,614,400 个区块
	halvingInterval := uint64((4 * 365 * 24 * 3600) / 10)

	// 测试第一个减半点
	halvedReward := CalculateBlockReward(halvingInterval)
	if halvedReward*2 != initialReward {
		t.Errorf("Expected halved reward to be half of initial, got %d, expected %d", halvedReward, initialReward/2)
	}
}

func TestConsensusEngine(t *testing.T) {
	ce := NewConsensusEngine()

	if ce == nil {
		t.Errorf("Expected consensus engine to be created")
	}

	if ce.GetCurrentBlockHeight() != 0 {
		t.Errorf("Expected initial height 0, got %d", ce.GetCurrentBlockHeight())
	}

	if ce.GetChainHead() != nil {
		t.Errorf("Expected initial chain head nil, got %v", ce.GetChainHead())
	}
}

func TestAddTransaction(t *testing.T) {
	ce := NewConsensusEngine()

	// 创建交易并添加签名（虽然测试中可以跳过签名验证）
	fromAddr := net.ParseIP("2001:db8::1")
	toAddr := net.ParseIP("2001:db8::2")

	tx := NewTransaction(TxTypeOnline, fromAddr, toAddr, 1000, 10, 1)

	// 为交易添加签名（使用空签名字节以通过验证）
	tx.Signature = make([]byte, 64) // 64 字节的空签名

	err := ce.AddTransaction(tx)
	if err != nil {
		t.Errorf("Expected no error adding transaction, got: %v", err)
	}

	pendingTxs := ce.GetPendingTransactions()
	if len(pendingTxs) != 1 {
		t.Errorf("Expected 1 pending transaction, got %d", len(pendingTxs))
	}

	err = ce.AddTransaction(tx)
	if err == nil {
		t.Errorf("Expected error adding duplicate transaction")
	}
}

func TestBlockHashToString(t *testing.T) {
	hash := []byte{0x01, 0x02, 0x03, 0x04}
	hashStr := BlockHashToString(hash)

	if hashStr != "01020304" {
		t.Errorf("Expected '01020304', got '%s'", hashStr)
	}
}

func TestTxHashToString(t *testing.T) {
	hash := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	hashStr := TxHashToString(hash)

	if hashStr != "aabbccdd" {
		t.Errorf("Expected 'aabbccdd', got '%s'", hashStr)
	}
}
