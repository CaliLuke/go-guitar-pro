// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestConformanceMultiStaffContext(t *testing.T) {
	runConformanceMultiStaffContext(newConformanceRun(t))
}

func runConformanceMultiStaffContext(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	primary := &song.Tracks[0]
	primary.Name = "Grand staff"
	primary.Offset = 3
	primary.Settings.Notation = true
	primary.Measures = slices.Clone(primary.Measures[:1])
	primary.Measures[0].Clef = MeasureClefTreble
	primary.Strings = []GuitarString{{Number: 1, Value: 67}, {Number: 2, Value: 55}}
	primary.Measures[0].Voices[0].Beats[0].Notes[0].String = 1
	primary.Measures[0].Voices[0].Beats[0].Notes[0].Value = 2
	upper := Staff{Measures: primary.Measures, Strings: primary.Strings, StandardNotationLineCount: 5}
	lowerMeasures := slices.Clone(primary.Measures)
	lowerMeasures[0].Clef = MeasureClefBass
	lowerMeasures[0].Voices = []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{{Value: 4, String: 2, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1}}}}}}
	lower := Staff{Measures: lowerMeasures, Strings: []GuitarString{{Number: 1, Value: 48}, {Number: 2, Value: 36}}, StandardNotationLineCount: 5}
	primary.Staves = []Staff{upper, lower}

	followingMeasures := slices.Clone(primary.Measures)
	followingMeasures[0].Clef = MeasureClefAlto
	followingMeasures[0].Voices = []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{{Value: 1, String: 1, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1}}}}}}
	followingStrings := []GuitarString{{Number: 1, Value: 60}}
	following := Track{
		Name: "Following", ChannelIndex: 0, Visible: true, Settings: TrackSettings{Notation: true},
		Measures: followingMeasures, Strings: followingStrings,
		Staves: []Staff{{Measures: followingMeasures, Strings: followingStrings, StandardNotationLineCount: 5}},
	}
	song.Tracks = append(song.Tracks, following)
	if finalizeErr := FinalizeSong(song); finalizeErr != nil {
		t.Fatal(finalizeErr)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("combined multi-staff score diagnostics = %#v", diagnostics)
	}
	run.Field("Song.Tracks", len(song.Tracks), 2)
	run.Field("Track.Staves", []int{len(song.Tracks[0].Staves), len(song.Tracks[1].Staves)}, []int{2, 1})
	run.Field("Track.Offset", []int32{song.Tracks[0].Offset, song.Tracks[1].Offset}, []int32{3, 0})
	run.Field("Staff.Strings", [][]GuitarString{song.Tracks[0].Staves[0].Strings, song.Tracks[0].Staves[1].Strings, song.Tracks[1].Staves[0].Strings}, [][]GuitarString{primary.Strings, lower.Strings, followingStrings})
	run.Field("Measure.Clef", []MeasureClef{song.Tracks[0].Staves[0].Measures[0].Clef, song.Tracks[0].Staves[1].Measures[0].Clef, song.Tracks[1].Staves[0].Measures[0].Clef}, []MeasureClef{MeasureClefTreble, MeasureClefBass, MeasureClefAlto})

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("combined multi-staff export = %v, %#v", err, report.Entries)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(roundTrip.Tracks) != 2 || len(roundTrip.Tracks[0].Staves) != 2 || len(roundTrip.Tracks[1].Staves) != 1 {
		t.Fatalf("round-trip ownership = %#v", roundTrip.Tracks)
	}
	gotTunings := [][]GuitarString{roundTrip.Tracks[0].Staves[0].Strings, roundTrip.Tracks[0].Staves[1].Strings, roundTrip.Tracks[1].Staves[0].Strings}
	wantTunings := [][]GuitarString{primary.Strings, lower.Strings, followingStrings}
	if !reflect.DeepEqual(gotTunings, wantTunings) {
		t.Fatalf("round-trip tunings = %#v, want %#v", gotTunings, wantTunings)
	}
	run.Wire("gpifMasterTrack.Tracks", []string{roundTrip.Tracks[0].Name, roundTrip.Tracks[1].Name}, []string{"Grand staff", "Following"})
	run.Wire("gpifTrack.Staves", []int{len(roundTrip.Tracks[0].Staves), len(roundTrip.Tracks[1].Staves)}, []int{2, 1})
	run.Wire("gpifStaffProperty.Pitches", gotTunings, wantTunings)
	run.Wire("gpifStaffProperty.Fret", roundTrip.Tracks[0].Offset, int32(3))
}

