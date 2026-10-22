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

source "$ROOT/check-helpers.sh"

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
run_gate "quality gate helper tests" ./check-helpers_test.sh
run_gate "AlphaTab semantic conformance" ./conformance/check.sh
run_gate "race tests and coverage" test_with_coverage

echo ""
if [[ "${#FAILED[@]}" -ne 0 ]]; then
  echo "${#FAILED[@]} quality gate(s) failed:"
  printf '  - %s\n' "${FAILED[@]}"
  exit 1
fi
echo "All quality gates passed"
