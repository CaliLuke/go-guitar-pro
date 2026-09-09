# Guitar Pro semantic support

Generated from [`conformance/feature-ledger.json`](../conformance/feature-ledger.json). Do not edit these tables by hand.

AlphaTab oracle: `@coderline/alphatab@1.8.4`, source `022a45c8e42370f9e12e68949d11eada370da83d`.

| Feature | Formats | Status | Issues | Reason |
| --- | --- | --- | --- | --- |
| `score-core` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12), [#13](https://github.com/CaliLuke/go-guitar-pro/issues/13) | The full corpus parses, while richer metadata and loss reporting remain tracked by issues 12 and 13. |
| `rhythm` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12), [#15](https://github.com/CaliLuke/go-guitar-pro/issues/15) | Selected duration, dot, tuplet, time-signature, and rest cases are compared; exact score time and the remaining authored rhythm model are tracked by issues 12 and 15. |
| `timing` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#14](https://github.com/CaliLuke/go-guitar-pro/issues/14), [#15](https://github.com/CaliLuke/go-guitar-pro/issues/15) | Selected absolute display starts agree after zero-origin normalization; exact score time and shared finalized-graph validation are tracked by issues 14 and 15. |
| `staff-ownership` | gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | The grand-staff contract compares exact track, staff, bar, voice, beat, and note hierarchy plus authored contents; broader model ownership and legacy projection are tracked by issue 12. |
| `grace-relationships` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | Selected authored ownership and source-fret cases agree after excluding display duration and playback pitch; complete cross-format grace coverage is tracked by issue 12. |
| `note-and-beat-semantics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | The canonical contract inventories these fields; a lossless semantic model is tracked by issue 12. |
| `tremolo-picking` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | Binary and GPIF tremolo-picking subdivisions are retained on notes; all six GPIF fixture beats agree with AlphaTab. |
| `harmonics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GPIF HFret values and common harmonic kinds are compared on import and GP8 export; distinctions that the legacy model collapses are tracked by issue 12. |
| `hairpins` | gp6, gp7, gp8 | supported | none | All eight GPIF hairpins and GP8 Decrescendo output agree with AlphaTab; legacy Diminuendo input remains accepted. |
| `tempo-automations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GPIF references 1–5 and absent or invalid defaults agree after BPM normalization; authored units, text, visibility, and interpolation remain part of issue 12. |
| `percussion-articulations` | gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GP7–8 GPIF articulation identity, per-track definitions, notation metadata, input values, and output MIDI are compared; cross-format semantic integration is tracked by issue 12. |

## Parse diagnostics

The diagnostic disposition is separate from semantic feature support. Strict parsing rejects only dispositions marked Yes.

| Disposition | Strict rejection | Meaning |
| --- | --- | --- |
| `invalid-data` | yes | The source contains a malformed value, duplicate object ID, or broken object reference. |
| `unknown-syntax` | yes | The source contains syntax that the GPIF audit does not recognize. |
| `unsupported-feature` | yes | The parser recognizes the source content but cannot represent it in Song. |
| `lossy-projection` | yes | The parser maps the source content to a less precise Song value. |
| `deliberate-default` | no | The parser applies a documented default when the source omits a value. |
| `deliberate-ignore` | no | The parser intentionally excludes source metadata that is outside the Song contract. |

Each diagnostic receipt names one source construct. Its feature value uses an ID from the semantic support table.

| Source construct | Feature | Disposition | Reason |
| --- | --- | --- | --- |
| `GPIF.ChannelStrip.Automation.Type.Unknown` | `score-core` | `unknown-syntax` | The channel-strip automation type is not recognized. |
| `GPIF.ChannelStrip.Automation.Unsupported` | `score-core` | `unsupported-feature` | The channel-strip automation has no Song destination. |
| `GPIF.MasterTrack.Automation.Type.Unknown` | `score-core` | `unknown-syntax` | The master-track automation type is not recognized. |
| `GPIF.MasterTrack.Automation.Tempo.Invalid` | `tempo-automations` | `invalid-data` | The opening tempo must be finite and positive. |
| `GPIF.MasterTrack.Automation.Tempo.LegacyOverflow` | `tempo-automations` | `lossy-projection` | The exact opening BPM is preserved but cannot be projected into the legacy int16 field. |
| `GPIF.MasterTrack.Automation.Tempo.Reference.Invalid` | `tempo-automations` | `invalid-data` | Applying the authored tempo reference unit must produce a finite positive BPM. |
| `GPIF.Note.Accent.Tenuto` | `note-and-beat-semantics` | `unsupported-feature` | Song has no destination for the GPIF tenuto accent bit. |
| `GPIF.Note.Property.ConcertPitch.Redundant` | `note-and-beat-semantics` | `deliberate-ignore` | The pitch agrees with the mapped absolute MIDI value. |
| `GPIF.Note.Property.TransposedPitch.Redundant` | `note-and-beat-semantics` | `deliberate-ignore` | The pitch agrees with the mapped absolute MIDI value. |
| `GPIF.Chord.Diagram.Fingering` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.ShowDiagram` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.ShowFingering` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.ShowName` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Staff.Property.CapoFret` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Staff.Property.Tuning.Label` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Staff.Property.Tuning.MissingPitches` | `staff-ownership` | `invalid-data` | The recognized source construct is missing its required payload. |
| `GPIF.Staff.Property.Unknown` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Track.Property.CapoFret` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.Property.Tuning.Label` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.Property.Tuning.MissingPitches` | `staff-ownership` | `invalid-data` | The recognized source construct is missing its required payload. |
| `GPIF.Track.Property.Unknown` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.ScoreCore` | `score-core` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.ScoreCore` | `score-core` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `Binary.Note.TimeIndependentDuration` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `Binary.RSE.MasterMetadata` | `score-core` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `Binary.RSE.TrackMetadata` | `score-core` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Asset.DuplicateID` | `score-core` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Asset.EmptyID` | `score-core` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.BackingTrack.AssetId.Reference` | `score-core` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Bar.DuplicateID` | `staff-ownership` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Bar.EmptyID` | `staff-ownership` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Bar.Voices.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Arpeggio.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Chord.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Dynamic.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Beat.Fadding.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Fadding.Lossy` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Beat.GraceNotes.InvalidValue` | `grace-relationships` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Hairpin.InvalidValue` | `hairpins` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Notes.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Ottavia.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.BarreFret` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.BarreString` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Brush.InvalidDirection` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Brush.MissingDirection` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.PickStroke.InvalidDirection` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.PickStroke.MissingDirection` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.Popped.MissingEnable` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.PrimaryPickupTone` | `note-and-beat-semantics` | `deliberate-ignore` | The notation-focused Song model intentionally excludes this playback metadata. |
| `GPIF.Beat.Property.PrimaryPickupVolume` | `note-and-beat-semantics` | `deliberate-ignore` | The notation-focused Song model intentionally excludes this playback metadata. |
| `GPIF.Beat.Property.Rasgueado` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Slapped.MissingEnable` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Beat.Property.VibratoWTremBar` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Beat.Property.WhammyBar` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarDestinationOffset` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarDestinationValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarExtend` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarMiddleOffset1` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarMiddleOffset2` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarMiddleValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarOriginOffset` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarOriginValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Rhythm.Reference` | `rhythm` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Tremolo.InvalidValue` | `tremolo-picking` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Wah` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.ChordDefinition.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.ChordDefinition.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.MasterBar.Bars.Reference` | `staff-ownership` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.MasterBar.TripletFeel.InvalidValue` | `rhythm` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.MasterTrack.Tracks.Reference` | `staff-ownership` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Note.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Note.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Note.Property.BendDestinationOffset.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendDestinationValue.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendMiddleOffset1.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendMiddleOffset2.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendMiddleValue.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendOriginOffset.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendOriginValue.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.ConcertPitch` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Element` | `percussion-articulations` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Fret.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicFret.MissingPayload` | `harmonics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicType.Feedback` | `harmonics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.HarmonicType.MissingPayload` | `harmonics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicType.Unsupported` | `harmonics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.HopoDestination` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.HopoOrigin` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.LeftHandTapped` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.Midi.MissingPayload` | `percussion-articulations` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.Octave` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Slide.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.String.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.Tapped` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.Tone` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.TransposedPitch` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Note.Property.Variation` | `percussion-articulations` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Vibrato` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Rhythm.DuplicateID` | `rhythm` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Rhythm.EmptyID` | `rhythm` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Rhythm.NoteValue.InvalidValue` | `rhythm` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.DuplicateID` | `staff-ownership` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Track.EmptyID` | `staff-ownership` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Track.Automation.Sound.Reference` | `score-core` | `invalid-data` | The sound automation must reference a sound in its track. |
| `GPIF.Track.Automation.SustainPedal` | `score-core` | `unsupported-feature` | Song has no destination for sustain-pedal automation. |
| `GPIF.Track.Automation.Type.Unknown` | `score-core` | `unknown-syntax` | The track automation type is not recognized. |
| `GPIF.Track.Transpose` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.UnknownAttribute.NoteAndBeat` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.Rhythm` | `rhythm` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.StaffOwnership` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.NoteAndBeat` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.Rhythm` | `rhythm` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.StaffOwnership` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Voice.Beats.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Voice.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Voice.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |

## Public model inventory

The inventory starts at `Song`. Unlisted roles are authored values. Compatibility and derived fields have explicit roles.

| Type | Feature | Field roles | Reason |
| --- | --- | --- | --- |
| `Song` | `score-core` | 31 authored, 1 compatibility, 0 derived, 0 out-of-scope | The root contains authored score data. Tempo is the legacy view of InitialTempo. |
| `Version` | `score-core` | 2 authored, 0 compatibility, 1 derived, 0 out-of-scope | The source version is preserved. Number is parsed from Data. |
| `Clipboard` | `score-core` | 7 authored, 0 compatibility, 0 derived, 0 out-of-scope | The binary clipboard range is authored source data. |
| `BackingTrack` | `score-core` | 9 authored, 0 compatibility, 0 derived, 0 out-of-scope | The backing-track record and embedded asset are authored source data. |
| `SyncPoint` | `timing` | 6 authored, 2 compatibility, 3 derived, 0 out-of-scope | The sync point preserves source values and checked timing views. |
| `VolumeAutomation` | `score-core` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The volume point is authored playback data. |
| `TempoAutomation` | `tempo-automations` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The tempo point is authored timing data. |
| `Lyrics` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The score lyrics are authored text data. |
| `LyricLine` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The lyric line is authored text data. |
| `PageSetup` | `score-core` | 17 authored, 0 compatibility, 0 derived, 0 out-of-scope | The page setup is authored display data. |
| `RseMasterEffect` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The master RSE effect is authored playback data. |
| `MidiChannel` | `score-core` | 10 authored, 0 compatibility, 0 derived, 0 out-of-scope | The MIDI channel contains authored playback values. |
| `MeasureHeader` | `rhythm` | 11 authored, 0 compatibility, 2 derived, 0 out-of-scope | The header contains authored bar data and derived absolute starts. |
| `Marker` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The marker is authored score data. |
| `SourceValue` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The wrapper preserves source presence and unknown values. |
| `Track` | `staff-ownership` | 22 authored, 2 compatibility, 0 derived, 0 out-of-scope | The track owns staves. Measures and Strings are first-staff compatibility views. |
| `Staff` | `staff-ownership` | 3 authored, 0 compatibility, 1 derived, 0 out-of-scope | The staff owns measures and tuning. PercussionTrack mirrors its track. |
| `TrackSettings` | `score-core` | 11 authored, 0 compatibility, 0 derived, 0 out-of-scope | The track settings are authored display data. |
| `PercussionArticulation` | `percussion-articulations` | 13 authored, 0 compatibility, 0 derived, 0 out-of-scope | The articulation preserves track-local notation and playback identity. |
| `TrackSound` | `score-core` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The sound definition is authored playback data. |
| `SoundAutomation` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The sound automation is authored playback data. |
| `TrackLyricLine` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The track lyric line is authored text data. |
| `TrackRse` | `score-core` | 4 authored, 0 compatibility, 0 derived, 0 out-of-scope | The track RSE record is authored playback data. |
| `RseEqualizer` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The equalizer contains authored playback values. |
| `RseInstrument` | `score-core` | 6 authored, 0 compatibility, 0 derived, 0 out-of-scope | The RSE instrument contains authored playback values. |
| `GuitarString` | `staff-ownership` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The string contains authored tuning data. |
| `Measure` | `timing` | 8 authored, 0 compatibility, 4 derived, 0 out-of-scope | The measure contains authored notation and finalized ownership and timing. |
| `Voice` | `note-and-beat-semantics` | 2 authored, 0 compatibility, 1 derived, 0 out-of-scope | The voice owns authored beats. MeasureIndex is finalized ownership data. |
| `Beat` | `note-and-beat-semantics` | 8 authored, 0 compatibility, 2 derived, 0 out-of-scope | The beat contains authored values and finalized starts. |
| `BeatDisplay` | `note-and-beat-semantics` | 7 authored, 0 compatibility, 0 derived, 0 out-of-scope | The beat display record is authored notation data. |
| `BeatEffects` | `note-and-beat-semantics` | 10 authored, 0 compatibility, 0 derived, 0 out-of-scope | The beat effect record contains authored notation and playback effects. |
| `BeatStroke` | `note-and-beat-semantics` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The stroke contains authored direction and duration. |
| `Note` | `note-and-beat-semantics` | 10 authored, 0 compatibility, 0 derived, 0 out-of-scope | The note contains authored pitch, articulation, duration, and effect values. |
| `NoteEffect` | `note-and-beat-semantics` | 17 authored, 0 compatibility, 0 derived, 0 out-of-scope | The note effect record contains authored note techniques. |
| `BendEffect` | `note-and-beat-semantics` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The bend effect contains authored bend data. |
| `BendPoint` | `note-and-beat-semantics` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The bend point contains authored curve data. |
| `GraceEffect` | `grace-relationships` | 10 authored, 1 compatibility, 0 derived, 0 out-of-scope | The grace effect contains authored occurrence data. Fret is a legacy view of ExactFret. |
| `HarmonicEffect` | `harmonics` | 4 authored, 1 compatibility, 0 derived, 0 out-of-scope | The harmonic effect preserves authored kind and pitch data. Fret is a legacy view. |
| `TremoloPickingEffect` | `tremolo-picking` | 1 authored, 0 compatibility, 0 derived, 0 out-of-scope | The effect preserves the authored tremolo subdivision. |
| `TrillEffect` | `note-and-beat-semantics` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The trill effect contains authored fret and duration data. |
| `MixTableChange` | `score-core` | 13 authored, 0 compatibility, 0 derived, 0 out-of-scope | The mix-table change contains authored playback changes. |
| `MixTableItem` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The mix-table item contains an authored value and transition duration. |
| `WahEffect` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The wah effect contains authored playback values. |
| `Duration` | `rhythm` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The duration preserves the authored base value, dots, and tuplet ratio. |
| `TimeSignature` | `rhythm` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The time signature contains authored meter values. |
| `KeySignature` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The key signature contains authored tonality values. |
| `ScoreTime` | `timing` | 0 authored, 0 compatibility, 0 derived, 0 out-of-scope | The private reduced terms provide a checked derived timing value. |
| `BarPosition` | `timing` | 0 authored, 0 compatibility, 0 derived, 0 out-of-scope | The private reduced terms provide a checked authored bar position. |
| `Chord` | `note-and-beat-semantics` | 19 authored, 0 compatibility, 0 derived, 0 out-of-scope | The chord contains authored identity, pitch, fingering, and diagram data. |
| `PitchClass` | `note-and-beat-semantics` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The pitch class contains authored spelling data. |
| `Barre` | `note-and-beat-semantics` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The barre contains authored chord fingering data. |

## Source dispatch inventory

The gate compares these cases with the source switches. Each default has an explicit disposition.

| Dispatch | Feature | Cases | Default | Reason |
| --- | --- | --- | --- | --- |
| `gpifAuditOwnedStaffProperty:property.Name` | `staff-ownership` | 4 | `unknown-syntax` | The audit classifies each track and staff property before import. |
| `gpifAuditNoteProperty:property.Name` | `note-and-beat-semantics` | 28 | `unknown-syntax` | The audit classifies each named note property before import. |
| `gpifAuditBeatProperty:property.Name` | `note-and-beat-semantics` | 19 | `unknown-syntax` | The audit classifies each named beat property before import. |
| `gpifApplyBeatEffects:p.Name` | `note-and-beat-semantics` | 5 | `delegated-to-audit` | The importer maps represented beat properties after the audit classifies all names. |
| `gpifNoteToNote:p.Name` | `note-and-beat-semantics` | 19 | `delegated-to-audit` | The importer maps represented note properties after the audit classifies all names. |
| `gpifAuditMasterAutomations:automation.Type` | `score-core` | 2 | `unknown-syntax` | The audit classifies each master-track automation before import. |
| `gpifAuditTrackAutomations:automation.Type` | `score-core` | 2 | `unknown-syntax` | The audit classifies each track automation and checks sound references before import. |
| `gpifAuditChannelStripAutomations:automation.Type` | `score-core` | 4 | `unknown-syntax` | The audit preserves volume automation and reports every other recognized channel-strip automation. |
