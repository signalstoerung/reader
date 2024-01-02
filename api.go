package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/users"
)

// store api tokens
var issuedTokens = make(map[string]string)

type tokenResponse struct {
	Token string
}

func tokenExists(t string) bool {
	for _, storedToken := range issuedTokens {
		if t == storedToken {
			return true
		}
	}
	return false
}

func apiLogin(w http.ResponseWriter, r *http.Request) {
	user := r.Form.Get("username")
	passw := r.Form.Get("password")

	if !isAlpha(user) {
		http.Error(w, "User name may only consist of letters.", http.StatusBadRequest)
		return
	}
	if user == "" || passw == "" {
		http.Error(w, "Missing user ID or password.", http.StatusBadRequest)
		return
	}
	err := users.VerifyUser(user, passw)
	if err != nil {
		log.Printf("User %v could not be verified - %v", user, err)
		http.Error(w, "User verification failed", http.StatusBadRequest)
		return
	} else {
		token, ok := issuedTokens[user]
		if !ok {
			v := signedCookieValue(user, uuid.New().String())
			token = base64.URLEncoding.EncodeToString([]byte(v))
			issuedTokens[user] = token
		}
		var response = tokenResponse{
			Token: token,
		}
		enc := json.NewEncoder(w)
		enc.Encode(response)
	}
}

func apiAddUser(w http.ResponseWriter, r *http.Request) {
	if !registrationsOpen {
		http.Error(w, "Registrations are closed.", http.StatusBadRequest)
		return
	}
	user := r.Form.Get("username")
	passw := r.Form.Get("password")

	if user == "" || passw == "" {
		http.Error(w, "Username or password cannot be empty.", http.StatusBadRequest)
		return
	}

	if !isAlpha(user) {
		http.Error(w, "Username can only consist of letters.", http.StatusBadRequest)
		return
	}
	err := users.CreateUser(user, passw)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Error creating new user.", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "Success")
	registrationsOpen = false // close registrations after first use.
}

func apiFeedList(w http.ResponseWriter) {
	feeds, err := feeds.AllFeeds()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(feeds)
}

func apiDeleteFeed(w http.ResponseWriter, r *http.Request) {
	var feed feeds.Feed
	id, err := strconv.Atoi(r.Form.Get("ID"))
	if err != nil {
		http.Error(w, "Error - ID not a number", http.StatusBadRequest)
		return
	}
	feed.ID = uint(id)
	err = feeds.DeleteFeed(feed)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting feed: %v", err), http.StatusBadRequest)
	} else {
		fmt.Fprintf(w, "Success")
	}
}

func apiAddFeed(w http.ResponseWriter, r *http.Request) {
	var feed feeds.Feed
	title := r.Form.Get("title")
	abbr := r.Form.Get("abbr")
	url, err := url.Parse(r.Form.Get("url"))
	if !isAlphaNum(title) || !isAlpha(abbr) || err != nil {
		http.Error(w, "Invalid characters in name or abbreviation, or invalid URL.", http.StatusBadRequest)
		return
	}
	feed.Name = title
	feed.Abbr = firstN(abbr, 4)
	feed.Url = url.String()
	err = feeds.CreateFeed(feed)
	if err != nil {
		log.Printf("Error writing new feed to database: %v", err)
		http.Error(w, "Error writing to db.", http.StatusInternalServerError)
		return
	} else {
		fmt.Fprint(w, "Success")
	}
}

func apiHeadlines(w http.ResponseWriter, r *http.Request) {
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

	var result []feeds.Item
	result, err = feeds.Items(filter, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading items: %v", err), http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(result)
}
