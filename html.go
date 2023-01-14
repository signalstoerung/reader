package main

import (
	"net/http"
	"os"
	"fmt"
	"html/template"
	"log"
)


type HeadlinesItem struct {
	Link string
	Title string
	Timestamp string
	FeedAbbr string
}

type HeadlinesPage struct {
	Headlines []HeadlinesItem
	Page int
	HasPreviousPage bool
	PreviousPage int
	NextPage int
	Filter string
}


/* functions that help with HTML output */

// emitHTMLFromFile sends HTML from a file to w (if file exists)
func emitHTMLFromFile(w http.ResponseWriter, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	} 
	fmt.Fprintf(w, string(data))
}

// emitFeedFilterHTML emits the HTML that allows the user to filter the view per feed.
func emitFeedFilterHTML (w http.ResponseWriter) {
	t := template.Must(template.ParseFiles("www/content-feed-filter.html"))

	var feeds []Feed
	result := db.Find(&feeds)
	if result.Error != nil {
		return
	}

	tValue := make([]string,len(feeds))
	
	for i,f := range feeds {
		tValue[i] = f.Abbr
	}
	t.Execute(w, tValue)
}

func returnError (w http.ResponseWriter, errorMsg string) {
	t, err := template.ParseFiles("www/content-row-div.html")
	if err != nil {
		log.Printf("Error parsing HTML template: %v",err)
		// we can still send the original error message when we return an internal server error
		http.Error(w, errorMsg, http.StatusInternalServerError)
	}
	emitHTMLFromFile(w, "www/header.html")
	err = t.Execute(w, errorMsg)
	if err != nil {
		log.Printf("Error executing HTML template: %v", err)
		// we won't get the template output, but we can still print the error message
		fmt.Fprintf(w, errorMsg)
	}
	emitHTMLFromFile(w, "www/footer.html")
}

