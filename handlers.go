package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/users"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// we may have gotten an error back through the redirect
		loginerr := r.FormValue("error")

		emitHTMLFromFile(w, "www/header.html")
		templ, err := template.ParseFiles("www/login-form.html")
		if err != nil {
			log.Println(err)
		}
		err = templ.Execute(w, loginerr)
		if err != nil {
			log.Println(err)
		}
		emitHTMLFromFile(w, "www/footer.html")
	}
	if r.Method == http.MethodPost {
		// if we get here, login was successful.
		session, ok := r.Context().Value(users.SessionContextKey).(users.Session)
		if !ok {
			err := fmt.Sprintf("Error retrieving session: expected users.Session, got %v", reflect.TypeOf(r.Context().Value(users.SessionContextKey)))
			log.Println(err)
			http.Error(w, err, http.StatusInternalServerError)
			return
		}
		emitHTMLFromFile(w, "www/header.html")
		templ := template.Must(template.ParseFiles("www/logged-in.html"))
		templ.Execute(w, session)
		emitHTMLFromFile(w, "www/footer.html")
	}
}

func loggedInHandler(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(users.SessionContextKey).(users.Session)
	if !ok {
		err := fmt.Sprintf("Error retrieving session: expected users.Session, got %v", reflect.TypeOf(r.Context().Value(users.SessionContextKey)))
		log.Println(err)
		http.Error(w, err, http.StatusInternalServerError)
		return
	}
	emitHTMLFromFile(w, "www/header.html")
	defer emitHTMLFromFile(w, "www/footer.html")
	templ, err := template.ParseFiles("www/logged-in.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	templ.Execute(w, session)
	if err != nil {
		log.Println(err)
	}
}

func headlinesHandler(w http.ResponseWriter, r *http.Request) {
	feedlist := getAllFeedsFromCacheOrDB().([]feeds.Item)
	feed := r.FormValue("feed")
	// check if 'feed' exists, if not, set it to ""
	if !slices.ContainsFunc(feedlist, func(elem feeds.Item) bool {
		return elem.FeedAbbr == feed
	}) {
		feed = ""
	}
	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil {
		page = 1
	}
	if page < 1 {
		page = 1
	}
	// page 1 --> index 0
	offset := (page - 1) * globalConfig.ResultsPerPage

	headlines := getItemsFromCacheOrDB(feed, globalConfig.ResultsPerPage, offset).([]feeds.Item)
	pageData := make(map[string]interface{})
	pageData["Headlines"] = ConvertItems(headlines)
	pageData["Feeds"] = feedlist
	// pageData["Message"] = "Fake it till ya make it."
	pageData["Page"] = page
	pageData["PrevPageLink"] = fmt.Sprintf("%s?page=%d&feed=%s", r.URL.Path, page-1, feed)
	if len(headlines) < globalConfig.ResultsPerPage {
		pageData["NextPageLink"] = ""
	} else {
		pageData["NextPageLink"] = fmt.Sprintf("%s?page=%d&feed=%s", r.URL.Path, page+1, feed)
	}

	emitHTMLFromFile(w, "www/header.html")
	defer emitHTMLFromFile(w, "www/footer.html")
	templ := template.Must(template.ParseFiles("www/main.html"))
	templ.Execute(w, pageData)
}

func archiveOrgHandler(w http.ResponseWriter, r *http.Request) {
	searchUrl := r.FormValue("url")
	if searchUrl == "" {
		http.Error(w, "No URL provided", http.StatusBadRequest)
		return
	}
	requestURL := fmt.Sprintf("http://archive.org/wayback/available?url=%s&timestamp=%s", searchUrl, time.Now().Format("20060102"))
	resp, err := http.Get(requestURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		errText := fmt.Sprintf("Error retrieving from archive.org: %v / %v", err, resp.Status)
		http.Error(w, errText, http.StatusInternalServerError)
		return
	}
	var jsonDecoded map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jsonDecoded)
	if err != nil {
		log.Printf("Error decoding JSON: %v", err)
		return
	}
	if _, ok := jsonDecoded["archived_snapshots"]; ok {
		archivedSnapshots, ok := jsonDecoded["archived_snapshots"].(map[string]interface{})
		if ok {
			closest, ok := archivedSnapshots["closest"].(map[string]interface{})
			if ok {
				archiveUrl, ok := closest["url"].(string)
				if ok {
					log.Printf("Found wayback URL %v for %v", closest["url"], searchUrl)
					http.Redirect(w, r, archiveUrl, http.StatusFound)
					return
				}
			}
		} else {
			log.Printf("Type assertion failed. Expected map[string]interface{}, got %v", reflect.TypeOf(jsonDecoded["archived_snapshots"]))
		}
	}
	// getting here means error
	log.Printf("Expected archived_snapshots:closest, got %v", jsonDecoded)
	http.Error(w, "Unable to retrieve archive link", http.StatusInternalServerError)
}

// emitHTMLFromFile sends HTML from a file to w (if file exists)
func emitHTMLFromFile(w http.ResponseWriter, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	fmt.Fprint(w, string(data))
}