func TestConformancePickupTupletTempo(t *testing.T) {
	runConformancePickupTupletTempo(newConformanceRun(t))
}

func runConformancePickupTupletTempo(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Anacrusis = true
	song.Tracks[0].Settings.Notation = true
	song.InitialTempo = KnownSourceValue(BPM(120))
	song.Tempo = 120
	song.TempoAutomations = []TempoAutomation{{Bar: 0, Position: 0, Tempo: 120}, {Bar: 1, Position: 0.25, Tempo: 132.5}}
	dottedTuplet := Duration{Value: uint16(DurationSixteenth), Dotted: true, TupletEnters: 7, TupletTimes: 4}
	beats := make([]Beat, 7)
	for index := range beats {
		beats[index] = Beat{Duration: dottedTuplet, Status: BeatStatusRest, Dynamics: MinVelocity + VelocityIncrement*3}
	}
	quarter := defaultDuration()
	song.Tracks[0].Measures[0].Voices = []Voice{
		{Beats: beats},
		{Beats: []Beat{{Duration: quarter, Status: BeatStatusRest, Dynamics: MinVelocity + VelocityIncrement*2}}},
	}
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("pickup combination diagnostics = %#v", diagnostics)
	}
	run.Field("Song.Anacrusis", song.Anacrusis, true)
	run.Field("Duration.Dotted", beats[0].Duration.Dotted, true)
	run.Field("Duration.TupletEnters", beats[0].Duration.TupletEnters, uint8(7))
	run.Field("Duration.TupletTimes", beats[0].Duration.TupletTimes, uint8(4))
	run.Field("Measure.Voices", len(song.Tracks[0].Measures[0].Voices), 2)
	run.Field("Beat.ExactStart", [2]int64{song.Tracks[0].Measures[0].Voices[0].Beats[1].ExactStart.Numerator(), song.Tracks[0].Measures[0].Voices[0].Beats[1].ExactStart.Denominator()}, [2]int64{8160, 7})
	run.Field("MeasureHeader.ExactStart", song.MeasureHeaders[1].ExactStart.FloorTicks(), int64(2400))
	run.Field("Song.TempoAutomations", song.TempoAutomations, []TempoAutomation{{Bar: 0, Position: 0, Tempo: 120}, {Bar: 1, Position: 0.25, Tempo: 132.5}})

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("pickup combination export = %v, %#v", err, report.Entries)
	}
	wire := extractAutomationWireDocument(t, data)
	run.Wire("gpifAutomation.Bar", conformanceAutomationInts(wire.masterAutomations, func(item conformanceAutomationWireAutomation) int { return item.Bar }), []int{0, 1})
	run.Wire("gpifAutomation.Position", conformanceAutomationFloats(wire.masterAutomations, func(item conformanceAutomationWireAutomation) float64 { return item.Position }), []float64{0, 0.25})
	run.Wire("gpifAutomation.Value", conformanceAutomationStrings(wire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Value }), []string{"120 2", "132.5 2"})
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0]
	if !roundTrip.Anacrusis || len(got.Voices) != 2 || got.Voices[0].Beats[0].Duration != dottedTuplet || !reflect.DeepEqual(roundTrip.TempoAutomations, song.TempoAutomations) {
		t.Fatalf("pickup combination round trip = %#v", roundTrip)
	}
}

func TestConformanceChordOccurrence(t *testing.T) {
	runConformanceChordOccurrence(newConformanceRun(t))
}

