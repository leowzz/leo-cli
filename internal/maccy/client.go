package maccy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultLimit     = 50
	MaxLimit         = 200
	maxResponseBytes = 200 << 20
)

type Entry struct {
	ID              string    `json:"id"`
	ContentSHA256   string    `json:"content_sha256"`
	PlainText       string    `json:"plain_text"`
	TextBytes       int64     `json:"text_bytes"`
	FirstCopiedAt   time.Time `json:"first_copied_at"`
	LastCopiedAt    time.Time `json:"last_copied_at"`
	OccurrenceCount int64     `json:"occurrence_count"`
	Score           float32   `json:"score,omitempty"`
}

type SearchMode string

const (
	SearchContains SearchMode = "contains"
	SearchFuzzy    SearchMode = "fuzzy"
	SearchHybrid   SearchMode = "hybrid"
)

type SearchParams struct {
	Query  string
	Mode   SearchMode
	Limit  int
	Cursor string
}

type SearchResponse struct {
	Entries    []Entry `json:"entries"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

type Client struct {
	entriesURL *url.URL
	token      string
	httpClient *http.Client
}

func New(baseURL, token string) (*Client, error) {
	return NewWithHTTPClient(baseURL, token, nil)
}

func NewWithHTTPClient(baseURL, token string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, errors.New("clipboard.base_url is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("clipboard.base_url must be an http or https URL, got %q", baseURL)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return nil, fmt.Errorf("clipboard.base_url must not contain credentials, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1/entries"
	parsed.RawPath = ""

	if strings.TrimSpace(token) == "" {
		return nil, errors.New("clipboard.token is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{entriesURL: parsed, token: token, httpClient: httpClient}, nil
}

func (c *Client) Search(ctx context.Context, params SearchParams) (SearchResponse, error) {
	if c == nil || c.entriesURL == nil {
		return SearchResponse{}, errors.New("clipboard client is not configured")
	}
	if params.Limit == 0 {
		params.Limit = DefaultLimit
	}
	if params.Limit < 1 || params.Limit > MaxLimit {
		return SearchResponse{}, fmt.Errorf("limit must be between 1 and %d", MaxLimit)
	}
	if params.Mode == "" {
		params.Mode = SearchContains
	}
	if params.Mode != SearchContains && params.Mode != SearchFuzzy && params.Mode != SearchHybrid {
		return SearchResponse{}, fmt.Errorf("mode must be contains, fuzzy, or hybrid")
	}
	if utf8.RuneCountInString(params.Query) > 256 {
		return SearchResponse{}, errors.New("q must not exceed 256 characters")
	}
	if params.Mode == SearchFuzzy && utf8.RuneCountInString(params.Query) < 3 {
		return SearchResponse{}, errors.New("fuzzy search requires q to contain at least 3 characters")
	}
	if params.Mode == SearchHybrid && strings.TrimSpace(params.Query) == "" {
		return SearchResponse{}, errors.New("hybrid search requires a non-empty q")
	}

	requestURL := *c.entriesURL
	query := requestURL.Query()
	query.Set("limit", strconv.Itoa(params.Limit))
	if params.Query != "" {
		query.Set("q", params.Query)
	}
	if params.Mode != SearchContains {
		query.Set("mode", string(params.Mode))
	}
	if params.Cursor != "" {
		query.Set("cursor", params.Cursor)
	}
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("create clipboard search request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("clipboard search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&apiErr)
		if apiErr.Error != "" {
			return SearchResponse{}, fmt.Errorf("clipboard search failed (%s): %s", resp.Status, apiErr.Error)
		}
		return SearchResponse{}, fmt.Errorf("clipboard search failed (%s)", resp.Status)
	}

	var result SearchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return SearchResponse{}, fmt.Errorf("decode clipboard search response: %w", err)
	}
	return result, nil
}
