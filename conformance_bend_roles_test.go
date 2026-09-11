// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type curveGraceFact struct {
	Track, Staff, Bar, Voice, Beat, Note       int
	GraceType, String, Fret, Duration, Dynamic int
	Bend                                       []retainedBendControl
	Slide                                      int
	Hammer                                     bool
}
type retainedBendControl struct{ Offset, Value float64 }

func TestConformanceBendControlRoles(t *testing.T) {
	runConformanceBendControlRoles(newConformanceRun(t))
}
func runConformanceBendControlRoles(run *conformanceRun) {
	t := run.t
	for _, values := range [][4]int8{{0, 2, 2, 0}, {4, 4, 4, 4}, {0, 0, 0, 0}} {
		song := conformanceRoleSong(t, values)
		bend := firstConformanceBend(t, song)
		run.Normalized("BendEffect.Points", len(bend.Points), 4)
		run.Preserved("BendPoint.ExactOffset", conformanceRoleOffsets(bend.Points), []float64{0, 50, 50, 35})
		run.Preserved("BendPoint.Position", []uint8{bend.Points[0].Position, bend.Points[1].Position, bend.Points[2].Position, bend.Points[3].Position}, []uint8{0, 6, 6, 4})
		run.Preserved("BendPoint.Value", []int8{bend.Points[0].Value, bend.Points[1].Value, bend.Points[2].Value, bend.Points[3].Value}, values[:])
		data, report := assertConsumerLossPolicy(t, song, []string{})
		run.Report("M12-BEND-CONTROL-ROLES", reportCodes(report), []string{})
		run.Wire("gpifProperty.Float", conformanceWireRoleOffsets(readCurveWire(t, data).bend), []string{"0", "50", "50", "35"})
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		run.Field("BendPoint.ExactOffset", conformanceRoleOffsets(firstConformanceBend(t, parsed).Points), []float64{0, 50, 50, 35})
	}
	for _, path := range []string{"testdata/gp7/canon-audio-track.gp", "testdata/gp8/canon-audio-track.gp"} {
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sourceWire := conformanceNoteBendWire(t, original, "177")
		run.Wire("gpifProperty.Float", conformanceWireRoleOffsets(sourceWire), []string{"0.000000", "50.000000", "50.000000", "35.000000"})
		if sourceWire["BendOriginValue"] != "100.000000" || sourceWire["BendMiddleValue"] != "100.000000" || sourceWire["BendDestinationValue"] != "100.000000" {
			t.Fatalf("note177 wire %#v", sourceWire)
		}
		song := conformanceCanonRoleSong(t, path)
		bend := song.Tracks[0].Measures[47].Voices[0].Beats[0].Notes[0].Effect.Bend
		run.Field("BendPoint.ExactOffset", conformanceRoleOffsets(bend.Points), []float64{0, 50, 50, 35})
		data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		// Resolve the exported occurrence through its voice and beat; note IDs can change.
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[47].Voices[0].Beats[0].Notes[0]
		if got.String != 1 || got.Value != 19 {
			t.Fatalf("note177 owner %#v", got)
		}
		run.Field("BendPoint.ExactOffset", conformanceRoleOffsets(got.Effect.Bend.Points), []float64{0, 50, 50, 35})
		run.Wire("gpifProperty.Float", conformanceWireRoleOffsets(conformanceOwnedBendWire(t, data, 0, 47, 0, 0, 0)), []string{"0", "50", "50", "35"})
	}
}
func conformanceRoleSong(t *testing.T, values [4]int8) *Song {
	t.Helper()
	song := conformanceCurveSong(t)
	points := make([]BendPoint, 4)
	for i, offset := range []float64{0, 50, 50, 35} {
		points[i] = BendPoint{Position: uint8(math.Round(offset * 12 / 100)), ExactOffset: &offset, Value: values[i]}
	}
	conformanceCurveSetCurve(song, "bend", &BendEffect{Points: points})
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("invalid role score %#v", diagnostics)
	}
	return song
}
func conformanceRoleOffsets(points []BendPoint) []float64 {
	result := make([]float64, len(points))
	for i, p := range points {
		result[i] = resolvedBendOffset(p)
	}
	return result
}
func conformanceWireRoleOffsets(props map[string]string) []string {
	return []string{props["BendOriginOffset"], props["BendMiddleOffset1"], props["BendMiddleOffset2"], props["BendDestinationOffset"]}
}
func conformanceCanonRoleSong(t *testing.T, path string) *Song {
	t.Helper()
	song, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err == nil || len(data) != 0 || !strings.Contains(err.Error(), "sound automation 1 has bar 179 position 2") {
		t.Fatalf("full source must retain sound-domain rejection: %v", err)
	}
	events := song.Tracks[2].SoundAutomations
	if len(events) != 3 || events[1] != (SoundAutomation{Bar: 179, Position: 2, Sound: 1}) {
		t.Fatalf("source sound events %#v", events)
	}
	song.Tracks[2].SoundAutomations = append(events[:1:1], events[2:]...)
	return song
}
func conformanceNoteBendWire(t *testing.T, data []byte, id string) map[string]string {
	t.Helper()
	var doc struct {
		Notes []struct {
			ID         string `xml:"id,attr"`
			Properties []struct {
				Name  string `xml:"name,attr"`
				Float string `xml:"Float"`
			} `xml:"Properties>Property"`
		} `xml:"Notes>Note"`
	}
	if err := xml.Unmarshal(readCurveGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	for _, note := range doc.Notes {
		if note.ID == id {
			result := map[string]string{}
			for _, p := range note.Properties {
				result[p.Name] = p.Float
			}
			return result
		}
	}
	t.Fatalf("missing wire note %s", id)
	return nil
}
func conformanceOwnedBendWire(t *testing.T, data []byte, track, bar, voice, beat, note int) map[string]string {
	t.Helper()
	var doc struct {
		MasterBars []struct {
			Bars string `xml:"Bars"`
		} `xml:"MasterBars>MasterBar"`
		Bars []struct {
			ID     string `xml:"id,attr"`
			Voices string `xml:"Voices"`
		} `xml:"Bars>Bar"`
		Voices []struct {
			ID    string `xml:"id,attr"`
			Beats string `xml:"Beats"`
		} `xml:"Voices>Voice"`
		Beats []struct {
			ID    string `xml:"id,attr"`
			Notes string `xml:"Notes"`
		} `xml:"Beats>Beat"`
	}
	if err := xml.Unmarshal(readCurveGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	barID := strings.Fields(doc.MasterBars[bar].Bars)[track]
	for _, b := range doc.Bars {
		if b.ID != barID {
			continue
		}
		voiceID := strings.Fields(b.Voices)[voice]
		for _, v := range doc.Voices {
			if v.ID != voiceID {
				continue
			}
			beatID := strings.Fields(v.Beats)[beat]
			for _, be := range doc.Beats {
				if be.ID == beatID {
					return conformanceNoteBendWire(t, data, strings.Fields(be.Notes)[note])
				}
			}
		}
	}
	t.Fatal("missing owned bend wire")
	return nil
}
func TestAlphaTabBendControlRoles(t *testing.T) {
	for _, values := range [][4]int8{{0, 2, 2, 0}, {4, 4, 4, 4}, {0, 0, 0, 0}} {
		data, _, err := ExportWithReport(conformanceRoleSong(t, values), ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var facts []curveGraceFact
		readAlphaTabOracleFacts(t, "--curve-grace", writeConformanceFixture(t, data), &facts)
		want := []retainedBendControl{{0, float64(values[0])}, {30, float64(values[1])}, {30, float64(values[2])}, {21, float64(values[3])}}
		if values[0] == values[1] {
			want = []retainedBendControl{want[0], want[3]}
		}
		if len(facts) != 1 || !reflect.DeepEqual(facts[0].Bend, want) {
			t.Fatalf("consumer controls %#v want %#v", facts, want)
		}
	}
	for _, path := range []string{"testdata/gp7/canon-audio-track.gp", "testdata/gp8/canon-audio-track.gp"} {
		var source, output []curveGraceFact
		readAlphaTabOracleFacts(t, "--curve-grace", path, &source)
		data, _, err := ExportWithReport(conformanceCanonRoleSong(t, path), ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		readAlphaTabOracleFacts(t, "--curve-grace", writeConformanceFixture(t, data), &output)
		match := func(f curveGraceFact) bool {
			return f.Track == 0 && f.Bar == 47 && f.Voice == 0 && f.Beat == 0 && f.String == 6 && f.Fret == 19
		}
		si, oi := slices.IndexFunc(source, match), slices.IndexFunc(output, match)
		if si < 0 || oi < 0 {
			t.Fatal("missing canon note177 consumer occurrence")
		}
		want := []retainedBendControl{{0, 4}, {21, 4}}
		if !reflect.DeepEqual(source[si].Bend, want) || !reflect.DeepEqual(output[oi].Bend, want) {
			t.Fatalf("canon retained controls source %#v output %#v", source[si], output[oi])
		}
	}
}
