package blockchain

import (
	"testing"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/jinba225/v6coin-protocol/pkg/state"
	"github.com/stretchr/testify/assert"
)

// mockValidator implements Validator interface for testing
type mockValidator struct {
	validateBlockFunc    func(block *consensus.Block, parent *consensus.Block) error
	validateTxFunc       func(tx *consensus.Transaction) error
	isValidBlockHashFunc func(hash []byte) bool
}

func (m *mockValidator) ValidateBlock(block *consensus.Block, parent *consensus.Block) error {
	if m.validateBlockFunc != nil {
		return m.validateBlockFunc(block, parent)
	}
	return nil
}

func (m *mockValidator) ValidateTransaction(tx *consensus.Transaction) error {
	if m.validateTxFunc != nil {
		return m.validateTxFunc(tx)
	}
	return nil
}

func (m *mockValidator) IsValidBlockHash(hash []byte) bool {
	if m.isValidBlockHashFunc != nil {
		return m.isValidBlockHashFunc(hash)
	}
	return true
}

// mockRewardDistributor implements RewardDistributor interface for testing
type mockRewardDistributor struct {
	distributeFunc func(block *consensus.Block, stateDB state.StateDB) error
}

func (m *mockRewardDistributor) DistributeBlockReward(block *consensus.Block, stateDB state.StateDB) error {
	if m.distributeFunc != nil {
		return m.distributeFunc(block, stateDB)
	}
	return nil
}

func createTestGenesisBlock() *consensus.Block {
	return &consensus.Block{
		Header: &consensus.BlockHeader{
			Version:       1,
			PrevBlockHash: []byte{}, // Empty for genesis
			MerkleRoot:    []byte("genesis-root"),
			Timestamp:     uint64(time.Now().Unix()),
			Height:        0,
			StateRoot:     []byte("genesis-state"),
		},
		Transactions: []*consensus.Transaction{},
	}
}

func createTestBlock(height uint64, prevHash []byte) *consensus.Block {
	return &consensus.Block{
		Header: &consensus.BlockHeader{
			Version:       1,
			PrevBlockHash: prevHash,
			MerkleRoot:    []byte("test-root"),
			Timestamp:     uint64(time.Now().Unix()),
			Height:        height,
			StateRoot:     []byte("test-state"),
		},
		Transactions: []*consensus.Transaction{},
	}
}

func TestNewBlockChain(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)
	assert.NotNil(t, bc)
	assert.Equal(t, genesis, bc.genesisBlock)
	assert.Equal(t, genesis, bc.chainHead)
	assert.Equal(t, uint64(0), bc.chainHead.Header.Height)
}

func TestNewBlockChainNilGenesis(t *testing.T) {
	config := &BlockChainConfig{
		GenesisBlock: nil,
	}

	_, err := NewBlockChain(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "genesis block required")
}

func TestAddBlock(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{
		validateBlockFunc: func(block *consensus.Block, parent *consensus.Block) error {
			// 验证区块高度是否正确
			if block.Header.Height != parent.Header.Height+1 {
				return assert.AnError
			}
			return nil
		},
	}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 添加区块 1
	block1 := createTestBlock(1, genesis.Hash())
	err = bc.AddBlock(block1)
	assert.NoError(t, err)
	assert.Equal(t, block1, bc.chainHead)
	assert.Equal(t, uint64(1), bc.chainHead.Header.Height)

	// 添加区块 2
	block2 := createTestBlock(2, block1.Hash())
	err = bc.AddBlock(block2)
	assert.NoError(t, err)
	assert.Equal(t, block2, bc.chainHead)
	assert.Equal(t, uint64(2), bc.chainHead.Header.Height)
}

func TestAddBlockDuplicate(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 添加区块
	block1 := createTestBlock(1, genesis.Hash())
	err = bc.AddBlock(block1)
	assert.NoError(t, err)

	// 尝试添加相同的区块
	err = bc.AddBlock(block1)
	assert.Error(t, err)
	assert.Equal(t, ErrDuplicateBlock, err)
}

