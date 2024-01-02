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
