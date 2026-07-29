package ajp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type Client struct {
	base        string
	http        *http.Client
	ua          string
	minInterval time.Duration
	retries     int
	retryWait   time.Duration
	lastReq     time.Time
	mu          sync.Mutex
}

type ClientOption func(*Client)

func WithMinInterval(d time.Duration) ClientOption {
	return func(c *Client) { c.minInterval = d }
}

func WithRetry(n int, wait time.Duration) ClientOption {
	return func(c *Client) { c.retries = n; c.retryWait = wait }
}

func NewClient(baseURL string, opts ...ClientOption) *Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 25 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	c := &Client{
		base: stringsTrimRightSlash(baseURL),
		http: &http.Client{
			Timeout:   45 * time.Second,
			Transport: transport,
		},
		ua:          defaultUA,
		minInterval: 250 * time.Millisecond,
		retries:     3,
		retryWait:   600 * time.Millisecond,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) Base() string { return c.base }

func stringsTrimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func (c *Client) throttle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.minInterval <= 0 {
		return
	}
	elapsed := time.Since(c.lastReq)
	if !c.lastReq.IsZero() && elapsed < c.minInterval {
		wait := c.minInterval - elapsed
		c.mu.Unlock()
		time.Sleep(wait)
		c.mu.Lock()
	}
}

func (c *Client) GetBytes(ctx context.Context, path string) ([]byte, int, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		c.throttle()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", c.ua)
		req.Header.Set("Accept", "application/json, text/html;q=0.9,*/*;q=0.8")
		resp, err := c.http.Do(req)
		c.mu.Lock()
		c.lastReq = time.Now()
		c.mu.Unlock()
		if err != nil {
			lastErr = err
			time.Sleep(c.retryWait * time.Duration(1<<attempt))
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, resp.StatusCode, readErr
		}
		if resp.StatusCode >= 500 && attempt < c.retries {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(c.retryWait * time.Duration(1<<attempt))
			continue
		}
		return body, resp.StatusCode, nil
	}
	return nil, 0, lastErr
}

func (c *Client) GetJSON(ctx context.Context, path string, dest any) error {
	body, code, err := c.GetBytes(ctx, path)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("GET %s: status %d", path, code)
	}
	return json.Unmarshal(body, dest)
}
