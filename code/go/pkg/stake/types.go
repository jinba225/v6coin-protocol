package stake

import (
	"time"
)

// 质押相关常量
const (
	// MinStakeAmount 最小质押金额（1000 V6）
	MinStakeAmount = 1000 * 1000000000 // nano-V6

	// UnstakePeriod 解押期（21 天）
	UnstakePeriod = 21 * 24 * time.Hour

	// MinUnstakePeriod 最小解押期（7 天）
	MinUnstakePeriod = 7 * 24 * time.Hour
)

// StakeRecord 表示质押记录
type StakeRecord struct {
	ValidatorID []byte    // 验证人 ID（16 字节 IPv6 地址）
	Amount      uint64    // 质押金额（nano-V6）
	StakeTime   time.Time // 质押时间
	UnstakeTime time.Time // 解押时间（零值表示未解押）
	Status      StakeStatus // 质押状态
	RewardIndex uint64    // 奖励索引（用于计算奖励）
}

// StakeStatus 质押状态
type StakeStatus int

const (
	StakeStatusActive   StakeStatus = iota // 活跃质押
	StakeStatusUnstaking                  // 解押中
	StakeStatusUnstaked                   // 已解押
	StakeStatusSlashed                    // 被罚没
)

// StakePool 质押池接口
type StakePool interface {
	// Stake 质押代币
	Stake(validatorID []byte, amount uint64) error

	// Unstake 解押代币
	Unstake(validatorID []byte, amount uint64) error

	// GetStake 获取质押记录
	GetStake(validatorID []byte) (*StakeRecord, error)

	// GetAllActiveStakes 获取所有活跃质押
	GetAllActiveStakes() []*StakeRecord

	// GetPendingUnstakes 获取待解押记录（超过解押期）
	GetPendingUnstakes() []*StakeRecord

	// ProcessUnstakes 处理解押
	ProcessUnstakes() int

	// GetTotalStaked 获取总质押金额
	GetTotalStaked() uint64

	// GetValidatorStake 获取验证人质押金额
	GetValidatorStake(validatorID []byte) uint64
}

// RewardCalculator 奖励计算器接口
type RewardCalculator interface {
	// CalculateBlockReward 计算区块奖励
	CalculateBlockReward(height uint64, proposer []byte) (map[string]uint64, error)

	// CalculateOnlineReward 计算在线奖励
	CalculateOnlineReward(record *StakeRecord, onlineDuration time.Duration) uint64

	// CalculateStakeReward 计算质押奖励
	CalculateStakeReward(record *StakeRecord, totalReward uint64) uint64
}

// ValidatorInfo 验证人信息
type ValidatorInfo struct {
	ValidatorID    []byte    // 验证人 ID
	StakeAmount    uint64    // 质押金额
	OnlineTime     time.Duration // 在线时长
	Status         StakeStatus // 状态
	LastOnlineTime time.Time // 最后在线时间
}

// UnstakeRequest 解押请求
type UnstakeRequest struct {
	ValidatorID []byte    // 验证人 ID
	Amount      uint64    // 解押金额
	RequestTime time.Time // 请求时间
	CompleteTime time.Time // 完成时间
}
