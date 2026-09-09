// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"testing"
)

func TestSemanticMatrixM24OracleCorrectness(t *testing.T) {
	runSemanticMatrixM24OracleCorrectness(newSemanticMatrixRun(t))
}

func runSemanticMatrixM24OracleCorrectness(run *semanticMatrixRun) {
	run.t.Run("comparator sensitivity", testM24ComparatorSensitivity)
	run.t.Run("Go adapters", testM24GoAdapterContracts)
}

func testM24ComparatorSensitivity(t *testing.T) {
	omitted := map[string]any{}
	explicitZero := map[string]any{"value": 0}
	differences := semanticDifferences(omitted, explicitZero)
	if !reflect.DeepEqual(differences, []semanticDifference{{Path: "/value", AlphaTab: float64(0)}}) {
		t.Fatalf("omitted versus zero differences = %#v", differences)
	}

	baseline := m24OwnershipProbe(61, 73)
	swapped := m24OwnershipProbe(73, 61)
	differences = semanticDifferences(baseline, swapped)
	want := []semanticDifference{
		{Path: "/tracks/0/staves/0/bars/0/voices/0/beats/0/notes/0/midi", Go: float64(61), AlphaTab: float64(73)},
		{Path: "/tracks/0/staves/1/bars/0/voices/0/beats/0/notes/0/midi", Go: float64(73), AlphaTab: float64(61)},
	}
	if !reflect.DeepEqual(differences, want) {
		t.Fatalf("ownership swap differences = %#v, want %#v", differences, want)
	}
	selectedBaseline := selectConformanceFeatures(baseline, []string{"staff-ownership"})
	selectedSwapped := selectConformanceFeatures(swapped, []string{"staff-ownership"})
	if differences := semanticDifferences(selectedBaseline, selectedSwapped); len(differences) != 2 {
		t.Fatalf("staff-ownership projection erased a swap: %#v", differences)
	}

	regular := m24BeatProbe("none")
	ottava := m24BeatProbe("8va")
	regular = selectConformanceFeatures(regular, []string{"note-and-beat-semantics"})
	ottava = selectConformanceFeatures(ottava, []string{"note-and-beat-semantics"})
	if differences := semanticDifferences(regular, ottava); len(differences) != 1 {
		t.Fatalf("note projection erased an octave change: %#v", differences)
	}
}

