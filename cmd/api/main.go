package main

import (
	"log"

	"leetcode-rss/internal/api"
	"leetcode-rss/internal/config"
	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/store"

	"github.com/clerk/clerk-sdk-go/v2"
)

type app struct {
	config         *config.Config
	store          store.Store
	leetcodeClient *leetcode.Client
	handlers       *api.Handlers
	publicHandlers *api.PublicFeedHandlers
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	lc := leetcode.New(cfg.LeetCode.GraphQLEndpoint, cfg.LeetCode.Cookie, cfg.LeetCode.CSRF)

	svc := api.UGCFeedService{
		Usernames: cfg.LeetCode.Usernames,
		LC:        lc,
		First:     cfg.LeetCode.MaxArticlesPerUser,
	}

	cache := api.NewCache(cfg.Cache.TTL)
	handlers := api.NewHandlers(svc, cache)

	var publicHandlers *api.PublicFeedHandlers
	s, err := store.NewStore(cfg.Database.URL)
	if err != nil {
		log.Printf("warning: failed to initialize database, public feeds disabled: %v", err)
	} else {
		defer s.Close()
		publicHandlers = api.NewPublicFeedHandlers(s, lc, cfg.Database.RSSCacheTTL)
		log.Printf("database initialized, public feeds enabled")
	}

	if cfg.Clerk.SecretKey != "" {
		clerk.SetKey(cfg.Clerk.SecretKey)
		log.Printf("clerk authentication enabled")
	}

	app := &app{
		config:         cfg,
		store:          s,
		leetcodeClient: lc,
		handlers:       handlers,
		publicHandlers: publicHandlers,
	}

	log.Printf("listening on :%d (users=%v)", cfg.Server.Port, cfg.LeetCode.Usernames)

	log.Fatal(app.serve())
}
