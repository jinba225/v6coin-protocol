package db

import (
	"context"
	"sync"
	"time"
)

// Cache 缓存接口
type Cache interface {
	Get(key []byte) ([]byte, bool)
	Set(key, value []byte)
	Delete(key []byte)
	Clear()
	Len() int
}

// LRUCache LRU 缓存实现
type LRUCache struct {
	maxSize int
	items   map[string]*cacheItem
	lru     []string
	mu      sync.RWMutex
}

type cacheItem struct {
	key   string
	value []byte
	expiry time.Time
}

// NewLRUCache 创建 LRU 缓存
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		maxSize: maxSize,
		items:   make(map[string]*cacheItem),
		lru:     make([]string, 0, maxSize),
	}
}

// Get 获取缓存
func (c *LRUCache) Get(key []byte) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyStr := string(key)
	item, exists := c.items[keyStr]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if !item.expiry.IsZero() && time.Now().After(item.expiry) {
		c.removeItem(keyStr)
		return nil, false
	}

	// 更新 LRU 位置
	c.updateLRU(keyStr)

	return item.value, true
}

// Set 设置缓存
func (c *LRUCache) Set(key, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyStr := string(key)

	// 检查是否已存在
	if _, exists := c.items[keyStr]; exists {
		c.items[keyStr].value = value
		c.updateLRU(keyStr)
		return
	}

	// 检查是否需要淘汰
	if len(c.items) >= c.maxSize {
		// 淘汰最旧的项
		if len(c.lru) > 0 {
			oldest := c.lru[0]
			c.removeItem(oldest)
		}
	}

	// 添加新项
	c.items[keyStr] = &cacheItem{
		key:   keyStr,
		value: value,
	}
	c.lru = append(c.lru, keyStr)
}

// SetWithExpiry 设置带过期时间的缓存
func (c *LRUCache) SetWithExpiry(key, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyStr := string(key)
	expiry := time.Now().Add(ttl)

	// 检查是否已存在
	if _, exists := c.items[keyStr]; exists {
		c.items[keyStr].value = value
		c.items[keyStr].expiry = expiry
		c.updateLRU(keyStr)
		return
	}

	// 检查是否需要淘汰
	if len(c.items) >= c.maxSize {
		if len(c.lru) > 0 {
			oldest := c.lru[0]
			c.removeItem(oldest)
		}
	}

	c.items[keyStr] = &cacheItem{
		key:   keyStr,
		value: value,
		expiry: expiry,
	}
	c.lru = append(c.lru, keyStr)
}

// Delete 删除缓存
func (c *LRUCache) Delete(key []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyStr := string(key)
	c.removeItem(keyStr)
}

// Clear 清空缓存
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
	c.lru = make([]string, 0, c.maxSize)
}

// Len 获取缓存大小
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// removeItem 移除项
func (c *LRUCache) removeItem(key string) {
	delete(c.items, key)
	for i, k := range c.lru {
		if k == key {
			c.lru = append(c.lru[:i], c.lru[i+1:]...)
			break
		}
	}
}

// updateLRU 更新 LRU 位置
func (c *LRUCache) updateLRU(key string) {
	for i, k := range c.lru {
		if k == key {
			c.lru = append(c.lru[:i], c.lru[i+1:]...)
			c.lru = append(c.lru, key)
			return
		}
	}
}

// CachedDatabase 带缓存的数据库
type CachedDatabase struct {
	store   KeyValueStore
	cache   Cache
	mu      sync.RWMutex
}

// NewCachedDatabase 创建带缓存的数据库
func NewCachedDatabase(store KeyValueStore, cacheSize int) *CachedDatabase {
	return &CachedDatabase{
		store: store,
		cache: NewLRUCache(cacheSize),
	}
}

// Put 存储键值对（同时更新缓存）
func (cd *CachedDatabase) Put(key, value []byte) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	if err := cd.store.Put(key, value); err != nil {
		return err
	}

	cd.cache.Set(key, value)
	return nil
}

// Get 获取值（先查缓存）
func (cd *CachedDatabase) Get(key []byte) ([]byte, error) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	// 先查缓存
	if value, found := cd.cache.Get(key); found {
		return value, nil
	}

	// 缓存未命中，查数据库
	value, err := cd.store.Get(key)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	cd.cache.Set(key, value)
	return value, nil
}

// Has 检查键是否存在（先查缓存）
func (cd *CachedDatabase) Has(key []byte) (bool, error) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	// 先查缓存
	if _, found := cd.cache.Get(key); found {
		return true, nil
	}

	// 缓存未命中，查数据库
	return cd.store.Has(key)
}

// Delete 删除键（同时删除缓存）
func (cd *CachedDatabase) Delete(key []byte) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	if err := cd.store.Delete(key); err != nil {
		return err
	}

	cd.cache.Delete(key)
	return nil
}

// NewBatch 创建批量操作
func (cd *CachedDatabase) NewBatch() Batch {
	return cd.store.NewBatch()
}

// Write 执行批量操作（失效缓存）
func (cd *CachedDatabase) Write(batch Batch) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	if err := cd.store.Write(batch); err != nil {
		return err
	}

	// 批量写入后清空缓存
	cd.cache.Clear()
	return nil
}

// NewIterator 创建迭代器
func (cd *CachedDatabase) NewIterator(prefix []byte) Iterator {
	return cd.store.NewIterator(prefix)
}

// Close 关闭数据库
func (cd *CachedDatabase) Close() error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	cd.cache.Clear()
	return cd.store.Close()
}

// Compact 压缩数据库
func (cd *CachedDatabase) Compact() error {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	return cd.store.Compact()
}

// Stat 获取数据库统计信息
func (cd *CachedDatabase) Stat() (*Stat, error) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	return cd.store.Stat()
}

// ClearCache 清空缓存
func (cd *CachedDatabase) ClearCache() {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	cd.cache.Clear()
}

// BackgroundCacheCleanup 后台缓存清理
func (cd *CachedDatabase) BackgroundCacheCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 定期清理过期缓存项
			cd.mu.Lock()
			// LRU 缓存会在 Get 时自动清理过期项
			// 这里可以添加额外的清理逻辑
			cd.mu.Unlock()
		}
	}
}
