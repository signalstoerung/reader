package main

import (
	"fmt"
	"log"
	"time"

	"github.com/signalstoerung/reader/internal/cache"
	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/users"
)

const (
	PathFeeds                        = "/feeds"
	PathItems                        = "/items"
	CacheDurationItems time.Duration = 15 * time.Minute
	CacheDurationFeeds time.Duration = 6 * time.Hour
)

func getAllFeedsFromCacheOrDB() interface{} {
	feedlist, err := cache.GlobalCache.Get(PathFeeds)
	if err != nil {
		feedlist, err = feeds.AllFeeds()
		if err != nil {
			log.Panic(err)
		}
		cache.GlobalCache.Add(PathFeeds, feedlist, time.Now().Add(CacheDurationFeeds))
	}
	return feedlist
}

func getItemsFromCacheOrDB(filter string, limit int, offset int, timestamp int64) interface{} {
	path := fmt.Sprintf("%s/%s/%d/%d/%d", PathItems, filter, limit, timestamp, offset)
	items, err := cache.GlobalCache.Get(path)
	if err != nil {
		items, err = feeds.Items(filter, limit, offset, timestamp)
		if err != nil {
			log.Panic(err)
		}
		cache.GlobalCache.Add(path, items, time.Now().Add(CacheDurationItems))
	}
	return items
}

func getUserKeywordsFromCacheorDB(username string) interface{} {
	path := fmt.Sprintf("/keywords/%v", username)
	kl, err := cache.GlobalCache.Get(path)
	if err != nil {
		kl, err = users.KeywordsForUser(username)
		if err != nil {
			log.Panic(err)
		}
		cache.GlobalCache.Add(path, kl, time.Now().Add(time.Hour*1))
	}
	return kl
}

func invalidateKeywordCacheForUser(username string) {
	path := fmt.Sprintf("/keywords/%v", username)
	cache.GlobalCache.Invalidate(path)

}
