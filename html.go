package main

import (
	"net/http"
	"os"
	"fmt"
)

/* functions that help with HTML output */

// emitHTMLFromFile sends HTML from a file to w (if file exists)
func emitHTMLFromFile(w http.ResponseWriter, r *http.Request, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	} 
	fmt.Fprintf(w, string(data))
}

// emitFeedFilterHTML emits the HTML that allows the user to filter the view per feed.
func emitFeedFilterHTML (w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "		<div class=\"col-auto order-5\">")
	defer fmt.Fprintf(w,"		</div>")
	var feeds []Feed
	result := db.Find(&feeds)
	if result.Error != nil {
		return
	}
	for _,f := range feeds {
		fmt.Fprintf(w,"<div><a href=\"/?filter=%v\">%v</a></div>",f.Abbr,f.Abbr)
	}
	fmt.Fprintf(w,"<div><a href=\"/\">Clear</a></div>")
}

