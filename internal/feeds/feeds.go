package feeds

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ErrNoDBConnection = errors.New("no database connection")
)

type Configuration struct {
	DB *gorm.DB
}

func (c *Configuration) OpenDatabase(path string) error {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}
	db.AutoMigrate(&Feed{})
	db.AutoMigrate(&Item{})
	c.DB = db
	return nil
}

// The Feed struct stores information about an RSS feed.
type Feed struct {
	gorm.Model
	Name string
	Abbr string
	Url  string
}

// The Item struct stores an item from an RSS feed.
type Item struct {
	gorm.Model
	Title              string
	FeedAbbr           string
	Link               string
	Description        string
	Content            string
	Hash               string `gorm:"uniqueIndex"`
	BreakingNewsScore  int
	BreakingNewsReason string
	PublishedParsed    *time.Time `gorm:"index"`
}

/** GLOBAL VARIABLES **/

var Config = Configuration{}

/*** UPDATE FEEDS ***/

// LoadFeedsIntoDB loads the feed list from the DB, then calls ingestFromUrlWriteToDB (concurrently) to load all feed items and write them to the DB (skipping duplicates).
func UpdateFeeds() error {
	var db *gorm.DB
	if db = Config.DB; db == nil {
		return ErrNoDBConnection
	}
	var feeds []Feed
	var wg sync.WaitGroup

	result := db.Find(&feeds)
	if result.RowsAffected == 0 {
		return errors.New("no feeds found")
	}
	if result.Error != nil {
		return result.Error
	}
	for _, f := range feeds {
		wg.Add(1)
		feed := f //  If you create a closure inside a loop and this closure accesses the loop variable, it doesn't capture the value of the loop variable at the moment the closure is created. Instead, it captures the variable itself. Solution: reassign to a new variable.
		go func() {
			defer wg.Done()
			ingestFromUrlWriteToDB(db, feed.Url, feed.Abbr)
		}()
	}
	wg.Wait()
	return nil
}

// goroutine called by ingestFromDB. Loads all items of a given feed (from url) and writes them to the DB if they're new.
func ingestFromUrlWriteToDB(db *gorm.DB, u string, abbr string) {
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
		dbItem := Item{Title: item.Title, FeedAbbr: abbr, Link: item.Link, Description: item.Description, Content: item.Content, Hash: hashBase64, PublishedParsed: item.PublishedParsed}
		result := db.Where(Item{Hash: hashBase64}).FirstOrCreate(&dbItem)
		if result.Error != nil {
			return
		}
	}
}

/** RETRIEVE ITEMS **/

func AllFeeds() ([]Feed, error) {
	if Config.DB == nil {
		return nil, ErrNoDBConnection
	}
	var feeds []Feed
	result := Config.DB.Find(&feeds)
	return feeds, result.Error
}

func Items(filter string, limit int, offset int) ([]Item, error) {
	var headlines []Item
	var db *gorm.DB
	if db = Config.DB; db == nil {
		return nil, ErrNoDBConnection
	}
	result := db.Limit(limit).Offset(offset).Order("published_parsed desc").Where(&Item{FeedAbbr: filter}).Find(&headlines)
	if result.Error != nil {
		return nil, result.Error
	}
	return headlines, nil
}

func UnscoredHeadlines() ([]Item, error) {
	var headlines []Item
	var db *gorm.DB
	if db = Config.DB; db == nil {
		return nil, ErrNoDBConnection
	}
	result := db.Raw("SELECT * from items WHERE breaking_news_score = 0 OR breaking_news_score IS NULL ORDER BY published_parsed DESC LIMIT 20").Scan(&headlines)
	if result.Error != nil {
		return nil, result.Error
	}
	return headlines, nil
}

func FirstUnscoredHeadline() (Item, error) {
	var db *gorm.DB
	if db = Config.DB; db == nil {
		return Item{}, ErrNoDBConnection
	}
	var headlines []Item
	result := db.Raw("SELECT * from items WHERE breaking_news_score = 0 OR breaking_news_score IS NULL ORDER BY published_parsed DESC LIMIT 20").Scan(&headlines)
	if result.Error != nil {
		return Item{}, result.Error
	}
	if result.RowsAffected < 15 {
		log.Printf("Only found %v headlines to score, aborting", result.RowsAffected)
		return Item{}, fmt.Errorf("only %v headlines found, aborting", result.RowsAffected)
	}
	// returning last element - this ensures that at least 20 headlines are collected before scoring is triggered.
	return headlines[len(headlines)-1], nil
}

// Returns the last 10 headlines scored 90 or higher
func RecentBreakingNews() ([]string, bool) {
	var alerts []string
	var db *gorm.DB
	if db = Config.DB; db == nil {
		return nil, false
	}
	result := db.Raw("select title from items where breaking_news_score > 89 order by published_parsed desc limit 10").Scan(&alerts)
	if result.Error != nil {
		log.Printf("Error retrieving alerts: %v", result.Error)
		return nil, false
	}
	if result.RowsAffected <= 1 {
		log.Printf("No headlines found: %v", result.RowsAffected)
		return nil, false
	}
	return alerts, true
}

/* CREATE */

func CreateFeed(f Feed) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	result := Config.DB.Create(&f)
	return result.Error
}

func CreateItem(i Item) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	result := Config.DB.Create(&i)
	return result.Error
}

/* UPDATE */

func SaveItem(i Item) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	result := Config.DB.Save(&i)
	return result.Error
}

func SaveFeed(f Feed) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	result := Config.DB.Save(&f)
	return result.Error
}

/* DELETE */

func DeleteFeed(f Feed) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	result := Config.DB.Delete(&f)
	return result.Error
}