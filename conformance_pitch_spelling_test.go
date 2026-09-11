// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"os"
	"slices"
	"strings"
	"testing"
)

type accidentalCase struct {
	mode              NoteAccidentalMode
	enum, step, token string
	midi              int16
	octave, consumer  int
}

func accidentalCases() []accidentalCase {
	return []accidentalCase{
		{NoteAccidentalDefault, "NoteAccidentalDefault", "", "", 61, 0, 0},
		{NoteAccidentalNatural, "NoteAccidentalNatural", "C", "", 60, 5, 2},
		{NoteAccidentalSharp, "NoteAccidentalSharp", "C", "#", 61, 5, 3},
		{NoteAccidentalDoubleSharp, "NoteAccidentalDoubleSharp", "C", "x", 62, 5, 4},
		{NoteAccidentalFlat, "NoteAccidentalFlat", "D", "b", 61, 5, 5},
		{NoteAccidentalDoubleFlat, "NoteAccidentalDoubleFlat", "D", "bb", 60, 5, 6},
		{NoteAccidentalSharp, "NoteAccidentalSharp", "B", "#", 60, 4, 3},
		{NoteAccidentalFlat, "NoteAccidentalFlat", "C", "b", 71, 6, 5},
	}
}
func accidentalSong(t *testing.T, mode NoteAccidentalMode, midi int16) *Song {
	t.Helper()
	song := conformanceCurveSong(t)
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	song.Tracks[0].Staves[0].Strings[0].Value = 60
	note.String = 1
	note.Value = midi - 60
	note.AccidentalMode = mode
	return song
}
func TestConformancePitchSpelling(t *testing.T) { runConformancePitchSpelling(newConformanceRun(t)) }
func runConformancePitchSpelling(run *conformanceRun) {
	t := run.t
	for _, test := range accidentalCases() {
		song := accidentalSong(t, test.mode, test.midi)
		note := song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		run.Preserved("Note.AccidentalMode", note.AccidentalMode, test.mode)
		data, report := assertConsumerLossPolicy(t, song, []string{})
		run.Report("M09-PITCH-SPELLING", reportCodes(report), []string{})
		var doc struct {
			Notes struct {
				Notes []struct {
					Properties struct {
						Properties []struct {
							Name  string `xml:"name,attr"`
							Pitch *struct {
								Step       string
								Accidental *string
								Octave     int
							}
						} `xml:"Property"`
					}
				} `xml:"Note"`
			}
		}
		if err := xml.Unmarshal(readCurveGPIF(t, data), &doc); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range doc.Notes.Notes[0].Properties.Properties {
			if p.Name == "ConcertPitch" {
				t.Fatal("writer invented a second spelling authority")
			}
			if p.Name != "TransposedPitch" {
				continue
			}
			found = true
			if p.Pitch == nil || p.Pitch.Accidental == nil {
				t.Fatal("missing explicit accidental")
			}
			run.Wire("gpifProperty.Pitch", p.Pitch.Step, test.step)
			run.Wire("gpifPitch.Step", p.Pitch.Step, test.step)
			run.Wire("gpifPitch.Accidental", *p.Pitch.Accidental, test.token)
			run.Wire("gpifPitch.Octave", p.Pitch.Octave, test.octave)
		}
		if found != (test.mode != NoteAccidentalDefault) {
			t.Fatalf("mode %v wire presence %v", test.mode, found)
		}
		parsed, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(parsed.Diagnostics) != 0 {
			t.Fatalf("authored output parse diagnostics %#v", parsed.Diagnostics)
		}
		got := parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		run.Enum("NoteAccidentalMode."+test.enum, got.AccidentalMode, test.mode)
		run.Field("Note.Value", got.Value, test.midi-60)
	}
	for _, reverse := range []bool{false, true} {
		result, err := ParseWithOptions(pitchSpellingSource(t, reverse), ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("gpifAuthoredAccidental:p.Name", result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].AccidentalMode, NoteAccidentalFlat)
		run.Dispatch("gpifAuditNoteSpelling:property.Name", result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].AccidentalMode, NoteAccidentalFlat)
		codes := make([]string, len(result.Diagnostics))
		for i, d := range result.Diagnostics {
			codes[i] = d.Code
		}
		run.Dispatch("gpifAuditNoteProperty:property.Name", codes, []string{"GPIF.Note.Pitch.Authority"})
	}
	song := accidentalSong(t, NoteAccidentalFlat, 61)
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].SwapAccidentals = true
	_, report := assertConsumerLossPolicy(t, song, []string{"gp8.omit.swap-accidentals"})
	run.Omitted("Note.SwapAccidentals", song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].SwapAccidentals, true)
	run.Report("M09-PITCH-SWAP-COMPATIBILITY", reportCodes(report), []string{"gp8.omit.swap-accidentals"})
}

