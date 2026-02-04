package governance

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrInvalidProposalData = errors.New("invalid proposal data")
	ErrExecutionFailed    = errors.New("proposal execution failed")
)

// ParameterChangeProposal 参数更改提案数据
type ParameterChangeProposal struct {
	Parameter string `json:"parameter"`
	Value     string `json:"value"`
	Reason    string `json:"reason"`
}

// ProtocolUpgradeProposal 协议升级提案数据
type ProtocolUpgradeProposal struct {
	Version    string   `json:"version"`
	BlockHeight uint64  `json:"block_height"`
	Reason     string   `json:"reason"`
	Validators []string `json:"validators"` // 需要升级的验证人列表
}

// FundUsageProposal 资金使用提案数据
type FundUsageProposal struct {
	Recipient  string `json:"recipient"`  // 接收地址
	Amount     uint64 `json:"amount"`     // 金额（nano-V6）
	Purpose    string `json:"purpose"`    // 用途说明
	Duration   uint64 `json:"duration"`   // 资金锁定期（秒）
}

// SimpleProposalExecutor 简单提案执行器实现
type SimpleProposalExecutor struct {
	// 可以添加依赖项，如配置管理器、资金池等
	config map[string]interface{}
}

// NewSimpleProposalExecutor 创建简单提案执行器
func NewSimpleProposalExecutor() *SimpleProposalExecutor {
	return &SimpleProposalExecutor{
		config: make(map[string]interface{}),
	}
}

// ExecuteProposal 执行提案
func (e *SimpleProposalExecutor) ExecuteProposal(proposal *Proposal) error {
	if proposal == nil {
		return ErrInvalidProposalData
	}

	switch proposal.Type {
	case ProposalTypeParameterChange:
		return e.executeParameterChange(proposal)
	case ProposalTypeProtocolUpgrade:
		return e.executeProtocolUpgrade(proposal)
	case ProposalTypeFundUsage:
		return e.executeFundUsage(proposal)
	default:
		return fmt.Errorf("unsupported proposal type: %d", proposal.Type)
	}
}

// ValidateProposal 验证提案
func (e *SimpleProposalExecutor) ValidateProposal(proposal *Proposal) error {
	if proposal == nil {
		return ErrInvalidProposalData
	}

	if len(proposal.Title) == 0 {
		return errors.New("proposal title cannot be empty")
	}

	if len(proposal.Description) == 0 {
		return errors.New("proposal description cannot be empty")
	}

	// 验证提案数据
	switch proposal.Type {
	case ProposalTypeParameterChange:
		return e.validateParameterChange(proposal)
	case ProposalTypeProtocolUpgrade:
		return e.validateProtocolUpgrade(proposal)
	case ProposalTypeFundUsage:
		return e.validateFundUsage(proposal)
	default:
		return fmt.Errorf("unsupported proposal type: %d", proposal.Type)
	}
}

// CheckProposalStatus 检查提案状态
func (e *SimpleProposalExecutor) CheckProposalStatus(proposal *Proposal) (ProposalStatus, error) {
	if proposal == nil {
		return ProposalStatusFailed, ErrInvalidProposalData
	}

	// 根据执行时间判断状态
	if proposal.ExecuteTime != nil {
		return ProposalStatusExecuted, nil
	}

	// 其他状态由 VoteManager 管理
	return proposal.Status, nil
}

// executeParameterChange 执行参数更改
func (e *SimpleProposalExecutor) executeParameterChange(proposal *Proposal) error {
	var paramChange ParameterChangeProposal
	err := json.Unmarshal(proposal.Data, &paramChange)
	if err != nil {
		return fmt.Errorf("failed to unmarshal parameter change: %w", err)
	}

	// 验证参数名
	if paramChange.Parameter == "" {
		return errors.New("parameter name cannot be empty")
	}

	// 在实际实现中，这里会更新配置
	// 简化版：只记录到内存
	e.config[paramChange.Parameter] = paramChange.Value

	proposal.Result = fmt.Sprintf("Parameter '%s' updated to '%s'", paramChange.Parameter, paramChange.Value)
	return nil
}

// executeProtocolUpgrade 执行协议升级
func (e *SimpleProposalExecutor) executeProtocolUpgrade(proposal *Proposal) error {
	var upgrade ProtocolUpgradeProposal
	err := json.Unmarshal(proposal.Data, &upgrade)
	if err != nil {
		return fmt.Errorf("failed to unmarshal protocol upgrade: %w", err)
	}

	// 验证版本
	if upgrade.Version == "" {
		return errors.New("version cannot be empty")
	}

	// 在实际实现中，这里会触发协议升级流程
	// 简化版：只记录
	proposal.Result = fmt.Sprintf("Protocol upgraded to version %s at height %d", upgrade.Version, upgrade.BlockHeight)

	return nil
}

// executeFundUsage 执行资金使用
func (e *SimpleProposalExecutor) executeFundUsage(proposal *Proposal) error {
	var fundUsage FundUsageProposal
	err := json.Unmarshal(proposal.Data, &fundUsage)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fund usage: %w", err)
	}

	// 验证接收地址
	if fundUsage.Recipient == "" {
		return errors.New("recipient cannot be empty")
	}

	// 验证金额
	if fundUsage.Amount == 0 {
		return errors.New("amount must be greater than 0")
	}

	// 在实际实现中，这里会从资金池转账
	// 简化版：只记录
	proposal.Result = fmt.Sprintf("Transferred %d nano-V6 to %s for: %s", fundUsage.Amount, fundUsage.Recipient, fundUsage.Purpose)

	return nil
}

// validateParameterChange 验证参数更改提案
func (e *SimpleProposalExecutor) validateParameterChange(proposal *Proposal) error {
	var paramChange ParameterChangeProposal
	err := json.Unmarshal(proposal.Data, &paramChange)
	if err != nil {
		return fmt.Errorf("failed to unmarshal parameter change: %w", err)
	}

	// 验证参数名
	if paramChange.Parameter == "" {
		return errors.New("parameter name cannot be empty")
	}

	// 验证值
	if paramChange.Value == "" {
		return errors.New("parameter value cannot be empty")
	}

	return nil
}

// validateProtocolUpgrade 验证协议升级提案
func (e *SimpleProposalExecutor) validateProtocolUpgrade(proposal *Proposal) error {
	var upgrade ProtocolUpgradeProposal
	err := json.Unmarshal(proposal.Data, &upgrade)
	if err != nil {
		return fmt.Errorf("failed to unmarshal protocol upgrade: %w", err)
	}

	// 验证版本
	if upgrade.Version == "" {
		return errors.New("version cannot be empty")
	}

	// 验证高度
	if upgrade.BlockHeight == 0 {
		return errors.New("block height must be greater than 0")
	}

	return nil
}

// validateFundUsage 验证资金使用提案
func (e *SimpleProposalExecutor) validateFundUsage(proposal *Proposal) error {
	var fundUsage FundUsageProposal
	err := json.Unmarshal(proposal.Data, &fundUsage)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fund usage: %w", err)
	}

	// 验证接收地址
	if fundUsage.Recipient == "" {
		return errors.New("recipient cannot be empty")
	}

	// 验证金额
	if fundUsage.Amount == 0 {
		return errors.New("amount must be greater than 0")
	}

	// 验证用途
	if fundUsage.Purpose == "" {
		return errors.New("purpose cannot be empty")
	}

	return nil
}
