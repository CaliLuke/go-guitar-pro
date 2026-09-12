// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"errors"
	"math"
	"os"
	"slices"
	"strings"
	"testing"
)

type conformanceAlphaTabTranspositionStaff struct {
	Track                 int   `json:"track"`
	Staff                 int   `json:"staff"`
	TranspositionPitch    int32 `json:"transpositionPitch"`
	DisplayPitch          int32 `json:"displayTranspositionPitch"`
	Keys                  []int `json:"keys"`
	FirstNoteString       int   `json:"firstNoteString"`
	FirstNoteFret         int   `json:"firstNoteFret"`
	FirstNoteSoundingMIDI int   `json:"firstNoteSoundingMidi"`
}

func TestConformanceTransposition(t *testing.T) {
	runConformanceTransposition(newConformanceRun(t))
}

func runConformanceTransposition(run *conformanceRun) {
	for key := int8(-7); key <= 7; key++ {
		for display := int32(-12); display <= 12; display++ {
			written := transposeKeySignature(KeySignature{Key: key}, display)
			run.Normalized("Measure.KeySignature", ((7*int(written.Key))%12+12)%12, ((7*int(key)-int(display))%12+12)%12)
		}
	}
	t := run.t
	source := conformanceTranspositionGPIF()
	result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("source diagnostics = %#v", result.Diagnostics)
	}

	wantDisplay := [][]int32{{4, 4}, {-13}, {14, 14}}
	wantKeys := [][]int8{{-4, -4}, {-5}, {-2, -2}}
	for trackIndex := range result.Song.Tracks {
		track := &result.Song.Tracks[trackIndex]
		for staffIndex := range track.Staves {
			staff := &track.Staves[staffIndex]
			run.ClaimPrimary(claimSite("transposition", "import", "M04-TRANSPOSITION", "distinct sounding and display offsets")).Preserved("Staff.DisplayTranspositionPitch", staff.DisplayTranspositionPitch, wantDisplay[trackIndex][staffIndex])
			run.Omitted("Staff.TranspositionPitch", staff.TranspositionPitch, int32(0))
			run.Normalized("Measure.KeySignature", staff.Measures[0].KeySignature.Key, wantKeys[trackIndex][staffIndex])
		}
	}
	if got := result.Song.MeasureHeaders[0].KeySignature.Key; got != 0 {
		t.Fatalf("concert master key = %d, want 0", got)
	}

	var sourceWire gpifDocument
	if unmarshalErr := xml.Unmarshal([]byte(source), &sourceWire); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	run.Wire("gpifTrack.Transpose", sourceWire.Tracks.Tracks[0].Transpose != nil, true)
	run.Wire("gpifTranspose.Chromatic", sourceWire.Tracks.Tracks[0].Transpose.Chromatic, int64(4))
	run.Wire("gpifTranspose.Octave", sourceWire.Tracks.Tracks[1].Transpose.Octave, int64(-2))

	// Staff-local edits are authoritative and never rewrite tuning/string/fret.
	track0 := &result.Song.Tracks[0]
	noteBefore := track0.Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	stringsBefore := slices.Clone(track0.Staves[0].Strings)
	track0.Staves[0].DisplayTranspositionPitch = -13
	track0.Staves[0].TranspositionPitch = 2
	track2 := &result.Song.Tracks[2]
	track2.Staves[1].DisplayTranspositionPitch = 5
	track2.Staves[1].TranspositionPitch = -12
	if got := track0.Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]; got.String != noteBefore.String || got.Value != noteBefore.Value || !slices.Equal(track0.Staves[0].Strings, stringsBefore) {
		t.Fatalf("transposition edit changed string/fret authority: note=%#v strings=%#v", got, track0.Staves[0].Strings)
	}
	run.Preserved("Staff.DisplayTranspositionPitch", track0.Staves[0].DisplayTranspositionPitch, int32(-13))
	run.Omitted("Staff.TranspositionPitch", track0.Staves[0].TranspositionPitch, int32(2))

	data, report, err := ExportWithReport(result.Song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"gp8.omit.staff-sounding-transposition", "gp8.omit.staff-display-transposition"} {
		if !hasExportCode(report, code) {
			t.Fatalf("export report lacks %s: %#v", code, report.Entries)
		}
	}
	wire := conformanceTranspositionWire(t, data)
	if got := *wire.Tracks.Tracks[0].Transpose; got != (gpifTranspose{Chromatic: 11, Octave: -2}) {
		t.Fatalf("edited negative transpose wire = %#v", got)
	}
	if got := *wire.Tracks.Tracks[2].Transpose; got != (gpifTranspose{Chromatic: 2, Octave: 1}) {
		t.Fatalf("positive octave transpose wire = %#v", got)
	}
	run.Wire("gpifTrack.Transpose", wire.Tracks.Tracks[0].Transpose != nil, true)
	run.Wire("gpifTranspose.Chromatic", wire.Tracks.Tracks[0].Transpose.Chromatic, int64(11))
	run.Wire("gpifTranspose.Octave", wire.Tracks.Tracks[0].Transpose.Octave, int64(-2))

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimPrimary(claimSite("transposition", "model", "M04-TRANSPOSITION", "distinct sounding and display offsets")).Preserved("Staff.DisplayTranspositionPitch", roundTrip.Tracks[0].Staves[0].DisplayTranspositionPitch, int32(-13))
	run.Omitted("Staff.TranspositionPitch", roundTrip.Tracks[0].Staves[0].TranspositionPitch, int32(0))
	if got := roundTrip.Tracks[0].Staves[0].Measures[0].KeySignature.Key; got != -5 {
		t.Fatalf("edited effective key = %d, want -5", got)
	}
	if got := roundTrip.Tracks[2].Staves[1].DisplayTranspositionPitch; got != 14 {
		t.Fatalf("later-staff target projection = %d, want track display 14", got)
	}

	partSounding := conformancePartSoundingGPIF("D")
	partResult, err := ParseWithOptions(conformanceGPIFArchive(t, partSounding), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := partResult.Song.Tracks[0].Staves[0].DisplayTranspositionPitch; got != -12 {
		t.Fatalf("PartSounding display transposition = %d, want -12", got)
	}
	if got := partResult.Song.Tracks[0].Staves[0].Measures[0].KeySignature.Key; got != -2 {
		t.Fatalf("PartSounding effective key = %d, want -2", got)
	}
	var partWire gpifDocument
	if unmarshalErr := xml.Unmarshal([]byte(partSounding), &partWire); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	run.Wire("gpifTrack.PartSounding", partWire.Tracks.Tracks[0].PartSounding != nil, true)
	run.Wire("gpifPartSounding.TranspositionPitch", partWire.Tracks.Tracks[0].PartSounding.TranspositionPitch, int64(-12))
	run.Wire("gpifPartSounding.NominalKey", partWire.Tracks.Tracks[0].PartSounding.NominalKey, "D")
	_, partReport, err := ExportWithReport(partResult.Song, ExportFormatGP8, ExportOptions{})
	if err != nil || !hasExportCode(partReport, "gp8.normalize.measure-key-authority") {
		t.Fatalf("PartSounding independent-key export report = %#v, %v", partReport, err)
	}
	zeroDisplay := strings.Replace(partSounding, `<TranspositionPitch>-12</TranspositionPitch>`, `<TranspositionPitch>0</TranspositionPitch>`, 1)
	zeroResult, err := Parse(conformanceGPIFArchive(t, zeroDisplay))
	if err != nil || zeroResult.Tracks[0].Staves[0].DisplayTranspositionPitch != 0 || zeroResult.Tracks[0].Staves[0].Measures[0].KeySignature.Key != -2 {
		t.Fatalf("zero-display independent key parse = %#v, %v", zeroResult, err)
	}
	_, zeroReport, err := ExportWithReport(zeroResult, ExportFormatGP8, ExportOptions{})
	if err != nil || !hasExportCode(zeroReport, "gp8.normalize.measure-key-authority") {
		t.Fatalf("zero-display independent-key export report = %#v, %v", zeroReport, err)
	}

	neutral := conformancePartSoundingGPIF("Neutral")
	neutralResult, err := ParseWithOptions(conformanceGPIFArchive(t, neutral), ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict Neutral PartSounding parse: %v", err)
	}
	if staff := neutralResult.Song.Tracks[0].Staves[0]; staff.DisplayTranspositionPitch != -12 || staff.Measures[0].KeySignature.Key != 0 {
		t.Fatalf("Neutral PartSounding = display %d key %d, want -12/0", staff.DisplayTranspositionPitch, staff.Measures[0].KeySignature.Key)
	}

	invalid := strings.Replace(conformanceTranspositionGPIF(), `<Chromatic>4</Chromatic><Octave>0</Octave>`, `<Chromatic>0</Chromatic><Octave>2147483648</Octave>`, 1)
	if _, err := Parse(conformanceGPIFArchive(t, invalid)); err == nil {
		t.Fatal("out-of-range octave transpose was accepted")
	}
	invalidPart := strings.Replace(partSounding, `<TranspositionPitch>-12</TranspositionPitch>`, `<TranspositionPitch>2147483648</TranspositionPitch>`, 1)
	if _, err := Parse(conformanceGPIFArchive(t, invalidPart)); err == nil {
		t.Fatal("out-of-range PartSounding transpose was accepted")
	}
	for _, pitch := range []int32{math.MinInt32, math.MaxInt32} {
		wireTranspose := gp8Transpose(pitch)
		got, boundaryErr := checkedGPIFTransposition(wireTranspose.Chromatic, wireTranspose.Octave)
		if boundaryErr != nil || got != pitch {
			t.Fatalf("transpose boundary %d round trip = %d, %v (wire %#v)", pitch, got, boundaryErr, wireTranspose)
		}
	}
	minWire := gp8Transpose(math.MinInt32)
	if _, err := checkedGPIFTransposition(minWire.Chromatic-1, minWire.Octave); err == nil {
		t.Fatal("transpose below math.MinInt32 was accepted")
	}
	maxWire := gp8Transpose(math.MaxInt32)
	if _, err := checkedGPIFTransposition(maxWire.Chromatic+1, maxWire.Octave); err == nil {
		t.Fatal("transpose above math.MaxInt32 was accepted")
	}

	badPitch := result.Song
	badPitch.Tracks[0].Staves[0].TranspositionPitch = math.MaxInt32
	diagnostics := ValidateSong(badPitch)
	if !hasScoreDiagnostic(diagnostics, "score.note.sounding-midi") {
		t.Fatalf("sounding pitch diagnostics = %#v", diagnostics)
	}

	both := strings.Replace(partSounding, `</PartSounding>`, `</PartSounding><Transpose><Chromatic>4</Chromatic><Octave>0</Octave></Transpose>`, 1)
	bothResult, bothErr := ParseWithOptions(conformanceGPIFArchive(t, both), ParseOptions{})
	if bothErr != nil || conformanceSourceAuditDiagnosticByCode(bothResult.Diagnostics, "GPIF.Track.Transposition.ConflictingValues") == nil || conformanceSourceAuditDiagnosticByCode(bothResult.Diagnostics, "GPIF.Track.PartSounding.NominalKey.Conflict") == nil || bothResult.Song.Tracks[0].Staves[0].DisplayTranspositionPitch != 4 || bothResult.Song.Tracks[0].Staves[0].Measures[0].KeySignature.Key != -4 {
		t.Fatalf("conflicting Transpose precedence = %#v, %v", bothResult, bothErr)
	}
	_, strictErr := ParseWithOptions(conformanceGPIFArchive(t, both), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticLossyProjection}})
	var policyErr *StrictParseError
	if !errors.As(strictErr, &policyErr) {
		t.Fatalf("strict conflicting Transpose parse = %v", strictErr)
	}
	invalidNominal := strings.Replace(partSounding, `<NominalKey>D</NominalKey>`, `<NominalKey>H</NominalKey>`, 1)
	invalidNominalResult, invalidNominalErr := ParseWithOptions(conformanceGPIFArchive(t, invalidNominal), ParseOptions{})
	if invalidNominalErr != nil || conformanceSourceAuditDiagnosticByCode(invalidNominalResult.Diagnostics, "GPIF.Track.PartSounding.NominalKey.InvalidValue") == nil {
		t.Fatalf("invalid PartSounding key = %#v, %v", invalidNominalResult, invalidNominalErr)
	}
}

