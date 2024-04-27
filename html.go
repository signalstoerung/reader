package main

import (
	"fmt"

	"github.com/signalstoerung/reader/internal/feeds"
	"github.com/signalstoerung/reader/internal/users"
)

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
	Redacted           bool
	BreakingNewsReason string
	Id                 int
	ItemId             int
}

const (
	HTMLHeaderPath         = "www/header.html"
	HTMLFooterPath         = "www/footer.html"
	HTMLMainHeadlinesPath  = "www/main.html"
	HTMLFeedFormPath       = "www/feedform.html"
	HTMLFeedFormResultPath = "www/feedform-result.html"
	HTMLRegisterFormPath   = "www/register-form.html"
	HTMLLoginFormPath      = "www/login-form.html"
	HTMLKeywordFormPath    = "www/keywordform.html"
)

func ConvertItems(in []feeds.Item, keywordList users.KeywordList) []HeadlineItem {
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

		// keywords override alert classes
		mode, keyword := keywordList.Match(item.Title)
		if mode == users.HighlightMode {
			item.BreakingNewsReason = fmt.Sprintf("* Keyword '%v' triggered * %v", keyword, item.BreakingNewsReason)
			alertClass = "alert"
		}
		if mode == users.SuppressMode {
			item.BreakingNewsReason = fmt.Sprintf("* Keyword '%v' triggered * %v", keyword, item.BreakingNewsReason)
			alertClass = "redacted"
		}

		returnItems = append(returnItems, HeadlineItem{
			Title:              item.Title,
			FeedAbbr:           item.FeedAbbr,
			Timestamp:          item.PublishedParsed.In(globalConfig.localTZ).Format("02 Jan 15:04"),
			Preview:            preview,
			Link:               item.Link,
			AlertClass:         alertClass,
			BreakingNewsReason: item.BreakingNewsReason,
			Id:                 count,
			ItemId:             int(item.ID),
		})
	}
	return returnItems
}
