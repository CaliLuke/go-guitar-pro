// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM15OrderedGraceExport(t *testing.T) {
	runSemanticMatrixM15OrderedGraceExport(newSemanticMatrixRun(t))
}

func runSemanticMatrixM15OrderedGraceExport(run *semanticMatrixRun) {
	t := run.t
	song := semanticExportProbeSong(t)
	track := &song.Tracks[0]
	track.Measures[0].Voices = []Voice{{Beats: []Beat{
		{
			Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte,
			Notes: []Note{
				m15GraceNote(5, 1, []GraceEffect{
					m15Grace(2, DurationSixteenth, MinVelocity+VelocityIncrement*4, false, 0, GraceEffectTransitionNone, false),
					m15Grace(3, DurationThirtySecond, Forte, true, 1, GraceEffectTransitionSlide, true),
					m15Grace(4, 64, MinVelocity+VelocityIncrement*6, false, 2, GraceEffectTransitionHammer, false),
				}),
				m15GraceNote(1, 6, []GraceEffect{
					m15Grace(1, DurationThirtySecond, Forte, true, 1, GraceEffectTransitionNone, false),
				}),
				m15GraceNote(2, 2, []GraceEffect{
					m15Grace(1, DurationSixteenth, MinVelocity+VelocityIncrement*4, false, 0, GraceEffectTransitionNone, false),
					m15Grace(3, 64, MinVelocity+VelocityIncrement*6, false, 2, GraceEffectTransitionNone, false),
				}),
			},
		},
		{Duration: defaultDuration(), Status: BeatStatusRest},
	}}, {Beats: []Beat{
		{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{m15GraceNote(3, 3, []GraceEffect{m15Grace(1, DurationThirtySecond, Forte, false, 0, GraceEffectTransitionNone, false)})}},
		{Duration: defaultDuration(), Status: BeatStatusRest},
	}}}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("grace baseline diagnostics = %#v", diagnostics)
	}
	notes := track.Measures[0].Voices[0].Beats[0].Notes
	noGrace := NoteEffect{}
	run.Field("NoteEffect.Graces", []int{len(notes[0].Effect.Graces), len(notes[1].Effect.Graces), len(notes[2].Effect.Graces), len(noGrace.Graces)}, []int{3, 1, 2, 0})
	wantPrimary := notes[0].Effect.Graces
	run.Preserved("GraceEffect.Duration", m15GraceDurations(wantPrimary), []uint8{DurationSixteenth, DurationThirtySecond, 64})
	run.Normalized("GraceEffect.Fret", m15GraceFrets(wantPrimary), []int8{2, 3, 4})
	run.Preserved("GraceEffect.ExactFret", m15GraceExactFrets(wantPrimary), []Fret{2, 3, 4})
	run.Preserved("GraceEffect.RawFret", []*int8{wantPrimary[0].RawFret, wantPrimary[1].RawFret, wantPrimary[2].RawFret}, []*int8{nil, nil, nil})
	run.Preserved("GraceEffect.PercussionArticulation", m15GraceArticulations(wantPrimary), []int{0, 0, 0})
	run.Preserved("GraceEffect.HasPercussionArticulation", m15GraceArticulationPresence(wantPrimary), []bool{false, false, false})
	run.Preserved("GraceEffect.IsDead", m15GraceDead(wantPrimary), []bool{false, true, false})
	run.Preserved("GraceEffect.IsOnBeat", m15GraceOnBeat(wantPrimary), []bool{false, true, false})
	run.Preserved("GraceEffect.Sequence", m15GraceSequences(wantPrimary), []uint8{0, 1, 2})
	run.Normalized("GraceEffect.Transition", m15GraceTransitions(wantPrimary), []GraceEffectTransition{GraceEffectTransitionNone, GraceEffectTransitionSlide, GraceEffectTransitionHammer})
	run.Preserved("GraceEffect.Velocity", m15GraceVelocities(wantPrimary), []int16{MinVelocity + VelocityIncrement*4, Forte, MinVelocity + VelocityIncrement*6})

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("canonical grace export = %v, %#v", err, report.Entries)
	}
	wire := extractM15Wire(t, data)
	run.Wire("gpifBeat.GraceNotes", wire.graceKinds, []string{"BeforeBeat", "OnBeat", "BeforeBeat", "BeforeBeat"})
	run.Wire("gpifBeat.Dynamic", wire.graceDynamics, []string{"MF", "F", "FF", "F"})
	run.Wire("gpifBeat.Notes", wire.graceNoteCounts, []int{2, 2, 2, 1})
	run.Wire("gpifBeat.Rhythm", wire.graceRhythmRefs, []string{"0", "1", "2", "1"})
	run.Wire("gpifRhythmRef.Ref", wire.graceRhythmRefs, []string{"0", "1", "2", "1"})
	run.Wire("gpifRhythm.NoteValue", wire.graceNoteValues, []string{"16th", "32nd", "64th", "32nd"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotVoice := roundTrip.Tracks[0].Measures[0].Voices[0]
	gotNotes := gotVoice.Beats[0].Notes
	run.Field("NoteEffect.Graces", []int{len(gotNotes[0].Effect.Graces), len(gotNotes[1].Effect.Graces), len(gotNotes[2].Effect.Graces), len(gotVoice.Beats[1].Notes)}, []int{3, 1, 2, 0})
	secondVoice := roundTrip.Tracks[0].Measures[0].Voices[1]
	if len(secondVoice.Beats) != 2 || len(secondVoice.Beats[0].Notes[0].Effect.Graces) != 1 || secondVoice.Beats[1].Status != BeatStatusRest {
		t.Fatalf("second grace voice = %#v", secondVoice)
	}
	gotPrimary := gotNotes[0].Effect.Graces
	run.Field("GraceEffect.Duration", m15GraceDurations(gotPrimary), []uint8{DurationSixteenth, DurationThirtySecond, 64})
	run.Field("GraceEffect.Fret", m15GraceFrets(gotPrimary), []int8{2, 3, 4})
	run.Field("GraceEffect.ExactFret", m15GraceExactFrets(gotPrimary), []Fret{2, 3, 4})
	run.Field("GraceEffect.IsDead", m15GraceDead(gotPrimary), []bool{false, true, false})
	run.Field("GraceEffect.IsOnBeat", m15GraceOnBeat(gotPrimary), []bool{false, true, false})
	run.Dispatch("parseGPIFWithContext:b.GraceNotes", m15GraceOnBeat(gotPrimary), []bool{false, true, false})
	run.Field("GraceEffect.Sequence", m15GraceSequences(gotPrimary), []uint8{0, 1, 2})
	run.Field("GraceEffect.Transition", m15GraceTransitions(gotPrimary), []GraceEffectTransition{GraceEffectTransitionNone, GraceEffectTransitionSlide, GraceEffectTransitionHammer})
	run.Enum("GraceEffectTransition.GraceEffectTransitionNone", gotPrimary[0].Transition, GraceEffectTransitionNone)
	run.Enum("GraceEffectTransition.GraceEffectTransitionSlide", gotPrimary[1].Transition, GraceEffectTransitionSlide)
	run.Enum("GraceEffectTransition.GraceEffectTransitionHammer", gotPrimary[2].Transition, GraceEffectTransitionHammer)
	run.Field("GraceEffect.Velocity", m15GraceVelocities(gotPrimary), []int16{MinVelocity + VelocityIncrement*4, Forte, MinVelocity + VelocityIncrement*6})

	reused := rewriteConformanceGPIF(t, data, func(gpif string) string {
		return strings.Replace(gpif, "<Beats>0 1 2 3 4</Beats>", "<Beats>0 1 2 3 4 0</Beats>", 1)
	})
	orphanSong, err := Parse(reused)
	if err != nil {
		t.Fatal(err)
	}
	orphanVoice := orphanSong.Tracks[0].Measures[0].Voices[0]
	if len(orphanVoice.Beats) != 3 || !orphanVoice.Beats[2].isGrace || len(orphanVoice.Beats[2].Notes) != 2 {
		t.Fatalf("boundary orphan beats = %#v", orphanVoice.Beats)
	}
	attached := &orphanVoice.Beats[0].Notes[0].Effect.Graces[0]
	orphan := &orphanVoice.Beats[2].Notes[0]
	if len(orphan.Effect.Graces) != 0 {
		t.Fatalf("orphan grace note has nested grace effects = %#v", orphan.Effect.Graces)
	}
	orphan.Value = 9
	if *attached.ExactFret != 2 {
		t.Fatal("mutating orphan grace changed attached occurrence")
	}
	if *orphanVoice.Beats[2].Start != *orphanVoice.Beats[1].Start+DurationQuarterTime {
		t.Fatalf("orphan start = %d, want rest end %d", *orphanVoice.Beats[2].Start, *orphanVoice.Beats[1].Start+DurationQuarterTime)
	}

	invalid := rewriteConformanceGPIF(t, data, func(gpif string) string {
		return strings.Replace(gpif, "<GraceNotes>BeforeBeat</GraceNotes>", "<GraceNotes>FutureGrace</GraceNotes>", 1)
	})
	parsed, err := ParseWithOptions(invalid, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(parsed.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == "GPIF.Beat.GraceNotes.InvalidValue" }) {
		t.Fatalf("diagnostics = %#v, want grace discriminator receipt", parsed.Diagnostics)
	}
	if _, strictErr := ParseWithOptions(invalid, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnsupportedFeature}}); strictErr == nil {
		t.Fatal("selected strict parse accepted an unsupported grace discriminator")
	}
}

