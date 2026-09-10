// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestExportPreflightReportsActualGP8Losses(t *testing.T) {
	clean := PreflightExport(syntheticGP8Song(), ExportFormatGP8, ExportOptions{})
	if len(clean.Entries) != 0 {
		t.Fatalf("clean preflight = %#v", clean.Entries)
	}

	song := syntheticGP8Song()
	song.BackingTrack = &BackingTrack{Name: "omitted audio", AudioData: []byte("audio")}
	song.SyncPoints = []SyncPoint{{Bar: 0}}
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Notes = append(beat.Notes, beat.Notes[0])
	beat.Notes[1].Velocity = beat.Notes[0].Velocity + 1

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.backing-track", "gp8.omit.sync-points", "gp8.normalize.note-velocity"} {
		if !hasExportReportEntry(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	velocity := exportReportEntry(report, "gp8.normalize.note-velocity")
	if velocity == nil || velocity.Location.Track != 0 || velocity.Location.Measure != 0 || velocity.Location.Beat != 0 {
		t.Fatalf("velocity entry location = %#v", velocity)
	}
}

func TestExportStrictLossPolicyUsesStableAllowlist(t *testing.T) {
	song := syntheticGP8Song()
	song.BackingTrack = &BackingTrack{Name: "omitted audio", AudioData: []byte("audio")}

	options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	data, report, err := ExportWithReport(song, ExportFormatGP8, options)
	var lossErr *ExportLossError
	if !errors.As(err, &lossErr) || len(data) != 0 || !hasExportReportEntry(report, "gp8.omit.backing-track") {
		t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report, err)
	}

	options.LossPolicy.AllowedCodes = []string{"gp8.omit.backing-track"}
	data, report, err = ExportWithReport(song, ExportFormatGP8, options)
	if err != nil || len(data) == 0 || !hasExportReportEntry(report, "gp8.omit.backing-track") {
		t.Fatalf("allowed export = %d bytes, %#v, %v", len(data), report, err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.BackingTrack != nil {
		t.Fatalf("independent output retained omitted backing track: %#v", roundTrip.BackingTrack)
	}
}

func TestExportStrictLossPolicyReportsVelocityQuantization(t *testing.T) {
	song := syntheticGP8Song()
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	for index := range beat.Notes {
		beat.Notes[index].Velocity = 100
	}

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if !hasExportReportEntry(report, "gp8.normalize.note-velocity") {
		t.Fatalf("velocity preflight = %#v, want quantization entry", report.Entries)
	}
	_, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	var lossErr *ExportLossError
	if !errors.As(err, &lossErr) {
		t.Fatalf("strict velocity export error = %v, want ExportLossError", err)
	}
}

func TestExportPreflightRejectsUnsupportedBeatDuration(t *testing.T) {
	song := syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Duration.Value = 3

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	entry := exportReportEntry(report, "gp8.reject.score.beat.duration")
	if entry == nil || entry.Disposition != ExportDispositionRejected {
		t.Fatalf("duration preflight = %#v, want rejected score entry", report.Entries)
	}
	if _, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{}); err == nil {
		t.Fatal("export accepted a duration rejected by serialization")
	}
}

func TestExportPreflightLocatesEachOmittedAutomation(t *testing.T) {
	song := syntheticGP8Song()
	song.SyncPoints = []SyncPoint{{Bar: 0}, {Bar: 1}}
	song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0}, {Track: 0, Bar: 1}}

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	want := map[string][]ScoreLocation{
		"gp8.omit.sync-points": {
			{Measure: 0},
			{Measure: 1},
		},
		"gp8.omit.volume-automations": {
			{Track: 0, Measure: 0},
			{Track: 0, Measure: 1},
		},
	}
	for code, locations := range want {
		var got []ScoreLocation
		for _, entry := range report.Entries {
			if entry.Code == code {
				got = append(got, entry.Location)
			}
		}
		if !reflect.DeepEqual(got, locations) {
			t.Errorf("%s locations = %#v, want %#v", code, got, locations)
		}
	}
}

