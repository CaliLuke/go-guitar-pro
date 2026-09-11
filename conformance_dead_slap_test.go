// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const (
	conformanceDeadSlapCase  = "M10-DEAD-SLAP"
	conformanceDeadSlapValue = "note-free authored dead slap"
)

type conformanceDeadSlapFact struct {
	Track       int  `json:"track"`
	Staff       int  `json:"staff"`
	Bar         int  `json:"bar"`
	Voice       int  `json:"voice"`
	Beat        int  `json:"beat"`
	DeadSlapped bool `json:"deadSlapped"`
	IsEmpty     bool `json:"isEmpty"`
	IsRest      bool `json:"isRest"`
	NoteCount   int  `json:"noteCount"`
}

func TestConformanceDeadSlap(t *testing.T) {
	runConformanceDeadSlap(newConformanceRun(t))
}

func runConformanceDeadSlap(run *conformanceRun) {
	t := run.t
	colors := parseTestFixture(t, "testdata/gp7/colors.gp")
	colorsBeat := colors.Tracks[0].Staves[0].Measures[1].Voices[0].Beats[1]
	run.ClaimPrimary(claimSite("dead-slap", "import", conformanceDeadSlapCase, conformanceDeadSlapValue)).Preserved("Beat.DeadSlapped", colorsBeat.DeadSlapped, true)
	if colorsBeat.Status != BeatStatusNormal || len(colorsBeat.Notes) != 0 {
		t.Fatalf("colors dead slap = status %d with %d notes, want normal with no notes", colorsBeat.Status, len(colorsBeat.Notes))
	}

	fixture := parseTestFixture(t, "testdata/gp7/dead-slap.gp")
	deadBeats := conformanceDeadSlapBeats(fixture)
	if len(deadBeats) != 9 {
		t.Fatalf("dead-slap fixture has %d authored occurrences, want 9", len(deadBeats))
	}
	for index, beat := range deadBeats {
		if beat.Status != BeatStatusNormal || len(beat.Notes) != 0 {
			t.Fatalf("dead-slap occurrence %d = status %d with %d notes, want normal with no notes", index, beat.Status, len(beat.Notes))
		}
	}

	song := semanticExportProbeSong(t)
	track := &song.Tracks[0]
	quarter := defaultDuration()
	track.Measures[0].Voices = []Voice{{Beats: []Beat{
		{Duration: quarter, Status: BeatStatusRest},
		{Duration: quarter, Status: BeatStatusEmpty},
		{Duration: quarter, Status: BeatStatusNormal, DeadSlapped: true},
	}}}
	track.Staves[0].Measures = track.Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	before := *song
	before.Tracks = slices.Clone(song.Tracks)
	before.Tracks[0].Measures = slices.Clone(song.Tracks[0].Measures)
	before.Tracks[0].Measures[0].Voices = slices.Clone(song.Tracks[0].Measures[0].Voices)
	before.Tracks[0].Measures[0].Voices[0].Beats = slices.Clone(song.Tracks[0].Measures[0].Voices[0].Beats)

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if !hasExportCode(report, "gp8.normalize.empty-beat") || len(report.Entries) != 1 {
		t.Fatalf("rest/empty/dead report = %#v, want only existing empty-beat normalization", report.Entries)
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        []string{"gp8.normalize.empty-beat"},
	}})
	if err != nil {
		t.Fatalf("dead-slap export: %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("dead-slap", "export", conformanceDeadSlapCase, conformanceDeadSlapValue)).Report(conformanceDeadSlapCase, reportCodes(report), []string{"gp8.normalize.empty-beat"})
	rawGPIF := string(conformanceBarreGPIF(t, data))
	if count := strings.Count(rawGPIF, "<DeadSlapped"); count != 1 {
		t.Fatalf("DeadSlapped marker count = %d, want 1", count)
	}
	deadStart := strings.Index(rawGPIF, "<Beat id=\"2\">")
	if deadStart < 0 {
		t.Fatal("dead-slap beat definition is absent")
	}
	deadEnd := strings.Index(rawGPIF[deadStart:], "</Beat>")
	if deadEnd < 0 || strings.Contains(rawGPIF[deadStart:deadStart+deadEnd], "<Notes") {
		t.Fatal("note-free dead slap emitted a Notes element")
	}

	wire := conformanceWireDocument(t, data).Beats.Beats
	if len(wire) != 3 {
		t.Fatalf("wire beats = %d, want 3", len(wire))
	}
	run.ClaimSerialization(claimSite("dead-slap", "export", conformanceDeadSlapCase, conformanceDeadSlapValue)).Wire("gpifBeat.DeadSlapped", []bool{wire[0].DeadSlapped != nil, wire[1].DeadSlapped != nil, wire[2].DeadSlapped != nil}, []bool{false, false, true})
	if wire[2].Notes != "" {
		t.Fatalf("dead-slap wire fabricated Notes %q", wire[2].Notes)
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripBeats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	run.ClaimPrimary(claimSite("dead-slap", "model", conformanceDeadSlapCase, conformanceDeadSlapValue), claimSite("dead-slap", "export", conformanceDeadSlapCase, conformanceDeadSlapValue)).Preserved("Beat.DeadSlapped", []bool{roundTripBeats[0].DeadSlapped, roundTripBeats[1].DeadSlapped, roundTripBeats[2].DeadSlapped}, []bool{false, false, true})
	if got := []BeatStatus{roundTripBeats[0].Status, roundTripBeats[1].Status, roundTripBeats[2].Status}; !slices.Equal(got, []BeatStatus{BeatStatusRest, BeatStatusRest, BeatStatusNormal}) {
		t.Fatalf("round-trip rest/empty/dead statuses = %v", got)
	}
	if len(roundTripBeats[2].Notes) != 0 {
		t.Fatalf("round-trip dead slap fabricated %d notes", len(roundTripBeats[2].Notes))
	}
	if !reflect.DeepEqual(*song, before) {
		t.Fatal("dead-slap preflight or export mutated the public song")
	}

	deadOnly := semanticExportProbeSong(t)
	deadOnlyBeat := &deadOnly.Tracks[0].Measures[0].Voices[0].Beats[0]
	deadOnlyBeat.Status = BeatStatusNormal
	deadOnlyBeat.Notes = nil
	deadOnlyBeat.DeadSlapped = true
	deadOnlyData, deadOnlyReport, deadOnlyErr := ExportWithReport(deadOnly, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if deadOnlyErr != nil || len(deadOnlyData) == 0 || len(deadOnlyReport.Entries) != 0 {
		t.Fatalf("strict dead-slap-only export = %d bytes, %#v, %v", len(deadOnlyData), deadOnlyReport.Entries, deadOnlyErr)
	}
	deadNote := Beat{Notes: []Note{{Kind: NoteTypeDead}}}
	slapEffect := Beat{Effect: BeatEffects{SlapEffect: SlapEffectSlapping}}
	if deadNote.DeadSlapped || len(deadNote.Notes) != 1 || deadNote.Notes[0].Kind != NoteTypeDead {
		t.Fatal("dead-note state implied a dead-slap marker or was changed")
	}
	if slapEffect.DeadSlapped || slapEffect.Effect.SlapEffect != SlapEffectSlapping {
		t.Fatal("dead-note or slap-effect state implied a dead-slap marker")
	}

	conformanceDeadSlapPresenceSpelling(t, data)
	conformanceDeadSlapDirectToggle(t, colors)
}

func conformanceDeadSlapPresenceSpelling(t *testing.T, data []byte) {
	t.Helper()
	gpif := string(conformanceBarreGPIF(t, data))
	gpif = strings.Replace(gpif, "<DeadSlapped></DeadSlapped>", "<DeadSlapped>false</DeadSlapped>", 1)
	if !strings.Contains(gpif, "<DeadSlapped>false</DeadSlapped>") {
		t.Fatal("could not create text-bearing DeadSlapped spelling")
	}
	result, err := ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict parse of present false-text marker: %v", err)
	}
	beat := result.Song.Tracks[0].Measures[0].Voices[0].Beats[2]
	if !beat.DeadSlapped || beat.Status != BeatStatusNormal || len(beat.Notes) != 0 {
		t.Fatalf("present false-text marker = %#v", beat)
	}
}

func conformanceDeadSlapDirectToggle(t *testing.T, song *Song) {
	t.Helper()
	beat := &song.Tracks[0].Measures[1].Voices[0].Beats[1]
	beforeStatus, beforeNotes := beat.Status, slices.Clone(beat.Notes)
	beat.DeadSlapped = false
	without, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(conformanceBarreGPIF(t, without)), "<DeadSlapped") {
		t.Fatal("cleared direct edit still emitted DeadSlapped")
	}
	beat.DeadSlapped = true
	with, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conformanceBarreGPIF(t, with)), "<DeadSlapped") {
		t.Fatal("restored direct edit did not emit DeadSlapped")
	}
	if beat.Status != beforeStatus || !reflect.DeepEqual(beat.Notes, beforeNotes) {
		t.Fatal("direct dead-slap edits changed beat structure")
	}
}

