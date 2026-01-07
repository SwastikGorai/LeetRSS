package rss

import "time"

type Feed struct {
	Title       string
	Link        string
	Description string
	Items       []Item
}

type Item struct {
	Title   string
	Link    string
	GUID    string
	PubDate time.Time
	Summary string
}
