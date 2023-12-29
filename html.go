package main

type HeadlinesItem struct {
	Link               string
	Title              string
	Timestamp          string
	FeedAbbr           string
	Description        string
	Content            string
	BreakingNewsScore  int
	BreakingNewsReason string
}

type HeadlinesPage struct {
	Headlines       []HeadlinesItem
	Page            int
	HasPreviousPage bool
	PreviousPage    int
	NextPage        int
	Filter          string
}
