#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFORMANCE="$ROOT/conformance"
EXPECTED_VERSION="$(node -p 'require(process.argv[1]).version' "$CONFORMANCE/oracle.json")"
SOURCE_REPOSITORY="$(node -p 'require(process.argv[1]).sourceRepository' "$CONFORMANCE/oracle.json")"
SOURCE_REF="$(node -p 'require(process.argv[1]).sourceRef' "$CONFORMANCE/oracle.json")"
SOURCE_REVISION="$(node -p 'require(process.argv[1]).sourceRevision' "$CONFORMANCE/oracle.json")"
REFERENCE="$ROOT/references/alphaTab"

if [[ ! -f "$CONFORMANCE/node_modules/@coderline/alphatab/package.json" ]] ||
  ! node -e 'const p=require(process.argv[1]); process.exit(p.version === process.argv[2] ? 0 : 1)' \
    "$CONFORMANCE/node_modules/@coderline/alphatab/package.json" "$EXPECTED_VERSION"; then
  npm --prefix "$CONFORMANCE" ci --ignore-scripts --no-audit --no-fund
fi

if [[ ! -d "$REFERENCE/.git" ]]; then
  mkdir -p "$ROOT/references"
  git clone --quiet --filter=blob:none --no-checkout --single-branch --depth=1 \
    --branch "$SOURCE_REF" "$SOURCE_REPOSITORY" "$REFERENCE"
fi
if ! git -C "$REFERENCE" rev-parse --verify "$SOURCE_REF^{commit}" >/dev/null 2>&1 ||
  ! git -C "$REFERENCE" cat-file -e "$SOURCE_REVISION^{commit}" 2>/dev/null; then
  git -C "$REFERENCE" fetch --quiet --filter=blob:none --depth=1 origin "$SOURCE_REF"
fi
if [[ "$(git -C "$REFERENCE" rev-parse "$SOURCE_REF^{commit}")" != "$SOURCE_REVISION" ]]; then
  echo "AlphaTab source ref does not resolve to the pinned revision" >&2
  exit 1
fi

node "$CONFORMANCE/verify.mjs"
node "$CONFORMANCE/sync-upstream-inventory.mjs" --check
go test -count=1 -run '^TestSemanticContractInventory$' "$ROOT"
node "$CONFORMANCE/sensitivity.mjs"
ALPHATAB_CONFORMANCE=1 go test -count=1 -run '^TestAlphaTab' "$ROOT"
