package main

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"net/http"
	"time"
	"os"
	"strconv"
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

var db *gorm.DB

func openDBConnection()  error {
	var err error
	db, err = gorm.Open(sqlite.Open("reader.db"), &gorm.Config{})
	return err
}

func initializeDB (db *gorm.DB) {
	db.AutoMigrate(&Feed{})
	db.AutoMigrate(&Item{})
	db.Create(&Feed{Name:"NYT Tech",Url:"https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml"})
	db.Create(&Feed{Name:"Reuters Tech",Url:"https://www.reutersagency.com/feed/?best-topics=tech&post_type=best"})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	limit := 15
	page := 0
	offset := 0
	result := make([]string,0,limit)
	
	pageQuery := r.URL.Query()
	if pageQuery.Get("page") != "" {
		p, err := strconv.Atoi(pageQuery.Get("page"))
		if err == nil {
			page = p
			offset = p*limit
		}
	}
	err := loadItemsFromDB(db, &result, limit, offset)
	if err != nil {
		fmt.Printf("Error in rootHandler: ",err)
	}
	for _,s := range result {
		fmt.Fprintf(w, s)
	}
	if page > 0 {
		fmt.Fprintf(w, "<a href=\"/?page=%v\">previous</a>",page-1)
	}
		fmt.Fprintf(w, "| Page %v | <a href=\"/?page=%v\">next</a>",page,page+1)
}


func main() {
	if err := openDBConnection(); err != nil {
		fmt.Printf("encountered an error: %v", err)
		os.Exit(1)
	}

// 	initializeDB(db)	


	http.HandleFunc("/", rootHandler)

	// 	// handler for dynamic content
	//
	// 	// handler for static files
	// 	staticFileHandler := http.FileServer(http.Dir("./www"))
	// 	http.Handle("/static/", staticFileHandler)
	//
	http.ListenAndServe(":80", nil)


// 	err = ingestFromDB(db)
// 	err = loadItemsFromDB(db, 15, 10)
// 	if err != nil {
// 		fmt.Printf("encountered an error: %v", err)
// 	}
}
