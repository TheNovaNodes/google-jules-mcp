package jules

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCleanSessionID(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"sessions/12345", "12345"},
		{"12345", "12345"},
		{" sessions/abc ", "abc"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := CleanSessionID(tc.input); got != tc.expected {
			t.Errorf("CleanSessionID(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestClientMissingAPIKey(t *testing.T) {
	client := NewClient("", WithDisableRetry(true))
	ctx := context.Background()

	_, err := client.ListSources(ctx)
	if err == nil || err.Error() != "JULES_API_KEY is not set" {
		t.Errorf("expected missing key error, got %v", err)
	}
}

func TestListSources(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/sources" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"sources": [
				{
					"name": "sources/github/owner/repo1",
					"githubRepo": { "owner": "owner", "repo": "repo1", "defaultBranch": { "displayName": "main" } }
				}
			]
		}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	sources, err := client.ListSources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Name != "sources/github/owner/repo1" {
		t.Errorf("expected source name 'sources/github/owner/repo1', got %q", sources[0].Name)
	}
}

func TestCreateSession(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var req CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Prompt != "Refactor tests" {
			t.Errorf("expected prompt 'Refactor tests', got %q", req.Prompt)
		}
		if req.SourceContext == nil || req.SourceContext.Source != "sources/github/o/r" {
			t.Errorf("unexpected sourceContext: %+v", req.SourceContext)
		}
		if req.Title != "Test Mission" {
			t.Errorf("expected title 'Test Mission', got %q", req.Title)
		}
		if req.RequirePlanApproval == nil || !*req.RequirePlanApproval {
			t.Errorf("expected requirePlanApproval=true")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "sessions/98765",
			"state": "QUEUED",
			"title": "Test Mission"
		}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	reqPlanAppr := true
	session, err := client.CreateSession(context.Background(), CreateSessionRequest{
		Prompt: "Refactor tests",
		Title:  "Test Mission",
		SourceContext: &SourceContext{
			Source: "sources/github/o/r",
			GithubRepoContext: &GithubRepoContext{
				StartingBranch: "master",
			},
		},
		RequirePlanApproval: &reqPlanAppr,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "98765" {
		t.Errorf("expected ID '98765', got %q", session.ID)
	}
	if session.URL != "https://jules.google.com/session/98765" {
		t.Errorf("expected URL 'https://jules.google.com/session/98765', got %q", session.URL)
	}
	if session.State != "QUEUED" {
		t.Errorf("expected state 'QUEUED', got %q", session.State)
	}
}

func TestGetSession(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/12345" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name": "sessions/12345",
			"state": "COMPLETED",
			"title": "Task Done"
		}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	session, err := client.GetSession(context.Background(), "sessions/12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.State != "COMPLETED" {
		t.Errorf("expected state 'COMPLETED', got %q", session.State)
	}
}

func TestSendMessage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/123:sendMessage" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var req SendMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Prompt != "Proceed with refactoring" {
			t.Errorf("unexpected prompt: %q", req.Prompt)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	err := client.SendMessage(context.Background(), "123", "Proceed with refactoring")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApprovePlan(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/123:approvePlan" || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	err := client.ApprovePlan(context.Background(), "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListActivities(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/123/activities" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("pageSize") != "5" {
			t.Errorf("expected pageSize=5, got %q", r.URL.Query().Get("pageSize"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"activities": [
				{
					"name": "act-1",
					"originator": "agent",
					"progressUpdated": { "title": "Cloned repo" }
				}
			]
		}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	resp, err := client.ListActivities(context.Background(), "123", 5, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(resp.Activities))
	}
}

func TestErrorTaxonomy(t *testing.T) {
	cases := []struct {
		status   int
		body     string
		expected string
	}{
		{404, `{}`, "Session or source not found — check session_id or source_name (use list_jules_sources)"},
		{401, `{}`, "JULES_API_KEY is invalid or lacks required permissions"},
		{403, `{}`, "JULES_API_KEY is invalid or lacks required permissions"},
		{400, `{"error":{"message":"Invalid repo name"}}`, "Invalid argument: Invalid repo name"},
		{429, `{}`, "Rate-limited by Google Jules API after 1 attempts; retry after backoff"},
		{503, `{}`, "Google Jules API unavailable (status 503); attempted 1 times"},
	}

	for _, tc := range cases {
		err := &APIError{
			StatusCode: tc.status,
			RawBody:    tc.body,
			Attempts:   1,
		}
		if err.Error() != tc.expected {
			t.Errorf("status %d error:\nexpected: %q\ngot:      %q", tc.status, tc.expected, err.Error())
		}
	}
}

func TestRetryBehavior(t *testing.T) {
	var attempts atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := attempts.Add(1)
		if current < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"sources": []}`))
	}))
	defer ts.Close()

	client := NewClient("secret-key",
		WithBaseURL(ts.URL),
		WithRetryAttempts(3),
	)
	// Override retry min wait for fast test execution
	client.minRetryWait = 10 * time.Millisecond
	client.maxRetryWait = 50 * time.Millisecond

	sources, err := client.ListSources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if len(sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(sources))
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestClientOptions(t *testing.T) {
	customClient := &http.Client{}
	logger := slog.Default()

	client := NewClient("key",
		WithBaseURL("https://custom.api/v1"),
		WithHTTPClient(customClient),
		WithRetryAttempts(5),
		WithLogger(logger),
	)

	if client.baseURL != "https://custom.api/v1" {
		t.Errorf("expected custom base URL")
	}
	if client.httpClient != customClient {
		t.Errorf("expected custom HTTP client")
	}
	if client.retryAttempts != 5 {
		t.Errorf("expected 5 retry attempts")
	}
	if client.logger != logger {
		t.Errorf("expected custom logger")
	}
	if client.APIKey() != "key" {
		t.Errorf("expected API key 'key'")
	}
}

func TestAPIErrorFormatting(t *testing.T) {
	cases := []struct {
		err      *APIError
		expected string
	}{
		{
			err:      &APIError{StatusCode: 400, RawBody: `{"error":{"message":"bad"}}`},
			expected: "Invalid argument: bad",
		},
		{
			err:      &APIError{StatusCode: 400, Message: "custom message"},
			expected: "Invalid argument: custom message",
		},
		{
			err:      &APIError{StatusCode: 400},
			expected: "Invalid argument provided to Jules API",
		},
		{
			err:      &APIError{StatusCode: 429, Attempts: 0},
			expected: "Rate-limited by Google Jules API; retry after backoff",
		},
		{
			err:      &APIError{StatusCode: 500, Attempts: 0},
			expected: "Google Jules API unavailable (status 500)",
		},
		{
			err:      &APIError{StatusCode: 418, Message: "I'm a teapot"},
			expected: "Jules API error (status 418): I'm a teapot",
		},
		{
			err:      &APIError{StatusCode: 418, RawBody: "I'm a teapot"},
			expected: "Jules API error (status 418): I'm a teapot",
		},
	}

	for _, tc := range cases {
		if tc.err.Error() != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, tc.err.Error())
		}
	}
}

func TestDoRequest_ContextCancel(t *testing.T) {
	client := NewClient("key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.ListSources(ctx)
	if err == nil || err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestDoRequest_InvalidURL(t *testing.T) {
	client := NewClient("key", WithBaseURL("http://[fe80::1%en0]/"))
	_, err := client.ListSources(context.Background())
	if err == nil {
		t.Errorf("expected error due to invalid URL, got nil")
	}
}

func TestCalculateBackoff_Limits(t *testing.T) {
	client := NewClient("key",
		WithRetryAttempts(5),
	)
	client.minRetryWait = 1 * time.Millisecond
	client.maxRetryWait = 5 * time.Millisecond

	b1 := client.calculateBackoff(10) // should hit max
	if b1 > 10*time.Millisecond || b1 < 1*time.Millisecond {
		t.Errorf("expected backoff near max, got %v", b1)
	}
}

func TestDoRequest_NetworkErrorRetries(t *testing.T) {
	// A server that closes connection immediately
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1")
	}))
	ts.Close() // Close immediately to simulate network error

	client := NewClient("key", WithBaseURL(ts.URL), WithRetryAttempts(2))
	client.minRetryWait = 1 * time.Millisecond
	client.maxRetryWait = 5 * time.Millisecond

	_, err := client.ListSources(context.Background())
	if err == nil {
		t.Errorf("expected network error")
	}
	if !strings.Contains(err.Error(), "network error after 2 attempts") {
		t.Errorf("expected retries in network error message, got: %v", err)
	}
}

func TestDoRequest_MarshalError(t *testing.T) {
	client := NewClient("key")
	// Channels cannot be marshaled
	_, err := client.doRequest(context.Background(), "POST", "/foo", make(chan int))
	if err == nil {
		t.Errorf("expected marshal error")
	}
}

func TestAPIEndpoints_HttpErrorsAndUnmarshal(t *testing.T) {
	// A server that returns unparseable JSON for GET, and 400 for POST
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{bad json`))
			if err != nil {
				t.Logf("err: %v", err)
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"error":{"message":"bad request"}}`))
			if err != nil {
				t.Logf("err: %v", err)
			}
		}
	}))
	defer ts.Close()

	client := NewClient("key", WithBaseURL(ts.URL), WithDisableRetry(true))
	ctx := context.Background()

	// ListSources unmarshal error
	_, err := client.ListSources(ctx)
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected unmarshal error for ListSources, got %v", err)
	}

	// CreateSession validation errors
	_, err = client.CreateSession(ctx, CreateSessionRequest{})
	if err == nil || !strings.Contains(err.Error(), "prompt cannot be empty") {
		t.Errorf("expected prompt error, got %v", err)
	}
	_, err = client.CreateSession(ctx, CreateSessionRequest{Prompt: "test"})
	if err == nil || !strings.Contains(err.Error(), "source cannot be empty") {
		t.Errorf("expected source error, got %v", err)
	}

	// CreateSession HTTP error
	_, err = client.CreateSession(ctx, CreateSessionRequest{
		Prompt:        "test",
		SourceContext: &SourceContext{Source: "test"},
	})
	if err == nil || !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected HTTP error for CreateSession, got %v", err)
	}

	// GetSession validation error
	_, err = client.GetSession(ctx, "")
	if err == nil || !strings.Contains(err.Error(), "session_id cannot be empty") {
		t.Errorf("expected session_id empty error, got %v", err)
	}

	// GetSession unmarshal error
	_, err = client.GetSession(ctx, "sess-1")
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected unmarshal error for GetSession, got %v", err)
	}

	// ListActivities validation error
	_, err = client.ListActivities(ctx, "", 10, "")
	if err == nil || !strings.Contains(err.Error(), "session_id cannot be empty") {
		t.Errorf("expected session_id empty error, got %v", err)
	}

	// ListActivities unmarshal error
	_, err = client.ListActivities(ctx, "sess-1", 10, "")
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("expected unmarshal error for ListActivities, got %v", err)
	}

	// SendMessage validation errors
	err = client.SendMessage(ctx, "", "test")
	if err == nil || !strings.Contains(err.Error(), "session_id cannot be empty") {
		t.Errorf("expected session_id empty error, got %v", err)
	}
	err = client.SendMessage(ctx, "sess-1", "")
	if err == nil || !strings.Contains(err.Error(), "prompt cannot be empty") {
		t.Errorf("expected prompt empty error, got %v", err)
	}

	// ApprovePlan validation error
	err = client.ApprovePlan(ctx, "")
	if err == nil || !strings.Contains(err.Error(), "session_id cannot be empty") {
		t.Errorf("expected session_id empty error, got %v", err)
	}
}

func TestListActivities_Params(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageSize") != "10" {
			t.Errorf("expected pageSize 10, got %v", r.URL.Query().Get("pageSize"))
		}
		if r.URL.Query().Get("pageToken") != "token123" {
			t.Errorf("expected pageToken token123, got %v", r.URL.Query().Get("pageToken"))
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{}`))
		if err != nil {
			t.Logf("err: %v", err)
		}
	}))
	defer ts.Close()

	client := NewClient("key", WithBaseURL(ts.URL), WithDisableRetry(true))
	_, _ = client.ListActivities(context.Background(), "sess-1", 10, "token123")
}

