package managers

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/bluele/gcache"
	"github.com/dk13danger/media-service/config"
)

type CacheManager struct {
	logger *logrus.Logger
	cache  gcache.Cache
}

func NewCacheManager(
	logger *logrus.Logger,
	cfg *config.CacheManager,
) *CacheManager {
	cache := gcache.New(cfg.Size).
		LRU().
		Expiration(time.Duration(cfg.Expiration) * time.Second).
		Build()
	return &CacheManager{
		logger: logger,
		cache:  cache,
	}
}

func (c *CacheManager) Set(key string) {
	c.cache.Set(key, true)
}

func (c *CacheManager) Remove(key string) {
	c.cache.Remove(key)
}

func (c *CacheManager) Get(key string) bool {
	value, err := c.cache.Get(key)
	if err == gcache.KeyNotFoundError {
		return false
	}
	if err != nil {
		return false
		//return nil, err
	}
	return value.(bool)
}
