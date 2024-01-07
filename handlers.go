package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/users"
	"golang.org/x/exp/slices"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// we may have gotten an error back through the redirect
		loginerr := r.FormValue("error")

		emitHTMLFromFile(w, HTMLHeaderPath)
		templ, err := template.ParseFiles(HTMLLoginFormPath)
		if err != nil {
			log.Println(err)
		}
		err = templ.Execute(w, loginerr)
		if err != nil {
			log.Println(err)
		}
		emitHTMLFromFile(w, HTMLFooterPath)
	}
	if r.Method == http.MethodPost {
		// if we get here, login should have been successful.
		_, ok := r.Context().Value(users.SessionContextKey).(users.Session)
		if !ok {
			err := fmt.Sprintf("Error retrieving session: expected users.Session, got %v", reflect.TypeOf(r.Context().Value(users.SessionContextKey)))
			log.Println(err)
			http.Error(w, err, http.StatusInternalServerError)
			return
		}
		// redirect to homepage
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func headlinesHandler(w http.ResponseWriter, r *http.Request) {
	feedlist := getAllFeedsFromCacheOrDB().([]feeds.Feed)
	feed := r.FormValue("feed")
	// check if 'feed' exists, if not, set it to ""
	if !slices.ContainsFunc(feedlist, func(elem feeds.Feed) bool {
		return elem.Abbr == feed
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

	emitHTMLFromFile(w, HTMLHeaderPath)
	defer emitHTMLFromFile(w, HTMLFooterPath)
	templ := template.Must(template.ParseFiles("www/main.html"))
	templ.Execute(w, pageData)
	session, ok := r.Context().Value(users.SessionContextKey).(users.Session)
	if !ok {
		log.Println("WARNING: no context found / page served anyway")
		log.Printf("%+v", r)
	} else {
		log.Printf("/items/%v/%v (user: %v)", feed, page, session.User)
	}

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

func feedEditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// show form
		emitHTMLFromFile(w, HTMLHeaderPath)
		defer emitHTMLFromFile(w, HTMLFooterPath)
		pageData := make(map[string]interface{})
		feedlist, err := feeds.AllFeeds()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pageData["Feeds"] = feedlist
		pageData["PageUrl"] = r.URL.Path
		templ := template.Must(template.ParseFiles(HTMLFeedFormPath))
		templ.Execute(w, pageData)
		return
	}
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		var resultMessage string
		switch r.FormValue("action") {
		case "add":
			feed, err := checkFeedForm(r.FormValue("name"), r.FormValue("abbr"), r.FormValue("url"))
			if err != nil {
				resultMessage = fmt.Sprintf("Adding feed failed. (%v)", err)
			} else {
				err = feeds.CreateFeed(feed)
				if err != nil {
					resultMessage = fmt.Sprintf("Creating feed failed. (%v)", err)
				} else {
					resultMessage = fmt.Sprintf("Feed %v successfully created.", feed.Name)
				}
			}
		case "delete":
			id, err := strconv.Atoi(r.FormValue("ID"))
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid ID: %v", err), http.StatusBadRequest)
				return
			}
			err = feeds.DeleteFeedById(uint(id))
			if err != nil {
				resultMessage = fmt.Sprintf("Error trying to delete feed: %v", err)
			} else {
				resultMessage = "Feed deleted."
			}
		default:
			http.Error(w, "Action not specified", http.StatusBadRequest)
			return
		}
		emitHTMLFromFile(w, HTMLHeaderPath)
		defer emitHTMLFromFile(w, HTMLFooterPath)
		templ := template.Must(template.ParseFiles(HTMLFeedFormResultPath))
		templ.Execute(w, resultMessage)
		return
	}
	http.Error(w, "Method not allowed", http.StatusBadRequest)
	// get directly from DB to avoid caching issues
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		emitHTMLFromFile(w, HTMLHeaderPath)
		var pageData = make(map[string]interface{})
		if registrationsOpen {
			pageData["signupsOpen"] = true
		}
		templ := template.Must(template.ParseFiles(HTMLRegisterFormPath))
		templ.Execute(w, pageData)
		emitHTMLFromFile(w, HTMLFooterPath)
		return
	}
	if r.Method == http.MethodPost {
		returnMessage := ""
		if !registrationsOpen {
			returnMessage = "Sorry, registrations are close"
		}
		if registrationsOpen {
			username := r.FormValue("userid")
			password := r.FormValue("password")
			if username == "" || password == "" {
				returnMessage = "Username or password missing"
			}
			if !isAlpha(username) {
				returnMessage = "Username should only consist of letters."
			}
			err := users.CreateUser(username, password)
			if err != nil {
				returnMessage = fmt.Sprintf("Error creating new user: %v", err)
			} else {
				returnMessage = "Account created. You can now log in."
			}
		}
		emitHTMLFromFile(w, HTMLHeaderPath)
		templ := template.Must(template.ParseFiles(HTMLLoginFormPath))
		templ.Execute(w, returnMessage)
		emitHTMLFromFile(w, HTMLFooterPath)
		return
	}
	http.Error(w, "Method not allowed", http.StatusBadRequest)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

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
