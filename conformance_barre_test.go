// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type conformanceBarreFact struct {
	Track int    `json:"track"`
	Staff int    `json:"staff"`
	Bar   int    `json:"bar"`
	Voice int    `json:"voice"`
	Beat  int    `json:"beat"`
	Fret  int    `json:"fret"`
	Shape string `json:"shape"`
}

type conformanceBarreWireFact struct {
	Fret   int
	String float64
}

func TestConformanceBarre(t *testing.T) {
	runConformanceBarre(newConformanceRun(t))
}

func runConformanceBarre(run *conformanceRun) {
	t := run.t
	source := parseTestFixture(t, "testdata/gp7/beat-barre.gp")
	sourceBeats := []Beat{
		source.Tracks[0].Measures[0].Voices[0].Beats[0],
		source.Tracks[0].Measures[1].Voices[0].Beats[0],
		source.Tracks[0].Measures[2].Voices[0].Beats[0],
	}
	wantSourceFrets := []Fret{3, 3, 24}
	wantSourceShapes := []BarreShape{BarreShapeFull, BarreShapeHalf, BarreShapeFull}
	run.ClaimPrimary(claimSite("barre", "import", "M10-BARRE", "distinct frets and full/half shapes")).Preserved("Beat.BarreFret", conformanceBarreFrets(t, sourceBeats), wantSourceFrets)
	run.Preserved("Beat.BarreShape", conformanceBarreShapes(sourceBeats), wantSourceShapes)
	archive, err := os.ReadFile("testdata/gp7/beat-barre.gp")
	if err != nil {
		t.Fatal(err)
	}
	reusedGPIF := strings.Replace(string(conformanceBarreGPIF(t, archive)), "<Beats>1</Beats>", "<Beats>0</Beats>", 1)
	reused, err := Parse(conformanceGPIFArchive(t, reusedGPIF))
	if err != nil {
		t.Fatal(err)
	}
	reusedFirst := &reused.Tracks[0].Measures[0].Voices[0].Beats[0]
	reusedSecond := &reused.Tracks[0].Measures[1].Voices[0].Beats[0]
	if reusedFirst.BarreFret == nil || reusedSecond.BarreFret == nil || reusedFirst.BarreFret == reusedSecond.BarreFret {
		t.Fatalf("reused beat barre fret ownership = %p, %p", reusedFirst.BarreFret, reusedSecond.BarreFret)
	}
	*reusedFirst.BarreFret = 8
	if *reusedSecond.BarreFret != 3 {
		t.Fatalf("editing one reused beat occurrence changed the other to %d", *reusedSecond.BarreFret)
	}

	// Beat-level and chord-diagram barres are independent authored values.
	song := semanticExportProbeSong(t)
	track := &song.Tracks[0]
	first := &track.Measures[0].Voices[0].Beats[0]
	firstFret := Fret(3)
	first.BarreFret = &firstFret
	first.BarreShape = BarreShapeFull
	first.Effect.Chord = &Chord{Name: "Independent chord", Strings: []int8{1, 1, 3, 3, 3, 1}, Barres: []Barre{{Fret: 3, Start: 3, End: 5}}}
	second := *first
	second.Notes = slices.Clone(first.Notes)
	second.Effect.Chord = nil
	secondFret := Fret(7)
	second.BarreFret = &secondFret
	second.BarreShape = BarreShapeHalf
	track.Measures[0].Voices[0].Beats = append(track.Measures[0].Voices[0].Beats, second)
	if err = FinalizeSong(song); err != nil {
		t.Fatal(err)
	}

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatalf("barre export: %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("barre", "export", "M10-BARRE", "distinct frets and full/half shapes")).Report("M10-BARRE", reportCodes(report), []string{})
	wire := conformanceBarreWire(t, data)
	wantWire := []conformanceBarreWireFact{{Fret: 3, String: 0}, {Fret: 7, String: 1}}
	run.ClaimSerialization(claimSite("barre", "export", "M10-BARRE", "distinct frets and full/half shapes")).Wire("gpifProperty.Fret", wire, wantWire)
	run.Wire("gpifProperty.String", conformanceBarreWireStrings(wire), []float64{0, 1})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripBeats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	run.ClaimPrimary(claimSite("barre", "model", "M10-BARRE", "distinct frets and full/half shapes"), claimSite("barre", "export", "M10-BARRE", "distinct frets and full/half shapes")).Preserved("Beat.BarreFret", conformanceBarreFrets(t, roundTripBeats), []Fret{3, 7})
	if got := conformanceBarreShapes(roundTripBeats); !slices.Equal(got, []BarreShape{BarreShapeFull, BarreShapeHalf}) {
		t.Fatalf("round-trip barre shapes = %v", got)
	}
	if roundTripBeats[0].Effect.Chord == nil || !reflect.DeepEqual(roundTripBeats[0].Effect.Chord.Barres, []Barre{{Fret: 3, Start: 3, End: 5}}) {
		t.Fatalf("round-trip chord = %#v, want independent diagram range", roundTripBeats[0].Effect.Chord)
	}
	if first.Effect.Chord.Barres[0].Fret != 3 || *first.BarreFret != 3 {
		t.Fatal("export mutated independent beat or chord barre data")
	}

	for _, boundaryFret := range []Fret{0, 1<<15 - 1} {
		boundary := semanticExportProbeSong(t)
		boundaryBeat := &boundary.Tracks[0].Measures[0].Voices[0].Beats[0]
		boundaryBeat.BarreFret = &boundaryFret
		boundaryBeat.BarreShape = BarreShapeHalf
		boundaryData, boundaryReport, boundaryErr := ExportWithReport(boundary, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if boundaryErr != nil || len(boundaryReport.Entries) != 0 {
			t.Fatalf("boundary barre fret %d export = %v, %#v", boundaryFret, boundaryErr, boundaryReport.Entries)
		}
		if got := conformanceBarreWire(t, boundaryData); !reflect.DeepEqual(got, []conformanceBarreWireFact{{Fret: int(boundaryFret), String: 1}}) {
			t.Fatalf("boundary barre fret %d wire = %#v", boundaryFret, got)
		}
	}

	for _, test := range []struct {
		name string
		set  func(*Beat)
		code string
	}{
		{name: "negative fret", set: func(beat *Beat) { fret := Fret(-1); beat.BarreFret = &fret; beat.BarreShape = BarreShapeFull }, code: "score.beat.barre-fret"},
		{name: "fret without shape", set: func(beat *Beat) { fret := Fret(1); beat.BarreFret = &fret }, code: "score.beat.barre-pair"},
		{name: "shape without fret", set: func(beat *Beat) { beat.BarreShape = BarreShapeHalf }, code: "score.beat.barre-pair"},
		{name: "unknown shape", set: func(beat *Beat) { fret := Fret(1); beat.BarreFret = &fret; beat.BarreShape = BarreShape(3) }, code: "score.beat.barre-shape"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			test.set(&song.Tracks[0].Measures[0].Voices[0].Beats[0])
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("barre diagnostics = %#v, want %s", diagnostics, test.code)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject."+test.code) {
				t.Fatalf("invalid barre export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
		})
	}

	run.Enum("BarreShape.BarreShapeNone", BarreShapeNone, BarreShape(0))
	run.Enum("BarreShape.BarreShapeFull", BarreShapeFull, BarreShape(1))
	run.Enum("BarreShape.BarreShapeHalf", BarreShapeHalf, BarreShape(2))
	for _, source := range []struct {
		wire float64
		want BarreShape
	}{{wire: 0, want: BarreShapeFull}, {wire: 1, want: BarreShapeHalf}} {
		beat := Beat{}
		gpifApplyBeatEffects(&gpifBeat{Properties: gpifProperties{Properties: []gpifProperty{{Name: "BarreString", String: &source.wire}}}}, &beat)
		run.Dispatch("gpifApplyBeatEffects:p.String", beat.BarreShape, source.want)
	}
	for _, name := range []string{"BarreFret", "BarreString"} {
		context := &parseContext{format: "GP8"}
		gpifAuditBarrePair(context, "beat", "/GPIF/Beats/Beat/Properties", []gpifProperty{{Name: name}})
		run.Dispatch("gpifAuditBarrePair:property.Name", len(context.diagnostics), 1)
	}
}

func conformanceBarreFrets(t *testing.T, beats []Beat) []Fret {
	t.Helper()
	result := make([]Fret, 0, len(beats))
	for index := range beats {
		if beats[index].BarreFret == nil {
			t.Fatalf("beat %d barre fret is nil", index)
		}
		result = append(result, *beats[index].BarreFret)
	}
	return result
}

func conformanceBarreShapes(beats []Beat) []BarreShape {
	result := make([]BarreShape, len(beats))
	for index := range beats {
		result[index] = beats[index].BarreShape
	}
	return result
}

func conformanceBarreWire(t *testing.T, data []byte) []conformanceBarreWireFact {
	t.Helper()
	var document gpifDocument
	if err := xml.Unmarshal(conformanceBarreGPIF(t, data), &document); err != nil {
		t.Fatal(err)
	}
	var result []conformanceBarreWireFact
	for _, beat := range document.Beats.Beats {
		var fact conformanceBarreWireFact
		foundFret := false
		foundString := false
		for _, property := range beat.Properties.Properties {
			switch property.Name {
			case "BarreFret":
				if property.Fret != nil {
					fact.Fret = *property.Fret
					foundFret = true
				}
			case "BarreString":
				if property.String != nil {
					fact.String = *property.String
					foundString = true
				}
			}
		}
		if foundFret || foundString {
			if !foundFret || !foundString {
				t.Fatalf("incomplete barre wire on beat %s", beat.ID)
			}
			result = append(result, fact)
		}
	}
	return result
}

func conformanceBarreWireStrings(facts []conformanceBarreWireFact) []float64 {
	result := make([]float64, len(facts))
	for index := range facts {
		result[index] = facts[index].String
	}
	return result
}

func TestAlphaTabPreservesBeatBarres(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture := "testdata/gp7/beat-barre.gp"
	var source []conformanceBarreFact
	readAlphaTabOracleFacts(t, "--barre", fixture, &source)
	want := []conformanceBarreFact{
		{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 0, Fret: 3, Shape: "full"},
		{Track: 0, Staff: 0, Bar: 1, Voice: 0, Beat: 0, Fret: 3, Shape: "half"},
		{Track: 0, Staff: 0, Bar: 2, Voice: 0, Beat: 0, Fret: 24, Shape: "full"},
	}
	if !reflect.DeepEqual(source, want) {
		t.Fatalf("AlphaTab source barre facts = %#v, want %#v", source, want)
	}

	song := parseTestFixture(t, fixture)
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceBarreFact
	readAlphaTabOracleFacts(t, "--barre", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output barre facts = %#v, want %#v", output, source)
	}

	editedBeat := &song.Tracks[0].Measures[1].Voices[0].Beats[0]
	editedFret := Fret(8)
	editedBeat.BarreFret = &editedFret
	editedBeat.BarreShape = BarreShapeFull
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--barre", writeConformanceFixture(t, edited), &output)
	if len(output) != len(source) || output[1].Fret != 8 || output[1].Shape != "full" || !reflect.DeepEqual(output[:1], source[:1]) || !reflect.DeepEqual(output[2:], source[2:]) {
		t.Fatalf("AlphaTab edited barre facts = %#v", output)
	}
	conformanceIndependentClaim(t, "field:Beat.BarreFret", claimAllStages("barre", "M10-BARRE", "distinct frets and full/half shapes")...)
}

func TestGPIFBarreDiagnostics(t *testing.T) {
	archive, err := os.ReadFile("testdata/gp7/beat-barre.gp")
	if err != nil {
		t.Fatal(err)
	}
	source := conformanceBarreGPIF(t, archive)
	for _, test := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "missing fret payload", old: "<Fret>3</Fret>", new: "", code: "GPIF.Beat.Property.BarreFret.MissingPayload"},
		{name: "negative fret", old: "<Fret>3</Fret>", new: "<Fret>-1</Fret>", code: "GPIF.Beat.Property.BarreFret.InvalidValue"},
		{name: "overflowing fret", old: "<Fret>3</Fret>", new: "<Fret>32768</Fret>", code: "GPIF.Beat.Property.BarreFret.InvalidValue"},
		{name: "missing shape payload", old: "<String>0</String>", new: "", code: "GPIF.Beat.Property.BarreString.MissingPayload"},
		{name: "fractional shape", old: "<String>0</String>", new: "<String>0.5</String>", code: "GPIF.Beat.Property.BarreString.InvalidValue"},
		{name: "unknown shape", old: "<String>0</String>", new: "<String>2</String>", code: "GPIF.Beat.Property.BarreString.InvalidValue"},
		{name: "missing fret property", old: "<Property name=\"BarreFret\">\n<Fret>3</Fret>\n</Property>", new: "", code: "GPIF.Beat.Property.Barre.Incomplete"},
		{name: "missing shape property", old: "<Property name=\"BarreString\">\n<String>0</String>\n</Property>", new: "", code: "GPIF.Beat.Property.Barre.Incomplete"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := strings.Replace(string(source), test.old, test.new, 1)
			if mutated == string(source) {
				t.Fatalf("source fragment %q was not found", test.old)
			}
			result, err := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{})
			if err != nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("barre source diagnostics = %#v, %v; want %s", result.Diagnostics, err, test.code)
			}
			_, strictErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{Strict: true})
			var strict *StrictParseError
			if !errors.As(strictErr, &strict) {
				t.Fatalf("strict barre parse = %v, want StrictParseError", strictErr)
			}
		})
	}
}

func conformanceBarreGPIF(t *testing.T, data []byte) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return readZipMember(t, archive, "Content/score.gpif")
}
