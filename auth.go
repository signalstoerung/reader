package main

/* Authentication functions */

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"

	"golang.org/x/crypto/bcrypt"
)

// Provides a secret key for signing cookies.
// TODO: Remove hard-coded secret and put into configuration file.
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
