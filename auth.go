package main

/* Authentication functions */

import (
	"net/http"
	"fmt"
	"errors"
	"log"
    "encoding/base64"
    "encoding/hex"
    "crypto/hmac"  
    "crypto/sha256"
    "golang.org/x/crypto/bcrypt"
    "github.com/google/uuid"
)

func secretKey() []byte {
	const secret = "23f7b439110cdae1bc133e42565fe17d5eb7dfec4a2522cc923e4aa313a12083" 
	key, err := hex.DecodeString(secret)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

func (u *User) setPassword(pw string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.Password = string(hash)
    log.Printf("Created new password for %s.",u.UserName)
	return nil
}

func (u *User) verifyPassword(pw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pw))
	if err != nil {
		log.Printf("User %s supplied wrong password: %s",u.UserName, pw)
		return err
	}
	log.Printf("User %s supplied correct password.",u.UserName)
	return nil
}

func (sessions *UserSessions) sessionExists(userName string) (int, error) {
	for i, u := range *sessions {
		if u.UserName == userName {
			return i, nil
		}
	}
	errormsg := fmt.Sprintf("No session found for %s.",userName)
	return 0, errors.New(errormsg)
}

func signedCookieValue (cookieName string, cookieValue string) string {
		// create signature for value
		mac := hmac.New(sha256.New, secretKey())
		mac.Write([]byte(cookieName))
		mac.Write([]byte(cookieValue))
		signature := mac.Sum(nil)

		// combine value with signature		
		return string(signature)+cookieValue	
}

func isAuthenticated (r *http.Request) bool {
	// get all cookies from the request
	cookies := r.Cookies()	
	if len(cookies) == 0 {
		log.Println("No cookie found.")
		return false
	}

	// there should only be one cookie sent, so we just get the first one
	cookie := cookies[0]

	i, err := userSessions.sessionExists(cookie.Name)
	if err != nil {
		log.Printf("No active session found for user %s.",cookie.Name)
		return false
	} 
	
	// get user name and active session ID for user
	userName := userSessions[i].UserName
	sessionId := userSessions[i].sessionId
	log.Printf("found session for %s with id %s", userName, sessionId)


	// generate expected cookie value from session ID for later comparison
	expectedValue := signedCookieValue(userName, sessionId.String())

	// decode cookie value	
	foundValue, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("cookie value not base64 encoded: %v",cookie.Value)
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

func checkPassword (w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form.", http.StatusInternalServerError)
		return
	}
	rawUserId := r.PostForm.Get("userid")
	rawPassword := r.PostForm.Get("password")
	
	if rawUserId == "" || rawPassword == "" {
		fmt.Fprint(w, "Missing user ID or password.")
		return
	}

	var maybeUser User
	result := db.Where(User{ UserName:rawUserId }).First(&maybeUser)
	if result.Error != nil {
		fmt.Fprint(w, "No user found or other error.")
		log.Printf("Error with user lookup: %v", result.Error)
		return
	}

	if err := maybeUser.verifyPassword(rawPassword); err != nil {
		// wrong password supplied
		fmt.Fprint(w, "Wrong password.")
		return
	} else {
		// correct password supplied
		
		// create session ID
		maybeUser.sessionId = uuid.New()
		
		log.Printf("Setting cookie for user %s with sessionid %s", maybeUser.UserName, maybeUser.sessionId.String())
		
		cookie := http.Cookie {
			Name: maybeUser.UserName,
			Value: maybeUser.sessionId.String(),
			Path: "/",
			MaxAge: 21600,
			HttpOnly: true,
			Secure: false,
			SameSite: http.SameSiteLaxMode,
		}
		
		cookie.Value = signedCookieValue(cookie.Name, cookie.Value)

		// encode value with base 64
		cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))
		
		http.SetCookie(w, &cookie)

		userSessions = append(userSessions, maybeUser)

		log.Printf("Successfully set the cookie %v", cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		
	}
}