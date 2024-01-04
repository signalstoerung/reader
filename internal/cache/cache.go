package cache

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	DurationMinimum time.Duration = 1 * time.Minute
)

type CacheItem struct {
	Expires time.Time
	Content interface{}
}

func (ci *CacheItem) Valid() bool {
	return ci.Expires.After(time.Now())
}

type Cache map[string]CacheItem

func (c Cache) Clean() {
	now := time.Now()
	for key, val := range c {
		if val.Expires.Before(now) {
			delete(c, key)
		}
	}
}

func (c Cache) Add(path string, content interface{}, expires time.Time) error {
	if expires.Before(time.Now().Add(DurationMinimum)) {
		return ErrExpiryTooShort
	}
	c[path] = CacheItem{
		Expires: expires,
		Content: content,
	}
	log.Printf("Cached %v until %v", path, expires)
	return nil
}

func (c Cache) Get(path string) (interface{}, error) {
	ci, ok := c[path]
	if !ok {
		log.Printf("%v not in cache", path)
		return nil, ErrNotInCache
	}
	if ci.Valid() {
		return ci.Content, nil
	} else {
		log.Printf("Cache: %v expired", path)
	}
	return nil, ErrNotInCache
}

func clean() {
	defer log.Println("Cache: Exited clean()")
OuterLoop:
	for {
		select {
		case <-CleanTicker.C:
			log.Println("Cache: cleaning")
			GlobalCache.Clean()
		case <-Cancel:
			CleanTicker.Stop()
			log.Println("package Cache received signal to cancel clean ticker")
			break OuterLoop
		}
	}
}

func init() {
	CleanTicker = time.NewTicker(5 * time.Minute)
	Cancel = make(chan struct{})
	go clean()
}

var (
	GlobalCache       = make(Cache)
	CleanTicker       *time.Ticker
	Cancel            chan struct{}
	ErrExpiryTooShort = fmt.Errorf("expiry too short - minimum %v minutes", DurationMinimum.Minutes())
	ErrNotInCache     = errors.New("not in cache")
)
