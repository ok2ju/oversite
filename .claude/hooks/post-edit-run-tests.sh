#!/usr/bin/env bash
# PostToolUse hook: auto-run tests when a test or source file is edited.
# Gives Claude immediate feedback on broken tests, mock issues, and syntax errors.
set -euo pipefail

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# --- Frontend test files (direct run) ---
if [[ "$FILE" == */frontend/src/*.test.ts ]] || [[ "$FILE" == */frontend/src/*.test.tsx ]]; then
  cd "$PROJECT_ROOT/frontend"
  npx vitest run "$FILE" --reporter=verbose 2>&1 | tail -40
  exit ${PIPESTATUS[0]}
fi

# --- Frontend source files (run related tests) ---
if [[ "$FILE" == */frontend/src/*.ts ]] || [[ "$FILE" == */frontend/src/*.tsx ]]; then
  cd "$PROJECT_ROOT/frontend"
  # Use vitest --related to find and run tests that import this file
  npx vitest run --related "$FILE" --reporter=verbose --passWithNoTests 2>&1 | tail -40
  exit ${PIPESTATUS[0]}
fi

# --- Go test files (direct run) ---
if [[ "$FILE" == */backend/*_test.go ]]; then
  cd "$PROJECT_ROOT/backend"
  REL_DIR=$(dirname "${FILE#*backend/}")
  go test -race -count=1 -timeout=30s "./${REL_DIR}/..." 2>&1 | tail -40
  exit ${PIPESTATUS[0]}
fi

# --- Go source files (run package tests) ---
if [[ "$FILE" == */backend/*.go ]]; then
  cd "$PROJECT_ROOT/backend"
  REL_DIR=$(dirname "${FILE#*backend/}")
  go test -race -count=1 -timeout=30s "./${REL_DIR}/..." 2>&1 | tail -40
  exit ${PIPESTATUS[0]}
fi

exit 0
