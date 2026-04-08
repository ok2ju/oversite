#!/usr/bin/env bash
# PostToolUse hook: run tsc --noEmit after TS/TSX edits to catch type errors.
# ESLint catches style issues but not type errors — this closes that gap.
set -euo pipefail

INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] && exit 0
[ ! -f "$FILE" ] && exit 0

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Only run on TypeScript files under frontend/src/
if [[ "$FILE" == */frontend/src/*.ts ]] || [[ "$FILE" == */frontend/src/*.tsx ]]; then
  cd "$PROJECT_ROOT/frontend"
  # Run typecheck; show only errors (not the "found X errors" noise)
  npx tsc --noEmit 2>&1 | head -40
  exit ${PIPESTATUS[0]}
fi

exit 0
