# Repository Guidelines

## Project Structure

This repository contains one Go package. Root Go files contain the public types
and format readers. `parser.go` detects the file format. `file_api.go` provides
the file-based API.

The `gpif_*.go` files contain GPIF types, parsing, audits, and model conversion.
The `gp8_*.go` files contain the GP8 export pipeline and target conversion.

Tests that exercise only exported behavior live in `integration/` and import the
library as a user would. White-box tests that require parser or writer internals
stay beside the source in the `goguitarpro` package. Name test files after a
stable feature or responsibility. Do not use issue numbers or semantic-matrix
family numbers in filenames.

`integration/compatibility_test.go` runs the compatibility suite. Test files are
grouped by version in `testdata/gp3/` through `testdata/gp8/`.

Keep known unsupported fixtures in the corpus. If the parser does not support a
fixture, add its path to `knownUnsupportedFixtures`. When support is complete,
remove the fixture path from `knownUnsupportedFixtures`.

## Build and Test Commands

- `prek run --all-files` runs every quality gate and is the prescribed entry point.
- `prek install` installs the pre-commit and pre-push hooks for the checkout.
- `./check.sh --fix` applies goimports and golangci-lint fixes before running the full gate.
- `go build ./...` compiles the package.
- `go test ./...` runs all compatibility tests.
- `go test -run TestParseRejectsShortData` runs one test.
- `go vet ./...` finds common Go errors.
- `golangci-lint run` runs the configured static checks.
- `go fmt ./...` formats all Go source files.
- `go test -v -run '^TestSemanticMatrixInventory$' .` shows semantic matrix coverage.

The full gate covers build, vet, goimports, module-tidy drift, golangci-lint,
duplication, race-enabled tests, and an 80% statement-coverage floor. This
repository is a library, so reachability-based `deadcode` is intentionally not
used for exported APIs.

The repository contains a library. It does not contain a command-line program.

## Semantic Matrix Workflow

Add the case to `conformance/feature-ledger.json`. Record its formats, stages,
non-default values, oracle, and limits. Register one focused executor in
`conformance_harness_test.go`. Put the test in the applicable domain-named
`conformance_*_test.go` file. Use the assertion recorder for every exact field,
wire field, or dispatch that the case covers. Matrix IDs such as `M12` are
stable ledger identifiers, not source-file organization.

For each public field, use one disposition-specific runtime assertion. The
assertion disposition must match the complete field partition.

Treat each exported enum constant name as a separate obligation, including
constants typed by conversion, arithmetic, iota repetition, or a type alias.
Give a same-valued alias its own behavior evidence instead of assuming that its
numeric value inherits another member's contract.

Run the semantic matrix inventory. Check that the covered total increases by
the intended amount. Do not set the matrix `complete` flag until every uncovered
set is empty. Run `./conformance/check.sh` after each completed family.

## External References

Local source checkouts used for implementation reference belong under the ignored
`references/` directory. AlphaTab is expected at `references/alphaTab`. If that
checkout is missing, recreate it from the repository root with:

```sh
git clone --branch develop https://github.com/CoderLine/alphaTab.git references/alphaTab
```

Do not add files from `references/` to this repository.

## Code Style

Use standard Go formatting and naming. Use `PascalCase` for exported identifiers. Use `camelCase` for private identifiers. Add a documentation comment to each new exported API. Return errors with operation context. Do not write logs from library code. Do not terminate a process from library code.

Keep each format detail near its related reader. Do not add application, storage, or HTTP code to this module.

## Test Guidelines

Use the standard `testing` package. Name tests `TestXxx`. Use `t.Run` for fixture cases. Add the smallest representative Guitar Pro file for each regression. Make sure that malformed data returns an error. Make sure that malformed data never causes a panic.

## Semantic Change Workflow

Use this workflow for changes to a reader, the public model, finalization,
validation, or export.

1. Read `docs/semantic-model.md` and `conformance/README.md`.
2. Read the related AlphaTab importer and model code in `references/alphaTab`.
3. Check the revision in `conformance/oracle.json` before you compare behavior.
4. Trace the source value through decoding, the model, finalization, validation,
   and export.
5. Add an explicit disposition when a stage does not preserve the value.
6. Add a failing public-API test with a valid, non-default value.
7. Use the pinned AlphaTab consumer for an independent export assertion.
8. Update `conformance/feature-ledger.json` in the same change.
9. Run `./conformance/check.sh` and `prek run --all-files`.

Use one authority for a semantic value and its compatibility fields. Document
the reconciliation rule for edits after parsing. Check a numeric boundary before
a narrowing conversion. Do not silently wrap or clamp an authored value.

Create independent mutable data for each parsed occurrence. If definitions share
data, expose an immutable definition and an explicit reference.

A successful parse, statement coverage, a self-round-trip, or a new snapshot is
not sufficient evidence. Keep each unresolved difference narrow and link it to
an open issue. Do not hide a difference with broad normalization.

AlphaTab is a reference implementation, not the authority for Guitar Pro behavior.
When consumers disagree, use controlled Guitar Pro authoring, reopening, and
playback evidence to establish correctness. Preserve correct format behavior
even when the pinned AlphaTab version loses it. Record an AlphaTab defect
separately from a library defect or a target-format limit. Do not close a ticket
from AlphaTab agreement when native Guitar Pro evidence contradicts it.

## Commits and Pull Requests

Use a short imperative subject. The history uses prefixes such as `feat:`, `test:`, and `build:`. In each pull request, state the affected formats and the public API changes. Include the results of `go test ./...`, `go vet ./...`, and `golangci-lint run`.

Before you close a ticket, review the exact current commit against the latest
acceptance criteria and its open dependency tickets. Do not close a ticket from
example fixes or coverage counts alone. Require each blocker to have a
reproducible, scoped failure. Treat free-form review suggestions as new scope,
not as acceptance criteria. After you fix the blockers, run the same acceptance
checklist once more.
