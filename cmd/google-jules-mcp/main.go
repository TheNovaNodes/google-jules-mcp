package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/TheNovaNodes/google-jules-mcp/internal/jules"
	"github.com/TheNovaNodes/google-jules-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// loadDotEnv parses a local .env file if present without external dependencies.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

func main() {
	loadDotEnv(".env")

	// FastMCP stdio server logs to stderr so stdin/stdout are preserved for MCP JSON-RPC
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	apiKey := os.Getenv("JULES_API_KEY")
	if apiKey == "" {
		logger.Warn("JULES_API_KEY environment variable is not set. MCP tools requiring API access will return an error.")
	}

	client := jules.NewClient(apiKey, jules.WithLogger(logger))
	srv := server.NewServer(client, logger)

	logger.Info("Starting Google Jules MCP Server (stdio transport)...")
	if err := mcpserver.ServeStdio(srv.MCPServer()); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: MCP Server terminated with error: %v\n", err)
		os.Exit(1)
	}
}
