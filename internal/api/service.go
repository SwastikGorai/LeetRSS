package api

import (
	"context"
	"fmt"
	"sort"
	"time"

	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/rss"
)

type UGCFeedService struct {
	Usernames []string
	LC        *leetcode.Client
	First     int
}

func (s UGCFeedService) Build(ctx context.Context, selfURL string) ([]byte, error) {
	allArticles := make([]leetcode.Article, 0)
	for _, username := range s.Usernames {
		articles, err := leetcode.FetchUserSolutionArticles(ctx, s.LC, username, s.First)
		if err != nil {
			return nil, fmt.Errorf("error fetching articles for user %s: %w", username, err)
		}
		allArticles = append(allArticles, articles...)
	}

	sort.Slice(allArticles, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339Nano, allArticles[i].CreatedAt)
		tj, _ := time.Parse(time.RFC3339Nano, allArticles[j].CreatedAt)
		return tj.Before(ti)
	})

	items := make([]rss.Item, 0, len(allArticles))
	for _, a := range allArticles {
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

	feedTitle := buildFeedTitle(s.Usernames)
	feedLink := buildFeedLink(s.Usernames)

	feed := rss.Feed{
		Title:       feedTitle,
		Link:        feedLink,
		SelfLink:    selfURL,
		Description: "Auto-generated RSS feed of LeetCode Solution Articles (Discuss).",
		Items:       items,
	}
	return rss.Render(feed)
}

func buildFeedTitle(usernames []string) string {
	if len(usernames) == 0 {
		return "LeetCode Solution Articles"
	}
	if len(usernames) == 1 {
		return fmt.Sprintf("LeetCode Solution Articles — %s", usernames[0])
	}
	return "LeetCode Solution Articles"
}

func buildFeedLink(usernames []string) string {
	if len(usernames) == 0 {
		return "https://leetcode.com/"
	}
	return fmt.Sprintf("https://leetcode.com/%s/", usernames[0])
}

func articleLink(a leetcode.Article) string {
	if a.QuestionSlug != "" {
		return fmt.Sprintf("https://leetcode.com/problems/%s/solutions/%d/%s/", a.QuestionSlug, a.TopicID, a.Slug)
	}
	return fmt.Sprintf("https://leetcode.com/discuss/post/%d/%s/", a.TopicID, a.Slug)
}
