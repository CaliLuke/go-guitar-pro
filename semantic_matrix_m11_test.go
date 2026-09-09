// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"slices"
	"testing"
)

func TestSemanticMatrixM11TechniqueDispositions(t *testing.T) {
	runSemanticMatrixM11TechniqueDispositions(newSemanticMatrixRun(t))
}

func runSemanticMatrixM11TechniqueDispositions(run *semanticMatrixRun) {
	t := run.t
	fingerings := []Fingering{FingeringOpen, FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle}
	for _, fingering := range fingerings {
		effect := defaultNoteEffect()
		effect.LeftHandFinger = fingering
		effect.RightHandFinger = fingering
		effect.HasLeftHandFinger = true
		effect.HasRightHandFinger = true
		run.Field("NoteEffect.LeftHandFinger", effect.LeftHandFinger, fingering)
		run.Field("NoteEffect.RightHandFinger", effect.RightHandFinger, fingering)
		run.Field("NoteEffect.HasLeftHandFinger", effect.HasLeftHandFinger, true)
		run.Field("NoteEffect.HasRightHandFinger", effect.HasRightHandFinger, true)

		song := m11Song(t)
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		note.Effect.LeftHandFinger = fingering
		note.Effect.RightHandFinger = fingering
		note.Effect.HasLeftHandFinger = true
		note.Effect.HasRightHandFinger = true
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		for _, code := range []string{"gp8.omit.left-hand-fingering", "gp8.omit.right-hand-fingering"} {
			if !hasExportCode(report, code) {
				t.Errorf("fingering %d report = %#v, want %s", fingering, report.Entries, code)
			}
		}
	}

	for _, value := range []uint16{uint16(DurationEighth), uint16(DurationSixteenth), uint16(DurationThirtySecond)} {
		song := m11Song(t)
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		duration := defaultDuration()
		duration.Value = value
		note.Effect.TremoloPicking = &TremoloPickingEffect{Duration: duration}
		if value == uint16(DurationThirtySecond) {
			note.Effect.Graces = []GraceEffect{{Duration: DurationThirtySecond, Fret: 2, Velocity: Forte}}
			run.Field("NoteEffect.Graces", note.Effect.Graces, []GraceEffect{{Duration: DurationThirtySecond, Fret: 2, Velocity: Forte}})
		}
		run.Field("NoteEffect.TremoloPicking", note.Effect.TremoloPicking.Duration.Value, value)
		run.Field("TremoloPickingEffect.Duration", note.Effect.TremoloPicking.Duration, duration)
		if report := PreflightExport(song, ExportFormatGP8, ExportOptions{}); !hasExportCode(report, "gp8.omit.tremolo-picking") {
			t.Errorf("tremolo duration %d report = %#v", value, report.Entries)
		}
	}

	song := m11Song(t)
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	wantEffect := NoteEffect{
		AccentuatedNote: true, HeavyAccentuatedNote: true, GhostNote: true, Staccato: true,
		PalmMute: true, DeadNote: true, LetRing: true, Vibrato: true, Hammer: true,
		Slides:         []SlideType{SlideShiftSlideTo, SlideLegatoSlideTo, SlideOutDownwards, SlideOutUpwards, SlideIntoFromBelow, SlideIntoFromAbove},
		Trill:          &TrillEffect{Fret: 7, Duration: func() Duration { d := defaultDuration(); d.Value = uint16(DurationSixteenth); return d }()},
		Harmonic:       &HarmonicEffect{Kind: HarmonicTypeNatural},
		LeftHandFinger: FingeringOpen, RightHandFinger: FingeringOpen,
	}
	note.Effect = wantEffect
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	values := extractGPIFLeafText(t, data)
	wire := extractM11WireNote(t, data)
	run.Field("Note.Effect", note.Effect, wantEffect)
	run.Field("NoteEffect.AccentuatedNote", note.Effect.AccentuatedNote, true)
	run.Field("NoteEffect.HeavyAccentuatedNote", note.Effect.HeavyAccentuatedNote, true)
	run.Field("NoteEffect.GhostNote", note.Effect.GhostNote, true)
	run.Field("NoteEffect.Staccato", note.Effect.Staccato, true)
	run.Field("NoteEffect.PalmMute", note.Effect.PalmMute, true)
	run.Field("NoteEffect.DeadNote", note.Effect.DeadNote, true)
	run.Field("NoteEffect.LetRing", note.Effect.LetRing, true)
	run.Field("NoteEffect.Vibrato", note.Effect.Vibrato, true)
	run.Field("NoteEffect.Hammer", note.Effect.Hammer, true)
	run.Field("NoteEffect.Slides", note.Effect.Slides, []SlideType{SlideShiftSlideTo, SlideLegatoSlideTo, SlideOutDownwards, SlideOutUpwards, SlideIntoFromBelow, SlideIntoFromAbove})
	run.Field("NoteEffect.Trill", note.Effect.Trill.Fret, int8(7))
	run.Field("TrillEffect.Fret", note.Effect.Trill.Fret, int8(7))
	run.Field("TrillEffect.Duration", note.Effect.Trill.Duration.Value, uint16(DurationSixteenth))
	run.Field("NoteEffect.Harmonic", note.Effect.Harmonic.Kind, HarmonicTypeNatural)
	run.Wire("gpifNote.Accent", values["GPIF/Notes/Note/Accent"], "13")
	run.Wire("gpifNote.AntiAccent", values["GPIF/Notes/Note/AntiAccent"], "Normal")
	run.Wire("gpifNote.LetRing", wire.LetRing != nil, true)
	run.Wire("gpifNote.Vibrato", values["GPIF/Notes/Note/Vibrato"], "Slight")
	run.Wire("gpifNote.Trill", values["GPIF/Notes/Note/Trill"], "7")
	run.Wire("gpifTrill.Fret", values["GPIF/Notes/Note/Trill"], "7")
	run.Wire("gpifProperty.Name", wire.propertyNames(), []string{"ConcertPitch", "TransposedPitch", "Fret", "Midi", "String", "HarmonicType", "Muted", "PalmMuted", "HopoOrigin", "Slide"})
	run.Wire("gpifProperty.Enable", wire.allEnabled("Muted", "PalmMuted", "HopoOrigin"), true)
	run.Wire("gpifProperty.Flags", wire.propertyText("Slide", "flags"), "63")
	run.Wire("gpifProperty.HType", wire.propertyText("HarmonicType", "type"), "Natural")

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect
	run.Field("NoteEffect.AccentuatedNote", got.AccentuatedNote, true)
	run.Field("NoteEffect.HeavyAccentuatedNote", got.HeavyAccentuatedNote, true)
	run.Field("NoteEffect.GhostNote", got.GhostNote, true)
	run.Field("NoteEffect.Staccato", got.Staccato, true)
	run.Field("NoteEffect.PalmMute", got.PalmMute, true)
	run.Field("NoteEffect.DeadNote", got.DeadNote, true)
	run.Field("NoteEffect.LetRing", got.LetRing, true)
	run.Field("NoteEffect.Vibrato", got.Vibrato, true)
	run.Field("NoteEffect.Hammer", got.Hammer, true)
	run.Field("NoteEffect.Slides", got.Slides, note.Effect.Slides)

	for _, test := range []struct {
		name  string
		set   func(*NoteEffect)
		check func(NoteEffect) bool
	}{
		{name: "accent", set: func(effect *NoteEffect) { effect.AccentuatedNote = true }, check: func(effect NoteEffect) bool { return effect.AccentuatedNote && !effect.HeavyAccentuatedNote }},
		{name: "heavy accent", set: func(effect *NoteEffect) { effect.HeavyAccentuatedNote = true }, check: func(effect NoteEffect) bool { return effect.HeavyAccentuatedNote && !effect.AccentuatedNote }},
		{name: "ghost", set: func(effect *NoteEffect) { effect.GhostNote = true }, check: func(effect NoteEffect) bool { return effect.GhostNote }},
		{name: "staccato", set: func(effect *NoteEffect) { effect.Staccato = true }, check: func(effect NoteEffect) bool { return effect.Staccato }},
		{name: "palm mute", set: func(effect *NoteEffect) { effect.PalmMute = true }, check: func(effect NoteEffect) bool { return effect.PalmMute }},
		{name: "dead", set: func(effect *NoteEffect) { effect.DeadNote = true }, check: func(effect NoteEffect) bool { return effect.DeadNote }},
		{name: "let ring", set: func(effect *NoteEffect) { effect.LetRing = true }, check: func(effect NoteEffect) bool { return effect.LetRing }},
		{name: "vibrato", set: func(effect *NoteEffect) { effect.Vibrato = true }, check: func(effect NoteEffect) bool { return effect.Vibrato }},
		{name: "hammer", set: func(effect *NoteEffect) { effect.Hammer = true }, check: func(effect NoteEffect) bool { return effect.Hammer }},
	} {
		t.Run(test.name, func(t *testing.T) {
			caseSong := m11Song(t)
			caseNote := &caseSong.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
			test.set(&caseNote.Effect)
			caseData, exportErr := Export(caseSong, ExportFormatGP8)
			if exportErr != nil {
				t.Fatal(exportErr)
			}
			parsed, parseErr := Parse(caseData)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			parsedEffect := parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect
			if !test.check(parsedEffect) {
				t.Errorf("round-trip effect = %#v", parsedEffect)
			}
		})
	}
}

