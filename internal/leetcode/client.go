package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Endpoint string
	Cookie   string
	CSRF     string
	Client   *http.Client
}

func New(endpoint, cookie, csrf string) *Client {
	return &Client{
		Endpoint: endpoint,
		Cookie:   cookie,
		CSRF:     csrf,
		Client:   http.DefaultClient,
	}
}

func (c *Client) Do(ctx context.Context, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.Endpoint,
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://leetcode.com/")
	req.Header.Set("User-Agent", "leetcode-rss/1.0")

	if c.Cookie != "" {
		req.Header.Set("Cookie", c.Cookie)
	}
	if c.CSRF != "" {
		req.Header.Set("x-csrftoken", c.CSRF)
		req.Header.Set("x-requested-with", "XMLHttpRequest")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("leetcode http %d: %s", resp.StatusCode, raw)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) PostJSON(ctx context.Context, body any, out any) error {
	return c.Do(ctx, body, out)
}
