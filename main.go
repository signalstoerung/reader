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
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

/* Types */

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
	Title           string
	FeedAbbr        string
	Link            string
	Hash            string `gorm:"uniqueIndex"`
	PublishedParsed *time.Time
}

// The User struct stores a user (with a session UUID)
type User struct {
	gorm.Model
	UserName  string
	Password  string
	sessionId uuid.UUID //unexported field should be ignored by gorm
}

type Config struct {
	UpdateFrequency   int    `yaml:"updateFrequency"`
	TimeZoneGMTOffset int    `yaml:"gmtOffset"`
	Secret            string `yaml:"secret"`
	ResultsPerPage    int    `yaml:"resultsPerPage"`
	DeeplApiKey       string `yaml:"deeplApiKey"`
	ApiToken          string `yaml:"apiToken"`
	localTZ           *time.Location
}

type UserSessions map[string]User

/* Global variables */

// The global variable db stores a pool of database connections. Safe for concurrent use.
var db *gorm.DB

// The global variable wg is used to synchronise goroutines
var wg sync.WaitGroup

// store user sessions
var userSessions UserSessions = make(map[string]User)

// allow registrations or not
var registrationsOpen bool = false

// logging level
var logDebugLevel bool = false

// configuration items read from config.yaml file
var globalConfig Config

/* Config */

func loadConfig() error {
	f, err := os.Open("db/config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&globalConfig)
	if err != nil {
		return err
	}
	offset := globalConfig.TimeZoneGMTOffset * 3600
	globalConfig.localTZ = time.FixedZone("Local", offset)
	return nil
}

/* DB functions */

// openDBConnection opens the database connection (using SQLite)
func openDBConnection() error {
	var err error
	db, err = gorm.Open(sqlite.Open("db/reader.db"), &gorm.Config{})
	return err
}

// initializeDB is called only if the database does not exist. It creates the necessary tables and seeds the DB with a few feeds.
func initializeDB(db *gorm.DB) {
	db.AutoMigrate(&Feed{})
	db.AutoMigrate(&Item{})
	db.AutoMigrate(&User{})
	db.Create(&Feed{Name: "NYT Wire", Abbr: "NYT", Url: "https://content.api.nytimes.com/svc/news/v3/all/recent.rss"})
	db.Create(&Feed{Name: "NOS Nieuws Algemeen", Abbr: "NOS", Url: "https://feeds.nos.nl/nosnieuwsalgemeen"})
	db.Create(&Feed{Name: "Tagesschau", Abbr: "ARD", Url: "https://www.tagesschau.de/xml/atom/"})
	db.Create(&Feed{Name: "CNBC Business", Abbr: "CNBC", Url: "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=10001147"})

	// allow new user registrations on initialization
	registrationsOpen = true

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
	limit := globalConfig.ResultsPerPage
	page := 1
	offset := 0
	filter := ""
	result := make([]HeadlinesItem, limit)

	pageQuery := r.URL.Query()
	if pageQuery.Get("page") != "" {
		p, err := strconv.Atoi(pageQuery.Get("page"))
		if err == nil {
			if p < 1 {
				// can't have negative page numbers
				// someone is messing with the input
				p = 1
			}
			page = p
			// we want the user to see the first page as Page 1, but we want offset to be 0
			// so subtract 1 from the page number shown to the user
			offset = (p - 1) * limit
		} else {
			log.Printf("Illegal value for page (%v). Ignoring.", pageQuery.Get("page"))
		}
	}

	if f := pageQuery.Get("filter"); f != "" {
		if isAlpha(f) {
			filter = firstN(f, 4)
		}
	}

	err := loadItems(db, &result, filter, limit, offset)
	if err != nil {
		returnError(w, err.Error())
		log.Println(err.Error())
		return
	}

	emitHTMLFromFile(w, "./www/header.html")
	defer emitHTMLFromFile(w, "./www/footer.html")

	emitFeedFilterHTML(w)

	// build struct	that will be passed to template
	pageStruct := HeadlinesPage{}

	pageStruct.Page = page
	pageStruct.Filter = filter
	pageStruct.Headlines = result

	if page > 1 {
		pageStruct.HasPreviousPage = true
		pageStruct.PreviousPage = page - 1
	} else {
		pageStruct.HasPreviousPage = false
	}
	pageStruct.NextPage = page + 1

	t := template.Must(template.ParseFiles("www/content-headlines.html"))
	err = t.Execute(w, pageStruct)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// updateFeedsHandler serves "/update/", which triggers an update to the feeds
func updateFeedsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}
	emitHTMLFromFile(w, "./www/header.html")
	defer emitHTMLFromFile(w, "./www/footer.html")

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
func adminFeedsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/login/", http.StatusSeeOther)
		return
	}
	if r.Method == "GET" {
		adminGetHandler(w, r)
	} else if r.Method == "POST" {
		adminPostHandler(w, r)
	} else {
		http.Error(w, "Invalid request.", http.StatusInternalServerError)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		emitHTMLFromFile(w, "./www/header.html")
		emitHTMLFromFile(w, "./www/login-form.html")
		emitHTMLFromFile(w, "./www/footer.html")
	} else if r.Method == "POST" {
		checkPassword(w, r)
	} else {
		http.Error(w, "Invalid request.", http.StatusInternalServerError)
	}
}

