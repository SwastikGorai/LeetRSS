package main

import (
	"log"

	"leetcode-rss/internal/api"
	"leetcode-rss/internal/config"
	"leetcode-rss/internal/leetcode"
	"leetcode-rss/internal/store"
)

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
	log.Printf("listening on :%d (users=%v)", cfg.Server.Port, cfg.LeetCode.Usernames)

	srv := newServer(cfg.Server.Port, routes(handlers, publicHandlers, cfg.Server.HandlerTimeout))
	log.Fatal(srv.ListenAndServe())
}