func TestGPIFPitchSpellingAuthority(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		source := pitchSpellingSource(t, reverse)
		result, err := ParseWithOptions(source, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		note := result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		if note.AccidentalMode != NoteAccidentalFlat || note.Value != 1 {
			t.Fatalf("property authority %#v", note)
		}
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "GPIF.Note.Pitch.Authority" {
			t.Fatalf("authority diagnostics %#v", result.Diagnostics)
		}
		// Editing the imported typed value and clearing it each control export.
		for _, mode := range []NoteAccidentalMode{NoteAccidentalSharp, NoteAccidentalDefault} {
			song := accidentalSong(t, mode, 61)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0] = note
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].AccidentalMode = mode
			data, _ := assertConsumerLossPolicy(t, song, []string{})
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if got := parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].AccidentalMode; got != mode {
				t.Fatalf("edited mode %v want %v", got, mode)
			}
		}
	}
}
func pitchSpellingSource(t *testing.T, reverse bool) []byte {
	t.Helper()
	base, err := Export(accidentalSong(t, NoteAccidentalDefault, 61), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	concert := `<Property name="ConcertPitch"><Pitch><Step>C</Step><Accidental>#</Accidental><Octave>5</Octave></Pitch></Property>`
	transposed := `<Property name="TransposedPitch"><Pitch><Step>D</Step><Accidental>b</Accidental><Octave>5</Octave></Pitch></Property>`
	if reverse {
		concert, transposed = transposed, concert
	}
	return rewriteConformanceGPIF(t, base, func(gpif string) string { return insertFirstNoteProperty(t, gpif, concert+transposed) })
}

type accidentalFact struct{ Track, Staff, Bar, Voice, Beat, Note, Mode, MIDI, Display, Fret, String, Articulation, PercussionMIDI int }

func TestAlphaTabPitchSpelling(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range accidentalCases() {
		data, _ := assertConsumerLossPolicy(t, accidentalSong(t, test.mode, test.midi), []string{})
		var facts []accidentalFact
		readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, data), &facts)
		if len(facts) != 1 || facts[0].Mode != test.consumer || facts[0].MIDI != int(test.midi) || facts[0].Display != int(test.midi) {
			t.Fatalf("raw %s facts %#v", test.enum, facts)
		}
	}
	for _, reverse := range []bool{false, true} {
		var facts []accidentalFact
		readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, pitchSpellingSource(t, reverse)), &facts)
		if len(facts) != 1 || facts[0].Mode != 5 || facts[0].MIDI != 61 {
			t.Fatalf("source priority %#v", facts)
		}
	}
	source, err := os.ReadFile("testdata/gp5/No Wah.gp5")
	if err != nil {
		t.Fatal(err)
	}
	song, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var before, after []accidentalFact
	readAlphaTabOracleFacts(t, "--accidental-facts", "testdata/gp5/No Wah.gp5", &before)
	readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, data), &after)
	if len(before) == 0 || len(after) == 0 || before[0].Mode != 0 || after[0].Mode != 0 || before[0].MIDI != after[0].MIDI {
		t.Fatalf("No Wah default spelling source %#v output %#v", before, after)
	}
}