func runConformanceChordOccurrence(run *conformanceRun) {
	t := run.t
	parsed, err := Parse(conformanceGPIFArchive(t, conformanceChordScopedGPIF))
	if err != nil {
		t.Fatal(err)
	}
	beats := parsed.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	first, second := beats[0].Effect.Chord, beats[2].Effect.Chord
	if first == second || !reflect.DeepEqual(*first, *second) {
		t.Fatal("shared source chord did not create independent equal occurrences")
	}
	first.Fingerings = []Fingering{FingeringIndex}
	first.Barres = []Barre{{Fret: 5, Start: 1, End: 1}}
	first.Strings[0] = 7
	*first.FirstFret = 5
	second.Fingerings = []Fingering{FingeringLittle}
	second.Barres = []Barre{{Fret: 1, Start: 1, End: 1}}
	run.Field("BeatEffects.Chord", []string{first.Name, second.Name}, []string{"Upper", "Upper"})
	run.Field("Chord.Fingerings", [][]Fingering{first.Fingerings, second.Fingerings}, [][]Fingering{{FingeringIndex}, {FingeringLittle}})
	run.Field("Chord.Barres", [][]Barre{first.Barres, second.Barres}, [][]Barre{{{Fret: 5, Start: 1, End: 1}}, {{Fret: 1, Start: 1, End: 1}}})
	run.Field("Chord.Strings", []int8{first.Strings[0], second.Strings[0]}, []int8{7, 1})
	run.Field("Chord.FirstFret", []uint8{*first.FirstFret, *second.FirstFret}, []uint8{5, 1})

	report := PreflightExport(parsed, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.chord-fingerings", "gp8.omit.chord-barres"} {
		if !hasExportCode(report, code) {
			t.Fatalf("chord combination report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(parsed, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotBeats := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	run.Wire("gpifBeat.Chord", []string{gotBeats[0].Effect.Chord.Name, gotBeats[2].Effect.Chord.Name}, []string{"Upper", "Upper"})
	run.Wire("gpifDiagram.Frets", []int8{gotBeats[0].Effect.Chord.Strings[0], gotBeats[2].Effect.Chord.Strings[0]}, []int8{7, 1})
	if len(gotBeats[0].Effect.Chord.Fingerings) != 0 || len(gotBeats[0].Effect.Chord.Barres) != 0 {
		t.Fatal("omitted chord details unexpectedly appeared in GP8 output")
	}
}

func TestConformanceGraceCombinations(t *testing.T) {
	runConformanceGraceCombinations(newConformanceRun(t))
}

func runConformanceGraceCombinations(run *conformanceRun) {
	t := run.t
	t.Run("pitched", func(t *testing.T) {
		song := semanticExportProbeSong(t)
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		note.TieOrigin = true
		note.Effect.Bend = &BendEffect{Points: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 2}}}
		note.Effect.Graces = []GraceEffect{conformanceGrace(2, DurationThirtySecond, Forte, false, 0, GraceEffectTransitionHammer, false)}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("pitched grace combination = %v, %#v", err, report.Entries)
		}
		gotSong, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := gotSong.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		run.Field("Note.TieOrigin", got.TieOrigin, true)
		run.Field("NoteEffect.Bend", got.Effect.Bend.Points, note.Effect.Bend.Points)
		run.Field("NoteEffect.Graces", len(got.Effect.Graces), 1)
		run.Field("GraceEffect.Transition", got.Effect.Graces[0].Transition, GraceEffectTransitionHammer)
	})

	t.Run("percussion", func(t *testing.T) {
		song := syntheticGP8Song()
		track := &song.Tracks[0]
		track.Mute = true
		track.PercussionArticulations = []PercussionArticulation{
			{ElementName: "Kit", ElementType: "percussion", Name: "Main", NoteheadDefault: "noteheadBlack", NoteheadHalf: "noteheadBlack", NoteheadWhole: "noteheadBlack", TechniquePlacement: "outside", InputMIDINumbers: []int{38}, OutputMIDINumber: 38},
			{ElementName: "Kit", ElementType: "percussion", Name: "Muted grace", NoteheadDefault: "noteheadXBlack", NoteheadHalf: "noteheadXBlack", NoteheadWhole: "noteheadXBlack", TechniquePlacement: "outside", InputMIDINumbers: []int{42}, OutputMIDINumber: 42},
		}
		note := &track.Measures[0].Voices[0].Beats[0].Notes[0]
		note.HasPercussionArticulation = true
		note.PercussionArticulation = 0
		note.Effect.Graces = []GraceEffect{{Duration: DurationThirtySecond, Fret: 38, ExactFret: ptrTo(Fret(38)), HasPercussionArticulation: true, PercussionArticulation: 1, IsDead: true, Velocity: Forte}}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("percussion grace combination = %v, %#v", err, report.Entries)
		}
		gotSong, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		gotTrack := gotSong.Tracks[0]
		got := gotTrack.Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[0]
		run.Field("Track.Mute", gotTrack.Mute, true)
		run.Field("GraceEffect.HasPercussionArticulation", got.HasPercussionArticulation, true)
		run.Field("GraceEffect.PercussionArticulation", got.PercussionArticulation, 1)
		run.Field("GraceEffect.IsDead", got.IsDead, true)
		run.Field("Track.PercussionArticulations", gotTrack.PercussionArticulations[:2], track.PercussionArticulations)
	})
}

