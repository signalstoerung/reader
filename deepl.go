package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type APIResponse struct {
	Translations []Translation `json:"translations"`
}

type Translation struct {
	SourceLanguage string `json:"detected_source_language"`
	Text           string `json:"text"`
}

var Client = http.Client{}

func translate(source string) (string, error) {
	if globalConfig.DeeplApiKey == "" {
		return "", errors.New("API key missing from config file")
	}
	v := url.Values{}
	v.Add("target_lang", "EN-US")
	v.Add("text", source)
	log.Printf("Values: %v", v.Encode())
	request, err := http.NewRequest(http.MethodPost, "https://api-free.deepl.com/v2/translate", strings.NewReader(v.Encode()))
	if err != nil {
		log.Printf("Error in new request: %v", err)
		return "", err
	}

	request.Header.Add("Authorization", fmt.Sprintf("DeepL-Auth-Key %v", globalConfig.DeeplApiKey))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := Client.Do(request)
	if err != nil {
		log.Printf("Error in Client.Do: %v", err)
		return "", err
	}
	defer response.Body.Close()
	// body, err := io.ReadAll(response.Body)

	// log.Printf("Got response: %v", string(body))
	// return "", err
	decoder := json.NewDecoder(response.Body)
	var apiResp = APIResponse{}
	err = decoder.Decode(&apiResp)
	if err != nil {
		log.Printf("Error in JSON.Decode: %v", err)
		return "", err
	}
	return apiResp.Translations[0].Text, nil
}
