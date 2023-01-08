/*
Reader is my "Poor Man's BBG" - news pulled in from various RSS feeds, timestamped and tagged with a source,
and displayed in the style of a ticker with only headlines.

It starts up a web server that serves:

  /         the headlines (paginated and optionally filtered)
  /feeds/   an interface to add or delete feeds
  /update/  manually trigger a feed update

In the backend, Reader uses an sqlite database. At first startup, when the DB does not exist, 
it is created and seeded with a couple of recommended feeds.

Automatic updates take place with the frequency (in minutes) defined by updateFrequency.

*/
package main

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/google/uuid"
	"net/http"
	"time"
	"os"
	"strconv"
	"errors"
	"log"
	"fmt"
	"sync"
)

/* Types */

// The Feed struct stores information about an RSS feed.
type Feed struct {
	gorm.Model
	Name string
	Abbr string
	Url string
}

// The Item struct stores an item from an RSS feed.
type Item struct {
	gorm.Model
	Title string
	FeedAbbr string
	Link string
	Hash string `gorm:"uniqueIndex"`
	PublishedParsed *time.Time
}

// The User struct stores a user (with a session UUID)
type User struct {
	gorm.Model
	UserName string
	Password string
	sessionId uuid.UUID //unexported field should be ignored by gorm
}

type UserSessions []User

/* Global variables */

// The global variable db stores a pool of database connections. Safe for concurrent use.
var db *gorm.DB

// The global variable wg is used to synchronise goroutines
var wg sync.WaitGroup

// Frequency of automatic updates, in minutes
const updateFrequency = 15 

// openDBConnection opens the database connection (using SQLite)
func openDBConnection()  error {
	var err error
	db, err = gorm.Open(sqlite.Open("db/reader.db"), &gorm.Config{})
	return err
}

// store user sessions
var userSessions UserSessions = make([]User,0,10)


/* DB functions */

// initializeDB is called only if the database does not exist. It creates the necessary tables and seeds the DB with a few feeds.
func initializeDB (db *gorm.DB) {
	db.AutoMigrate(&Feed{})
	db.AutoMigrate(&Item{})
	db.AutoMigrate(&User{})
	db.Create(&Feed{Name:"NYT Wire",Abbr:"NYT",Url:"https://content.api.nytimes.com/svc/news/v3/all/recent.rss"})
	db.Create(&Feed{Name:"NOS Nieuws Algemeen",Abbr:"NOS",Url:"https://feeds.nos.nl/nosnieuwsalgemeen"})
	db.Create(&Feed{Name:"Tagesschau",Abbr:"ARD",Url:"https://www.tagesschau.de/xml/atom/"})
	db.Create(&Feed{Name:"CNBC Business",Abbr:"CNBC",Url:"https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=10001147"})
	
	// create a dummy user
	dummyUser := User{UserName: "wrgfst", Password:""}
	dummyUser.setPassword("M47Ks8eMJK4z")
	db.Create(&dummyUser)

	// load feeds
	if err := ingestFromDB(db); err != nil {
		log.Printf("encountered an error: %v", err)
		os.Exit(1)
	}

}

/* Request handler functions */

// rootHandler serves "/"
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}
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
		log.Printf("Error in rootHandler: ",err)
	}
	for _,s := range result {
		fmt.Fprintf(w, s)
	}
	if page > 0 {
		fmt.Fprintf(w, "<a href=\"/?page=%v&filter=%v\">Previous</a>",page-1, filter)
	} else {
		fmt.Fprintf(w, "Previous")
	}
		fmt.Fprintf(w, " | Page %v | <a href=\"/?page=%v&filter=%v\">Next</a>",page,page+1,filter)
}

// updateFeedsHandler serves "/update/", which triggers an update to the feeds
func updateFeedsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}
	emitHTMLFromFile(w, r, "./www/header.html")
	defer emitHTMLFromFile(w, r, "./www/footer.html")

	log.Print("Updating feeds...")
	err := ingestFromDB(db)
	if err != nil {
		fmt.Fprintf(w, "<div>Error updating feeds: %v</div>", err)
	} else {
		fmt.Fprintf(w, "<div>Feeds updated successfully.</div>")
	}
	fmt.Fprintf(w, "<div><a href=\"/\">Return to homepage</a></div>")
}

// adminFeedsHandler serves "/feeds/", which allows deletion and creation of feeds.
// it calls adminGetHandler or adminPostHandler depending on request method.
func adminFeedsHandler (w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}
	if r.Method == "GET" {
		adminGetHandler (w, r)
	} else if r.Method == "POST" {
		adminPostHandler (w, r)
	} else {
		http.Error(w, "Invalid request.", http.StatusInternalServerError)
	}
}

func loginHandler (w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		emitHTMLFromFile(w, r, "./www/header.html")
		emitHTMLFromFile(w, r, "./www/login-form.html")
		emitHTMLFromFile(w, r, "./www/footer.html")
	} else if r.Method == "POST" {
		checkPassword(w, r)
	} else {
		http.Error(w, "Invalid request.", http.StatusInternalServerError)	
	}
}

/* MAIN */

func main() {
//	recreate reader.db if it doesn't exist
	if _, err := os.Stat("./db/reader.db"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Print("reader.db doesn't exist, recreating...'")
			if err := openDBConnection(); err != nil {
				log.Printf("encountered an error: %v", err)
				os.Exit(1)
			}
			initializeDB(db)	
		} else {
			log.Print(err)
			os.Exit(1)
		}
	} else {
		if err := openDBConnection(); err != nil {
			log.Printf("encountered an error: %v", err)
			os.Exit(1)
		}
	}

// register handlers
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/update/", updateFeedsHandler)
	http.HandleFunc("/feeds/", adminFeedsHandler)
	http.HandleFunc("/login/", loginHandler)
	staticFileHandler := http.FileServer(http.Dir("./www"))
	http.Handle("/static/", staticFileHandler)


// start a ticker for periodic refresh using the const updateFrequency
	ticker := time.NewTicker(updateFrequency * time.Minute)
	quit := make(chan int)
	defer close(quit)
	log.Print("Starting ticker for periodic update.")
	go periodicUpdates(ticker, quit)

// serve web app
	log.Print("Starting to serve.")
	http.ListenAndServe(":80", nil)
}
