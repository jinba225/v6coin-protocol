package tx

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

var (
	// ErrOfflineTxExpired 离线交易已过期
	ErrOfflineTxExpired = errors.New("offline transaction expired")
	// ErrOfflineTxTimestampInvalid 时间戳无效
	ErrOfflineTxTimestampInvalid = errors.New("offline transaction timestamp is invalid")
	// ErrOfflineTxFutureTimestamp 时间戳在未来
	ErrOfflineTxFutureTimestamp = errors.New("offline transaction timestamp is in the future")
	// ErrOfflineTxReplayDetected 检测到重放攻击
	ErrOfflineTxReplayDetected = errors.New("offline transaction replay detected")
	// ErrOfflineTxNonceTooLow Nonce 过低
	ErrOfflineTxNonceTooLow = errors.New("offline transaction nonce too low")
)

// OfflineTxValidator 离线交易验证器
type OfflineTxValidator struct {
	// 交易缓存（防止重放攻击）
	seenTransactions map[string]*OfflineTxInfo
	mu               sync.RWMutex

	// 配置参数
	validityPeriod  time.Duration // 有效期（默认 7 天）
	clockSkewTolerance time.Duration // 时钟偏差容忍度（默认 5 分钟）
	maxFutureTime   time.Duration // 最大未来时间（默认 10 分钟）
}

// OfflineTxInfo 离线交易信息
type OfflineTxInfo struct {
	TxHash     []byte    // 交易哈希
	From       []byte    // 发送方地址
	Nonce      uint64    // Nonce
	FirstSeen  time.Time // 首次 seen 时间
	ExpiryTime time.Time // 过期时间
}

// NewOfflineTxValidator 创建离线交易验证器
func NewOfflineTxValidator() *OfflineTxValidator {
	return &OfflineTxValidator{
		seenTransactions:     make(map[string]*OfflineTxInfo),
		validityPeriod:       7 * 24 * time.Hour, // 7 天
		clockSkewTolerance:   5 * time.Minute,    // 5 分钟
		maxFutureTime:        10 * time.Minute,   // 10 分钟
	}
}

// SetValidityPeriod 设置有效期
func (v *OfflineTxValidator) SetValidityPeriod(period time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.validityPeriod = period
}

// SetClockSkewTolerance 设置时钟偏差容忍度
func (v *OfflineTxValidator) SetClockSkewTolerance(tolerance time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.clockSkewTolerance = tolerance
}

// ValidateOfflineTx 验证离线交易
func (v *OfflineTxValidator) ValidateOfflineTx(tx *consensus.Transaction, currentNonce uint64) error {
	// 1. 验证交易类型
	if tx.Type != consensus.TxTypeOffline {
		return errors.New("not an offline transaction")
	}

	// 2. 验证时间戳
	if err := v.validateTimestamp(tx); err != nil {
		return err
	}

	// 3. 验证有效期
	if err := v.validateExpiry(tx); err != nil {
		return err
	}

	// 4. 验证 Nonce
	if err := v.validateNonce(tx, currentNonce); err != nil {
		return err
	}

	// 5. 检查重放攻击
	if err := v.checkReplay(tx); err != nil {
		return err
	}

	return nil
}

// validateTimestamp 验证时间戳
func (v *OfflineTxValidator) validateTimestamp(tx *consensus.Transaction) error {
	// 检查时间戳不为零
	if tx.Timestamp == 0 {
		return ErrOfflineTxTimestampInvalid
	}

	// 获取当前时间
	now := time.Now()

	// 解析交易时间戳
	txTime := time.Unix(int64(tx.Timestamp), 0)

	// 检查时间戳是否在未来（考虑时钟偏差）
	if txTime.Sub(now) > v.maxFutureTime {
		return ErrOfflineTxFutureTimestamp
	}

	// 检查时间戳是否过于久远
	if now.Sub(txTime) > v.validityPeriod+v.clockSkewTolerance {
		return ErrOfflineTxExpired
	}

	return nil
}

// validateExpiry 验证有效期
func (v *OfflineTxValidator) validateExpiry(tx *consensus.Transaction) error {
	txTime := time.Unix(int64(tx.Timestamp), 0)
	now := time.Now()

	// 计算交易年龄
	age := now.Sub(txTime)

	// 检查是否超过有效期
	if age > v.validityPeriod {
		return ErrOfflineTxExpired
	}

	return nil
}

