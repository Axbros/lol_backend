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
	loanCachePrefixKey = "loan:"
	// LoanExpireTime expire time
	LoanExpireTime = 5 * time.Minute
)

var _ LoanCache = (*loanCache)(nil)

// LoanCache cache interface
type LoanCache interface {
	Set(ctx context.Context, id uint64, data *model.Loan, duration time.Duration) error
	Get(ctx context.Context, id uint64) (*model.Loan, error)
	MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.Loan, error)
	MultiSet(ctx context.Context, data []*model.Loan, duration time.Duration) error
	Del(ctx context.Context, id uint64) error
	SetPlaceholder(ctx context.Context, id uint64) error
	IsPlaceholderErr(err error) bool
}

// loanCache define a cache struct
type loanCache struct {
	cache cache.Cache
}

// NewLoanCache new a cache
func NewLoanCache(cacheType *database.CacheType) LoanCache {
	jsonEncoding := encoding.JSONEncoding{}
	cachePrefix := ""

	cType := strings.ToLower(cacheType.CType)
	switch cType {
	case "redis":
		c := cache.NewRedisCache(cacheType.Rdb, cachePrefix, jsonEncoding, func() interface{} {
			return &model.Loan{}
		})
		return &loanCache{cache: c}
	case "memory":
		c := cache.NewMemoryCache(cachePrefix, jsonEncoding, func() interface{} {
			return &model.Loan{}
		})
		return &loanCache{cache: c}
	}

	return nil // no cache
}

// GetLoanCacheKey cache key
func (c *loanCache) GetLoanCacheKey(id uint64) string {
	return loanCachePrefixKey + utils.Uint64ToStr(id)
}

// Set write to cache
func (c *loanCache) Set(ctx context.Context, id uint64, data *model.Loan, duration time.Duration) error {
	if data == nil || id == 0 {
		return nil
	}
	cacheKey := c.GetLoanCacheKey(id)
	err := c.cache.Set(ctx, cacheKey, data, duration)
	if err != nil {
		return err
	}
	return nil
}

// Get cache value
func (c *loanCache) Get(ctx context.Context, id uint64) (*model.Loan, error) {
	var data *model.Loan
	cacheKey := c.GetLoanCacheKey(id)
	err := c.cache.Get(ctx, cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiSet multiple set cache
func (c *loanCache) MultiSet(ctx context.Context, data []*model.Loan, duration time.Duration) error {
	valMap := make(map[string]interface{})
	for _, v := range data {
		cacheKey := c.GetLoanCacheKey(v.ID)
		valMap[cacheKey] = v
	}

	err := c.cache.MultiSet(ctx, valMap, duration)
	if err != nil {
		return err
	}

	return nil
}

// MultiGet multiple get cache, return key in map is id value
func (c *loanCache) MultiGet(ctx context.Context, ids []uint64) (map[uint64]*model.Loan, error) {
	var keys []string
	for _, v := range ids {
		cacheKey := c.GetLoanCacheKey(v)
		keys = append(keys, cacheKey)
	}

	itemMap := make(map[string]*model.Loan)
	err := c.cache.MultiGet(ctx, keys, itemMap)
	if err != nil {
		return nil, err
	}

	retMap := make(map[uint64]*model.Loan)
	for _, id := range ids {
		val, ok := itemMap[c.GetLoanCacheKey(id)]
		if ok {
			retMap[id] = val
		}
	}

	return retMap, nil
}

// Del delete cache
func (c *loanCache) Del(ctx context.Context, id uint64) error {
	cacheKey := c.GetLoanCacheKey(id)
	err := c.cache.Del(ctx, cacheKey)
	if err != nil {
		return err
	}
	return nil
}

// SetPlaceholder set placeholder value to cache
func (c *loanCache) SetPlaceholder(ctx context.Context, id uint64) error {
	cacheKey := c.GetLoanCacheKey(id)
	return c.cache.SetCacheWithNotFound(ctx, cacheKey)
}

// IsPlaceholderErr check if cache is placeholder error
func (c *loanCache) IsPlaceholderErr(err error) bool {
	return errors.Is(err, cache.ErrPlaceholder)
}
