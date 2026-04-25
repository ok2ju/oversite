#!/usr/bin/env bash
# PostToolUse hook: auto-format files after edits (fast, runs on every edit)
# FE: prettier + eslint --fix | BE: gofmt + goimports
# Also tracks edited files for Stop hooks (tests, typecheck)
set -euo pipefail

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Track edited files for Stop hooks
echo "$FILE" >> "$PROJECT_ROOT/.claude/.edited-files"

# --- TypeScript/TSX → prettier + eslint --fix ---
if [[ "$FILE" == */frontend/src/*.ts ]] || [[ "$FILE" == */frontend/src/*.tsx ]]; then
  cd "$PROJECT_ROOT/frontend"
  # Run prettier if installed (graceful skip otherwise)
  if [ -x "node_modules/.bin/prettier" ]; then
    npx prettier --write "$FILE" 2>/dev/null || true
  fi
  npx eslint --fix "$FILE" 2>/dev/null || true
  exit 0
fi

# --- Go → gofmt + goimports ---
if [[ "$FILE" == */backend/*.go ]]; then
  gofmt -w "$FILE" 2>/dev/null || true
  command -v goimports >/dev/null 2>&1 && goimports -w "$FILE" 2>/dev/null || true
  exit 0
fi

# --- Root-level Go (Wails app, internal/) → gofmt + goimports ---
if [[ "$FILE" == *.go ]] && [[ "$FILE" != */backend/*.go ]]; then
  gofmt -w "$FILE" 2>/dev/null || true
  command -v goimports >/dev/null 2>&1 && goimports -w "$FILE" 2>/dev/null || true
  exit 0
fi

exit 0
