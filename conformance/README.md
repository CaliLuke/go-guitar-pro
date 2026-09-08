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

Run the gate with:

```sh
./conformance/check.sh
```

Refresh expected snapshots only after classifying every changed path in
`feature-ledger.json`:

```sh
ALPHATAB_CONFORMANCE=1 ALPHATAB_CONFORMANCE_UPDATE=1 go test -run '^TestAlphaTab' .
```

The contract records authored structure and semantics. AlphaTab grace beats are
normalized onto their owning note, and their raw source fret is compared with
Go's `GraceEffect.RawFret`. AlphaTab's fixed display duration and derived
playback pitch are not treated as authored GP3–5 grace values. Tick origins are
normalized to zero. Derived layout, rendering, and playback links are
inventoried but excluded from authored comparisons.
