package governance

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrProposalNotFound      = errors.New("proposal not found")
	ErrInvalidProposalID     = errors.New("invalid proposal ID")
	ErrProposalAlreadyExists = errors.New("proposal already exists")
	ErrInvalidProposer       = errors.New("invalid proposer ID")
	ErrInsufficientDeposit   = errors.New("insufficient deposit")
	ErrInvalidProposalType   = errors.New("invalid proposal type")
	ErrVotingPeriodEnded     = errors.New("voting period has ended")
	ErrAlreadyVoted          = errors.New("already voted")
	ErrProposalNotPending     = errors.New("proposal is not pending")
)

// MemoryVoteManager 内存投票管理器实现
type MemoryVoteManager struct {
	proposals    map[string]*Proposal // proposal ID -> proposal
	votes        map[string][]*Vote   // proposal ID -> votes
	mu           sync.RWMutex
	executor     ProposalExecutor
	weightCalc   VoteWeightCalculator
	stateDB      StateDB
}

// StateDB 状态数据库接口（简化版）
type StateDB interface {
	GetBalance(address []byte) (uint64, error)
	GetNonce(address []byte) (uint64, error)
}

// NewMemoryVoteManager 创建内存投票管理器
func NewMemoryVoteManager(executor ProposalExecutor, weightCalc VoteWeightCalculator, stateDB StateDB) *MemoryVoteManager {
	return &MemoryVoteManager{
		proposals:  make(map[string]*Proposal),
		votes:      make(map[string][]*Vote),
		executor:   executor,
		weightCalc: weightCalc,
		stateDB:    stateDB,
	}
}

// CreateProposal 创建提案
func (m *MemoryVoteManager) CreateProposal(proposal *Proposal) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证提案人 ID
	if len(proposal.ProposerID) != 16 {
		return ErrInvalidProposer
	}

	// 验证提案类型
	if proposal.Type < 0 || proposal.Type > 3 {
		return ErrInvalidProposalType
	}

	// 验证押金
	if proposal.Deposit < MinProposalDeposit {
		return ErrInsufficientDeposit
	}

	// 生成提案 ID
	proposalID := m.generateProposalID(proposal)
	proposal.ID = proposalID

	// 检查提案是否已存在
	if _, exists := m.proposals[string(proposalID)]; exists {
		return ErrProposalAlreadyExists
	}

	// 设置初始状态
	proposal.Status = ProposalStatusPending
	proposal.CreatedTime = time.Now()
	proposal.VotingEndTime = proposal.CreatedTime.Add(VotingPeriod)

	// 存储提案
	m.proposals[string(proposalID)] = proposal
	m.votes[string(proposalID)] = make([]*Vote, 0)

	return nil
}

// Vote 投票
func (m *MemoryVoteManager) Vote(proposalID, voterID []byte, decision VoteDecision, weight uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(proposalID) != 32 {
		return ErrInvalidProposalID
	}

	if len(voterID) != 16 {
		return ErrInvalidProposer
	}

	// 获取提案
	key := string(proposalID)
	proposal, exists := m.proposals[key]
	if !exists {
		return ErrProposalNotFound
	}

	// 检查提案状态
	if proposal.Status != ProposalStatusPending {
		return ErrProposalNotPending
	}

	// 检查投票期是否结束
	if time.Now().After(proposal.VotingEndTime) {
		return ErrVotingPeriodEnded
	}

	// 检查是否已投票
	for _, vote := range m.votes[key] {
		if string(vote.VoterID) == string(voterID) {
			return ErrAlreadyVoted
		}
	}

	// 创建投票
	vote := &Vote{
		ProposalID: proposalID,
		VoterID:    voterID,
		Decision:   decision,
		VoteTime:   time.Now(),
		VoteWeight: weight,
	}

	// 记录投票
	m.votes[key] = append(m.votes[key], vote)

	// 更新提案计票
	switch decision {
	case VoteDecisionYes:
		proposal.YesVotes += weight
	case VoteDecisionNo:
		proposal.NoVotes += weight
	case VoteDecisionAbstain:
		proposal.AbstainVotes += weight
	}

	return nil
}

// GetProposal 获取提案
func (m *MemoryVoteManager) GetProposal(proposalID []byte) (*Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(proposalID) != 32 {
		return nil, ErrInvalidProposalID
	}

	key := string(proposalID)
	proposal, exists := m.proposals[key]
	if !exists {
		return nil, ErrProposalNotFound
	}

	return proposal, nil
}

// GetAllProposals 获取所有提案
func (m *MemoryVoteManager) GetAllProposals() []*Proposal {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Proposal, 0, len(m.proposals))
	for _, proposal := range m.proposals {
		result = append(result, proposal)
	}

	return result
}

