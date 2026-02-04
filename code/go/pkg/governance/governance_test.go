package governance

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/stake"
	"github.com/jinba225/v6coin-protocol/pkg/state"
	"github.com/stretchr/testify/assert"
)

// MockConsensusEngine 模拟共识引擎
type MockConsensusEngine struct{}

func (m *MockConsensusEngine) GetNodeContribution(validatorID []byte) (*NodeContribution, error) {
	return &NodeContribution{
		OnlineTime: 3600 * 24 * 45, // 45 天
		Forwarded:  1000000000,
		PacketLoss: 0.05,
		Score:       85.0,
	}, nil
}

// MockStateDB 模拟状态数据库
type MockStateDB struct{}

func (m *MockStateDB) GetBalance(address []byte) (uint64, error) {
	return 1000000000, nil // 1000 V6
}

func (m *MockStateDB) GetNonce(address []byte) (uint64, error) {
	return 0, nil
}

func (m *MockStateDB) GetAccount(address []byte) (*state.Account, error) {
	return &state.Account{
		Nonce:   0,
		Balance: 1000000000,
	}, nil
}

func (m *MockStateDB) SetAccount(address []byte, account *state.Account) error {
	return nil
}

func (m *MockStateDB) HasAccount(address []byte) (bool, error) {
	return true, nil
}

func (m *MockStateDB) CurrentRoot() []byte {
	root := make([]byte, 32)
	return root
}

func (m *MockStateDB) Commit() ([]byte, error) {
	root := make([]byte, 32)
	return root, nil
}

func (m *MockStateDB) Close() error {
	return nil
}

func createTestVoterID(id byte) []byte {
	voterID := make([]byte, 16)
	voterID[15] = id
	return voterID
}

func createTestProposalID() []byte {
	proposalID := make([]byte, 32)
	for i := range proposalID {
		proposalID[i] = byte(i)
	}
	return proposalID
}

func TestNewProposal(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposerID := createTestVoterID(1)

	proposal := &Proposal{
		ProposerID:  proposerID,
		Type:       ProposalTypeParameterChange,
		Title:      "Test Proposal",
		Description: "This is a test proposal",
		Deposit:    MinProposalDeposit,
	}

	err := manager.CreateProposal(proposal)
	assert.NoError(t, err)

	// 验证提案已创建
	assert.NotNil(t, proposal.ID)
	assert.Len(t, proposal.ID, 32)
	assert.Equal(t, ProposalStatusPending, proposal.Status)

	// 可以通过 ID 获取提案
	retrieved, _ := manager.GetProposal(proposal.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, proposal.Title, retrieved.Title)
}

func TestNewProposalInvalidProposer(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposal := &Proposal{
		ProposerID:  []byte{}, // 无效 ID
		Type:       ProposalTypeParameterChange,
		Title:      "Test Proposal",
		Description: "This is a test proposal",
		Deposit:    MinProposalDeposit,
	}

	err := manager.CreateProposal(proposal)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProposer, err)
}

func TestNewProposalInsufficientDeposit(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposerID := createTestVoterID(1)

	proposal := &Proposal{
		ProposerID:  proposerID,
		Type:       ProposalTypeParameterChange,
		Title:      "Test Proposal",
		Description: "This is a test proposal",
		Deposit:    MinProposalDeposit - 1, // 低于最小押金
	}

	err := manager.CreateProposal(proposal)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientDeposit, err)
}

func TestVote(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	// 创建提案
	proposal := &Proposal{
		ProposerID:  createTestVoterID(1),
		Type:        ProposalTypeParameterChange,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     MinProposalDeposit,
	}

	manager.CreateProposal(proposal)

	// 投票
	voterID := createTestVoterID(2)
	totalSupply := uint64(1000000000) // 简化：假设总供应量

	weight, _ := weightCalc.CalculateVoteWeight(voterID, totalSupply)

	err := manager.Vote(proposal.ID, voterID, VoteDecisionYes, weight)
	assert.NoError(t, err)

	// 验证投票已记录
	yes, no, abstain := manager.CountVotes(proposal.ID)
	assert.Greater(t, yes, uint64(0))
	assert.Equal(t, uint64(0), no)
	assert.Equal(t, uint64(0), abstain)
}

func TestVoteInvalidVoter(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposal := &Proposal{
		ProposerID:  createTestVoterID(1),
		Type:        ProposalTypeParameterChange,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     MinProposalDeposit,
	}

	manager.CreateProposal(proposal)

	// 使用无效投票人 ID
	err := manager.Vote(proposal.ID, []byte{}, VoteDecisionYes, 100)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProposer, err)
}

func TestVoteTwice(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposal := &Proposal{
		ProposerID:  createTestVoterID(1),
		Type:        ProposalTypeParameterChange,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     MinProposalDeposit,
	}

	manager.CreateProposal(proposal)

	// 第一次投票
	voterID := createTestVoterID(2)
	totalSupply := uint64(1000000000)
	weight, _ := weightCalc.CalculateVoteWeight(voterID, totalSupply)

	err := manager.Vote(proposal.ID, voterID, VoteDecisionYes, weight)
	assert.NoError(t, err)

	// 第二次投票（应该失败）
	err = manager.Vote(proposal.ID, voterID, VoteDecisionNo, weight)
	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyVoted, err)
}

