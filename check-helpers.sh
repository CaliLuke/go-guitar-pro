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
  if ! output="$(goimports -l .)"; then
    echo "goimports failed" >&2
    return 1
  fi
  if [[ -n "$output" ]]; then
    echo "$output"
    return 1
  fi
}

check_duplication() {
  local output
  if ! output="$(dupl -threshold 75 .)"; then
    echo "dupl failed" >&2
    return 1
  fi
  echo "$output"
  if grep -q '^found ' <<<"$output"; then
    return 1
  fi
}

test_with_coverage() {
  local report cover_output coverage
  if ! report="$(mktemp /tmp/go-guitar-pro-coverage.XXXXXX)"; then
    echo "could not create coverage report" >&2
    return 1
  fi
  if ! go test -race -count=1 -timeout=600s -covermode=atomic -coverprofile="$report" ./...; then
    rm -f "$report"
    return 1
  fi
  if ! cover_output="$(go tool cover -func="$report")"; then
    echo "could not read coverage report" >&2
    rm -f "$report"
    return 1
  fi
  coverage="$(awk '/^total:/ {gsub(/%/, "", $3); print $3}' <<<"$cover_output")"
  if [[ -z "$coverage" ]]; then
    echo "coverage report has no total" >&2
    rm -f "$report"
    return 1
  fi
  echo "total coverage: ${coverage}% (minimum 80.0%)"
  rm -f "$report"
  awk -v coverage="$coverage" 'BEGIN { exit !(coverage + 0 >= 80.0) }'
}
