package main

import "github.com/signalstoerung/reader/internal/feeds"

type HeadlinesPage struct {
	Headlines       []feeds.Item
	Page            int
	HasPreviousPage bool
	PreviousPage    int
	NextPage        int
	Filter          string
}

type HeadlineItem struct {
	Title              string
	FeedAbbr           string
	Timestamp          string
	Preview            string
	Link               string
	AlertClass         string
	BreakingNewsReason string
	Id                 int
}

func ConvertItems(in []feeds.Item) []HeadlineItem {
	var returnItems = make([]HeadlineItem, 0, len(in))
	for count, item := range in {
		var preview string
		if item.Description != "" {
			preview = item.Description
		} else {
			preview = item.Content
		}
		var alertClass string
		switch {
		case item.BreakingNewsScore > 90:
			alertClass = "alert"
		case item.BreakingNewsScore > 80:
			alertClass = "rush"
		case item.BreakingNewsScore > 70:
			alertClass = "highlight"
		default:
			alertClass = ""
		}
		returnItems = append(returnItems, HeadlineItem{
			Title:              item.Title,
			FeedAbbr:           item.FeedAbbr,
			Timestamp:          item.PublishedParsed.Format("02 Jan 15:04"),
			Preview:            preview,
			Link:               item.Link,
			AlertClass:         alertClass,
			BreakingNewsReason: item.BreakingNewsReason,
			Id:                 count,
		})
	}
	return returnItems
}
