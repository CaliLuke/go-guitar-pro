# Guitar Pro semantic support

Generated from [`conformance/feature-ledger.json`](../conformance/feature-ledger.json). Do not edit these tables by hand.

AlphaTab oracle: `@coderline/alphatab@1.8.4`, source `022a45c8e42370f9e12e68949d11eada370da83d`.

| Feature | Formats | Status | Issues | Reason |
| --- | --- | --- | --- | --- |
| `score-core` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix accounts for the public score model, fixtures, validation, and explicit GP8 conversion limits; audited upstream-only model surface is recorded by issue 34. |
| `midi-bank` | gp5, gp6, gp7, gp8 | supported | none | Legacy track banks and GPIF sound-bank definitions survive import, public editing, GP8 export, and pinned AlphaTab consumption with exact event order. |
| `sustain-pedal` | gp6, gp7, gp8 | supported | none | GPIF sustain-pedal down and release automations survive in exact source order on staff 0. Continuing bars without an explicit marker derive one position-0 hold, and GP8 writes only down and release records for pinned AlphaTab to reconstruct. |
| `rhythm` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix covers public duration, exact timing, meter, tuplets, rests, and binary discriminants; audited upstream-only model surface is recorded by issue 34. |
| `timing` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix covers public exact time and finalized graph validation, with no corpus timing differences; audited upstream-only model surface is recorded by issue 34. |
| `staff-ownership` | gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix compares public ownership paths, compatibility authority, and multi-staff export behavior; audited upstream-only model surface is recorded by issue 34. |
| `transposition` | gp6, gp7, gp8 | partial | [#56](https://github.com/CaliLuke/go-guitar-pro/issues/56) | Display transposition survives import and first-staff GP8 export. PartSounding nominal keys independently derive effective keys, including Neutral as zero. GP8 lacks independent effective keys, sounding transposition, and per-staff display offsets, and consumers reset percussion display offsets; each limit is reported. |
| `clef-octave` | gp6, gp7, gp8 | supported | none | Every GPIF bar-level clef octave survives import, post-parse editing, GP8 export, and pinned AlphaTab consumption independently from Beat.Octave. |
| `legato-slurs` | gp6, gp7, gp8 | partial | [#50](https://github.com/CaliLuke/go-guitar-pro/issues/50) | GPIF and Go preserve authored legato flags. Export reports destinations that differ from pinned AlphaTab derivation. Separate note-level slurs remain outside this contract. |
| `fermata` | gp6, gp7, gp8 | partial | [#47](https://github.com/CaliLuke/go-guitar-pro/issues/47) | GPIF and Go preserve exact fermata offsets, types, and lengths. Export reports pinned consumer offset movement and collisions; ordinary retained positions remain supported. |
| `free-time` | gp6, gp7, gp8 | supported | none | The authored GPIF FreeTime marker survives exact master-bar import, post-parse editing, GP8 export, and pinned AlphaTab consumption without replacing numeric meter timing. |
| `grace-relationships` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix covers authored grace order, ownership, source fret, transitions, orphan graces, and export policy; audited upstream-only model surface is recorded by issue 34. |
| `note-and-beat-semantics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix inventories and tests public note and beat fields, effects, strict export decisions, and independent consumption; audited upstream-only model surface is recorded by issue 34. |
| `tremolo-picking` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#57](https://github.com/CaliLuke/go-guitar-pro/issues/57) | Binary and GPIF marks 1 through 3 are retained on the beat and export as exact GPIF subdivisions. The legacy note field remains a documented fallback. GP3 has no source record, and AlphaTab's model-only marks 0, 4, and 5 plus BuzzRoll are not representable in Guitar Pro GPIF. |
| `harmonics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | GPIF and binary harmonic kinds and fret values are compared on import and GP8 export, including feedback harmonics; audited upstream-only model surface is recorded by issue 34. |
| `hairpins` | gp6, gp7, gp8 | supported | none | All eight GPIF hairpins and GP8 Decrescendo output agree with AlphaTab; legacy Diminuendo input remains accepted. |
| `tempo-automations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The public automation record and GP8 writer preserve ordered tempo values, interpolation, text, and per-event visibility; the remaining partial occurrence contract is tracked separately. |
| `percussion-articulations` | gp7, gp8 | partial | [#34](https://github.com/CaliLuke/go-guitar-pro/issues/34) | The matrix covers public articulation identity, every resolved staff, notation, playback, validation, and export policy; audited upstream-only model surface is recorded by issue 34. |
| `brush` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | [#78](https://github.com/CaliLuke/go-guitar-pro/issues/78) | M10-BRUSH preserves binary and GPIF kind, direction, exact authored timing, source absence, compatibility edits, and supported GP8 wire values with scoped diagnostics for malformed or inexpressible values. |
| `beat-lyrics` | gp6, gp7, gp8 | supported | [#76](https://github.com/CaliLuke/go-guitar-pro/issues/76) | Beat.Lyrics preserves GPIF beat-scoped lines as independent ordered occurrence data through public edits and GP8 export, without merging them into beat text or score and track lyrics. |
| `chord-diagram` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#79](https://github.com/CaliLuke/go-guitar-pro/issues/79) | GPIF and Go preserve chord ranges and finger assignments. Export reports annular barres ignored by pinned AlphaTab. Legacy chord description and interval-omission fields remain separate losses. |
| `beat-vibrato` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | [#77](https://github.com/CaliLuke/go-guitar-pro/issues/77) | Beat vibrato strength survives GP3-8 import, deterministic compatibility reconciliation, GP8 export, and pinned AlphaTab consumption without sharing authority with note vibrato. |
| `golpe` | gp6, gp7, gp8 | supported | [#83](https://github.com/CaliLuke/go-guitar-pro/issues/83) | BeatEffects.Golpe preserves GPIF Thumb and Finger values as independent authored beat marks through direct edits, exact GP8 output, and pinned AlphaTab consumption. |
| `beaming` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | [#75](https://github.com/CaliLuke/go-guitar-pro/issues/75) | Canonical custom groups and beat-level beam and stem overrides survive GP5 or GPIF import and GP8 export. Legacy Beat.Display raw fields retain individual explicit target omissions. |

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
| `GPIF.ChannelStrip.Automation.Pan.Value.Invalid` | `score-core` | `invalid-data` | The pan automation value must be numeric. |
| `GPIF.ChannelStrip.Automation.Type.Unknown` | `score-core` | `unknown-syntax` | The channel-strip automation type is not recognized. |
| `GPIF.ChannelStrip.Automation.Volume.Range.Invalid` | `score-core` | `invalid-data` | The volume automation position and value must be within 0..1. |
| `GPIF.ChannelStrip.Automation.Volume.Value.Invalid` | `score-core` | `invalid-data` | The volume automation value must be finite. |
| `GPIF.ChannelStrip.Automation.Unsupported` | `score-core` | `unsupported-feature` | The channel-strip automation has no Song destination. |
| `GPIF.MasterTrack.Automation.Type.Unknown` | `score-core` | `unknown-syntax` | The master-track automation type is not recognized. |
| `GPIF.MasterTrack.Automation.Tempo.Invalid` | `tempo-automations` | `invalid-data` | The opening tempo must be finite and positive. |
| `GPIF.MasterTrack.Automation.Tempo.LegacyOverflow` | `tempo-automations` | `lossy-projection` | The exact opening BPM is preserved but cannot be projected into the legacy int16 field. |
| `GPIF.MasterTrack.Automation.Tempo.Reference.Invalid` | `tempo-automations` | `invalid-data` | Applying the authored tempo reference unit must produce a finite positive BPM. |
| `GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid` | `score-core` | `invalid-data` | A sync point must contain a valid non-negative frame offset and score location. |
| `GPIF.Note.Property.Muted.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The muted property must contain its Enable payload. |
| `GPIF.Note.Property.PalmMuted.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The palm-muted property must contain its Enable payload. |
| `GPIF.Note.Property.Tapped.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The tapped property must contain its Enable payload. |
| `GPIF.Note.Property.HopoOrigin.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The hammer origin property must contain its Enable payload. |
| `GPIF.Note.Property.HopoDestination.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The hammer destination property must contain its Enable payload. |
| `GPIF.Note.Property.LeftHandTapped.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The left-hand-tapped property must contain its Enable payload. |
| `GPIF.Note.Property.Slide.InvalidFlags` | `note-and-beat-semantics` | `invalid-data` | Slide flags must be a non-negative integer. |
| `GPIF.Note.Property.Slide.UnknownFlags` | `note-and-beat-semantics` | `unsupported-feature` | The slide flag contains a bit that the public enum does not define. |
| `GPIF.Note.Property.BendNumber.Invalid` | `note-and-beat-semantics` | `invalid-data` | A bend number must be finite and inside the public curve range. |
| `GPIF.Note.Property.BendNumber.Quantized` | `note-and-beat-semantics` | `lossy-projection` | Note bend heights still use a narrower integer semitone scale; exact note offsets are preserved separately. |
| `GPIF.Beat.Whammy.Invalid` | `note-and-beat-semantics` | `invalid-data` | A whammy number must be finite and inside the public curve range. |
| `GPIF.Beat.Whammy.Quantized` | `note-and-beat-semantics` | `lossy-projection` | The public whammy curve uses a narrower integer scale. |
| `GPIF.Note.Property.ConcertPitch.Redundant` | `note-and-beat-semantics` | `deliberate-ignore` | The pitch agrees with the mapped absolute MIDI value. |
| `GPIF.Note.Property.TransposedPitch.Redundant` | `note-and-beat-semantics` | `deliberate-ignore` | The pitch agrees with the mapped absolute MIDI value. |
| `GPIF.Chord.Diagram.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Staff.Property.CapoFret.MissingFret` | `staff-ownership` | `invalid-data` | A capo property needs an explicit fret value. |
| `GPIF.Staff.Property.CapoFret.Negative` | `staff-ownership` | `invalid-data` | A capo fret cannot be negative. |
| `GPIF.Staff.Property.Tuning.MissingPitches` | `staff-ownership` | `invalid-data` | The recognized source construct is missing its required payload. |
| `GPIF.Staff.Property.Unknown` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Track.Property.CapoFret.MissingFret` | `staff-ownership` | `invalid-data` | A capo property needs an explicit fret value. |
| `GPIF.Track.Property.CapoFret.Negative` | `staff-ownership` | `invalid-data` | A capo fret cannot be negative. |
| `GPIF.Track.Property.Tuning.MissingPitches` | `staff-ownership` | `invalid-data` | The recognized source construct is missing its required payload. |
| `GPIF.Track.Property.Unknown` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.ScoreCore` | `score-core` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.ScoreCore` | `score-core` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `Binary.Note.TimeIndependentDuration` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `Binary.Duration.Tuplet.Unsupported` | `rhythm` | `unsupported-feature` | The binary tuplet discriminator is outside the supported mapping. |
| `Binary.Note.TremoloPicking.Subdivision.Unsupported` | `tremolo-picking` | `unsupported-feature` | The binary tremolo-picking subdivision is outside the supported mapping. |
| `Binary.Note.Trill.Period.Unsupported` | `note-and-beat-semantics` | `unsupported-feature` | The binary trill period is outside the supported mapping. |
| `Binary.Note.Harmonic.Kind.Unsupported` | `harmonics` | `unsupported-feature` | The GP3/4 harmonic kind is outside the supported mapping. |
| `Binary.Note.HarmonicV5.Kind.Unsupported` | `harmonics` | `unsupported-feature` | The GP5 harmonic kind is outside the supported mapping. |
| `Binary.RSE.MasterMetadata` | `score-core` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `Binary.RSE.TrackMetadata` | `score-core` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Asset.DuplicateID` | `score-core` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Asset.EmptyID` | `score-core` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.BackingTrack.AssetId.Reference` | `score-core` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.BackingTrack.EmbeddedFile.Empty` | `score-core` | `invalid-data` | An enabled local backing-track asset must contain audio bytes. |
| `GPIF.BackingTrack.EmbeddedFile.Reference` | `score-core` | `invalid-data` | An enabled local backing-track asset must name an archive entry that exists. |
| `GPIF.BackingTrack.FramePadding.Invalid` | `score-core` | `invalid-data` | Backing-track frame padding must be a signed 64-bit frame count. |
| `GPIF.Bar.DuplicateID` | `staff-ownership` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Bar.EmptyID` | `staff-ownership` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Bar.Voices.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Arpeggio.InvalidValue` | `brush` | `unsupported-feature` | Arpeggio direction must be Up or Down; other values have no lossless public representation. |
| `GPIF.Beat.Chord.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Dynamic.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Beaming.XProperty.Duplicate` | `beaming` | `invalid-data` | Each canonical beat beaming XProperty can occur at most once. |
| `GPIF.Beat.Beaming.XProperty.InvalidValue` | `beaming` | `invalid-data` | A recognized beat beaming XProperty must use its defined integer value. |
| `GPIF.Beat.Brush.Duration.Duplicate` | `brush` | `invalid-data` | A beat can contain at most one exact brush-duration XProperty. |
| `GPIF.Beat.Brush.Duration.Invalid` | `brush` | `invalid-data` | GPIF brush timing must be an integral value within the signed 32-bit target range. |
| `GPIF.Beat.Brush.Duration.Orphan` | `brush` | `invalid-data` | Exact brush timing requires a Brush property or Arpeggio element on the same beat. |
| `GPIF.Beat.Brush.Kind.Conflict` | `brush` | `invalid-data` | A beat cannot declare both Brush and Arpeggio ownership. |
| `GPIF.Beat.Brush.Property.Duplicate` | `brush` | `invalid-data` | A beat can contain at most one Brush property. |
| `GPIF.Beat.XProperty.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | An unrecognized beat XProperty must remain visible to strict parsing instead of becoming structurally accepted through the generic integer container. |
| `GPIF.Beat.TransposedPitchStemOrientation.InvalidValue` | `beaming` | `invalid-data` | A transposed-pitch stem orientation must be Undefined, Upward, or Downward. |
| `GPIF.Beat.UserTransposedPitchStemOrientation.InvalidValue` | `beaming` | `invalid-data` | A user stem override must be Undefined, Upward, or Downward. |
| `GPIF.Beat.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Beat.Fadding.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | The source fade value is not a defined GPIF Fadding token. |
| `GPIF.Beat.Golpe.InvalidValue` | `golpe` | `unsupported-feature` | The source golpe value is not Thumb, Finger, or absent. |
| `GPIF.Beat.GraceNotes.InvalidValue` | `grace-relationships` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Hairpin.InvalidValue` | `hairpins` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Legato.InvalidDestination` | `legato-slurs` | `invalid-data` | An authored beat-level legato destination attribute must contain the exact GPIF boolean true or false. |
| `GPIF.Beat.Legato.InvalidOrigin` | `legato-slurs` | `invalid-data` | An authored beat-level legato origin attribute must contain the exact GPIF boolean true or false. |
| `GPIF.Beat.Notes.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Ottavia.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Barre.Incomplete` | `note-and-beat-semantics` | `invalid-data` | A beat-level barre fret and shape must occur as a pair. |
| `GPIF.Beat.Property.BarreFret.InvalidValue` | `note-and-beat-semantics` | `invalid-data` | A beat-level barre fret must fit the checked public Fret range. |
| `GPIF.Beat.Property.BarreFret.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | A BarreFret property must contain its Fret payload. |
| `GPIF.Beat.Property.BarreString.InvalidValue` | `note-and-beat-semantics` | `invalid-data` | A beat-level BarreString payload must be 0 for full or 1 for half. |
| `GPIF.Beat.Property.BarreString.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | A BarreString property must contain its String payload. |
| `GPIF.Beat.Property.Brush.InvalidDirection` | `brush` | `unsupported-feature` | Brush direction must be Up or Down; other values have no lossless public representation. |
| `GPIF.Beat.Property.Brush.MissingDirection` | `brush` | `invalid-data` | A Brush property must contain its Direction payload. |
| `GPIF.Beat.Property.PickStroke.InvalidDirection` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.PickStroke.MissingDirection` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.Popped.MissingEnable` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.PrimaryPickupTone` | `note-and-beat-semantics` | `deliberate-ignore` | The notation-focused Song model intentionally excludes this playback metadata. |
| `GPIF.Beat.Property.PrimaryPickupVolume` | `note-and-beat-semantics` | `deliberate-ignore` | The notation-focused Song model intentionally excludes this playback metadata. |
| `GPIF.Beat.Property.Slapped.MissingEnable` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Beat.Property.VibratoWTremBar.InvalidStrength` | `beat-vibrato` | `unsupported-feature` | Only the GPIF Slight and Wide beat-vibrato strengths are defined. |
| `GPIF.Beat.Property.VibratoWTremBar.MissingStrength` | `beat-vibrato` | `invalid-data` | A beat-vibrato property must contain its Strength payload. |
| `GPIF.Beat.Property.WhammyBarExtend` | `note-and-beat-semantics` | `deliberate-ignore` | The GPIF extension marker has no documented playback or notation effect. |
| `GPIF.Beat.Rhythm.Reference` | `rhythm` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Tremolo.InvalidValue` | `tremolo-picking` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.ChordDefinition.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.ChordDefinition.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Bar.SimileMark.InvalidValue` | `rhythm` | `unsupported-feature` | Unknown simile-mark text has no defined public enum value. |
| `GPIF.Bar.Ottavia.InvalidValue` | `clef-octave` | `unsupported-feature` | Unknown clef-octave text has no defined public enum value. |
| `GPIF.MasterBar.Fermata.Type.InvalidValue` | `fermata` | `unsupported-feature` | Unknown fermata type text has no defined public enum value. |
| `GPIF.MasterBar.Fermata.Length.InvalidValue` | `fermata` | `invalid-data` | A fermata length must be finite and non-negative before conversion. |
| `GPIF.MasterBar.Fermata.Offset.InvalidValue` | `fermata` | `invalid-data` | A fermata offset must be a non-negative rational value representable as exact score ticks. |
| `GPIF.MasterBar.Fermata.Offset.Duplicate` | `fermata` | `invalid-data` | One master bar cannot contain two authored fermatas at the same exact offset. |
| `GPIF.MasterBar.Bars.Cardinality` | `staff-ownership` | `invalid-data` | The ordered bar references must cover each track and staff exactly once, except that -1 can replace one whole track. |
| `GPIF.MasterBar.Bars.Reference` | `staff-ownership` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.MasterBar.Directions.Jump.InvalidValue` | `score-core` | `unsupported-feature` | Unknown navigation-jump text has no defined public enum value. |
| `GPIF.MasterBar.Directions.Target.InvalidValue` | `score-core` | `unsupported-feature` | Unknown navigation-target text has no defined public enum value. |
| `GPIF.MasterBar.Beaming.Duration.Duplicate` | `beaming` | `invalid-data` | A custom master-bar beaming rule can declare its duration only once. |
| `GPIF.MasterBar.Beaming.Duration.InvalidValue` | `beaming` | `invalid-data` | A custom beaming duration must use a supported note-value denominator. |
| `GPIF.MasterBar.Beaming.Duration.Missing` | `beaming` | `invalid-data` | Authored custom groups require their exact duration denominator. |
| `GPIF.MasterBar.Beaming.Group.Duplicate` | `beaming` | `invalid-data` | Each indexed custom beaming group can occur at most once. |
| `GPIF.MasterBar.Beaming.Group.Gap` | `beaming` | `invalid-data` | Custom beaming group indices must be contiguous from zero. |
| `GPIF.MasterBar.Beaming.Group.InvalidValue` | `beaming` | `invalid-data` | A custom group must be positive and fit the checked GPIF integer range; only trailing zero padding is accepted. |
| `GPIF.MasterBar.Beaming.Groups.Missing` | `beaming` | `invalid-data` | An authored custom beaming duration requires at least one group. |
| `GPIF.MasterBar.XProperty.Unknown` | `score-core` | `unknown-syntax` | An unrecognized master-bar XProperty must remain visible to strict parsing instead of becoming structurally accepted through the generic integer container. |
| `GPIF.MasterBar.TripletFeel.InvalidValue` | `rhythm` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.MasterTrack.Tracks.Reference` | `staff-ownership` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Note.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Note.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Note.InstrumentArticulation.Invalid` | `percussion-articulations` | `invalid-data` | A percussion articulation identity must be non-negative before conversion. |
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
| `GPIF.Note.Property.HarmonicFret.Invalid` | `harmonics` | `invalid-data` | A harmonic fret must be finite and within the public harmonic-fret range. |
| `GPIF.Note.Property.HarmonicFret.MissingPayload` | `harmonics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicType.MissingPayload` | `harmonics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicType.Unsupported` | `harmonics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.HopoDestination` | `note-and-beat-semantics` | `lossy-projection` | Song derives hammer destinations and has no authored field for the GPIF HopoDestination marker. |
| `GPIF.Note.Property.Midi.MissingPayload` | `percussion-articulations` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.Octave` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Slide.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.String.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.Tone` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.TransposedPitch` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Note.Property.Variation` | `percussion-articulations` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.LeftFingering.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | The source left-hand fingering token is not one of P, I, M, A, or C. |
| `GPIF.Note.RightFingering.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | The source right-hand fingering token is not one of P, I, M, A, or C. |
| `GPIF.Note.Vibrato.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | The source note vibrato token is not one of None, Slight, or Wide. |
| `GPIF.Rhythm.DuplicateID` | `rhythm` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Rhythm.EmptyID` | `rhythm` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Rhythm.NoteValue.InvalidValue` | `rhythm` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.DuplicateID` | `staff-ownership` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Track.EmptyID` | `staff-ownership` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Track.Automation.Sound.Reference` | `score-core` | `invalid-data` | The sound automation must reference a sound in its track. |
| `GPIF.MasterBar.Key.Mode.InvalidValue` | `score-core` | `unsupported-feature` | The key mode must use an accepted major or minor spelling; permissive parsing projects an unknown spelling to major. |
| `GPIF.Track.Automation.SustainPedal.Bar.Invalid` | `sustain-pedal` | `invalid-data` | A sustain-pedal automation must reference an existing measure. |
| `GPIF.Track.Automation.SustainPedal.Order.Invalid` | `sustain-pedal` | `invalid-data` | Sustain-pedal positions within one measure must be strictly increasing in source order. |
| `GPIF.Track.Automation.SustainPedal.Position.Invalid` | `sustain-pedal` | `invalid-data` | A sustain-pedal position must be finite and within 0 through 1. |
| `GPIF.Track.Automation.SustainPedal.Value.Invalid` | `sustain-pedal` | `invalid-data` | A sustain-pedal value must use reference 1 for down or 3 for release. |
| `GPIF.Track.Automation.Type.Unknown` | `score-core` | `unknown-syntax` | The track automation type is not recognized. |
| `GPIF.Track.Lyrics.Undispatched` | `score-core` | `lossy-projection` | The public lyric model keeps the lines but not the source dispatch state. |
| `GPIF.Track.PartSounding.NominalKey.Conflict` | `transposition` | `lossy-projection` | The PartSounding nominal key conflicts with the authoritative Transpose key offset. |
| `GPIF.Track.PartSounding.NominalKey.InvalidValue` | `transposition` | `unsupported-feature` | The PartSounding nominal key is neither Neutral nor one of the twelve recognized GPIF spellings. |
| `GPIF.Track.Transposition.ConflictingValues` | `transposition` | `lossy-projection` | Transpose is authoritative when it conflicts with the legacy PartSounding pitch. |
| `GPIF.UnknownAttribute.NoteAndBeat` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.Rhythm` | `rhythm` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.StaffOwnership` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.NoteAndBeat` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.Rhythm` | `rhythm` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.StaffOwnership` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Voice.Beats.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Voice.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Voice.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Beat.Property.ConflictingDuplicate` | `note-and-beat-semantics` | `invalid-data` | Repeated beat properties must not provide conflicting authored values. |
| `GPIF.Note.Property.ConflictingDuplicate` | `note-and-beat-semantics` | `invalid-data` | Repeated note properties must not provide conflicting authored values. |
| `GPIF.Staff.Property.ConflictingDuplicate` | `staff-ownership` | `invalid-data` | Repeated staff properties must not provide conflicting authored values. |
| `GPIF.Track.AudioEngineState.InvalidValue` | `score-core` | `unknown-syntax` | The source audio-engine state must be MIDI or RSE. |
| `GPIF.Track.Property.ConflictingDuplicate` | `staff-ownership` | `invalid-data` | Repeated track properties must not provide conflicting authored values. |
| `GPIF.Track.Sound.Channel` | `score-core` | `unsupported-feature` | The public sound model has no destination for a per-sound MIDI channel. |
| `GPIF.Track.Sound.MIDI.Bank.Invalid` | `midi-bank` | `invalid-data` | Each GPIF bank-select component must fit the seven-bit MIDI MSB or LSB range. |
| `GPIF.Note.Ornament.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Only Turn, InvertedTurn, UpperMordent and LowerMordent are defined source ornament spellings. |
| `GPIF.Beat.Property.Rasgueado.MissingPattern` | `note-and-beat-semantics` | `invalid-data` | A named rasgueado property requires its pattern payload. |
| `GPIF.Beat.Property.Rasgueado` | `note-and-beat-semantics` | `unsupported-feature` | An unknown source pattern has no named semantic destination; no default gesture is substituted. |
| `Binary.Beat.Wah.Unsupported` | `note-and-beat-semantics` | `unsupported-feature` | Legacy wah values below -1 have no supported pedal event; the raw value remains in MixTableChange.Wah. |
| `GPIF.Beat.Wah` | `note-and-beat-semantics` | `unsupported-feature` | Only Open and Closed are supported GPIF beat wah events; unknown values are diagnosed without substituting a state. |
| `GPIF.Chord.Diagram.Property.ShowDiagram` | `note-and-beat-semantics` | `invalid-data` | Chord visibility properties require a literal true or false value. |
| `GPIF.Chord.Diagram.Property.ShowFingering` | `note-and-beat-semantics` | `invalid-data` | Chord visibility properties require a literal true or false value. |
| `GPIF.Chord.Diagram.Property.ShowName` | `note-and-beat-semantics` | `invalid-data` | Chord visibility properties require a literal true or false value. |

## Public model inventory

The inventory starts at `Song`. Unlisted roles are authored values. Compatibility and derived fields have explicit roles.

| Type | Feature | Field roles | Reason |
| --- | --- | --- | --- |
| `Version` | `score-core` | 2 authored, 0 compatibility, 1 derived, 0 out-of-scope | The source version is preserved. Number is parsed from Data. |
| `Clipboard` | `score-core` | 7 authored, 0 compatibility, 0 derived, 0 out-of-scope | The binary clipboard range is authored source data. |
| `BackingTrack` | `score-core` | 9 authored, 0 compatibility, 0 derived, 0 out-of-scope | The backing-track record and embedded asset are authored source data. |
| `SyncPoint` | `timing` | 6 authored, 2 compatibility, 3 derived, 0 out-of-scope | The sync point preserves source values and checked timing views. |
| `VolumeAutomation` | `score-core` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The volume point is authored playback data. |
| `TempoAutomation` | `tempo-automations` | 6 authored, 0 compatibility, 0 derived, 0 out-of-scope | The tempo point preserves authored timing, interpolation, annotation, and visibility data. |
| `Lyrics` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The score lyrics are authored text data. |
| `LyricLine` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The lyric line is authored text data. |
| `PageSetup` | `score-core` | 17 authored, 0 compatibility, 0 derived, 0 out-of-scope | The page setup is authored display data. |
| `RseMasterEffect` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The master RSE effect is authored playback data. |
| `MidiChannel` | `score-core` | 10 authored, 0 compatibility, 0 derived, 0 out-of-scope | The MIDI channel contains authored playback values. |
| `MeasureHeader` | `rhythm` | 14 authored, 1 compatibility, 2 derived, 0 out-of-scope | The header contains authored bar data and derived absolute starts. BeamingRules is the optional exact-bar custom grouping and never derives from TimeSignature. Directions is the complete navigation-marker set; Direction is its legacy single-value compatibility view. Fermatas is the authoritative master-bar hold collection. FreeTime is an independent authored presence marker and does not replace numeric meter timing. |
| `BeamingRules` | `beaming` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | A custom master-bar rule owns its note-value slice duration and an independent ordered group-size array. |
| `Fermata` | `fermata` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The fermata preserves one authored master-bar offset, symbol type, and finite length. |
| `Marker` | `score-core` | 3 authored, 1 compatibility, 0 derived, 0 out-of-scope | Letter and Text are independent authored section fields. Title is the legacy caption; a post-parse title edit overrides text and preserves the letter. |
| `SourceValue` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The wrapper preserves source presence and unknown values. |
| `Track` | `staff-ownership` | 22 authored, 3 compatibility, 0 derived, 0 out-of-scope | The track owns staves. CapoFret is a first-staff compatibility scalar; Measures and Strings are first-staff compatibility views. |
| `Staff` | `staff-ownership` | 7 authored, 0 compatibility, 1 derived, 0 out-of-scope | The staff owns its capo, display and sounding transposition, measures, tuning pitches, and tuning label. PercussionTrack mirrors its track. |
| `TrackSettings` | `score-core` | 11 authored, 0 compatibility, 0 derived, 0 out-of-scope | The track settings are authored display data. |
| `PercussionArticulation` | `percussion-articulations` | 13 authored, 0 compatibility, 0 derived, 0 out-of-scope | The articulation preserves track-local notation and playback identity. |
| `TrackSound` | `score-core` | 6 authored, 0 compatibility, 0 derived, 0 out-of-scope | The sound definition owns its authored program and combined MIDI bank. The first sound is authoritative; MidiChannel mirrors it on import and supplies the fallback only when no explicit sounds exist. |
| `SoundAutomation` | `score-core` | 6 authored, 0 compatibility, 0 derived, 0 out-of-scope | The sound automation preserves authored playback, interpolation, annotation, and visibility data. |
| `TrackLyricLine` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The track lyric line is authored text data. |
| `TrackRse` | `score-core` | 4 authored, 0 compatibility, 0 derived, 0 out-of-scope | The track RSE record is authored playback data. |
| `RseEqualizer` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The equalizer contains authored playback values. |
| `RseInstrument` | `score-core` | 6 authored, 0 compatibility, 0 derived, 0 out-of-scope | The RSE instrument contains authored playback values. |
| `GuitarString` | `staff-ownership` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The string contains authored tuning data. |
| `Measure` | `timing` | 11 authored, 0 compatibility, 4 derived, 0 out-of-scope | The measure contains authored notation, first-staff sustain markers, and finalized ownership and timing. |
| `SustainPedalMarker` | `sustain-pedal` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The marker preserves one ordered measure-relative sustain-pedal action. |
| `Voice` | `note-and-beat-semantics` | 2 authored, 0 compatibility, 1 derived, 0 out-of-scope | The voice owns authored beats. MeasureIndex is finalized ownership data. |
| `Beat` | `note-and-beat-semantics` | 16 authored, 0 compatibility, 2 derived, 0 out-of-scope | The beat contains authored values and finalized starts. BeamingMode controls the connection to the next beat; inversion and preferred direction are independent authored stem overrides. Beat-level barre fields, dead-slap marks, legato endpoints, and ordered lyric lines are independent authored values. |
| `BeatLegato` | `legato-slurs` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The occurrence-owned legato record preserves independent authored phrase endpoints, including excerpt boundaries. |
| `BeatDisplay` | `note-and-beat-semantics` | 7 authored, 0 compatibility, 0 derived, 0 out-of-scope | The beat display record is authored notation data. |
| `BeatStroke` | `brush` | 3 authored, 1 compatibility, 0 derived, 0 out-of-scope | The stroke preserves authored kind and direction. ExactDuration is the exact tick-timing authority, and Duration is its note-value compatibility view. |
| `NoteEffect` | `note-and-beat-semantics` | 23 authored, 0 compatibility, 0 derived, 0 out-of-scope | The note effect record contains authored note techniques and explicit fingering presence. |
| `BendEffect` | `note-and-beat-semantics` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The bend effect contains authored bend data. |
| `BendPoint` | `note-and-beat-semantics` | 3 authored, 1 compatibility, 0 derived, 0 out-of-scope | The bend point preserves authored curve data. Position is the legacy note-offset view when ExactOffset is present. |
| `GraceEffect` | `grace-relationships` | 10 authored, 1 compatibility, 0 derived, 0 out-of-scope | The grace effect contains authored occurrence data. Fret is a legacy view of ExactFret. |
| `HarmonicEffect` | `harmonics` | 4 authored, 1 compatibility, 0 derived, 0 out-of-scope | The harmonic effect preserves authored kind and pitch data. Fret is a legacy view. |
| `TremoloPickingEffect` | `tremolo-picking` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The effect preserves the authored tremolo subdivision and independently classifies its notation style. |
| `TrillEffect` | `note-and-beat-semantics` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The trill effect contains authored fret and duration data. |
| `MixTableChange` | `score-core` | 13 authored, 0 compatibility, 0 derived, 0 out-of-scope | The mix-table change contains authored playback changes. |
| `MixTableItem` | `score-core` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The mix-table item contains an authored value and transition duration. |
| `WahEffect` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The wah effect contains authored playback values. |
| `Duration` | `rhythm` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The duration preserves the authored base value, dots, and tuplet ratio. |
| `TimeSignature` | `rhythm` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The time signature contains authored meter values. |
| `KeySignature` | `score-core` | 2 authored, 0 compatibility, 0 derived, 0 out-of-scope | The key signature contains authored tonality values. |
| `ScoreTime` | `timing` | 0 authored, 0 compatibility, 0 derived, 0 out-of-scope | The private reduced terms provide a checked derived timing value. |
| `BarPosition` | `timing` | 0 authored, 0 compatibility, 0 derived, 0 out-of-scope | The private reduced terms provide a checked authored bar position. |
| `PitchClass` | `note-and-beat-semantics` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | The pitch class contains authored spelling data. |
| `Barre` | `note-and-beat-semantics` | 3 authored, 0 compatibility, 0 derived, 0 out-of-scope | The barre contains authored chord fingering data. |
| `PanAutomation` | `score-core` | 5 authored, 0 compatibility, 0 derived, 0 out-of-scope | Authored normalized pan points are independent from initial channel balance. |
| `Song` | `score-core` | 32 authored, 1 compatibility, 0 derived, 0 out-of-scope | The root contains authored score data. Tempo is the legacy view of InitialTempo. |
| `Note` | `note-and-beat-semantics` | 12 authored, 0 compatibility, 0 derived, 0 out-of-scope | The note contains authored pitch, articulation, duration, and effect values. |
| `BeatEffects` | `note-and-beat-semantics` | 17 authored, 2 compatibility, 0 derived, 0 out-of-scope | Fade is the authored authority. FadeIn is its legacy compatibility view. The remaining fields contain authored notation and playback effects. Tap/slap/pop are independent; the imported legacy enum uses Pop, Slap, Tap priority and edits reconcile explicitly. WahPedal is a beat event with explicit reconciliation against legacy mix-table wah values. |
| `Chord` | `note-and-beat-semantics` | 22 authored, 0 compatibility, 0 derived, 0 out-of-scope | The chord contains authored identity, pitch, fingering, and diagram data. |

Every field also has one target conversion disposition. The gate compares this partition with the public model inventory.

| Target disposition | Fields |
| --- | --- |
| `preserved` | 263 |
| `normalized` | 37 |
| `omitted` | 103 |
| `rejected` | 0 |
| `derived` | 14 |
| `out-of-scope` | 0 |

Each public field has disposition-bearing runtime evidence in the semantic matrix.

## Semantic matrix obligations

The matrix traces formats, stages, value shapes, evidence roles, and typed evidence sources. Structural schema evidence cannot satisfy a semantic leaf or behavior obligation. The current ledger has 129 behavior cases, 1 structural cases, 31 justified structural wire wrappers, and 199 discovered public enum members.

## GPIF wire inventory

The schema inventory records every decoded GPIF field. This inventory detects schema changes only. The source dispatch and public model inventories define semantic handling.

| Wire role | Fields |
| --- | --- |
| `schema` | 263 |

## Source dispatch inventory

The gate compares these cases with the source switches. Each default has an explicit disposition.

| Dispatch | Feature | Cases | Evidence | Default | Reason |
| --- | --- | --- | --- | --- | --- |
| `buildTrack:track.Name` | `score-core` | 1 | `section-track-names` | `unsupported-feature` | An authored empty short name remains empty in the final consumer only when the full name is also empty; a nonempty full name produces a scoped consumer-loss report. |
| `gpifAuditOwnedStaffProperty:property.Name` | `staff-ownership` | 4 | `tuning-label-preservation` | `unknown-syntax` | The audit classifies each track and staff property before import. |
| `gpifAuditBeatProperty:property.Name` | `note-and-beat-semantics` | 19 | `gpif-property-dispatch` | `unknown-syntax` | The audit classifies each named beat property before import. |
| `gpifBeatWhammyProperties:property.Name` | `note-and-beat-semantics` | 8 | `gpif-property-dispatch` | `delegated-to-audit` | The GP6 importer reconstructs the authored whammy curve after the audit validates each named property. |
| `gpifApplyBeatEffects:p.Strength` | `note-and-beat-semantics` | 2 | `beat-vibrato-preservation` | `delegated-to-audit` | The importer preserves each audited GPIF beat-vibrato strength. |
| `gpifApplyBeatEffects:p.String` | `note-and-beat-semantics` | 2 | `beat-barre-preservation` | `delegated-to-audit` | The importer maps the two audited GPIF BarreString values to typed public shapes. |
| `gpifAuditBarrePair:property.Name` | `note-and-beat-semantics` | 2 | `beat-barre-preservation` | `delegated-to-audit` | The audit verifies that the two beat-level barre properties occur as a pair. |
| `gpifNoteToNote:n.Vibrato` | `note-and-beat-semantics` | 2 | `gpif-property-dispatch` | `delegated-to-audit` | The importer preserves each supported GPIF note-vibrato strength after the audit classifies unknown values. |
| `gpifAuditMasterAutomations:automation.Type` | `score-core` | 2 | `automation-dispatch-diagnostic` | `unknown-syntax` | The audit classifies each master-track automation before import. |
| `gpifAuditTrackAutomations:automation.Type` | `score-core` | 2 | `automation-dispatch-diagnostic` | `unknown-syntax` | The audit classifies each track automation, checks sound references, and validates sustain-pedal values, locations, and order before import. |
| `parseGPIFWithContext:automation.Type` | `score-core` | 1 | `automation-dispatch-diagnostic` | `delegated-to-audit` | The importer maps only sound automations after the audit classifies all track automation types. |
| `gpifReadVolumeAutomations:automation.Type` | `score-core` | 1 | `automation-dispatch-diagnostic` | `delegated-to-audit` | The importer maps only validated volume automations after the channel-strip audit. |
| `gpifReadSustainPedals:automation.Type` | `sustain-pedal` | 1 | `sustain-pedal-preservation` | `delegated-to-audit` | The importer maps validated track sustain-pedal automations to staff-0 measures and derives holds across bars with no explicit marker. |
| `gpifReadSyncPoints:automation.Type` | `timing` | 1 | `timing-finalization` | `delegated-to-audit` | The importer maps sync points after the master automation audit. |
| `gpifReadTempoAutomations:auto.Type` | `tempo-automations` | 1 | `tempo-compatibility-authority` | `delegated-to-audit` | The importer maps tempo automations after the master automation audit. |
| `gpifAuditNoteProperty:property.HType` | `harmonics` | 7 | `harmonic-conversion` | `unsupported-feature` | The audit classifies each harmonic type before import. |
| `gpifNoteToNote:p.HType` | `harmonics` | 6 | `harmonic-conversion` | `delegated-to-audit` | The importer maps represented harmonic types after the audit. |
| `readDuration:iTuplet` | `rhythm` | 9 | `timing-finalization` | `unsupported-feature` | The binary reader maps every supported legacy tuplet discriminator. |
| `gpifRhythmToDuration:r.AugmentationDot.Count` | `rhythm` | 3 | `timing-finalization` | `unsupported-feature` | The GPIF reader maps every supported augmentation-dot count. |
| `ExactScoreTime:validated.Dots` | `rhythm` | 2 | `timing-finalization` | `delegated-to-audit` | Exact duration conversion applies the reviewed single- and double-dot factors. |
| `readTremoloPicking:val` | `tremolo-picking` | 3 | `tremolo-import` | `unsupported-feature` | The binary reader maps every supported tremolo-picking subdivision. |
| `readTrill:period` | `note-and-beat-semantics` | 3 | `gpif-property-dispatch` | `unsupported-feature` | The binary reader maps every supported trill subdivision. |
| `readHarmonic:kind` | `harmonics` | 7 | `harmonic-conversion` | `unsupported-feature` | The GP3/4 reader maps every supported harmonic discriminator. |
| `readHarmonicV5ForNote:kind` | `harmonics` | 5 | `harmonic-conversion` | `unsupported-feature` | The GP5 reader maps every supported harmonic discriminator. |
| `gpifReadStaffStrings:property.Name` | `staff-ownership` | 1 | `staff-ownership` | `delegated-to-audit` | The importer maps the classified tuning property. |
| `gpifReadTuningName:property.Name` | `staff-ownership` | 1 | `tuning-label-preservation` | `delegated-to-audit` | The importer maps the classified tuning label independently from pitch values. |
| `gpifAuditChordIDs:property.Name` | `note-and-beat-semantics` | 2 | `chord-occurrence-isolation` | `delegated-to-audit` | The audit checks IDs in both supported chord collection spellings. |
| `gpifReadChordProperties:property.Name` | `note-and-beat-semantics` | 2 | `chord-occurrence-isolation` | `delegated-to-audit` | The importer maps both supported chord collection spellings. |
| `gpifAuditDiagnostics:property.Name` | `percussion-articulations` | 1 | `percussion-identity` | `delegated-to-audit` | The audit uses valid MIDI properties when it checks percussion fallbacks. |
| `isPercussionTrack:t.InstrumentSet.Type` | `percussion-articulations` | 3 | `percussion-identity` | `delegated-to-audit` | Known instrument-set spellings map to one percussion-track value. |
| `gpifNormalizePercussionArticulation:element.Type` | `percussion-articulations` | 1 | `percussion-identity` | `delegated-to-audit` | Percussion elements use their authored articulation identity. |
| `gpifRhythmToDuration:r.NoteValue` | `rhythm` | 8 | `timing-finalization` | `unsupported-feature` | The importer maps each supported GPIF note value to one public duration. |
| `gpifReadCapo:property.Name` | `staff-ownership` | 1 | `capo-precedence` | `delegated-to-audit` | The importer maps the classified capo property to Track.CapoFret. |
| `gpifXMLAuditStart:element.Name.Local` | `score-core` | 5 | `unclassified-gpif-wire-field` | `delegated-to-audit` | The XML audit records graph object identifiers for diagnostic locations. |
| `gpifApplyBeatEffects:p.Direction` | `brush` | 2 | `brush-preservation` | `delegated-to-audit` | The importer maps both supported brush directions. |
| `gpifAuditBrush:property.Name` | `brush` | 1 | `brush-preservation` | `delegated-to-audit` | The audit checks duplicate and conflicting Brush property ownership. |
| `parseGPIFWithContext:mb.Key.Mode` | `score-core` | 5 | `key-mode-preservation` | `delegated-to-audit` | The importer accepts the exact supported GPIF key-mode spellings after the audit reports all other values. |
| `parseGPIFWithContext:mb.TripletFeel` | `rhythm` | 6 | `timing-finalization` | `delegated-to-audit` | The importer maps every supported master-bar triplet-feel value. |
| `parseGPIFWithContext:bar.Ottavia` | `clef-octave` | 4 | `clef-octave-preservation` | `delegated-to-audit` | The importer maps every supported bar-level clef octave independently from beat octave notation. |
| `validateGP8Staff:element.Type` | `percussion-articulations` | 1 | `percussion-identity` | `delegated-to-audit` | The exporter validates percussion articulation MIDI boundaries. |
| `parseGPIFWithContext:b.GraceNotes` | `grace-relationships` | 2 | `grace-order-preservation` | `unsupported-feature` | The importer preserves supported before-beat and on-beat grace ordering. |
| `gpifAuditDiagnostics:beat.Fadding` | `note-and-beat-semantics` | 4 | `gpif-property-dispatch` | `unsupported-feature` | The audit classifies each fading value before the importer maps it. |
| `gpifApplyBeatEffects:b.Ottavia` | `note-and-beat-semantics` | 4 | `gpif-property-dispatch` | `delegated-to-audit` | The importer maps each supported octave-shift value. |
| `gpifApplyBeatEffects:b.Arpeggio` | `brush` | 2 | `brush-preservation` | `delegated-to-audit` | The importer maps each supported arpeggio direction. |
| `gpifAuditDiagnostics:track.AudioEngineState` | `score-core` | 3 | `audio-engine-state` | `unknown-syntax` | The audit accepts the two known playback engines and reports any other token. |
| `gpifReadPanAutomations:a.Type` | `score-core` | 1 | `automation-dispatch-diagnostic` | `delegated-to-audit` | The channel-strip pan event is retained independently from static channel balance. |
| `gpifAuditChannelStripAutomations:automation.Type` | `score-core` | 4 | `automation-dispatch-diagnostic` | `unknown-syntax` | Volume and pan have authored destinations; other recognized channel-strip automation remains omitted. |
| `gpifAuditNoteProperty:property.Name` | `note-and-beat-semantics` | 28 | `gpif-property-dispatch` | `unknown-syntax` | The audit classifies each named note property before import. |
| `gpifNoteToNote:p.Name` | `note-and-beat-semantics` | 20 | `gpif-property-dispatch` | `delegated-to-audit` | The importer maps represented note properties after the audit classifies all names. |
| `gpifNoteOrnament:n.Ornament` | `note-and-beat-semantics` | 4 | `note-ornaments` | `unsupported-feature` | The four GPIF spellings retain exact variants. Absence maps to None; the source audit reports unknown strings before conversion. |
| `gpifApplyBeatEffects:p.Name` | `note-and-beat-semantics` | 8 | `gpif-property-dispatch` | `delegated-to-audit` | The importer maps represented beat properties after the audit classifies all names. |

## Behavioral contracts

Each represented feature has a public-API test and pinned independent-consumer evidence. Mutation-sensitive contracts are exercised by `conformance/sensitivity.mjs`.

| Contract | Feature | Public API test | Independent test | Mutation check | Reason |
| --- | --- | --- | --- | --- | --- |
| `pick-stroke-preservation` | `note-and-beat-semantics` | `TestConformancePickStroke` | `TestAlphaTabPreservesPickStroke` | no | Up, down, and absent pick marks retain their exact direction independently from brush through strict GP8 export, Go reimport, and pinned AlphaTab. |
| `chord-rest-dynamic-policy` | `note-and-beat-semantics` | `TestConformanceChordAndRestDynamics` | `TestAlphaTabChordAndRestDynamics` | no | Two-note velocities 47/95 and explicit beat/rest dynamics 47/48 preserve authored values, emit P, and report exact target losses; strict export requires each applicable code. |
| `simile-mark-preservation` | `rhythm` | `TestParseGPIFPreservesSimileMarks` | `TestAlphaTabPreservesSimileMarks` | no | One-bar and both halves of two-bar simile repeats survive the public model and an independently consumed GP8 export. |
| `key-mode-preservation` | `score-core` | `TestConformanceKeyModes` | `TestAlphaTabPreservesKeyModes` | no | Exact supported major and minor spellings retain independent nonzero accidental counts across adjacent master bars and canonical GP8 export; unknown spellings remain explicit. |
| `transposition-preservation` | `transposition` | `TestConformanceTransposition` | `TestAlphaTabPreservesTransposition` | no | Display offsets and independently derived effective keys survive GPIF import and pinned AlphaTab consumption; GP8 target limits for independent keys, sounding offsets, and differing later-staff values receive scoped reports. |
| `midi-bank-preservation` | `midi-bank` | `TestConformanceMIDIBank` | `TestAlphaTabPreservesMIDIBanks` | no | Full-range bank definitions, initial channel mirrors, and duplicate-position sound changes survive checked GP5/GPIF import and GP8 export in source order. |
| `reference-aware-midi-program-validation` | `score-core` | `TestConformanceMIDIProgramReferences` | `TestAlphaTabPreservesSelectedMIDIProgram` | no | Actual GP3 and GP4 unused -1 slots no longer invalidate GP8 export, while selected invalid values remain exact track-located rejections and a valid non-default program survives the pinned consumer. |
| `clef-octave-preservation` | `clef-octave` | `TestConformanceClefOctave` | `TestAlphaTabPreservesClefOctaves` | no | Every supported bar-level clef octave survives at its exact staff and measure location independently from Beat.Octave. |
| `fermata-preservation` | `fermata` | `TestConformanceFermatas` | `TestAlphaTabPreservesFermatas` | no | GPIF preserves exact authored holds. The pinned consumer derives beat association and can move or overwrite holds; scoped reports protect strict export. |
| `free-time-preservation` | `free-time` | `TestConformanceFreeTime` | `TestAlphaTabPreservesFreeTime` | no | Every authored free-time marker survives on its exact master bar without changing numeric meter timing. |
| `direction-preservation` | `score-core` | `TestConformanceDirections` | `TestAlphaTabPreservesDirections` | no | Every navigation target and jump survives as one canonical set, including simultaneous markers and documented legacy pointer reconciliation. |
| `unclassified-model-field` | `score-core` | `TestSemanticContractInventory` | none | yes | A new public field must receive a semantic role before the gate passes. |
| `unclassified-source-dispatch` | `note-and-beat-semantics` | `TestSemanticContractInventory` | none | yes | A new semantic dispatch case must receive a source disposition before the gate passes. |
| `automation-dispatch-diagnostic` | `score-core` | `TestGPIFAutomationDispatchDiagnostics` | `TestAlphaTabInputConformance` | yes | Unknown, unsupported, and invalid automation records remain visible to strict policy. |
| `supported-effect-serialization` | `hairpins` | `TestExportGP8PreservesHairpins` | `TestAlphaTabExportConformance` | yes | Non-default hairpins survive GP8 serialization and independent consumption. |
| `shared-authored-export-validation` | `score-core` | `TestGP8ExportRejectsSharedAuthoredInvariants` | none | yes | Export uses the same authored-value diagnostics as programmatic validation. |
| `tempo-compatibility-authority` | `tempo-automations` | `TestGP8ExportReconcilesSemanticAndLegacyTempo` | `TestAlphaTabTempoReferences` | yes | A post-parse legacy tempo edit remains authoritative and fractional semantic tempo remains exact. |
| `chord-occurrence-isolation` | `note-and-beat-semantics` | `TestRepeatedGPIFChordOccurrencesOwnMutablePayloads` | none | yes | Repeated chord references do not alias mutable occurrence data. |
| `master-bar-narrowing-boundary` | `rhythm` | `TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries` | `TestAlphaTabInputConformance` | yes | Values are checked before narrowing and modulo aliases are rejected. |
| `reported-effect-loss` | `note-and-beat-semantics` | `TestGP8StrictExportReportsUnsupportedBeatAndNoteEffects` | `TestAlphaTabExportConformance` | no | Each represented non-default effect is emitted or produces a stable conversion decision. |
| `grace-order-preservation` | `grace-relationships` | `TestExportGP8PreservesOrderedMultipleGraceNotes` | `TestAlphaTabExportConformance` | no | Grace order and attachment survive GP8 conversion. |
| `timing-finalization` | `timing` | `TestFinalizeSongIsIdempotentAndAcceptsSpecialStructures` | `TestAlphaTabGPIFTiming` | no | Finalization is stable and exact timing agrees with the independent consumer. |
| `staff-ownership` | `staff-ownership` | `TestParseGPIFPreservesGrandStaff` | `TestAlphaTabMultiStaffTrackOrdering` | no | Grand-staff ownership and ordering survive public parsing. |
| `tuning-label-preservation` | `staff-ownership` | `TestConformanceTuningLabels` | `TestAlphaTabPreservesTuningLabels` | no | Named and explicitly empty staff-local tuning labels remain independent from identical pitch arrays, string order, and capo through import, public editing, GP8 export, and pinned consumer import. |
| `tremolo-import` | `tremolo-picking` | `TestConformanceTremoloPicking` | `TestAlphaTabPreservesTremoloPicking` | no | Binary and GPIF marks 1 through 3 retain their exact rate on the beat and through the independent GP8 consumer. |
| `harmonic-conversion` | `harmonics` | `TestExportGP8PreservesHarmonicsAndWhammyCurves` | `TestAlphaTabExportConformance` | no | Represented harmonic values survive GP8 conversion. |
| `percussion-identity` | `percussion-articulations` | `TestGPIFPercussionPreservesArticulations` | `TestAlphaTabInputConformance` | no | Percussion identity and notation metadata agree with the independent consumer. |
| `native-percussion-fallbacks` | `percussion-articulations` | `TestConformanceNativePercussionFallbacks` | `TestAlphaTabPreservesNativePercussionFallbacks` | no | Every scoped native input keeps its exact element, notation, playback, main-note, and grace-note identity; IDs without a matching native definition remain precisely rejected. |
| `gpif-property-dispatch` | `note-and-beat-semantics` | `TestParseWithOptionsReportsGPIFContentLoss` | `TestAlphaTabInputConformance` | no | Named properties are preserved or produce explicit parse diagnostics. |
| `master-bar-denominator-boundary` | `rhythm` | `TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries` | `TestAlphaTabInputConformance` | yes | A denominator that exceeds uint16 cannot wrap to a valid value. |
| `unclassified-semantic-selector` | `grace-relationships` | `TestSemanticContractInventory` | none | yes | A case on any selector switch must receive a source disposition. |
| `unclassified-gpif-wire-field` | `score-core` | `TestSemanticContractInventory` | none | yes | A new decoded GPIF field must receive a source disposition. |
| `inspected-track-capo` | `staff-ownership` | `TestGP8StrictExportCoversInspectedSemanticFields` | `TestAlphaTabPreservesInspectedCapo` | no | A nonzero capo survives GP8 conversion and independent consumption. |
| `inspected-note-duration-percent` | `note-and-beat-semantics` | `TestGP8StrictExportCoversInspectedSemanticFields` | none | yes | Strict export reports a non-default duration percentage before it emits bytes. |
| `chord-diagram-preservation` | `chord-diagram` | `TestConformanceChordDefinitions` | `TestAlphaTabPreservesChordDiagrams` | yes | Representable ranges and explicit or synthesized finger positions preserve string orientation, first fret, scope, occurrence ownership, and checked target mappings. |
| `note-fingering-preservation` | `note-and-beat-semantics` | `TestConformanceFingering` | `TestAlphaTabPreservesFingering` | yes | Distinct left and right Thumb through Little values retain explicit presence, exact P/I/M/A/C wire spellings, and pinned-consumer identity; authored Open remains a narrow target omission. |
| `inspected-bend-points` | `note-and-beat-semantics` | `TestGP8StrictExportCoversInspectedSemanticFields` | none | no | The conversion reports point-count loss and curves that GPIF shared middle values would normalize. |
| `field-disposition-evidence` | `note-and-beat-semantics` | `TestSemanticContractInventory` | none | yes | A field claim must match the disposition proved by its focused evidence. |
| `capo-precedence` | `staff-ownership` | `TestGPIFCapoUsesStaffFallbackAndRejectsNarrowing` | `TestAlphaTabGPIFCapoPrecedence` | no | Import and diagnostics use the effective staff capo values that the independent consumer uses. |
| `independent-staff-capos` | `staff-ownership` | `TestConformanceStaffCapo` | `TestAlphaTabPreservesIndependentStaffCapos` | no | Distinct authored staff capos and their sounding pitches survive GP8 export without collapsing to the legacy track scalar. |
| `structural-export-resilience` | `staff-ownership` | `TestConformanceStructuralResilience` | none | yes | Valid generated public graphs keep their complete topology and note values through GP8 export and reimport. |
| `audio-engine-state` | `score-core` | `TestConformanceSourceAudit` | none | no | MIDI and RSE map to distinct public playback states, and unknown states remain visible. |
| `beat-dynamic-quantization` | `note-and-beat-semantics` | `TestConformanceDynamicQuantization` | `TestAlphaTabGP8ReadsNormalAndRestDynamic` | yes | Each authored dynamic either survives as its canonical marking or produces an exact normalization decision. |
| `legato-preservation` | `legato-slurs` | `TestConformanceLegato` | `TestAlphaTabPreservesLegato` | no | GPIF retains occurrence-owned legato flags. Export reports each authored destination that differs from the pinned consumer destination. |
| `beat-barre-preservation` | `note-and-beat-semantics` | `TestConformanceBarre` | `TestAlphaTabPreservesBeatBarres` | no | Paired checked fret and full/half shape values survive as occurrence-owned beat-level marks through GPIF import, public edits, GP8 export, and independent consumer checks without merging with chord diagram barres. |
| `beat-vibrato-preservation` | `beat-vibrato` | `TestConformanceBeatVibrato` | `TestAlphaTabPreservesBeatVibrato` | yes | Binary presence maps to Slight while GPIF Slight and Wide remain distinct through typed and legacy edits, exact GP8 Strength output, reimport, and pinned-consumer loading without conflating note vibrato. |
| `beat-fade-preservation` | `note-and-beat-semantics` | `TestConformanceBeatFade` | `TestAlphaTabPreservesBeatFade` | yes | Binary presence maps to FadeIn while every GPIF fade value remains distinct through typed and legacy edits, exact Fadding output, reimport, and pinned-consumer loading. |
| `golpe-preservation` | `golpe` | `TestConformanceGolpe` | `TestAlphaTabPreservesGolpe` | yes | None, Thumb, and Finger remain distinct through occurrence-owned GPIF import, direct public edits, exact GP8 output, reimport, and pinned-consumer loading without changing notes or other beat techniques. |
| `dead-slap-preservation` | `note-and-beat-semantics` | `TestConformanceDeadSlap` | `TestAlphaTabPreservesDeadSlap` | yes | A note-free dead slap remains a normal authored beat with no synthetic note, distinct from rests and empty beats, through GPIF import, public edits, exact marker output, Go reimport, and pinned-consumer loading. |
| `brush-preservation` | `brush` | `TestConformanceBrush` | `TestAlphaTabPreservesBrush` | yes | Binary stroke codes and GPIF Brush or Arpeggio spellings retain exact authored timing, source absence, occurrence isolation, and deterministic compatibility edits through GP8 output and pinned-consumer loading. |
| `beat-lyrics-preservation` | `beat-lyrics` | `TestConformanceBeatLyrics` | `TestAlphaTabPreservesBeatLyrics` | yes | Ordered beat-scoped lyric lines, including empty and Unicode values, survive as independently editable occurrences with exact absent-versus-empty GPIF representation and remain distinct from FreeText and score or track lyrics. |
| `beaming-preservation` | `beaming` | `TestConformanceBeaming` | `TestAlphaTabPreservesBeaming` | yes | Authored master-bar groups, beam connection modes, inverted stems, and explicit up/down directions survive binary or GPIF import, public edits, exact GP8 wire output, and pinned-consumer loading. |
| `sustain-pedal-preservation` | `sustain-pedal` | `TestConformanceSustainPedals` | `TestAlphaTabPreservesSustainPedals` | no | Ordered staff-0 down and release markers survive import, public editing, and GP8 export while empty continuing bars receive one derived hold marker that is skipped on the wire. Validation rejects a Down in a bar entered with the pedal down because consumers reinterpret it as Hold from bar-entry state. |
| `unclassified-public-enum-member` | `note-and-beat-semantics` | `TestSemanticMatrixInventory` | none | yes | A new public enum member must have focused behavioral evidence. |
| `whammy-owner-context` | `note-and-beat-semantics` | `TestParseBinaryWhammyPreservesDipsAndHolds` | none | yes | Beat whammy dips and holds must not pass through note-bend canonicalization or discard negative controls. |
| `whammy-corpus-projection` | `note-and-beat-semantics` | `TestWhammyProjectionAdaptersExposeBeatCurves` | `TestAlphaTabWhammyCorpusConformance` | yes | Both corpus adapters must expose whammy curves so semantic comparison can detect regressions. |
| `whammy-target-interpretation` | `note-and-beat-semantics` | `TestGP8WhammyMiddleHoldReportsInterpretedLoss` | `TestAlphaTabGP8WhammyTargetInterpretation` | yes | Loss reporting must compare the authored curve with the curve consumers interpret, not only the raw GPIF controls. |
| `octave-variant-conformance` | `note-and-beat-semantics` | `TestConformanceBeatEffects` | none | yes | Each octave variant must survive GP8 export, Go reimport, and independent consumption. |
| `semantic-wire-requires-behavior` | `score-core` | `TestSemanticMatrixInventory` | none | yes | A structural schema round trip cannot satisfy a semantic wire-field obligation. |
| `semantic-wire-executable-assertion` | `score-core` | `TestSemanticMatrixInventory` | none | yes | A semantic wire mapping must have an exact executable assertion in its focused case. |
| `semantic-obligation-shape` | `score-core` | `TestSemanticMatrixInventory` | none | yes | Every executable case must declare the dimensions needed to interpret its evidence. |
| `percussion-resolved-staff-resources` | `percussion-articulations` | `TestGP8PercussionUsesEveryResolvedStaff` | none | yes | Every resolved staff, including a grace-only staff, contributes required percussion resources. |
| `percussion-resource-order` | `staff-ownership` | `TestGP8PercussionUsesEveryResolvedStaff` | none | yes | Staves precede InstrumentSet so the pinned consumer has staff context for percussion resources. |
| `percussion-articulation-lookup` | `percussion-articulations` | `TestGP8PercussionRejectsMissingArticulationResource` | none | yes | A missing exported percussion resource is an error. |
| `percussion-line-count-policy` | `staff-ownership` | `TestGP8PercussionLineCountPolicy` | none | yes | A later conflicting line count is reported and refused by strict export. |
| `isolated-midi-controller-report` | `score-core` | `TestConformancePlaybackRouting` | none | yes | Each grouped MIDI-effect loss condition is exercised with only one controller set. |
| `isolated-harmonic-member-report` | `harmonics` | `TestConformanceHarmonicVariants` | none | yes | Pitch and octave each trigger the shared harmonic omission report in isolation. |
| `isolated-chord-legacy-report` | `note-and-beat-semantics` | `TestConformanceChordDefinitions` | none | yes | Every legacy chord field triggers the shared omission report in isolation. |
| `section-track-names` | `score-core` | `TestConformanceSectionTrackNames` | `TestAlphaTabSectionTrackNames` | yes | Distinct authored values, empty and absent records, post-parse authority, exact XML and final consumer results, with scoped empty-name and boundary-whitespace reports. |
| `bend-control-roles` | `note-and-beat-semantics` | `TestConformanceBendControlRoles` | `TestAlphaTabBendControlRoles` | no | Named roles retain exact offsets; pinned consumer retains the same standard-gesture controls from source and output. |
| `grace-bend-omission` | `note-and-beat-semantics` | `TestConformanceGraceBendPolicy` | `TestAlphaTabGraceBendPolicy` | no | Real binary transition2 survives Parse; strict export requires the exact reported omission and never invents a bend curve. The independent consumer limitation is explicit. |
| `duration-percent-target-loss` | `note-and-beat-semantics` | `TestConformanceNoteDurationPercentLoss` | `TestAlphaTabNoteDurationPercentLoss` | no | Exact authored0.5/0.75 omit with selective strict refusal; consumer1 leaves rhythm, ties and let-ring intact. |
| `trill-speed-target-limit-16` | `note-and-beat-semantics` | `TestConformanceTrillSixteenth` | `TestAlphaTabTrillSpeeds` | no | Authored trill duration16 retains target fret7 and unrelated rhythm/effects; pinned consumer speed is16 and the exact policy reflects this boundary. |
| `trill-speed-target-limit-32` | `note-and-beat-semantics` | `TestConformanceTrillThirtySecond` | `TestAlphaTabTrillSpeeds` | no | Authored trill duration32 retains target fret7 and unrelated rhythm/effects; pinned consumer speed is16 and the exact policy reflects this boundary. |
| `trill-speed-target-limit-64` | `note-and-beat-semantics` | `TestConformanceTrillSixtyFourth` | `TestAlphaTabTrillSpeeds` | no | Authored trill duration64 retains target fret7 and unrelated rhythm/effects; pinned consumer speed is16 and the exact policy reflects this boundary. |
| `string-number-display` | `note-and-beat-semantics` | `TestConformanceStringNumberDisplay` | `TestAlphaTabStringNumberDisplay` | yes | Exact display flags, independent note occurrence edits, GPIF Enable presence and raw final consumer string/fret/MIDI facts. |
| `metadata-text` | `score-core` | `TestConformanceMetadataText` | `TestAlphaTabMetadataText` | no | Exact raw consumer text including notices and tabber; precise boundary and legacy omissions. |
| `assigned-lyrics` | `score-core` | `TestConformanceAssignedLyrics` | `TestAlphaTabAssignedLyrics` | no | Ordered source index conversion, explicit track authority, exact wire and final consumer dispatch with separate beat ownership. |
| `exact-whammy-offsets` | `note-and-beat-semantics` | `TestConformanceExactWhammyOffsets` | `TestAlphaTabExactWhammyOffsets` | no | Shared ExactOffset authority preserves fractional and bounded nonmonotonic whammy roles; raw pinned consumer controls retain distinct35 and35.5 percent middle offsets. |
| `note-ornaments` | `note-and-beat-semantics` | `TestConformanceNoteOrnaments` | `TestAlphaTabNoteOrnaments` | no | Four authored variants and None remain independent per occurrence; unknown source strings and public enum values are diagnosed without changing string or fret. |
| `chord-display` | `note-and-beat-semantics` | `TestConformanceChordDisplay` | `TestAlphaTabChordDisplay` | no | Each explicit true and false flag retains scoped and independently editable occurrences through raw final consumer loading. |
