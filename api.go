package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

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
	var maybeUser User
	result := db.Where(User{UserName: user}).First(&maybeUser)
	if result.Error != nil {
		http.Error(w, "Wrong user name or password.", http.StatusBadRequest)
		return
	}
	if err := maybeUser.verifyPassword(passw); err != nil {
		// wrong password supplied
		http.Error(w, "Wrong user name or password.", http.StatusBadRequest)
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

func apiFeeds(w http.ResponseWriter) {
	var feeds []Feed
	result := db.Find(&feeds)
	if result.Error != nil {
		http.Error(w, fmt.Sprintf("Error: %v", result.Error), http.StatusInternalServerError)
		return
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(feeds)

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

	result := make([]HeadlinesItem, limit)
	err = loadItems(db, &result, filter, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading items: %v", err), http.StatusInternalServerError)
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(result)
}
