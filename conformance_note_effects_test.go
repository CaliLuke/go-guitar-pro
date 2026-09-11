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

func TestConformanceTechniqueDispositions(t *testing.T) {
	runConformanceTechniqueDispositions(newConformanceRun(t))
}

func runConformanceTechniqueDispositions(run *conformanceRun) {
	t := run.t
	graces := []GraceEffect{{Duration: DurationThirtySecond, Fret: 2, Velocity: Forte}}
	run.Preserved("NoteEffect.Graces", graces, []GraceEffect{{Duration: DurationThirtySecond, Fret: 2, Velocity: Forte}})
	for _, source := range []struct {
		period byte
		want   uint16
	}{{1, uint16(DurationSixteenth)}, {2, uint16(DurationThirtySecond)}, {3, uint16(DurationSixtyFourth)}} {
		effect, err := (&Song{}).readTrill(newCursor([]byte{7, source.period}))
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("readTrill:period", effect.Duration.Value, source.want)
	}
	for _, unknown := range []struct {
		code  string
		data  []byte
		parse func(*cursor) error
	}{
		{"Binary.Note.TremoloPicking.Subdivision.Unsupported", []byte{99}, func(cursor *cursor) error { _, err := (&Song{}).readTremoloPicking(cursor); return err }},
		{"Binary.Note.Trill.Period.Unsupported", []byte{7, 99}, func(cursor *cursor) error { _, err := (&Song{}).readTrill(cursor); return err }},
	} {
		context := &parseContext{format: "GP5"}
		if err := unknown.parse(newCursorWithContext(unknown.data, context)); err != nil {
			t.Fatal(err)
		}
		if diagnostic := conformanceSourceAuditDiagnosticByCode(context.diagnostics, unknown.code); diagnostic == nil || diagnostic.Kind != ParseDiagnosticUnsupportedFeature {
			t.Fatalf("%s diagnostics = %#v", unknown.code, context.diagnostics)
		}
	}
	for index, accent := range []NoteAccent{NoteAccentNone, NoteAccentNormal, NoteAccentHeavy, NoteAccentTenuto} {
		run.Enum([]string{"NoteAccent.NoteAccentNone", "NoteAccent.NoteAccentNormal", "NoteAccent.NoteAccentHeavy", "NoteAccent.NoteAccentTenuto"}[index], accent, NoteAccent(index))
		run.ClaimPrimary(claimSite("tenuto", "import", "M11-TECHNIQUE-DISPOSITIONS", "tenuto accent"), claimSite("tenuto", "model", "M11-TECHNIQUE-DISPOSITIONS", "tenuto accent"), claimSite("tenuto", "export", "M11-TECHNIQUE-DISPOSITIONS", "tenuto accent")).Preserved("NoteEffect.Accent", NoteEffect{Accent: accent}.Accent, accent)
	}
	for index, vibrato := range []NoteVibrato{NoteVibratoNone, NoteVibratoSlight, NoteVibratoWide} {
		run.Enum([]string{"NoteVibrato.NoteVibratoNone", "NoteVibrato.NoteVibratoSlight", "NoteVibrato.NoteVibratoWide"}[index], vibrato, NoteVibrato(index))
		run.Preserved("NoteEffect.VibratoStrength", NoteEffect{VibratoStrength: vibrato}.VibratoStrength, vibrato)
	}
	run.Preserved("NoteEffect.Tapped", NoteEffect{Tapped: true}.Tapped, true)
	run.ClaimPrimary(claimSite("tapping", "model", "M11-TECHNIQUE-DISPOSITIONS", "hammer, tap, and left-hand-tap origins"), claimSite("tapping", "export", "M11-TECHNIQUE-DISPOSITIONS", "hammer, tap, and left-hand-tap origins")).Preserved("NoteEffect.LeftHandTapped", NoteEffect{LeftHandTapped: true}.LeftHandTapped, true)
	fingerings := []Fingering{FingeringOpen, FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle}
	for index, fingering := range fingerings {
		effect := defaultNoteEffect()
		effect.LeftHandFinger = fingering
		effect.RightHandFinger = fingering
		effect.HasLeftHandFinger = true
		effect.HasRightHandFinger = true
		run.ClaimPrimary(claimSite("fingering", "import", "M11-TECHNIQUE-DISPOSITIONS", "all named fingerings including thumb and open"), claimSite("fingering", "model", "M11-TECHNIQUE-DISPOSITIONS", "all named fingerings including thumb and open")).Preserved("NoteEffect.LeftHandFinger", effect.LeftHandFinger, fingering)
		run.Preserved("NoteEffect.RightHandFinger", effect.RightHandFinger, fingering)
		run.Preserved("NoteEffect.HasLeftHandFinger", effect.HasLeftHandFinger, true)
		run.Preserved("NoteEffect.HasRightHandFinger", effect.HasRightHandFinger, true)

		song := conformanceTechniqueSong(t)
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		note.Effect.LeftHandFinger = fingering
		note.Effect.RightHandFinger = fingering
		note.Effect.HasLeftHandFinger = true
		note.Effect.HasRightHandFinger = true
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		wantOmission := fingering == FingeringOpen
		run.Enum([]string{"Fingering.FingeringOpen", "Fingering.FingeringThumb", "Fingering.FingeringIndex", "Fingering.FingeringMiddle", "Fingering.FingeringAnnular", "Fingering.FingeringLittle"}[index], hasExportCode(report, "gp8.omit.left-hand-fingering") && hasExportCode(report, "gp8.omit.right-hand-fingering"), wantOmission)
		for _, code := range []string{"gp8.omit.left-hand-fingering", "gp8.omit.right-hand-fingering"} {
			if hasExportCode(report, code) != wantOmission {
				t.Errorf("fingering %d report = %#v, %s presence want %t", fingering, report.Entries, code, wantOmission)
			}
		}
	}

	song := conformanceTechniqueSong(t)
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
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimReport(claimSite("note-vibrato", "export", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants"), claimSite("accent", "export", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants"), claimSite("tenuto", "export", "M11-TECHNIQUE-DISPOSITIONS", "tenuto accent"), claimSite("staccato", "export", "M11-TECHNIQUE-DISPOSITIONS", "staccato"), claimSite("dead-ghost", "export", "M11-TECHNIQUE-DISPOSITIONS", "dead and ghost notes"), claimSite("palm-let-ring", "export", "M11-TECHNIQUE-DISPOSITIONS", "palm mute and let ring"), claimSite("slides", "export", "M11-TECHNIQUE-DISPOSITIONS", "all slide kinds including both pick-slide directions and combined flags"), claimSite("tapping", "export", "M11-TECHNIQUE-DISPOSITIONS", "hammer, tap, and left-hand-tap origins")).Report("M11-TECHNIQUE-DISPOSITIONS", reportCodes(report), []string{"gp8.normalize.track-view", "gp8.omit.hammer-origin-consumer"})
	values := extractGPIFLeafText(t, data)
	wire := extractTechniqueWireNote(t, data)
	run.Preserved("Note.Effect", note.Effect, wantEffect)
	run.ClaimPrimary(claimSite("accent", "import", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants"), claimSite("accent", "model", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants")).Preserved("NoteEffect.AccentuatedNote", note.Effect.AccentuatedNote, true)
	run.Preserved("NoteEffect.HeavyAccentuatedNote", note.Effect.HeavyAccentuatedNote, true)
	run.ClaimPrimary(claimSite("dead-ghost", "import", "M11-TECHNIQUE-DISPOSITIONS", "dead and ghost notes"), claimSite("dead-ghost", "model", "M11-TECHNIQUE-DISPOSITIONS", "dead and ghost notes")).Preserved("NoteEffect.GhostNote", note.Effect.GhostNote, true)
	run.ClaimPrimary(claimSite("staccato", "import", "M11-TECHNIQUE-DISPOSITIONS", "staccato"), claimSite("staccato", "model", "M11-TECHNIQUE-DISPOSITIONS", "staccato")).Preserved("NoteEffect.Staccato", note.Effect.Staccato, true)
	run.ClaimPrimary(claimSite("palm-let-ring", "import", "M11-TECHNIQUE-DISPOSITIONS", "palm mute and let ring"), claimSite("palm-let-ring", "model", "M11-TECHNIQUE-DISPOSITIONS", "palm mute and let ring")).Preserved("NoteEffect.PalmMute", note.Effect.PalmMute, true)
	run.Preserved("NoteEffect.DeadNote", note.Effect.DeadNote, true)
	run.Preserved("NoteEffect.LetRing", note.Effect.LetRing, true)
	run.ClaimPrimary(claimSite("note-vibrato", "model", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants")).Preserved("NoteEffect.Vibrato", note.Effect.Vibrato, true)
	run.Preserved("NoteEffect.Hammer", note.Effect.Hammer, true)
	run.ClaimPrimary(claimSite("slides", "import", "M11-TECHNIQUE-DISPOSITIONS", "all slide kinds including both pick-slide directions and combined flags"), claimSite("slides", "model", "M11-TECHNIQUE-DISPOSITIONS", "all slide kinds including both pick-slide directions and combined flags")).Preserved("NoteEffect.Slides", note.Effect.Slides, []SlideType{SlideShiftSlideTo, SlideLegatoSlideTo, SlideOutDownwards, SlideOutUpwards, SlideIntoFromBelow, SlideIntoFromAbove})
	run.ClaimPrimary(claimSite("trill", "import", "M11-TECHNIQUE-DISPOSITIONS", "trill fret and duration"), claimSite("trill", "model", "M11-TECHNIQUE-DISPOSITIONS", "trill fret and duration")).Preserved("NoteEffect.Trill", note.Effect.Trill.Fret, int8(7))
	run.Preserved("TrillEffect.Fret", note.Effect.Trill.Fret, int8(7))
	run.Normalized("TrillEffect.Duration", note.Effect.Trill.Duration.Value, uint16(DurationSixteenth))
	run.Preserved("NoteEffect.Harmonic", note.Effect.Harmonic.Kind, HarmonicTypeNatural)
	run.ClaimSerialization(claimSite("accent", "export", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants"), claimSite("tenuto", "export", "M11-TECHNIQUE-DISPOSITIONS", "tenuto accent")).Wire("gpifNote.Accent", values["GPIF/Notes/Note/Accent"], "13")
	run.ClaimSerialization(claimSite("staccato", "export", "M11-TECHNIQUE-DISPOSITIONS", "staccato")).Wire("gpifNote.AntiAccent", values["GPIF/Notes/Note/AntiAccent"], "Normal")
	run.ClaimSerialization(claimSite("palm-let-ring", "export", "M11-TECHNIQUE-DISPOSITIONS", "palm mute and let ring")).Wire("gpifNote.LetRing", wire.LetRing != nil, true)
	run.ClaimSerialization(claimSite("note-vibrato", "export", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants")).Wire("gpifNote.Vibrato", values["GPIF/Notes/Note/Vibrato"], "Slight")
	run.Wire("gpifNote.Trill", values["GPIF/Notes/Note/Trill"], "7")
	run.Wire("gpifTrill.Fret", values["GPIF/Notes/Note/Trill"], "7")
	run.ClaimSerialization(claimSite("dead-ghost", "export", "M11-TECHNIQUE-DISPOSITIONS", "dead and ghost notes")).Wire("gpifProperty.Name", wire.propertyNames(), []string{"Fret", "Midi", "String", "HarmonicType", "Muted", "PalmMuted", "HopoOrigin", "Slide"})
	run.ClaimSerialization(claimSite("tapping", "export", "M11-TECHNIQUE-DISPOSITIONS", "hammer, tap, and left-hand-tap origins")).Wire("gpifProperty.Enable", wire.allEnabled("Muted", "PalmMuted", "HopoOrigin"), true)
	run.ClaimSerialization(claimSite("slides", "export", "M11-TECHNIQUE-DISPOSITIONS", "all slide kinds including both pick-slide directions and combined flags")).Wire("gpifProperty.Flags", wire.propertyText("Slide", "flags"), "63")
	run.Wire("gpifProperty.HType", wire.propertyText("HarmonicType", "type"), "Natural")

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect
	run.ClaimPrimary(claimSite("accent", "export", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants")).Field("NoteEffect.AccentuatedNote", got.AccentuatedNote, true)
	run.Field("NoteEffect.HeavyAccentuatedNote", got.HeavyAccentuatedNote, true)
	run.ClaimPrimary(claimSite("dead-ghost", "export", "M11-TECHNIQUE-DISPOSITIONS", "dead and ghost notes")).Field("NoteEffect.GhostNote", got.GhostNote, true)
	run.ClaimPrimary(claimSite("staccato", "export", "M11-TECHNIQUE-DISPOSITIONS", "staccato")).Field("NoteEffect.Staccato", got.Staccato, true)
	run.ClaimPrimary(claimSite("palm-let-ring", "export", "M11-TECHNIQUE-DISPOSITIONS", "palm mute and let ring")).Field("NoteEffect.PalmMute", got.PalmMute, true)
	run.Field("NoteEffect.DeadNote", got.DeadNote, true)
	run.Field("NoteEffect.LetRing", got.LetRing, true)
	run.ClaimPrimary(claimSite("note-vibrato", "export", "M11-TECHNIQUE-DISPOSITIONS", "all note-accent and vibrato-strength variants")).Field("NoteEffect.Vibrato", got.Vibrato, true)
	run.Field("NoteEffect.Hammer", got.Hammer, true)
	run.ClaimPrimary(claimSite("slides", "export", "M11-TECHNIQUE-DISPOSITIONS", "all slide kinds including both pick-slide directions and combined flags")).Field("NoteEffect.Slides", got.Slides, note.Effect.Slides)

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
			caseSong := conformanceTechniqueSong(t)
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

func TestConformanceSourceDistinctions(t *testing.T) {
	runConformanceSourceDistinctions(newConformanceRun(t))
}

func runConformanceSourceDistinctions(run *conformanceRun) {
	t := run.t
	empty := ""
	for _, property := range []gpifProperty{{Name: "Muted"}, {Name: "PalmMuted"}, {Name: "HopoOrigin"}} {
		context := &parseContext{format: "GPIF"}
		gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", property)
		if !slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Kind == ParseDiagnosticInvalidData && diagnostic.SourcePath != ""
		}) {
			t.Errorf("%s diagnostics = %#v, want invalid missing payload", property.Name, context.diagnostics)
		}
	}

	for _, property := range []gpifProperty{{Name: "Muted", Enable: &empty}, {Name: "PalmMuted", Enable: &empty}, {Name: "HopoOrigin", Enable: &empty}} {
		context := &parseContext{format: "GPIF"}
		gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", property)
		if slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Kind == ParseDiagnosticInvalidData }) {
			t.Errorf("%s enabled diagnostics = %#v", property.Name, context.diagnostics)
		}
		note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{property}}}, 6, false)
		if err != nil {
			t.Fatal(err)
		}
		got := note.Effect.DeadNote || note.Effect.PalmMute || note.Effect.Hammer
		run.ClaimPrimary(claimSite("tapping", "import", "M11-SOURCE-DISTINCTIONS", "hammer, tap, and left-hand-tap origins")).Dispatch("gpifNoteToNote:p.Name", got, true)
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
		wire string
		want NoteVibrato
	}{{wire: "Slight", want: NoteVibratoSlight}, {wire: "Wide", want: NoteVibratoWide}} {
		vibratoNote, vibratoErr := gpifNoteToNote(&gpifNote{Vibrato: test.wire}, 6, false)
		if vibratoErr != nil {
			t.Fatal(vibratoErr)
		}
		run.ClaimPrimary(claimSite("note-vibrato", "import", "M11-SOURCE-DISTINCTIONS", "all note-accent and vibrato-strength variants")).Dispatch("gpifNoteToNote:n.Vibrato", vibratoNote.Effect.VibratoStrength, test.want)
	}

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
		{flags: "64", want: []SlideType{SlidePickSlideDown}},
		{flags: "128", want: []SlideType{SlidePickSlideUp}},
		{flags: "255", want: []SlideType{SlideShiftSlideTo, SlideLegatoSlideTo, SlideOutDownwards, SlideOutUpwards, SlideIntoFromBelow, SlideIntoFromAbove, SlidePickSlideDown, SlidePickSlideUp}},
	} {
		slideNote, slideErr := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Slide", Flags: &test.flags}}}}, 6, false)
		if slideErr != nil {
			t.Fatal(slideErr)
		}
		run.Dispatch("gpifNoteToNote:p.Name", slideNote.Effect.Slides, test.want)
		for _, slide := range test.want {
			run.Enum(semanticSlideMember(slide), slices.Contains(slideNote.Effect.Slides, slide), true)
		}
	}
	run.Enum("SlideType.SlideNone", len([]SlideType(nil)), 0)

	flags := "256"
	context := &parseContext{format: "GPIF"}
	gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", gpifProperty{Name: "Slide", Flags: &flags})
	if !slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Kind == ParseDiagnosticUnsupportedFeature && diagnostic.Code == "GPIF.Note.Property.Slide.UnknownFlags"
	}) {
		t.Errorf("slide diagnostics = %#v, want unknown flag", context.diagnostics)
	}

	for _, test := range []struct {
		property gpifProperty
		wantLoss bool
	}{
		{property: gpifProperty{Name: "Tapped", Enable: &empty}},
		{property: gpifProperty{Name: "HopoOrigin", Enable: &empty}},
		{property: gpifProperty{Name: "HopoDestination", Enable: &empty}},
		{property: gpifProperty{Name: "LeftHandTapped", Enable: &empty}},
	} {
		context := &parseContext{format: "GPIF"}
		gpifAuditNoteProperty(context, "n1", "/GPIF/Notes/Note[@id=\"n1\"]", test.property)
		run.Dispatch("gpifAuditNoteProperty:property.Name", slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Kind == ParseDiagnosticLossyProjection
		}), test.wantLoss)
	}

	vibratoContext := &parseContext{format: "GPIF"}
	gpifAuditDiagnostics(gpifDocument{Notes: gpifNotes{Notes: []gpifNote{{ID: "n1", Vibrato: "Wide", Accent: 0x10}}}}, vibratoContext)
	if len(vibratoContext.diagnostics) != 0 {
		t.Errorf("supported wide vibrato and tenuto diagnostics = %#v", vibratoContext.diagnostics)
	}
	invalidVibratoContext := &parseContext{format: "GPIF"}
	gpifAuditDiagnostics(gpifDocument{Notes: gpifNotes{Notes: []gpifNote{{ID: "n1", Vibrato: "Extreme"}}}}, invalidVibratoContext)
	if !slices.ContainsFunc(invalidVibratoContext.diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.Note.Vibrato.InvalidValue"
	}) {
		t.Errorf("invalid vibrato diagnostics = %#v", invalidVibratoContext.diagnostics)
	}

	song := conformanceTechniqueSong(t)
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Effect.LeftHandFinger = FingeringOpen
	note.Effect.HasLeftHandFinger = true
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(data) != 0 || !errors.As(err, &lossErr) {
		t.Fatalf("strict thumb export = %d bytes, %v", len(data), err)
	}
}

