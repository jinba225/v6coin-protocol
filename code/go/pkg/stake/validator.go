package stake

import (
	"sync"
	"time"
)

// ValidatorManager 验证人管理器
type ValidatorManager struct {
	validators map[string]*ValidatorInfo // validator ID -> info
	mu         sync.RWMutex
	stakePool  StakePool
}

// NewValidatorManager 创建验证人管理器
func NewValidatorManager(stakePool StakePool) *ValidatorManager {
	return &ValidatorManager{
		validators: make(map[string]*ValidatorInfo),
		stakePool:  stakePool,
	}
}

// AddOrUpdateValidator 添加或更新验证人
func (m *ValidatorManager) AddOrUpdateValidator(validatorID []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(validatorID) != 16 {
		return ErrInvalidValidatorID
	}

	key := string(validatorID)

	// 检查是否已存在
	if info, exists := m.validators[key]; exists {
		// 更新最后在线时间
		info.LastOnlineTime = time.Now()
		return nil
	}

	// 获取质押信息
	stakeAmount := m.stakePool.GetValidatorStake(validatorID)

	// 创建新的验证人信息
	info := &ValidatorInfo{
		ValidatorID:    validatorID,
		StakeAmount:    stakeAmount,
		OnlineTime:     0,
		Status:         StakeStatusActive,
		LastOnlineTime: time.Now(),
	}

	m.validators[key] = info
	return nil
}

// GetValidator 获取验证人信息
func (m *ValidatorManager) GetValidator(validatorID []byte) (*ValidatorInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(validatorID) != 16 {
		return nil, ErrInvalidValidatorID
	}

	key := string(validatorID)
	info, exists := m.validators[key]
	if !exists {
		return nil, ErrValidatorNotFound
	}

	return info, nil
}

// GetAllValidators 获取所有验证人
func (m *ValidatorManager) GetAllValidators() []*ValidatorInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ValidatorInfo, 0, len(m.validators))
	for _, info := range m.validators {
		result = append(result, info)
	}

	return result
}

// GetActiveValidators 获取活跃验证人
func (m *ValidatorManager) GetActiveValidators() []*ValidatorInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ValidatorInfo, 0)
	for _, info := range m.validators {
		if info.Status == StakeStatusActive {
			result = append(result, info)
		}
	}

	return result
}

// UpdateOnlineTime 更新验证人在线时长
func (m *ValidatorManager) UpdateOnlineTime(validatorID []byte, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(validatorID) != 16 {
		return ErrInvalidValidatorID
	}

	key := string(validatorID)
	info, exists := m.validators[key]
	if !exists {
		return ErrValidatorNotFound
	}

	info.OnlineTime += duration
	info.LastOnlineTime = time.Now()

	return nil
}

// RemoveValidator 移除验证人
func (m *ValidatorManager) RemoveValidator(validatorID []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(validatorID) != 16 {
		return ErrInvalidValidatorID
	}

	key := string(validatorID)
	delete(m.validators, key)

	return nil
}

// GetValidatorCount 获取验证人数量
func (m *ValidatorManager) GetValidatorCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.validators)
}

// GetActiveValidatorCount 获取活跃验证人数量
func (m *ValidatorManager) GetActiveValidatorCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, info := range m.validators {
		if info.Status == StakeStatusActive {
			count++
		}
	}

	return count
}