func TestAlphaTabPreservesTransposition(t *testing.T) {
	requireAlphaTabConformance(t)
	const upstream = "references/alphaTab/packages/alphatab/test-data/guitarpro8/e.gp"
	data, err := os.ReadFile(upstream)
	if err != nil {
		t.Fatal(err)
	}
	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if staff := song.Tracks[0].Staves[0]; staff.DisplayTranspositionPitch != 4 || staff.Measures[0].KeySignature.Key != -4 {
		t.Fatalf("e.gp transposition = display %d key %d, want 4/-4", staff.DisplayTranspositionPitch, staff.Measures[0].KeySignature.Key)
	}
	exported, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if hasExportCode(report, "gp8.omit.staff-display-transposition") {
		t.Fatalf("single-staff display transposition reported as omitted: %#v", report.Entries)
	}
	var facts []conformanceAlphaTabTranspositionStaff
	readAlphaTabOracleFacts(t, "--transposition", writeConformanceFixture(t, exported), &facts)
	if len(facts) != 1 || facts[0].DisplayPitch != 4 || facts[0].TranspositionPitch != 0 || !slices.Equal(facts[0].Keys, []int{-4}) || facts[0].FirstNoteString != 5 || facts[0].FirstNoteFret != 1 || facts[0].FirstNoteSoundingMIDI != 60 {
		t.Fatalf("AlphaTab transposition facts = %#v", facts)
	}
	for _, test := range []struct {
		nominal string
		key     int
	}{{nominal: "D", key: -2}, {nominal: "Neutral", key: 0}} {
		t.Run("PartSounding "+test.nominal, func(t *testing.T) {
			var partFacts []conformanceAlphaTabTranspositionStaff
			fixture := writeConformanceFixture(t, conformanceGPIFArchive(t, conformancePartSoundingGPIF(test.nominal)))
			readAlphaTabOracleFacts(t, "--transposition", fixture, &partFacts)
			if len(partFacts) < 1 || partFacts[0].DisplayPitch != -12 || !slices.Equal(partFacts[0].Keys, []int{test.key}) {
				t.Fatalf("AlphaTab PartSounding %s facts = %#v, want display -12 key %d", test.nominal, partFacts, test.key)
			}
		})
	}
	conformanceIndependentClaim(t, "field:Staff.DisplayTranspositionPitch", claimSite("transposition", "import", "M04-TRANSPOSITION", "distinct sounding and display offsets"), claimSite("transposition", "model", "M04-TRANSPOSITION", "distinct sounding and display offsets"))
}

