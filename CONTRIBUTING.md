# Contributing to Google Jules MCP Server

First off, thank you for considering contributing to the Google Jules MCP Server! It's people like you that make the Antigravity Agent Ecosystem such a great environment to build and create within.

## Code of Conduct
By participating in this project, you are expected to uphold our Code of Conduct. Please be respectful and considerate of others when submitting issues, pull requests, or commenting.

## How Can I Contribute?

### Reporting Bugs
This section guides you through submitting a bug report. Following these guidelines helps maintainers and the community understand your report, reproduce the behavior, and find related reports.

*   **Use a clear and descriptive title** for the issue to identify the problem.
*   **Describe the exact steps which reproduce the problem** in as many details as possible.
*   **Provide specific examples to demonstrate the steps**.

### Suggesting Enhancements
This section guides you through submitting an enhancement suggestion, including completely new features and minor improvements to existing functionality.

*   **Use a clear and descriptive title** for the issue to identify the suggestion.
*   **Provide a step-by-step description of the suggested enhancement** in as many details as possible.
*   **Explain why this enhancement would be useful** to most users.

### Pull Requests
*   Fill in the required template.
*   Do not include issue numbers in the PR title.
*   Follow the styleguides.
*   After you submit your pull request, verify that all status checks are passing.

## Styleguides

### Go Guidelines
* All code must be formatted correctly (`gofmt -s -w .`) and pass static analysis (`go vet ./...`).
* Ensure all new features or bug fixes are covered by appropriate unit tests (`go test -v -race -cover ./...`).
* Maintain zero-leakage credential hygiene: never commit secrets, tokens, or hardcoded API keys.
* For stdio MCP protocol hygiene, never print to `os.Stdout`. Route all logs to `os.Stderr` via `log/slog`.

## Setting Up Your Development Environment
1. Fork the repository and clone your fork locally.
2. Ensure you have Go 1.22+ installed (`go version`).
3. Run tests with race detection: `make test` (or `go test -v -race -cover ./...`).
4. Build the static binary: `make build` (outputs to `bin/google-jules-mcp`).
5. Verify linting: `make lint` (or `go vet ./...`).

Thank you for your contributions!

