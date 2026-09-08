package formatter

import (
	"fmt"
	"strings"
	"time"

	"github.com/TheNovaNodes/google-jules-mcp/internal/jules"
)

const (
	MaxPromptEchoLength = 200
	MaxPatchLength      = 300
	MaxOutputBytes      = 4096
)

// TruncateString cuts a string to maxLen and adds ellipsis if exceeded.
func TruncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// FormatSources formats the list of sources into a clean markdown table.
func FormatSources(sources []jules.Source) string {
	if len(sources) == 0 {
		return "No connected GitHub sources found for this Jules account."
	}

	var sb strings.Builder
	sb.WriteString("### Connected Jules Sources\n\n")
	sb.WriteString("| Source Name | Default Branch |\n")
	sb.WriteString("|-------------|----------------|\n")

	for _, s := range sources {
		branch := "*(not specified)*"
		if s.GithubRepo != nil && s.GithubRepo.DefaultBranch != nil && s.GithubRepo.DefaultBranch.DisplayName != "" {
			branch = fmt.Sprintf("`%s`", s.GithubRepo.DefaultBranch.DisplayName)
		}
		sb.WriteString(fmt.Sprintf("| `%s` | %s |\n", s.Name, branch))
	}

	return sb.String()
}

// FormatDelegationResult formats the result of delegating a task to Jules (R3).
func FormatDelegationResult(session *jules.Session, sourceName, branch string, prompt string) string {
	var sb strings.Builder
	sb.WriteString("### 🚀 Task Delegated to Google Jules\n\n")
	sb.WriteString(fmt.Sprintf("- **Session ID:** `%s`\n", session.ID))
	sb.WriteString(fmt.Sprintf("- **Status:** `%s`\n", session.State))
	if session.URL != "" {
		sb.WriteString(fmt.Sprintf("- **Web Console:** [%s](%s)\n", session.URL, session.URL))
	}
	sb.WriteString(fmt.Sprintf("- **Source:** `%s`\n", sourceName))
	if branch != "" {
		sb.WriteString(fmt.Sprintf("- **Starting Branch:** `%s`\n", branch))
	} else {
		sb.WriteString("- **Starting Branch:** *(repository default)*\n")
	}
	if session.Title != "" {
		sb.WriteString(fmt.Sprintf("- **Title:** %s\n", session.Title))
	}
	sb.WriteString(fmt.Sprintf("- **Prompt Summary:** %s\n", TruncateString(prompt, MaxPromptEchoLength)))
	sb.WriteString("\n*Use `check_jules_status` or `list_jules_activities` to monitor execution.*")

	return sb.String()
}

// FormatSessionStatus formats session status concisely.
func FormatSessionStatus(session *jules.Session) string {
	var sb strings.Builder
	sb.WriteString("### 📋 Jules Session Status\n\n")
	sb.WriteString(fmt.Sprintf("- **Session ID:** `%s`\n", session.ID))
	sb.WriteString(fmt.Sprintf("- **State:** `%s`\n", session.State))
	if session.URL != "" {
		sb.WriteString(fmt.Sprintf("- **URL:** [%s](%s)\n", session.URL, session.URL))
	}
	if session.Title != "" {
		sb.WriteString(fmt.Sprintf("- **Title:** %s\n", session.Title))
	}
	if session.CreateTime != nil {
		sb.WriteString(fmt.Sprintf("- **Created:** %s\n", session.CreateTime.UTC().Format(time.RFC3339)))
	}
	if session.UpdateTime != nil {
		sb.WriteString(fmt.Sprintf("- **Updated:** %s\n", session.UpdateTime.UTC().Format(time.RFC3339)))
	}
	if session.Prompt != "" {
		sb.WriteString(fmt.Sprintf("- **Prompt:** %s\n", TruncateString(session.Prompt, MaxPromptEchoLength)))
	}

	return sb.String()
}

// FormatActivities formats activity events with patch truncation and total byte limit (R3).
func FormatActivities(sessionID string, activities []jules.Activity) string {
	if len(activities) == 0 {
		return fmt.Sprintf("No activities recorded yet for session `%s`.", sessionID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 📜 Activities for Session `%s` (%d events)\n\n", sessionID, len(activities)))

	for i, act := range activities {
		var entry strings.Builder
		ts := ""
		if act.CreateTime != nil {
			ts = act.CreateTime.UTC().Format("15:04:05")
		}

		originator := act.Originator
		if originator == "" {
			originator = "system"
		}

		eventType, summary := summarizeActivity(&act)
		entry.WriteString(fmt.Sprintf("`[%s]` **%s** (%s): %s\n", ts, eventType, originator, summary))

		// Process artifacts
		for _, art := range act.Artifacts {
			if art.ChangeSet != nil && art.ChangeSet.GitPatch != nil {
				patch := art.ChangeSet.GitPatch.UnidiffPatch
				if patch != "" {
					entry.WriteString(fmt.Sprintf("  - *Patch preview*: `%s`\n", TruncateString(strings.ReplaceAll(patch, "\n", " "), MaxPatchLength)))
				}
			}
			if art.BashOutput != nil {
				entry.WriteString(fmt.Sprintf("  - *Command*: `%s` (output: %s)\n",
					TruncateString(art.BashOutput.Command, 100),
					TruncateString(strings.ReplaceAll(art.BashOutput.Output, "\n", " "), 100)))
			}
		}

		// Check if adding this entry exceeds output cap
		if sb.Len()+entry.Len() > MaxOutputBytes {
			sb.WriteString(fmt.Sprintf("\n*(Output truncated: %d additional activities omitted to conserve context)*\n", len(activities)-i))
			break
		}

		sb.WriteString(entry.String())
	}

	return sb.String()
}

func summarizeActivity(act *jules.Activity) (string, string) {
	switch {
	case act.PlanGenerated != nil:
		stepCount := 0
		if act.PlanGenerated.Plan != nil {
			stepCount = len(act.PlanGenerated.Plan.Steps)
		}
		return "planGenerated", fmt.Sprintf("Proposed plan with %d steps", stepCount)
	case act.PlanApproved != nil:
		return "planApproved", "Plan approved; execution initiated"
	case act.ProgressUpdated != nil:
		desc := act.ProgressUpdated.Title
		if desc == "" {
			desc = act.ProgressUpdated.Description
		}
		if desc == "" {
			desc = "Working on task..."
		}
		return "progressUpdated", TruncateString(desc, 150)
	case act.SessionCompleted != nil:
		return "sessionCompleted", "Mission completed successfully. Pull Request opened."
	case act.SessionFailed != nil:
		reason := act.SessionFailed.Reason
		if reason == "" {
			reason = "Unknown failure"
		}
		return "sessionFailed", fmt.Sprintf("Mission failed: %s", reason)
	case act.AgentMessaged != nil:
		msg := act.AgentMessaged.Message
		if msg == "" {
			msg = act.AgentMessaged.Prompt
		}
		return "agentMessaged", TruncateString(msg, 150)
	case act.UserMessaged != nil:
		msg := act.UserMessaged.Message
		if msg == "" {
			msg = act.UserMessaged.Prompt
		}
		return "userMessaged", TruncateString(msg, 150)
	default:
		return "event", "Activity recorded"
	}
}