func TestSemanticMatrixM15GraceAuthorityAndLoss(t *testing.T) {
	runSemanticMatrixM15GraceAuthorityAndLoss(newSemanticMatrixRun(t))
}

func runSemanticMatrixM15GraceAuthorityAndLoss(run *semanticMatrixRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	rawFret := int8(4)
	exactFret := Fret(5)
	note.Effect.Graces = []GraceEffect{{
		Duration:   3,
		Fret:       4,
		ExactFret:  &exactFret,
		RawFret:    &rawFret,
		IsOnBeat:   false,
		Sequence:   0,
		Transition: GraceEffectTransitionBend,
		Velocity:   0,
	}}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	run.Enum("GraceEffectTransition.GraceEffectTransitionBend", hasExportCode(report, "gp8.omit.grace-bend-transition"), true)
	wantCodes := []string{
		"gp8.normalize.grace-duration",
		"gp8.normalize.grace-fret-authority",
		"gp8.omit.grace-raw-fret",
		"gp8.normalize.grace-velocity",
		"gp8.omit.grace-bend-transition",
	}
	for _, code := range wantCodes {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict grace export = %d bytes, %v", len(strictData), strictErr)
	}
	data, exported, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(reportCodes(exported), reportCodes(report)) {
		t.Fatalf("preflight codes = %v, export codes = %v", reportCodes(report), reportCodes(exported))
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces
	if len(got) != 1 {
		t.Fatalf("graces = %#v, want one", got)
	}
	run.Field("GraceEffect.Duration", got[0].Duration, DurationThirtySecond)
	run.Field("GraceEffect.Fret", got[0].Fret, int8(5))
	run.Field("GraceEffect.ExactFret", *got[0].ExactFret, Fret(5))
	run.Field("GraceEffect.RawFret", got[0].RawFret, (*int8)(nil))
	run.Field("GraceEffect.Velocity", got[0].Velocity, DefaultVelocity)
	run.Field("GraceEffect.Transition", got[0].Transition, GraceEffectTransitionNone)

	zeroSong := semanticExportProbeSong(t)
	zero := int8(0)
	zeroExact := Fret(0)
	zeroGrace := GraceEffect{Duration: DurationThirtySecond, Fret: 0, ExactFret: &zeroExact, RawFret: &zero, Velocity: Forte}
	zeroSong.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces = []GraceEffect{zeroGrace}
	zeroReport := PreflightExport(zeroSong, ExportFormatGP8, ExportOptions{})
	if !hasExportCode(zeroReport, "gp8.omit.grace-raw-fret") || hasExportCode(zeroReport, "gp8.normalize.grace-fret-authority") {
		t.Fatalf("explicit zero grace report = %#v", zeroReport.Entries)
	}
	zeroData, _, err := ExportWithReport(zeroSong, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.grace-raw-fret"}}})
	if err != nil {
		t.Fatal(err)
	}
	zeroRoundTrip, err := Parse(zeroData)
	if err != nil {
		t.Fatal(err)
	}
	zeroGot := zeroRoundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[0]
	run.Field("GraceEffect.Fret", zeroGot.Fret, int8(0))
	run.Field("GraceEffect.ExactFret", *zeroGot.ExactFret, Fret(0))
	run.Field("GraceEffect.RawFret", zeroGot.RawFret, (*int8)(nil))
}

func TestSemanticMatrixM15BinaryGraceSource(t *testing.T) {
	runSemanticMatrixM15BinaryGraceSource(newSemanticMatrixRun(t))
}

func runSemanticMatrixM15BinaryGraceSource(run *semanticMatrixRun) {
	t := run.t
	song := parseTestFixture(t, "testdata/gp5/motherload-percussion-grace.gp5")
	var found *GraceEffect
	var ownerValue int16
	for trackIndex := range song.Tracks {
		for measureIndex := range song.Tracks[trackIndex].Measures {
			for voiceIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices {
				for beatIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats {
					for noteIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						note := &song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex]
						if len(note.Effect.Graces) != 0 {
							found = &note.Effect.Graces[0]
							ownerValue = note.Value
							break
						}
					}
				}
			}
		}
	}
	if found == nil || found.RawFret == nil {
		t.Fatal("GP5 fixture has no source grace")
	}
	run.Field("GraceEffect.RawFret", *found.RawFret, int8(0))
	run.Field("GraceEffect.Fret", found.Fret, int8(ownerValue))
	run.Field("GraceEffect.ExactFret", found.ExactFret, (*Fret)(nil))
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if !hasExportCode(report, "gp8.omit.grace-raw-fret") {
		t.Fatalf("GP5 grace report = %#v, want raw source-fret omission", report.Entries)
	}
}

