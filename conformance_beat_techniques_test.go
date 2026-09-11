// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"strings"
	"testing"
)

const beatTechniquesCase = "M10-BEAT-TECHNIQUES"

func TestConformanceBeatTechniques(t *testing.T) { runConformanceBeatTechniques(newConformanceRun(t)) }
func runConformanceBeatTechniques(run *conformanceRun) {
	for mask := 0; mask < 8; mask++ {
		song := consumerLimitSong(run.t)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
		beat.Effect.Tap, beat.Effect.Slap, beat.Effect.Pop = mask&1 != 0, mask&2 != 0, mask&4 != 0
		codes := []string{}
		if beat.Effect.Tap {
			codes = append(codes, "gp8.normalize.beat-tap-note-flag")
		}
		output, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: codes}})
		if err != nil {
			run.t.Fatal(err, report)
		}
		run.Report(beatTechniquesCase, reportCodes(report), codes)
		parsed, err := Parse(output)
		if err != nil {
			run.t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
		run.Preserved("BeatEffects.Tap", got.Tap, beat.Effect.Tap)
		run.Preserved("BeatEffects.Slap", got.Slap, beat.Effect.Slap)
		run.Preserved("BeatEffects.Pop", got.Pop, beat.Effect.Pop)
		wantLegacy := SlapEffectNone
		if mask&1 != 0 {
			wantLegacy = SlapEffectTapping
		}
		if mask&2 != 0 {
			wantLegacy = SlapEffectSlapping
		}
		if mask&4 != 0 {
			wantLegacy = SlapEffectPopping
		}
		run.Normalized("BeatEffects.SlapEffect", got.SlapEffect, wantLegacy)
		doc := conformanceWireDocument(run.t, output)
		names := []string{}
		for _, p := range doc.Beats.Beats[0].Properties.Properties {
			if p.Name == "Slapped" || p.Name == "Popped" {
				if p.Enable == nil {
					run.t.Fatal("missing Enable")
				}
				names = append(names, p.Name)
			}
		}
		want := []string{}
		if beat.Effect.Slap {
			want = append(want, "Slapped")
		}
		if beat.Effect.Pop {
			want = append(want, "Popped")
		}
		run.Wire("gpifProperty.Name", names, want)
		run.Wire("gpifProperty.Enable", len(names), len(want))
	}
	for _, test := range []struct {
		code  string
		setup func(*Beat)
	}{
		{"gp8.omit.beat-tap", func(b *Beat) { b.Notes = nil; b.Status = BeatStatusRest; b.Effect.Tap = true }},
		{"gp8.normalize.beat-tap-note-authority", func(b *Beat) { b.Notes[0].Effect.Tapped = true }},
		{"gp8.normalize.beat-technique-authority", func(b *Beat) { b.Effect.Slap = true; b.Effect.SlapEffect = SlapEffectPopping }},
	} {
		song := consumerLimitSong(run.t)
		test.setup(&song.Tracks[0].Measures[0].Voices[0].Beats[0])
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		run.Report(beatTechniquesCase, reportCodes(report), []string{test.code})
	}

	source := parseTestFixture(run.t, "testdata/gp3/other-effects.gp3")
	run.Preserved("BeatEffects.Tap", source.Tracks[0].Measures[0].Voices[0].Beats[2].Effect.Tap, true)
	run.Preserved("BeatEffects.Slap", source.Tracks[0].Measures[0].Voices[0].Beats[3].Effect.Slap, true)
	run.Preserved("BeatEffects.Pop", source.Tracks[0].Measures[1].Voices[0].Beats[0].Effect.Pop, true)
}

