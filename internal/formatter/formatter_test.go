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

func TestFormatSessionStatus(t *testing.T) {
	createTime := time.Date(2023, 10, 20, 15, 0, 0, 0, time.UTC)
	updateTime := time.Date(2023, 10, 20, 15, 30, 0, 0, time.UTC)

	session := &jules.Session{
		ID:    "sess-123",
		State: "IN_PROGRESS",
		URL:   "https://jules.google.com/session/sess-123",
		Title: "Fixing bugs",
		SourceContext: &jules.SourceContext{
			Source: "sources/github/owner/repo",
			GithubRepoContext: &jules.GithubRepoContext{
				StartingBranch: "master",
			},
		},
		CreateTime: &createTime,
		UpdateTime: &updateTime,
		Prompt:     "Please fix all the bugs in the system. " + strings.Repeat("Very long prompt. ", 20), // Exceeds MaxPromptEchoLength
	}

	out := FormatSessionStatus(session)

	if !strings.Contains(out, "Session ID:** `sess-123`") {
		t.Errorf("expected session ID, got: %s", out)
	}
	if !strings.Contains(out, "State:** `IN_PROGRESS`") {
		t.Errorf("expected state, got: %s", out)
	}
	if !strings.Contains(out, "URL:** [https://jules.google.com/session/sess-123]") {
		t.Errorf("expected URL, got: %s", out)
	}
	if !strings.Contains(out, "Title:** Fixing bugs") {
		t.Errorf("expected title, got: %s", out)
	}
	if !strings.Contains(out, "Source:** `sources/github/owner/repo`") {
		t.Errorf("expected Source, got: %s", out)
	}
	if !strings.Contains(out, "Starting Branch / Ref:** `master`") {
		t.Errorf("expected Starting Branch / Ref, got: %s", out)
	}
	if !strings.Contains(out, "Created:** 2023-10-20T15:00:00Z") {
		t.Errorf("expected create time, got: %s", out)
	}
	if !strings.Contains(out, "Updated:** 2023-10-20T15:30:00Z") {
		t.Errorf("expected update time, got: %s", out)
	}
	if !strings.Contains(out, "Prompt:** Please fix all") {
		t.Errorf("expected prompt, got: %s", out)
	}
	if strings.Contains(out, strings.Repeat("Very long prompt. ", 20)) {
		t.Errorf("prompt was not truncated properly")
	}
	if len(out) > 650 {
		t.Errorf("prompt was not truncated properly, total length %d", len(out))
	}
}

func TestFormatSessionStatus_Minimal(t *testing.T) {
	session := &jules.Session{
		ID:    "sess-456",
		State: "QUEUED",
	}

	out := FormatSessionStatus(session)
	if !strings.Contains(out, "Session ID:** `sess-456`") {
		t.Errorf("expected session ID, got: %s", out)
	}
	if strings.Contains(out, "URL:**") {
		t.Errorf("did not expect URL, got: %s", out)
	}
	if strings.Contains(out, "Title:**") {
		t.Errorf("did not expect Title, got: %s", out)
	}
	if strings.Contains(out, "Created:**") {
		t.Errorf("did not expect Created time, got: %s", out)
	}
	if strings.Contains(out, "Updated:**") {
		t.Errorf("did not expect Updated time, got: %s", out)
	}
	if strings.Contains(out, "Prompt:**") {
		t.Errorf("did not expect Prompt, got: %s", out)
	}
}

func TestFormatDelegationResult_Minimal(t *testing.T) {
	session := &jules.Session{
		ID:    "123",
		State: "QUEUED",
	}

	out := FormatDelegationResult(session, "src", "", "prompt")
	if !strings.Contains(out, "Starting Branch:** *(repository default)*") {
		t.Errorf("expected fallback default branch text, got: %s", out)
	}
	if strings.Contains(out, "Web Console:**") {
		t.Errorf("did not expect web console link")
	}
	if strings.Contains(out, "Title:**") {
		t.Errorf("did not expect title")
	}
}

func TestFormatSources_Empty(t *testing.T) {
	out := FormatSources(nil)
	if out != "No connected GitHub sources found for this Jules account." {
		t.Errorf("expected empty message, got %q", out)
	}
}

