package server

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/TheNovaNodes/google-jules-mcp/internal/jules"
)

// TestReadmeMatchesToolRegistry verifies R6 (Docs <-> Registry contract).
// This test ensures that the tool table in README.md matches the server's registered tools exactly.
func TestReadmeMatchesToolRegistry(t *testing.T) {
	readmePath := "../../README.md"
	file, err := os.Open(readmePath)
	if err != nil {
		// Try from repo root if running from root
		readmePath = "README.md"
		file, err = os.Open(readmePath)
		if err != nil {
			t.Fatalf("failed to open README.md: %v", err)
		}
	}
	defer file.Close()

	var readmeTools []string
	toolRegex := regexp.MustCompile(`^\|\s*` + "`" + `([a-zA-Z0-9_]+)` + "`" + `\s*\|`)

	scanner := bufio.NewScanner(file)
	inTable := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "| Tool |") {
			inTable = true
			continue
		}
		if inTable {
			if !strings.HasPrefix(line, "|") {
				inTable = false
				continue
			}
			matches := toolRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				readmeTools = append(readmeTools, matches[1])
			}
		}
	}

	if len(readmeTools) == 0 {
		t.Fatal("no tools parsed from README.md tool table; check formatting")
	}

	client := jules.NewClient("test")
	srv := NewServer(client, nil)

	registeredTools := []string{
		"list_jules_sources",
		"delegate_task_to_jules",
		"check_jules_status",
		"get_jules_session",
		"list_jules_activities",
		"send_jules_message",
		"approve_jules_plan",
		"get_jules_patch",
	}

	sort.Strings(readmeTools)
	sort.Strings(registeredTools)

	if len(readmeTools) != len(registeredTools) {
		t.Fatalf("tool count mismatch between README (%d) and server registry (%d)\nREADME: %+v\nRegistry: %+v",
			len(readmeTools), len(registeredTools), readmeTools, registeredTools)
	}

	for i := range registeredTools {
		if readmeTools[i] != registeredTools[i] {
			t.Errorf("mismatch at index %d: README has %q, server registry has %q",
				i, readmeTools[i], registeredTools[i])
		}
	}

	// Verify the server actually instantiated
	if srv.mcpServer == nil {
		t.Fatal("mcpServer is nil")
	}
}
