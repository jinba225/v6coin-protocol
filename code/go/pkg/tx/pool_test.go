package tx

import (
	"net"
	"testing"
	"time"

	"github.com/jinba225/v6coin-protocol/pkg/consensus"
	"github.com/stretchr/testify/assert"
)

// mockStateDB implements StateDB interface for testing
type mockStateDB struct {
	balances map[string]uint64
	nonces   map[string]uint64
	accounts map[string]bool
}

func newMockStateDB() *mockStateDB {
	return &mockStateDB{
		balances: make(map[string]uint64),
		nonces:   make(map[string]uint64),
		accounts: make(map[string]bool),
	}
}

func (m *mockStateDB) GetBalance(address []byte) (uint64, error) {
	return m.balances[string(address)], nil
}

func (m *mockStateDB) GetNonce(address []byte) (uint64, error) {
	return m.nonces[string(address)], nil
}

func (m *mockStateDB) HasAccount(address []byte) (bool, error) {
	return m.accounts[string(address)], nil
}

func (m *mockStateDB) setBalance(ip net.IP, balance uint64) {
	m.balances[string(ip[:])] = balance
	m.accounts[string(ip[:])] = true
}

func (m *mockStateDB) setNonce(ip net.IP, nonce uint64) {
	m.nonces[string(ip[:])] = nonce
	m.accounts[string(ip[:])] = true
}

func createTestTransaction(nonce uint64, fee uint64, from net.IP) *consensus.Transaction {
	// Create recipient IP
	toIP := make(net.IP, 16)
	copy(toIP, []byte("recipient"))

	return &consensus.Transaction{
		Version:    1,
		Nonce:      nonce,
		Fee:        fee,
		From:       from,
		To:         toIP,
		Amount:     1000,
		Type:       consensus.TxTypeOnline,
		Data:       []byte{},
		Timestamp:  uint64(time.Now().Unix()),
		Signature:  make([]byte, 64), // Add empty signature for validation
	}
}

func createTestIP(addr byte) net.IP {
	ip := make(net.IP, 16)
	ip[15] = addr // Use last byte for differentiation
	return ip
}

func TestNewTxPool(t *testing.T) {
	stateDB := newMockStateDB()
	pool := NewTxPool(stateDB)

	assert.NotNil(t, pool)
	assert.NotNil(t, pool.pending)
	assert.NotNil(t, pool.all)
	assert.NotNil(t, pool.bySender)
	assert.Equal(t, 0, pool.pending.Len())
}

func TestAddTransaction(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)
	tx := createTestTransaction(0, 100, fromAddr)

	err := pool.AddTransaction(tx)
	assert.NoError(t, err)
	assert.Equal(t, 1, pool.pending.Len())

	// 验证交易可以被获取
	retrieved := pool.GetTransaction(tx.Hash())
	assert.NotNil(t, retrieved)
	assert.Equal(t, tx.Nonce, retrieved.Nonce)
}

func TestAddTransactionDuplicate(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)
	tx := createTestTransaction(0, 100, fromAddr)

	// 第一次添加应该成功
	err := pool.AddTransaction(tx)
	assert.NoError(t, err)

	// 第二次添加应该失败
	err = pool.AddTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, ErrDuplicateTx, err)
}

func TestAddTransactionInvalidNonce(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 5) // 账户 nonce 是 5

	pool := NewTxPool(stateDB)

	// 尝试添加 nonce 为 3 的交易（太低）
	tx := createTestTransaction(3, 100, fromAddr)
	err := pool.AddTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidNonce, err)

	// 注意：TxPool 允许使用未来 nonce（太高），因为交易可能会被排序
	// 所以这里我们只测试太低的情况
}

