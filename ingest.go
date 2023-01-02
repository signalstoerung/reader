package main

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

// loads the feed list from the DB, then calls ingestFromUrlWriteToDB to load all feed items and write them to the DB (skipping duplicates).
func ingestFromDB(db *gorm.DB) error {
	var feeds []Feed

	fmt.Println("Trying to ingest from DB.")
	result := db.Find(&feeds)
	fmt.Println(result.RowsAffected, " rows found.")
	if result.RowsAffected == 0 {
		return errors.New("No feeds found.")
	}
	if result.Error != nil {
		return result.Error
	}
	for _, f := range feeds {
		err := ingestFromUrlWriteToDB(db, f.Url)
		if err != nil {
			return err
		}
	}
	return nil
}

// func ingestFromUrl(u string) error {
// 	fp := gofeed.NewParser()
// 	feed, err := fp.ParseURL(u)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println(feed.Title)
// 	for _, item := range feed.Items {
// 		fmt.Printf("%v -- %v\n", item.PublishedParsed.Format("Jan 02 15:04"), item.Title)
// 	}
// 	return nil
// }

func ingestFromUrlWriteToDB(db *gorm.DB, u string) error {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(u)
	if err != nil {
		return err
	}
	fmt.Println(feed.Title)
	for _, item := range feed.Items {
		fmt.Printf("%v -- %v\n", item.PublishedParsed.Format("Jan 02 15:04"), item.Title)
		hash := sha1.Sum([]byte(item.Link))
		hashBase64 := base64.StdEncoding.EncodeToString(hash[:])
		dbItem := Item{Title: item.Title, Link: item.Link, Hash: hashBase64, PublishedParsed: item.PublishedParsed}
		// 		result := db.Create(&dbItem)
		result := db.Where(Item{Hash: hashBase64}).FirstOrCreate(&dbItem)
		fmt.Println("Gorm rows affected: ", result.RowsAffected)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}
