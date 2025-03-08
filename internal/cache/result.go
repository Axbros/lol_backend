package cache

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-dev-frame/sponge/pkg/cache"
	"github.com/go-dev-frame/sponge/pkg/encoding"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"lol/internal/database"
	"lol/internal/model"
)

const (
	// cache prefix key, must end with a colon
	resultCachePrefixKey = "result:"
	// ResultExpireTime expire time
	ResultExpireTime = 5 * time.Minute
)

var _ ResultCache = (*resultCache)(nil)

// ResultCache cache interface
type ResultCache interface {
	Set(ctx context.Context, id uint64, data *model.Result, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.Result, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.Result, error)
	MultiSet(ctx context.Context, data []*model.Result, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetPlaceholder(ctx context.Context, id uint64) error
	IsPlaceholderErr(err error) bool
}

// resultCache define a cache struct
type resultCache struct {
	cache cache.Cache
}

// NewResultCache new a cache
func NewResultCache(cacheType *database.CacheType) ResultCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.Result{}
		})
		return &resultCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.Result{}
		})
		return &resultCache{cache: c}
	}

	return nil // no cache
}

// GetResultCacheKey cache key
func (c *resultCache) GetResultCacheKey(id uint64) string {
	return resultCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *resultCache) Set(ctx context.Context, id uint64, data *model.Result, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetResultCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *resultCache) Get(ctx context.Context, id uint64) (*model.Result, error) {
	var data *model.Result
	cacheKey := c.GetResultCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *resultCache) MultiSet(ctx context.Context, data []*model.Result, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetResultCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *resultCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.Result, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetResultCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.Result)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.Result)
	for _, id := range ids {
		val, ok := itemMap[c.GetResultCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *resultCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetResultCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetPlaceholder set placeholder value to cache
func (c *resultCache) SetPlaceholder(ctx context.Context, id uint64) error {
	cacheKey := c.GetResultCacheKey(id)
	return c.cache.SetCacheWithNotFound(ctx, cacheKey)
}

// IsPlaceholderErr check if cache is placeholder error
func (c *resultCache) IsPlaceholderErr(err error) bool {
	return errors.Is(err, cache.ErrPlaceholder)
}
