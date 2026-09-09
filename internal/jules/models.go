package jules

import "time"

// Source represents a connected GitHub repository in Google Jules.
type Source struct {
	Name       string      `json:"name"`                 // e.g. "sources/github/owner/repo"
	ID         string      `json:"id,omitempty"`         // e.g. "github/owner/repo"
	GithubRepo *GithubRepo `json:"githubRepo,omitempty"` // Repository details
}

// GithubRepo details.
type GithubRepo struct {
	Owner         string        `json:"owner,omitempty"`
	Repo          string        `json:"repo,omitempty"`
	DefaultBranch *BranchDetail `json:"defaultBranch,omitempty"`
}

// BranchDetail contains branch information.
type BranchDetail struct {
	DisplayName string `json:"displayName,omitempty"` // e.g. "main", "master", "develop"
}

// ListSourcesResponse is the response from GET /v1alpha/sources.
type ListSourcesResponse struct {
	Sources       []Source `json:"sources"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

// Session represents a Google Jules remote execution session.
type Session struct {
	Name          string         `json:"name"`         // e.g. "sessions/12345"
	ID            string         `json:"id,omitempty"` // extracted or raw id
	Title         string         `json:"title,omitempty"`
	Prompt        string         `json:"prompt,omitempty"`
	State         string         `json:"state"`         // QUEUED, IN_PROGRESS, AWAITING_USER_FEEDBACK, COMPLETED, FAILED
	URL           string         `json:"url,omitempty"` // Web UI link: https://jules.google.com/session/{id}
	CreateTime    *time.Time     `json:"createTime,omitempty"`
	UpdateTime    *time.Time     `json:"updateTime,omitempty"`
	SourceContext *SourceContext `json:"sourceContext,omitempty"`
}

// SourceContext defines the repository and branch context for a session.
type SourceContext struct {
	Source            string             `json:"source"`
	GithubRepoContext *GithubRepoContext `json:"githubRepoContext,omitempty"`
}

// GithubRepoContext defines GitHub-specific session parameters.
type GithubRepoContext struct {
	StartingBranch string `json:"startingBranch,omitempty"`
}

// CreateSessionRequest is the payload for POST /v1alpha/sessions.
type CreateSessionRequest struct {
	Prompt              string         `json:"prompt"`
	SourceContext       *SourceContext `json:"sourceContext"`
	Title               string         `json:"title,omitempty"`
	RequirePlanApproval *bool          `json:"requirePlanApproval,omitempty"`
}

// SendMessageRequest is the payload for POST /v1alpha/sessions/{id}:sendMessage.
type SendMessageRequest struct {
	Prompt string `json:"prompt"`
}

// ApprovePlanRequest is the payload for POST /v1alpha/sessions/{id}:approvePlan.
type ApprovePlanRequest struct{}

// Activity represents an event in the Jules session execution stream.
type Activity struct {
	Name             string            `json:"name,omitempty"`
	CreateTime       *time.Time        `json:"createTime,omitempty"`
	Originator       string            `json:"originator,omitempty"` // "agent", "user", "system"
	PlanGenerated    *PlanGenerated    `json:"planGenerated,omitempty"`
	PlanApproved     *PlanApproved     `json:"planApproved,omitempty"`
	ProgressUpdated  *ProgressUpdated  `json:"progressUpdated,omitempty"`
	SessionCompleted *SessionCompleted `json:"sessionCompleted,omitempty"`
	SessionFailed    *SessionFailed    `json:"sessionFailed,omitempty"`
	AgentMessaged    *AgentMessaged    `json:"agentMessaged,omitempty"`
	UserMessaged     *UserMessaged     `json:"userMessaged,omitempty"`
	Artifacts        []Artifact        `json:"artifacts,omitempty"`
}

// PlanGenerated contains the proposed plan steps.
type PlanGenerated struct {
	Plan *Plan `json:"plan,omitempty"`
}

// Plan details.
type Plan struct {
	Steps []PlanStep `json:"steps,omitempty"`
}

// PlanStep is an individual item in a plan.
type PlanStep struct {
	Index       *int   `json:"index,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// PlanApproved marks plan approval.
type PlanApproved struct {
	Plan *Plan `json:"plan,omitempty"`
}

// ProgressUpdated contains an update message.
type ProgressUpdated struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// SessionCompleted marks success.
type SessionCompleted struct{}

// SessionFailed marks failure.
type SessionFailed struct {
	Reason string `json:"reason,omitempty"`
}

// AgentMessaged represents a message from the agent.
type AgentMessaged struct {
	Message string `json:"message,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
}

// UserMessaged represents a message from the user.
type UserMessaged struct {
	Message string `json:"message,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
}

// Artifact represents an artifact produced by Jules (patches, bash output, etc.).
type Artifact struct {
	ChangeSet  *ChangeSet  `json:"changeSet,omitempty"`
	BashOutput *BashOutput `json:"bashOutput,omitempty"`
}

// ChangeSet contains a git patch.
type ChangeSet struct {
	Source   string    `json:"source,omitempty"`
	GitPatch *GitPatch `json:"gitPatch,omitempty"`
}

// GitPatch contains unidiff data.
type GitPatch struct {
	BaseCommitID string `json:"baseCommitId,omitempty"`
	UnidiffPatch string `json:"unidiffPatch,omitempty"`
}

// BashOutput contains command execution output.
type BashOutput struct {
	Command string `json:"command,omitempty"`
	Output  string `json:"output,omitempty"`
}

// ListActivitiesResponse is the response from GET /v1alpha/sessions/{id}/activities.
type ListActivitiesResponse struct {
	Activities    []Activity `json:"activities"`
	NextPageToken string     `json:"nextPageToken,omitempty"`
}
