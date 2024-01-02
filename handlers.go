package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"reflect"

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

// emitHTMLFromFile sends HTML from a file to w (if file exists)
func emitHTMLFromFile(w http.ResponseWriter, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	fmt.Fprint(w, string(data))
}
