package tx

import (
	"net"
	"testing"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/stretchr/testify/assert"
)

// ==================== 资产迁移测试 ====================

func createTestMigrationProof() *MigrationProof {
	sig := make([]byte, 64)
	sig[0] = 0x01 // 设置非零值以通过签名格式验证

	newAddr := make([]byte, 16)
	newAddr[0] = 0x01

	lockHash := make([]byte, 32)
	lockHash[0] = 0x01

	return &MigrationProof{
		OldChainID:     "eth-mainnet",
		OldAddress:     "0x1234567890123456789012345678901234567890",
		LockTxHash:     lockHash,
		LockAmount:     1000000000000, // 1000 V6
		LockTimestamp:  uint64(time.Now().Unix()),
		NewAddress:     newAddr,
		MigrationAmount: 1000000000000, // 1000 V6
		Signature:      sig,
		Timestamp:      uint64(time.Now().Unix()),
	}
}

func TestMigrationValidatorValidProof(t *testing.T) {
	validator := NewMigrationValidator()
	proof := createTestMigrationProof()

	// 设置足够高的迁移额度限制
	validator.SetMigrationLimit(string(proof.NewAddress), 10000000000000) // 10000 V6

	amount, err := validator.ValidateMigration(proof)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000000000000), amount)
}

func TestMigrationValidatorInvalidSignature(t *testing.T) {
	validator := NewMigrationValidator()
	proof := createTestMigrationProof()
	proof.Signature = []byte{0x01} // 无效签名长度

	_, err := validator.ValidateMigration(proof)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature length")
}

func TestMigrationValidatorAmountMismatch(t *testing.T) {
	validator := NewMigrationValidator()
	proof := createTestMigrationProof()
	proof.MigrationAmount = proof.LockAmount + 1 // 超过锁定金额

	_, err := validator.ValidateMigration(proof)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "migration amount cannot exceed lock amount")
}

func TestMigrationValidatorExpiredProof(t *testing.T) {
	validator := NewMigrationValidator()
	proof := createTestMigrationProof()
	proof.Timestamp = uint64(time.Now().Add(-40 * 24 * time.Hour).Unix()) // 40 天前

	_, err := validator.ValidateMigration(proof)
	assert.Error(t, err)
	assert.Equal(t, ErrMigrationExpired, err)
}

func TestMigrationValidatorReplayAttack(t *testing.T) {
	validator := NewMigrationValidator()
	proof := createTestMigrationProof()

	// 设置足够高的迁移额度限制
	validator.SetMigrationLimit(string(proof.NewAddress), 10000000000000) // 10000 V6

	// 第一次验证应该成功
	_, err := validator.ValidateMigration(proof)
	assert.NoError(t, err)

	// 第二次验证应该失败（重放攻击）
	_, err = validator.ValidateMigration(proof)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already used")
}

func TestMigrationValidatorMigrationLimit(t *testing.T) {
	validator := NewMigrationValidator()
	proof := createTestMigrationProof()

	// 设置迁移额度限制
	validator.SetMigrationLimit(string(proof.NewAddress), 500000000000) // 500 V6

	_, err := validator.ValidateMigration(proof)
	assert.Error(t, err)
	assert.Equal(t, ErrMigrationExceedsLimit, err)
}

// ==================== 离线交易测试 ====================

func createTestOfflineTx(nonce uint64) *consensus.Transaction {
	fromIP := net.ParseIP("::1")
	toIP := net.ParseIP("::2")

	return &consensus.Transaction{
		Version:   0x0100,
		Type:      consensus.TxTypeOffline,
		From:      fromIP,
		To:        toIP,
		Amount:    1000000000, // 1 V6
		Fee:       10000,
		Nonce:     nonce,
		Timestamp: uint64(time.Now().Unix()),
		Signature: make([]byte, 64),
		Data:      []byte{},
	}
}

func TestOfflineTxValidatorValidTx(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(1)
	currentNonce := uint64(0)

	err := validator.ValidateOfflineTx(tx, currentNonce)
	assert.NoError(t, err)
}

func TestOfflineTxValidatorExpiredTx(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(1)
	tx.Timestamp = uint64(time.Now().Add(-10 * 24 * time.Hour).Unix()) // 10 天前

	currentNonce := uint64(0)
	err := validator.ValidateOfflineTx(tx, currentNonce)
	assert.Error(t, err)
	assert.Equal(t, ErrOfflineTxExpired, err)
}