func registrationHandler(w http.ResponseWriter, r *http.Request) {
	if !registrationsOpen {
		emitHTMLFromFile(w, "./www/header.html")
		fmt.Fprint(w, "<b>Sorry, no new signups are allowed.</b>")
		emitHTMLFromFile(w, "./www/footer.html")
		return
	}
	if r.Method == "GET" {
		emitHTMLFromFile(w, "./www/header.html")
		emitHTMLFromFile(w, "./www/registration-form.html")
		emitHTMLFromFile(w, "./www/footer.html")
	} else if r.Method == "POST" {
		registerNewUser(w, r)
	} else {
		http.Error(w, "Invalid request.", http.StatusInternalServerError)
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Must use POST", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form data", http.StatusInternalServerError)
		return
	}
	token := r.Form.Get("token")
	if token != globalConfig.ApiToken {
		http.Error(w, "Wrong or missing API token", http.StatusBadRequest)
		return
	}
	// we have a verified token at this point

	path := r.URL.Path
	log.Printf("%v requested", path)

	if path == "/api/feeds/" {
		var feeds []Feed
		result := db.Find(&feeds)
		if result.Error != nil {
			http.Error(w, fmt.Sprintf("Error: %v", result.Error), http.StatusInternalServerError)
			return
		}
		encoder := json.NewEncoder(w)
		encoder.Encode(feeds)
	}

	if path == "/api/headlines/" {
		var limit, page, offset int
		var filter string
		var err error

		limit, err = strconv.Atoi(r.Form.Get("limit"))
		if err != nil {
			limit = globalConfig.ResultsPerPage
		}

		page, err = strconv.Atoi(r.Form.Get("page"))
		if err != nil {
			page = 1
		}

		filter = r.Form.Get("filter")
		if !isAlpha(filter) {
			filter = ""
		}

		offset = (page - 1) * limit

		result := make([]HeadlinesItem, limit)
		err = loadItems(db, &result, filter, limit, offset)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error loading items: %v", err), http.StatusInternalServerError)
			return
		}

		encoder := json.NewEncoder(w)
		encoder.Encode(result)
		return
	}

}

/* MAIN */

func main() {
	// load config
	if err := loadConfig(); err != nil {
		log.Printf("Couldn't load configuation (%v).", err)
		panic("Couldn't load configuration file.")
	}

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
	http.HandleFunc("/register/", registrationHandler)
	http.HandleFunc("/api/", apiHandler)
	staticFileHandler := http.FileServer(http.Dir("./www"))
	http.Handle("/static/", staticFileHandler)

	// start a ticker for periodic refresh using the const updateFrequency
	ticker := time.NewTicker(time.Duration(globalConfig.UpdateFrequency) * time.Minute)
	quit := make(chan int)
	defer close(quit)
	log.Printf("Starting ticker for periodic update (%v minutes).", globalConfig.UpdateFrequency)
	go periodicUpdates(ticker, quit)

	// serve web app
	log.Print("Starting to serve.")
	http.ListenAndServe(":8000", nil)
}