func conformanceDeadSlapBeats(song *Song) []Beat {
	var result []Beat
	for _, track := range song.Tracks {
		for _, staff := range track.Staves {
			for _, measure := range staff.Measures {
				for _, voice := range measure.Voices {
					for _, beat := range voice.Beats {
						if beat.DeadSlapped {
							result = append(result, beat)
						}
					}
				}
			}
		}
	}
	return result
}

func TestAlphaTabPreservesDeadSlap(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range []struct {
		fixture string
		count   int
	}{
		{fixture: "testdata/gp7/colors.gp", count: 1},
		{fixture: "testdata/gp7/dead-slap.gp", count: 9},
	} {
		var source []conformanceDeadSlapFact
		readAlphaTabOracleFacts(t, "--dead-slap", test.fixture, &source)
		if len(source) != test.count {
			t.Fatalf("AlphaTab %s dead-slap facts = %d, want %d", test.fixture, len(source), test.count)
		}
		if slices.ContainsFunc(source, func(fact conformanceDeadSlapFact) bool {
			return !fact.DeadSlapped || fact.IsEmpty || fact.IsRest || fact.NoteCount != 0
		}) {
			t.Fatalf("AlphaTab source has noncanonical dead-slap fact: %#v", source)
		}
		song := parseTestFixture(t, test.fixture)
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		var output []conformanceDeadSlapFact
		readAlphaTabOracleFacts(t, "--dead-slap", writeConformanceFixture(t, data), &output)
		if !reflect.DeepEqual(output, source) {
			t.Fatalf("AlphaTab output dead-slap facts = %#v, want %#v", output, source)
		}
	}
	conformanceIndependentClaim(t, "field:Beat.DeadSlapped", claimAllStages("dead-slap", conformanceDeadSlapCase, conformanceDeadSlapValue)...)
}