func TestConformanceEffectsAndDynamics(t *testing.T) {
	runConformanceEffectsAndDynamics(newConformanceRun(t))
}

func runConformanceEffectsAndDynamics(run *conformanceRun) {
	t := run.t
	song := semanticExportProbeSong(t)
	track := &song.Tracks[0]
	beat := &track.Measures[0].Voices[0].Beats[0]
	note := &beat.Notes[0]
	harmonicFret := int8(5)
	harmonicExact := float64(5)
	note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, Fret: &harmonicFret, FretFloat: &harmonicExact}
	note.Effect.Bend = &BendEffect{Points: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 2}}}
	note.Effect.Slides = []SlideType{SlideShiftSlideTo}
	note.Effect.Hammer = true
	mezzoPiano := MinVelocity + VelocityIncrement*3
	fortissimo := MinVelocity + VelocityIncrement*6
	rest := Beat{Duration: defaultDuration(), Status: BeatStatusRest, Dynamics: mezzoPiano, Effect: BeatEffects{Hairpin: HairpinCrescendo}}
	chord := &Chord{Name: "Dm", Strings: []int8{-1, 0, 2, 3, 1, 0}}
	chordBeat := Beat{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: fortissimo, Effect: BeatEffects{Hairpin: HairpinDiminuendo, Chord: chord}, Notes: []Note{{Value: 1, String: 1, Kind: NoteTypeNormal, Velocity: fortissimo, DurationPercent: 1}}}
	track.Measures[0].Voices[0].Beats = []Beat{*beat, rest, chordBeat}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("effects combination export = %v, %#v", err, report.Entries)
	}
	gotSong, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotBeats := gotSong.Tracks[0].Measures[0].Voices[0].Beats
	gotNote := gotBeats[0].Notes[0]
	run.Field("NoteEffect.Harmonic", gotNote.Effect.Harmonic != nil, true)
	run.Field("HarmonicEffect.Kind", gotNote.Effect.Harmonic.Kind, HarmonicTypeArtificial)
	run.Field("HarmonicEffect.Fret", *gotNote.Effect.Harmonic.Fret, harmonicFret)
	run.Field("HarmonicEffect.FretFloat", *gotNote.Effect.Harmonic.FretFloat, harmonicExact)
	run.Field("NoteEffect.Bend", gotNote.Effect.Bend != nil, true)
	run.Field("BendEffect.Points", gotNote.Effect.Bend.Points, note.Effect.Bend.Points)
	run.Field("NoteEffect.Slides", gotNote.Effect.Slides, []SlideType{SlideShiftSlideTo})
	run.Field("NoteEffect.Hammer", gotNote.Effect.Hammer, true)
	run.Field("Beat.Status", gotBeats[1].Status, BeatStatusRest)
	run.Field("Beat.Dynamics", []int16{gotBeats[1].Dynamics, gotBeats[2].Dynamics}, []int16{mezzoPiano, fortissimo})
	run.Field("BeatEffects.Hairpin", []Hairpin{gotBeats[1].Effect.Hairpin, gotBeats[2].Effect.Hairpin}, []Hairpin{HairpinCrescendo, HairpinDiminuendo})
	run.Field("BeatEffects.Chord", gotBeats[2].Effect.Chord.Name, "Dm")
}

