package users

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Session struct {
	Id    string
	User  string
	Admin bool
}

type CustomClaims struct {
	jwt.RegisteredClaims
	Admin bool `json:"adm"`
}

type ContextKey string

const (
	tokenExpiryDuration time.Duration = 5 * time.Minute // set to 5 minutes for testing purposes only
	SessionContextKey   ContextKey    = "session"
	ErrLoginFailed                    = "login failed"
	ErrMustUsePost                    = "must use POST"
	ErrInvalidToken                   = "token invalid"
	ErrInvalidAudience                = "audience invalid"
)

func init() {
	key := make([]byte, 64)
	_, err := rand.Read(key)
	if err != nil {
		log.Printf("Unexpected error creating signing key: %v", err)
		panic(err)
	}
	log.Printf("DEBUG INFO (REMOVE FROM PRODUCTION) signing key created: %v", key)
	Config.Secret = key
}

func createJwt(user string, admin bool) (string, error) {
	signingKey := Config.Secret
	claims := CustomClaims{
		Admin: admin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "unxpctd.xyz",
			Subject:   user,
			Audience:  []string{"unxpctd.xyz"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiryDuration)),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(signingKey)
	return ss, err
}

func signingKey(t *jwt.Token) (interface{}, error) {
	signingKey := Config.Secret
	return signingKey, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// returns session Id if jwt is valid, otherwise error
func decodeJwt(tokenString string) (Session, error) {
	//	log.Printf("Attempting to decode token: %v", tokenString)
	token, err := jwt.Parse(tokenString, signingKey, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return Session{}, err
	}
	if !token.Valid {
		return Session{}, fmt.Errorf("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)

	// check audience
	audience, err := claims.GetAudience()
	if err != nil {
		return Session{}, err
	}
	if !contains(audience, "unxpctd.xyz") {
		log.Printf("Invalid audience: %v", audience)
		return Session{}, fmt.Errorf("invalid audience")
	}
	// return Session object
	session := Session{
		User:  claims["sub"].(string),
		Admin: claims["adm"].(bool),
		Id:    claims["jti"].(string),
	}
	return session, nil
}

func LoginMiddleware(loginFailedRedirect string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// POST = someone is trying to log in
		if r.Method == http.MethodPost {
			user := r.FormValue("userid")
			pass := r.FormValue("password")
			err := VerifyUser(user, pass)
			if err != nil {
				// not authenticated
				urlParams := url.Values{}
				urlParams.Add("error", ErrLoginFailed)
				redirectUrl := loginFailedRedirect + "?" + urlParams.Encode()
				log.Printf("Login of user %v failed, redirecting to %v", user, redirectUrl)
				http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
				return
			} else {
				token, err := createJwt(user, false) // Admin field not implemented
				if err != nil {
					log.Printf("error creating jwt: %v", err)
					http.Error(w, "error creating jwt", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     "jwt-session",
					Value:    token,
					Path:     "/",
					Expires:  time.Now().Add(tokenExpiryDuration),
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
				session := Session{
					User:  user,
					Admin: false, // Admin not implemented, hard-coded as false
					Id:    "new", // will get replaced with the JWT ID the first time the cookie is read
				}
				ctx := context.WithValue(r.Context(), SessionContextKey, session)
				next(w, r.WithContext(ctx))
				return
			}
		} else {
			// GET = do nothing, next handler will show login form
			next(w, r)
			return
		}
	}
}

// looks for a cookie that identifies the user; if not found, redirects to url provided as noSessionRedirect
func SessionMiddleware(noSessionRedirect string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			cookie, err := r.Cookie("jwt-session")
			if err == nil {
				// we have a cookie, so we can decode it
				sess, err := decodeJwt(cookie.Value)
				if err != nil {
					log.Printf("Error decoding jwt: %v. Redirecting.", err)
					// deleting cookie...may or may not work?!
					http.SetCookie(w, &http.Cookie{Name: "jwt-session", Value: "", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
					// redirect to login screen
					http.Redirect(w, r, noSessionRedirect, http.StatusSeeOther)
					return
				}
				log.Printf("Found JWT for user %v", sess.User)
				ctx := context.WithValue(r.Context(), SessionContextKey, sess)
				next(w, r.WithContext(ctx))
				return
			}

			// if we get here then there's no cookie, so not authorized
			// redirect to url provided
			log.Printf("No cookie found, not authorized, redirecting")
			http.Redirect(w, r, noSessionRedirect, http.StatusSeeOther)
			return
		}
	}
}
