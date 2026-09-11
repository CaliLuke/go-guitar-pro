// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"reflect"
	"strings"
	"testing"
)

const ornamentCase = "M11-NOTE-ORNAMENTS"
const ornamentValue = "all note ornaments"

func TestConformanceNoteOrnaments(t *testing.T) { runConformanceNoteOrnaments(newConformanceRun(t)) }
func runConformanceNoteOrnaments(run *conformanceRun) {
	t := run.t
	for _, c := range []struct {
		kind         NoteOrnament
		member, wire string
	}{{NoteOrnamentNone, "NoteOrnamentNone", ""}, {NoteOrnamentTurn, "NoteOrnamentTurn", "Turn"}, {NoteOrnamentInvertedTurn, "NoteOrnamentInvertedTurn", "InvertedTurn"}, {NoteOrnamentUpperMordent, "NoteOrnamentUpperMordent", "UpperMordent"}, {NoteOrnamentLowerMordent, "NoteOrnamentLowerMordent", "LowerMordent"}} {
		song := semanticExportProbeSong(t)
		song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Ornament = c.kind
		data, report := assertConsumerLossPolicy(t, song, []string{})
		run.ClaimReport(claimSite("ornaments", "export", ornamentCase, ornamentValue)).Report(ornamentCase, reportCodes(report), []string{})
		doc := conformanceWireDocument(t, data)
		run.ClaimSerialization(claimSite("ornaments", "export", ornamentCase, ornamentValue)).Wire("gpifNote.Ornament", doc.Notes.Notes[0].Ornament, c.wire)
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Ornament
		run.ClaimPrimary(claimAllStages("ornaments", ornamentCase, ornamentValue)...).Preserved("Note.Ornament", got, c.kind)
		run.Enum("NoteOrnament."+c.member, got, c.kind)
		run.Dispatch("gpifNoteOrnament:n.Ornament", gpifNoteOrnament(gpifNote{Ornament: c.wire}), c.kind)
		if c.kind == NoteOrnamentTurn {
			raw := strings.Replace(string(conformanceBarreGPIF(t, data)), "<Ornament>Turn</Ornament>", "<Ornament>UnspecifiedTurn</Ornament>", 1)
			invalid := conformanceGPIFArchive(t, raw)
			result, parseErr := ParseWithOptions(invalid, ParseOptions{})
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			found := false
			for _, d := range result.Diagnostics {
				if d.Code == "GPIF.Note.Ornament.InvalidValue" {
					found = true
					if d.ObjectID != "0" || d.Kind != ParseDiagnosticUnsupportedFeature {
						t.Fatal(d)
					}
				}
			}
			if !found {
				t.Fatal("unknown ornament lacks diagnostic")
			}
			if _, strictErr := ParseWithOptions(invalid, ParseOptions{Strict: true}); strictErr == nil {
				t.Fatal("unknown ornament accepted strictly")
			}
		}
	}
}

type ornamentFact struct {
	Track, Staff, Bar, Voice, Beat, Note int
	Ornament                             string
	String, Fret                         int
}

func TestAlphaTabNoteOrnaments(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, c := range []struct {
		kind NoteOrnament
		want string
	}{{NoteOrnamentNone, "None"}, {NoteOrnamentInvertedTurn, "InvertedTurn"}, {NoteOrnamentTurn, "Turn"}, {NoteOrnamentUpperMordent, "UpperMordent"}, {NoteOrnamentLowerMordent, "LowerMordent"}} {
		song := semanticExportProbeSong(t)
		song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Ornament = c.kind
		data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil {
			t.Fatal(err)
		}
		var facts []ornamentFact
		readAlphaTabOracleFacts(t, "--note-ornaments", writeConformanceFixture(t, data), &facts)
		want := []ornamentFact{{Ornament: c.want, String: 6, Fret: 3}}
		if !reflect.DeepEqual(facts, want) {
			t.Fatalf("consumer %#v want %#v", facts, want)
		}
	}
	var source, output []ornamentFact
	readAlphaTabOracleFacts(t, "--note-ornaments", "testdata/gp7/ornaments.gp", &source)
	wants := []string{"UpperMordent", "UpperMordent", "UpperMordent", "LowerMordent", "LowerMordent", "LowerMordent", "InvertedTurn", "InvertedTurn", "InvertedTurn", "Turn", "Turn", "Turn"}
	if len(source) != len(wants) {
		t.Fatal(source)
	}
	for i, want := range wants {
		if source[i].Ornament != want {
			t.Fatal(source)
		}
	}
	song := parseTestFixture(t, "testdata/gp7/ornaments.gp")
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--note-ornaments", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(source, output) {
		t.Fatalf("source/output ornaments differ %#v %#v", source, output)
	}
	conformanceIndependentClaim(t, "field:Note.Ornament", claimAllStages("ornaments", ornamentCase, ornamentValue)...)
}