func conformancePartSoundingGPIF(nominalKey string) string {
	source := strings.Replace(conformanceOwnershipGPIF,
		`<Name>Grand before</Name>`, `<Name>Grand before</Name><PartSounding><NominalKey>`+nominalKey+`</NominalKey><TranspositionPitch>-12</TranspositionPitch></PartSounding>`, 1)
	return strings.Replace(source, `<Pitches>50</Pitches>`, `<Pitches>45 50</Pitches>`, 1)
}

func conformanceTranspositionGPIF() string {
	source := strings.Replace(conformanceOwnershipGPIF, `<Name>Grand before</Name>`, `<Name>Grand before</Name><Transpose><Chromatic>4</Chromatic><Octave>0</Octave></Transpose>`, 1)
	source = strings.Replace(source, `<Pitches>50</Pitches>`, `<Pitches>45 50</Pitches>`, 1)
	source = strings.Replace(source, `<Name>Single middle</Name>`, `<Name>Single middle</Name><Transpose><Chromatic>11</Chromatic><Octave>-2</Octave></Transpose>`, 1)
	source = strings.Replace(source, `<Name>Grand after</Name>`, `<Name>Grand after</Name><Transpose><Chromatic>2</Chromatic><Octave>1</Octave></Transpose>`, 1)
	return strings.Replace(source, `<MasterBar><Time>4/4</Time>`, `<MasterBar><Key><AccidentalCount>0</AccidentalCount><Mode>Major</Mode></Key><Time>4/4</Time>`, 1)
}

func conformanceTranspositionWire(t *testing.T, data []byte) gpifDocument {
	t.Helper()
	return conformanceWireDocument(t, data)
}
