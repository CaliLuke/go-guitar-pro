// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const wahCase = "M10-WAH"

func TestConformanceWah(t *testing.T) { runConformanceWah(newConformanceRun(t)) }
func runConformanceWah(run *conformanceRun) {
	song := consumerLimitSong(run.t)
	states := []WahPedal{WahPedalNone, WahPedalOpen, WahPedalClosed}
	names := []string{"WahPedal.WahPedalNone", "WahPedal.WahPedalOpen", "WahPedal.WahPedalClosed"}
	for i, state := range states {
		song.Tracks[0].Measures[i].Voices[0].Beats[0].Effect.WahPedal = state
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		run.t.Fatal(err, report)
	}
	run.Report(wahCase, reportCodes(report), []string{})
	parsed, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil {
		run.t.Fatal(err)
	}
	for i, state := range states {
		got := parsed.Song.Tracks[0].Measures[i].Voices[0].Beats[0].Effect.WahPedal
		run.Preserved("BeatEffects.WahPedal", got, state)
		run.Enum(names[i], got, state)
	}
	document := conformanceWireDocument(run.t, data)
	tokens := []string{}
	for _, b := range document.Beats.Beats {
		if b.Wah != "" {
			tokens = append(tokens, b.Wah)
		}
	}
	run.Wire("gpifBeat.Wah", tokens, []string{"Open", "Closed"})
	source := parseTestFixture(run.t, "testdata/gp5/Wah.gp5")
	for i, b := range source.Tracks[0].Measures[0].Voices[0].Beats {
		want := WahPedalOpen
		if i%2 == 1 {
			want = WahPedalClosed
		}
		run.Preserved("BeatEffects.WahPedal", b.Effect.WahPedal, want)
	}
	for _, test := range []struct {
		value int8
		want  WahPedal
		codes []string
	}{
		{-2, WahPedalNone, []string{"gp8.omit.wah-legacy-state", "gp8.omit.wah-display"}},
		{-1, WahPedalNone, []string{}},
		{0, WahPedalOpen, []string{"gp8.omit.wah-display"}},
		{99, WahPedalOpen, []string{"gp8.normalize.wah-legacy-value", "gp8.omit.wah-display"}},
		{100, WahPedalClosed, []string{"gp8.omit.wah-display"}},
		{127, WahPedalClosed, []string{"gp8.normalize.wah-legacy-value", "gp8.omit.wah-display"}},
	} {
		probe := consumerLimitSong(run.t)
		b := &probe.Tracks[0].Measures[0].Voices[0].Beats[0]
		b.Effect.MixTableChange = &MixTableChange{Wah: &WahEffect{Value: test.value}}
		output, r, e := ExportWithReport(probe, ExportFormatGP8, ExportOptions{})
		if e != nil {
			run.t.Fatal(e)
		}
		run.Report(wahCase, reportCodes(r), test.codes)
		out, e := Parse(output)
		if e != nil {
			run.t.Fatal(e)
		}
		run.Preserved("BeatEffects.WahPedal", out.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.WahPedal, test.want)
	}
}

func TestAlphaTabWah(t *testing.T) {
	requireAlphaTabConformance(t)
	baseline, err := Export(consumerLimitSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var expected []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, baseline), &expected)
	song := consumerLimitSong(t)
	for i, state := range []WahPedal{WahPedalNone, WahPedalOpen, WahPedalClosed} {
		song.Tracks[0].Measures[i].Voices[0].Beats[0].Effect.WahPedal = state
	}
	output, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var got []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &got)
	for i := range expected {
		if expected[i].Voice == 0 && expected[i].Beat == 0 {
			expected[i].Wah = expected[i].Bar
		}
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("states=%#v want=%#v", got, expected)
	}
	source, err := os.ReadFile("testdata/gp5/Wah.gp5")
	if err != nil {
		t.Fatal(err)
	}
	// Offset 1927 is the complete original GP5 second bar's wah byte (-2).
	if len(source) <= 1927 || source[1927] != 254 {
		t.Fatal("source wah boundary moved")
	}
	var original []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", "testdata/gp5/Wah.gp5", &original)
	for _, value := range []int8{-128, -2, -1, 0, 1, 99, 100, 101, 127} {
		input := slices.Clone(source)
		input[1927] = byte(value)
		var raw []beatTechniqueFact
		readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, input), &raw)
		want := slices.Clone(original)
		state := 0
		if value >= 100 {
			state = 2
		} else if value >= 0 {
			state = 1
		}
		want[9].Wah = state
		if !reflect.DeepEqual(raw, want) {
			t.Fatalf("binary value %d source=%#v want=%#v", value, raw, want)
		}
		parsed, e := Parse(input)
		if e != nil {
			t.Fatal(e)
		}
		output, e = Export(parsed, ExportFormatGP8)
		if e != nil {
			t.Fatal(e)
		}
		var target []beatTechniqueFact
		readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &target)
		// GP5 has two voices here; GPIF fills voices 2 and 3 with empty beats.
		wantTarget := []beatTechniqueFact{}
		for i, fact := range want {
			wantTarget = append(wantTarget, fact)
			if i == len(want)-1 || want[i+1].Bar != fact.Bar {
				for voice := 2; voice < 4; voice++ {
					wantTarget = append(wantTarget, beatTechniqueFact{Bar: fact.Bar, Voice: voice, LeftHandTapped: []bool{}})
				}
			}
		}
		if !reflect.DeepEqual(target, wantTarget) {
			t.Fatalf("binary value %d target=%#v want=%#v", value, target, wantTarget)
		}
	}
}

func TestWahUnsupportedSource(t *testing.T) {
	baseline, err := Export(consumerLimitSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	source := strings.Replace(string(conformanceBarreGPIF(t, baseline)), `<Beat id="0">`, `<Beat id="0"><Wah>FutureWah</Wah>`, 1)
	archive := conformanceGPIFArchive(t, source)
	parsed, err := ParseWithOptions(archive, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(parsed.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == "GPIF.Beat.Wah" && d.ObjectID == "0" }) {
		t.Fatal(parsed.Diagnostics)
	}
	if parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.WahPedal != WahPedalNone {
		t.Fatal("invented state")
	}
	if _, err = ParseWithOptions(archive, ParseOptions{Strict: true}); err == nil {
		t.Fatal("strict accepted unknown GPIF wah")
	}
}