// validateNonce 验证 Nonce
func (v *OfflineTxValidator) validateNonce(tx *consensus.Transaction, currentNonce uint64) error {
	// Nonce 必须大于等于当前 Nonce
	if tx.Nonce < currentNonce {
		return fmt.Errorf("%w: current=%d, tx=%d", ErrOfflineTxNonceTooLow, currentNonce, tx.Nonce)
	}

	// Nonce 不应过大（防止攻击）
	// 允许最多提前 100 个 Nonce
	maxFutureNonce := currentNonce + 100
	if tx.Nonce > maxFutureNonce {
		return fmt.Errorf("nonce too far in future: current=%d, tx=%d, max=%d",
			currentNonce, tx.Nonce, maxFutureNonce)
	}

	return nil
}

// checkReplay 检查重放攻击
func (v *OfflineTxValidator) checkReplay(tx *consensus.Transaction) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 生成交易唯一标识
	txKey := v.getTxKey(tx)

	// 检查是否已见过此交易
	if info, exists := v.seenTransactions[txKey]; exists {
		// 检查是否过期
		if time.Now().Before(info.ExpiryTime) {
			// 交易仍在有效期内，确认为重放攻击
			return ErrOfflineTxReplayDetected
		}
		// 交易已过期，允许重新处理（更新信息）
		delete(v.seenTransactions, txKey)
	}

	// 记录交易信息
	txHash := tx.Hash()
	txTime := time.Unix(int64(tx.Timestamp), 0)

	v.seenTransactions[txKey] = &OfflineTxInfo{
		TxHash:     txHash,
		From:       tx.From,
		Nonce:      tx.Nonce,
		FirstSeen:  time.Now(),
		ExpiryTime: txTime.Add(v.validityPeriod),
	}

	return nil
}

// getTxKey 生成交易唯一标识
func (v *OfflineTxValidator) getTxKey(tx *consensus.Transaction) string {
	// 使用发送方地址和 Nonce 作为唯一标识
	// 这确保了同一地址的相同 Nonce 只能被使用一次
	return string(tx.From) + ":" + fmt.Sprintf("%d", tx.Nonce)
}

// GetTxInfo 获取交易信息
func (v *OfflineTxValidator) GetTxInfo(tx *consensus.Transaction) (*OfflineTxInfo, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	txKey := v.getTxKey(tx)
	info, exists := v.seenTransactions[txKey]
	if !exists {
		return nil, errors.New("transaction not found")
	}

	return info, nil
}

// CleanupExpired 清理过期的交易记录
func (v *OfflineTxValidator) CleanupExpired() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	// 找出所有过期记录
	for key, info := range v.seenTransactions {
		if now.After(info.ExpiryTime) {
			toRemove = append(toRemove, key)
		}
	}

	// 删除过期记录
	for _, key := range toRemove {
		delete(v.seenTransactions, key)
	}

	return len(toRemove)
}

// GetSeenTxCount 获取已 seen 交易数量
func (v *OfflineTxValidator) GetSeenTxCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return len(v.seenTransactions)
}

// ValidateTransactionBatch 批量验证交易（优化版）
func (v *OfflineTxValidator) ValidateTransactionBatch(txs []*consensus.Transaction, currentNonce uint64) error {
	// 简化版：逐个验证
	// 实际实现可以批量处理以提高性能
	for _, tx := range txs {
		if err := v.ValidateOfflineTx(tx, currentNonce); err != nil {
			return err
		}
	}
	return nil
}

// GetRemainingTime 获取交易剩余有效时间
func (v *OfflineTxValidator) GetRemainingTime(tx *consensus.Transaction) (time.Duration, error) {
	txTime := time.Unix(int64(tx.Timestamp), 0)
	expiryTime := txTime.Add(v.validityPeriod)
	remaining := time.Until(expiryTime)

	if remaining <= 0 {
		return 0, ErrOfflineTxExpired
	}

	return remaining, nil
}

// IsExpired 检查交易是否已过期
func (v *OfflineTxValidator) IsExpired(tx *consensus.Transaction) bool {
	txTime := time.Unix(int64(tx.Timestamp), 0)
	expiryTime := txTime.Add(v.validityPeriod)
	return time.Now().After(expiryTime)
}

// Reset 重置验证器状态（用于测试）
func (v *OfflineTxValidator) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.seenTransactions = make(map[string]*OfflineTxInfo)
}