func TestSemanticMatrixM15CombinedGraceEffects(t *testing.T) {
	runSemanticMatrixM15CombinedGraceEffects(newSemanticMatrixRun(t))
}

func runSemanticMatrixM15CombinedGraceEffects(run *semanticMatrixRun) {
	t := run.t
	t.Run("grace tie and bend", func(t *testing.T) {
		song := semanticExportProbeSong(t)
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		note.TieOrigin = true
		note.Effect.Bend = &BendEffect{Points: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 2}}}
		note.Effect.Graces = []GraceEffect{m15Grace(2, DurationThirtySecond, Forte, false, 0, GraceEffectTransitionHammer, false)}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("combined pitched export = %v, %#v", err, report.Entries)
		}
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		if !got.TieOrigin || got.Effect.Bend == nil || !slices.Equal(got.Effect.Bend.Points, note.Effect.Bend.Points) || len(got.Effect.Graces) != 1 {
			t.Fatalf("combined pitched note = %#v", got)
		}
		run.Field("NoteEffect.Graces", len(got.Effect.Graces), 1)
		run.Field("GraceEffect.ExactFret", *got.Effect.Graces[0].ExactFret, Fret(2))
		run.Field("GraceEffect.Transition", got.Effect.Graces[0].Transition, GraceEffectTransitionHammer)
	})

	t.Run("percussion grace identity and mute", func(t *testing.T) {
		song := syntheticGP8Song()
		track := &song.Tracks[0]
		track.PercussionArticulations = []PercussionArticulation{
			{ElementName: "Kit", ElementType: "percussion", Name: "Main", NoteheadDefault: "noteheadBlack", InputMIDINumbers: []int{38}, OutputMIDINumber: 38},
			{ElementName: "Kit", ElementType: "percussion", Name: "Muted grace", NoteheadDefault: "noteheadXBlack", InputMIDINumbers: []int{42}, OutputMIDINumber: 42},
		}
		note := &track.Measures[0].Voices[0].Beats[0].Notes[0]
		note.PercussionArticulation = 0
		note.HasPercussionArticulation = true
		note.Effect.Graces = []GraceEffect{{
			Duration: DurationThirtySecond, Fret: 38, ExactFret: ptrTo(Fret(38)),
			PercussionArticulation: 1, HasPercussionArticulation: true,
			IsDead: true, IsOnBeat: true, Sequence: 0, Velocity: Forte,
		}}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("combined percussion export = %v, %#v", err, report.Entries)
		}
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces
		if len(got) != 1 {
			t.Fatalf("percussion graces = %#v", got)
		}
		run.Field("GraceEffect.PercussionArticulation", got[0].PercussionArticulation, 1)
		run.Field("GraceEffect.HasPercussionArticulation", got[0].HasPercussionArticulation, true)
		run.Field("GraceEffect.IsDead", got[0].IsDead, true)
		run.Field("GraceEffect.IsOnBeat", got[0].IsOnBeat, true)
	})
}

