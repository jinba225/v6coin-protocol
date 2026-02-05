package db

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrBatchTooLarge 批量操作过大
	ErrBatchTooLarge = errors.New("batch too large")
	// ErrIteratorClosed 迭代器已关闭
	ErrIteratorClosed = errors.New("iterator closed")
)

// BatchBuilder 批量操作构建器
type BatchBuilder struct {
	batch    Batch
	store    KeyValueStore
	size     int
	maxSize  int
 autoflush bool
}

// NewBatchBuilder 创建批量操作构建器
func NewBatchBuilder(store KeyValueStore, maxSize int, autoflush bool) *BatchBuilder {
	return &BatchBuilder{
		batch:    store.NewBatch(),
		store:    store,
		maxSize:  maxSize,
		autoflush: autoflush,
	}
}

// Put 添加键值对
func (bb *BatchBuilder) Put(key, value []byte) error {
	size := len(key) + len(value)

	// 检查是否超过批量大小限制
	if bb.maxSize > 0 && bb.size+size > bb.maxSize {
		if bb.autoflush {
			// 自动刷新
			if err := bb.Flush(); err != nil {
				return err
			}
		} else {
			return ErrBatchTooLarge
		}
	}

	if err := bb.batch.Put(key, value); err != nil {
		return err
	}

	bb.size += size
	return nil
}

// Delete 删除键
func (bb *BatchBuilder) Delete(key []byte) error {
	size := len(key)

	if bb.maxSize > 0 && bb.size+size > bb.maxSize {
		if bb.autoflush {
			if err := bb.Flush(); err != nil {
				return err
			}
		} else {
			return ErrBatchTooLarge
		}
	}

	if err := bb.batch.Delete(key); err != nil {
		return err
	}

	bb.size += size
	return nil
}

// Flush 刷新批量操作
func (bb *BatchBuilder) Flush() error {
	if bb.batch.Len() == 0 {
		return nil
	}

	if err := bb.store.Write(bb.batch); err != nil {
		return err
	}

	bb.batch = bb.store.NewBatch()
	bb.size = 0
	return nil
}

// Len 获取批量操作大小
func (bb *BatchBuilder) Len() int {
	return bb.batch.Len()
}

// Size 获取数据大小
func (bb *BatchBuilder) Size() int {
	return bb.size
}

// Reset 重置批量操作
func (bb *BatchBuilder) Reset() {
	bb.batch.Reset()
	bb.size = 0
}

// PrefixIterator 前缀迭代器辅助函数
func PrefixIterator(store KeyValueStore, prefix []byte) Iterator {
	return store.NewIterator(prefix)
}

// ForeEach 遍历所有键值对
func ForEach(store KeyValueStore, prefix []byte, fn func(key, value []byte) error) error {
	iter := PrefixIterator(store, prefix)
	defer iter.Release()

	for iter.Next() {
		if err := fn(iter.Key(), iter.Value()); err != nil {
			return err
		}
	}

	return iter.Error()
}

// Count 统计键数量
func Count(store KeyValueStore, prefix []byte) (int, error) {
	count := 0
	err := ForEach(store, prefix, func(key, value []byte) error {
		count++
		return nil
	})
	return count, err
}

// DeletePrefix 删除所有指定前缀的键
func DeletePrefix(store KeyValueStore, prefix []byte) error {
	batch := store.NewBatch()
	count := 0

	iter := PrefixIterator(store, prefix)
	defer iter.Release()

	for iter.Next() {
		if err := batch.Delete(iter.Key()); err != nil {
			return err
		}
		count++
	}

	if count > 0 {
		return store.Write(batch)
	}

	return iter.Error()
}

// BytesToUint64 字节转 uint64
func BytesToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// Uint64ToBytes uint64 转字节
func Uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// BytesToUint32 字节转 uint32
func BytesToUint32(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

// Uint32ToBytes uint32 转字节
func Uint32ToBytes(n uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return b
}

// Backup 数据库备份
type Backup struct {
	store    KeyValueStore
	batch    Batch
	started  time.Time
}

// NewBackup 创建备份
func NewBackup(store KeyValueStore) *Backup {
	return &Backup{
		store:   store,
		batch:   store.NewBatch(),
		started: time.Now(),
	}
}

// Add 添加键到备份
func (b *Backup) Add(key []byte) error {
	value, err := b.store.Get(key)
	if err != nil {
		return err
	}

	return b.batch.Put(key, value)
}

// AddPrefix 添加前缀的所有键到备份
func (b *Backup) AddPrefix(prefix []byte) error {
	iter := PrefixIterator(b.store, prefix)
	defer iter.Release()

	for iter.Next() {
		if err := b.batch.Put(iter.Key(), iter.Value()); err != nil {
			return err
		}
	}

	return iter.Error()
}

// Save 保存备份到文件
func (b *Backup) Save(target KeyValueStore) error {
	// 写入所有备份数据
	return target.Write(b.batch)
}

// Restore 从备份恢复
func Restore(target, source KeyValueStore, prefix []byte) error {
	backup := NewBackup(source)
	if err := backup.AddPrefix(prefix); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return backup.Save(target)
}

// CompactRange 压缩指定范围
func CompactRange(store KeyValueStore, from, to []byte) error {
	// 通用实现：压缩整个数据库
	return store.Compact()
}

// EstimateSize 估算数据大小
func EstimateSize(store KeyValueStore, prefix []byte) (int64, error) {
	var size int64

	err := ForEach(store, prefix, func(key, value []byte) error {
		size += int64(len(key) + len(value))
		return nil
	})

	return size, err
}
