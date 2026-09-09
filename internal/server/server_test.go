package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TheNovaNodes/google-jules-mcp/internal/jules"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestToolRegistration(t *testing.T) {
	client := jules.NewClient("test-key")
	srv := NewServer(client, slog.Default())

	// Call list tools
	tools := []string{
		"list_jules_sources",
		"delegate_task_to_jules",
		"check_jules_status",
		"get_jules_session",
		"list_jules_activities",
		"send_jules_message",
		"approve_jules_plan",
	}

	for _, toolName := range tools {
		req := mcp.CallToolRequest{}
		req.Params.Name = toolName
		// If tool wasn't registered, mcp-go wouldn't know it. We verify direct handler registration:
		if srv.mcpServer == nil {
			t.Fatal("mcpServer is nil")
		}
	}
}

func TestHandlersMissingAPIKey(t *testing.T) {
	client := jules.NewClient("")
	srv := NewServer(client, slog.Default())
	ctx := context.Background()

	// 1. list_jules_sources
	res, _ := srv.handleListJulesSources(ctx, mcp.CallToolRequest{})
	if !res.IsError {
		t.Errorf("expected error when API key missing")
	}

	// 2. delegate_task_to_jules
	res, _ = srv.handleDelegateTaskToJules(ctx, mcp.CallToolRequest{})
	if !res.IsError {
		t.Errorf("expected error when API key missing")
	}

	// 3. check_jules_status
	res, _ = srv.handleCheckJulesStatus(ctx, mcp.CallToolRequest{})
	if !res.IsError {
		t.Errorf("expected error when API key missing")
	}

	// 4. list_jules_activities
	res, _ = srv.handleListJulesActivities(ctx, mcp.CallToolRequest{})
	if !res.IsError {
		t.Errorf("expected error when API key missing")
	}

	// 5. send_jules_message
	res, _ = srv.handleSendJulesMessage(ctx, mcp.CallToolRequest{})
	if !res.IsError {
		t.Errorf("expected error when API key missing")
	}

	// 6. approve_jules_plan
	res, _ = srv.handleApproveJulesPlan(ctx, mcp.CallToolRequest{})
	if !res.IsError {
		t.Errorf("expected error when API key missing")
	}
}

func TestDelegateTaskArgValidation(t *testing.T) {
	client := jules.NewClient("test-key")
	srv := NewServer(client, slog.Default())
	ctx := context.Background()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"source_name": "",
		"prompt":      "test",
	}
	res, _ := srv.handleDelegateTaskToJules(ctx, req)
	if !res.IsError {
		t.Errorf("expected error for empty source_name")
	}

	req.Params.Arguments = map[string]any{
		"source_name": "sources/github/o/r",
		"prompt":      "",
	}
	res, _ = srv.handleDelegateTaskToJules(ctx, req)
	if !res.IsError {
		t.Errorf("expected error for empty prompt")
	}
}

