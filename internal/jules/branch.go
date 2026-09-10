package jules

import (
	"context"
	"strings"
)

// BranchResolver defines the interface for resolving the starting branch of a repo.
type BranchResolver interface {
	ResolveStartingBranch(ctx context.Context, sourceName string, explicitBranch string) (string, error)
}

// NormalizeSourceName standardizes any repository reference format into the canonical
// Google Jules format "sources/github/owner/repo".
func NormalizeSourceName(input string) string {
	s := strings.TrimSpace(input)
	s = strings.TrimPrefix(s, "https://github.com/")
	s = strings.TrimPrefix(s, "http://github.com/")
	s = strings.TrimPrefix(s, "github.com/")
	s = strings.TrimSuffix(s, ".git")
	s = strings.Trim(s, "/")
	if strings.HasPrefix(s, "sources/github/") {
		return s
	}
	if strings.HasPrefix(s, "github/") {
		return "sources/" + s
	}
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		return "sources/github/" + s
	}
	return s
}

// ResolveStartingBranch implements the R1 branch resolution specification:
// 1. Explicit starting_branch argument (branch, tag, or commit SHA) takes highest precedence.
// 2. Auto policy: resolve source default branch from ListSources (defaultBranch.displayName).
// 3. Canonical source normalization handles all variants: "sources/github/owner/repo", "github/owner/repo", "owner/repo", "https://github.com/owner/repo".
// 4. If source cannot be resolved or has no default branch, return "" (omitted from payload, letting Google use repo default).
// 5. Literal "main" is never hardcoded as a fallback.
func (c *Client) ResolveStartingBranch(ctx context.Context, sourceName string, explicitBranch string) (string, error) {
	trimmedExplicit := strings.TrimSpace(explicitBranch)
	if trimmedExplicit != "" {
		return trimmedExplicit, nil
	}

	sources, err := c.ListSources(ctx)
	if err != nil {
		c.logger.WarnContext(ctx, "Failed to fetch sources for branch resolution, defaulting to empty (omitted)", "source", sourceName, "error", err)
		return "", nil // Omit field, let Google pick repo default
	}

	canonSource := NormalizeSourceName(sourceName)
	for _, s := range sources {
		if strings.EqualFold(NormalizeSourceName(s.Name), canonSource) || strings.EqualFold(NormalizeSourceName(s.ID), canonSource) {
			if s.GithubRepo != nil && s.GithubRepo.DefaultBranch != nil && s.GithubRepo.DefaultBranch.DisplayName != "" {
				return s.GithubRepo.DefaultBranch.DisplayName, nil
			}
		}
	}

	c.logger.WarnContext(ctx, "Source default branch not found in ListSources, omitting startingBranch", "source", sourceName)
	return "", nil // Omit field, let Google pick repo default
}