func TestConformanceLyricsPlaybackAutomation(t *testing.T) {
	runConformanceLyricsPlaybackAutomation(newConformanceRun(t))
}

func runConformanceLyricsPlaybackAutomation(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	track.Settings.Notation = true
	track.Lyrics = []TrackLyricLine{{Offset: 2, Text: "M22 lyric"}}
	track.Sounds = []TrackSound{{Name: "Clean", Label: "C", Path: "factory/clean", Role: "main", Program: 27}, {Name: "Lead", Label: "L", Path: "factory/lead", Role: "solo", Program: 81}}
	track.SoundAutomations = []SoundAutomation{{Bar: 1, Position: 0.5, Sound: 1}}
	track.Mute = true
	track.Solo = true
	song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 1, Position: 0.75, Value: 0.625, Linear: true}}
	run.Field("Track.Lyrics", track.Lyrics, []TrackLyricLine{{Offset: 2, Text: "M22 lyric"}})
	run.Field("Track.Sounds", track.Sounds, []TrackSound{{Name: "Clean", Label: "C", Path: "factory/clean", Role: "main", Program: 27}, {Name: "Lead", Label: "L", Path: "factory/lead", Role: "solo", Program: 81}})
	run.Field("Track.SoundAutomations", track.SoundAutomations, []SoundAutomation{{Bar: 1, Position: 0.5, Sound: 1}})
	run.Field("Track.Mute", track.Mute, true)
	run.Field("Track.Solo", track.Solo, true)
	run.Field("Song.VolumeAutomations", song.VolumeAutomations, []VolumeAutomation{{Track: 0, Bar: 1, Position: 0.75, Value: 0.625, Linear: true}})
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.normalize.playback-state", "gp8.omit.volume-automations"} {
		if !hasExportCode(report, code) {
			t.Fatalf("playback combination report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.playback-state", "gp8.omit.volume-automations"}}})
	if err != nil {
		t.Fatal(err)
	}
	gotSong, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := gotSong.Tracks[0]
	run.Wire("gpifTrack.Lyrics", got.Lyrics, track.Lyrics)
	run.Wire("gpifTrack.Automations", got.SoundAutomations, track.SoundAutomations)
	run.Wire("gpifTrack.PlaybackState", [2]bool{got.Mute, got.Solo}, [2]bool{true, false})
	if len(gotSong.VolumeAutomations) != 0 {
		t.Fatal("omitted volume automation appeared in GP8 output")
	}
}

func TestConformanceBackingSyncTempo(t *testing.T) {
	runConformanceBackingSyncTempo(newConformanceRun(t))
}

func runConformanceBackingSyncTempo(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.InitialTempo = KnownSourceValue(BPM(132.5))
	song.Tempo = 133
	song.TempoAutomations = []TempoAutomation{{Bar: 0, Position: 0, Tempo: 132.5}}
	barPosition, err := NewBarPositionFromFloat64(0.5)
	if err != nil {
		t.Fatal(err)
	}
	audioFrame, err := NewAudioFrame(48510)
	if err != nil {
		t.Fatal(err)
	}
	wantBacking := BackingTrack{Name: "Take", Source: "embedded", AssetID: "a", OriginalFilePath: "take.wav", OriginalFileSHA1: "abc", EmbeddedFilePath: "Content/a.wav", AudioData: []byte{1, 2, 3}, FramePadding: 4410, Enabled: true}
	wantSync := SyncPoint{Bar: 1, Position: 0.5, BarPosition: barPosition, BarOccurrence: 2, FrameOffset: 48510, AudioFrame: audioFrame, MediaTimeMS: 1000, ModifiedTempo: 132.5, OriginalTempo: 120, Linear: true, Visible: true}
	song.BackingTrack = &wantBacking
	song.SyncPoints = []SyncPoint{wantSync}
	run.Field("Song.BackingTrack", *song.BackingTrack, wantBacking)
	run.Field("Song.SyncPoints", song.SyncPoints, []SyncPoint{wantSync})
	run.Field("SyncPoint.BarOccurrence", song.SyncPoints[0].BarOccurrence, 2)
	run.Field("SyncPoint.BarPosition", song.SyncPoints[0].BarPosition, barPosition)
	run.Field("SyncPoint.AudioFrame", song.SyncPoints[0].AudioFrame, audioFrame)
	run.Field("Song.InitialTempo", song.InitialTempo.Value, BPM(132.5))
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.backing-track", "gp8.omit.sync-points"} {
		if !hasExportCode(report, code) {
			t.Fatalf("backing combination report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.backing-track", "gp8.omit.sync-points"}}})
	if err != nil {
		t.Fatal(err)
	}
	wire := extractAutomationWireDocument(t, data)
	run.Wire("gpifAutomation.Value", conformanceAutomationStrings(wire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Value }), []string{"132.5 2"})
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.BackingTrack != nil || len(got.SyncPoints) != 0 || got.InitialTempo.Value != BPM(132.5) {
		t.Fatalf("backing combination round trip = %#v", got)
	}
}