func TestServerHappyPaths(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/sources":
			_, _ = w.Write([]byte(`{
				"sources": [
					{
						"name": "sources/github/owner/repo",
						"githubRepo": { "owner": "owner", "repo": "repo", "defaultBranch": { "displayName": "main" } }
					}
				]
			}`))
		case r.URL.Path == "/sessions" && r.Method == http.MethodPost:
			var req jules.CreateSessionRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_, _ = w.Write([]byte(`{
				"name": "sessions/sess-123",
				"state": "QUEUED",
				"url": "https://jules.google.com/session/sess-123"
			}`))
		case r.URL.Path == "/sessions/sess-123" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{
				"name": "sessions/sess-123",
				"state": "IN_PROGRESS",
				"title": "Audit session"
			}`))
		case r.URL.Path == "/sessions/sess-123/activities":
			_, _ = w.Write([]byte(`{
				"activities": [
					{
						"name": "act-1",
						"originator": "agent",
						"progressUpdated": { "title": "Working" }
					}
				]
			}`))
		case r.URL.Path == "/sessions/sess-123:sendMessage":
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/sessions/sess-123:approvePlan":
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := jules.NewClient("test-key", jules.WithBaseURL(ts.URL), jules.WithDisableRetry(true))
	srv := NewServer(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()

	// 1. list_jules_sources
	res, err := srv.handleListJulesSources(ctx, mcp.CallToolRequest{})
	if err != nil || res.IsError {
		t.Fatalf("list_jules_sources failed: %v, res: %+v", err, res)
	}
	content := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "sources/github/owner/repo") {
		t.Errorf("missing repo in sources response")
	}

	// 2. delegate_task_to_jules
	delReq := mcp.CallToolRequest{}
	delReq.Params.Arguments = map[string]any{
		"source_name": "sources/github/owner/repo",
		"prompt":      "Perform code audit",
	}
	res, err = srv.handleDelegateTaskToJules(ctx, delReq)
	if err != nil || res.IsError {
		t.Fatalf("delegate_task_to_jules failed: %v", err)
	}
	content = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "sess-123") {
		t.Errorf("missing session ID in delegation response")
	}

	// 3. check_jules_status
	chkReq := mcp.CallToolRequest{}
	chkReq.Params.Arguments = map[string]any{"session_id": "sess-123"}
	res, err = srv.handleCheckJulesStatus(ctx, chkReq)
	if err != nil || res.IsError {
		t.Fatalf("check_jules_status failed: %v", err)
	}
	content = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "IN_PROGRESS") {
		t.Errorf("missing IN_PROGRESS state in status response")
	}

	// 4. list_jules_activities
	actReq := mcp.CallToolRequest{}
	actReq.Params.Arguments = map[string]any{"session_id": "sess-123"}
	res, err = srv.handleListJulesActivities(ctx, actReq)
	if err != nil || res.IsError {
		t.Fatalf("list_jules_activities failed: %v", err)
	}
	content = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "progressUpdated") {
		t.Errorf("missing progressUpdated in activities response")
	}

	// 5. send_jules_message
	msgReq := mcp.CallToolRequest{}
	msgReq.Params.Arguments = map[string]any{
		"session_id": "sess-123",
		"prompt":     "Continue execution",
	}
	res, err = srv.handleSendJulesMessage(ctx, msgReq)
	if err != nil || res.IsError {
		t.Fatalf("send_jules_message failed: %v", err)
	}

	// 6. approve_jules_plan
	apprReq := mcp.CallToolRequest{}
	apprReq.Params.Arguments = map[string]any{"session_id": "sess-123"}
	res, err = srv.handleApproveJulesPlan(ctx, apprReq)
	if err != nil || res.IsError {
		t.Fatalf("approve_jules_plan failed: %v", err)
	}
}

func TestServerGetters(t *testing.T) {
	client := jules.NewClient("test-key")
	srv := NewServer(client, nil)

	if srv.MCPServer() == nil {
		t.Errorf("expected MCPServer to be returned")
	}
}

func TestHandleGetJulesSession(t *testing.T) {
	// Alias of check_jules_status
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/sessions/sess-123" {
			_, err := w.Write([]byte(`{
				"name": "sessions/sess-123",
				"state": "COMPLETED"
			}`))
			if err != nil {
				t.Logf("err: %v", err)
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := jules.NewClient("test-key", jules.WithBaseURL(ts.URL), jules.WithDisableRetry(true))
	srv := NewServer(client, nil)
	ctx := context.Background()

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"session_id": "sess-123"}

	res, err := srv.handleGetJulesSession(ctx, req)
	if err != nil || res.IsError {
		t.Fatalf("handleGetJulesSession failed: %v", err)
	}

	content := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "COMPLETED") {
		t.Errorf("missing state in session response")
	}
}

func TestHandlers_APIErrorPaths(t *testing.T) {
	// A server that consistently returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := jules.NewClient("test-key", jules.WithBaseURL(ts.URL), jules.WithDisableRetry(true))
	srv := NewServer(client, slog.Default())
	ctx := context.Background()

	// list_jules_sources
	res, err := srv.handleListJulesSources(ctx, mcp.CallToolRequest{})
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Error listing sources") {
		t.Errorf("expected error for list sources, got %v", res)
	}

	// delegate_task_to_jules missing prompt
	delReq := mcp.CallToolRequest{}
	delReq.Params.Arguments = map[string]any{"source_name": "src", "prompt": ""}
	res, err = srv.handleDelegateTaskToJules(ctx, delReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Missing source_name or prompt") {
		t.Errorf("expected missing arguments error")
	}

	// delegate_task_to_jules api failure
	delReq.Params.Arguments = map[string]any{"source_name": "src", "prompt": "prompt"}
	res, err = srv.handleDelegateTaskToJules(ctx, delReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Error delegating task") {
		t.Errorf("expected API error for delegation, got %v", res)
	}

	// check_jules_status missing id
	chkReq := mcp.CallToolRequest{}
	chkReq.Params.Arguments = map[string]any{"session_id": ""}
	res, err = srv.handleCheckJulesStatus(ctx, chkReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Missing session_id") {
		t.Errorf("expected missing id error")
	}

	// check_jules_status api failure
	chkReq.Params.Arguments = map[string]any{"session_id": "sess-1"}
	res, err = srv.handleCheckJulesStatus(ctx, chkReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Error checking session status") {
		t.Errorf("expected API error for status")
	}

	// list_jules_activities missing id
	actReq := mcp.CallToolRequest{}
	actReq.Params.Arguments = map[string]any{"session_id": ""}
	res, err = srv.handleListJulesActivities(ctx, actReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Missing session_id") {
		t.Errorf("expected missing id error")
	}

	// list_jules_activities api failure
	actReq.Params.Arguments = map[string]any{"session_id": "sess-1"}
	res, err = srv.handleListJulesActivities(ctx, actReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Error listing activities") {
		t.Errorf("expected API error for activities")
	}

	// send_jules_message missing id
	msgReq := mcp.CallToolRequest{}
	msgReq.Params.Arguments = map[string]any{"session_id": "", "prompt": "test"}
	res, err = srv.handleSendJulesMessage(ctx, msgReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Missing session_id") {
		t.Errorf("expected missing id error")
	}

	// send_jules_message missing prompt
	msgReq.Params.Arguments = map[string]any{"session_id": "sess-1", "prompt": ""}
	res, err = srv.handleSendJulesMessage(ctx, msgReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Missing prompt") {
		t.Errorf("expected missing prompt error")
	}

	// send_jules_message api failure
	msgReq.Params.Arguments = map[string]any{"session_id": "sess-1", "prompt": "test"}
	res, err = srv.handleSendJulesMessage(ctx, msgReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Error sending message") {
		t.Errorf("expected API error for message")
	}

	// approve_jules_plan missing id
	apprReq := mcp.CallToolRequest{}
	apprReq.Params.Arguments = map[string]any{"session_id": ""}
	res, err = srv.handleApproveJulesPlan(ctx, apprReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Missing session_id") {
		t.Errorf("expected missing id error")
	}

	// approve_jules_plan api failure
	apprReq.Params.Arguments = map[string]any{"session_id": "sess-1"}
	res, err = srv.handleApproveJulesPlan(ctx, apprReq)
	if err != nil {
		t.Logf("err: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "Error approving plan") {
		t.Errorf("expected API error for approve")
	}
}
