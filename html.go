package main

type HeadlinesItem struct {
	Link      string
	Title     string
	Timestamp string
	FeedAbbr  string
}

type HeadlinesPage struct {
	Headlines       []HeadlinesItem
	Page            int
	HasPreviousPage bool
	PreviousPage    int
	NextPage        int
	Filter          string
}
