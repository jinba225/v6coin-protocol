package governance

import (
	"fmt"
	"net"
	"time"
)

// SimpleVoteWeightCalculator 简单投票权重计算器
type SimpleVoteWeightCalculator struct {
	consensusEngine ConsensusEngine
	stakePool       StakePool
}

// ConsensusEngine 共识引擎接口
type ConsensusEngine interface {
	GetNodeContribution(validatorID []byte) (*NodeContribution, error)
}

// NodeContribution 节点贡献
type NodeContribution struct {
	OnlineTime  float64 // 在线时长（秒）
	Forwarded   uint64  // 转发量
	PacketLoss  float64 // 丢包率
	Score        float64 // 贡献分数
}

// StakePool 质押池接口
type StakePool interface {
	GetValidatorStake(validatorID []byte) uint64
}

// NewSimpleVoteWeightCalculator 创建简单投票权重计算器
func NewSimpleVoteWeightCalculator(consensusEngine ConsensusEngine, stakePool StakePool) *SimpleVoteWeightCalculator {
	return &SimpleVoteWeightCalculator{
		consensusEngine: consensusEngine,
		stakePool:       stakePool,
	}
}

// CalculateVoteWeight 计算投票权重
// 返回值：PoC 贡献度权重 (60%) + 持币量权重 (40%)
// 单节点上限 5%
func (c *SimpleVoteWeightCalculator) CalculateVoteWeight(voterID []byte, totalSupply uint64) (uint64, error) {
	if len(voterID) != 16 {
		return 0, fmt.Errorf("invalid voter ID length")
	}

	// 1. 计算 PoC 贡献度权重（60%）
	pocWeight := c.calculatePocWeight(voterID)

	// 2. 计算持币量权重（40%）
	stakeWeight := c.calculateStakeWeight(voterID, totalSupply)

	// 3. 合并权重
	totalWeight := pocWeight + stakeWeight

	// 4. 单节点上限 5%
	maxWeight := totalSupply / 20 // 5% of total supply
	if totalWeight > maxWeight {
		totalWeight = maxWeight
	}

	return totalWeight, nil
}

// calculatePocWeight 计算 PoC 贡献度权重
func (c *SimpleVoteWeightCalculator) calculatePocWeight(voterID []byte) uint64 {
	if c.consensusEngine == nil {
		return 0
	}

	// 获取节点贡献
	contribution, err := c.consensusEngine.GetNodeContribution(voterID)
	if err != nil {
		return 0
	}

	// 贡献度分数（0-100）
	// 在线时长 60% + 转发量 30% + 丢包率 10%
	score := contribution.Score

	// 权重 = 总供应量 * 60% * (分数 / 100)
	// 这里简化计算，实际应该传入总供应量
	// 返回相对权重（0-60）
	return uint64(score * 0.6 * 1000000000)
}

// calculateStakeWeight 计算持币量权重
func (c *SimpleVoteWeightCalculator) calculateStakeWeight(voterID []byte, totalSupply uint64) uint64 {
	if c.stakePool == nil {
		return 0
	}

	// 获取质押金额
	stakeAmount := c.stakePool.GetValidatorStake(voterID)

	// 权重 = (质押金额 / 总供应量) * 40%
	weight := uint64((float64(stakeAmount) / float64(totalSupply)) * 0.4 * float64(totalSupply))

	return weight
}

// VoteWithValidator 使用验证人投票（简化版）
func (c *SimpleVoteWeightCalculator) VoteWithValidator(
	voterIP net.IP,
	proposalID []byte,
	decision VoteDecision,
	totalSupply uint64,
) (*Vote, error) {
	voterID := voterIP.To16()

	// 计算投票权重
	weight, err := c.CalculateVoteWeight(voterID, totalSupply)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate vote weight: %w", err)
	}

	// 创建投票
	vote := &Vote{
		ProposalID:  proposalID,
		VoterID:    voterID,
		Decision:   decision,
		VoteTime:   time.Now(),
		VoteWeight: weight,
	}

	return vote, nil
}

// GetVoterPower 获取投票人权力（用于调试）
func (c *SimpleVoteWeightCalculator) GetVoterPower(voterID []byte, totalSupply uint64) (poc, stake uint64, err error) {
	if len(voterID) != 16 {
		return 0, 0, fmt.Errorf("invalid voter ID length")
	}

	// 计算 PoC 权重
	if c.consensusEngine != nil {
		contribution, _ := c.consensusEngine.GetNodeContribution(voterID)
		poc = uint64(contribution.Score * 0.6 * 1000000000)
	}

	// 计算质押权重
	if c.stakePool != nil {
		stakeAmount := c.stakePool.GetValidatorStake(voterID)
		stake = uint64((float64(stakeAmount) / float64(totalSupply)) * 0.4 * float64(totalSupply))
	}

	return poc, stake, nil
}