func TestPitchSpellingContextPolicies(t *testing.T) {
	runConformancePitchSpellingContexts(newConformanceRun(t))
}
func runConformancePitchSpellingContexts(run *conformanceRun) {
	t := run.t
	for _, kind := range []string{"natural harmonic", "sounding transposition", "absolute note", "percussion"} {
		s, want, _ := accidentalContextSong(t, kind)
		note := &s.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
		data, report := assertConsumerLossPolicy(t, s, want)
		run.Report("M09-PITCH-CONTEXT", reportCodes(report), want)
		run.Wire("gpifProperty.Pitch", strings.Contains(string(readCurveGPIF(t, data)), "TransposedPitch"), false)
		run.Field("Note.AccidentalMode", note.AccidentalMode, NoteAccidentalNatural)
		if report.Entries[len(report.Entries)-1].Location != (ScoreLocation{}) {
			t.Fatalf("context loss location %#v", report)
		}
	}
}
func accidentalContextSong(t *testing.T, kind string) (*Song, []string, int) {
	t.Helper()
	s := accidentalSong(t, NoteAccidentalNatural, 60)
	note := &s.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	want := []string{"gp8.omit.note-accidental-context"}
	consumerMIDI := 60
	switch kind {
	case "natural harmonic":
		fret := 12.0
		note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: &fret}
	case "sounding transposition":
		s.Tracks[0].Staves[0].TranspositionPitch = 1
		want = []string{"gp8.omit.staff-sounding-transposition", "gp8.omit.note-accidental-context"}
	case "absolute note":
		note.String = 0
		note.Value = 61
		consumerMIDI = 0
	case "percussion":
		s.Tracks[0].PercussionTrack = true
		s.Tracks[0].Staves[0].PercussionTrack = true
		note.String = 0
		note.Value = 38
		consumerMIDI = 0 // The raw pitched-value getter is zero for percussion.
	default:
		t.Fatalf("unknown context %s", kind)
	}
	return s, want, consumerMIDI
}
func TestAlphaTabPitchSpellingContexts(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, kind := range []string{"natural harmonic", "sounding transposition", "absolute note", "percussion"} {
		song, codes, midi := accidentalContextSong(t, kind)
		data, _ := assertConsumerLossPolicy(t, song, codes)
		var facts []accidentalFact
		readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, data), &facts)
		if len(facts) != 1 || facts[0].Mode != 0 || facts[0].MIDI != midi || (kind == "percussion" && (facts[0].Articulation != 0 || facts[0].PercussionMIDI != 38)) {
			t.Fatalf("context %s raw consumer %#v", kind, facts)
		}
	}
}

func TestPitchSpellingDisplayContext(t *testing.T) {
	// C# sounding pitch becomes written Eb after the explicit display offset.
	s := accidentalSong(t, NoteAccidentalFlat, 61)
	staff := &s.Tracks[0].Staves[0]
	staff.DisplayTranspositionPitch = -2
	for i := range staff.Measures {
		staff.Measures[i].KeySignature = transposeKeySignature(s.MeasureHeaders[i].KeySignature, -2)
	}
	data, report, err := ExportWithReport(s, ExportFormatGP8, ExportOptions{})
	if err != nil || len(data) == 0 {
		t.Fatalf("display spelling %#v", report)
	}
	wire := extractGPIFLeafText(t, data)
	if wire["GPIF/Notes/Note/Properties/Property/Pitch/Step"] != "E" || wire["GPIF/Notes/Note/Properties/Property/Pitch/Accidental"] != "b" {
		t.Fatalf("written spelling %#v", wire)
	}
	requireAlphaTabConformance(t)
	var facts []accidentalFact
	readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, data), &facts)
	if len(facts) != 1 || facts[0].Mode != 5 || facts[0].MIDI != 61 || facts[0].Display != 63 {
		t.Fatalf("display raw consumer %#v", facts)
	}
}

func TestGPIFPitchSpellingInvalidSource(t *testing.T) {
	base := pitchSpellingSource(t, false)
	for _, mutation := range [][2]string{{"<Step>D</Step>", "<Step>E</Step>"}, {"<Step>D</Step>", "<Step>H</Step>"}, {"<Accidental>b</Accidental>", "<Accidental>?</Accidental>"}, {"<Octave>5</Octave>", "<Octave>12</Octave>"}} {
		data := rewriteConformanceGPIF(t, base, func(gpif string) string { return strings.Replace(gpif, mutation[0], mutation[1], 1) })
		result, err := ParseWithOptions(data, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData}})
		if err == nil || !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == "GPIF.Note.Pitch.Invalid" }) {
			t.Fatalf("invalid spelling %v diagnostics %#v error %v", mutation, result.Diagnostics, err)
		}
	}
}