func TestGetProposalVotes(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposal := &Proposal{
		ProposerID:  createTestVoterID(1),
		Type:        ProposalTypeParameterChange,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     MinProposalDeposit,
	}

	manager.CreateProposal(proposal)

	// 添加多个投票
	totalSupply := uint64(1000000000)
	for i := 2; i <= 4; i++ {
		voterID := createTestVoterID(byte(i))
		weight, _ := weightCalc.CalculateVoteWeight(voterID, totalSupply)

		decision := VoteDecisionYes
		if i%2 == 0 {
			decision = VoteDecisionNo
		}

		manager.Vote(proposal.ID, voterID, decision, weight)
	}

	// 获取投票
	votes, _ := manager.GetProposalVotes(proposal.ID)
	assert.Len(t, votes, 3)
}

func TestCheckProposalResult(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	proposal := &Proposal{
		ProposerID:     createTestVoterID(1),
		Type:           ProposalTypeParameterChange,
		Title:          "Test Proposal",
		Description:    "This is a test proposal",
		Deposit:        MinProposalDeposit,
	}

	manager.CreateProposal(proposal)

	// 手动设置 VotingEndTime 为过去时间（模拟投票期已结束）
	proposal.VotingEndTime = time.Now().Add(-1 * time.Hour)

	// 检查结果（没有投票，应该被拒绝）
	status, err := manager.CheckProposalResult(proposal.ID)
	assert.NoError(t, err)
	assert.Equal(t, ProposalStatusRejected, status)
}

func TestProcessProposals(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	// 创建提案
	proposal := &Proposal{
		ProposerID:    createTestVoterID(1),
		Type:          ProposalTypeParameterChange,
		Title:         "Test Proposal",
		Description:   "This is a test proposal",
		Deposit:       MinProposalDeposit,
		VotingEndTime:  time.Now().Add(-1 * time.Hour), // 投票期已结束
	}

	manager.CreateProposal(proposal)

	// 处理提案
	processed := manager.ProcessProposals()
	assert.GreaterOrEqual(t, processed, 0)
}

func TestGetAllProposals(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	// 创建多个提案
	for i := 1; i <= 3; i++ {
		proposal := &Proposal{
			ProposerID:  createTestVoterID(byte(i)),
			Type:        ProposalTypeParameterChange,
			Title:       fmt.Sprintf("Test Proposal %d", i),
			Description: fmt.Sprintf("This is test proposal %d", i),
			Deposit:     MinProposalDeposit,
		}
		manager.CreateProposal(proposal)
	}

	// 获取所有提案
	allProposals := manager.GetAllProposals()
	assert.Len(t, allProposals, 3)
}

func TestGetActiveProposals(t *testing.T) {
	stateDB := &MockStateDB{}
	executor := NewSimpleProposalExecutor()
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))
	manager := NewMemoryVoteManager(executor, weightCalc, stateDB)

	// 创建提案（默认是待投票状态）
	proposal := &Proposal{
		ProposerID:  createTestVoterID(1),
		Type:        ProposalTypeParameterChange,
		Title:       "Test Proposal",
		Description: "This is a test proposal",
		Deposit:     MinProposalDeposit,
	}

	manager.CreateProposal(proposal)

	// 获取活跃提案
	activeProposals := manager.GetActiveProposals()
	assert.Len(t, activeProposals, 1)
	assert.Equal(t, ProposalStatusPending, activeProposals[0].Status)
}

func TestExecuteParameterChange(t *testing.T) {
	executor := NewSimpleProposalExecutor()

	// 创建参数更改提案数据
	paramChange := ParameterChangeProposal{
		Parameter: "max_block_size",
		Value:     "2000000",
		Reason:    "Increase block size for better throughput",
	}

	data, _ := json.Marshal(paramChange)

	proposal := &Proposal{
		ID:          createTestProposalID(),
		Type:        ProposalTypeParameterChange,
		Title:       "Increase Block Size",
		Description: "Increase max block size from 1M to 2M",
		Data:        data,
	}

	err := executor.ExecuteProposal(proposal)
	assert.NoError(t, err)
	assert.Contains(t, proposal.Result, "Parameter")
	assert.Contains(t, proposal.Result, "max_block_size")
	assert.Contains(t, proposal.Result, "2000000")
}

func TestValidateProposal(t *testing.T) {
	executor := NewSimpleProposalExecutor()

	// 测试无效提案（空标题）
	proposal := &Proposal{
		Type:        ProposalTypeParameterChange,
		Title:       "", // 无效
		Description: "Description",
	}

	err := executor.ValidateProposal(proposal)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}

func TestVoteWeightCalculation(t *testing.T) {
	stateDB := &MockStateDB{}
	weightCalc := NewSimpleVoteWeightCalculator(&MockConsensusEngine{}, stake.NewMemoryStakePool(stateDB, nil))

	voterID := createTestVoterID(1)
	totalSupply := uint64(1000000000000) // 1000 V6

	// 计算投票权重
	weight, err := weightCalc.CalculateVoteWeight(voterID, totalSupply)
	assert.NoError(t, err)
	assert.Greater(t, weight, uint64(0))

	// 验证权重不超过 5%
	maxWeight := totalSupply / 20
	assert.LessOrEqual(t, weight, maxWeight)
}
