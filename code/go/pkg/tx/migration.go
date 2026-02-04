package tx

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/crypto"
)

var (
	// ErrMigrationInvalidSignature 旧链签名无效
	ErrMigrationInvalidSignature = errors.New("invalid old chain signature")
	// ErrMigrationAssetNotLocked 资产未锁定
	ErrMigrationAssetNotLocked = errors.New("asset not locked on old chain")
	// ErrMigrationAmountMismatch 迁移金额不匹配
	ErrMigrationAmountMismatch = errors.New("migration amount mismatch")
	// ErrMigrationExpired 迁移证明过期
	ErrMigrationExpired = errors.New("migration proof expired")
	// ErrMigrationAlreadyUsed 迁移证明已使用
	ErrMigrationAlreadyUsed = errors.New("migration proof already used")
	// ErrMigrationExceedsLimit 超过迁移额度限制
	ErrMigrationExceedsLimit = errors.New("migration exceeds limit")
)

// MigrationProof 资产迁移证明
type MigrationProof struct {
	// 旧链信息
	OldChainID    string    // 旧链 ID
	OldAddress    string    // 旧链地址
	LockTxHash    []byte    // 锁定交易哈希（32 字节）
	LockAmount    uint64    // 锁定金额（nano-V6）
	LockTimestamp uint64    // 锁定时间戳

	// 新链信息
	NewAddress    []byte    // 新链地址（16 字节 IPv6）
	MigrationAmount uint64  // 迁移金额

	// 验证信息
	Signature     []byte    // 旧链私钥签名
	Timestamp     uint64    // 证明时间戳
}

// MigrationValidator 资产迁移验证器
type MigrationValidator struct {
	// 已使用的迁移证明（防止重复迁移）
	usedProofs    map[string]bool
	// 每个地址的迁移额度限制（防止过度迁移）
	limits        map[string]uint64
	// 迁移证明有效期（30 天）
	proofValidity time.Duration
}

// NewMigrationValidator 创建迁移验证器
func NewMigrationValidator() *MigrationValidator {
	return &MigrationValidator{
		usedProofs:    make(map[string]bool),
		limits:        make(map[string]uint64),
		proofValidity: 30 * 24 * time.Hour,
	}
}

// SetMigrationLimit 设置迁移额度限制
func (mv *MigrationValidator) SetMigrationLimit(address string, limit uint64) {
	mv.limits[address] = limit
}

// ValidateMigration 验证资产迁移证明
// 返回验证后的迁移金额和错误
func (mv *MigrationValidator) ValidateMigration(proof *MigrationProof) (uint64, error) {
	// 1. 验证证明格式
	if err := mv.validateProofFormat(proof); err != nil {
		return 0, fmt.Errorf("invalid proof format: %w", err)
	}

	// 2. 生成证明 ID
	proofID := mv.generateProofID(proof)

	// 3. 检查证明是否已被使用
	if mv.isProofUsed(proofID) {
		return 0, ErrMigrationAlreadyUsed
	}

	// 4. 验证证明有效期
	if err := mv.validateProofExpiry(proof); err != nil {
		return 0, err
	}

	// 5. 验证旧链签名
	if err := mv.validateOldChainSignature(proof); err != nil {
		return 0, err
	}

	// 6. 验证锁定金额
	if err := mv.validateLockAmount(proof); err != nil {
		return 0, err
	}

	// 7. 验证迁移额度
	if err := mv.validateMigrationLimit(proof); err != nil {
		return 0, err
	}

	// 8. 标记证明已使用
	mv.markProofUsed(proofID)

	return proof.MigrationAmount, nil
}

// validateProofFormat 验证证明格式
func (mv *MigrationValidator) validateProofFormat(proof *MigrationProof) error {
	// 检查必需字段
	if proof.OldChainID == "" {
		return errors.New("old chain ID is required")
	}

	if proof.OldAddress == "" {
		return errors.New("old address is required")
	}

	if len(proof.LockTxHash) != 32 {
		return errors.New("lock transaction hash must be 32 bytes")
	}

	if len(proof.NewAddress) != 16 {
		return errors.New("new address must be 16 bytes (IPv6)")
	}

	if len(proof.Signature) == 0 {
		return errors.New("signature is required")
	}

	// 检查金额合理性
	if proof.LockAmount == 0 {
		return errors.New("lock amount cannot be zero")
	}

	if proof.MigrationAmount == 0 {
		return errors.New("migration amount cannot be zero")
	}

	if proof.MigrationAmount > proof.LockAmount {
		return errors.New("migration amount cannot exceed lock amount")
	}

	return nil
}

// validateProofExpiry 验证证明有效期
func (mv *MigrationValidator) validateProofExpiry(proof *MigrationProof) error {
	// 检查证明时间戳
	proofTime := time.Unix(int64(proof.Timestamp), 0)
	age := time.Since(proofTime)

	if age < 0 {
		return errors.New("proof timestamp is in the future")
	}

	if age > mv.proofValidity {
		return ErrMigrationExpired
	}

	// 检查锁定交易时间（不能超过 90 天）
	lockTime := time.Unix(int64(proof.LockTimestamp), 0)
	if time.Since(lockTime) > 90*24*time.Hour {
		return errors.New("lock transaction is too old (max 90 days)")
	}

	return nil
}

