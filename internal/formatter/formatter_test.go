package formatter

import (
	"strings"
	"testing"
	"time"

	"github.com/TheNovaNodes/google-jules-mcp/internal/jules"
)

func TestTruncateString(t *testing.T) {
	longStr := strings.Repeat("a", 250)
	truncated := TruncateString(longStr, 200)
	if len(truncated) != 203 { // 200 + "..."
		t.Errorf("expected length 203, got %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Errorf("expected suffix '...', got %q", truncated)
	}

	shortStr := "hello"
	if got := TruncateString(shortStr, 200); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestFormatSources(t *testing.T) {
	sources := []jules.Source{
		{
			Name: "sources/github/owner/repo-main",
			GithubRepo: &jules.GithubRepo{
				DefaultBranch: &jules.BranchDetail{DisplayName: "main"},
			},
		},
		{
			Name: "sources/github/owner/repo-master",
			GithubRepo: &jules.GithubRepo{
				DefaultBranch: &jules.BranchDetail{DisplayName: "master"},
			},
		},
	}

	out := FormatSources(sources)
	if !strings.Contains(out, "Connected Jules Sources") {
		t.Errorf("expected table header, got: %s", out)
	}
	if !strings.Contains(out, "`sources/github/owner/repo-main` | `main`") {
		t.Errorf("expected row for repo-main, got: %s", out)
	}
	if !strings.Contains(out, "`sources/github/owner/repo-master` | `master`") {
		t.Errorf("expected row for repo-master, got: %s", out)
	}
}

func TestFormatDelegationResult(t *testing.T) {
	session := &jules.Session{
		ID:    "123456789",
		State: "QUEUED",
		URL:   "https://jules.google.com/session/123456789",
		Title: "Refactor architecture",
	}

	hugePrompt := strings.Repeat("Super complex prompt instruction. ", 20) // ~680 chars
	out := FormatDelegationResult(session, "sources/github/owner/repo", "master", hugePrompt)

	if !strings.Contains(out, "Session ID:** `123456789`") {
		t.Errorf("expected session ID, got: %s", out)
	}
	if !strings.Contains(out, "Starting Branch:** `master`") {
		t.Errorf("expected starting branch master, got: %s", out)
	}
	if !strings.Contains(out, "Web Console:** [https://jules.google.com/session/123456789]") {
		t.Errorf("expected web console link, got: %s", out)
	}

	// Verify R3: prompt echo must be truncated to <= 200 chars (+ "...")
	if strings.Contains(out, hugePrompt) {
		t.Errorf("huge prompt was not truncated! violates R3 response hygiene")
	}
}

func TestFormatActivities(t *testing.T) {
	hugePatch := strings.Repeat("+ func NewFeature() { ... }\n", 30) // ~870 chars
	now := time.Now()

	activities := []jules.Activity{
		{
			Name:       "act-1",
			Originator: "agent",
			CreateTime: &now,
			ProgressUpdated: &jules.ProgressUpdated{
				Title: "Creating new feature",
			},
			Artifacts: []jules.Artifact{
				{
					ChangeSet: &jules.ChangeSet{
						GitPatch: &jules.GitPatch{
							UnidiffPatch: hugePatch,
						},
					},
				},
			},
		},
	}

	out := FormatActivities("session-123", activities)
	if !strings.Contains(out, "Activities for Session `session-123`") {
		t.Errorf("missing header in activities output")
	}
	if !strings.Contains(out, "progressUpdated") {
		t.Errorf("missing activity kind")
	}
	// Verify R3: patch preview must be truncated to <= 300 chars
	if strings.Contains(out, hugePatch) {
		t.Errorf("huge patch was not truncated! violates R3")
	}
}

func TestFormatActivitiesCapped(t *testing.T) {
	now := time.Now()
	var manyActs []jules.Activity
	for i := 0; i < 50; i++ {
		manyActs = append(manyActs, jules.Activity{
			Name:       "act",
			Originator: "agent",
			CreateTime: &now,
			ProgressUpdated: &jules.ProgressUpdated{
				Title: strings.Repeat("Long detailed description of steps performed. ", 5),
			},
		})
	}

	out := FormatActivities("session-cap", manyActs)
	if len(out) > MaxOutputBytes+500 {
		t.Errorf("output exceeded max cap: %d bytes", len(out))
	}
	if !strings.Contains(out, "Output truncated") {
		t.Errorf("expected truncation notice when exceeding byte cap")
	}
}
