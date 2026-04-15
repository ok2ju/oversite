#!/usr/bin/env bash
# Stop hook: run tests for files edited during this turn
# Runs once when Claude's turn ends — not on every edit
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TRACKING_FILE="$PROJECT_ROOT/.claude/.edited-files"

# Nothing edited this turn → skip
[ ! -f "$TRACKING_FILE" ] && exit 0

# Read unique file paths
CHANGED=$(sort -u "$TRACKING_FILE")
[ -z "$CHANGED" ] && exit 0

# Categorize changed files
FE_TEST_FILES=()
FE_SOURCE_FILES=()
BE_PACKAGES=()
ROOT_GO_PACKAGES=()

while IFS= read -r file; do
  if [[ "$file" == */frontend/src/*.test.ts ]] || [[ "$file" == */frontend/src/*.test.tsx ]]; then
    FE_TEST_FILES+=("$file")
  elif [[ "$file" == */frontend/src/*.ts ]] || [[ "$file" == */frontend/src/*.tsx ]]; then
    FE_SOURCE_FILES+=("$file")
  elif [[ "$file" == */backend/*.go ]]; then
    pkg_dir=$(dirname "${file#*backend/}")
    BE_PACKAGES+=("./$pkg_dir/...")
  elif [[ "$file" == *.go ]]; then
    # Root-level Go files (Wails app, internal/)
    rel="${file#$PROJECT_ROOT/}"
    pkg_dir=$(dirname "$rel")
    if [ "$pkg_dir" = "." ]; then
      ROOT_GO_PACKAGES+=("./...")
    else
      ROOT_GO_PACKAGES+=("./$pkg_dir/...")
    fi
  fi
done <<< "$CHANGED"

# Deduplicate Go packages
if [ "${#BE_PACKAGES[@]}" -gt 0 ]; then
  UNIQUE_PKGS=$(printf '%s\n' "${BE_PACKAGES[@]}" | sort -u)
  BE_PACKAGES=()
  while IFS= read -r pkg; do
    [ -n "$pkg" ] && BE_PACKAGES+=("$pkg")
  done <<< "$UNIQUE_PKGS"
fi

FAILED=0

# --- Frontend tests ---
if [ ${#FE_TEST_FILES[@]} -gt 0 ] || [ ${#FE_SOURCE_FILES[@]} -gt 0 ]; then
  cd "$PROJECT_ROOT/frontend"

  if [ ${#FE_TEST_FILES[@]} -gt 0 ]; then
    echo "=== Frontend tests (direct) ==="
    npx vitest run "${FE_TEST_FILES[@]}" --reporter=verbose 2>&1 | tail -40 || { FAILED=1; echo "FAIL: frontend direct tests"; }
  fi

  if [ ${#FE_SOURCE_FILES[@]} -gt 0 ]; then
    echo "=== Frontend tests (related) ==="
    npx vitest run --related "${FE_SOURCE_FILES[@]}" --reporter=verbose --passWithNoTests 2>&1 | tail -40 || { FAILED=1; echo "FAIL: frontend related tests"; }
  fi
fi

# --- Backend tests ---
if [ ${#BE_PACKAGES[@]} -gt 0 ]; then
  cd "$PROJECT_ROOT/backend"
  echo "=== Backend tests ==="
  go test -race -count=1 -timeout=60s "${BE_PACKAGES[@]}" 2>&1 | tail -40 || { FAILED=1; echo "FAIL: backend tests"; }
fi

# --- Root-level Go tests (Wails app, internal/) ---
if [ ${#ROOT_GO_PACKAGES[@]} -gt 0 ]; then
  UNIQUE_ROOT_PKGS=$(printf '%s\n' "${ROOT_GO_PACKAGES[@]}" | sort -u)
  ROOT_GO_PACKAGES=()
  while IFS= read -r pkg; do
    [ -n "$pkg" ] && ROOT_GO_PACKAGES+=("$pkg")
  done <<< "$UNIQUE_ROOT_PKGS"

  cd "$PROJECT_ROOT"
  echo "=== Root Go tests ==="
  GOCACHE="${TMPDIR:-/tmp}/go-build" go test -race -count=1 -timeout=60s -buildvcs=false "${ROOT_GO_PACKAGES[@]}" 2>&1 | tail -40 || { FAILED=1; echo "FAIL: root Go tests"; }
fi

exit $FAILED
