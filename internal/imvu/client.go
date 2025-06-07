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

// Client represents an HTTP client for IMVU API
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	headers    map[string]string
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// NewClient creates a new IMVU API client with the given options
func NewClient(options ...ClientOption) (*Client, error) {
	// Create cookie jar for storing cookies
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Default client configuration
	client := &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		baseURL:   "https://api.imvu.com",
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

	// Apply options
	for _, option := range options {
		option(client)
	}

	return client, nil
}

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithUserAgent sets the User-Agent header for the client
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithHeader adds or updates a header for the client
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithTimeout sets the timeout for the client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// Request makes an HTTP request to the IMVU API
func (c *Client) Request(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	// Construct the full URL
	fullURL := c.baseURL + path

	// Create request body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create the request
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", c.userAgent)

	// Set default headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	// Set request-specific headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set Referer for browser-like behavior
	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", "https://pt.secure.imvu.com/")
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Get makes a GET request to the IMVU API
func (c *Client) Get(path string, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodGet, path, nil, headers)
}

// Post makes a POST request to the IMVU API
func (c *Client) Post(path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodPost, path, body, headers)
}

// Put makes a PUT request to the IMVU API
func (c *Client) Put(path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodPut, path, body, headers)
}

// Delete makes a DELETE request to the IMVU API
func (c *Client) Delete(path string, headers map[string]string) (*http.Response, error) {
	return c.Request(http.MethodDelete, path, nil, headers)
}

// GetCookies returns all cookies for a given URL
func (c *Client) GetCookies(urlStr string) ([]*http.Cookie, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	return c.httpClient.Jar.Cookies(parsedURL), nil
}

// SetCookies sets cookies for a given URL
func (c *Client) SetCookies(urlStr string, cookies []*http.Cookie) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	c.httpClient.Jar.SetCookies(parsedURL, cookies)
	return nil
}