func semanticSlideMember(value SlideType) string {
	switch value {
	case SlideIntoFromAbove:
		return "SlideType.SlideIntoFromAbove"
	case SlideIntoFromBelow:
		return "SlideType.SlideIntoFromBelow"
	case SlideShiftSlideTo:
		return "SlideType.SlideShiftSlideTo"
	case SlideLegatoSlideTo:
		return "SlideType.SlideLegatoSlideTo"
	case SlideOutDownwards:
		return "SlideType.SlideOutDownwards"
	case SlideOutUpwards:
		return "SlideType.SlideOutUpwards"
	case SlidePickSlideDown:
		return "SlideType.SlidePickSlideDown"
	case SlidePickSlideUp:
		return "SlideType.SlidePickSlideUp"
	default:
		return "SlideType.SlideNone"
	}
}

func TestConformanceValidation(t *testing.T) {
	runConformanceValidation(newConformanceRun(t))
}

func runConformanceValidation(run *conformanceRun) {
	t := run.t
	tests := []struct {
		name string
		code string
		set  func(*Note)
	}{
		{name: "left fingering", code: "score.note.left-hand-fingering", set: func(note *Note) { note.Effect.LeftHandFinger = Fingering(-2) }},
		{name: "right fingering", code: "score.note.right-hand-fingering", set: func(note *Note) { note.Effect.RightHandFinger = Fingering(5) }},
		{name: "slide", code: "score.note.slide", set: func(note *Note) { note.Effect.Slides = []SlideType{SlideType(7)} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := conformanceTechniqueSong(t)
			test.set(&song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0])
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Errorf("diagnostics = %#v, want %s", diagnostics, test.code)
			}
		})
	}

	for _, fingering := range []Fingering{FingeringOpen, FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle} {
		song := conformanceTechniqueSong(t)
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

func conformanceTechniqueSong(t *testing.T) *Song {
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

type conformanceTechniqueWireProperty struct {
	Name   string  `xml:"name,attr"`
	Enable *string `xml:"Enable"`
	Flags  *string `xml:"Flags"`
	HType  *string `xml:"HType"`
}

type conformanceTechniqueWireNote struct {
	LetRing    *struct{} `xml:"LetRing"`
	Properties struct {
		Items []conformanceTechniqueWireProperty `xml:"Property"`
	} `xml:"Properties"`
}

func (note conformanceTechniqueWireNote) propertyNames() []string {
	result := make([]string, len(note.Properties.Items))
	for index, property := range note.Properties.Items {
		result[index] = property.Name
	}
	return result
}

func (note conformanceTechniqueWireNote) allEnabled(names ...string) bool {
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

func (note conformanceTechniqueWireNote) propertyText(name, payload string) string {
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

func extractTechniqueWireNote(t *testing.T, data []byte) conformanceTechniqueWireNote {
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
				Items []conformanceTechniqueWireNote `xml:"Note"`
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
	return conformanceTechniqueWireNote{}
}
