package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func adminPostHandler(w http.ResponseWriter, r *http.Request) {
	var newFeed = Feed{}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form.", http.StatusInternalServerError)
		return
	}
	rawTitle := r.PostForm.Get("title")
	rawAbbr := r.PostForm.Get("abbr")
	rawUrl := r.PostForm.Get("url")
	_, err := url.ParseRequestURI(rawUrl)
//  	fmt.Printf("%v %v %v\n",isAlphaNum(rawTitle), isAlpha(rawAbbr), err)
	if isAlphaNum(rawTitle) && isAlpha(rawAbbr) && err == nil {
		newFeed.Name = rawTitle
		newFeed.Abbr = firstN(rawAbbr, 4)
		newFeed.Url = rawUrl
	} else {
		http.Error(w, "Invalid data.", http.StatusInternalServerError)
		return
	}
	result := db.Create(&newFeed)
	if result.Error != nil {
		http.Error(w, "Could not create feed.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/feeds/", http.StatusSeeOther)
}


func adminGetHandler(w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query()
	if del := q.Get("delete"); del != "" {
		id, err := strconv.Atoi(del)
		if err != nil {
			emitHTMLFromFile(w, r, "./www/header.html")
			defer emitHTMLFromFile(w, r, "./www/footer.html")
			fmt.Fprintf(w,"Error: %v",err)
			return
		}
		result := db.Delete(&Feed{},id)
		if result.Error != nil {
			emitHTMLFromFile(w, r, "./www/header.html")
			defer emitHTMLFromFile(w, r, "./www/footer.html")
			fmt.Fprintf(w,"Error: %v",result.Error)
			return
		}
		http.Redirect(w, r, "/feeds/", http.StatusSeeOther)
	
	} else {
		emitHTMLFromFile(w, r, "./www/header.html")
		defer emitHTMLFromFile(w, r, "./www/footer.html")

		var feeds []Feed
		result := db.Find(&feeds)
		if result.Error != nil {
			fmt.Fprintf(w, "Error loading feeds: %v", result.Error)
			return
		}
		emitHTMLFromFile(w, r, "./www/feed-edit-start.html")
		for _, f := range feeds {
			fmt.Fprintf(w, "<div class=\"row\">")
			fmt.Fprintf(w, "<div class=\"col-2 mb-5\">%v</div>", f.Name)
			fmt.Fprintf(w, "<div class=\"col-1\">%v</div>", f.Abbr)
			fmt.Fprintf(w, "<div class=\"col-5\">%v</div>", f.Url)
			fmt.Fprintf(w, "<div class=\"col-2\"><a href=\"/feeds/?delete=%v\"><button type=\"button\" class=\"btn btn-warning\">Delete</button></a></div>", f.ID)
			fmt.Fprintf(w, "</div>")

		}
		emitHTMLFromFile(w, r, "./www/feed-edit-end.html")
	}

}
