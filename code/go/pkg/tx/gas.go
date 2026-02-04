package tx

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
)

var (
	// ErrGasLimitTooLow Gas 限制过低
	ErrGasLimitTooLow = fmt.Errorf("gas limit too low")
	// ErrGasPriceTooLow Gas 价格过低
	ErrGasPriceTooLow = fmt.Errorf("gas price too low")
	// ErrInsufficientGasFee Gas 费用不足
	ErrInsufficientGasFee = fmt.Errorf("insufficient gas fee")
)

// GasConfig Gas 配置参数
type GasConfig struct {
	// 基础 Gas 费用（单位：nano-V6）
	BaseGasPrice uint64 // 当前基础 Gas 价格（动态调整）
	MinGasPrice  uint64 // 最小 Gas 价格

	// Gas 限制
	MinGasLimit      uint64 // 最小 Gas 限制
	MaxGasLimit      uint64 // 最大 Gas 限制
	BlockGasLimit    uint64 // 区块 Gas 限制

	// 复杂度计算参数
	GasPerByte       uint64 // 每字节数据的 Gas
	GasPerSignature  uint64 // 每个签名的 Gas
	GasPerContract   uint64 // 合约调用的 Gas
	GasPerStorage    uint64 // 存储操作的 Gas
	GasPerLog        uint64 // 日志操作的 Gas

	// 动态调整参数
	PriceAdjustmentFactor float64 // 价格调整因子（0-1）
	BlockTimeTarget       time.Duration // 目标出块时间
}

// DefaultGasConfig 默认 Gas 配置
func DefaultGasConfig() *GasConfig {
	return &GasConfig{
		BaseGasPrice:          1,        // 1 nano-V6 per gas
		MinGasPrice:           1,
		MinGasLimit:           21000,
		MaxGasLimit:           10000000,
		BlockGasLimit:         100000000,
		GasPerByte:            2,
		GasPerSignature:       5000,
		GasPerContract:        10000,
		GasPerStorage:         20000,
		GasPerLog:             375,
		PriceAdjustmentFactor: 0.1,
		BlockTimeTarget:       10 * time.Second,
	}
}

// GasCalculator Gas 计算器
type GasCalculator struct {
	config      *GasConfig
	currentPrice uint64
	mu          sync.RWMutex

	// 价格历史（用于动态调整）
	priceHistory []uint64
	maxHistory   int
}

// NewGasCalculator 创建 Gas 计算器
func NewGasCalculator(config *GasConfig) *GasCalculator {
	if config == nil {
		config = DefaultGasConfig()
	}

	return &GasCalculator{
		config:       config,
		currentPrice: config.BaseGasPrice,
		priceHistory: make([]uint64, 0, 100),
		maxHistory:   100,
	}
}

// CalculateGas 计算 Gas 费用
// 返回：Gas 限制、Gas 价格、总费用、错误
func (gc *GasCalculator) CalculateGas(tx *consensus.Transaction) (gasLimit uint64, gasPrice uint64, totalFee uint64, err error) {
	gc.mu.RLock()
	gasPrice = gc.currentPrice
	gc.mu.RUnlock()

	// 计算 Gas 限制
	gasLimit = gc.CalculateGasLimit(tx)

	// 验证 Gas 限制
	if gasLimit < gc.config.MinGasLimit {
		return 0, 0, 0, ErrGasLimitTooLow
	}

	// 计算总费用
	totalFee = gasLimit * gasPrice

	return gasLimit, gasPrice, totalFee, nil
}

// CalculateGasLimit 计算 Gas 限制
func (gc *GasCalculator) CalculateGasLimit(tx *consensus.Transaction) uint64 {
	// 基础 Gas（固定开销）
	baseGas := uint64(21000)

	// 数据 Gas
	dataGas := gc.calculateDataGas(tx.Data)

	// 签名 Gas
	sigGas := gc.config.GasPerSignature

	// 交易类型特定 Gas
	typeGas := gc.calculateTypeGas(tx)

	// 总 Gas
	totalGas := baseGas + dataGas + sigGas + typeGas

	// 应用上限
	if totalGas > gc.config.MaxGasLimit {
		totalGas = gc.config.MaxGasLimit
	}

	return totalGas
}