func TestPitchSpellingGraceIsolation(t *testing.T) {
	s := accidentalSong(t, NoteAccidentalFlat, 61)
	n := &s.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	n.Effect.Graces = []GraceEffect{{Fret: 0, Duration: 32, Velocity: Forte}}
	data, _ := assertConsumerLossPolicy(t, s, []string{})
	requireAlphaTabConformance(t)
	var facts []accidentalFact
	readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, data), &facts)
	if len(facts) != 2 || facts[0].Mode != 0 || facts[0].MIDI != 60 || facts[1].Mode != 5 || facts[1].MIDI != 61 {
		t.Fatalf("owner spelling leaked to grace %#v", facts)
	}
}

func TestGPIFGraceSpellingLimit(t *testing.T) {
	result, err := ParseFileWithOptions("testdata/gp7/grace.gp", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
		return d.Code == "GPIF.Note.Pitch.Context" && strings.Contains(d.Reason, "ordered GraceEffect") && d.Location.NoteID != ""
	}) {
		t.Fatalf("missing precise grace-spelling limit %#v", result.Diagnostics)
	}
	// An orphan retains its Note, so the ordered-grace loss does not apply.
	grace := defaultBeat()
	grace.Notes = []Note{{String: 1, AccidentalMode: NoteAccidentalSharp}}
	context := &parseContext{}
	orphans := gpifApplyPendingGrace(nil, []gpifPendingGrace{{beat: grace, noteIDs: []string{"authored"}}}, false, context)
	if len(context.diagnostics) != 0 || len(orphans) != 1 || orphans[0].Notes[0].AccidentalMode != NoteAccidentalSharp {
		t.Fatalf("orphan lost mode %#v %#v", orphans, context.diagnostics)
	}
}

func TestGPIFNumberedPitchSpellingLimit(t *testing.T) {
	result, err := ParseFileWithOptions("testdata/gp7/numbered.gp", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
		return d.Code == "GPIF.Note.Pitch.Context" && d.Location.NoteID == "14" && strings.Contains(d.Reason, "negative derived fret")
	}) {
		t.Fatal("missing numbered-note spelling limitation")
	}
	n := result.Song.Tracks[0].Measures[0].Voices[0].Beats[14].Notes[0]
	if n.Value != 37 || n.String != 6 || n.AccidentalMode != NoteAccidentalDefault {
		t.Fatalf("negative-fret fallback changed %#v", n)
	}
}

func TestGPIFAbsentAccidentalAuthority(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		base, _ := assertConsumerLossPolicy(t, accidentalSong(t, NoteAccidentalDefault, 60), []string{})
		accidental := ""
		want, rawMode := NoteAccidentalDefault, 0
		if explicit {
			accidental = "<Accidental></Accidental>"
			want, rawMode = NoteAccidentalNatural, 2
		}
		source := rewriteConformanceGPIF(t, base, func(gpif string) string {
			return insertFirstNoteProperty(t, gpif, `<Property name="ConcertPitch"><Pitch><Step>D</Step><Accidental>bb</Accidental><Octave>5</Octave></Pitch></Property><Property name="TransposedPitch"><Pitch><Step>C</Step>`+accidental+`<Octave>5</Octave></Pitch></Property>`)
		})
		result, err := ParseWithOptions(source, ParseOptions{})
		if err != nil || result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].AccidentalMode != want || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "GPIF.Note.Pitch.Authority" {
			t.Fatalf("absent/empty precedence result %#v error %v", result, err)
		}
		if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
			var facts []accidentalFact
			readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, source), &facts)
			if len(facts) != 1 || facts[0].Mode != rawMode || facts[0].MIDI != 60 {
				t.Fatalf("absent/empty raw facts %#v", facts)
			}
		}
	}
}