func TestGP8StrictExportReportsUnsupportedBeatAndNoteEffects(t *testing.T) {
	tests := []struct {
		name string
		code string
		set  func(*Beat, *Note)
	}{
		{name: "beat mix table", code: "gp8.omit.beat-mix-table-change", set: func(beat *Beat, _ *Note) {
			beat.Effect.MixTableChange = &MixTableChange{}
		}},
		{name: "rasgueado", code: "gp8.omit.rasgueado", set: func(beat *Beat, _ *Note) {
			beat.Effect.HasRasgueado = true
		}},
		{name: "pick stroke", code: "gp8.omit.pick-stroke", set: func(beat *Beat, _ *Note) {
			beat.Effect.PickStroke = BeatStrokeDirectionUp
		}},
		{name: "slap effect", code: "gp8.omit.slap-effect", set: func(beat *Beat, _ *Note) {
			beat.Effect.SlapEffect = SlapEffectSlapping
		}},
		{name: "beat vibrato", code: "gp8.omit.beat-vibrato", set: func(beat *Beat, _ *Note) {
			beat.Effect.Vibrato = true
		}},
		{name: "stroke duration", code: "gp8.normalize.stroke-duration", set: func(beat *Beat, _ *Note) {
			beat.Effect.Stroke = BeatStroke{Direction: BeatStrokeDirectionUp, Duration: NoteValue(DurationSixteenth)}
		}},
		{name: "tremolo picking", code: "gp8.omit.tremolo-picking", set: func(_ *Beat, note *Note) {
			note.Effect.TremoloPicking = &TremoloPickingEffect{Duration: defaultDuration()}
		}},
		{name: "left fingering", code: "gp8.omit.left-hand-fingering", set: func(_ *Beat, note *Note) {
			note.Effect.LeftHandFinger = FingeringIndex
		}},
		{name: "right fingering", code: "gp8.omit.right-hand-fingering", set: func(_ *Beat, note *Note) {
			note.Effect.RightHandFinger = FingeringMiddle
		}},
		{name: "bend summary", code: "gp8.omit.bend-summary", set: func(_ *Beat, note *Note) {
			note.Effect.Bend = &BendEffect{Kind: BendTypeBend, Value: 2, Points: []BendPoint{{}, {Position: 12, Value: 2}}}
		}},
		{name: "bend point vibrato", code: "gp8.omit.bend-point-vibrato", set: func(_ *Beat, note *Note) {
			note.Effect.Bend = &BendEffect{Points: []BendPoint{{Vibrato: true}, {Position: 12, Value: 2}}}
		}},
		{name: "trill duration", code: "gp8.normalize.trill-duration", set: func(_ *Beat, note *Note) {
			note.Effect.Trill = &TrillEffect{Fret: 3, Duration: defaultDuration()}
		}},
		{name: "harmonic pitch", code: "gp8.omit.harmonic-pitch", set: func(_ *Beat, note *Note) {
			pitch := PitchClass{Note: "C"}
			note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeNatural, Pitch: &pitch}
		}},
		{name: "grace bend transition", code: "gp8.omit.grace-bend-transition", set: func(_ *Beat, note *Note) {
			note.Effect.Graces = []GraceEffect{{Transition: GraceEffectTransitionBend, Duration: DurationThirtySecond, Velocity: Forte}}
		}},
		{name: "oversized whammy curve", code: "gp8.omit.whammy-curve", set: func(beat *Beat, _ *Note) {
			beat.Effect.TremoloBar = &BendEffect{Points: make([]BendPoint, 5)}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := syntheticGP8Song()
			beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
			note := &beat.Notes[0]
			test.set(beat, note)

			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
				LossPolicy: ExportLossPolicy{RequirePreservation: true},
			})
			var lossErr *ExportLossError
			if len(data) != 0 || !errors.As(err, &lossErr) {
				t.Fatalf("strict export = %d bytes, %v, want ExportLossError", len(data), err)
			}
			entry := exportReportEntry(report, test.code)
			if entry == nil {
				t.Fatalf("report = %#v, want %s", report.Entries, test.code)
			}
			if entry.Location.Track != 0 || entry.Location.Measure != 0 || entry.Location.Voice != 0 || entry.Location.Beat != 1 {
				t.Fatalf("entry location = %#v", entry.Location)
			}
		})
	}
}

