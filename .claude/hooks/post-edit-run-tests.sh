#!/usr/bin/env bash
# PostToolUse hook: auto-run tests when a test file is edited.
# Gives Claude immediate feedback on broken tests, mock issues, and syntax errors.
set -euo pipefail

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# --- Frontend test files ---
if [[ "$FILE" == */frontend/src/*.test.ts ]] || [[ "$FILE" == */frontend/src/*.test.tsx ]]; then
  cd "$PROJECT_ROOT/frontend"
  # Run only the edited test file with a timeout
  npx vitest run "$FILE" --reporter=verbose 2>&1 | tail -40
  exit ${PIPESTATUS[0]}
fi

# --- Go test files ---
if [[ "$FILE" == */backend/*_test.go ]]; then
  cd "$PROJECT_ROOT/backend"
  # Derive the Go package path from the file
  REL_DIR=$(dirname "${FILE#*backend/}")
  # Run tests for just this package with race detector
  go test -race -count=1 -timeout=30s "./${REL_DIR}/..." 2>&1 | tail -40
  exit ${PIPESTATUS[0]}
fi

exit 0
