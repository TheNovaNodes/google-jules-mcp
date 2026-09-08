package jules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL       = "https://jules.googleapis.com/v1alpha"
	DefaultTimeout       = 60 * time.Second
	DefaultRetryAttempts = 3
	DefaultMinRetryWait  = 1 * time.Second
	DefaultMaxRetryWait  = 10 * time.Second
)

// Client handles interaction with the Google Jules v1alpha REST API.
type Client struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	retryAttempts int
	minRetryWait  time.Duration
	maxRetryWait  time.Duration
	disableRetry  bool
	logger        *slog.Logger
}

// Option allows configuring the Jules client.
type Option func(*Client)

// WithBaseURL overrides the API endpoint URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

// WithHTTPClient provides a custom http.Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithRetryAttempts configures max retry attempts.
func WithRetryAttempts(attempts int) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithDisableRetry disables retries (R7: JULES_DISABLE_RETRY).
func WithDisableRetry(disable bool) Option {
	return func(c *Client) {
		c.disableRetry = disable
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new resilient JulesClient.
func NewClient(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("JULES_API_KEY")
	}

	disableRetry := os.Getenv("JULES_DISABLE_RETRY") == "1" || strings.ToLower(os.Getenv("JULES_DISABLE_RETRY")) == "true"

	c := &Client{
		baseURL:       DefaultBaseURL,
		apiKey:        apiKey,
		retryAttempts: DefaultRetryAttempts,
		minRetryWait:  DefaultMinRetryWait,
		maxRetryWait:  DefaultMaxRetryWait,
		disableRetry:  disableRetry,
		logger:        slog.Default(),
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// APIKey returns the configured API key.
func (c *Client) APIKey() string {
	return c.apiKey
}

// CleanSessionID normalizes session IDs (e.g. "sessions/123" -> "123").
func CleanSessionID(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	return strings.TrimPrefix(sessionID, "sessions/")
}

// isRetryable checks if an HTTP status code or error warrants a retry.
func isRetryable(statusCode int, err error) bool {
	if err != nil {
		return true // Connection reset, timeout, etc.
	}
	return statusCode == 429 || statusCode >= 500
}

// calculateBackoff computes exponential backoff with jitter.
func (c *Client) calculateBackoff(attempt int) time.Duration {
	mult := 1 << attempt
	wait := c.minRetryWait * time.Duration(mult)
	if wait > c.maxRetryWait {
		wait = c.maxRetryWait
	}
	// Add jitter +/- 20%
	jitter := time.Duration(rand.Float64()*0.4*float64(wait) - 0.2*float64(wait))
	res := wait + jitter
	if res < c.minRetryWait {
		res = c.minRetryWait
	}
	return res
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any) ([]byte, error) {
	if c.apiKey == "" {
		return nil, errors.New("JULES_API_KEY is not set")
	}

	reqURL := fmt.Sprintf("%s/%s", c.baseURL, strings.TrimPrefix(endpoint, "/"))

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	attempts := 1
	if !c.disableRetry && c.retryAttempts > 1 {
		attempts = c.retryAttempts
	}

	var lastErr error
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt < attempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var reader io.Reader
		if bodyBytes != nil {
			reader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Goog-Api-Key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.logger.WarnContext(ctx, "HTTP request failed", "attempt", attempt+1, "url", reqURL, "error", err)
			if attempt < attempts-1 {
				time.Sleep(c.calculateBackoff(attempt))
				continue
			}
			break
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if readErr != nil {
			lastErr = readErr
			if attempt < attempts-1 {
				time.Sleep(c.calculateBackoff(attempt))
				continue
			}
			break
		}

		lastStatus = resp.StatusCode
		lastBody = respBody

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Check if error is retryable
		if isRetryable(resp.StatusCode, nil) && attempt < attempts-1 {
			c.logger.WarnContext(ctx, "Transient error from Jules API, retrying",
				"status", resp.StatusCode, "attempt", attempt+1)
			time.Sleep(c.calculateBackoff(attempt))
			continue
		}

		return nil, &APIError{
			StatusCode: resp.StatusCode,
			RawBody:    string(respBody),
			Attempts:   attempt + 1,
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("network error after %d attempts: %w", attempts, lastErr)
	}

	return nil, &APIError{
		StatusCode: lastStatus,
		RawBody:    string(lastBody),
		Attempts:   attempts,
	}
}

// ListSources lists all connected GitHub repositories (GET /v1alpha/sources).
func (c *Client) ListSources(ctx context.Context) ([]Source, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/sources", nil)
	if err != nil {
		return nil, err
	}

	var resp ListSourcesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse sources response: %w", err)
	}

	return resp.Sources, nil
}

// CreateSession delegates a new task to Google Jules (POST /v1alpha/sessions).
func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	if req.Prompt == "" {
		return nil, errors.New("prompt cannot be empty")
	}
	if req.SourceContext == nil || req.SourceContext.Source == "" {
		return nil, errors.New("source cannot be empty")
	}

	data, err := c.doRequest(ctx, http.MethodPost, "/sessions", req)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	session.ID = CleanSessionID(session.Name)
	if session.URL == "" && session.ID != "" {
		session.URL = fmt.Sprintf("https://jules.google.com/session/%s", session.ID)
	}

	return &session, nil
}

// GetSession retrieves the status of an existing session (GET /v1alpha/sessions/{id}).
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	cleanID := CleanSessionID(sessionID)
	if cleanID == "" {
		return nil, errors.New("session_id cannot be empty")
	}

	data, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/sessions/%s", cleanID), nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session response: %w", err)
	}

	session.ID = cleanID
	if session.URL == "" && session.ID != "" {
		session.URL = fmt.Sprintf("https://jules.google.com/session/%s", session.ID)
	}

	return &session, nil
}

// ListActivities lists the execution activities for a session (GET /v1alpha/sessions/{id}/activities).
func (c *Client) ListActivities(ctx context.Context, sessionID string, pageSize int, pageToken string) (*ListActivitiesResponse, error) {
	cleanID := CleanSessionID(sessionID)
	if cleanID == "" {
		return nil, errors.New("session_id cannot be empty")
	}

	params := url.Values{}
	if pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}

	endpoint := fmt.Sprintf("/sessions/%s/activities", cleanID)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	data, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp ListActivitiesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse activities response: %w", err)
	}

	return &resp, nil
}

// SendMessage sends an interactive reply to a stalled or awaiting-feedback session
// (POST /v1alpha/sessions/{id}:sendMessage).
func (c *Client) SendMessage(ctx context.Context, sessionID, prompt string) error {
	cleanID := CleanSessionID(sessionID)
	if cleanID == "" {
		return errors.New("session_id cannot be empty")
	}
	if strings.TrimSpace(prompt) == "" {
		return errors.New("prompt cannot be empty")
	}

	req := SendMessageRequest{Prompt: prompt}
	_, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/sessions/%s:sendMessage", cleanID), req)
	return err
}

// ApprovePlan approves a plan generated by Jules when requirePlanApproval was set
// (POST /v1alpha/sessions/{id}:approvePlan).
func (c *Client) ApprovePlan(ctx context.Context, sessionID string) error {
	cleanID := CleanSessionID(sessionID)
	if cleanID == "" {
		return errors.New("session_id cannot be empty")
	}

	req := ApprovePlanRequest{}
	_, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/sessions/%s:approvePlan", cleanID), req)
	return err
}
