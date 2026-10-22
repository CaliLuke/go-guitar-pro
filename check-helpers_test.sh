#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
TEMP_DIR="$(mktemp -d /tmp/go-guitar-pro-check-helpers.XXXXXX)"
trap 'rm -rf "$TEMP_DIR"' EXIT

mkdir -p "$TEMP_DIR/bin"
cat >"$TEMP_DIR/bin/goimports" <<'EOF'
#!/usr/bin/env bash
case "${GOIMPORTS_MODE:-clean}" in
  fail) exit 42 ;;
  violation) echo 'misformatted.go' ;;
esac
EOF
cat >"$TEMP_DIR/bin/dupl" <<'EOF'
#!/usr/bin/env bash
case "${DUPL_MODE:-clean}" in
  fail) exit 42 ;;
  violation) echo 'found duplicate code' ;;
esac
EOF
cat >"$TEMP_DIR/bin/go" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == test ]]; then
  for arg in "$@"; do
    if [[ "$arg" == -coverprofile=* ]]; then
      printf 'mode: atomic\n' >"${arg#-coverprofile=}"
    fi
  done
  [[ "${GO_TEST_MODE:-clean}" != fail ]]
  exit
fi
if [[ "$1" == tool && "$2" == cover ]]; then
  case "${COVER_MODE:-clean}" in
    fail) exit 42 ;;
    missing-total) echo 'no total row' ;;
    below-floor) echo 'total: (statements) 70.0%' ;;
    *) echo 'total: (statements) 90.0%' ;;
  esac
  exit
fi
exit 2
EOF
chmod +x "$TEMP_DIR/bin/goimports" "$TEMP_DIR/bin/dupl" "$TEMP_DIR/bin/go"
PATH="$TEMP_DIR/bin:$PATH"
source "$ROOT/check-helpers.sh"

expect_success() {
  local label="$1"
  shift
  if ! "$@" >/dev/null 2>&1; then
    echo "expected success: $label" >&2
    exit 1
  fi
}

expect_failure() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo "expected failure: $label" >&2
    exit 1
  fi
}

goimports_fail() { GOIMPORTS_MODE=fail check_goimports; }
goimports_clean() { GOIMPORTS_MODE=clean check_goimports; }
goimports_violation() { GOIMPORTS_MODE=violation check_goimports; }
dupl_fail() { DUPL_MODE=fail check_duplication; }
dupl_clean() { DUPL_MODE=clean check_duplication; }
dupl_violation() { DUPL_MODE=violation check_duplication; }
go_test_fail() { GO_TEST_MODE=fail test_with_coverage; }
coverage_fail() { COVER_MODE=fail test_with_coverage; }
coverage_missing_total() { COVER_MODE=missing-total test_with_coverage; }
coverage_below_floor() { COVER_MODE=below-floor test_with_coverage; }
coverage_clean() { COVER_MODE=clean test_with_coverage; }

expect_failure 'goimports command error' goimports_fail
expect_success 'goimports clean output' goimports_clean
expect_failure 'goimports formatting findings' goimports_violation

expect_failure 'dupl command error' dupl_fail
expect_success 'dupl clean output' dupl_clean
expect_failure 'dupl findings' dupl_violation

expect_failure 'go test failure' go_test_fail
expect_failure 'coverage command error' coverage_fail
expect_failure 'coverage output without total' coverage_missing_total
expect_failure 'coverage below floor' coverage_below_floor
expect_success 'coverage above floor' coverage_clean

echo 'quality-gate helper checks passed'