func TestGP8ExportRejectsSharedAuthoredInvariants(t *testing.T) {
	tests := []struct {
		name string
		code string
		set  func(*Song)
	}{
		{name: "half tuplet", code: "gp8.reject.score.beat.duration", set: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].Beats[1].Duration.TupletEnters = 3
			song.Tracks[0].Measures[0].Voices[0].Beats[1].Duration.TupletTimes = 0
		}},
		{name: "channel reference", code: "gp8.reject.score.track.channel-reference", set: func(song *Song) {
			song.Tracks[0].ChannelIndex = len(song.Channels)
		}},
		{name: "lyrics track reference", code: "gp8.reject.score.lyrics.track-reference", set: func(song *Song) {
			song.Lyrics = Lyrics{TrackIndex: len(song.Tracks), Lines: []LyricLine{{Text: "words", StartMeasureIndex: 0}}}
		}},
		{name: "lyrics measure reference", code: "gp8.reject.score.lyrics.measure-reference", set: func(song *Song) {
			song.Lyrics = Lyrics{TrackIndex: 0, Lines: []LyricLine{{Text: "words", StartMeasureIndex: len(song.MeasureHeaders)}}}
		}},
		{name: "negative capo", code: "gp8.reject.score.track.capo", set: func(song *Song) {
			song.Tracks[0].CapoFret = -1
		}},
		{name: "sound reference", code: "gp8.reject.score.sound-automation.reference", set: func(song *Song) {
			song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Sound: 4}}
		}},
		{name: "tempo automation", code: "gp8.reject.score.tempo-automation.value", set: func(song *Song) {
			song.TempoAutomations = []TempoAutomation{{Bar: 0, Tempo: math.NaN()}}
		}},
		{name: "volume automation", code: "gp8.reject.score.volume-automation.value", set: func(song *Song) {
			song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Position: 2, Value: math.NaN()}}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := syntheticGP8Song()
			test.set(song)
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if len(data) != 0 || err == nil {
				t.Fatalf("export = %d bytes, %v, want rejection", len(data), err)
			}
			entry := exportReportEntry(report, test.code)
			if entry == nil || entry.Disposition != ExportDispositionRejected {
				t.Fatalf("report = %#v, want %s rejection", report.Entries, test.code)
			}
		})
	}
}

func TestGP8ExportReconcilesSemanticAndLegacyTempo(t *testing.T) {
	t.Run("legacy edit wins", func(t *testing.T) {
		song, err := Parse(mustReadFixture(t, "testdata/gp5/notes.gp5"))
		if err != nil {
			t.Fatal(err)
		}
		song.Tempo = 90

		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !hasExportReportEntry(report, "gp8.normalize.tempo-compatibility") {
			t.Fatalf("tempo report = %#v, want compatibility decision", report.Entries)
		}
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if roundTrip.Tempo != 90 || roundTrip.InitialTempo.Value != BPM(90) {
			t.Fatalf("round-trip tempo = %d/%v, want 90", roundTrip.Tempo, roundTrip.InitialTempo)
		}
	})

	t.Run("semantic-only fractional tempo", func(t *testing.T) {
		song := syntheticGP8Song()
		song.Tempo = 0
		song.InitialTempo = KnownSourceValue(BPM(132.5))

		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatalf("semantic-only export report = %#v, error = %v", report.Entries, err)
		}
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if roundTrip.InitialTempo.State != SourceValueKnown || roundTrip.InitialTempo.Value != BPM(132.5) {
			t.Fatalf("round-trip initial tempo = %#v, want 132.5", roundTrip.InitialTempo)
		}
	})

	t.Run("strict conflict is refused", func(t *testing.T) {
		song := syntheticGP8Song()
		song.InitialTempo = KnownSourceValue(BPM(120.5))
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
			LossPolicy: ExportLossPolicy{RequirePreservation: true},
		})
		var lossErr *ExportLossError
		if len(data) != 0 || !errors.As(err, &lossErr) || !hasExportReportEntry(report, "gp8.normalize.tempo-compatibility") {
			t.Fatalf("strict conflict = %d bytes, %#v, %v", len(data), report.Entries, err)
		}
	})

	t.Run("legacy edit replaces stale opening automation", func(t *testing.T) {
		song := syntheticGP8Song()
		song.InitialTempo = KnownSourceValue(BPM(132))
		song.Tempo = 90
		song.TempoAutomations = []TempoAutomation{{Bar: 0, Tempo: 132}}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !hasExportReportEntry(report, "gp8.normalize.tempo-compatibility") {
			t.Fatalf("report = %#v, want tempo compatibility decision", report.Entries)
		}
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if roundTrip.Tempo != 90 {
			t.Fatalf("round-trip tempo = %d, want 90", roundTrip.Tempo)
		}
	})
}