func TestSemanticMatrixM11SourceDistinctions(t *testing.T) {
	runSemanticMatrixM11SourceDistinctions(newSemanticMatrixRun(t))
}

func runSemanticMatrixM11SourceDistinctions(run *semanticMatrixRun) {
	t := run.t
	empty := ""
	for _, property := range []gpifProperty{{Name: "Muted"}, {Name: "PalmMuted"}, {Name: "HopoOrigin"}} {
		context := &parseContext{format: "GPIF"}
		gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", property, nil)
		if !slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Kind == ParseDiagnosticInvalidData && diagnostic.SourcePath != ""
		}) {
			t.Errorf("%s diagnostics = %#v, want invalid missing payload", property.Name, context.diagnostics)
		}
	}

	for _, property := range []gpifProperty{{Name: "Muted", Enable: &empty}, {Name: "PalmMuted", Enable: &empty}, {Name: "HopoOrigin", Enable: &empty}} {
		context := &parseContext{format: "GPIF"}
		gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", property, nil)
		if slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Kind == ParseDiagnosticInvalidData }) {
			t.Errorf("%s enabled diagnostics = %#v", property.Name, context.diagnostics)
		}
		note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{property}}}, 6, false)
		if err != nil {
			t.Fatal(err)
		}
		got := note.Effect.DeadNote || note.Effect.PalmMute || note.Effect.Hammer
		run.Dispatch("gpifNoteToNote:p.Name", got, true)
	}
	enabled := "Enabled"
	for _, property := range []gpifProperty{{Name: "Muted", Enable: &enabled}, {Name: "PalmMuted", Enable: &enabled}, {Name: "HopoOrigin", Enable: &enabled}} {
		note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{property}}}, 6, false)
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("gpifNoteToNote:p.Name", note.Effect.DeadNote || note.Effect.PalmMute || note.Effect.Hammer, true)
	}
	emptyNote, err := gpifNoteToNote(&gpifNote{}, 6, false)
	if err != nil {
		t.Fatal(err)
	}
	run.Dispatch("gpifNoteToNote:p.Name", emptyNote.Effect.DeadNote || emptyNote.Effect.PalmMute || emptyNote.Effect.Hammer, false)

	for _, test := range []struct {
		flags string
		want  []SlideType
	}{
		{flags: "0"},
		{flags: "1", want: []SlideType{SlideShiftSlideTo}},
		{flags: "2", want: []SlideType{SlideLegatoSlideTo}},
		{flags: "4", want: []SlideType{SlideOutDownwards}},
		{flags: "8", want: []SlideType{SlideOutUpwards}},
		{flags: "16", want: []SlideType{SlideIntoFromBelow}},
		{flags: "32", want: []SlideType{SlideIntoFromAbove}},
		{flags: "63", want: []SlideType{SlideShiftSlideTo, SlideLegatoSlideTo, SlideOutDownwards, SlideOutUpwards, SlideIntoFromBelow, SlideIntoFromAbove}},
	} {
		slideNote, slideErr := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Slide", Flags: &test.flags}}}}, 6, false)
		if slideErr != nil {
			t.Fatal(slideErr)
		}
		run.Dispatch("gpifNoteToNote:p.Name", slideNote.Effect.Slides, test.want)
	}

	flags := "64"
	context := &parseContext{format: "GPIF"}
	gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", gpifProperty{Name: "Slide", Flags: &flags}, nil)
	if !slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Kind == ParseDiagnosticUnsupportedFeature && diagnostic.Code == "GPIF.Note.Property.Slide.UnknownFlags"
	}) {
		t.Errorf("slide diagnostics = %#v, want unknown flag", context.diagnostics)
	}

	for _, property := range []gpifProperty{{Name: "Tapped", Enable: &empty}, {Name: "HopoOrigin", Enable: &empty}, {Name: "HopoDestination", Enable: &empty}, {Name: "LeftHandTapped", Enable: &empty}} {
		context := &parseContext{format: "GPIF"}
		gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", property, nil)
		run.Dispatch("gpifAuditNoteProperty:property.Name", slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Kind == ParseDiagnosticLossyProjection
		}), true)
	}

	vibratoContext := &parseContext{format: "GPIF"}
	gpifAuditDiagnostics(gpifDocument{Notes: gpifNotes{Notes: []gpifNote{{ID: "n1", Vibrato: "Wide", Accent: 0x10}}}}, vibratoContext)
	for _, code := range []string{"GPIF.Note.Vibrato", "GPIF.Note.Accent.Tenuto"} {
		if !slices.ContainsFunc(vibratoContext.diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Code == code
		}) {
			t.Errorf("diagnostics = %#v, want %s", vibratoContext.diagnostics, code)
		}
	}

	song := m11Song(t)
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Effect.LeftHandFinger = FingeringThumb
	note.Effect.HasLeftHandFinger = true
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(data) != 0 || !errors.As(err, &lossErr) {
		t.Fatalf("strict thumb export = %d bytes, %v", len(data), err)
	}
}

