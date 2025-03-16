package main

import (
	"flag"
	"fmt"
	"github.com/signalstoerung/reader/internal/feeds"
	"log"
	"net/url"
	"os"
)

func main() {
	var command string
	var dbPath string
	flag.StringVar(&dbPath, "db", "./archive.db", "path to the DB file")
	flag.Parse()

	log.Printf("DB: %v\n", dbPath)
	err := feeds.Config.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("Could not open DB: %v", err)
	}

	command = flag.Arg(0)

	switch command {
	case "add":
		feed := flag.Arg(1)
		abbr := flag.Arg(2)
		if abbr == "" || feed == "" {
			fmt.Println("Missing argument to 'add': add url abbr")
			os.Exit(1)
		}
		log.Printf("add feed %v with abbr %v\n", feed, abbr)
		feedUrl, err := url.Parse(feed)
		if err != nil || (feedUrl.Scheme != "http" && feedUrl.Scheme != "https") {
			fmt.Printf("Expecting http(s) url. (%v)", err)
			os.Exit(1)
		}
		feeds.CreateFeed(feeds.Feed{Name: abbr, Abbr: abbr, Url: feedUrl.String()})
	default:
		log.Println("poll & update db")
		err = feeds.UpdateFeeds()
		if err != nil {
			log.Printf("Error updating feeds: %v", err)
			os.Exit(2)
		}
	}

}