func TestCreateSession_URLGeneration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"name":"sessions/123"}`))
		if err != nil {
			t.Logf("err: %v", err)
		}
	}))
	defer ts.Close()

	client := NewClient("key", WithBaseURL(ts.URL), WithDisableRetry(true))
	s, err := client.CreateSession(context.Background(), CreateSessionRequest{Prompt: "test", SourceContext: &SourceContext{Source: "test"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.URL != "https://jules.google.com/session/123" {
		t.Errorf("expected URL generation, got %v", s.URL)
	}
}

func TestGetSession_URLGeneration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"name":"sessions/123"}`))
		if err != nil {
			t.Logf("err: %v", err)
		}
	}))
	defer ts.Close()

	client := NewClient("key", WithBaseURL(ts.URL), WithDisableRetry(true))
	s, err := client.GetSession(context.Background(), "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.URL != "https://jules.google.com/session/123" {
		t.Errorf("expected URL generation, got %v", s.URL)
	}
}

func TestDoRequest_ReadBodyError(t *testing.T) {
	// A server that returns a Content-Length larger than the body
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, err := w.Write([]byte(`{}`))
		if err != nil {
			t.Logf("err: %v", err)
		}
	}))
	defer ts.Close()

	client := NewClient("key", WithBaseURL(ts.URL), WithDisableRetry(true))
	_, err := client.ListSources(context.Background())
	if err == nil {
		t.Errorf("expected read error, got nil")
	}
}