func TestSemanticMatrixM11Validation(t *testing.T) {
	runSemanticMatrixM11Validation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM11Validation(run *semanticMatrixRun) {
	t := run.t
	tests := []struct {
		name string
		code string
		set  func(*Note)
	}{
		{name: "left fingering", code: "score.note.left-hand-fingering", set: func(note *Note) { note.Effect.LeftHandFinger = Fingering(-2) }},
		{name: "right fingering", code: "score.note.right-hand-fingering", set: func(note *Note) { note.Effect.RightHandFinger = Fingering(5) }},
		{name: "slide", code: "score.note.slide", set: func(note *Note) { note.Effect.Slides = []SlideType{SlideType(5)} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := m11Song(t)
			test.set(&song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0])
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Errorf("diagnostics = %#v, want %s", diagnostics, test.code)
			}
		})
	}

	for _, fingering := range []Fingering{FingeringOpen, FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle} {
		song := m11Song(t)
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		note.Effect.LeftHandFinger = fingering
		note.Effect.RightHandFinger = fingering
		note.Effect.HasLeftHandFinger = true
		note.Effect.HasRightHandFinger = true
		run.Field("NoteEffect.LeftHandFinger", note.Effect.LeftHandFinger, fingering)
		run.Field("NoteEffect.RightHandFinger", note.Effect.RightHandFinger, fingering)
		run.Field("NoteEffect.HasLeftHandFinger", note.Effect.HasLeftHandFinger, true)
		run.Field("NoteEffect.HasRightHandFinger", note.Effect.HasRightHandFinger, true)
		if diagnostics := ValidateSong(song); slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool {
			return diagnostic.Code == "score.note.left-hand-fingering" || diagnostic.Code == "score.note.right-hand-fingering"
		}) {
			t.Errorf("fingering %d diagnostics = %#v", fingering, diagnostics)
		}
	}
}

