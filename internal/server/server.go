package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/TheNovaNodes/google-jules-mcp/internal/formatter"
	"github.com/TheNovaNodes/google-jules-mcp/internal/jules"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Server coordinates the MCP tool registry and delegates to JulesClient.
type Server struct {
	mcpServer   *mcpserver.MCPServer
	julesClient *jules.Client
	logger      *slog.Logger
}

// NewServer creates and registers all MCP tools for Google Jules.
func NewServer(client *jules.Client, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	mcpSrv := mcpserver.NewMCPServer("google-jules-mcp", "1.0.0")

	s := &Server{
		mcpServer:   mcpSrv,
		julesClient: client,
		logger:      logger,
	}

	s.registerTools()
	return s
}

// MCPServer returns the underlying mcpserver.MCPServer instance.
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcpServer
}

func (s *Server) registerTools() {
	// 1. list_jules_sources
	s.mcpServer.AddTool(
		mcp.NewTool("list_jules_sources",
			mcp.WithDescription("List all GitHub repositories and default branches connected to the user's Jules account. Use this to find the correct source_name."),
		),
		s.handleListJulesSources,
	)

	// 2. delegate_task_to_jules
	s.mcpServer.AddTool(
		mcp.NewTool("delegate_task_to_jules",
			mcp.WithDescription("Delegate a long-running, asynchronous task to Google Jules. Jules will clone the repository in a cloud VM and propose a Pull Request."),
			mcp.WithString("source_name", mcp.Required(), mcp.Description("GitHub source name connected to Jules (e.g. 'sources/github/owner/repo')")),
			mcp.WithString("prompt", mcp.Required(), mcp.Description("Detailed instructions for what Jules should do in the repository")),
			mcp.WithString("starting_branch", mcp.Description("Git branch to start from. If omitted, auto-resolves to source default branch.")),
			mcp.WithString("title", mcp.Description("Optional descriptive title for the session")),
			mcp.WithBoolean("require_plan_approval", mcp.Description("If true, Jules pauses for plan approval before execution (default: false)")),
		),
		s.handleDelegateTaskToJules,
	)

	// 3. check_jules_status
	s.mcpServer.AddTool(
		mcp.NewTool("check_jules_status",
			mcp.WithDescription("Get the current execution state and status of a remote Jules session."),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("The ID of the session returned by delegate_task_to_jules")),
		),
		s.handleCheckJulesStatus,
	)

	// 4. get_jules_session
	s.mcpServer.AddTool(
		mcp.NewTool("get_jules_session",
			mcp.WithDescription("Retrieve comprehensive details and metadata of a Jules session (state, timestamps, URL, prompt)."),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("The ID of the session to inspect")),
		),
		s.handleGetJulesSession,
	)

	// 5. list_jules_activities
	s.mcpServer.AddTool(
		mcp.NewTool("list_jules_activities",
			mcp.WithDescription("List execution activity events, progress steps, and code patch previews for a Jules session."),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("The ID of the session")),
			mcp.WithNumber("page_size", mcp.Description("Maximum number of activities to return (default: 10)")),
			mcp.WithString("page_token", mcp.Description("Optional pagination token for next page")),
		),
		s.handleListJulesActivities,
	)

	// 6. send_jules_message
	s.mcpServer.AddTool(
		mcp.NewTool("send_jules_message",
			mcp.WithDescription("Send an interactive message or instructions to a Jules session (e.g. when session is in AWAITING_USER_FEEDBACK)."),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("The ID of the session")),
			mcp.WithString("prompt", mcp.Required(), mcp.Description("Message or guidance to send to Jules")),
		),
		s.handleSendJulesMessage,
	)

	// 7. approve_jules_plan
	s.mcpServer.AddTool(
		mcp.NewTool("approve_jules_plan",
			mcp.WithDescription("Approve a proposed execution plan for a Jules session paused in AWAITING_PLAN_APPROVAL. WARNING: Releases human gate for this delegated mission."),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("The ID of the session")),
		),
		s.handleApproveJulesPlan,
	)
}

func (s *Server) handleListJulesSources(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.julesClient.APIKey() == "" {
		return mcp.NewToolResultError("Error: JULES_API_KEY environment variable is not set for the MCP server."), nil
	}

	sources, err := s.julesClient.ListSources(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "list_jules_sources failed", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error listing sources: %v", err)), nil
	}

	formatted := formatter.FormatSources(sources)
	return mcp.NewToolResultText(formatted), nil
}

