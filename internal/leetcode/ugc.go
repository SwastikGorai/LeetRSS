package leetcode

import (
	"context"
	"fmt"
)

type ugcReq struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName,omitempty"`
}

const queryUGCUserSolutions = `
query ugcArticleUserSolutionArticles(
  $username: String!,
  $orderBy: ArticleOrderByEnum,
  $skip: Int,
  $before: String,
  $after: String,
  $first: Int,
  $last: Int
) {
  ugcArticleUserSolutionArticles(
    username: $username
    orderBy: $orderBy
    skip: $skip
    before: $before
    after: $after
    first: $first
    last: $last
  ) {
    totalNum
    pageInfo { hasNextPage }
    edges {
      node {
        topicId
        uuid
        title
        slug
        createdAt
        hitCount
        questionSlug
        questionTitle
        reactions { count reactionType }
      }
    }
  }
}
`

func FetchUserSolutionArticles(ctx context.Context, c *Client, username string, first int) ([]Article, error) {
	req := ugcReq{
		Query:         queryUGCUserSolutions,
		OperationName: "ugcArticleUserSolutionArticles",
		Variables: map[string]interface{}{
			"username": username,
			"orderBy":  "MOST_RECENT",
			"skip":     0,
			"first":    first,
		},
	}

	var env UGCArticlesEnvelope
	if err := c.PostJSON(ctx, req, &env); err != nil {
		return nil, err
	}
	if len(env.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", env.Errors[0].Message)
	}

	edges := env.Data.UgcArticleUserSolutionArticles.Edges
	out := make([]Article, 0, len(edges))
	for _, e := range edges {
		out = append(out, e.Node)
	}
	return out, nil
}