func TestAlphaTabBeatTechniques(t *testing.T) {
	requireAlphaTabConformance(t)
	baseline, err := Export(consumerLimitSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var expected []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, baseline), &expected)
	for mask := 0; mask < 32; mask++ {
		song := consumerLimitSong(t)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
		beat.Effect.Tap, beat.Effect.Slap, beat.Effect.Pop = mask&1 != 0, mask&2 != 0, mask&4 != 0
		beat.Notes[0].Effect.Tapped, beat.Notes[0].Effect.LeftHandTapped = mask&8 != 0, mask&16 != 0
		output, exportErr := Export(song, ExportFormatGP8)
		if exportErr != nil {
			t.Fatal(exportErr)
		}
		var got []beatTechniqueFact
		readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &got)
		expected[0].Tap, expected[0].Slap, expected[0].Pop = mask&9 != 0, mask&2 != 0, mask&4 != 0
		expected[0].LeftHandTapped[0] = mask&16 != 0
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("mask %d got=%#v want=%#v", mask, got, expected)
		}
	}
	var original []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", "testdata/gp3/other-effects.gp3", &original)
	source := parseTestFixture(t, "testdata/gp3/other-effects.gp3")
	output, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var target []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &target)
	// GPIF export fills the three unused voices with one empty beat per bar.
	expectedOriginal := []beatTechniqueFact{}
	for i, fact := range original {
		expectedOriginal = append(expectedOriginal, fact)
		if i == len(original)-1 || original[i+1].Bar != fact.Bar {
			for voice := 1; voice < 4; voice++ {
				expectedOriginal = append(expectedOriginal, beatTechniqueFact{Track: fact.Track, Staff: fact.Staff, Bar: fact.Bar, Voice: voice, LeftHandTapped: []bool{}})
			}
		}
	}
	if !reflect.DeepEqual(expectedOriginal, target) {
		t.Fatalf("original=%#v target=%#v", expectedOriginal, target)
	}
	// These literal source properties exercise independent import and source order.
	for _, properties := range []string{`<Property name="Slapped"><Enable/></Property><Property name="Popped"><Enable/></Property>`, `<Property name="Popped"><Enable/></Property><Property name="Slapped"><Enable/></Property>`} {
		xml := strings.Replace(string(conformanceBarreGPIF(t, baseline)), `<Beat id="0">`, `<Beat id="0"><Properties>`+properties+`</Properties>`, 1)
		xml = strings.Replace(xml, `<Note id="0">`, `<Note id="0"><Properties><Property name="Tapped"><Enable/></Property><Property name="LeftHandTapped"><Enable/></Property></Properties>`, 1)
		archive := conformanceGPIFArchive(t, xml)
		var sourceFacts []beatTechniqueFact
		readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, archive), &sourceFacts)
		if len(sourceFacts) == 0 || !sourceFacts[0].Tap || !sourceFacts[0].Slap || !sourceFacts[0].Pop || !sourceFacts[0].LeftHandTapped[0] {
			t.Fatal(sourceFacts)
		}
		parsed, e := Parse(archive)
		if e != nil {
			t.Fatal(e)
		}
		effect := parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
		if !effect.Tap || !effect.Slap || !effect.Pop || effect.SlapEffect != SlapEffectPopping {
			t.Fatal(effect)
		}
		output, e = Export(parsed, ExportFormatGP8)
		if e != nil {
			t.Fatal(e)
		}
		var out []beatTechniqueFact
		readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &out)
		if !reflect.DeepEqual(sourceFacts, out) {
			t.Fatalf("combined source=%#v output=%#v", sourceFacts, out)
		}
	}
}

func TestBeatTechniquesUnknownSource(t *testing.T) {
	baseline, err := Export(consumerLimitSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	for _, property := range []string{"Slapped", "Popped", "UnknownBeatTechnique"} {
		source := strings.Replace(string(conformanceBarreGPIF(t, baseline)), `<Beat id="0">`, `<Beat id="0"><Properties><Property name="`+property+`"/></Properties>`, 1)
		archive := conformanceGPIFArchive(t, source)
		parsed, e := ParseWithOptions(archive, ParseOptions{})
		if e != nil {
			t.Fatal(e)
		}
		code := "GPIF.Beat.Property." + property + ".MissingEnable"
		if property == "UnknownBeatTechnique" {
			code = "GPIF.Beat.Property.Unknown"
		}
		found := false
		for _, d := range parsed.Diagnostics {
			if d.Code == code && d.ObjectID == "0" {
				found = true
			}
		}
		if !found {
			t.Fatal(parsed.Diagnostics)
		}
		b := parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[0]
		if b.Effect.Tap || b.Effect.Slap || b.Effect.Pop {
			t.Fatal("fabricated technique")
		}
		if _, e = ParseWithOptions(archive, ParseOptions{Strict: true}); e == nil {
			t.Fatal("strict accepted malformed source")
		}
	}
}
