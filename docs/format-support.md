# Guitar Pro semantic support

Generated from [`conformance/feature-ledger.json`](../conformance/feature-ledger.json). Do not edit this table by hand.

AlphaTab oracle: `@coderline/alphatab@1.8.4`, source `022a45c8e42370f9e12e68949d11eada370da83d`.

| Feature | Formats | Status | Issues | Reason |
| --- | --- | --- | --- | --- |
| `score-core` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12), [#13](https://github.com/CaliLuke/go-guitar-pro/issues/13) | The full corpus parses, while richer metadata and loss reporting remain tracked by issues 12 and 13. |
| `rhythm` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | Selected binary and GPIF rhythm fixtures have no unexplained AlphaTab differences. |
| `timing` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | Binary and GPIF absolute display starts agree after zero-origin normalization across measures, independent voices, tuplets, empty and populated pickups, and attached or orphan grace notes. |
| `staff-ownership` | gp6, gp7, gp8 | supported | none | GPIF staff order, measures, voices, notes, clefs, and tuning agree with the pinned AlphaTab grand-staff fixture; legacy track fields expose the first staff. |
| `grace-relationships` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | Selected authored ownership and source-fret cases agree after excluding display duration and playback pitch; complete cross-format grace coverage is tracked by issue 12. |
| `note-and-beat-semantics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | The canonical contract inventories these fields; a lossless semantic model is tracked by issue 12. |
| `tremolo-picking` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | Binary and GPIF tremolo-picking subdivisions are retained on notes; all six GPIF fixture beats agree with AlphaTab. |
| `harmonics` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | GPIF HFret values and harmonic kinds agree with AlphaTab on import and GP8 export, including fractional values and property-order variants. |
| `hairpins` | gp6, gp7, gp8 | supported | none | All eight GPIF hairpins and GP8 Decrescendo output agree with AlphaTab; legacy Diminuendo input remains accepted. |
| `tempo-automations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GPIF references 1–5 and absent or invalid defaults agree after BPM normalization; authored units, text, visibility, and interpolation remain part of issue 12. |
| `percussion-articulations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#10](https://github.com/CaliLuke/go-guitar-pro/issues/10) | Binary drum values are exposed, but GPIF articulation identity and per-track definitions are lost; see issue 10. |