func TestGP8StrictExportReportsScoreLyricsOmission(t *testing.T) {
	song := syntheticGP8Song()
	song.Lyrics = Lyrics{TrackIndex: 0, Lines: []LyricLine{{Text: "authored words"}}}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	var lossErr *ExportLossError
	if len(data) != 0 || !errors.As(err, &lossErr) || !hasExportReportEntry(report, "gp8.omit.score-lyrics") {
		t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
}

func TestGP8StrictExportCoversInspectedSemanticFields(t *testing.T) {
	t.Run("track capo", func(t *testing.T) {
		song := semanticExportProbeSong(t)
		song.Tracks[0].CapoFret = 2
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
			LossPolicy: ExportLossPolicy{RequirePreservation: true},
		})
		if err != nil || len(data) == 0 {
			t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
		}
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if roundTrip.Tracks[0].CapoFret != 2 {
			t.Fatalf("round-trip capo = %d, want 2", roundTrip.Tracks[0].CapoFret)
		}
	})

	t.Run("supported bend curves", func(t *testing.T) {
		curves := [][]BendPoint{
			{{Position: 0, Value: 0}, {Position: 12, Value: 2}},
			{{Position: 0, Value: 0}, {Position: 6, Value: 2}, {Position: 12, Value: 0}},
			{{Position: 0, Value: 0}, {Position: 3, Value: 2}, {Position: 9, Value: 2}, {Position: 12, Value: 0}},
		}
		for _, points := range curves {
			song := semanticExportProbeSong(t)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Bend = &BendEffect{Points: points}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
				LossPolicy: ExportLossPolicy{RequirePreservation: true},
			})
			if err != nil || len(data) == 0 {
				t.Fatalf("%d-point strict export = %d bytes, %#v, %v", len(points), len(data), report.Entries, err)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			bend := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Bend
			// GPIF can duplicate one coincident middle control to identify a
			// standard bend-release gesture. Compare the canonical public curve.
			if bend == nil || !reflect.DeepEqual(simplifyBendPoints(bend.Points), simplifyBendPoints(points)) {
				t.Fatalf("%d-point round-trip = %#v, want %#v", len(points), bend, points)
			}
		}
	})

	tests := []struct {
		name string
		code string
		set  func(*Song)
	}{
		{name: "note duration percent", code: "gp8.omit.note-duration-percent", set: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].DurationPercent = 0.5
		}},
		{name: "zero note duration percent", code: "gp8.omit.note-duration-percent", set: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].DurationPercent = 0
		}},
		{name: "chord barres", code: "gp8.omit.chord-barres", set: func(song *Song) {
			firstFret := uint8(3)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = &Chord{
				Name: "barre", FirstFret: &firstFret, Strings: []int8{3, 3, 4, 5, 5, 3},
				Barres: []Barre{{Fret: 3, Start: 1, End: 6}},
			}
		}},
		{name: "five point bend", code: "gp8.omit.bend-curve", set: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Bend = &BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 3, Value: 2}, {Position: 6, Value: 0},
				{Position: 9, Value: 2}, {Position: 12, Value: 0},
			}}
		}},
		{name: "unrepresentable bend initial hold", code: "gp8.normalize.bend-curve", set: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Bend = &BendEffect{Points: []BendPoint{
				{Position: 0, Value: 2}, {Position: 3, Value: 2}, {Position: 12, Value: 0},
			}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			test.set(song)
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
				LossPolicy: ExportLossPolicy{RequirePreservation: true},
			})
			var lossErr *ExportLossError
			if len(data) != 0 || !errors.As(err, &lossErr) || !hasExportReportEntry(report, test.code) {
				t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
		})
	}
}

func semanticExportProbeSong(t *testing.T) *Song {
	t.Helper()
	header := defaultMeasureHeader()
	measure := Measure{HeaderIndex: 0, Voices: []Voice{{Beats: []Beat{{
		Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte,
		Notes: []Note{{Value: 3, String: 1, DurationPercent: 1, Kind: NoteTypeNormal, Velocity: Forte}},
	}}}}}
	song := &Song{
		Tempo: 120, MeasureHeaders: []MeasureHeader{header},
		Channels: []MidiChannel{{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}},
		Tracks: []Track{{
			Name: "Guitar", ChannelIndex: 0, Visible: true,
			Settings: TrackSettings{Notation: true},
			Strings:  []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}, {Number: 3, Value: 55}, {Number: 4, Value: 50}, {Number: 5, Value: 45}, {Number: 6, Value: 40}},
			Measures: []Measure{measure},
		}},
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("probe song diagnostics = %#v", diagnostics)
	}
	return song
}

func hasExportReportEntry(report ExportReport, code string) bool {
	return exportReportEntry(report, code) != nil
}

func exportReportEntry(report ExportReport, code string) *ExportReportEntry {
	for index := range report.Entries {
		if report.Entries[index].Code == code {
			return &report.Entries[index]
		}
	}
	return nil
}
