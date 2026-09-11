# Ticket review handoff

Fresh agents reviewed all 46 open tickets. Forty tickets passed their bounded acceptance criteria. Six tickets remain unresolved. Every GitHub ticket stays open.

The verified source commit is `7789ca791d8d322ed23332876cc47ec324992995` on branch `codex/solve-outstanding-tickets`. Later commits contain review records, probe receipts, and generated bookkeeping.

## Verification

- `prek run --all-files` passed, including build, vet, formatting, lint, module checks, duplication, conformance, and race tests.
- All 81 sensitivity mutations were detected. Statement coverage was 92.5%, above the 80% requirement.
- The independent final probe checked 377 fixtures. It found no new parse, export, or consumer failures against either preceding receipt.
- The pinned oracle is AlphaTab 1.8.4 at revision `022a45c8e42370f9e12e68949d11eada370da83d`.

## Solved tickets

The following tickets have a fresh PASS and the `solved` label:

52, 55, 61, 62, 64, 65, 66, 69, 70, 72, 73, 84, 87, 88, 89, 90, 91, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 107, 108, 109, 110, 111, 112, 113, 114, 116, 117.

PASS applies to each ticket's recorded criteria and limits. It does not mean every format preserves every related feature.

## Unresolved tickets

| Ticket | Remaining failure or missing input |
| --- | --- |
| [40](https://github.com/CaliLuke/go-guitar-pro/issues/40) | The umbrella remains unresolved while its five dependent tickets remain unresolved. |
| [85](https://github.com/CaliLuke/go-guitar-pro/issues/85) | Program 73 moves from source MIDI tick 960 to target tick 0 on both channels. GPIF position remains exact; the consumer changes timing. |
| [86](https://github.com/CaliLuke/go-guitar-pro/issues/86) | The pinned consumer flattens different notation flags within one track to the first staff. |
| [92](https://github.com/CaliLuke/go-guitar-pro/issues/92) | Slashed beats work. Different slash flags within one track still encounter the staff limitation in ticket 86. |
| [106](https://github.com/CaliLuke/go-guitar-pro/issues/106) | Different numbered-notation flags within one track still encounter the staff limitation in ticket 86. |
| [115](https://github.com/CaliLuke/go-guitar-pro/issues/115) | The native asymmetric partial-capo fixture and a retaining independent consumer pitch receipt are missing. The saved zero-capo control does not satisfy either input. |

## Evidence and next review

`final-ticket-reviews.json` contains the seven independent review receipts, acceptance snapshots, tested commits, commands, and remaining failures. Each work item links this evidence.

`final-probe-summary.json` records the final corpus comparison. The full result is in `../probe-results.json`. Raw differences remain visible.

`final-review-probes/index.json` maps preserved review inputs to their original paths and hashes. Review probes are historical evidence. Some document pre-fix failures.

For a fresh review, check the current issue criteria and dependencies first. Read the matching work item and final review receipt. Run the focused tests and inspect the pinned consumer facts. Recheck the full gate before disposition if source changes.

The user requested that tickets remain open for another agent's verification. This pass did not close tickets or submit issue comments.
