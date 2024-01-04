package main

import (
	"fmt"
	"log"
	"time"

	"github.com/signalstoerung/reader/internal/cache"
	"github.com/signalstoerung/reader/internal/feeds"
)

const (
	PathFeeds = "/feeds"
	PathItems = "/items"
)

func getAllFeedsFromCacheOrDB() interface{} {
	feedlist, err := cache.GlobalCache.Get(PathFeeds)
	if err != nil {
		feedlist, err = feeds.AllFeeds()
		if err != nil {
			log.Panic(err)
		}
		cache.GlobalCache.Add(PathFeeds, feedlist, time.Now().Add(6*time.Hour))
	}
	return feedlist
}

func getItemsFromCacheOrDB(filter string, limit int, offset int) interface{} {
	path := fmt.Sprintf("%s/%s/%d/%d", PathItems, filter, limit, offset)
	items, err := cache.GlobalCache.Get(path)
	if err != nil {
		items, err = feeds.Items(filter, limit, offset)
		if err != nil {
			log.Panic(err)
		}
		cache.GlobalCache.Add(path, items, time.Now().Add(5*time.Minute))
	}
	return items
}
