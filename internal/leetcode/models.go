package leetcode

type UGCArticlesResponse struct {
	Data struct {
		UgcArticleUserSolutionArticles struct {
			Edges []struct {
				Node Article
			} `json:"edges"`
		} `json:"ugcArticleUserSolutionArticles"`
	} `json:"data"`
}

type UGCArticlesEnvelope struct {
	Data struct {
		UgcArticleUserSolutionArticles struct {
			TotalNum int `json:"totalNum"`
			PageInfo struct {
				HasNextPage bool `json:"hasNextPage"`
			} `json:"pageInfo"`
			Edges []struct {
				Node Article `json:"node"`
			} `json:"edges"`
		} `json:"ugcArticleUserSolutionArticles"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type Article struct {
	TopicID       int    `json:"topicId"`
	UUID          string `json:"uuid"`
	Title         string `json:"title"`
	Slug          string `json:"slug"`
	CreatedAt     string `json:"createdAt"`
	HitCount      int    `json:"hitCount"`
	QuestionSlug  string `json:"questionSlug"`
	QuestionTitle string `json:"questionTitle"`
}