func TestAddBlockInvalidParent(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 创建一个父哈希不存在的区块
	invalidParentHash := []byte("invalid-parent-hash")
	block := createTestBlock(1, invalidParentHash)

	err = bc.AddBlock(block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parent not found")
}

func TestAddBlockValidationFailure(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{
		validateBlockFunc: func(block *consensus.Block, parent *consensus.Block) error {
			return assert.AnError
		},
	}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	block1 := createTestBlock(1, genesis.Hash())
	err = bc.AddBlock(block1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block validation failed")
}

func TestGetBlock(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 测试获取创世区块
	block := bc.GetBlock(genesis.Hash())
	assert.NotNil(t, block)
	assert.Equal(t, genesis, block)

	// 测试获取不存在的区块
	nonExistentHash := []byte("non-existent")
	block = bc.GetBlock(nonExistentHash)
	assert.Nil(t, block)
}

func TestGetBlockByHeight(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 添加一些区块
	block1 := createTestBlock(1, genesis.Hash())
	bc.AddBlock(block1)

	block2 := createTestBlock(2, block1.Hash())
	bc.AddBlock(block2)

	// 测试获取不同高度的区块
	blocks := bc.GetBlockByHeight(0)
	assert.Len(t, blocks, 1)
	assert.Equal(t, genesis, blocks[0])

	blocks = bc.GetBlockByHeight(1)
	assert.Len(t, blocks, 1)
	assert.Equal(t, block1, blocks[0])

	blocks = bc.GetBlockByHeight(2)
	assert.Len(t, blocks, 1)
	assert.Equal(t, block2, blocks[0])

	// 测试获取不存在的高度
	blocks = bc.GetBlockByHeight(999)
	assert.Nil(t, blocks)
}

func TestGetChainHead(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 初始链头应该是创世区块
	head := bc.GetChainHead()
	assert.Equal(t, genesis, head)

	// 添加区块后，链头应该更新
	block1 := createTestBlock(1, genesis.Hash())
	bc.AddBlock(block1)

	head = bc.GetChainHead()
	assert.Equal(t, block1, head)
	assert.Equal(t, uint64(1), head.Header.Height)
}

func TestGetCurrentHeight(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 初始高度应该是 0
	height := bc.GetCurrentHeight()
	assert.Equal(t, uint64(0), height)

	// 添加区块后，高度应该增加
	block1 := createTestBlock(1, genesis.Hash())
	bc.AddBlock(block1)

	height = bc.GetCurrentHeight()
	assert.Equal(t, uint64(1), height)

	block2 := createTestBlock(2, block1.Hash())
	bc.AddBlock(block2)

	height = bc.GetCurrentHeight()
	assert.Equal(t, uint64(2), height)
}

func TestHasBlock(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 测试创世区块存在
	block := bc.GetBlock(genesis.Hash())
	assert.NotNil(t, block)

	// 测试不存在的区块
	nonExistentHash := []byte("non-existent")
	block = bc.GetBlock(nonExistentHash)
	assert.Nil(t, block)

	// 添加区块后测试
	block1 := createTestBlock(1, genesis.Hash())
	bc.AddBlock(block1)

	block = bc.GetBlock(block1.Hash())
	assert.NotNil(t, block)
}

func TestCalculateChainDifficulty(t *testing.T) {
	genesis := createTestGenesisBlock()
	stateDB := state.NewMemoryStateDB()
	validator := &mockValidator{}
	rewardDist := &mockRewardDistributor{}

	config := &BlockChainConfig{
		GenesisBlock: genesis,
		StateDB:      stateDB,
		Validator:    validator,
		RewardDist:   rewardDist,
	}

	bc, err := NewBlockChain(config)
	assert.NoError(t, err)

	// 添加几个区块
	block1 := createTestBlock(1, genesis.Hash())
	bc.AddBlock(block1)

	block2 := createTestBlock(2, block1.Hash())
	bc.AddBlock(block2)

	block3 := createTestBlock(3, block2.Hash())
	bc.AddBlock(block3)

	// 计算从创世区块到区块 3 的难度
	difficulty := bc.calculateChainDifficulty(block3, 0)
	assert.Greater(t, difficulty, uint64(0))
	// 难度应该是区块数量
	assert.Equal(t, uint64(4), difficulty) // genesis + 3 blocks
}