func TestDeadSlapHasNoBinarySource(t *testing.T) {
	for _, fixture := range []string{"testdata/gp3/Effects.gp3", "testdata/gp4/Effects.gp4", "testdata/gp5/Effects.gp5"} {
		if _, err := os.Stat(fixture); err != nil {
			t.Fatal(err)
		}
		if got := conformanceDeadSlapBeats(parseTestFixture(t, fixture)); len(got) != 0 {
			t.Fatalf("%s synthesized %d dead-slap beats", fixture, len(got))
		}
	}
}

func TestDeadSlapStrictParseRejectsUnknownNeighborOnly(t *testing.T) {
	song := semanticExportProbeSong(t)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.DeadSlapped = true
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	known, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil || slices.ContainsFunc(known.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.UnknownElement.NoteAndBeat" && strings.Contains(diagnostic.SourcePath, "DeadSlapped")
	}) {
		t.Fatalf("recognized DeadSlapped diagnostics = %#v, %v", known.Diagnostics, err)
	}
	gpif := strings.Replace(string(conformanceBarreGPIF(t, data)), "<DeadSlapped></DeadSlapped>", "<UnknownDeadSlapped></UnknownDeadSlapped>", 1)
	_, err = ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{Strict: true})
	var strict *StrictParseError
	if !errors.As(err, &strict) {
		t.Fatalf("unknown neighbor strict parse = %v, want StrictParseError", err)
	}
}
