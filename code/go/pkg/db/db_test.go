package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试辅助函数
func createTestDB(t *testing.T) KeyValueStore {
	dir := filepath.Join(os.TempDir(), "v6coin-test-db")
	// 清理旧的测试数据
	os.RemoveAll(dir)

	db, err := NewFileDB(dir)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	return db
}

func cleanupTestDB(t *testing.T, db KeyValueStore) {
	db.Close()
	dir := filepath.Join(os.TempDir(), "v6coin-test-db")
	os.RemoveAll(dir)
}

// ==================== 基本操作测试 ====================

func TestFileDB_PutAndGet(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	key := []byte("test-key")
	value := []byte("test-value")

	// Put
	err := db.Put(key, value)
	assert.NoError(t, err)

	// Get
	retrieved, err := db.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestLevelDB_GetNotFound(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	key := []byte("non-existent-key")

	value, err := db.Get(key)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, value)
}

func TestLevelDB_Has(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	key := []byte("test-key")
	value := []byte("test-value")

	// 不存在
	exists, err := db.Has(key)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Put 后存在
	db.Put(key, value)
	exists, err = db.Has(key)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestLevelDB_Delete(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	key := []byte("test-key")
	value := []byte("test-value")

	// Put
	db.Put(key, value)

	// Delete
	err := db.Delete(key)
	assert.NoError(t, err)

	// 验证删除
	exists, _ := db.Has(key)
	assert.False(t, exists)
}

func TestLevelDB_UpdateValue(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	key := []byte("test-key")
	value1 := []byte("value1")
	value2 := []byte("value2")

	// Put 第一次
	db.Put(key, value1)
	retrieved, _ := db.Get(key)
	assert.Equal(t, value1, retrieved)

	// Put 第二次（更新）
	db.Put(key, value2)
	retrieved, _ = db.Get(key)
	assert.Equal(t, value2, retrieved)
}

// ==================== 批量操作测试 ====================

func TestLevelDB_BatchWrite(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	batch := db.NewBatch()

	// 添加多个操作
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2)}
		batch.Put(key, value)
	}

	// 写入
	err := db.Write(batch)
	assert.NoError(t, err)

	// 验证
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value, err := db.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, []byte{byte(i * 2)}, value)
	}
}

func TestLevelDB_BatchDelete(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	// 先写入数据
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2)}
		db.Put(key, value)
	}

	// 批量删除
	batch := db.NewBatch()
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		batch.Delete(key)
	}

	err := db.Write(batch)
	assert.NoError(t, err)

	// 验证删除
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		exists, _ := db.Has(key)
		assert.False(t, exists)
	}

	// 验证保留
	for i := 5; i < 10; i++ {
		key := []byte{byte(i)}
		exists, _ := db.Has(key)
		assert.True(t, exists)
	}
}

func TestLevelDB_BatchReset(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	batch := db.NewBatch()

	// 添加操作
	key := []byte("test-key")
	value := []byte("test-value")
	batch.Put(key, value)

	assert.Equal(t, 1, batch.Len())

	// 重置
	batch.Reset()
	assert.Equal(t, 0, batch.Len())
}

// ==================== 迭代器测试 ====================

func TestLevelDB_Iterator(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	// 写入数据
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2)}
		db.Put(key, value)
	}

	// 迭代
	iter := db.NewIterator(nil)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
		assert.NotNil(t, iter.Key())
		assert.NotNil(t, iter.Value())
	}

	assert.Equal(t, 10, count)
	assert.NoError(t, iter.Error())
}

func TestLevelDB_PrefixIterator(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	// 写入不同前缀的数据
	prefix1 := []byte{0x01}
	prefix2 := []byte{0x02}

	for i := 0; i < 5; i++ {
		key1 := append([]byte{0x01}, byte(i))
		key2 := append([]byte{0x02}, byte(i))
		db.Put(key1, []byte{byte(i)})
		db.Put(key2, []byte{byte(i + 10)})
	}
	_ = prefix2 // 使用 prefix2

	// 迭代前缀1
	iter := db.NewIterator(prefix1)
	defer iter.Release()

	count := 0
	for iter.Next() {
		count++
		assert.Equal(t, byte(0x01), iter.Key()[0])
	}

	assert.Equal(t, 5, count)
}

// ==================== 桶测试 ====================

func TestBucket(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	bucket := NewBucket("test", db)

	key := []byte("key")
	value := []byte("value")

	// Put
	err := bucket.Put(key, value)
	assert.NoError(t, err)

	// Get（直接从数据库读取，验证前缀）
	fullKey := []byte("test/" + string(key))
	retrieved, err := db.Get(fullKey)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Get 从桶
	retrieved, err = bucket.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

// ==================== 缓存测试 ====================

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(3)

	key1 := []byte("key1")
	value1 := []byte("value1")
	key2 := []byte("key2")
	value2 := []byte("value2")
	key3 := []byte("key3")
	value3 := []byte("value3")
	key4 := []byte("key4")
	value4 := []byte("value4")

	// 添加 3 个项
	cache.Set(key1, value1)
	cache.Set(key2, value2)
	cache.Set(key3, value3)

	assert.Equal(t, 3, cache.Len())

	// 添加第 4 个项（应该淘汰 key1，最旧的）
	cache.Set(key4, value4)

	assert.Equal(t, 3, cache.Len())

	// key1 应该被淘汰
	_, found := cache.Get(key1)
	assert.False(t, found)

	// key4 应该存在
	val, found := cache.Get(key4)
	assert.True(t, found)
	assert.Equal(t, value4, val)
}

