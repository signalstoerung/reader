package main

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Feed struct {
	gorm.Model
	Name string
	Url string
}

type Item struct {
	gorm.Model
	Title string
	Link string
	Hash string `gorm:"uniqueIndex"`
	PublishedParsed *time.Time
}

func initializeDB (db *gorm.DB) {
	db.AutoMigrate(&Feed{})
	db.AutoMigrate(&Item{})
	db.Create(&Feed{Name:"NYT Tech",Url:"https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml"})
	db.Create(&Feed{Name:"Reuters Tech",Url:"https://www.reutersagency.com/feed/?best-topics=tech&post_type=best"})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, web.")
	fmt.Println(r)
}

func main() {
	// 	// handler for dynamic content
	// 	http.HandleFunc("/", rootHandler)
	//
	// 	// handler for static files
	// 	staticFileHandler := http.FileServer(http.Dir("./www"))
	// 	http.Handle("/static/", staticFileHandler)
	//
	// 	http.ListenAndServe(":80", nil)

	db, err := gorm.Open(sqlite.Open("reader.db"), &gorm.Config{})
	if err != nil {
		fmt.Printf("encountered an error: %v", err)
	}
	
// 	initializeDB(db)	

// 	err = ingestFromDB(db)
	err = loadItemsFromDB(db, 15, 10)
	if err != nil {
		fmt.Printf("encountered an error: %v", err)
	}
}
