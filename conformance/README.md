# AlphaTab semantic conformance

This directory contains a development-only differential oracle. It compares a
normalized semantic score from go-guitar-pro with the same source imported by a
pinned AlphaTab package, and separately compares source semantics with a Go GP8
export imported by AlphaTab. No Node dependency is linked into the production Go
package.

The oracle is pinned in `oracle.json` and `package-lock.json`. Upgrading it is a
reviewed change: update both pins, regenerate the upstream inventory, inspect all
semantic differences, and then refresh snapshots explicitly. Default importer
settings are repeated in `oracle.json` so a package-default change cannot be
accepted silently.

The gate creates the ignored `references/alphaTab` checkout at the pinned tag
when it is missing, then verifies the exact revision and all inventoried source
hashes. A clean checkout therefore performs the same source-drift checks as a
developer checkout.

With the pinned source checked out at `references/alphaTab`, refresh upstream
source hashes, importer test invocations and fixture references, and discovered
model symbols with:

```sh
node conformance/sync-upstream-inventory.mjs
```

New symbols are emitted as `unclassified`, which fails the gate until each one is
given an explicit disposition, feature, and reason. The discovery set covers the
score, master-bar, track, staff, bar, voice, beat, note, and automation models
used by the canonical contract. There is no automatic classifier or acceptance
mode for oracle upgrades.

The semantic contract starts at the public `Song` type. The gate discovers each
reachable public structure and field. It compares that set with the model
inventory in `feature-ledger.json`. The gate also inventories named GPIF source
dispatch cases. A new field or dispatch case fails until its contract is added.
Each public field has one target disposition. Each source case has one source
disposition and one behavioral evidence link. Automation cases also identify
the consumer dispatch or diagnostic that handles the case.

The same ledger contains the M01 through M25 semantic matrix. Each matrix case
names its evidence role, typed evidence sources, formats, stages, value shapes,
oracle, and limits. A registered test executes the case and records each exact
field, wire field, dispatch, or public enum member that it checks.
The inventory rejects a case assignment when the matching assertion does not
run. One runtime assertion for each public field also records its target
disposition. The inventory rejects a conflict between this declaration and the
ledger. A new public field fails until both records exist. This check does not
replace stage-specific behavior evidence. Run the progress report with:

```sh
go test -v -run '^TestSemanticMatrixInventory$' .
```

The `complete` flag stays false while the report has uncovered constructs or
behavioral obligations. Set it to true only when all uncovered sets are empty.
An XML schema round trip has a structural role. It cannot satisfy a semantic
leaf, dispatch, public field, or public enum obligation. The inventory also
resolves exported constant types across the complete Go package. Explicit
types, conversions, typed arithmetic, iota repetition, and type aliases are
covered. Each exported constant name, including a same-valued alias, fails
until a focused case executes its behavior.

Ownership cases compare ordered track, staff, bar, voice, beat, and note paths.
They do not use aggregate counts as semantic evidence. The compatibility case
also checks the same staff 0 authority rule through finalization, validation,
and export.

The ledger also names focused public API tests for each represented feature.
Each feature links to a pinned AlphaTab test. This gives the contract an
independent consumer. The verifier fails if a named test is removed or if a
feature loses either form of evidence.

`sensitivity.mjs` applies a fixed set of valid defects through Go overlays. Each
defect must make its focused test fail for the expected reason. The set covers
model, wire, dispatch, and enum inventory; diagnostics; serialization;
independent adapters; strict policy; ownership; and checked numeric narrowing.
Mutants cover distinct value classes, not only one named example. This is
selective mutation testing. It measures whether the tests detect realistic
faults. Statement coverage measures execution breadth. Use both signals.
Neither signal proves that the code has no defects.

Run the gate with:

```sh
./conformance/check.sh
```

Run only the mutation checks with:

```sh
node conformance/sensitivity.mjs
```

Refresh expected snapshots only after classifying every changed path in
`feature-ledger.json`:

```sh
ALPHATAB_CONFORMANCE=1 ALPHATAB_CONFORMANCE_UPDATE=1 go test -run '^TestAlphaTab' .
```

Refresh the whole-corpus receipt with:

```sh
ALPHATAB_CONFORMANCE=1 ALPHATAB_CORPUS_UPDATE=1 go test -run '^TestSemanticMatrixM23WholeCorpusAccounting$' .
```

This command retains classifications only for unchanged fixture, difference,
and semantic paths. The strict run fails until each new or changed difference
has a narrow reason and an open issue.

The contract records authored structure and semantics. AlphaTab grace beats are
normalized onto their owning note, and their raw source fret is compared with
Go's `GraceEffect.RawFret`. AlphaTab's fixed display duration and derived
playback pitch are not treated as authored GP3–5 grace values. Tick origins are
normalized to zero. Derived layout, rendering, and playback links are
inventoried but excluded from authored comparisons.
