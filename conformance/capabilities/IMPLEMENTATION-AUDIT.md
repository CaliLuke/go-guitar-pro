# Implementation audit — 2026-09-10

Audited library commit: `88356593a4ebb9b2f9859200a713838780d566b3`.
Oracle: AlphaTab 1.8.4, source revision `022a45c8e42370f9e12e68949d11eada370da83d`.

Reviewed the nine committed implementations for tickets #41–#49 against their latest acceptance criteria, source code, edit contracts, diagnostics, and independent consumer tests. The concurrent, uncommitted legato work for #50 was excluded. Repeat and simile regressions were included in the existing gate; this review does not re-audit every historical capability.

Eight slices have no blocker from this review. Ticket #47 requires a consumer-loss policy before acceptance. No issue was closed by this audit.

## Results

| Ticket | Decision | Checked scope |
| --- | --- | --- |
| [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41#issuecomment-5626717036) | No blocker found | Same-position linear/step tempo and sound events retain order, text, and supported visibility. Checked the opening-text edit contract, numeric validation, and the explicit sound-visibility consumer-loss report. Bank and preroll scope stays in #51/#64. |
| [#42](https://github.com/CaliLuke/go-guitar-pro/issues/42#issuecomment-5626717180) | No blocker found | Checked enabled Local audio metadata and exact asset bytes, independent consumer loading, signed-32-bit frame boundaries, missing/empty/reserved assets, and failure without overwriting an existing file. Playback and the separate archive interoperability backlog (#68) remain outside this slice. |
| [#43](https://github.com/CaliLuke/go-guitar-pro/issues/43#issuecomment-5626717324) | No blocker found | Checked independent staff capos 2/5 and consumer pitches 69/57, legacy-track versus staff edit authority, first-staff compatibility, and numeric boundary rejection. |
| [#44](https://github.com/CaliLuke/go-guitar-pro/issues/44#issuecomment-5626717468) | No blocker found | Checked all four clef octave shifts across staves, coexistence with a different beat octave, post-parse edits through the pinned consumer, and invalid source/public enum rejection. |
| [#45](https://github.com/CaliLuke/go-guitar-pro/issues/45#issuecomment-5626717598) | No blocker found | Checked all five targets and fourteen jumps, GP5 and GPIF import, simultaneous markers, deterministic deduplication, legacy-pointer versus collection edit/clear authority, and pinned-consumer output. Navigation traversal remains excluded. |
| [#46](https://github.com/CaliLuke/go-guitar-pro/issues/46#issuecomment-5626717746) | No blocker found | Compared the documentation changes with the active importer/writer and diagnostic inventory. Typed accent and vibrato authority, independent tapping flags, missing-payload diagnostics, HopoDestination loss, and beat-vibrato loss remain consistent. This documentation review does not extend any supported format/stage. |
| [#47](https://github.com/CaliLuke/go-guitar-pro/issues/47#issuecomment-5626717898) | Follow-up required | Exact XML and Go model preservation work, but distinct offsets can collapse to one hold in AlphaTab without a strict-export loss report. |
| [#48](https://github.com/CaliLuke/go-guitar-pro/issues/48#issuecomment-5626718065) | No blocker found | Checked GPIF presence semantics, retained numeric meter and exact timing across empty/short/long voices, flag edits, adjacent bars, and pinned-consumer output. No performance-timing claim is made. |
| [#49](https://github.com/CaliLuke/go-guitar-pro/issues/49#issuecomment-5626718217) | No blocker found | Checked Major/major/Minor/minor with nonzero accidentals and adjacent changes, post-parse edits, and unknown-mode diagnostics. Pinned-consumer checks pass. |

## Confirmed fermata consumer loss

Both valid inputs export under `RequirePreservation: true` with zero report entries:

| Authored score-tick offsets | GPIF quarter-note offsets | Pinned AlphaTab result |
| --- | --- | --- |
| 1/2 and 3/4 | 1/1920 and 1/1280 | One Long hold at tick 0 |
| 122 and 123 | 61/480 and 41/320 | One Long hold at tick 122 |

The first hold is Short with length 0.25. The second is Long with length 1.5. The consumer overwrites the first hold in both cases. Both generated ZIP files load directly; no repackaging was needed.

The writer preserves exact rational XML. Pinned `GpifParser._parseFermata` truncates `((numerator / denominator) * 960) | 0`. `MasterBar.addFermata` stores each result in a map keyed by that integer. Even an integral authored position can move: `(41 / 320) * 960` is just below 123 in JavaScript and truncates to 122.

Keep exact authored positions. Add a narrow preflight disposition for target movement and collisions, with locations and strict-policy refusal unless an exact retaining representation is proven. Test single moved offsets, both collision pairs, and ordinary retained positions. Do not change the model or normalize comparator results to conceal the consumer loss.

The existing sub-tick test only checks wire values and Go reimport. Existing AlphaTab assertions use other positions. This explains why the full gate passes despite the loss.

## Reproduction

Use a disposable checkout of the audited commit and install the pinned conformance dependencies. Copy [fermata-consumer-repro.go.txt](audit-repros/fermata-consumer-repro.go.txt) to the repository root as `progress_audit_test.go`, then run:

```sh
ALPHATAB_CONFORMANCE=1 go test -count=1 -v -run '^TestAudit(Fractional|Integer)FermataConsumer$' .
```

Both tests fail with `lost a fermata: got 1, want 2`. Remove the temporary root test after running it. The saved reproduction is opt-in evidence, not an intentionally failing addition to the normal test suite.

## Verification receipt

- `prek run --all-files`: PASS on the clean audited commit, before adding the temporary reproductions. This includes conformance, build, vet, formatting, lint, race tests, and the coverage gate.
- Existing focused tests with `ALPHATAB_CONFORMANCE=1`: PASS for automation semantics/validation, backing-track export, staff capo, clef octave, directions, fermatas, free time, key modes, and `TestAlphaTabPreserves*`.
- Additional fractional fermata collision reproduction: FAIL as expected; strict export returned 2,769 bytes with no diagnostics and AlphaTab retained one hold at 0.
- Additional whole-tick fermata collision reproduction: FAIL as expected; strict export returned 2,771 bytes with no diagnostics and AlphaTab retained one hold at 122.

These results apply to the exact audited commit. Later feature changes need their own acceptance review. The inventory records the eight scoped passes as audit evidence, without changing their workflow status. Fermata GP8 export is partial until the consumer-limit policy is resolved.


## Query audit decisions

```sql
SELECT w.id, w.issue_url,
       json_extract(e.value, '$.decision') AS decision,
       json_extract(e.value, '$.commit') AS audited_commit,
       json_extract(e.value, '$.reference') AS review_comment
FROM work_item w, json_each(w.evidence_json) e
WHERE json_extract(e.value, '$.kind') = 'implementation-audit'
ORDER BY w.id;
```