func testM24GoAdapterContracts(t *testing.T) {
	checks := []struct {
		name string
		got  string
		want string
	}{
		{name: "treble clef", got: goClef(MeasureClefTreble), want: "treble"},
		{name: "bass clef", got: goClef(MeasureClefBass), want: "bass"},
		{name: "tenor clef", got: goClef(MeasureClefTenor), want: "tenor"},
		{name: "alto clef", got: goClef(MeasureClefAlto), want: "alto"},
		{name: "unknown clef", got: goClef(MeasureClef(99)), want: "unknown:99"},
		{name: "no triplet feel", got: goTripletFeel(TripletFeelNone), want: "none"},
		{name: "eighth triplet feel", got: goTripletFeel(TripletFeelEighth), want: "triplet-8th"},
		{name: "sixteenth triplet feel", got: goTripletFeel(TripletFeelSixteenth), want: "triplet-16th"},
		{name: "unknown triplet feel", got: goTripletFeel(TripletFeel(99)), want: "unknown:99"},
		{name: "empty beat", got: goBeatStatus(BeatStatusEmpty), want: "empty"},
		{name: "normal beat", got: goBeatStatus(BeatStatusNormal), want: "normal"},
		{name: "rest beat", got: goBeatStatus(BeatStatusRest), want: "rest"},
		{name: "unknown beat status", got: goBeatStatus(BeatStatus(99)), want: "unknown:99"},
		{name: "rest note", got: goNoteKind(NoteTypeRest), want: "rest"},
		{name: "normal note", got: goNoteKind(NoteTypeNormal), want: "normal"},
		{name: "tie note", got: goNoteKind(NoteTypeTie), want: "tie"},
		{name: "dead note", got: goNoteKind(NoteTypeDead), want: "dead"},
		{name: "unknown note kind", got: goNoteKind(NoteType(99)), want: "unknown:99"},
		{name: "no hairpin", got: goHairpin(HairpinNone), want: "none"},
		{name: "crescendo", got: goHairpin(HairpinCrescendo), want: "crescendo"},
		{name: "decrescendo", got: goHairpin(HairpinDiminuendo), want: "decrescendo"},
		{name: "unknown hairpin", got: goHairpin(Hairpin(99)), want: "unknown:99"},
		{name: "no octave", got: goOctave(OctaveNone), want: "none"},
		{name: "ottava", got: goOctave(OctaveOttava), want: "8va"},
		{name: "quindicesima", got: goOctave(OctaveQuindicesima), want: "15ma"},
		{name: "ottava bassa", got: goOctave(OctaveOttavaBassa), want: "8vb"},
		{name: "quindicesima bassa", got: goOctave(OctaveQuindicesimaBassa), want: "15mb"},
		{name: "unknown octave", got: goOctave(Octave(99)), want: "unknown:99"},
		{name: "no harmonic", got: goHarmonicKind(0), want: "none"},
		{name: "natural harmonic", got: goHarmonicKind(HarmonicTypeNatural), want: "natural"},
		{name: "artificial harmonic", got: goHarmonicKind(HarmonicTypeArtificial), want: "artificial"},
		{name: "tapped harmonic", got: goHarmonicKind(HarmonicTypeTapped), want: "tap"},
		{name: "pinch harmonic", got: goHarmonicKind(HarmonicTypePinch), want: "pinch"},
		{name: "semi harmonic", got: goHarmonicKind(HarmonicTypeSemi), want: "semi"},
		{name: "unknown harmonic", got: goHarmonicKind(HarmonicType(99)), want: "unknown:99"},
		{name: "no grace transition", got: goGraceTransition(GraceEffectTransitionNone), want: "none"},
		{name: "grace slide", got: goGraceTransition(GraceEffectTransitionSlide), want: "slide"},
		{name: "grace bend", got: goGraceTransition(GraceEffectTransitionBend), want: "bend"},
		{name: "grace hammer", got: goGraceTransition(GraceEffectTransitionHammer), want: "hammer"},
		{name: "unknown grace transition", got: goGraceTransition(GraceEffectTransition(99)), want: "unknown:99"},
		{name: "slide in below", got: goSlide(SlideIntoFromBelow), want: "into-from-below"},
		{name: "slide in above", got: goSlide(SlideIntoFromAbove), want: "into-from-above"},
		{name: "no slide", got: goSlide(SlideNone), want: "none"},
		{name: "shift slide", got: goSlide(SlideShiftSlideTo), want: "shift"},
		{name: "legato slide", got: goSlide(SlideLegatoSlideTo), want: "legato"},
		{name: "slide out down", got: goSlide(SlideOutDownwards), want: "out-down"},
		{name: "slide out up", got: goSlide(SlideOutUpwards), want: "out-up"},
		{name: "unknown slide", got: goSlide(SlideType(99)), want: "unknown:99"},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if check.got != check.want {
				t.Fatalf("adapter value = %q, want %q", check.got, check.want)
			}
		})
	}

	articulations := normalizeGoPercussionArticulations([]PercussionArticulation{
		{InputMIDINumbers: nil},
		{InputMIDINumbers: []int{0}},
	})
	missing := articulations[0].(map[string]any)["inputMidiNumber"]
	zero := articulations[1].(map[string]any)["inputMidiNumber"]
	if missing != nil || zero != 0 {
		t.Fatalf("percussion input presence = missing %#v, explicit zero %#v", missing, zero)
	}

	if got := normalizeGoRepeatCount(-1); got != 0 {
		t.Fatalf("no repeat count = %d, want 0", got)
	}
	if got := normalizeGoRepeatCount(5); got != 6 {
		t.Fatalf("legacy repeat close 5 = %d, want source count 6", got)
	}

	for index, want := range []string{"ppp", "pp", "p", "mp", "mf", "f", "ff", "fff"} {
		velocity := MinVelocity + int16(index)*VelocityIncrement
		if got := goDynamic(velocity); got != want {
			t.Errorf("dynamic %d = %q, want %q", velocity, got, want)
		}
	}
	if got := goDynamic(1); got != "1" {
		t.Fatalf("unknown dynamic = %q, want visible source value", got)
	}
	track := Track{Offset: 2}
	staff := Staff{Strings: []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}}}
	if got := normalizeGoTuning(staff.Strings); !reflect.DeepEqual(got, []any{int8(64), int8(59)}) {
		t.Fatalf("Go tuning orientation = %#v", got)
	}
	fretted := normalizeGoNote(&track, &staff, &Note{String: 1, Value: 3}).(map[string]any)
	if fretted["string"] != int8(1) || fretted["fret"] != int16(3) || fretted["midi"] != 69 {
		t.Fatalf("fretted note normalization = %#v", fretted)
	}
	absolute := normalizeGoNote(&track, &staff, &Note{String: 0, Value: 60}).(map[string]any)
	if absolute["string"] != int8(0) || absolute["fret"] != nil || absolute["midi"] != 60 {
		t.Fatalf("absolute note normalization = %#v", absolute)
	}
	percussionTrack := Track{
		PercussionTrack: true,
		Staves: []Staff{{
			Strings: []GuitarString{{Number: 1}, {Number: 2}},
			Measures: []Measure{{Clef: MeasureClefTreble, Voices: []Voice{{Beats: []Beat{{
				Status: BeatStatusNormal, Notes: []Note{{String: 2, Value: 38}},
			}}}}}},
		}},
	}
	percussionSong := Song{Tracks: []Track{percussionTrack}}
	percussionStaff := normalizeGoStaves(&percussionSong, 0)[0].(map[string]any)
	percussionBar := percussionStaff["bars"].([]any)[0].(map[string]any)
	percussionNote := percussionBar["voices"].([]any)[0].(map[string]any)["beats"].([]any)[0].(map[string]any)["notes"].([]any)[0].(map[string]any)
	if percussionStaff["percussion"] != true || len(percussionStaff["tuning"].([]any)) != 0 || percussionBar["clef"] != "neutral" || percussionNote["string"] != int8(-1) || percussionNote["fret"] != nil {
		t.Fatalf("track-level percussion normalization = staff %#v, bar %#v, note %#v", percussionStaff, percussionBar, percussionNote)
	}

	quarter := DurationQuarterTime
	score := normalizeGoScore(&Song{
		TempoAutomations: []TempoAutomation{{Bar: 0, Position: 0.25, Tempo: 132.5}},
		MeasureHeaders:   []MeasureHeader{{Start: quarter, RepeatClose: -1}},
	}).(map[string]any)
	masterBar := score["masterBars"].([]any)[0].(map[string]any)
	if masterBar["index"] != 0 || masterBar["start"] != int64(0) {
		t.Fatalf("master-bar zero-based origin = %#v", masterBar)
	}
	automation := score["tempoAutomations"].([]any)[0].(map[string]any)
	if automation["position"] != 0.25 || automation["value"] != 132.5 {
		t.Fatalf("fractional tempo automation = %#v", automation)
	}
}

func m24OwnershipProbe(first, second int) any {
	note := func(midi int) any {
		return map[string]any{"midi": midi}
	}
	staff := func(midi int) any {
		return map[string]any{"bars": []any{map[string]any{
			"voices": []any{map[string]any{
				"beats": []any{map[string]any{"notes": []any{note(midi)}}},
			}},
		}}}
	}
	return map[string]any{"tracks": []any{map[string]any{
		"staves": []any{staff(first), staff(second)},
	}}}
}

func m24BeatProbe(octave string) any {
	return map[string]any{"tracks": []any{map[string]any{
		"staves": []any{map[string]any{
			"bars": []any{map[string]any{
				"voices": []any{map[string]any{
					"beats": []any{map[string]any{"octave": octave}},
				}},
			}},
		}},
	}}}
}
