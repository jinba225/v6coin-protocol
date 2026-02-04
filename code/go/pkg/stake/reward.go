package stake

import (
	"math"
	"time"
)

// SimpleRewardCalculator 简单奖励计算器实现
type SimpleRewardCalculator struct {
	// BaseReward 基础区块奖励（nano-V6）
	BaseReward uint64

	// OnlineWeight 在线时长权重（0-1）
	OnlineWeight float64

	// StakeWeight 质押比例权重（0-1）
	StakeWeight float64

	// MaxValidators 最大验证人数量
	MaxValidators int
}

// NewSimpleRewardCalculator 创建简单奖励计算器
func NewSimpleRewardCalculator() *SimpleRewardCalculator {
	return &SimpleRewardCalculator{
		BaseReward:    100 * 1000000000, // 100 V6
		OnlineWeight:  0.6,              // 60% 在线时长权重
		StakeWeight:   0.4,              // 40% 质押比例权重
		MaxValidators: 100,
	}
}

// CalculateBlockReward 计算区块奖励
func (c *SimpleRewardCalculator) CalculateBlockReward(height uint64, proposer []byte) (map[string]uint64, error) {
	rewards := make(map[string]uint64)

	// 计算区块奖励（考虑减半）
	blockReward := c.calculateBlockRewardByHeight(height)

	// 提议人获得基础奖励
	rewards[string(proposer)] = blockReward

	return rewards, nil
}

// CalculateOnlineReward 计算在线奖励
func (c *SimpleRewardCalculator) CalculateOnlineReward(record *StakeRecord, onlineDuration time.Duration) uint64 {
	if record == nil || record.Status != StakeStatusActive {
		return 0
	}

	// 计算在线分数（最大 1.0，基于 90 天窗口）
	onlineWindow := 90 * 24 * time.Hour
	onlineScore := math.Min(float64(onlineDuration)/float64(onlineWindow), 1.0)

	// 在线奖励 = 基础奖励 * 在线权重 * 在线分数 * 质押比例
	onlineReward := float64(c.BaseReward) * c.OnlineWeight * onlineScore
	stakeRatio := float64(record.Amount) / float64(MinStakeAmount)

	return uint64(onlineReward * math.Min(stakeRatio, 2.0)) // 最多 2 倍
}

// CalculateStakeReward 计算质押奖励
func (c *SimpleRewardCalculator) CalculateStakeReward(record *StakeRecord, totalReward uint64) uint64 {
	if record == nil || record.Status != StakeStatusActive {
		return 0
	}

	// 质押奖励 = 总奖励 * 质押权重 * (质押金额 / 最小质押)
	stakeRatio := float64(record.Amount) / float64(MinStakeAmount)
	stakeReward := float64(totalReward) * c.StakeWeight * math.Min(stakeRatio, 2.0)

	return uint64(stakeReward)
}

// calculateBlockRewardByHeight 根据高度计算区块奖励（考虑减半）
func (c *SimpleRewardCalculator) calculateBlockRewardByHeight(height uint64) uint64 {
	// 每 4 年减半一次
	// 4 年 = (4 * 365 * 24 * 3600) / 10 = 12,614,400 个区块
	halvingInterval := uint64((4 * 365 * 24 * 3600) / 10)
	halvings := height / halvingInterval

	// 计算奖励：初始奖励 / (2 ^ halvings)
	reward := c.BaseReward >> halvings

	// 最小奖励为 1 nano-V6
	if reward < 1 {
		reward = 1
	}

	return reward
}

// CalculateRewardDistribution 计算奖励分配
func (c *SimpleRewardCalculator) CalculateRewardDistribution(
	validators []*ValidatorInfo,
	totalReward uint64,
) map[string]uint64 {
	if len(validators) == 0 {
		return make(map[string]uint64)
	}

	rewards := make(map[string]uint64)
	totalWeight := 0.0

	// 计算总权重
	for _, v := range validators {
		if v.Status != StakeStatusActive {
			continue
		}

		// 权重 = 在线分数 + 质押分数
		onlineScore := c.calculateOnlineScore(v.OnlineTime)
		stakeScore := c.calculateStakeScore(v.StakeAmount)

		weight := onlineScore*c.OnlineWeight + stakeScore*c.StakeWeight
		totalWeight += weight
	}

	// 分配奖励
	if totalWeight > 0 {
		for _, v := range validators {
			if v.Status != StakeStatusActive {
				continue
			}

			onlineScore := c.calculateOnlineScore(v.OnlineTime)
			stakeScore := c.calculateStakeScore(v.StakeAmount)
			weight := onlineScore*c.OnlineWeight + stakeScore*c.StakeWeight

			reward := uint64((float64(totalReward) * weight / totalWeight))
			rewards[string(v.ValidatorID)] = reward
		}
	}

	return rewards
}

// calculateOnlineScore 计算在线分数
func (c *SimpleRewardCalculator) calculateOnlineScore(onlineTime time.Duration) float64 {
	onlineWindow := 90 * 24 * time.Hour
	score := float64(onlineTime) / float64(onlineWindow)

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateStakeScore 计算质押分数
func (c *SimpleRewardCalculator) calculateStakeScore(stakeAmount uint64) float64 {
	// 质押分数 = 质押金额 / 最小质押
	// 最大为 2.0（质押两倍最小金额）
	score := float64(stakeAmount) / float64(MinStakeAmount)

	if score > 2.0 {
		score = 2.0
	}

	return score
}
