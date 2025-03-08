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
	paymentHistoryCachePrefixKey = "paymentHistory:"
	// PaymentHistoryExpireTime expire time
	PaymentHistoryExpireTime = 5 * time.Minute
)

var _ PaymentHistoryCache = (*paymentHistoryCache)(nil)

// PaymentHistoryCache cache interface
type PaymentHistoryCache interface {
	Set(ctx context.Context, id uint64, data *model.PaymentHistory, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.PaymentHistory, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.PaymentHistory, error)
	MultiSet(ctx context.Context, data []*model.PaymentHistory, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetPlaceholder(ctx context.Context, id uint64) error
	IsPlaceholderErr(err error) bool
}

// paymentHistoryCache define a cache struct
type paymentHistoryCache struct {
	cache cache.Cache
}

// NewPaymentHistoryCache new a cache
func NewPaymentHistoryCache(cacheType *database.CacheType) PaymentHistoryCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.PaymentHistory{}
		})
		return &paymentHistoryCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.PaymentHistory{}
		})
		return &paymentHistoryCache{cache: c}
	}

	return nil // no cache
}

// GetPaymentHistoryCacheKey cache key
func (c *paymentHistoryCache) GetPaymentHistoryCacheKey(id uint64) string {
	return paymentHistoryCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *paymentHistoryCache) Set(ctx context.Context, id uint64, data *model.PaymentHistory, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetPaymentHistoryCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *paymentHistoryCache) Get(ctx context.Context, id uint64) (*model.PaymentHistory, error) {
	var data *model.PaymentHistory
	cacheKey := c.GetPaymentHistoryCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *paymentHistoryCache) MultiSet(ctx context.Context, data []*model.PaymentHistory, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetPaymentHistoryCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *paymentHistoryCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.PaymentHistory, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetPaymentHistoryCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.PaymentHistory)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.PaymentHistory)
	for _, id := range ids {
		val, ok := itemMap[c.GetPaymentHistoryCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *paymentHistoryCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetPaymentHistoryCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetPlaceholder set placeholder value to cache
func (c *paymentHistoryCache) SetPlaceholder(ctx context.Context, id uint64) error {
	cacheKey := c.GetPaymentHistoryCacheKey(id)
	return c.cache.SetCacheWithNotFound(ctx, cacheKey)
}

// IsPlaceholderErr check if cache is placeholder error
func (c *paymentHistoryCache) IsPlaceholderErr(err error) bool {
	return errors.Is(err, cache.ErrPlaceholder)
}
