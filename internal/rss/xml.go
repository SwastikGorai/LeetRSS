package rss

import (
	"bytes"
	"encoding/xml"
	"time"
)

type rssXML struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr,omitempty"`
	Channel channelXML `xml:"channel"`
}

type channelXML struct {
	AtomLink    *atomLinkXML `xml:"atom:link,omitempty"`
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	Items       []itemXML    `xml:"item"`
}

type itemXML struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        guidXML `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Description string  `xml:"description,omitempty"`
}

type atomLinkXML struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type guidXML struct {
	IsPermaLink string `xml:"isPermaLink,attr,omitempty"`
	Value       string `xml:",chardata"`
}

func Render(feed Feed) ([]byte, error) {
	items := make([]itemXML, 0, len(feed.Items))
	for _, it := range feed.Items {
		items = append(items, itemXML{
			Title: it.Title,
			Link:  it.Link,
			GUID: guidXML{
				IsPermaLink: "false",
				Value:       it.GUID,
			},
			PubDate:     it.PubDate.UTC().Format(time.RFC1123Z),
			Description: it.Summary,
		})
	}

	var atomNS string
	var atomLink *atomLinkXML
	if feed.SelfLink != "" {
		atomNS = "http://www.w3.org/2005/Atom"
		atomLink = &atomLinkXML{
			Href: feed.SelfLink,
			Rel:  "self",
			Type: "application/rss+xml",
		}
	}

	out := rssXML{
		Version: "2.0",
		AtomNS:  atomNS,
		Channel: channelXML{
			AtomLink:    atomLink,
			Title:       feed.Title,
			Link:        feed.Link,
			Description: feed.Description,
			Items:       items,
		},
	}

	raw, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return bytes.Join([][]byte{[]byte(xml.Header), raw}, nil), nil
}
