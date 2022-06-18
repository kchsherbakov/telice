package main

import (
	"github.com/patrickmn/go-cache"
	"time"
)

type CacheProvider interface {
	Save(key string, value interface{})
	TryGet(key string) (interface{}, bool)
}

type inMemoryCacheProvider struct {
	cache *cache.Cache
}

func NewInMemoryCacheProvider() *inMemoryCacheProvider {
	return &inMemoryCacheProvider{cache.New(5*time.Minute, 10*time.Minute)}
}

func (p *inMemoryCacheProvider) Save(key string, value interface{}) {
	p.cache.Set(key, value, cache.DefaultExpiration)
}

func (p *inMemoryCacheProvider) TryGet(key string) (interface{}, bool) {
	return p.cache.Get(key)
}
