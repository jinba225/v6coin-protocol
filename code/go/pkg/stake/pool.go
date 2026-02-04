package stake

import (
	"errors"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/state"
)

var (
	ErrInsufficientStake     = errors.New("insufficient stake amount")
	ErrMinStakeAmount        = errors.New("amount below minimum stake")
	ErrValidatorNotFound     = errors.New("validator not found")
	ErrAlreadyUnstaking      = errors.New("already unstaking")
	ErrUnstakePeriodNotMet   = errors.New("unstake period not met")
	ErrInvalidValidatorID    = errors.New("invalid validator ID")
)

// MemoryStakePool 内存质押池实现
type MemoryStakePool struct {
	stakes       map[string]*StakeRecord // validator ID -> stake record
	unstakes     []*StakeRecord          // 解押队列
	mu           sync.RWMutex
	stateDB      state.StateDB
	rewardCalc   RewardCalculator
	totalStaked  uint64
}

// NewMemoryStakePool 创建新的内存质押池
func NewMemoryStakePool(stateDB state.StateDB, rewardCalc RewardCalculator) *MemoryStakePool {
	return &MemoryStakePool{
		stakes:     make(map[string]*StakeRecord),
		unstakes:   make([]*StakeRecord, 0),
		stateDB:    stateDB,
		rewardCalc: rewardCalc,
	}
}

// Stake 质押代币
func (p *MemoryStakePool) Stake(validatorID []byte, amount uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(validatorID) != 16 {
		return ErrInvalidValidatorID
	}

	if amount < MinStakeAmount {
		return ErrMinStakeAmount
	}

	key := string(validatorID)

	// 检查是否已有质押记录
	if record, exists := p.stakes[key]; exists {
		if record.Status == StakeStatusActive {
			// 增加质押金额
			record.Amount += amount
			p.totalStaked += amount
			return nil
		} else if record.Status == StakeStatusUnstaking {
			return ErrAlreadyUnstaking
		}
	}

	// 创建新的质押记录
	record := &StakeRecord{
		ValidatorID: validatorID,
		Amount:      amount,
		StakeTime:   time.Now(),
		UnstakeTime: time.Time{}, // 零值表示未解押
		Status:      StakeStatusActive,
		RewardIndex: 0,
	}

	p.stakes[key] = record
	p.totalStaked += amount

	return nil
}

// Unstake 解押代币
func (p *MemoryStakePool) Unstake(validatorID []byte, amount uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(validatorID) != 16 {
		return ErrInvalidValidatorID
	}

	key := string(validatorID)
	record, exists := p.stakes[key]
	if !exists {
		return ErrValidatorNotFound
	}

	if record.Status != StakeStatusActive {
		return ErrAlreadyUnstaking
	}

	if amount > record.Amount {
		return ErrInsufficientStake
	}

	// 检查最小质押期
	if time.Since(record.StakeTime) < MinUnstakePeriod {
		return ErrUnstakePeriodNotMet
	}

	// 更新记录状态
	if amount == record.Amount {
		// 全部解押
		record.Status = StakeStatusUnstaking
		record.UnstakeTime = time.Now()
		p.unstakes = append(p.unstakes, record)
	} else {
		// 部分解押（创建新记录）
		newRecord := &StakeRecord{
			ValidatorID: validatorID,
			Amount:      amount,
			StakeTime:   record.StakeTime,
			UnstakeTime: time.Now(),
			Status:      StakeStatusUnstaking,
			RewardIndex: record.RewardIndex,
		}
		p.unstakes = append(p.unstakes, newRecord)

		// 减少原记录金额
		record.Amount -= amount
		p.totalStaked -= amount
	}

	return nil
}

// GetStake 获取质押记录
func (p *MemoryStakePool) GetStake(validatorID []byte) (*StakeRecord, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(validatorID) != 16 {
		return nil, ErrInvalidValidatorID
	}

	key := string(validatorID)
	record, exists := p.stakes[key]
	if !exists {
		return nil, ErrValidatorNotFound
	}

	return record, nil
}

// GetAllActiveStakes 获取所有活跃质押
func (p *MemoryStakePool) GetAllActiveStakes() []*StakeRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*StakeRecord, 0)
	for _, record := range p.stakes {
		if record.Status == StakeStatusActive {
			result = append(result, record)
		}
	}

	return result
}

// GetPendingUnstakes 获取待解押记录
func (p *MemoryStakePool) GetPendingUnstakes() []*StakeRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	result := make([]*StakeRecord, 0)

	for _, record := range p.unstakes {
		// 检查是否超过解押期
		if record.Status == StakeStatusUnstaking &&
			now.Sub(record.UnstakeTime) >= UnstakePeriod {
			result = append(result, record)
		}
	}

	return result
}

// ProcessUnstakes 处理解押
func (p *MemoryStakePool) ProcessUnstakes() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	processed := 0

	// 处理所有可以解押的记录
	newUnstakes := make([]*StakeRecord, 0)
	for _, record := range p.unstakes {
		if record.Status == StakeStatusUnstaking &&
			now.Sub(record.UnstakeTime) >= UnstakePeriod {
			// 标记为已解押
			record.Status = StakeStatusUnstaked
			p.totalStaked -= record.Amount

			// 从 stakes 中移除（如果是全部解押）
			key := string(record.ValidatorID)
			if original, exists := p.stakes[key]; exists &&
				original.Status == StakeStatusUnstaking {
				delete(p.stakes, key)
			}

			processed++
		} else {
			newUnstakes = append(newUnstakes, record)
		}
	}

	p.unstakes = newUnstakes
	return processed
}

// GetTotalStaked 获取总质押金额
func (p *MemoryStakePool) GetTotalStaked() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.totalStaked
}

// GetValidatorStake 获取验证人质押金额
func (p *MemoryStakePool) GetValidatorStake(validatorID []byte) uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(validatorID) != 16 {
		return 0
	}

	key := string(validatorID)
	record, exists := p.stakes[key]
	if !exists {
		return 0
	}

	if record.Status == StakeStatusActive {
		return record.Amount
	}

	return 0
}

// GetStakeCount 获取质押数量
func (p *MemoryStakePool) GetStakeCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, record := range p.stakes {
		if record.Status == StakeStatusActive {
			count++
		}
	}

	return count
}

// GetTopValidators 获取质押金额最高的 N 个验证人
func (p *MemoryStakePool) GetTopValidators(n int) []*ValidatorInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 收集所有活跃质押
	validators := make([]*ValidatorInfo, 0)
	for _, record := range p.stakes {
		if record.Status == StakeStatusActive {
			validators = append(validators, &ValidatorInfo{
				ValidatorID: record.ValidatorID,
				StakeAmount: record.Amount,
				Status:      record.Status,
			})
		}
	}

	// 按质押金额排序（简单实现）
	// 实际应该使用堆或优先队列
	for i := 0; i < len(validators); i++ {
		for j := i + 1; j < len(validators); j++ {
			if validators[j].StakeAmount > validators[i].StakeAmount {
				validators[i], validators[j] = validators[j], validators[i]
			}
		}
	}

	// 返回前 N 个
	if n > len(validators) {
		n = len(validators)
	}

	return validators[:n]
}
