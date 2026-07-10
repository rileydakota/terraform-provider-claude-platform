// Package client is a minimal typed client for the Anthropic Admin API
// (https://api.anthropic.com/v1/organizations/*) and the WIF token exchange
// endpoint. No official Go SDK covers this surface.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	DefaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2023-06-01"
	maxAttempts    = 4
)

// Options configures a Client. At least one of AdminAPIKey or OAuthToken is
// required. When both are set, the OAuth token is used for every request.
type Options struct {
	BaseURL     string
	AdminAPIKey string
	OAuthToken  string
	UserAgent   string
	HTTPClient  *http.Client
}

type Client struct {
	baseURL     string
	adminAPIKey string
	oauthToken  string
	userAgent   string
	http        *http.Client
}

func New(opts Options) (*Client, error) {
	if opts.AdminAPIKey == "" && opts.OAuthToken == "" {
		return nil, errors.New("either an admin API key or an OAuth token must be configured")
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		baseURL:     baseURL,
		adminAPIKey: opts.AdminAPIKey,
		oauthToken:  opts.OAuthToken,
		userAgent:   opts.UserAgent,
		http:        httpClient,
	}, nil
}

// HasOAuth reports whether the client authenticates with an OAuth bearer
// token (required for the WIF endpoints).
func (c *Client) HasOAuth() bool { return c.oauthToken != "" }

// requireOAuth guards endpoints that reject admin API keys: service accounts,
// federation issuers, and federation rules.
func (c *Client) requireOAuth() error {
	if c.oauthToken == "" {
		return errors.New("this endpoint requires an OAuth token with the org:admin scope; " +
			"admin API keys are not accepted. Configure the provider's oauth_token or federation block")
	}
	return nil
}

// APIError is the standard Anthropic error envelope.
type APIError struct {
	StatusCode int
	ErrType    string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("claude platform api error (http %d, type %q, request_id %q): %s",
		e.StatusCode, e.ErrType, e.RequestID, e.Message)
}

// IsNotFound reports whether err is an APIError with HTTP status 404.
func IsNotFound(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == http.StatusNotFound
}

type errorEnvelope struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
	}

	for attempt := 1; ; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, reader)
		if err != nil {
			return err
		}
		req.Header.Set("anthropic-version", apiVersion)
		if c.oauthToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.oauthToken)
		} else {
			req.Header.Set("x-api-key", c.adminAPIKey)
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < maxAttempts {
				sleep(ctx, backoff(attempt, ""))
				continue
			}
			return err
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return readErr
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < maxAttempts {
				sleep(ctx, backoff(attempt, resp.Header.Get("retry-after")))
				continue
			}
		}

		if resp.StatusCode >= 400 {
			var env errorEnvelope
			apiErr := &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
			if json.Unmarshal(respBody, &env) == nil && env.Error.Message != "" {
				apiErr.ErrType = env.Error.Type
				apiErr.Message = env.Error.Message
				apiErr.RequestID = env.RequestID
			}
			return apiErr
		}

		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		}
		return nil
	}
}

func backoff(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 && secs <= 120 {
			return time.Duration(secs) * time.Second
		}
	}
	base := time.Duration(1<<uint(attempt-1)) * time.Second
	return base + time.Duration(rand.Int63n(int64(500*time.Millisecond)))
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// listPage is the pagination envelope used by the classic admin endpoints
// (users, invites, workspaces, workspace members, api keys).
type listPage[T any] struct {
	Data    []T    `json:"data"`
	HasMore bool   `json:"has_more"`
	FirstID string `json:"first_id"`
	LastID  string `json:"last_id"`
}

// cursorPage is the pagination envelope used by the WIF endpoints
// (service accounts, federation issuers, federation rules).
type cursorPage[T any] struct {
	Data     []T     `json:"data"`
	NextPage *string `json:"next_page"`
}

// listAll auto-paginates a classic (after_id) list endpoint.
func listAll[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("limit", "100")
	var all []T
	for {
		var page listPage[T]
		if err := c.do(ctx, http.MethodGet, path, query, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.HasMore || page.LastID == "" {
			return all, nil
		}
		query.Set("after_id", page.LastID)
	}
}

// listAllCursor auto-paginates a WIF (page/next_page) list endpoint.
func listAllCursor[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	if query == nil {
		query = url.Values{}
	}
	query.Set("limit", "100")
	var all []T
	for {
		var page cursorPage[T]
		if err := c.do(ctx, http.MethodGet, path, query, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if page.NextPage == nil || *page.NextPage == "" {
			return all, nil
		}
		query.Set("page", *page.NextPage)
	}
}
