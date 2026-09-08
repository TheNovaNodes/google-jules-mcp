package jules

import (
	"context"
	"strings"
)

// BranchResolver defines the interface for resolving the starting branch of a repo.
type BranchResolver interface {
	ResolveStartingBranch(ctx context.Context, sourceName string, explicitBranch string) (string, error)
}

// ResolveStartingBranch implements the R1 branch resolution specification:
// 1. Explicit starting_branch argument takes highest precedence.
// 2. Auto policy: resolve source default branch from ListSources (defaultBranch.displayName).
// 3. If source cannot be resolved or has no default branch, return "" (omitted from payload, letting Google use repo default).
// 4. Literal "main" is never hardcoded as a fallback.
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

	normSource := strings.TrimSpace(sourceName)
	for _, s := range sources {
		if strings.EqualFold(s.Name, normSource) || strings.EqualFold(s.ID, normSource) {
			if s.GithubRepo != nil && s.GithubRepo.DefaultBranch != nil && s.GithubRepo.DefaultBranch.DisplayName != "" {
				return s.GithubRepo.DefaultBranch.DisplayName, nil
			}
		}
	}

	c.logger.WarnContext(ctx, "Source default branch not found in ListSources, omitting startingBranch", "source", sourceName)
	return "", nil // Omit field, let Google pick repo default
}
