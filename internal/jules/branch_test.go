package jules

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNoHardcodedMainForMasterDefaultRepo verifies R1:
// - A repository whose default branch is "master" must resolve to "master".
// - An unknown repository or missing default branch must resolve to "" (omitted).
// - Literal "main" must NEVER appear as a fallback.
func TestNoHardcodedMainForMasterDefaultRepo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sources" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"sources": [
					{
						"name": "sources/github/owner/master-repo",
						"githubRepo": {
							"owner": "owner",
							"repo": "master-repo",
							"defaultBranch": { "displayName": "master" }
						}
					},
					{
						"name": "sources/github/owner/initial-setup-repo",
						"githubRepo": {
							"owner": "owner",
							"repo": "initial-setup-repo",
							"defaultBranch": { "displayName": "initial-setup" }
						}
					},
					{
						"name": "sources/github/owner/no-branch-repo",
						"githubRepo": {
							"owner": "owner",
							"repo": "no-branch-repo"
						}
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := NewClient("test-key", WithBaseURL(ts.URL), WithDisableRetry(true))
	ctx := context.Background()

	// Case 1: master-default repo
	branch, err := client.ResolveStartingBranch(ctx, "sources/github/owner/master-repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected branch 'master', got %q", branch)
	}

	// Case 2: initial-setup repo
	branch, err = client.ResolveStartingBranch(ctx, "sources/github/owner/initial-setup-repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "initial-setup" {
		t.Errorf("expected branch 'initial-setup', got %q", branch)
	}

	// Case 3: repo without default branch (must return "", NEVER "main")
	branch, err = client.ResolveStartingBranch(ctx, "sources/github/owner/no-branch-repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch to allow omission, got %q (must not fallback to main)", branch)
	}

	// Case 4: unknown repo (must return "", NEVER "main")
	branch, err = client.ResolveStartingBranch(ctx, "sources/github/owner/unknown-repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch for unknown repo, got %q (must not fallback to main)", branch)
	}

	// Case 5: explicit branch overrides source default
	branch, err = client.ResolveStartingBranch(ctx, "sources/github/owner/master-repo", "feature-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "feature-x" {
		t.Errorf("expected explicit branch 'feature-x', got %q", branch)
	}

	// Case 6: explicit commit SHA overrides source default (Issue #11)
	branch, err = client.ResolveStartingBranch(ctx, "sources/github/owner/master-repo", "38cb99a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "38cb99a" {
		t.Errorf("expected explicit commit SHA '38cb99a', got %q", branch)
	}

	// Case 7: un-normalized source name resolves correctly
	branch, err = client.ResolveStartingBranch(ctx, "owner/master-repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected branch 'master' for 'owner/master-repo', got %q", branch)
	}
}

func TestNormalizeSourceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sources/github/owner/repo", "sources/github/owner/repo"},
		{"github/owner/repo", "sources/github/owner/repo"},
		{"owner/repo", "sources/github/owner/repo"},
		{"https://github.com/owner/repo", "sources/github/owner/repo"},
		{"http://github.com/owner/repo.git", "sources/github/owner/repo"},
		{"  owner/repo  ", "sources/github/owner/repo"},
		{"custom-source", "custom-source"},
	}

	for _, tt := range tests {
		got := NormalizeSourceName(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeSourceName(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}
