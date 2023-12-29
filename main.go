/*
Reader is my "Poor Man's Bloomberg" - news pulled in from various RSS feeds, timestamped and tagged with a source,
and displayed in the style of a ticker with only headlines.

It starts up a web server that serves:

	/         the headlines (paginated and optionally filtered)
	/feeds/   an interface to add or delete feeds
	/update/  manually trigger a feed update

In the backend, Reader uses an sqlite database, stored in the subfolder ./db/. At first startup, when the DB does not exist,
it is created and seeded with a couple of recommended feeds.

A config file named config.yaml needs to be present in ./db/ as well, or Reader will panic.
*/
package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/signalstoerung/reader/internal/openai"
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
	Title              string
	FeedAbbr           string
	Link               string
	Description        string
	Content            string
	Hash               string `gorm:"uniqueIndex"`
	BreakingNewsScore  int
	BreakingNewsReason string
	PublishedParsed    *time.Time
}

// The User struct stores a user (with a session UUID)
type User struct {
	gorm.Model
	UserName string
	Password string
}

// The Config struct stores global configuration variables, as imported from the config.yaml file.
type Config struct {
	UpdateFrequency   int    `yaml:"updateFrequency"`
	TimeZoneGMTOffset int    `yaml:"gmtOffset"`
	Secret            string `yaml:"secret"`
	ResultsPerPage    int    `yaml:"resultsPerPage"`
	DeeplApiKey       string `yaml:"deeplApiKey"`
	OpenAIToken       string `yaml:"openAiToken"`
	localTZ           *time.Location
}

// UserSessions type, for storing information about logged-in users.
type UserSessions map[string]User

/* Global variables */

// The global variable db stores a (pool of) database connections. Safe for concurrent use.
var db *gorm.DB

// The global variable wg is used to synchronise goroutines
var wg sync.WaitGroup

// store api tokens
var issuedTokens = make(map[string]string)

// allow registrations or not
var registrationsOpen bool = false

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
	openai.Stats.ApiKey = globalConfig.OpenAIToken
	return nil
}

/* DB functions */

// openDBConnection opens the database connection (using SQLite)
func openDBConnection() error {
	var err error
	db, err = gorm.Open(sqlite.Open("db/reader.db"), &gorm.Config{})
	db.AutoMigrate(&Feed{})
	db.AutoMigrate(&Item{})
	db.AutoMigrate(&User{})
	return err
}

// initializeDB is called only if the database does not exist. It creates the necessary tables and seeds the DB with a few feeds.
func initializeDB(db *gorm.DB) {
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

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	_, path, found := strings.Cut(r.URL.Path, "/proxy/https:/")
	if !found {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	path = "https://" + path

	path = expandUrlRecursive(path)

	// strip URL parameters
	pathStripped, _, _ := strings.Cut(path, "?")

	archivePath := "https://archive.is/newest/" + pathStripped

	http.Redirect(w, r, archivePath, http.StatusMovedPermanently)
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

	path := r.URL.Path

	// client trying to log in
	if path == "/api/gettoken/" {
		apiLogin(w, r)
		return
	}

	if path == "/api/user/add/" {
		apiAddUser(w, r)
		return
	}

	token := r.Form.Get("token")

	if !tokenExists(token) {
		http.Error(w, "Wrong or missing API token", http.StatusBadRequest)
		return
	}

	switch path {
	case "/api/feeds/":
		apiFeedList(w)
	case "/api/feed/add/":
		apiAddFeed(w, r)
	case "/api/feed/delete/":
		apiDeleteFeed(w, r)
	case "/api/headlines/":
		apiHeadlines(w, r)
	default:
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
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
	http.HandleFunc("/api/", apiHandler)
	http.HandleFunc("/proxy/", proxyHandler)
	// http.HandleFunc("/openaitest/", openAITestHandler)
	staticFileHandler := http.FileServer(http.Dir("./www/static"))
	http.Handle("/", staticFileHandler)

	// start a ticker for periodic refresh using the const updateFrequency
	tickerUpdating := time.NewTicker(time.Duration(globalConfig.UpdateFrequency) * time.Minute)
	quit := make(chan int)
	defer close(quit)
	log.Printf("Starting ticker for periodic update (%v minutes).", globalConfig.UpdateFrequency)
	go periodicUpdates(tickerUpdating, quit)

	// serve web app
	log.Print("Starting to serve.")
	err := http.ListenAndServe(":8000", nil)
	log.Println(err)
	tickerUpdating.Stop()
}
