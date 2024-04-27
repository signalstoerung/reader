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
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/signalstoerung/reader/internal/cache"
	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/openai"
	"github.com/signalstoerung/reader/internal/users"
	"gopkg.in/yaml.v3"
)

/* Types */

// The Config struct stores global configuration variables, as imported from the config.yaml file.
type Config struct {
	UpdateFrequency   int    `yaml:"updateFrequency"`
	TimeZoneGMTOffset int    `yaml:"gmtOffset"`
	Secret            string `yaml:"secret"`
	ResultsPerPage    int    `yaml:"resultsPerPage"`
	DeeplApiKey       string `yaml:"deeplApiKey"`
	OpenAIToken       string `yaml:"openAiToken"`
	Debug             bool   `yaml:"-"`
	AIActive          bool   `yaml:"-"`
	localTZ           *time.Location
}

/* Global variables */

// allow registrations or not
var registrationsOpen bool = false

// configuration items read from config.yaml file
var globalConfig Config

/* Config */

func loadConfig(path string) error {
	f, err := os.Open(path)
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
func openDBConnection(path string) error {
	err := users.Config.OpenDatabase(path)
	if err != nil {
		return err
	}
	err = feeds.Config.OpenDatabase(path)
	return err
}

// initializeDB is called only if the database does not exist. It creates the necessary tables and seeds the DB with a few feeds.
func initializeDB() {
	feeds.CreateFeed(feeds.Feed{Name: "NYT Wire", Abbr: "NYT", Url: "https://content.api.nytimes.com/svc/news/v3/all/recent.rss"})
	feeds.CreateFeed(feeds.Feed{Name: "NOS Nieuws Algemeen", Abbr: "NOS", Url: "https://feeds.nos.nl/nosnieuwsalgemeen"})
	feeds.CreateFeed(feeds.Feed{Name: "Tagesschau", Abbr: "ARD", Url: "https://www.tagesschau.de/infoservices/alle-meldungen-100~rss2.xml"})
	feeds.CreateFeed(feeds.Feed{Name: "CNBC Business", Abbr: "CNBC", Url: "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=10001147"})

	// allow new user registrations on initialization
	registrationsOpen = true

	// load feeds
	if err := feeds.UpdateFeeds(); err != nil {
		log.Printf("encountered an error: %v", err)
		os.Exit(1)
	}

}

/* MAIN */

func main() {
	var debug bool
	var newscontextFilePath string
	var configFilePath string
	var dbFilePath string
	var aiActive bool
	var promptFile string

	// FLAGS
	flag.BoolVar(&debug, "debug", false, "Activate debug options and logging")
	flag.StringVar(&newscontextFilePath, "context", "./db/newscontext.txt", "File path to a text file describing the news context")
	flag.StringVar(&configFilePath, "config", "./db/config.yaml", "File path to a yaml config file")
	flag.StringVar(&dbFilePath, "db", "./db/reader.db", "File path to sqlite database")
	flag.BoolVar(&aiActive, "ai", true, "AI headline scoring active; turn off for testing to avoid charges")
	flag.BoolVar(&registrationsOpen, "register", false, "Allow registration once at startup")
	flag.StringVar(&promptFile, "promptfile", "db/gpt-prompt.txt", "File containing the GPT prompt for headline scoring")
	flag.Parse()
	// load config
	if err := loadConfig(configFilePath); err != nil {
		log.Printf("Couldn't load configuation (%v).", err)
		panic("Couldn't load configuration file.")
	}
	// set global Debug option based on command line flag
	globalConfig.Debug = debug
	globalConfig.AIActive = aiActive
	openai.Debug = debug

	if aiActive {
		log.Println("AI headline scoring active.")
		err := setPromptFromFile(promptFile)
		if err != nil {
			log.Printf("Could not set prompt, using default (%v)", err)
		}
	} else {
		log.Println("AI headline scoring inactive.")
	}

	//	recreate reader.db if it doesn't exist
	if _, err := os.Stat(dbFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("%v doesn't exist, recreating...", dbFilePath)
			if err := openDBConnection(dbFilePath); err != nil {
				log.Printf("encountered an error: %v", err)
				os.Exit(1)
			}
			initializeDB()
		} else {
			log.Print(err)
			os.Exit(1)
		}
	} else {
		if err := openDBConnection(dbFilePath); err != nil {
			log.Printf("encountered an error: %v", err)
			os.Exit(1)
		}
	}

	// register handlers
	http.HandleFunc("/", users.SessionMiddleware("/login/", headlinesHandler))
	http.HandleFunc("/login/", users.LoginMiddleware("/login", loginHandler))
	http.HandleFunc("/logout/", users.DeleteCookie(logoutHandler))
	http.HandleFunc("/register/", signupHandler)
	http.HandleFunc("/feeds/", users.SessionMiddleware("/login/", feedEditHandler))
	http.HandleFunc("/keywords/", users.SessionMiddleware("/login/", keywordEditHandler))
	http.HandleFunc("/saved/", users.SessionMiddleware("/login", savedItemsHandler))
	http.HandleFunc("/archiveorg/", users.SessionMiddleware("/login/", archiveOrgHandler))
	http.HandleFunc("/proxy/", users.SessionMiddleware("/login/", proxyHandler))
	staticFileHandler := http.FileServer(http.Dir("./www"))
	http.Handle("/static/", staticFileHandler)

	// extra routes for icons etc.
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/static/icons/favicon.ico")
	})
	http.HandleFunc("/site.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/static/site.webmanifest")
	})
	http.HandleFunc("/apple-touch-icon.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./www/static/icons/apple-touch-icon.png")
	})

	if debug {
		// add debug options here
		log.Printf("Debug mode on.")
	}

	// start a ticker for periodic refresh using the const updateFrequency
	tickerUpdating := time.NewTicker(time.Duration(globalConfig.UpdateFrequency) * time.Minute)
	quit := make(chan int)
	defer close(quit)
	log.Printf("Starting ticker for periodic update (%v minutes).", globalConfig.UpdateFrequency)
	go periodicUpdates(tickerUpdating, quit)

	if debug {
		feeds.UpdateFeeds()
		triggerScoring()
	}

	// serve web app
	log.Print("Starting to serve.")
	err := http.ListenAndServe(":8000", nil)
	log.Println(err)
	tickerUpdating.Stop()
	cache.CleanTicker.Stop()
	log.Println("Stopped tickers.")
	log.Println("Exiting.")
}
