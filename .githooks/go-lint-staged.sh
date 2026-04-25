#!/usr/bin/env bash
set -euo pipefail

# Derive unique Go packages from staged file paths, then lint those packages.
# Usage: go-lint-staged.sh <file1.go> <file2.go> ...

if [ $# -eq 0 ]; then
  exit 0
fi

# Extract unique package directories (bash 3.2 safe -- no associative arrays)
packages=$(printf '%s\n' "$@" | xargs -n1 dirname | sort -u | sed 's|^|./|')
pkg_count=$(echo "$packages" | wc -l | tr -d ' ')

echo "[pre-commit] golangci-lint: ${pkg_count} package(s)"

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "[pre-commit] ERROR: golangci-lint not found on PATH" >&2
  echo "[pre-commit] Install: brew install golangci-lint" >&2
  exit 1
fi

# shellcheck disable=SC2086
exec golangci-lint run $packages