func (s *Server) handleDelegateTaskToJules(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.julesClient.APIKey() == "" {
		return mcp.NewToolResultError("Error: JULES_API_KEY environment variable is not set for the MCP server."), nil
	}

	sourceName := strings.TrimSpace(req.GetString("source_name", ""))
	prompt := strings.TrimSpace(req.GetString("prompt", ""))
	startingBranch := strings.TrimSpace(req.GetString("starting_branch", ""))
	title := strings.TrimSpace(req.GetString("title", ""))
	requirePlanApproval := req.GetBool("require_plan_approval", false)

	if sourceName == "" || prompt == "" {
		return mcp.NewToolResultError("Error: Missing source_name or prompt."), nil
	}

	// R1: Resolve starting branch
	resolvedBranch, err := s.julesClient.ResolveStartingBranch(ctx, sourceName, startingBranch)
	if err != nil {
		s.logger.WarnContext(ctx, "Branch resolution error, falling back to omit", "error", err)
	}

	sessionReq := jules.CreateSessionRequest{
		Prompt: prompt,
		Title:  title,
		SourceContext: &jules.SourceContext{
			Source: sourceName,
		},
	}

	if resolvedBranch != "" {
		sessionReq.SourceContext.GithubRepoContext = &jules.GithubRepoContext{
			StartingBranch: resolvedBranch,
		}
	}

	if requirePlanApproval {
		sessionReq.RequirePlanApproval = &requirePlanApproval
	}

	session, err := s.julesClient.CreateSession(ctx, sessionReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "delegate_task_to_jules failed", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error delegating task: %v", err)), nil
	}

	formatted := formatter.FormatDelegationResult(session, sourceName, resolvedBranch, prompt)
	return mcp.NewToolResultText(formatted), nil
}

func (s *Server) handleCheckJulesStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.julesClient.APIKey() == "" {
		return mcp.NewToolResultError("Error: JULES_API_KEY environment variable is not set for the MCP server."), nil
	}

	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	if sessionID == "" {
		return mcp.NewToolResultError("Error: Missing session_id."), nil
	}

	session, err := s.julesClient.GetSession(ctx, sessionID)
	if err != nil {
		s.logger.ErrorContext(ctx, "check_jules_status failed", "session_id", sessionID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error checking session status: %v", err)), nil
	}

	formatted := formatter.FormatSessionStatus(session)
	return mcp.NewToolResultText(formatted), nil
}

func (s *Server) handleGetJulesSession(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Rich alias for session inspection
	return s.handleCheckJulesStatus(ctx, req)
}

func (s *Server) handleListJulesActivities(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.julesClient.APIKey() == "" {
		return mcp.NewToolResultError("Error: JULES_API_KEY environment variable is not set for the MCP server."), nil
	}

	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	if sessionID == "" {
		return mcp.NewToolResultError("Error: Missing session_id."), nil
	}

	pageSize := req.GetInt("page_size", 10)
	pageToken := strings.TrimSpace(req.GetString("page_token", ""))

	resp, err := s.julesClient.ListActivities(ctx, sessionID, pageSize, pageToken)
	if err != nil {
		s.logger.ErrorContext(ctx, "list_jules_activities failed", "session_id", sessionID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error listing activities: %v", err)), nil
	}

	cleanID := jules.CleanSessionID(sessionID)
	formatted := formatter.FormatActivities(cleanID, resp.Activities)
	return mcp.NewToolResultText(formatted), nil
}

func (s *Server) handleSendJulesMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.julesClient.APIKey() == "" {
		return mcp.NewToolResultError("Error: JULES_API_KEY environment variable is not set for the MCP server."), nil
	}

	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	prompt := strings.TrimSpace(req.GetString("prompt", ""))

	if sessionID == "" {
		return mcp.NewToolResultError("Error: Missing session_id."), nil
	}
	if prompt == "" {
		return mcp.NewToolResultError("Error: Missing prompt message."), nil
	}

	err := s.julesClient.SendMessage(ctx, sessionID, prompt)
	if err != nil {
		s.logger.ErrorContext(ctx, "send_jules_message failed", "session_id", sessionID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error sending message to session: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message successfully sent to Jules session `%s`.", jules.CleanSessionID(sessionID))), nil
}

func (s *Server) handleApproveJulesPlan(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if s.julesClient.APIKey() == "" {
		return mcp.NewToolResultError("Error: JULES_API_KEY environment variable is not set for the MCP server."), nil
	}

	sessionID := strings.TrimSpace(req.GetString("session_id", ""))
	if sessionID == "" {
		return mcp.NewToolResultError("Error: Missing session_id."), nil
	}

	// R2: Mandatory WARNING log when releasing plan approval gate
	s.logger.WarnContext(ctx, "RELEASING HUMAN-GATE: approve_jules_plan invoked", "session_id", sessionID)

	err := s.julesClient.ApprovePlan(ctx, sessionID)
	if err != nil {
		s.logger.ErrorContext(ctx, "approve_jules_plan failed", "session_id", sessionID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Error approving plan: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Plan successfully approved for Jules session `%s`. Execution resumed.", jules.CleanSessionID(sessionID))), nil
}