func m15GraceNote(fret int16, guitarString int8, graces []GraceEffect) Note {
	return Note{
		Value: fret, String: guitarString, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1,
		Effect: NoteEffect{Graces: graces, LeftHandFinger: FingeringOpen, RightHandFinger: FingeringOpen},
	}
}

func m15Grace(fret int8, duration uint8, velocity int16, onBeat bool, sequence uint8, transition GraceEffectTransition, dead bool) GraceEffect {
	exact := Fret(fret)
	return GraceEffect{Duration: duration, Fret: fret, ExactFret: &exact, IsDead: dead, IsOnBeat: onBeat, Sequence: sequence, Transition: transition, Velocity: velocity}
}

func m15GraceDurations(graces []GraceEffect) []uint8 {
	result := make([]uint8, len(graces))
	for index := range graces {
		result[index] = graces[index].Duration
	}
	return result
}

func m15GraceFrets(graces []GraceEffect) []int8 {
	result := make([]int8, len(graces))
	for index := range graces {
		result[index] = graces[index].Fret
	}
	return result
}

func m15GraceExactFrets(graces []GraceEffect) []Fret {
	result := make([]Fret, len(graces))
	for index := range graces {
		if graces[index].ExactFret != nil {
			result[index] = *graces[index].ExactFret
		}
	}
	return result
}

