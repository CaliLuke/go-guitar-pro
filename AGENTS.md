# Repository Guidelines

## Project Structure

This repository is one Go package. The root files contain the public data types and format readers. `parser.go` detects the file format. `goguitarpro.go` provides the file-based API. Format-specific logic uses focused files such as `gpif.go`, `gpx_filesystem.go`, and `gp7_zip.go`.

`goguitarpro_test.go` runs the compatibility suite. Test files are grouped by version in `testdata/gp3/` through `testdata/gp8/`. Keep known unsupported fixtures in the corpus. If the parser does not support a fixture, add its path to `knownUnsupportedFixtures`. When the parser supports the fixture, remove its path.

## Build and Test Commands

- `go build ./...` compiles the package.
- `go test ./...` runs all compatibility tests.
- `go test -run TestParseRejectsShortData` runs one test.
- `go vet ./...` finds common Go errors.
- `golangci-lint run` runs the configured static checks.
- `go fmt ./...` formats all Go source files.

The repository contains a library. It does not contain a command-line program.

## Code Style

Use standard Go formatting and naming. Use `PascalCase` for exported identifiers. Use `camelCase` for private identifiers. Add a documentation comment to each new exported API. Return errors with operation context. Do not write logs from library code. Do not terminate a process from library code.

Keep each format detail near its related reader. Do not add application, storage, or HTTP code to this module.

## Test Guidelines

Use the standard `testing` package. Name tests `TestXxx`. Use `t.Run` for fixture cases. Add the smallest representative Guitar Pro file for each regression. Make sure that malformed data returns an error. Make sure that malformed data never causes a panic.

## Commits and Pull Requests

Use a short imperative subject. The history uses prefixes such as `feat:`, `test:`, and `build:`. In each pull request, state the affected formats and the public API changes. Include the results of `go test ./...`, `go vet ./...`, and `golangci-lint run`.
