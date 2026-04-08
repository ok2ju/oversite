#!/usr/bin/env bash
# PostToolUse hook: auto-lint TS/TSX and auto-format Go files after edits
set -euo pipefail

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# TypeScript/TSX files under frontend/ → eslint --fix
if [[ "$FILE" == */frontend/src/*.ts ]] || [[ "$FILE" == */frontend/src/*.tsx ]]; then
  cd "$PROJECT_ROOT/frontend"
  npx eslint --fix "$FILE" 2>/dev/null || true
  exit 0
fi

# Go files under backend/ → gofmt
if [[ "$FILE" == */backend/*.go ]]; then
  gofmt -w "$FILE" 2>/dev/null || true
  goimports -w "$FILE" 2>/dev/null || true
  exit 0
fi

exit 0