func TestOfflineTxValidatorFutureTimestamp(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(1)
	tx.Timestamp = uint64(time.Now().Add(20 * time.Minute).Unix()) // 20 分钟后

	currentNonce := uint64(0)
	err := validator.ValidateOfflineTx(tx, currentNonce)
	assert.Error(t, err)
	assert.Equal(t, ErrOfflineTxFutureTimestamp, err)
}

func TestOfflineTxValidatorNonceTooLow(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(5) // Nonce = 5

	currentNonce := uint64(10) // 当前 Nonce = 10
	err := validator.ValidateOfflineTx(tx, currentNonce)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonce too low")
}

func TestOfflineTxValidatorReplayAttack(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(1)
	currentNonce := uint64(0)

	// 第一次验证应该成功
	err := validator.ValidateOfflineTx(tx, currentNonce)
	assert.NoError(t, err)

	// 第二次验证应该失败（重放攻击）
	err = validator.ValidateOfflineTx(tx, currentNonce)
	assert.Error(t, err)
	assert.Equal(t, ErrOfflineTxReplayDetected, err)
}

func TestOfflineTxValidatorCleanupExpired(t *testing.T) {
	validator := NewOfflineTxValidator()

	// 创建一个过期交易
	tx := createTestOfflineTx(1)
	tx.Timestamp = uint64(time.Now().Add(-10 * 24 * time.Hour).Unix())

	currentNonce := uint64(0)
	validator.ValidateOfflineTx(tx, currentNonce)

	// 清理过期记录
	count := validator.CleanupExpired()
	assert.GreaterOrEqual(t, count, 0)
}

func TestOfflineTxValidatorRemainingTime(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(1)
	tx.Timestamp = uint64(time.Now().Add(-24 * time.Hour).Unix()) // 1 天前

	remaining, err := validator.GetRemainingTime(tx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, remaining, 143*time.Hour) // 应该还剩约 6 天
	assert.LessOrEqual(t, remaining, 7*24*time.Hour)
}

func TestOfflineTxValidatorIsExpired(t *testing.T) {
	validator := NewOfflineTxValidator()
	tx := createTestOfflineTx(1)

	// 未过期
	assert.False(t, validator.IsExpired(tx))

	// 过期
	tx.Timestamp = uint64(time.Now().Add(-10 * 24 * time.Hour).Unix())
	assert.True(t, validator.IsExpired(tx))
}

// ==================== Gas 计算测试 ====================

func createTestTransactionForGas(txType consensus.TxType, dataSize int) *consensus.Transaction {
	fromIP := net.ParseIP("::1")
	toIP := net.ParseIP("::2")

	data := make([]byte, dataSize)

	return &consensus.Transaction{
		Version:   0x0100,
		Type:      txType,
		From:      fromIP,
		To:        toIP,
		Amount:    1000000000,
		Fee:       10000,
		Nonce:     1,
		Timestamp: uint64(time.Now().Unix()),
		Signature: make([]byte, 64),
		Data:      data,
	}
}

func TestGasCalculatorBasicTx(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	tx := createTestTransactionForGas(consensus.TxTypeOnline, 0)
	gasLimit, gasPrice, totalFee, err := calculator.CalculateGas(tx)

	assert.NoError(t, err)
	assert.Greater(t, gasLimit, uint64(0))
	assert.Greater(t, gasPrice, uint64(0))
	assert.Equal(t, totalFee, gasLimit*gasPrice)
}

func TestGasCalculatorOfflineTx(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	tx := createTestTransactionForGas(consensus.TxTypeOffline, 100)
	gasLimit, _, _, err := calculator.CalculateGas(tx)

	assert.NoError(t, err)
	// 离线交易 Gas 应该比在线交易高
	assert.Greater(t, gasLimit, uint64(21000))
}

func TestGasCalculatorGovernanceTx(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	tx := createTestTransactionForGas(consensus.TxTypeGovernance, 1000)
	gasLimit, _, _, err := calculator.CalculateGas(tx)

	assert.NoError(t, err)
	// 治理交易 Gas 应该很高
	assert.Greater(t, gasLimit, uint64(100000))
}

func TestGasCalculatorMigrationTx(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	tx := createTestTransactionForGas(consensus.TxTypeMigration, 500)
	gasLimit, _, _, err := calculator.CalculateGas(tx)

	assert.NoError(t, err)
	// 迁移交易 Gas 应该适中
	assert.Greater(t, gasLimit, uint64(40000))
}

func TestGasCalculatorDataGas(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	// 小数据
	tx1 := createTestTransactionForGas(consensus.TxTypeOffline, 100)
	gasLimit1, _, _, _ := calculator.CalculateGas(tx1)

	// 大数据
	tx2 := createTestTransactionForGas(consensus.TxTypeOffline, 1000)
	gasLimit2, _, _, _ := calculator.CalculateGas(tx2)

	// 大数据交易 Gas 应该更高
	assert.Greater(t, gasLimit2, gasLimit1)
}

