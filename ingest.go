package main

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

// ingestFromDB loads the feed list from the DB, then calls ingestFromUrlWriteToDB (concurrently) to load all feed items and write them to the DB (skipping duplicates).
func ingestFromDB(db *gorm.DB) error {
	var feeds []Feed

	result := db.Find(&feeds)
	if result.RowsAffected == 0 {
		return errors.New("no feeds found")
	}
	if result.Error != nil {
		return result.Error
	}
	for _, f := range feeds {
		wg.Add(1)
		go ingestFromUrlWriteToDB(db, f.Url, f.Abbr)
	}
	wg.Wait()
	return nil
}

// goroutine called by ingestFromDB. Loads all items of a given feed (from url) and writes them to the DB if they're new.
func ingestFromUrlWriteToDB(db *gorm.DB, u string, abbr string) {
	defer wg.Done()
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(u)
	if err != nil {
		return
	}
	log.Printf("Updating %s.", feed.Title)
	for _, item := range feed.Items {
		// this is our way of avoiding duplicates. We hash the link and then check the DB for this hash.
		// It's a unique key, so trying to insert a duplicate will throw an error. Hence we use gorm's "First or create", which is roughly the same as "INSERT IGNORE"
		hash := sha1.Sum([]byte(item.Link))
		hashBase64 := base64.StdEncoding.EncodeToString(hash[:])
		dbItem := Item{Title: item.Title, FeedAbbr: abbr, Link: item.Link, Hash: hashBase64, PublishedParsed: item.PublishedParsed}
		result := db.Where(Item{Hash: hashBase64}).FirstOrCreate(&dbItem)
		if result.Error != nil {
			return
		}
	}
}

// periodicUpdates waits for a tick to be transmitted from a time.Ticker and then triggers an update of the feeds.
// It terminates when receiving anything on the q (quit) channel (or if the channel closes).
func periodicUpdates(t *time.Ticker, q chan int) {
	for {
		select {
		case <-t.C:
			log.Print("Periodic feed update triggered.")
			ingestFromDB(db)
		case <-q:
			t.Stop()
			return
		}
	}
}
