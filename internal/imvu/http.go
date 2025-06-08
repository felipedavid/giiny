package imvu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"golang.org/x/net/publicsuffix"
)

const baseURL = "https://api.imvu.com"

type HTTPClient struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	headers    map[string]string
}

func (c *HTTPClient) AddHeader(key, value string) {
	c.headers[key] = value
}

type ClientOption func(*HTTPClient)

func NewClient(options ...ClientOption) (*HTTPClient, error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &HTTPClient{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		baseURL:   baseURL,
		userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36",
		headers: map[string]string{
			"Accept":             "application/json; charset=utf-8",
			"Accept-Language":    "en-US,en;q=0.9",
			"Content-Type":       "application/json; charset=UTF-8",
			"Sec-Ch-Ua":          "\"Not.A/Brand\";v=\"99\", \"Chromium\";v=\"136\"",
			"Sec-Ch-Ua-Mobile":   "?0",
			"Sec-Ch-Ua-Platform": "\"Linux\"",
			"Sec-Fetch-Dest":     "empty",
			"Sec-Fetch-Mode":     "cors",
			"Sec-Fetch-Site":     "same-site",
			"X-Imvu-Application": "welcome/1",
		},
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *HTTPClient) {
		c.baseURL = baseURL
	}
}

func WithUserAgent(userAgent string) ClientOption {
	return func(c *HTTPClient) {
		c.userAgent = userAgent
	}
}

func WithHeader(key, value string) ClientOption {
	return func(c *HTTPClient) {
		c.headers[key] = value
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

func (c *HTTPClient) Request(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", "https://pt.secure.imvu.com/")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (c *HTTPClient) Get(path string, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodGet, path, nil, headers)
}

func (c *HTTPClient) Post(path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodPost, path, body, headers)
}

func (c *HTTPClient) Put(path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodPut, path, body, headers)
}

func (c *HTTPClient) Delete(path string, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodDelete, path, nil, headers)
}

func (c *HTTPClient) GetCookies(urlStr string) ([]*http.Cookie, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	return c.httpClient.Jar.Cookies(parsedURL), nil
}

func (c *HTTPClient) SetCookies(urlStr string, cookies []*http.Cookie) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	c.httpClient.Jar.SetCookies(parsedURL, cookies)
	return nil
}
