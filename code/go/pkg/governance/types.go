package governance

import (
	"time"
)

// 治理相关常量
const (
	// MinProposalDeposit 最小提案押金（100 V6）
	MinProposalDeposit = 100 * 1000000000

	// VotingPeriod 投票期（7 天）
	VotingPeriod = 7 * 24 * time.Hour

	// ExecutionDelay 执行延迟（1 天）
	ExecutionDelay = 24 * time.Hour

	// MaxProposals 最大提案数量
	MaxProposals = 1000
)

// ProposalType 提案类型
type ProposalType int

const (
	ProposalTypeParameterChange ProposalType = iota // 参数更改
	ProposalTypeProtocolUpgrade                      // 协议升级
	ProposalTypeFundUsage                           // 资金使用
	ProposalTypeOther                                // 其他
)

// ProposalStatus 提案状态
type ProposalStatus int

const (
	ProposalStatusPending   ProposalStatus = iota // 待投票
	ProposalStatusApproved                          // 已通过
	ProposalStatusRejected                          // 已拒绝
	ProposalStatusExecuted                          // 已执行
	ProposalStatusFailed                             // 执行失败
)

// Proposal 提案
type Proposal struct {
	ID            []byte         // 提案 ID（32 字节哈希）
	ProposerID    []byte         // 提案人 ID（16 字节 IPv6 地址）
	Type          ProposalType   // 提案类型
	Title         string         // 提案标题
	Description   string         // 提案描述
	Data          []byte         // 提案数据（类型特定的数据）
	Deposit       uint64         // 押金金额（nano-V6）
	CreatedTime   time.Time      // 创建时间
	VotingEndTime time.Time      // 投票结束时间
	Status        ProposalStatus  // 提案状态
	YesVotes      uint64         // 赞成票数
	NoVotes       uint64         // 反对票数
	AbstainVotes  uint64         // 弃权票数
	ExecuteTime   *time.Time     // 执行时间（nil 表示未执行）
	Result        string         // 执行结果
}

// Vote 投票
type Vote struct {
	ProposalID  []byte // 提案 ID
	VoterID    []byte // 投票人 ID（16 字节 IPv6 地址）
	Decision   VoteDecision // 投票决定
	VoteTime   time.Time // 投票时间
	VoteWeight uint64   // 投票权重
}

// VoteDecision 投票决定
type VoteDecision int

const (
	VoteDecisionYes     VoteDecision = iota // 赞成
	VoteDecisionNo                        // 反对
	VoteDecisionAbstain                   // 弃权
)

// ProposalExecutor 提案执行器接口
type ProposalExecutor interface {
	// ExecuteProposal 执行提案
	ExecuteProposal(proposal *Proposal) error

	// ValidateProposal 验证提案
	ValidateProposal(proposal *Proposal) error

	// CheckProposalStatus 检查提案状态
	CheckProposalStatus(proposal *Proposal) (ProposalStatus, error)
}

// VoteManager 投票管理器接口
type VoteManager interface {
	// CreateProposal 创建提案
	CreateProposal(proposal *Proposal) error

	// Vote 投票
	Vote(proposalID, voterID []byte, decision VoteDecision, weight uint64) error

	// GetProposal 获取提案
	GetProposal(proposalID []byte) (*Proposal, error)

	// GetAllProposals 获取所有提案
	GetAllProposals() []*Proposal

	// GetActiveProposals 获取活跃提案
	GetActiveProposals() []*Proposal

	// GetProposalVotes 获取提案的投票
	GetProposalVotes(proposalID []byte) []*Vote

	// CountVotes 统计投票
	CountVotes(proposalID []byte) (yes, no, abstain uint64)

	// CheckProposalResult 检查提案结果
	CheckProposalResult(proposalID []byte) (ProposalStatus, error)

	// ProcessProposals 处理提案（检查投票是否结束，执行通过的提案）
	ProcessProposals() int
}

// VoteWeightCalculator 投票权重计算器接口
type VoteWeightCalculator interface {
	// CalculateVoteWeight 计算投票权重
	// 返回值：PoC 贡献度权重 (60%) + 持币量权重 (40%)
	// 单节点上限 5%
	CalculateVoteWeight(voterID []byte, totalSupply uint64) (uint64, error)
}
