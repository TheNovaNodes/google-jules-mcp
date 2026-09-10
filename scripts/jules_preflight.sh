#!/usr/bin/env bash
# jules_preflight.sh — Pre-flight anti-duplicate and anti-spam validator for Jules delegations (Phase 0)
set -euo pipefail

REPO="${1:-}"
if [[ -z "$REPO" ]]; then
  echo "Usage: $0 <owner/repo or sources/github/owner/repo>"
  exit 1
fi

# Normalize repo to owner/repo
REPO="${REPO#sources/github/}"
REPO="${REPO#github/}"
REPO="${REPO#https://github.com/}"
REPO="${REPO%.git}"

echo "🔍 [Phase 0 Pre-flight] Validating target repository: $REPO..."

# Check GH_TOKEN availability
if [[ -z "${GH_TOKEN:-}" ]] && [[ -f /root/projects/.credentials/TheNovaNodes.env ]]; then
  # shellcheck disable=SC1091
  source /root/projects/.credentials/TheNovaNodes.env
  export GH_TOKEN="${GITHUB_PAT_NOVANODES:-}"
fi

# 1. Check GitHub open PRs
echo "👉 Checking open PRs in $REPO..."
OPEN_PRS=$(gh pr list --repo "$REPO" --state open --json number,title,headRefName --jq '.[] | "  #\(.number): \(.title) (\(.headRefName))"' 2>/dev/null || true)
if [[ -n "$OPEN_PRS" ]]; then
  echo "⚠️  WARNING: Open PRs found in $REPO:"
  echo "$OPEN_PRS"
  echo "🚨 Check carefully before delegating new tasks to Jules to prevent duplicate PRs!"
else
  echo "✅ No open PRs found in $REPO."
fi

# 2. Check open issues
echo "👉 Checking open issues in $REPO..."
OPEN_ISSUES=$(gh issue list --repo "$REPO" --state open --limit 5 --json number,title --jq '.[] | "  #\(.number): \(.title)"' 2>/dev/null || true)
if [[ -n "$OPEN_ISSUES" ]]; then
  echo "ℹ️  Open issues in $REPO (top 5):"
  echo "$OPEN_ISSUES"
else
  echo "✅ No open issues found in $REPO."
fi

echo "🛡️  Pre-flight validation complete for $REPO."