func TestLRUCacheExpiry(t *testing.T) {
	cache := NewLRUCache(10)

	key := []byte("key")
	value := []byte("value")

	// 设置 100ms 过期时间
	cache.SetWithExpiry(key, value, 100*time.Millisecond)

	// 立即获取
	val, found := cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, value, val)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 应该已过期
	val, found = cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestLRUCacheDelete(t *testing.T) {
	cache := NewLRUCache(10)

	key := []byte("key")
	value := []byte("value")

	cache.Set(key, value)

	// 删除
	cache.Delete(key)

	// 验证
	_, found := cache.Get(key)
	assert.False(t, found)
}

func TestLRUCacheClear(t *testing.T) {
	cache := NewLRUCache(10)

	// 添加多个项
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2)}
		cache.Set(key, value)
	}

	assert.Equal(t, 5, cache.Len())

	// 清空
	cache.Clear()

	assert.Equal(t, 0, cache.Len())
}

func TestCachedDatabase(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	cachedDB := NewCachedDatabase(db, 100)

	key := []byte("test-key")
	value := []byte("test-value")

	// Put
	err := cachedDB.Put(key, value)
	assert.NoError(t, err)

	// Get 第一次（从数据库）
	retrieved, err := cachedDB.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Get 第二次（从缓存）
	retrieved, err = cachedDB.Get(key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Has
	exists, err := cachedDB.Has(key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Delete
	err = cachedDB.Delete(key)
	assert.NoError(t, err)

	// 验证删除
	exists, _ = cachedDB.Has(key)
	assert.False(t, exists)
}

// ==================== 工具函数测试 ====================

func TestBatchBuilder(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	batcher := NewBatchBuilder(db, 1000, true)

	// 添加多个操作
	for i := 0; i < 100; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i * 2)}
		err := batcher.Put(key, value)
		assert.NoError(t, err)
	}

	// 刷新
	err := batcher.Flush()
	assert.NoError(t, err)

	// 验证
	for i := 0; i < 100; i++ {
		key := []byte{byte(i)}
		exists, _ := db.Has(key)
		assert.True(t, exists)
	}
}

func TestForEach(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	// 写入数据
	for i := 0; i < 10; i++ {
		key := append([]byte("prefix-"), byte(i))
		value := []byte{byte(i * 2)}
		db.Put(key, value)
	}

	// 遍历
	prefix := []byte("prefix-")
	count := 0
	err := ForEach(db, prefix, func(key, value []byte) error {
		count++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestCount(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	// 写入数据
	for i := 0; i < 10; i++ {
		key := append([]byte("prefix-"), byte(i))
		value := []byte{byte(i * 2)}
		db.Put(key, value)
	}

	// 计数
	prefix := []byte("prefix-")
	count, err := Count(db, prefix)

	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestBytesToUint64(t *testing.T) {
	tests := []struct {
		n     uint64
		bytes []byte
	}{
		{0, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{1, []byte{0, 0, 0, 0, 0, 0, 0, 1}},
		{256, []byte{0, 0, 0, 0, 0, 0, 1, 0}},
		{0xFFFFFFFFFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.n, BytesToUint64(tt.bytes))
			assert.Equal(t, tt.bytes, Uint64ToBytes(tt.n))
		})
	}
}

func TestBytesToUint32(t *testing.T) {
	tests := []struct {
		n     uint32
		bytes []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{1, []byte{0, 0, 0, 1}},
		{256, []byte{0, 0, 1, 0}},
		{0xFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.n, BytesToUint32(tt.bytes))
			assert.Equal(t, tt.bytes, Uint32ToBytes(tt.n))
		})
	}
}

func TestDeletePrefix(t *testing.T) {
	db := createTestDB(t)
	defer cleanupTestDB(t, db)

	// 写入不同前缀的数据
	prefix1 := []byte("prefix1-")
	prefix2 := []byte("prefix2-")

	for i := 0; i < 10; i++ {
		key1 := append(prefix1, byte(i))
		key2 := append(prefix2, byte(i))
		db.Put(key1, []byte{byte(i)})
		db.Put(key2, []byte{byte(i + 10)})
	}

	// 删除 prefix1
	err := DeletePrefix(db, prefix1)
	assert.NoError(t, err)

	// 验证 prefix1 已删除
	count1, _ := Count(db, prefix1)
	assert.Equal(t, 0, count1)

	// 验证 prefix2 仍存在
	count2, _ := Count(db, prefix2)
	assert.Equal(t, 10, count2)
}
