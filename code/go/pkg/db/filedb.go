package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileDB 基于文件系统的简单键值存储
type FileDB struct {
	path   string
	data   map[string][]byte
	mu     sync.RWMutex
	closed bool
}

// NewFileDB 创建文件数据库
func NewFileDB(path string) (KeyValueStore, error) {
	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db := &FileDB{
		path: path,
		data: make(map[string][]byte),
	}

	// 加载现有数据
	if err := db.load(); err != nil {
		return nil, fmt.Errorf("failed to load database: %w", err)
	}

	return db, nil
}

// Put 存储键值对
func (f *FileDB) Put(key, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return ErrDatabaseClosed
	}

	// 写入内存
	f.data[string(key)] = value

	// 持久化到文件
	return f.writeFile(key, value)
}

// Get 获取值
func (f *FileDB) Get(key []byte) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return nil, ErrDatabaseClosed
	}

	value, exists := f.data[string(key)]
	if !exists {
		return nil, ErrNotFound
	}

	// 返回副本
	result := make([]byte, len(value))
	copy(result, value)
	return result, nil
}

// Has 检查键是否存在
func (f *FileDB) Has(key []byte) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return false, ErrDatabaseClosed
	}

	_, exists := f.data[string(key)]
	return exists, nil
}

// Delete 删除键
func (f *FileDB) Delete(key []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return ErrDatabaseClosed
	}

	keyStr := string(key)
	delete(f.data, keyStr)

	// 删除文件
	return os.Remove(f.getFilePath(key))
}

// NewBatch 创建批量操作
func (f *FileDB) NewBatch() Batch {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return &fileDBBatch{}
	}

	return &fileDBBatch{
		db:    f,
		puts:  make(map[string][]byte),
		dels:  make(map[string]bool),
	}
}

// Write 执行批量操作
func (f *FileDB) Write(batch Batch) error {
	fdb, ok := batch.(*fileDBBatch)
	if !ok {
		return fmt.Errorf("invalid batch type")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return ErrDatabaseClosed
	}

	// 应用删除
	for keyStr := range fdb.dels {
		delete(f.data, keyStr)
		key := []byte(keyStr)
		os.Remove(f.getFilePath(key))
	}

	// 应用写入
	for keyStr, value := range fdb.puts {
		key := []byte(keyStr)
		f.data[keyStr] = value
		if err := f.writeFile(key, value); err != nil {
			return err
		}
	}

	return nil
}

// NewIterator 创建迭代器
func (f *FileDB) NewIterator(prefix []byte) Iterator {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return &fileDBIterator{}
	}

	keys := make([][]byte, 0)
	prefixStr := string(prefix)

	for keyStr := range f.data {
		if prefix == nil || len(prefix) == 0 || len(keyStr) >= len(prefixStr) && keyStr[:len(prefixStr)] == prefixStr {
			key := []byte(keyStr)
			keys = append(keys, key)
		}
	}

	// 排序
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i], keys[j]) < 0
	})

	return &fileDBIterator{
		db:   f,
		keys: keys,
		index: -1,
	}
}

// Close 关闭数据库
func (f *FileDB) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return nil
	}

	f.closed = true
	return nil
}

// Compact 压缩数据库（清理孤立的文件）
func (f *FileDB) Compact() error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return ErrDatabaseClosed
	}

	// 扫描目录，删除不在内存中的文件
	err := filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// 从文件路径提取键
		key := f.extractKeyFromPath(path)
		if key == nil {
			return nil
		}

		// 检查键是否存在于内存中
		if _, exists := f.data[string(key)]; !exists {
			// 删除孤立文件
			os.Remove(path)
		}

		return nil
	})

	return err
}

// Stat 获取数据库统计信息
func (f *FileDB) Stat() (*Stat, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return nil, ErrDatabaseClosed
	}

	stat := &Stat{
		Keys: len(f.data),
	}

	// 计算数据大小
	for _, value := range f.data {
		stat.Size += int64(len(value))
	}

	// 计算磁盘大小
	err := filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			stat.Size += info.Size()
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return stat, nil
}

