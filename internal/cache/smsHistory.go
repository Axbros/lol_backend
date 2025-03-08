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
	smsHistoryCachePrefixKey = "smsHistory:"
	// SmsHistoryExpireTime expire time
	SmsHistoryExpireTime = 5 * time.Minute
)

var _ SmsHistoryCache = (*smsHistoryCache)(nil)

// SmsHistoryCache cache interface
type SmsHistoryCache interface {
	Set(ctx context.Context, id uint64, data *model.SmsHistory, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.SmsHistory, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.SmsHistory, error)
	MultiSet(ctx context.Context, data []*model.SmsHistory, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetPlaceholder(ctx context.Context, id uint64) error
	IsPlaceholderErr(err error) bool
}

// smsHistoryCache define a cache struct
type smsHistoryCache struct {
	cache cache.Cache
}

// NewSmsHistoryCache new a cache
func NewSmsHistoryCache(cacheType *database.CacheType) SmsHistoryCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.SmsHistory{}
		})
		return &smsHistoryCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.SmsHistory{}
		})
		return &smsHistoryCache{cache: c}
	}

	return nil // no cache
}

// GetSmsHistoryCacheKey cache key
func (c *smsHistoryCache) GetSmsHistoryCacheKey(id uint64) string {
	return smsHistoryCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *smsHistoryCache) Set(ctx context.Context, id uint64, data *model.SmsHistory, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetSmsHistoryCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *smsHistoryCache) Get(ctx context.Context, id uint64) (*model.SmsHistory, error) {
	var data *model.SmsHistory
	cacheKey := c.GetSmsHistoryCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *smsHistoryCache) MultiSet(ctx context.Context, data []*model.SmsHistory, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetSmsHistoryCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *smsHistoryCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.SmsHistory, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetSmsHistoryCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.SmsHistory)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.SmsHistory)
	for _, id := range ids {
		val, ok := itemMap[c.GetSmsHistoryCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *smsHistoryCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetSmsHistoryCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetPlaceholder set placeholder value to cache
func (c *smsHistoryCache) SetPlaceholder(ctx context.Context, id uint64) error {
	cacheKey := c.GetSmsHistoryCacheKey(id)
	return c.cache.SetCacheWithNotFound(ctx, cacheKey)
}

// IsPlaceholderErr check if cache is placeholder error
func (c *smsHistoryCache) IsPlaceholderErr(err error) bool {
	return errors.Is(err, cache.ErrPlaceholder)
}