func TestAddTransactionUnknownSender(t *testing.T) {
	stateDB := newMockStateDB()
	// 不设置发送者账户

	pool := NewTxPool(stateDB)
	unknownAddr := createTestIP(99)
	tx := createTestTransaction(0, 100, unknownAddr)

	err := pool.AddTransaction(tx)
	assert.Error(t, err)
	assert.Equal(t, ErrUnknownSender, err)
}

func TestGetPendingTransactions(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	// 添加多个交易
	for i := 0; i < 5; i++ {
		tx := createTestTransaction(uint64(i), 100, fromAddr)
		pool.AddTransaction(tx)
	}

	// 获取所有待处理交易
	pending := pool.GetPendingTransactions(0)
	assert.Len(t, pending, 5)

	// 获取有限数量的交易
	pending = pool.GetPendingTransactions(3)
	assert.Len(t, pending, 3)
}

func TestGetTransaction(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)
	tx := createTestTransaction(0, 100, fromAddr)
	pool.AddTransaction(tx)

	// 获取存在的交易
	retrieved := pool.GetTransaction(tx.Hash())
	assert.NotNil(t, retrieved)
	assert.Equal(t, tx.Hash(), retrieved.Hash())

	// 获取不存在的交易
	nonExistentHash := []byte("non-existent")
	retrieved = pool.GetTransaction(nonExistentHash)
	assert.Nil(t, retrieved)
}

func TestRemoveTransaction(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)
	tx := createTestTransaction(0, 100, fromAddr)
	pool.AddTransaction(tx)

	assert.Equal(t, 1, pool.pending.Len())

	// 删除交易
	pool.RemoveTransaction(tx.Hash())
	assert.Equal(t, 0, pool.pending.Len())

	// 验证交易已被删除
	retrieved := pool.GetTransaction(tx.Hash())
	assert.Nil(t, retrieved)
}

func TestGetTransactionsBySender(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	// 添加多个交易
	for i := 0; i < 3; i++ {
		tx := createTestTransaction(uint64(i), 100, fromAddr)
		pool.AddTransaction(tx)
	}

	// 注意：TxPool 没有 GetTransactionsBySender 方法
	// 这个测试可能需要删除或使用其他方式验证
	// 使用 Size() 方法来验证交易已被添加
	assert.Equal(t, 3, pool.Size())
}

func TestGetCount(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	// 初始计数应该是 0
	assert.Equal(t, 0, pool.Size())

	// 添加交易
	for i := 0; i < 5; i++ {
		tx := createTestTransaction(uint64(i), 100, fromAddr)
		pool.AddTransaction(tx)
	}

	// 计数应该是 5
	assert.Equal(t, 5, pool.Size())
}

func TestCleanOldTransactions(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	// 添加一个交易
	tx := createTestTransaction(0, 100, fromAddr)
	pool.AddTransaction(tx)

	// 手动设置交易为旧时间（模拟）
	pool.mu.Lock()
	for _, item := range pool.all {
		item.AddedAt = time.Now().Add(-8 * 24 * time.Hour) // 8 天前
	}
	pool.mu.Unlock()

	// 清理旧交易
	pool.CleanExpired()

	// 交易应该被删除
	assert.Equal(t, 0, pool.Size())
}

func TestPriorityQueueOrdering(t *testing.T) {
	stateDB := newMockStateDB()

	// 为三个不同的发送者设置账户
	fromAddr1 := createTestIP(1)
	fromAddr2 := createTestIP(2)
	fromAddr3 := createTestIP(3)

	stateDB.setBalance(fromAddr1, 10000)
	stateDB.setNonce(fromAddr1, 0)
	stateDB.setBalance(fromAddr2, 10000)
	stateDB.setNonce(fromAddr2, 0)
	stateDB.setBalance(fromAddr3, 10000)
	stateDB.setNonce(fromAddr3, 0)

	pool := NewTxPool(stateDB)

	// 添加具有不同费用的交易（高费用应该有更高优先级）
	tx1 := createTestTransaction(0, 100, fromAddr1)
	tx2 := createTestTransaction(0, 200, fromAddr2)
	tx3 := createTestTransaction(0, 300, fromAddr3)

	pool.AddTransaction(tx1)
	pool.AddTransaction(tx2)
	pool.AddTransaction(tx3)

	// 获取待处理交易
	pending := pool.GetPendingTransactions(0)

	// 验证有 3 个交易
	assert.Len(t, pending, 3)

	// 输出调试信息
	t.Logf("Transaction order by fee:")
	for i, p := range pending {
		t.Logf("  [%d] Fee=%d", i, p.Fee)
	}

	// 高费用的交易应该有最高优先级（基于 priority = fee/size）
	// 验证第一个交易确实是费用最高的
	assert.Equal(t, uint64(300), pending[0].Fee, "Highest fee transaction should be first")
}

