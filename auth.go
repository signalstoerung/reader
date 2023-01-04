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
)

func secretKey() []byte {
	const secret = "23f7b439110cdae1bc133e42565fe17d5eb7dfec4a2522cc923e4aa313a12083" 
	key, err := hex.DecodeString(secret)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

func isAuthenticated (r *http.Request) bool {
	cookieName := "readerlogin"
	cookieExpectedValue := "just trying things out here."

    cookie, err := r.Cookie(cookieName)
    if err != nil {
		switch {
			case errors.Is(err, http.ErrNoCookie):
				log.Println("No cookie found.")
			default:
				log.Println(err)
		}
		return false
	}
	
	signedValue, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("cookie value not base64 encoded: %v",cookie.Value)
	}

    // A SHA256 HMAC signature has a fixed length of 32 bytes. To avoid a potential
    // 'index out of range' panic in the next step, we need to check sure that the
    // length of the signed cookie value is at least this long. We'll use the 
    // sha256.Size constant here, rather than 32, just because it makes our code
    // a bit more understandable at a glance.
	  if len(signedValue) < sha256.Size {
        log.Print("Invalid cookie length.")
        return false
    }

    // Split apart the signature and original cookie value.
    signature := signedValue[:sha256.Size]
    value := signedValue[sha256.Size:]
    log.Printf("INFO: cookie value %v, signature %v", value, signature)

    // Recalculate the HMAC signature of the cookie name and original value.
    mac := hmac.New(sha256.New, secretKey())
    mac.Write([]byte(cookieName))
    mac.Write([]byte(cookieExpectedValue))
    expectedSignature := mac.Sum(nil)

// Check that the recalculated signature matches the signature we received
    // in the cookie. If they match, we can be confident that the cookie name
    // and value haven't been edited by the client.
    if !hmac.Equal([]byte(signature), expectedSignature) {
    	log.Print("Cookie signature doesn't match.")
        return false
    }
	
	if string(value) == "just trying things out here." {
		log.Print("Authenticated via cookie.")
		return true	
	}
	return false
}

func checkPassword (w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Could not parse form.", http.StatusInternalServerError)
		return
	}
	rawUserId := r.PostForm.Get("userid")
	rawPassword := r.PostForm.Get("password")
	if rawUserId == "admin" && rawPassword == "M47Ks8eMJK4z" {
		cookie := http.Cookie {
			Name: "readerlogin",
			Value: "just trying things out here.",
			Path:     "/",
			MaxAge: 120,
			HttpOnly: true,
			Secure: false,
			SameSite: http.SameSiteLaxMode,
		}

		// create signature for value
		mac := hmac.New(sha256.New, secretKey())
		mac.Write([]byte(cookie.Name))
		mac.Write([]byte(cookie.Value))
		signature := mac.Sum(nil)

		// combine value with signature		
		cookie.Value=string(signature)+cookie.Value

		// encode value with base 64 for special characters
		cookie.Value = base64.URLEncoding.EncodeToString([]byte(cookie.Value))
		
		http.SetCookie(w, &cookie)

		log.Printf("Successfully set the cookie %v", cookie)
	} else {
		fmt.Fprint(w, "Wrong user/password combination.")
		return
	}	
}