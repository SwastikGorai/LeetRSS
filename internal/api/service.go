package api

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/rss"

	"golang.org/x/sync/errgroup"
)

type UGCFeedService struct {
	Usernames []string
	LC        *leetcode.Client
	First     int
}

func (s UGCFeedService) Build(ctx context.Context, selfURL string) ([]byte, error) {
	first := s.First
	if first <= 0 {
		first = 15
	}
	if first > 50 {
		first = 50
	}
	allArticles := make([]leetcode.Article, 0)
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	maxConcurrentFetches := 4
	if len(s.Usernames) > 0 && len(s.Usernames) < maxConcurrentFetches {
		maxConcurrentFetches = len(s.Usernames)
	}
	sem := make(chan struct{}, maxConcurrentFetches)
	for _, username := range s.Usernames {
		username := username
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()
			articles, err := leetcode.FetchUserSolutionArticles(ctx, s.LC, username, first)
			if err != nil {
				return fmt.Errorf("error fetching articles for user %s: %w", username, err)
			}
			mu.Lock()
			allArticles = append(allArticles, articles...)
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	type timedArticle struct {
		Article   leetcode.Article
		CreatedAt time.Time
		OK        bool
	}
	timed := make([]timedArticle, 0, len(allArticles))
	for _, a := range allArticles {
		t, err := time.Parse(time.RFC3339Nano, a.CreatedAt) //createdAt is like 2026-01-07T03:52:30.464981+00:00
		if err != nil {
			timed = append(timed, timedArticle{Article: a})
			continue
		}
		timed = append(timed, timedArticle{Article: a, CreatedAt: t, OK: true})
	}
	sort.Slice(timed, func(i, j int) bool {
		ai, aj := timed[i], timed[j]
		if ai.OK != aj.OK {
			return ai.OK
		}
		if ai.OK && aj.OK {
			if ai.CreatedAt.Equal(aj.CreatedAt) {
				return ai.Article.TopicID > aj.Article.TopicID
			}
			return ai.CreatedAt.After(aj.CreatedAt)
		}
		return ai.Article.TopicID > aj.Article.TopicID
	})
	items := make([]rss.Item, 0, len(timed))
	for _, a := range timed {
		t := time.Unix(0, 0).UTC()
		if a.OK {
			t = a.CreatedAt
		}

		link := articleLink(a.Article)
		guid := fmt.Sprintf("%d:%s", a.Article.TopicID, a.Article.UUID)

		items = append(items, rss.Item{
			Title:   a.Article.Title,
			Link:    link,
			GUID:    guid,
			PubDate: t,
			Summary: fmt.Sprintf("Solution for %s (%s). Hits: %d", a.Article.QuestionTitle, a.Article.QuestionSlug, a.Article.HitCount),
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
