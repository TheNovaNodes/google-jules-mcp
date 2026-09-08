package jules

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
