package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/signalstoerung/reader/internal/openai"
	"golang.org/x/exp/slices"
)

// scores the 20 most recent headlines for their breaking news value
func HeadlinesToScore() (string, error) {
	var headlines []Item
	compiledHeadlines := ""
	result := db.Limit(20).Order("published_parsed DESC").Where(&Item{BreakingNewsScore: 0, BreakingNewsReason: ""}).Find(&headlines)
	if result.Error != nil {
		return "", result.Error
	}
	for _, headline := range headlines {
		compiledHeadlines += fmt.Sprintf("- %s (ID: %d)\n", headline.Title, headline.ID)
	}

	return compiledHeadlines, nil
}

func ScoreHeadlines() error {
	var headlines []Item
	compiledHeadlines := ""
	result := db.Raw("SELECT * from items WHERE breaking_news_score = 0 OR breaking_news_score IS NULL ORDER BY published_parsed DESC LIMIT 20").Scan(&headlines)
	if result.Error != nil {
		return result.Error
	}
	for _, headline := range headlines {
		compiledHeadlines += fmt.Sprintf("- %s (ID: %d)\n", headline.Title, headline.ID)
	}

	//	log.Println("Compiled headlines:")
	//	log.Println(compiledHeadlines)
	//log.Println(headlines)

	scored, err := openai.ScoreHeadlines(compiledHeadlines)
	if err != nil {
		return err
	}

	//	log.Println(scored)

	var jsonDecoded map[string]interface{}
	if err := json.Unmarshal([]byte(scored), &jsonDecoded); err != nil {
		return err
	}
	// log.Printf("Decoded JSON object:\n")
	// log.Printf("%+v", jsonDecoded)
	news, ok := jsonDecoded["news"].([]interface{})
	if !ok {
		return fmt.Errorf("expected []map[string]interface{}, got %v", reflect.TypeOf(jsonDecoded["news"]))
	}

	for _, headline := range headlines {
		headlineIdx := slices.IndexFunc(news, func(elem interface{}) bool {
			e, ok := elem.(map[string]interface{})
			if ok {
				switch e["ID"].(type) {
				case float64:
					return uint(e["ID"].(float64)) == headline.ID
				case string:
					id, err := strconv.Atoi(e["ID"].(string))
					if err != nil {
						return false
					}
					return uint(id) == headline.ID
				default:
					log.Printf("Unexpected type for e[ID]")
					return false
				}
			} else {
				log.Printf("elem expected map[string]interface{}, got %v", reflect.TypeOf(elem))
				return false
			}
		})
		if headlineIdx != -1 {
			elem, ok := news[headlineIdx].(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected map[string]interface, got %v", reflect.TypeOf(news[headlineIdx]))
			}
			log.Printf("Headline Scored:\n")
			log.Printf("headline.ID = %d, jsonDecoded.ID = %v", headline.ID, elem["ID"])
			log.Printf("headline.Title = '%s', json headline = %s", headline.Title, elem["headline"])
			score, ok := elem["confidence"].(float64)
			if !ok {
				return errors.New("json 'confidence' not int")
			}
			headline.BreakingNewsScore = int(score)
			reason, ok := elem["reason"].(string)
			if !ok {
				return errors.New("json 'reason' not string")
			}
			headline.BreakingNewsReason = reason
			log.Printf("Score: %d (%s)", int(score), reason)
			result := db.Save(&headline)
			if result.Error != nil {
				return result.Error
			}
		} else {
			// Index not found
			// we still set the breaking news score to something other than 0, so that this headline is not reviewed again and again
			//			log.Printf("This headline was not selected: %s", headline.Title)
			headline.BreakingNewsScore = -1
			headline.BreakingNewsReason = "N/A"
			result := db.Save(&headline)
			if result.Error != nil {
				return result.Error
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
		first, err := firstUnscoredHeadline()
		if err != nil {
			log.Printf("Error in scoring scheduler: %v", err)
			ticker.Stop()
			close(cancel)
			return
		}
		if first.PublishedParsed.Before(time.Now().Add(-1 * time.Hour)) {
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

func firstUnscoredHeadline() (Item, error) {
	var headlines []Item
	result := db.Raw("SELECT * from items WHERE breaking_news_score = 0 OR breaking_news_score IS NULL ORDER BY published_parsed DESC LIMIT 20").Scan(&headlines)
	if result.Error != nil {
		return Item{}, result.Error
	}
	if result.RowsAffected < 15 {
		log.Printf("Only found %v headlines to score, aborting", result.RowsAffected)
		return Item{}, fmt.Errorf("only %v headlines found, aborting", result.RowsAffected)
	}
	// returning last element - this ensures that at least 20 headlines are collected before scoring is triggered.
	return headlines[len(headlines)-1], nil
}

// this should be called after each DB update
func triggerScoring() {
	log.Println("Scoring of headlines triggered")
	tickerScoring := time.NewTicker(1 * time.Minute)
	cancel := make(chan (struct{}))
	go scheduleScoring(tickerScoring, cancel)
}
