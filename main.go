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
	Abbr string
	Url string
}

type Item struct {
	gorm.Model
	Title string
	FeedAbbr string
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
	db.Create(&Feed{Name:"NYT Tech",Abbr:"NYT",Url:"https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml"})
	db.Create(&Feed{Name:"NOS Nieuws Algemeen",Abbr:"NOS",Url:"https://feeds.nos.nl/nosnieuwsalgemeen"})
	db.Create(&Feed{Name:"Tagesschau",Abbr:"ARD",Url:"https://www.tagesschau.de/xml/atom/"})

	// load feeds
	if err := ingestFromDB(db); err != nil {
		fmt.Printf("encountered an error: %v", err)
		os.Exit(1)
	}

}

// sends HTML from a file to w (if file exists)
func emitHTMLFromFile(w http.ResponseWriter, r *http.Request, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	} 
	fmt.Fprintf(w, string(data))
}

func emitFeedFilterHTML (w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "		<div class=\"col-auto order-5\">")
	defer fmt.Fprintf(w,"		</div>")
	var feeds []Feed
	result := db.Find(&feeds)
	if result.Error != nil {
		return
	}
	for _,f := range feeds {
		fmt.Fprintf(w,"<div><a href=\"/?filter=%v\">%v</a></div>",f.Abbr,f.Abbr)
	}
	fmt.Fprintf(w,"<div><a href=\"/\">Clear</a></div>")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	limit := 15
	page := 0
	offset := 0
	filter := ""
	result := make([]string,0,limit)
	
	emitHTMLFromFile(w, r, "./www/header.html")
	defer emitHTMLFromFile(w, r, "./www/footer.html")

	emitFeedFilterHTML(w, r)

	fmt.Fprintf(w, "		<div class=\"col\">")
	defer fmt.Fprintf(w, "		</div>")
	
	pageQuery := r.URL.Query()
	if pageQuery.Get("page") != "" {
		p, err := strconv.Atoi(pageQuery.Get("page"))
		if err == nil {
			page = p
			offset = p*limit
		}
	}
	
	if f := pageQuery.Get("filter"); f != "" {
		if isAlpha(f) {
			filter=firstN(f,4)
		}
	}
	
	err := loadItemsFromDB(db, &result, filter, limit, offset)
	if err != nil {
		fmt.Printf("Error in rootHandler: ",err)
	}
	for _,s := range result {
		fmt.Fprintf(w, s)
	}
	if page > 0 {
		fmt.Fprintf(w, "<a href=\"/?page=%v&filter=%v\">Previous</a>",page-1, filter)
	} else {
		fmt.Fprintf(w, "Previous")
	}
		fmt.Fprintf(w, " | Page %v | <a href=\"/?page=%v&filter=%v\">next</a>",page,page+1,filter)
}

func updateFeedsHandler(w http.ResponseWriter, r *http.Request) {
	emitHTMLFromFile(w, r, "./www/header.html")
	defer emitHTMLFromFile(w, r, "./www/footer.html")

	err := ingestFromDB(db)
	if err != nil {
		fmt.Fprintf(w, "<div>Error updating feeds: %v</div>", err)
	} else {
		fmt.Fprintf(w, "<div>Feeds updated successfully.</div>")
	}
	fmt.Fprintf(w, "<div><a href=\"/\">Return to homepage</a></div>")
}

func adminFeedsHandler (w http.ResponseWriter, r *http.Request) {
	emitHTMLFromFile(w, r, "./www/header.html")
	defer emitHTMLFromFile(w, r, "./www/footer.html")
}


func main() {
	if err := openDBConnection(); err != nil {
		fmt.Printf("encountered an error: %v", err)
		os.Exit(1)
	}

//	uncomment this line when DB needs to be recreated
// 	initializeDB(db)	
	

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/update/", updateFeedsHandler)
	http.HandleFunc("/feeds/", adminFeedsHandler)
	staticFileHandler := http.FileServer(http.Dir("./www"))
	http.Handle("/static/", staticFileHandler)
	//
	http.ListenAndServe(":80", nil)


}
