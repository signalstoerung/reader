package main

/* Authentication functions */

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Provides a secret key for signing cookies.
func secretKey() []byte {
	key, err := hex.DecodeString(globalConfig.Secret)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

// sets the password for a User
func (u *User) setPassword(pw string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	log.Printf("Created new password for %s.", u.UserName)
	return nil
}

// verifies a user-provided password
func (u *User) verifyPassword(pw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pw))
	if err != nil {
		log.Printf("User %s supplied wrong password: %s", u.UserName, pw)
		return err
	}
	log.Printf("User %s supplied correct password.", u.UserName)
	return nil
}

// signedCookieValue takes the cookie name and cookie value and returns a signed value
func signedCookieValue(cookieName string, cookieValue string) string {
	// create signature for value
	mac := hmac.New(sha256.New, secretKey())
	mac.Write([]byte(cookieName))
	mac.Write([]byte(cookieValue))
	signature := mac.Sum(nil)

	// combine value with signature
	return string(signature) + cookieValue
}

// isAuthenticated is called by every URL handler that requires the user to be logged in.
// It checks if a cookie is present and verifies it.
func isAuthenticated(r *http.Request) bool {
	// get all cookies from the request
	cookies := r.Cookies()
	if len(cookies) == 0 {
		if logDebugLevel {
			log.Println("No cookie found.")
		}
		return false
	}

	// there should only be one cookie sent, so we just get the first one
	cookie := cookies[0]

	u, ok := userSessions[cookie.Name]
	if !ok {
		if logDebugLevel {
			log.Printf("No active session found for user %s.", cookie.Name)
		}
		return false
	}

	// get user name and active session ID for user
	userName := u.UserName
	sessionId := u.sessionId

	if logDebugLevel {
		log.Printf("found session for %s with id %s", userName, sessionId)
	}

	// generate expected cookie value from session ID for later comparison
	expectedValue := signedCookieValue(userName, sessionId.String())

	// decode cookie value
	foundValue, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("cookie value not base64 encoded: %v", cookie.Value)
	}

	// Check that the recalculated signature matches the signature we received
	// in the cookie. If they match, we can be confident that the cookie name
	// and value haven't been edited by the client.
	if !hmac.Equal([]byte(foundValue), []byte(expectedValue)) {
		log.Print("Cookie signature doesn't match.")
		return false
	}

	return true
}

// checkPassword is called by the http handler to process a login form.
// redirects user to "/" if successful.
func checkPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form.", http.StatusInternalServerError)
		return
	}
	rawUserId := r.PostForm.Get("userid")

	if !isAlpha(rawUserId) {
		returnError(w, "User name may only consist of letters.")
		return
	}
	rawPassword := r.PostForm.Get("password")

	if rawUserId == "" || rawPassword == "" {
		returnError(w, "Missing user ID or password.")
		return
	}

	var maybeUser User
	result := db.Where(User{UserName: rawUserId}).First(&maybeUser)
	if result.Error != nil {
		returnError(w, "Sorry, we couldn't log you in with that user name or password.")
		log.Printf("Error with user lookup: %v", result.Error)
		return
	}

	if err := maybeUser.verifyPassword(rawPassword); err != nil {
		// wrong password supplied
		returnError(w, "Sorry, we couldn't log you in with that user name or password.")
		return
	} else {
		// correct password supplied

		// create session ID
		maybeUser.sessionId = uuid.New()

		if logDebugLevel {
			log.Printf("Setting cookie for user %s with sessionid %s", maybeUser.UserName, maybeUser.sessionId.String())
		}

		cookie := http.Cookie{
			Name:     maybeUser.UserName,
			Value:    maybeUser.sessionId.String(),
			Path:     "/",
			MaxAge:   21600,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}

		cookie.Value = signedCookieValue(cookie.Name, cookie.Value)

		// encode value with base 64
		cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))

		http.SetCookie(w, &cookie)

		userSessions[maybeUser.UserName] = maybeUser

		http.Redirect(w, r, "/", http.StatusSeeOther)

	}
}

// registerNewUser processes the form for user registration
func registerNewUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		returnError(w, "Could not parse form.")
		return
	}
	rawUserId := r.PostForm.Get("userid")
	rawPassword := r.PostForm.Get("password")

	if rawUserId == "" || rawPassword == "" {
		returnError(w, "Missing user ID or password.")
		return
	}

	if !isAlpha(rawUserId) {
		returnError(w, "User name can only consist of letters.")
		return
	}

	newUser := User{UserName: rawUserId}
	err := newUser.setPassword(rawPassword)
	if err != nil {
		msg := fmt.Sprintf("Error creating new user: password could not be set (%s)", err)
		returnError(w, msg)
		return
	}

	result := db.Create(&newUser)
	if result.Error != nil {
		msg := fmt.Sprintf("Error creating new user: %s", result.Error)
		returnError(w, msg)
	}

	emitHTMLFromFile(w, "./www/header.html")
	t := template.Must(template.ParseFiles("www/content-row-div.html"))
	t.Execute(w, "User created. You can now return to the homepage and log in.")
	emitHTMLFromFile(w, "./www/footer.html")

	// close registrations - we're assuming this is a single-user instance.
	registrationsOpen = false
}