// GetActiveProposals 获取活跃提案
func (m *MemoryVoteManager) GetActiveProposals() []*Proposal {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Proposal, 0)
	for _, proposal := range m.proposals {
		if proposal.Status == ProposalStatusPending {
			result = append(result, proposal)
		}
	}

	return result
}

// GetProposalVotes 获取提案的投票
func (m *MemoryVoteManager) GetProposalVotes(proposalID []byte) ([]*Vote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(proposalID) != 32 {
		return nil, ErrInvalidProposalID
	}

	key := string(proposalID)
	if _, exists := m.proposals[key]; !exists {
		return nil, ErrProposalNotFound
	}

	votes := m.votes[key]
	result := make([]*Vote, len(votes))
	copy(result, votes)

	return result, nil
}

// CountVotes 统计投票
func (m *MemoryVoteManager) CountVotes(proposalID []byte) (yes, no, abstain uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(proposalID) != 32 {
		return 0, 0, 0
	}

	key := string(proposalID)
	proposal, exists := m.proposals[key]
	if !exists {
		return 0, 0, 0
	}

	return proposal.YesVotes, proposal.NoVotes, proposal.AbstainVotes
}

// CheckProposalResult 检查提案结果
func (m *MemoryVoteManager) CheckProposalResult(proposalID []byte) (ProposalStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(proposalID) != 32 {
		return ProposalStatusFailed, ErrInvalidProposalID
	}

	key := string(proposalID)
	proposal, exists := m.proposals[key]
	if !exists {
		return ProposalStatusFailed, ErrProposalNotFound
	}

	// 如果不是待投票状态，直接返回当前状态
	if proposal.Status != ProposalStatusPending {
		return proposal.Status, nil
	}

	// 检查投票期是否结束
	if time.Now().Before(proposal.VotingEndTime) {
		return ProposalStatusPending, nil
	}

	// 计算总投票权重
	totalVotes := proposal.YesVotes + proposal.NoVotes + proposal.AbstainVotes
	if totalVotes == 0 {
		// 没有投票，拒绝
		proposal.Status = ProposalStatusRejected
		return ProposalStatusRejected, nil
	}

	// 计算通过率（赞成票占总投票的百分比）
	// 需要超过 50% 才能通过
	approvalRate := float64(proposal.YesVotes) / float64(totalVotes)

	if approvalRate > 0.5 {
		proposal.Status = ProposalStatusApproved
	} else {
		proposal.Status = ProposalStatusRejected
	}

	return proposal.Status, nil
}

// ProcessProposals 处理提案
func (m *MemoryVoteManager) ProcessProposals() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	processed := 0

	for _, proposal := range m.proposals {
		// 只处理待投票的提案
		if proposal.Status != ProposalStatusPending {
			continue
		}

		// 检查投票期是否结束
		if time.Now().Before(proposal.VotingEndTime) {
			continue
		}

		// 检查提案结果
		status, _ := m.CheckProposalResult(proposal.ID)

		if status == ProposalStatusApproved {
			// 延迟执行
			time.Sleep(ExecutionDelay)

			// 执行提案
			err := m.executor.ExecuteProposal(proposal)
			if err != nil {
				proposal.Status = ProposalStatusFailed
				proposal.Result = fmt.Sprintf("Execution failed: %v", err)
			} else {
				proposal.Status = ProposalStatusExecuted
				now := time.Now()
				proposal.ExecuteTime = &now
				proposal.Result = "Execution successful"
			}

			processed++
		} else if status == ProposalStatusRejected {
			// 被拒绝
			proposal.Status = ProposalStatusRejected
			proposal.Result = "Proposal rejected"
			processed++
		}
	}

	return processed
}

// generateProposalID 生成提案 ID
func (m *MemoryVoteManager) generateProposalID(proposal *Proposal) []byte {
	// 使用提案内容生成哈希
	h := sha256.New()
	h.Write(proposal.ProposerID)
	h.Write([]byte{byte(proposal.Type)})
	h.Write([]byte(proposal.Title))
	h.Write([]byte(proposal.Description))
	h.Write(proposal.Data)
	binary.Write(h, binary.BigEndian, proposal.Deposit)
	binary.Write(h, binary.BigEndian, uint64(proposal.CreatedTime.Unix()))

	return h.Sum(nil)
}

// GetProposalCount 获取提案数量
func (m *MemoryVoteManager) GetProposalCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.proposals)
}

// GetProposalByIndex 按索引获取提案
func (m *MemoryVoteManager) GetProposalByIndex(index int) (*Proposal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if index < 0 || index >= len(m.proposals) {
		return nil, ErrProposalNotFound
	}

	i := 0
	for _, proposal := range m.proposals {
		if i == index {
			return proposal, nil
		}
		i++
	}

	return nil, ErrProposalNotFound
}
