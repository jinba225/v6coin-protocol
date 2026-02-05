package db

import (
	"errors"
)

var (
	// ErrNotFound 键不存在
	ErrNotFound = errors.New("key not found")
	// ErrDatabaseClosed 数据库已关闭
	ErrDatabaseClosed = errors.New("database closed")
)

// KeyValueStore 键值存储接口
type KeyValueStore interface {
	// 基本操作
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Has(key []byte) (bool, error)
	Delete(key []byte) error

	// 批量操作
	NewBatch() Batch
	Write(batch Batch) error

	// 迭代器
	NewIterator(prefix []byte) Iterator

	// 数据库管理
	Close() error
	Compact() error
	Stat() (*Stat, error)
}

// Batch 批量操作接口
type Batch interface {
	Put(key, value []byte) error
	Delete(key []byte) error
	Len() int
	Write() error
	Reset()
}

// Iterator 迭代器接口
type Iterator interface {
	Next() bool
	Prev() bool
	Seek(key []byte) bool
	Key() []byte
	Value() []byte
	Release()
	Error() error
}

// Stat 数据库统计信息
type Stat struct {
	// 键值对数量
	Keys int
	// 数据大小（字节）
	Size int64
	// 内存使用（字节）
	Mem int64
}

// Bucket 桶（命名空间）
type Bucket struct {
	name   string
	store  KeyValueStore
	prefix []byte
}

// NewBucket 创建桶
func NewBucket(name string, store KeyValueStore) *Bucket {
	return &Bucket{
		name:   name,
		store:  store,
		prefix: []byte(name + "/"),
	}
}

// Put 存储键值对
func (b *Bucket) Put(key, value []byte) error {
	fullKey := append(b.prefix, key...)
	return b.store.Put(fullKey, value)
}

// Get 获取值
func (b *Bucket) Get(key []byte) ([]byte, error) {
	fullKey := append(b.prefix, key...)
	return b.store.Get(fullKey)
}

// Has 检查键是否存在
func (b *Bucket) Has(key []byte) (bool, error) {
	fullKey := append(b.prefix, key...)
	return b.store.Has(fullKey)
}

// Delete 删除键
func (b *Bucket) Delete(key []byte) error {
	fullKey := append(b.prefix, key...)
	return b.store.Delete(fullKey)
}

// NewBatch 创建批量操作
func (b *Bucket) NewBatch() Batch {
	return &bucketBatch{
		bucket: b,
		batch:  b.store.NewBatch(),
	}
}

// NewIterator 创建迭代器
func (b *Bucket) NewIterator() Iterator {
	return b.store.NewIterator(b.prefix)
}

// bucketBatch 桶的批量操作
type bucketBatch struct {
	bucket *Bucket
	batch  Batch
}

func (bb *bucketBatch) Put(key, value []byte) error {
	fullKey := append(bb.bucket.prefix, key...)
	return bb.batch.Put(fullKey, value)
}

func (bb *bucketBatch) Delete(key []byte) error {
	fullKey := append(bb.bucket.prefix, key...)
	return bb.batch.Delete(fullKey)
}

func (bb *bucketBatch) Len() int {
	return bb.batch.Len()
}

func (bb *bucketBatch) Write() error {
	return bb.batch.Write()
}

func (bb *bucketBatch) Reset() {
	bb.batch.Reset()
}
