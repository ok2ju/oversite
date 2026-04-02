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
# shellcheck disable=SC2086
exec go tool golangci-lint run $packages