func TestSummarizeActivity(t *testing.T) {
	cases := []struct {
		name       string
		act        *jules.Activity
		expectType string
		expectDesc string
	}{
		{
			name: "planGenerated with steps",
			act: &jules.Activity{
				PlanGenerated: &jules.PlanGenerated{
					Plan: &jules.Plan{
						Steps: []jules.PlanStep{{}, {}},
					},
				},
			},
			expectType: "planGenerated",
			expectDesc: "Proposed plan with 2 steps",
		},
		{
			name: "planGenerated without plan",
			act: &jules.Activity{
				PlanGenerated: &jules.PlanGenerated{},
			},
			expectType: "planGenerated",
			expectDesc: "Proposed plan with 0 steps",
		},
		{
			name: "planApproved",
			act: &jules.Activity{
				PlanApproved: &jules.PlanApproved{},
			},
			expectType: "planApproved",
			expectDesc: "Plan approved; execution initiated",
		},
		{
			name: "progressUpdated title",
			act: &jules.Activity{
				ProgressUpdated: &jules.ProgressUpdated{Title: "Updating"},
			},
			expectType: "progressUpdated",
			expectDesc: "Updating",
		},
		{
			name: "progressUpdated desc",
			act: &jules.Activity{
				ProgressUpdated: &jules.ProgressUpdated{Description: "Working..."},
			},
			expectType: "progressUpdated",
			expectDesc: "Working...",
		},
		{
			name: "progressUpdated empty",
			act: &jules.Activity{
				ProgressUpdated: &jules.ProgressUpdated{},
			},
			expectType: "progressUpdated",
			expectDesc: "Working on task...",
		},
		{
			name: "sessionCompleted",
			act: &jules.Activity{
				SessionCompleted: &jules.SessionCompleted{},
			},
			expectType: "sessionCompleted",
			expectDesc: "Mission completed successfully. Pull Request opened.",
		},
		{
			name: "sessionFailed with reason",
			act: &jules.Activity{
				SessionFailed: &jules.SessionFailed{Reason: "timeout"},
			},
			expectType: "sessionFailed",
			expectDesc: "Mission failed: timeout",
		},
		{
			name: "sessionFailed without reason",
			act: &jules.Activity{
				SessionFailed: &jules.SessionFailed{},
			},
			expectType: "sessionFailed",
			expectDesc: "Mission failed: Unknown failure",
		},
		{
			name: "agentMessaged message",
			act: &jules.Activity{
				AgentMessaged: &jules.AgentMessaged{Message: "hello"},
			},
			expectType: "agentMessaged",
			expectDesc: "hello",
		},
		{
			name: "agentMessaged prompt",
			act: &jules.Activity{
				AgentMessaged: &jules.AgentMessaged{Prompt: "prompt"},
			},
			expectType: "agentMessaged",
			expectDesc: "prompt",
		},
		{
			name: "userMessaged message",
			act: &jules.Activity{
				UserMessaged: &jules.UserMessaged{Message: "user"},
			},
			expectType: "userMessaged",
			expectDesc: "user",
		},
		{
			name: "userMessaged prompt",
			act: &jules.Activity{
				UserMessaged: &jules.UserMessaged{Prompt: "u-prompt"},
			},
			expectType: "userMessaged",
			expectDesc: "u-prompt",
		},
		{
			name:       "default",
			act:        &jules.Activity{},
			expectType: "event",
			expectDesc: "Activity recorded",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, desc := summarizeActivity(tc.act)
			if kind != tc.expectType {
				t.Errorf("expected type %s, got %s", tc.expectType, kind)
			}
			if desc != tc.expectDesc {
				t.Errorf("expected desc %s, got %s", tc.expectDesc, desc)
			}
		})
	}
}

func TestFormatActivities_Empty(t *testing.T) {
	out := FormatActivities("sess-1", nil)
	if out != "No activities recorded yet for session `sess-1`." {
		t.Errorf("expected empty message, got %q", out)
	}
}

func TestFormatActivities_SystemOriginator(t *testing.T) {
	act := jules.Activity{
		Name: "test",
		// Originator is empty, should default to "system"
	}
	out := FormatActivities("sess-1", []jules.Activity{act})
	if !strings.Contains(out, "(system):") {
		t.Errorf("expected system originator, got %q", out)
	}
}

func TestFormatActivities_BashOutput(t *testing.T) {
	act := jules.Activity{
		Artifacts: []jules.Artifact{
			{
				BashOutput: &jules.BashOutput{
					Command: "ls",
					Output:  "file.txt\n",
				},
			},
		},
	}
	out := FormatActivities("sess-1", []jules.Activity{act})
	if !strings.Contains(out, "Command*: `ls` (output: file.txt)") {
		t.Errorf("expected bash output formatting, got %q", out)
	}
}

func TestFormatActivities_WithNextPageToken(t *testing.T) {
	act := jules.Activity{
		Name:       "act-page",
		Originator: "agent",
		ProgressUpdated: &jules.ProgressUpdated{
			Title: "In progress",
		},
	}
	out := FormatActivities("sess-page", []jules.Activity{act}, "token-12345")
	if !strings.Contains(out, "**Next Page Token:** `token-12345`") {
		t.Errorf("expected next page token in output, got %q", out)
	}
}

func TestFormatPatch(t *testing.T) {
	patchWithBase := &jules.GitPatch{
		BaseCommitID: "abc123def",
		UnidiffPatch: "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new\n",
	}

	out := FormatPatch("sess-patch-1", patchWithBase)
	if !strings.Contains(out, "Git Patch for Session `sess-patch-1`") {
		t.Errorf("expected patch header, got %q", out)
	}
	if !strings.Contains(out, "**Base Commit:** `abc123def`") {
		t.Errorf("expected base commit, got %q", out)
	}
	if !strings.Contains(out, "```diff\ndiff --git") {
		t.Errorf("expected diff block, got %q", out)
	}

	patchNoBaseNoNewline := &jules.GitPatch{
		UnidiffPatch: "+line without newline",
	}
	out2 := FormatPatch("sess-patch-2", patchNoBaseNoNewline)
	if strings.Contains(out2, "**Base Commit:**") {
		t.Errorf("expected no base commit line when empty, got %q", out2)
	}
	if !strings.Contains(out2, "```diff\n+line without newline\n```") {
		t.Errorf("expected trailing newline before closing code block, got %q", out2)
	}
}