// calculateDataGas 计算数据 Gas
func (gc *GasCalculator) calculateDataGas(data []byte) uint64 {
	// 零字节（0x00）成本较低，非零字节成本较高
	// 这是 EVM 兼容的设计
	zeroGas := uint64(4)
	nonZeroGas := uint64(68)

	zeroCount := 0
	nonZeroCount := 0

	for _, b := range data {
		if b == 0 {
			zeroCount++
		} else {
			nonZeroCount++
		}
	}

	return uint64(zeroCount)*zeroGas + uint64(nonZeroCount)*nonZeroGas
}

// calculateTypeGas 计算交易类型特定 Gas
func (gc *GasCalculator) calculateTypeGas(tx *consensus.Transaction) uint64 {
	switch tx.Type {
	case consensus.TxTypeOnline:
		// 在线交易：嵌入 IPv6 报头，成本较低
		return 5000

	case consensus.TxTypeOffline:
		// 离线交易：需要存储，成本适中
		return 10000

	case consensus.TxTypeMigration:
		// 资产迁移：需要验证旧链，成本较高
		return 40000

	case consensus.TxTypeStake:
		// 质押：更新质押状态
		return 50000

	case consensus.TxTypeUnstake:
		// 解押：更新质押状态，加入解押队列
		return 50000

	case consensus.TxTypeGovernance:
		// 治理提案：需要存储和广播
		return 100000

	case consensus.TxTypeVote:
		// 投票：更新投票状态
		return 5000

	default:
		// 未知类型：使用默认值
		return 10000
	}
}

// EstimateGasPrice 估算 Gas 价格
// 考虑网络拥堵程度和交易优先级
func (gc *GasCalculator) EstimateGasPrice(priority float64) uint64 {
	gc.mu.RLock()
	basePrice := gc.currentPrice
	gc.mu.RUnlock()

	// 根据优先级调整价格
	// priority: 0.0 (最低) - 1.0 (最高)
	multiplier := 1.0 + priority*0.5 // 最高 1.5 倍

	estimatedPrice := uint64(float64(basePrice) * multiplier)

	// 确保不低于最小价格
	if estimatedPrice < gc.config.MinGasPrice {
		estimatedPrice = gc.config.MinGasPrice
	}

	return estimatedPrice
}

// GetCurrentPrice 获取当前 Gas 价格
func (gc *GasCalculator) GetCurrentPrice() uint64 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return gc.currentPrice
}

// UpdateGasPrice 更新 Gas 价格（动态调整）
// based on: 区块满载程度、网络拥堵、交易池大小
func (gc *GasCalculator) UpdateGasPrice(blockUtilization float64, poolSize int) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	// 记录历史
	gc.priceHistory = append(gc.priceHistory, gc.currentPrice)
	if len(gc.priceHistory) > gc.maxHistory {
		gc.priceHistory = gc.priceHistory[1:]
	}

	// 计算目标价格
	// blockUtilization: 0.0 (空) - 1.0 (满)
	targetPrice := gc.calculateTargetPrice(blockUtilization, poolSize)

	// 平滑调整（避免价格剧烈波动）
	factor := gc.config.PriceAdjustmentFactor
	newPrice := uint64(float64(gc.currentPrice)*(1-factor) + float64(targetPrice)*factor)

	// 更新价格
	gc.currentPrice = newPrice

	// 确保不低于最小价格
	if gc.currentPrice < gc.config.MinGasPrice {
		gc.currentPrice = gc.config.MinGasPrice
	}
}

