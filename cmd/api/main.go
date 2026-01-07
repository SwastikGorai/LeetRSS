package main

import (
	"log"

	"leetcode-rss/internal/api"
	"leetcode-rss/internal/config"
	"leetcode-rss/internal/leetcode"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	lc := leetcode.New(cfg.LeetCode.GraphQLEndpoint, cfg.LeetCode.Cookie, cfg.LeetCode.CSRF)

	svc := api.UGCFeedService{
		Username: cfg.LeetCode.Username,
		LC:       lc,
		First:    15,
	}

	cache := api.NewCache(cfg.Cache.TTL)
	handlers := api.NewHandlers(svc, cache)

	log.Printf("listening on :%d (user=%s)", cfg.Server.Port, cfg.LeetCode.Username)

	srv := newServer(cfg.Server.Port, routes(handlers))
	log.Fatal(srv.ListenAndServe())
}
