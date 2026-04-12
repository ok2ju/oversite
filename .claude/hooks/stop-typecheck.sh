#!/usr/bin/env bash
# Stop hook: run type checking and static analysis for changed files
# Heavier checks that run once when Claude's turn ends
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TRACKING_FILE="$PROJECT_ROOT/.claude/.edited-files"

# Nothing edited this turn → skip
[ ! -f "$TRACKING_FILE" ] && exit 0

# Read tracked files and clean up (this is the last Stop hook)
CHANGED=$(sort -u "$TRACKING_FILE")
rm -f "$TRACKING_FILE"
[ -z "$CHANGED" ] && exit 0

HAS_FE=false
HAS_BE=false

while IFS= read -r file; do
  if [[ "$file" == */frontend/src/*.ts ]] || [[ "$file" == */frontend/src/*.tsx ]]; then
    HAS_FE=true
  elif [[ "$file" == */backend/*.go ]]; then
    HAS_BE=true
  fi
done <<< "$CHANGED"

FAILED=0

# --- TypeScript type check ---
if $HAS_FE; then
  cd "$PROJECT_ROOT/frontend"
  echo "=== TypeScript type check ==="
  npx tsc --noEmit 2>&1 | head -40 || FAILED=1
fi

# --- Go vet ---
if $HAS_BE; then
  cd "$PROJECT_ROOT/backend"
  echo "=== Go vet ==="
  go vet ./... 2>&1 | head -40 || FAILED=1
fi

exit $FAILED