func m11Song(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	song.Tracks[0].Measures = slices.Clone(song.Tracks[0].Measures[:1])
	quarter := defaultDuration()
	song.Tracks[0].Measures[0].Voices = []Voice{{Beats: []Beat{{Duration: quarter, Status: BeatStatusNormal, Notes: []Note{{Value: 1, String: 1, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}}}}}}
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

type m11WireProperty struct {
	Name   string  `xml:"name,attr"`
	Enable *string `xml:"Enable"`
	Flags  *string `xml:"Flags"`
	HType  *string `xml:"HType"`
}

type m11WireNote struct {
	LetRing    *struct{} `xml:"LetRing"`
	Properties struct {
		Items []m11WireProperty `xml:"Property"`
	} `xml:"Properties"`
}

func (note m11WireNote) propertyNames() []string {
	result := make([]string, len(note.Properties.Items))
	for index, property := range note.Properties.Items {
		result[index] = property.Name
	}
	return result
}

func (note m11WireNote) allEnabled(names ...string) bool {
	for _, name := range names {
		found := false
		for _, property := range note.Properties.Items {
			if property.Name == name && property.Enable != nil {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (note m11WireNote) propertyText(name, payload string) string {
	for _, property := range note.Properties.Items {
		if property.Name != name {
			continue
		}
		switch payload {
		case "flags":
			if property.Flags != nil {
				return *property.Flags
			}
		case "type":
			if property.HType != nil {
				return *property.HType
			}
		}
	}
	return ""
}

func extractM11WireNote(t *testing.T, data []byte) m11WireNote {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name != "Content/score.gpif" {
			continue
		}
		reader, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		var document struct {
			Notes struct {
				Items []m11WireNote `xml:"Note"`
			} `xml:"Notes"`
		}
		if err := xml.Unmarshal(contents, &document); err != nil {
			t.Fatal(err)
		}
		if len(document.Notes.Items) == 0 {
			t.Fatal("GPIF has no notes")
		}
		return document.Notes.Items[0]
	}
	t.Fatal("GP8 archive has no score.gpif")
	return m11WireNote{}
}
