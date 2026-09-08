# Guitar Pro semantic support

Generated from [`conformance/feature-ledger.json`](../conformance/feature-ledger.json). Do not edit this table by hand.

AlphaTab oracle: `@coderline/alphatab@1.8.4`, source `022a45c8e42370f9e12e68949d11eada370da83d`.

| Feature | Formats | Status | Issues | Reason |
| --- | --- | --- | --- | --- |
| `score-core` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12), [#13](https://github.com/CaliLuke/go-guitar-pro/issues/13) | The full corpus parses, while richer metadata and loss reporting remain tracked by issues 12 and 13. |
| `rhythm` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | Selected binary and GPIF rhythm fixtures have no unexplained AlphaTab differences. |
| `timing` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#9](https://github.com/CaliLuke/go-guitar-pro/issues/9) | Binary timing is populated, but GPIF measure and beat timing is missing; see issue 9. |
| `staff-ownership` | gp6, gp7, gp8 | partial | [#4](https://github.com/CaliLuke/go-guitar-pro/issues/4) | The second staff and its 18 notes are missing from the Go model; see issue 4. |
| `grace-relationships` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | Selected authored ownership and source-fret cases agree after excluding display duration and playback pitch; complete cross-format grace coverage is tracked by issue 12. |
| `note-and-beat-semantics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | The canonical contract inventories these fields; a lossless semantic model is tracked by issue 12. |
| `tremolo-picking` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#5](https://github.com/CaliLuke/go-guitar-pro/issues/5) | All six GPIF tremolo-picking values are discarded; see issue 5. |
| `harmonics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#6](https://github.com/CaliLuke/go-guitar-pro/issues/6) | GPIF HFret values are missing on import and Go exports the nonstandard Float child; see issue 6. |
| `hairpins` | gp6, gp7, gp8 | partial | [#7](https://github.com/CaliLuke/go-guitar-pro/issues/7) | Three Decrescendo beats are lost and GP8 output uses Diminuendo; see issue 7. |
| `tempo-automations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#8](https://github.com/CaliLuke/go-guitar-pro/issues/8) | Quarter-note reference fixtures agree, but non-quarter reference units are ignored; see issue 8. |
| `percussion-articulations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#10](https://github.com/CaliLuke/go-guitar-pro/issues/10) | Binary drum values are exposed, but GPIF articulation identity and per-track definitions are lost; see issue 10. |