// validateOldChainSignature 验证旧链签名
// 简化版：验证签名格式和哈希
// 实际实现需要连接旧链节点验证
func (mv *MigrationValidator) validateOldChainSignature(proof *MigrationProof) error {
	// 1. 验证签名长度（Ed25519 签名 64 字节）
	if len(proof.Signature) != 64 {
		return fmt.Errorf("invalid signature length: expected 64, got %d", len(proof.Signature))
	}

	// 2. 验证签名（使用公钥恢复）
	// 简化版：这里假设签名已经包含公钥信息
	// 实际实现需要：
	//   - 从签名中提取公钥
	//   - 使用 crypto.VerifySignature 验证
	//   - 验证公钥与 OldAddress 匹配

	// 模拟验证：检查签名是否为有效 Ed25519 签名
	// 实际应该调用 crypto.VerifySignature(message, proof.Signature, publicKey)
	if !crypto.IsValidSignatureFormat(proof.Signature) {
		return ErrMigrationInvalidSignature
	}

	// TODO: 实际实现需要连接旧链节点
	// - 验证 LockTxHash 存在
	// - 验证资产已锁定
	// - 验证锁定金额
	// - 验证锁定地址与 OldAddress 匹配

	return nil
}

// validateLockAmount 验证锁定金额
func (mv *MigrationValidator) validateLockAmount(proof *MigrationProof) error {
	// 1. 验证迁移金额不超过锁定金额
	if proof.MigrationAmount > proof.LockAmount {
		return ErrMigrationAmountMismatch
	}

	// 2. 验证最小迁移金额（1 V6）
	minAmount := uint64(1000000000)
	if proof.MigrationAmount < minAmount {
		return fmt.Errorf("migration amount too small (minimum %d nano-V6)", minAmount)
	}

	return nil
}

// validateMigrationLimit 验证迁移额度
func (mv *MigrationValidator) validateMigrationLimit(proof *MigrationProof) error {
	// 获取地址的迁移额度限制
	// 使用新地址作为 key
	key := string(proof.NewAddress)
	limit, exists := mv.limits[key]

	if !exists {
		// 默认限制：总供应量的 0.1%（防止过度迁移）
		// 假设总供应量为 10 亿 V6
		limit = 1000000000 // 1000 V6
	}

	// 计算已迁移金额
	// 简化版：这里只做单次验证
	// 实际实现需要追踪每个地址的累计迁移金额

	if proof.MigrationAmount > limit {
		return ErrMigrationExceedsLimit
	}

	return nil
}

// generateProofID 生成证明 ID
func (mv *MigrationValidator) generateProofID(proof *MigrationProof) string {
	h := sha256.New()
	h.Write([]byte(proof.OldChainID))
	h.Write([]byte(proof.OldAddress))
	h.Write(proof.LockTxHash)
	h.Write(proof.NewAddress)
	binary.Write(h, binary.BigEndian, proof.MigrationAmount)
	binary.Write(h, binary.BigEndian, proof.Timestamp)

	return string(h.Sum(nil))
}

// isProofUsed 检查证明是否已使用
func (mv *MigrationValidator) isProofUsed(proofID string) bool {
	used, exists := mv.usedProofs[proofID]
	return exists && used
}

// markProofUsed 标记证明已使用
func (mv *MigrationValidator) markProofUsed(proofID string) {
	mv.usedProofs[proofID] = true
}

// buildSignatureMessage 构建签名消息
func (mv *MigrationValidator) buildSignatureMessage(proof *MigrationProof) []byte {
	h := sha256.New()
	h.Write([]byte(proof.OldChainID))
	h.Write([]byte(proof.OldAddress))
	h.Write(proof.LockTxHash)
	binary.Write(h, binary.BigEndian, proof.LockAmount)
	binary.Write(h, binary.BigEndian, proof.LockTimestamp)
	h.Write(proof.NewAddress)
	binary.Write(h, binary.BigEndian, proof.MigrationAmount)
	binary.Write(h, binary.BigEndian, proof.Timestamp)

	return h.Sum(nil)
}

// GetUsedProofCount 获取已使用的证明数量
func (mv *MigrationValidator) GetUsedProofCount() int {
	return len(mv.usedProofs)
}

// ClearExpiredProofs 清理过期的证明记录（可选的内存优化）
func (mv *MigrationValidator) ClearExpiredProofs() {
	// 简化版：不实现
	// 实际实现需要追踪证明使用时间，定期清理过期记录
}

// VerifyAssetLockOnOldChain 验证旧链资产锁定
// 这是一个占位符方法，实际实现需要连接旧链节点
func (mv *MigrationValidator) VerifyAssetLockOnOldChain(oldChainID string, lockTxHash []byte, amount uint64) error {
	// TODO: 实际实现需要：
	// 1. 连接到旧链节点（RPC 或轻客户端）
	// 2. 查询锁定交易
	// 3. 验证交易状态（已确认）
	// 4. 验证锁定金额
	// 5. 验证锁定时间（未过期）
	// 6. 验证资产未被花费

	return fmt.Errorf("old chain verification not implemented for chain: %s", oldChainID)
}