func TestConformanceSelectiveAllowlist(t *testing.T) {
	runConformanceSelectiveAllowlist(newConformanceRun(t))
}

func runConformanceSelectiveAllowlist(run *conformanceRun) {
	t := run.t
	song := semanticExportProbeSong(t)
	song.Name = "M22 preserved"
	song.Writer = "omitted writer"
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.DurationPercent = 0.5
	run.Field("Song.Writer", song.Writer, "omitted writer")
	run.Field("Note.DurationPercent", note.DurationPercent, float32(0.5))
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.writer", "gp8.omit.note-duration-percent"} {
		if !hasExportCode(report, code) {
			t.Fatalf("selective report = %#v, want %s", report.Entries, code)
		}
	}
	for _, allowed := range []string{"gp8.omit.writer", "gp8.omit.note-duration-percent"} {
		data, strictReport, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{allowed}}})
		var lossErr *ExportLossError
		if len(data) != 0 || !errors.As(err, &lossErr) || !reflect.DeepEqual(strictReport, report) || len(lossErr.Entries) != 1 || lossErr.Entries[0].Code == allowed {
			t.Fatalf("selective %s export = %d bytes, %#v, %#v, %v", allowed, len(data), strictReport, lossErr, err)
		}
	}
	data, allowedReport, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.writer", "gp8.omit.note-duration-percent"}}})
	if err != nil || !reflect.DeepEqual(allowedReport, report) {
		t.Fatalf("fully allowlisted export = %d bytes, %#v, %v", len(data), allowedReport, err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Song.Name", song.Name, "M22 preserved")
	run.Wire("gpifScore.Title", got.Name, "M22 preserved")
	if got.Writer != "" || got.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].DurationPercent != 1 {
		t.Fatalf("selective omissions round trip = %#v", got)
	}
}

func TestConformanceLegacyEditAuthority(t *testing.T) {
	runConformanceLegacyEditAuthority(newConformanceRun(t))
}

func runConformanceLegacyEditAuthority(run *conformanceRun) {
	t := run.t
	song, err := Parse(mustReadFixture(t, "testdata/gp5/notes.gp5"))
	if err != nil {
		t.Fatal(err)
	}
	oldInitial := song.InitialTempo
	song.Tempo = 90
	if finalizeErr := FinalizeSong(song); finalizeErr != nil {
		t.Fatal(finalizeErr)
	}
	run.Field("Song.Tempo", song.Tempo, int16(90))
	run.Field("Song.InitialTempo", song.InitialTempo, oldInitial)
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if !hasExportCode(report, "gp8.normalize.tempo-compatibility") {
		t.Fatalf("legacy edit preflight = %#v, want tempo compatibility decision", report.Entries)
	}
	data, exported, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil || !reflect.DeepEqual(exported, report) {
		t.Fatalf("legacy edit export = %d bytes, %#v, %v", len(data), exported, err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Wire("gpifAutomation.Value", got.InitialTempo.Value, BPM(90))
	if got.Tempo != 90 || got.InitialTempo.Value != BPM(90) {
		t.Fatalf("stale compatibility reversed legacy edit: %d/%#v", got.Tempo, got.InitialTempo)
	}
}