// getFilePath 获取键对应的文件路径
func (f *FileDB) getFilePath(key []byte) string {
	// 使用键的哈希作为文件名
	hash := simpleHash(key)
	prefix := fmt.Sprintf("%02x", hash[0])
	filename := fmt.Sprintf("%02x%02x%02x.dat", hash[1], hash[2], hash[3])
	return filepath.Join(f.path, prefix, filename)
}

// writeFile 写入文件
func (f *FileDB) writeFile(key, value []byte) error {
	filePath := f.getFilePath(key)

	// 创建目录
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 写入文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 格式：[键长度][键][值长度][值]
	if err := binary.Write(file, binary.BigEndian, uint32(len(key))); err != nil {
		return err
	}
	if _, err := file.Write(key); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, uint32(len(value))); err != nil {
		return err
	}
	if _, err := file.Write(value); err != nil {
		return err
	}

	return nil
}

// load 加载数据
func (f *FileDB) load() error {
	return filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// 读取文件
		key, value, err := f.readFile(path)
		if err != nil {
			return err
		}

		if key != nil {
			f.data[string(key)] = value
		}

		return nil
	})
}

// readFile 读取文件
func (f *FileDB) readFile(path string) ([]byte, []byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	// 读取键长度
	var keyLen uint32
	if err := binary.Read(file, binary.BigEndian, &keyLen); err != nil {
		return nil, nil, err
	}

	// 读取键
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(file, key); err != nil {
		return nil, nil, err
	}

	// 读取值长度
	var valueLen uint32
	if err := binary.Read(file, binary.BigEndian, &valueLen); err != nil {
		return nil, nil, err
	}

	// 读取值
	value := make([]byte, valueLen)
	if _, err := io.ReadFull(file, value); err != nil {
		return nil, nil, err
	}

	return key, value, nil
}

// extractKeyFromPath 从文件路径提取键
func (f *FileDB) extractKeyFromPath(path string) []byte {
	key, _, _ := f.readFile(path)
	return key
}

// simpleHash 简单哈希函数
func simpleHash(data []byte) []byte {
	hash := make([]byte, 4)
	for i, b := range data {
		hash[i%4] ^= b
	}
	return hash
}

// fileDBBatch 文件数据库批量操作
type fileDBBatch struct {
	db   *FileDB
	puts map[string][]byte
	dels map[string]bool
}

func (b *fileDBBatch) Put(key, value []byte) error {
	b.puts[string(key)] = value
	delete(b.dels, string(key))
	return nil
}

func (b *fileDBBatch) Delete(key []byte) error {
	b.dels[string(key)] = true
	delete(b.puts, string(key))
	return nil
}

func (b *fileDBBatch) Len() int {
	return len(b.puts) + len(b.dels)
}

func (b *fileDBBatch) Write() error {
	return b.db.Write(b)
}

func (b *fileDBBatch) Reset() {
	b.puts = make(map[string][]byte)
	b.dels = make(map[string]bool)
}

// fileDBIterator 文件数据库迭代器
type fileDBIterator struct {
	db     *FileDB
	keys   [][]byte
	index  int
	closed bool
}

func (it *fileDBIterator) Next() bool {
	if it.closed {
		return false
	}
	it.index++
	return it.index < len(it.keys)
}

func (it *fileDBIterator) Prev() bool {
	if it.closed {
		return false
	}
	it.index--
	return it.index >= 0
}

func (it *fileDBIterator) Seek(key []byte) bool {
	if it.closed {
		return false
	}

	// 二分查找
	low, high := 0, len(it.keys)
	for low < high {
		mid := (low + high) / 2
		cmp := bytes.Compare(it.keys[mid], key)
		if cmp < 0 {
			low = mid + 1
		} else {
			high = mid
		}
	}

	it.index = low - 1
	return it.Next()
}

func (it *fileDBIterator) Key() []byte {
	if it.closed || it.index < 0 || it.index >= len(it.keys) {
		return nil
	}
	return it.keys[it.index]
}

func (it *fileDBIterator) Value() []byte {
	if it.closed || it.index < 0 || it.index >= len(it.keys) {
		return nil
	}

	key := it.keys[it.index]
	value, err := it.db.Get(key)
	if err != nil {
		return nil
	}
	return value
}

func (it *fileDBIterator) Release() {
	it.closed = true
}

func (it *fileDBIterator) Error() error {
	return nil
}