func m15GraceArticulations(graces []GraceEffect) []int {
	result := make([]int, len(graces))
	for index := range graces {
		result[index] = graces[index].PercussionArticulation
	}
	return result
}

func m15GraceArticulationPresence(graces []GraceEffect) []bool {
	result := make([]bool, len(graces))
	for index := range graces {
		result[index] = graces[index].HasPercussionArticulation
	}
	return result
}

func m15GraceDead(graces []GraceEffect) []bool {
	result := make([]bool, len(graces))
	for index := range graces {
		result[index] = graces[index].IsDead
	}
	return result
}

func m15GraceOnBeat(graces []GraceEffect) []bool {
	result := make([]bool, len(graces))
	for index := range graces {
		result[index] = graces[index].IsOnBeat
	}
	return result
}

func m15GraceSequences(graces []GraceEffect) []uint8 {
	result := make([]uint8, len(graces))
	for index := range graces {
		result[index] = graces[index].Sequence
	}
	return result
}

func m15GraceTransitions(graces []GraceEffect) []GraceEffectTransition {
	result := make([]GraceEffectTransition, len(graces))
	for index := range graces {
		result[index] = graces[index].Transition
	}
	return result
}

func m15GraceVelocities(graces []GraceEffect) []int16 {
	result := make([]int16, len(graces))
	for index := range graces {
		result[index] = graces[index].Velocity
	}
	return result
}

type m15Wire struct {
	graceKinds      []string
	graceDynamics   []string
	graceNoteCounts []int
	graceRhythmRefs []string
	graceNoteValues []string
}

func extractM15Wire(t *testing.T, data []byte) m15Wire {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Beats struct {
			Items []struct {
				Dynamic    string `xml:"Dynamic"`
				GraceNotes string `xml:"GraceNotes"`
				Notes      string `xml:"Notes"`
				Rhythm     struct {
					Ref string `xml:"ref,attr"`
				} `xml:"Rhythm"`
			} `xml:"Beat"`
		} `xml:"Beats"`
		Rhythms struct {
			Items []struct {
				ID        string `xml:"id,attr"`
				NoteValue string `xml:"NoteValue"`
			} `xml:"Rhythm"`
		} `xml:"Rhythms"`
	}
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	rhythmValues := make(map[string]string, len(document.Rhythms.Items))
	for _, rhythm := range document.Rhythms.Items {
		rhythmValues[rhythm.ID] = rhythm.NoteValue
	}
	var result m15Wire
	for _, beat := range document.Beats.Items {
		if beat.GraceNotes == "" {
			continue
		}
		result.graceKinds = append(result.graceKinds, beat.GraceNotes)
		result.graceDynamics = append(result.graceDynamics, beat.Dynamic)
		result.graceNoteCounts = append(result.graceNoteCounts, len(strings.Fields(beat.Notes)))
		result.graceRhythmRefs = append(result.graceRhythmRefs, beat.Rhythm.Ref)
		result.graceNoteValues = append(result.graceNoteValues, rhythmValues[beat.Rhythm.Ref])
	}
	return result
}
