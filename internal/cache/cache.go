package cache

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	DurationMinimum time.Duration = 1 * time.Minute
)

var (
	GlobalCache       = make(Cache)
	CleanTicker       *time.Ticker
	Cancel            chan struct{}
	ErrExpiryTooShort = fmt.Errorf("expiry too short - minimum %v minutes", DurationMinimum.Minutes())
	ErrNotInCache     = errors.New("not in cache")
	mutex             sync.Mutex
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
	mutex.Lock()
	defer mutex.Unlock()
	now := time.Now()
	for key, val := range c {
		if val.Expires.Before(now) {
			delete(c, key)
		}
	}
}

func (c Cache) Add(path string, content interface{}, expires time.Time) error {
	mutex.Lock()
	defer mutex.Unlock()
	if expires.Before(time.Now().Add(DurationMinimum)) {
		return ErrExpiryTooShort
	}
	c[path] = CacheItem{
		Expires: expires,
		Content: content,
	}
	// log.Printf("Cached %v until %v", path, expires)
	return nil
}

func (c Cache) Invalidate(path string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(c, path)
}

func (c Cache) Get(path string) (interface{}, error) {
	mutex.Lock()
	defer mutex.Unlock()
	ci, ok := c[path]
	if !ok {
		// log.Printf("%v not in cache", path)
		return nil, ErrNotInCache
	}
	if ci.Valid() {
		return ci.Content, nil
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
