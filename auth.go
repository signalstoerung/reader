package main

/* Cookies */

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
)

// Provides a secret key for signing cookies.
func secretKey() []byte {
	key, err := hex.DecodeString(globalConfig.Secret)
	if err != nil {
		log.Fatal(err)
	}
	return key
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
