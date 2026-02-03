package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryStateDB(t *testing.T) {
	db := NewMemoryStateDB()
	assert.NotNil(t, db)
	assert.NotNil(t, db.accounts)
	assert.NotNil(t, db.rootHash)
}

func TestGetAccountNonExistent(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	account, err := db.GetAccount(address)
	assert.NoError(t, err)
	assert.Nil(t, account) // 不存在的账户应该返回 nil，不是错误
}

func TestSetAndGetAccount(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 创建一个新账户
	account := NewAccount()
	account.Balance = 1000
	account.Nonce = 5

	// 设置账户
	err := db.SetAccount(address, account)
	assert.NoError(t, err)

	// 获取账户
	retrieved, err := db.GetAccount(address)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, uint64(1000), retrieved.Balance)
	assert.Equal(t, uint64(5), retrieved.Nonce)

	// 确保是副本，不是同一个对象
	retrieved.Balance = 2000
	original, _ := db.GetAccount(address)
	assert.Equal(t, uint64(1000), original.Balance)
}

func TestSetNilAccount(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 先设置一个账户
	account := NewAccount()
	account.Balance = 1000
	db.SetAccount(address, account)

	// 验证账户存在
	retrieved, _ := db.GetAccount(address)
	assert.NotNil(t, retrieved)

	// 设置 nil 应该删除账户
	err := db.SetAccount(address, nil)
	assert.NoError(t, err)

	// 验证账户已被删除
	retrieved, err = db.GetAccount(address)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestGetBalance(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 不存在的账户余额应该是 0
	balance, err := db.GetBalance(address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), balance)

	// 设置账户余额
	account := NewAccount()
	account.Balance = 5000
	db.SetAccount(address, account)

	// 获取余额
	balance, err = db.GetBalance(address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5000), balance)
}

func TestGetNonce(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 不存在的账户 nonce 应该是 0
	nonce, err := db.GetNonce(address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), nonce)

	// 设置账户 nonce
	account := NewAccount()
	account.Nonce = 10
	db.SetAccount(address, account)

	// 获取 nonce
	nonce, err = db.GetNonce(address)
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), nonce)
}

func TestHasAccount(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 不存在的账户
	has, err := db.HasAccount(address)
	assert.NoError(t, err)
	assert.False(t, has)

	// 设置账户
	account := NewAccount()
	db.SetAccount(address, account)

	// 检查账户是否存在
	has, err = db.HasAccount(address)
	assert.NoError(t, err)
	assert.True(t, has)
}

func TestSetBalance(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 设置余额
	account := NewAccount()
	account.Balance = 1000
	err := db.SetAccount(address, account)
	assert.NoError(t, err)

	// 获取余额
	balance, _ := db.GetBalance(address)
	assert.Equal(t, uint64(1000), balance)
}

func TestAddBalance(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 添加余额到不存在的账户
	account := NewAccount()
	account.AddBalance(500)
	db.SetAccount(address, account)

	balance, _ := db.GetBalance(address)
	assert.Equal(t, uint64(500), balance)

	// 添加更多余额
	account, _ = db.GetAccount(address)
	account.AddBalance(300)
	db.SetAccount(address, account)

	balance, _ = db.GetBalance(address)
	assert.Equal(t, uint64(800), balance)
}

func TestSubBalance(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 先设置余额
	account := NewAccount()
	account.Balance = 1000
	db.SetAccount(address, account)

	// 减少余额
	account, _ = db.GetAccount(address)
	err := account.SubBalance(400)
	assert.NoError(t, err)
	db.SetAccount(address, account)

	balance, _ := db.GetBalance(address)
	assert.Equal(t, uint64(600), balance)
}

func TestSubBalanceInsufficient(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 设置余额
	account := NewAccount()
	account.Balance = 100
	db.SetAccount(address, account)

	// 尝试减少超过余额的金额
	account, _ = db.GetAccount(address)
	err := account.SubBalance(200)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientBalance, err)

	// 余额应该保持不变
	balance, _ := db.GetBalance(address)
	assert.Equal(t, uint64(100), balance)
}

func TestSetNonce(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 设置 nonce
	account := NewAccount()
	account.Nonce = 7
	err := db.SetAccount(address, account)
	assert.NoError(t, err)

	nonce, _ := db.GetNonce(address)
	assert.Equal(t, uint64(7), nonce)
}

func TestIncrementNonce(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 增加不存在的账户的 nonce
	account := NewAccount()
	newNonce := account.IncrementNonce()
	db.SetAccount(address, account)
	assert.Equal(t, uint64(1), newNonce)

	// 再次增加
	account, _ = db.GetAccount(address)
	newNonce = account.IncrementNonce()
	db.SetAccount(address, account)
	assert.Equal(t, uint64(2), newNonce)

	// 验证
	nonce, _ := db.GetNonce(address)
	assert.Equal(t, uint64(2), nonce)
}

func TestCurrentRoot(t *testing.T) {
	db := NewMemoryStateDB()

	root := db.CurrentRoot()
	assert.NotNil(t, root)
	assert.Len(t, root, 32)
}

