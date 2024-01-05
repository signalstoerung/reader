package main

import (
	"errors"
	"net/http"
	"net/url"
	"unicode"

	"github.com/signalstoerung/reader/internal/feeds"
)

// isAlphaNum checks if a string is only letters, numbers and spaces (for user-supplied feed titles)
func isAlphaNum(s string) bool {
	for _, l := range s {
		if unicode.IsLetter(l) || unicode.IsNumber(l) || unicode.IsSpace(l) {
			// continue
		} else {
			return false
		}
	}
	return true
}

// isAlpha checks if a string is only letters (for user-supplied feed abbreviations)
func isAlpha(s string) bool {
	for _, l := range s {
		if !unicode.IsLetter(l) {
			return false
		}
	}
	return true
}

// firstN returns the first n letters of a string (to ensure feed abbreviations are max 4 letters)
func firstN(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}

func expandUrlRecursive(shortUrl string) string {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("not so fast, my friend")
		},
	}
	resp, _ := client.Get(shortUrl)
	if resp.StatusCode == 301 {
		return expandUrlRecursive(resp.Header["Location"][0])
	}
	return shortUrl
}

func checkFeedForm(name string, abbr string, formUrl string) (resultItem feeds.Feed, err error) {
	if !isAlphaNum(name) || !isAlpha(abbr) {
		err = errors.New("Name or abbr contains invalid characters")
		return
	}
	_, err = url.Parse(formUrl)
	if err != nil {
		return
	}
	resultItem = feeds.Feed{
		Name: name,
		Abbr: abbr,
		Url:  formUrl,
	}
	return
}