func TestGasCalculatorPriceUpdate(t *testing.T) {
	config := DefaultGasConfig()
	config.BaseGasPrice = 100 // 设置更高的基础价格以便观察变化
	calculator := NewGasCalculator(config)

	// 初始价格
	initialPrice := calculator.GetCurrentPrice()

	// 模拟高区块利用率（拥堵）
	calculator.UpdateGasPrice(0.9, 8000)
	newPrice := calculator.GetCurrentPrice()

	// 价格应该上涨或保持不变
	assert.GreaterOrEqual(t, newPrice, initialPrice)
}

func TestGasCalculatorPriceDecrease(t *testing.T) {
	config := DefaultGasConfig()
	config.BaseGasPrice = 1000 // 设置更高的基础价格
	calculator := NewGasCalculator(config)

	// 先设置高价格
	calculator.UpdateGasPrice(0.9, 8000)
	highPrice := calculator.GetCurrentPrice()

	// 模拟低区块利用率（空闲）
	calculator.UpdateGasPrice(0.1, 1000)
	lowPrice := calculator.GetCurrentPrice()

	// 价格应该下降或保持不变
	assert.LessOrEqual(t, lowPrice, highPrice)
}

func TestGasCalculatorEstimatePrice(t *testing.T) {
	config := DefaultGasConfig()
	config.BaseGasPrice = 100 // 设置更高的基础价格
	calculator := NewGasCalculator(config)

	// 低优先级
	lowPrice := calculator.EstimateGasPrice(0.0)

	// 高优先级
	highPrice := calculator.EstimateGasPrice(1.0)

	// 高优先级价格应该更高或相等
	assert.GreaterOrEqual(t, highPrice, lowPrice)
}

func TestGasCalculatorRefund(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	gasLimit := uint64(100000)
	gasUsed := uint64(50000)
	gasPrice := uint64(10)

	actualUsed, refund := calculator.RefundGas(gasLimit, gasUsed, gasPrice)

	assert.Equal(t, gasUsed, actualUsed)
	assert.Equal(t, uint64(500000), refund) // (100000-50000) * 10
}

func TestGasCalculatorValidateFee(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	tx := createTestTransactionForGas(consensus.TxTypeOffline, 100)

	// 足够的费用
	_, _, requiredFee, _ := calculator.CalculateGas(tx)
	err := calculator.ValidateGasFee(tx, requiredFee)
	assert.NoError(t, err)

	// 不足的费用
	err = calculator.ValidateGasFee(tx, requiredFee-1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient gas fee")
}

func TestGasCalculatorComplexity(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	// 简单交易
	simpleTx := createTestTransactionForGas(consensus.TxTypeOnline, 0)
	simpleComplexity := calculator.CalculateTransactionComplexity(simpleTx)

	// 复杂交易
	complexTx := createTestTransactionForGas(consensus.TxTypeGovernance, 1000)
	complexTx.Amount = 10000000000000 // 大额
	complexComplexity := calculator.CalculateTransactionComplexity(complexTx)

	// 复杂交易复杂度应该更高
	assert.Greater(t, complexComplexity, simpleComplexity)
}

func TestGasCalculatorPriceHistory(t *testing.T) {
	config := DefaultGasConfig()
	calculator := NewGasCalculator(config)

	// 更新几次价格
	calculator.UpdateGasPrice(0.5, 5000)
	calculator.UpdateGasPrice(0.6, 6000)
	calculator.UpdateGasPrice(0.7, 7000)

	// 获取历史
	history := calculator.GetPriceHistory()
	assert.Len(t, history, 3)

	// 获取平均价格
	avgPrice := calculator.GetAveragePrice()
	assert.Greater(t, avgPrice, uint64(0))
}

func TestGasCalculatorPercentile(t *testing.T) {
	config := DefaultGasConfig()
	config.BaseGasPrice = 1000 // 设置更高的基础价格
	calculator := NewGasCalculator(config)

	// 更新价格
	calculator.UpdateGasPrice(0.5, 5000)
	calculator.UpdateGasPrice(0.6, 6000)
	calculator.UpdateGasPrice(0.7, 7000)
	calculator.UpdateGasPrice(0.8, 8000)
	calculator.UpdateGasPrice(0.9, 9000)

	// 获取中位数
	medianPrice := calculator.GetPricePercentile(0.5)
	assert.Greater(t, medianPrice, uint64(0))
	assert.LessOrEqual(t, medianPrice, uint64(9000))
}