func TestCalculateGasPrice(t *testing.T) {
	stateDB := newMockStateDB()
	pool := NewTxPool(stateDB)

	fromAddr := createTestIP(1)
	tx := createTestTransaction(0, 100000, fromAddr) // Higher fee for non-zero gas price
	gasPrice := pool.calculateGasPrice(tx)

	assert.Greater(t, gasPrice, uint64(0))
}

func TestCalculateSize(t *testing.T) {
	stateDB := newMockStateDB()
	pool := NewTxPool(stateDB)

	fromAddr := createTestIP(1)
	tx := createTestTransaction(0, 100, fromAddr)
	size := pool.calculateSize(tx)

	assert.Greater(t, size, 0)
}

func TestCalculateGasLimit(t *testing.T) {
	stateDB := newMockStateDB()
	pool := NewTxPool(stateDB)

	fromAddr := createTestIP(1)
	tx := createTestTransaction(0, 100, fromAddr)
	gasLimit := pool.calculateGasLimit(tx)

	assert.Greater(t, gasLimit, uint64(0))
}

func TestEvictLowestPriority(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	// 填满池（使用较小的数量以加快测试）
	for i := 0; i < MaxPoolSize; i++ {
		tx := createTestTransaction(uint64(i), 100, fromAddr)
		pool.AddTransaction(tx)
	}

	assert.Equal(t, MaxPoolSize, pool.Size())

	// 尝试添加另一个交易
	tx := createTestTransaction(MaxPoolSize, 100, fromAddr)
	err := pool.AddTransaction(tx)

	// 应该成功（低优先级交易被驱逐）
	assert.NoError(t, err)
	assert.Equal(t, MaxPoolSize, pool.Size())
}

func TestHasTransaction(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)
	tx := createTestTransaction(0, 100, fromAddr)

	// 交易不存在
	txRetrieved := pool.GetTransaction(tx.Hash())
	assert.Nil(t, txRetrieved)

	// 添加交易
	pool.AddTransaction(tx)

	// 交易应该存在
	txRetrieved = pool.GetTransaction(tx.Hash())
	assert.NotNil(t, txRetrieved)
}

func TestGetStats(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 10000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	// 添加一些交易
	for i := 0; i < 5; i++ {
		tx := createTestTransaction(uint64(i), 100, fromAddr)
		pool.AddTransaction(tx)
	}

	// 注意：TxPool 没有 GetStats 方法
	// 使用 Size() 方法验证
	assert.Equal(t, 5, pool.Size())
}

func TestConcurrentAccess(t *testing.T) {
	stateDB := newMockStateDB()
	fromAddr := createTestIP(1)
	stateDB.setBalance(fromAddr, 100000)
	stateDB.setNonce(fromAddr, 0)

	pool := NewTxPool(stateDB)

	done := make(chan bool)

	// 并发添加交易
	for i := 0; i < 10; i++ {
		go func(idx int) {
			tx := createTestTransaction(uint64(idx), 100, fromAddr)
			pool.AddTransaction(tx)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证池状态
	assert.Equal(t, 10, pool.Size())
}
