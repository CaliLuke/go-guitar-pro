// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestConformanceExactWhammyOffsets(t *testing.T) {
	runConformanceExactWhammyOffsets(newConformanceRun(t))
}
func runConformanceExactWhammyOffsets(run *conformanceRun) {
	t := run.t
	for _, offsets := range [][]float64{{0, 35, 35.5, 100}, {0, 35, 35.5, 25}, {1.25, 35.123456789, 65.5, 99.75}} {
		song := exactWhammySong(t, offsets)
		points := song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar.Points
		run.Preserved("BendPoint.ExactOffset", conformanceRoleOffsets(points), offsets)
		run.Preserved("BendPoint.Position", []uint8{points[1].Position, points[2].Position}, []uint8{4, uint8(math.Round(offsets[2] * 12 / 100))})
		run.Preserved("BendPoint.Value", []int8{points[0].Value, points[1].Value, points[2].Value, points[3].Value}, []int8{0, -2, -2, 0})
		run.Preserved("BeatEffects.TremoloBar", len(points), 4)
		run.Normalized("BendEffect.Points", len(points), 4)
		data, report := assertConsumerLossPolicy(t, song, []string{})
		run.Report("M12-EXACT-WHAMMY-OFFSETS", reportCodes(report), []string{})
		wire := readCurveWire(t, data).whammy
		for i, name := range []string{"OriginOffset", "MiddleOffset1", "MiddleOffset2", "DestinationOffset"} {
			attr := strings.ToLower(name[:1]) + name[1:]
			actual, err := strconv.ParseFloat(wire[attr], 64)
			if err != nil {
				t.Fatal(err)
			}
			run.Wire("gpifWhammy."+name, actual, offsets[i])
		}
		parsed, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range parsed.Diagnostics {
			if d.Code == "GPIF.Beat.Whammy.Quantized" {
				t.Fatalf("exact offset quantized: %#v", d)
			}
		}
		run.Field("BendPoint.ExactOffset", conformanceRoleOffsets(parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar.Points), offsets)
	}
	// GP6 named-property path, with fields deliberately out of role order.
	base, err := Export(conformanceCurveSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	source := rewriteConformanceGPIF(t, base, func(gpif string) string {
		return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", `<Properties><Property name="WhammyBar"><Enable/></Property><Property name="WhammyBarDestinationOffset"><Float>25</Float></Property><Property name="WhammyBarMiddleOffset2"><Float>35.5</Float></Property><Property name="WhammyBarMiddleValue"><Float>-50</Float></Property><Property name="WhammyBarMiddleOffset1"><Float>35</Float></Property><Property name="WhammyBarOriginOffset"><Float>1.25</Float></Property></Properties>`)
	})
	result, err := ParseWithOptions(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got := result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar
	run.Field("BendPoint.ExactOffset", conformanceRoleOffsets(got.Points), []float64{1.25, 35, 35.5, 25})
	for _, d := range result.Diagnostics {
		if d.Code == "GPIF.Beat.Whammy.Quantized" {
			t.Fatalf("named exact offset quantized %#v", d)
		}
	}
	binaryCurve, err := (&Song{}).readBendEffect(newCursor(binaryBendRecord(t, 0, 21, 60)))
	if err != nil {
		t.Fatal(err)
	}
	run.Field("BendPoint.ExactOffset", conformanceRoleOffsets(binaryCurve.Points), []float64{0, 35, 100})
	for _, invalid := range []int32{-1, 61, 2147483647} {
		if _, err := (&Song{}).readBendEffect(newCursor(binaryBendRecord(t, 0, invalid))); err == nil {
			t.Fatalf("invalid raw offset %d accepted", invalid)
		}
	}
	// Legacy post-parse edits take precedence and require the exact allowance.
	conflict := exactWhammySong(t, []float64{0, 35, 65.5, 100})
	curve := conflict.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar
	curve.Points[1] = importedBendPoint(35, -2, false)
	curve.Points[1].Position = 6
	data, report := assertConsumerLossPolicy(t, conflict, []string{"gp8.normalize.whammy-offset-authority"})
	run.Report("M12-WHAMMY-OFFSET-AUTHORITY", reportCodes(report), []string{"gp8.normalize.whammy-offset-authority"})
	run.Wire("gpifWhammy.MiddleOffset1", readCurveWire(t, data).whammy["middleOffset1"], "50.000000")
}

func exactWhammySong(t *testing.T, offsets []float64) *Song {
	t.Helper()
	s := conformanceCurveSong(t)
	points := make([]BendPoint, len(offsets))
	for i, offset := range offsets {
		points[i] = bendPointWithOffset(offset, 0)
		if i == 1 || i == 2 {
			points[i].Value = -2
		}
	}
	conformanceCurveSetCurve(s, "whammy", &BendEffect{Points: points})
	return s
}

type retainedWhammyFact struct {
	Track, Staff, Bar, Voice, Beat int
	Points                         []retainedBendControl
}

func TestAlphaTabExactWhammyOffsets(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, offsets := range [][]float64{{0, 35, 35.5, 100}, {0, 35, 35.5, 25}, {1.25, 35, 65.5, 99.75}} {
		song := exactWhammySong(t, offsets)
		data, _ := assertConsumerLossPolicy(t, song, []string{})
		var facts []retainedWhammyFact
		readAlphaTabOracleFacts(t, "--whammy-controls", writeConformanceFixture(t, data), &facts)
		want := []retainedWhammyFact{{Points: []retainedBendControl{{offsets[0] * 0.6, 0}, {offsets[1] * 0.6, -2}, {offsets[2] * 0.6, -2}, {offsets[3] * 0.6, 0}}}}
		if !reflect.DeepEqual(facts, want) {
			t.Fatalf("raw consumer %#v want %#v", facts, want)
		}
	}
	var facts []retainedWhammyFact
	readAlphaTabOracleFacts(t, "--whammy-controls", "testdata/gp6/tremolo.gpx", &facts)
	if len(facts) != 4 || !slices.Equal(facts[3].Points, []retainedBendControl{{0, -4}, {15, -12}, {30.599999999999998, -12}, {45, 0}}) {
		t.Fatalf("GP6 retained 51%% source %#v", facts)
	}
}

func TestWhammyExactOffsetOccurrences(t *testing.T) {
	definition := &gpifBeat{Whammy: &gpifWhammy{OriginValue: "0", MiddleValue: "-50", DestinationValue: "0", OriginOffset: "0", MiddleOffset1: "35", MiddleOffset2: "35.5", DestinationOffset: "100"}}
	first, second := Beat{}, Beat{}
	gpifApplyBeatEffects(definition, &first)
	gpifApplyBeatEffects(definition, &second)
	if first.Effect.TremoloBar.Points[1].ExactOffset == second.Effect.TremoloBar.Points[1].ExactOffset {
		t.Fatal("reused beat shares mutable exact offset")
	}
	*first.Effect.TremoloBar.Points[1].ExactOffset = 36
	if *second.Effect.TremoloBar.Points[1].ExactOffset != 35 || *first.Effect.TremoloBar.Points[2].ExactOffset != 35.5 {
		t.Fatal("editing one offset changed another occurrence or role")
	}
}

func TestAlphaTabWhammyOffsetAuthority(t *testing.T) {
	requireAlphaTabConformance(t)
	s := exactWhammySong(t, []float64{0, 35, 65.5, 100})
	curve := s.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar
	curve.Points[1] = importedBendPoint(35, -2, false)
	changed := 36.25
	curve.Points[1].ExactOffset = &changed
	curve.Points[1].Position = 6
	data, _ := assertConsumerLossPolicy(t, s, []string{"gp8.normalize.whammy-offset-authority"})
	var facts []retainedWhammyFact
	readAlphaTabOracleFacts(t, "--whammy-controls", writeConformanceFixture(t, data), &facts)
	want := []retainedWhammyFact{{Points: []retainedBendControl{{0, 0}, {30, -2}, {39.3, -2}, {60, 0}}}}
	if !reflect.DeepEqual(facts, want) {
		t.Fatalf("legacy authority raw controls %#v want %#v", facts, want)
	}
}

func TestWhammyNonmonotonicGestureLoss(t *testing.T) {
	s := exactWhammySong(t, []float64{0, 35, 35.5, 25})
	s.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar.Points[3].Value = -4
	data, _ := assertConsumerLossPolicy(t, s, []string{"gp8.normalize.whammy-curve"})
	wire := readCurveWire(t, data).whammy
	if wire["middleOffset1"] != "35.000000" || wire["middleOffset2"] != "35.500000" || wire["destinationOffset"] != "25.000000" {
		t.Fatalf("wire reordered roles %#v", wire)
	}
	requireAlphaTabConformance(t)
	var facts []retainedWhammyFact
	readAlphaTabOracleFacts(t, "--whammy-controls", writeConformanceFixture(t, data), &facts)
	want := []retainedWhammyFact{{Points: []retainedBendControl{{0, 0}, {15, -4}}}}
	if !reflect.DeepEqual(facts, want) {
		t.Fatalf("consumer gesture %#v want %#v", facts, want)
	}
}