// calculateTargetPrice 计算目标 Gas 价格
func (gc *GasCalculator) calculateTargetPrice(blockUtilization float64, poolSize int) uint64 {
	// 基于区块利用率
	basePrice := float64(gc.currentPrice)

	// 区块满载程度影响
	// 如果区块利用率 > 80%，提高价格
	// 如果区块利用率 < 20%，降低价格
	var utilizationFactor float64
	if blockUtilization > 0.8 {
		// 拥堵：提高价格
		utilizationFactor = 1.0 + (blockUtilization-0.8)*2.5 // 最多 1.5 倍
	} else if blockUtilization < 0.2 {
		// 空闲：降低价格
		utilizationFactor = 0.5 + (blockUtilization/0.2)*0.5 // 最低 0.5 倍
	} else {
		utilizationFactor = 1.0
	}

	// 交易池大小影响
	// 如果交易池很大（> 5000），适度提高价格
	var poolFactor float64
	if poolSize > 5000 {
		poolFactor = 1.0 + float64(poolSize-5000)/10000.0 // 最多 1.5 倍
	} else {
		poolFactor = 1.0
	}

	targetPrice := uint64(basePrice * utilizationFactor * poolFactor)

	// 确保不低于最小价格
	if targetPrice < gc.config.MinGasPrice {
		targetPrice = gc.config.MinGasPrice
	}

	return targetPrice
}

// RefundGas 计算 Gas 退款
// 返回：实际使用的 Gas、退款金额
func (gc *GasCalculator) RefundGas(gasLimit uint64, gasUsed uint64, gasPrice uint64) (actualUsed uint64, refund uint64) {
	// 实际使用的 Gas 不能超过限制
	if gasUsed > gasLimit {
		gasUsed = gasLimit
	}

	// 计算退款
	refundGas := gasLimit - gasUsed
	refund = refundGas * gasPrice

	return gasUsed, refund
}

// ValidateGasFee 验证 Gas 费用是否足够
func (gc *GasCalculator) ValidateGasFee(tx *consensus.Transaction, providedFee uint64) error {
	// 计算所需费用
	_, _, requiredFee, err := gc.CalculateGas(tx)
	if err != nil {
		return err
	}

	// 检查费用是否足够
	if providedFee < requiredFee {
		return fmt.Errorf("%w: provided=%d, required=%d", ErrInsufficientGasFee, providedFee, requiredFee)
	}

	return nil
}

// GetPriceHistory 获取价格历史
func (gc *GasCalculator) GetPriceHistory() []uint64 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	history := make([]uint64, len(gc.priceHistory))
	copy(history, gc.priceHistory)
	return history
}

// GetAveragePrice 获取平均价格
func (gc *GasCalculator) GetAveragePrice() uint64 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	if len(gc.priceHistory) == 0 {
		return gc.currentPrice
	}

	sum := uint64(0)
	for _, price := range gc.priceHistory {
		sum += price
	}

	return sum / uint64(len(gc.priceHistory))
}

// GetPricePercentile 获取价格百分位数
func (gc *GasCalculator) GetPricePercentile(percentile float64) uint64 {
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	if len(gc.priceHistory) == 0 {
		return gc.currentPrice
	}

	// 复制并排序
	sorted := make([]uint64, len(gc.priceHistory))
	copy(sorted, gc.priceHistory)

	// 简单冒泡排序（用于小数据集）
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// 计算百分位数
	index := int(math.Floor(float64(len(sorted)) * percentile))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// CalculateTransactionComplexity 计算交易复杂度（用于优先级排序）
func (gc *GasCalculator) CalculateTransactionComplexity(tx *consensus.Transaction) float64 {
	// 复杂度因素：
	// 1. 数据大小
	dataComplexity := float64(len(tx.Data)) / 1000.0

	// 2. 交易类型
	typeComplexity := 0.0
	switch tx.Type {
	case consensus.TxTypeGovernance:
		typeComplexity = 2.0
	case consensus.TxTypeMigration:
		typeComplexity = 1.5
	case consensus.TxTypeStake, consensus.TxTypeUnstake:
		typeComplexity = 1.0
	case consensus.TxTypeOffline:
		typeComplexity = 0.5
	case consensus.TxTypeOnline:
		typeComplexity = 0.1
	}

	// 3. 金额（大额交易优先级更高）
	amountComplexity := math.Log1p(float64(tx.Amount) / 1e9) // Log(1 + amount in V6)

	return dataComplexity + typeComplexity + amountComplexity
}

// Reset 重置计算器（用于测试）
func (gc *GasCalculator) Reset() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.currentPrice = gc.config.BaseGasPrice
	gc.priceHistory = make([]uint64, 0, gc.maxHistory)
}