func TestCommit(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 设置一些状态
	account := NewAccount()
	account.Balance = 1000
	db.SetAccount(address, account)

	// 提交状态
	root, err := db.Commit()
	assert.NoError(t, err)
	assert.NotNil(t, root)
	assert.Len(t, root, 32)

	// 状态应该保持不变
	balance, _ := db.GetBalance(address)
	assert.Equal(t, uint64(1000), balance)
}

func TestClose(t *testing.T) {
	db := NewMemoryStateDB()

	err := db.Close()
	assert.NoError(t, err)
}

func TestAccountNewAccount(t *testing.T) {
	account := NewAccount()

	assert.NotNil(t, account)
	assert.Equal(t, uint64(0), account.Nonce)
	assert.Equal(t, uint64(0), account.Balance)
	assert.Empty(t, account.CodeHash)
	assert.Empty(t, account.StorageRoot)
	assert.False(t, account.LastUpdated.IsZero())
}

func TestAccountCopy(t *testing.T) {
	original := NewAccount()
	original.Balance = 1000
	original.Nonce = 5
	original.CodeHash = []byte("code-hash")
	original.StorageRoot = []byte("storage-root")

	copy := original.Copy()

	assert.Equal(t, original.Balance, copy.Balance)
	assert.Equal(t, original.Nonce, copy.Nonce)
	assert.Equal(t, original.CodeHash, copy.CodeHash)
	assert.Equal(t, original.StorageRoot, copy.StorageRoot)

	// 修改副本不应该影响原始对象
	copy.Balance = 2000
	assert.Equal(t, uint64(1000), original.Balance)
}

func TestAccountCopyNil(t *testing.T) {
	var account *Account = nil
	copy := account.Copy()
	assert.Nil(t, copy)
}

func TestAccountIsEmpty(t *testing.T) {
	// nil 账户
	var account *Account = nil
	assert.True(t, account.IsEmpty())

	// 新账户
	newAccount := NewAccount()
	assert.True(t, newAccount.IsEmpty())

	// 有余额的账户
	accountWithBalance := NewAccount()
	accountWithBalance.Balance = 100
	assert.False(t, accountWithBalance.IsEmpty())

	// 有 nonce 的账户
	accountWithNonce := NewAccount()
	accountWithNonce.Nonce = 1
	assert.False(t, accountWithNonce.IsEmpty())
}

func TestAccountHasCode(t *testing.T) {
	account := NewAccount()

	// 新账户没有代码
	assert.False(t, account.HasCode())

	// 设置代码哈希
	account.CodeHash = []byte("some-code")
	assert.True(t, account.HasCode())

	// 零哈希不算有代码
	account.CodeHash = make([]byte, 32)
	assert.False(t, account.HasCode())
}

func TestAccountHasStorage(t *testing.T) {
	account := NewAccount()

	// 新账户没有存储
	assert.False(t, account.HasStorage())

	// 设置存储根
	account.StorageRoot = []byte("some-storage")
	assert.True(t, account.HasStorage())

	// 零哈希不算有存储
	account.StorageRoot = make([]byte, 32)
	assert.False(t, account.HasStorage())
}

func TestAccountAddBalance(t *testing.T) {
	account := NewAccount()
	initialTime := account.LastUpdated

	account.AddBalance(100)
	assert.Equal(t, uint64(100), account.Balance)

	account.AddBalance(50)
	assert.Equal(t, uint64(150), account.Balance)

	// LastUpdated 应该被更新
	assert.True(t, account.LastUpdated.After(initialTime))
}

func TestAccountSubBalance(t *testing.T) {
	account := NewAccount()
	account.Balance = 100

	err := account.SubBalance(40)
	assert.NoError(t, err)
	assert.Equal(t, uint64(60), account.Balance)
}

func TestAccountSubBalanceInsufficient(t *testing.T) {
	account := NewAccount()
	account.Balance = 50

	err := account.SubBalance(100)
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientBalance, err)
	assert.Equal(t, uint64(50), account.Balance) // 余额不应该改变
}

func TestAccountIncrementNonce(t *testing.T) {
	account := NewAccount()
	initialTime := account.LastUpdated

	newNonce := account.IncrementNonce()
	assert.Equal(t, uint64(1), newNonce)
	assert.Equal(t, uint64(1), account.Nonce)

	newNonce = account.IncrementNonce()
	assert.Equal(t, uint64(2), newNonce)
	assert.Equal(t, uint64(2), account.Nonce)

	// LastUpdated 应该被更新
	assert.True(t, account.LastUpdated.After(initialTime))
}

func TestConcurrentAccess(t *testing.T) {
	db := NewMemoryStateDB()
	address := []byte("test-address")

	// 并发写入
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			account := NewAccount()
			account.Balance = uint64(idx * 100)
			db.SetAccount(address, account)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证状态一致性
	account, err := db.GetAccount(address)
	assert.NoError(t, err)
	assert.NotNil(t, account)
}
