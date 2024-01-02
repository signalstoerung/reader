package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/openai"
	"golang.org/x/exp/slices"
)

func ScoreHeadlines() error {
	// get a batch of unscored headlines
	headlines, err := feeds.UnscoredHeadlines()
	if err != nil {
		log.Printf("Error retrieving unscored headlines: %v", err)
		return err
	}
	// compile them into one neatly printed string that can be attached to the OpenAI prompt
	compiledHeadlines := ""
	for _, headline := range headlines {
		compiledHeadlines += fmt.Sprintf("- %s (ID: %d)\n", headline.Title, headline.ID)
	}

	// get recent breaking news headlines for context
	recent, ok := feeds.RecentBreakingNews()
	if !ok {
		log.Printf("Error retrieving recent headlines.")
		recent = []string{}
	}

	// get AI to score the headlines - returns a JSON string
	scored, err := openai.ScoreHeadlines(compiledHeadlines, recent)
	if err != nil {
		return err
	}

	// decode JSON string into map[string]interface{}
	var jsonDecoded map[string]interface{}
	if err := json.Unmarshal([]byte(scored), &jsonDecoded); err != nil {
		return err
	}

	// headlines should be an array in the 'news' field
	// check if the type is as expected
	news, ok := jsonDecoded["news"].([]interface{})
	if !ok {
		return fmt.Errorf("expected []interface{}, got %v", reflect.TypeOf(jsonDecoded["news"]))
	}

	// Now we iterate over the original headlines ([]Item) and check if we find a match in what we got back from the OpenAI API
	for _, headline := range headlines {
		// get index of match
		headlineIdx := slices.IndexFunc(news, func(elem interface{}) bool {
			// first we assert that the element of 'news' is a map[string]interface{}, i.e. an unmarshalled JSON object
			// we reassign it to e so that we get proper compiler type checks
			e, ok := elem.(map[string]interface{})
			if ok {
				// during testing, OpenAI sometimes returned a JSON number (ID: 1), sometimes a string (ID: "1"), so we work with both
				switch e["ID"].(type) {
				case float64:
					// returns true if the ID of elem is the same as headline.ID == we have a match
					return uint(e["ID"].(float64)) == headline.ID
				case string:
					id, err := strconv.Atoi(e["ID"].(string))
					if err != nil {
						return false
					}
					// same as above, but then after a string->int conversion
					return uint(id) == headline.ID
				default:
					// log unexpected type
					log.Printf("Unexpected type for e[ID]: %v", reflect.TypeOf(e["ID"]))
					return false
				}
			} else {
				log.Printf("elem expected map[string]interface{}, got %v", reflect.TypeOf(elem))
				return false
			}
		})
		// match: Index is NOT -1
		if headlineIdx != -1 {
			// confirm type so that compiler type checks work correctly
			elem, ok := news[headlineIdx].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected map[string]interface, got %v", reflect.TypeOf(news[headlineIdx]))
			}
			log.Printf("Headline Scored:\n")
			log.Printf("headline.ID = %d, jsonDecoded.ID = %v", headline.ID, elem["ID"])
			log.Printf("headline.Title = '%s', json headline = %s", headline.Title, elem["headline"])
			// in testing, confidence was reliably a float
			score, ok := elem["confidence"].(float64)
			if !ok {
				return errors.New("json 'confidence' not float64")
			}
			headline.BreakingNewsScore = int(score)
			reason, ok := elem["reason"].(string)
			if !ok {
				return errors.New("json 'reason' not string")
			}
			headline.BreakingNewsReason = reason
			log.Printf("Score: %d (%s)", int(score), reason)
			err := feeds.SaveItem(headline)
			if err != nil {
				return err
			}
		} else {
			// Index not found
			// we still set the breaking news score to something other than 0, so that this headline is not reviewed again and again
			//			log.Printf("This headline was not selected: %s", headline.Title)
			headline.BreakingNewsScore = -1
			headline.BreakingNewsReason = "N/A"
			err = feeds.SaveItem(headline)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// func openAITestHandler(w http.ResponseWriter, r *http.Request) {
// 	err := ScoreHeadlines()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	fmt.Fprintf(w, "Success!\n")
// }

// Launch this as a go routine
func goScore(cancel chan struct{}) {
	err := ScoreHeadlines()
	if err != nil {
		log.Printf("An error occurred running ScoreHeadlines: %v", err)
		log.Printf("Sending message to cancel ticker")
		cancel <- struct{}{}
	}
}

func scheduleScoring(ticker *time.Ticker, cancel chan struct{}) {
	log.Println("Scheduling loop started")
	defer log.Println("Scheduling loop stopped")

Outerloop:
	for {
		first, err := feeds.FirstUnscoredHeadline()
		if err != nil {
			log.Printf("Error in scoring scheduler: %v", err)
			ticker.Stop()
			close(cancel)
			return
		}
		if first.PublishedParsed.Before(time.Now().Add(-5 * time.Hour)) {
			log.Printf("No recent headlines to score (found date: %v).", first.PublishedParsed.Format("Jan 2 - 15:04"))
			ticker.Stop()
			close(cancel)
			return
		}
		select {
		case <-cancel:
			ticker.Stop()
			log.Println("Received cancel signal for HeadlineScoring ticker.")
			break Outerloop
		case <-ticker.C:
			goScore(cancel)
		}
	}
}

// this should be called after each DB update
func triggerScoring() {
	log.Println("Scoring of headlines triggered")
	tickerScoring := time.NewTicker(1 * time.Minute)
	cancel := make(chan (struct{}))
	go scheduleScoring(tickerScoring, cancel)
}

func breakingTestHandler(w http.ResponseWriter, r *http.Request) {
	recent, ok := feeds.RecentBreakingNews()
	if ok {
		fmt.Fprintln(w, "Recent headlines:")
		for _, hl := range recent {
			fmt.Fprintf(w, "- %s\n", hl)
		}
	}
}
