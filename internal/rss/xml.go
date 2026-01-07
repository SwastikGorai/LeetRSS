package rss

import (
	"encoding/xml"
	"time"
)

type rssXML struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel channelXML `xml:"channel"`
}

type channelXML struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []itemXML `xml:"item"`
}

type itemXML struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

func Render(feed Feed) ([]byte, error) {
	items := make([]itemXML, 0, len(feed.Items))
	for _, it := range feed.Items {
		items = append(items, itemXML{
			Title:   it.Title,
			Link:    it.Link,
			GUID:    it.GUID,
			PubDate: it.PubDate.UTC().Format(time.RFC1123Z),
		})
	}

	out := rssXML{
		Version: "2.0",
		Channel: channelXML{
			Title:       feed.Title,
			Link:        feed.Link,
			Description: feed.Description,
			Items:       items,
		},
	}
	return xml.MarshalIndent(out, "", "  ")
}
