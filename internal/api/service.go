package api

import (
	"context"
	"fmt"
	"time"

	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/rss"
)

type UGCFeedService struct {
	Username string
	LC       *leetcode.Client
	First    int
}

func (s UGCFeedService) Build(ctx context.Context) ([]byte, error) {
	articles, err := leetcode.FetchUserSolutionArticles(ctx, s.LC, s.Username, s.First)
	if err != nil {
		return nil, err
	}

	items := make([]rss.Item, 0, len(articles))
	for _, a := range articles {
		t, err := time.Parse(time.RFC3339Nano, a.CreatedAt) //createdAt is like 2026-01-07T03:52:30.464981+00:00
		if err != nil {
			t = time.Now().UTC()
		}

		link := articleLink(a)
		guid := fmt.Sprintf("%d:%s", a.TopicID, a.UUID)

		items = append(items, rss.Item{
			Title:   a.Title,
			Link:    link,
			GUID:    guid,
			PubDate: t,
			Summary: fmt.Sprintf("Solution for %s (%s). Hits: %d", a.QuestionTitle, a.QuestionSlug, a.HitCount),
		})
	}

	feed := rss.Feed{
		Title:       fmt.Sprintf("LeetCode Solution Articles — %s", s.Username),
		Link:        fmt.Sprintf("https://leetcode.com/%s/", s.Username),
		Description: "Auto-generated RSS feed of your LeetCode Solution Articles (Discuss).",
		Items:       items,
	}
	return rss.Render(feed)
}

func articleLink(a leetcode.Article) string {
	if a.QuestionSlug != "" {
		return fmt.Sprintf("https://leetcode.com/problems/%s/solutions/%d/%s/", a.QuestionSlug, a.TopicID, a.Slug)
	}
	return fmt.Sprintf("https://leetcode.com/discuss/post/%d/%s/", a.TopicID, a.Slug)
}
