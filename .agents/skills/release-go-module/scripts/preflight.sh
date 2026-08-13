#!/usr/bin/env bash

set -euo pipefail

version=${1:-}
expected_module=github.com/CaliLuke/go-guitar-pro

if [[ ! $version =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH" >&2
  exit 2
fi

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [[ $(git branch --show-current) != main ]]; then
  echo "release branch must be main" >&2
  exit 1
fi

if [[ -n $(git status --porcelain) ]]; then
  echo "working tree must be clean" >&2
  git status --short >&2
  exit 1
fi

module_path=$(go list -m -f '{{.Path}}')
if [[ $module_path != "$expected_module" ]]; then
  echo "module path is $module_path, want $expected_module" >&2
  exit 1
fi

if git show-ref --verify --quiet "refs/tags/$version"; then
  echo "tag $version already exists" >&2
  exit 1
fi

for required_file in LICENSE README.md THIRD_PARTY_NOTICES.md go.mod go.sum; do
  if [[ ! -f $required_file ]]; then
    echo "required file is missing: $required_file" >&2
    exit 1
  fi
done

unformatted=$(gofmt -l .)
if [[ -n $unformatted ]]; then
  echo "Go files need formatting:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go mod tidy
if [[ -n $(git status --porcelain) ]]; then
  echo "go mod tidy changed the working tree; inspect and commit the change" >&2
  git status --short >&2
  exit 1
fi

go test ./...
go vet ./...
golangci-lint run

echo "preflight passed for $version at $(git rev-parse HEAD)"
