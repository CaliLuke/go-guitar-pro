#!/usr/bin/env bash
# Code quality gates — run before committing or pushing.
# Usage: ./check.sh [--fix]
set -euo pipefail

FIX="${1:-}"
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

if [[ -n "$FIX" && "$FIX" != "--fix" ]]; then
  echo "usage: ./check.sh [--fix]" >&2
  exit 2
fi

FAILED=()

run_gate() {
  local name="$1"
  shift
  echo ""
  echo "▶ $name"
  if "$@"; then
    return
  fi
  FAILED+=("$name")
}

check_goimports() {
  local output
  output="$(goimports -l .)"
  if [[ -n "$output" ]]; then
    echo "$output"
    return 1
  fi
}

check_duplication() {
  local output
  output="$(dupl -threshold 75 .)"
  echo "$output"
  if grep -q '^found ' <<<"$output"; then
    return 1
  fi
}

test_with_coverage() {
  local report coverage
  report="$(mktemp /tmp/go-guitar-pro-coverage.XXXXXX)"
  if ! go test -race -count=1 -timeout=120s -covermode=atomic -coverprofile="$report" ./...; then
    rm -f "$report"
    return 1
  fi
  coverage="$(go tool cover -func="$report" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"
  echo "total coverage: ${coverage}% (minimum 80.0%)"
  rm -f "$report"
  awk -v coverage="$coverage" 'BEGIN { exit !(coverage + 0 >= 80.0) }'
}

run_gate "go build" go build ./...
run_gate "go vet" go vet ./...

if [[ "$FIX" == "--fix" ]]; then
  run_gate "goimports" goimports -w .
else
  run_gate "goimports" check_goimports
fi

run_gate "go mod tidy drift" go mod tidy -diff

if [[ "$FIX" == "--fix" ]]; then
  run_gate "golangci-lint" golangci-lint run --fix
else
  run_gate "golangci-lint" golangci-lint run
fi

run_gate "duplication" check_duplication
run_gate "AlphaTab semantic conformance" ./conformance/check.sh
run_gate "race tests and coverage" test_with_coverage

echo ""
if [[ "${#FAILED[@]}" -ne 0 ]]; then
  echo "${#FAILED[@]} quality gate(s) failed:"
  printf '  - %s\n' "${FAILED[@]}"
  exit 1
fi
echo "All quality gates passed"
