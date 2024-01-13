package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/openai"
)

func errPanic(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type OpenAIReturn struct {
	News []ScoredHeadline `json:"news"`
}

type ScoredHeadline struct {
	ID         interface{} `json:"ID"`
	Headline   string      `json:"headline"`
	Confidence int         `json:"confidence"`
	Reason     string      `json:"reason"`
}

func scoreChunk(headlines []feeds.Item) (items []feeds.Item, err error) {
	compiledHeadlines := ""
	for _, headline := range headlines {
		compiledHeadlines += fmt.Sprintf("- %s (ID: %d)\n", headline.Title, headline.ID)
	}
	// get AI to score the headlines - returns a JSON string
	scored, err := openai.ScoreHeadlines(compiledHeadlines, []string{})
	if err != nil {
		return
	}

	// decode JSON string into map[string]interface{}
	var jsonDecoded OpenAIReturn
	if err = json.Unmarshal([]byte(scored), &jsonDecoded); err != nil {
		return
	}

	if len(jsonDecoded.News) == 0 {
		log.Printf("%+v", scored)
		log.Printf("%+v", jsonDecoded)
		log.Println(".News is empty!")
		return
	}

	scoredHeadlines := jsonDecoded.News

	// iterate over headlines, look for matches
	for _, headline := range headlines {
		idx := slices.IndexFunc(scoredHeadlines, func(e ScoredHeadline) bool {
			var id uint
			switch e.ID.(type) {
			case float64:
				id = uint(e.ID.(float64))
			case string:
				idInt, err := strconv.Atoi(e.ID.(string))
				if err != nil {
					return false
				}
				id = uint(idInt)
			}
			return id == headline.ID
		})
		var newItem = feeds.Item{
			ID:    headline.ID,
			Title: headline.Title,
		}
		if idx > -1 {
			newItem.BreakingNewsReason = scoredHeadlines[idx].Reason
			newItem.BreakingNewsScore = scoredHeadlines[idx].Confidence
		} else {
			newItem.BreakingNewsReason = "N/A"
			newItem.BreakingNewsScore = -1
		}
		items = append(items, newItem)
	}
	return
}

func scoreHeadlines(headlines []feeds.Item, prompt string) (scoredItems []feeds.Item, err error) {
	const chunkSize = 20
	var counter int = 0
	openai.SetGptPrompt(prompt)
	for counter < len(headlines)-(chunkSize-1) {
		log.Println(counter)
		items, err := scoreChunk(headlines[counter : counter+chunkSize])
		errPanic(err)
		scoredItems = append(scoredItems, items...)
		counter += chunkSize
	}
	return
}

func main() {
	var prompt1Path string
	var prompt2Path string
	var dbPath string
	var openAIKey string
	flag.StringVar(&prompt1Path, "p1", "prompt1.txt", "Path to the file containing prompt #1")
	flag.StringVar(&prompt2Path, "p2", "prompt2.txt", "Path to the file containing prompt #2")
	flag.StringVar(&dbPath, "db", "/Users/nimi/Documents/dev/go/reader/db/reader.db", "Path to the database")
	flag.StringVar(&openAIKey, "apikey", "", "OpenAI API key")
	flag.Parse()

	if openAIKey == "" {
		log.Fatal("No API key provided")
	}
	openai.Stats.ApiKey = openAIKey

	// load prompts
	p1, err := os.Open(prompt1Path)
	errPanic(err)
	p2, err := os.Open(prompt2Path)
	errPanic(err)
	prompt1, err := io.ReadAll(p1)
	errPanic(err)
	prompt2, err := io.ReadAll(p2)
	errPanic(err)

	err = feeds.Config.OpenDatabase(dbPath)
	errPanic(err)
	headlinesToTest, err := feeds.Items("", 100, 0)
	errPanic(err)

	itemsA, err := scoreHeadlines(headlinesToTest, string(prompt1))
	errPanic(err)
	itemsB, err := scoreHeadlines(headlinesToTest, string(prompt2))
	errPanic(err)
	fmt.Println(" A ; B ; Headline ; Reason A ; Reason B")
	for i := range itemsA {
		fmt.Printf("%3d;%3d;%s;%s;%s\n", itemsA[i].BreakingNewsScore, itemsB[i].BreakingNewsScore, itemsA[i].Title, itemsA[i].BreakingNewsReason, itemsB[i].BreakingNewsReason)
	}
}
