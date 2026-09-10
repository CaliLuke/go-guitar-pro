// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestExportGP8HonorsReplacedLegacyMeasureView(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/notes.gp")
	track := &song.Tracks[0]
	track.Measures = slices.Clone(track.Measures)
	replacement := defaultBeat()
	replacement.Notes = []Note{{Value: 17, String: 1, Kind: NoteTypeNormal, Velocity: Forte}}
	track.Measures[0].Voices = []Voice{{Beats: []Beat{replacement}}}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0].Value
	if got != 17 {
		t.Fatalf("replaced legacy measure exported fret %d, want 17", got)
	}
}

const multiStaffFollowedByTrackGPIF = `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>7.0</GPVersion>
  <Score><Title>Staff ownership</Title></Score>
  <MasterTrack><Tracks>0 1</Tracks></MasterTrack>
  <Tracks>
    <Track id="0">
      <Name>Piano</Name>
      <Instrument ref="keyboardGrandStaff"/>
      <Staves>
        <Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property></Properties></Staff>
        <Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property></Properties></Staff>
      </Staves>
      <MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection>
    </Track>
    <Track id="1">
      <Name>Cello</Name>
      <Staves><Staff><Properties><Property name="Tuning"><Pitches>36</Pitches></Property></Properties></Staff></Staves>
      <MidiConnection><Port>0</Port><PrimaryChannel>2</PrimaryChannel><SecondaryChannel>3</SecondaryChannel></MidiConnection>
    </Track>
  </Tracks>
  <MasterBars><MasterBar><Time>4/4</Time><Bars>0 1 2</Bars></MasterBar></MasterBars>
  <Bars>
    <Bar id="0"><Clef>G2</Clef><Voices>-1</Voices></Bar>
    <Bar id="1"><Clef>F4</Clef><Voices>-1</Voices></Bar>
    <Bar id="2"><Clef>C3</Clef><Voices>-1</Voices></Bar>
  </Bars>
  <Voices/><Beats/><Notes/><Rhythms/>
</GPIF>`

func TestParseGPIFPreservesGrandStaff(t *testing.T) {
	song, err := ParseFile("testdata/gp7/grand-staff.gp")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Tracks) != 1 {
		t.Fatalf("tracks = %d, want 1", len(song.Tracks))
	}
	track := &song.Tracks[0]
	if len(track.Staves) != 2 {
		t.Fatalf("staves = %d, want 2", len(track.Staves))
	}
	if got := countStaffNotes(&track.Staves[0]); got != 35 {
		t.Errorf("staff 1 notes = %d, want 35", got)
	}
	if got := countStaffNotes(&track.Staves[1]); got != 18 {
		t.Errorf("staff 2 notes = %d, want 18", got)
	}
	if got := countStaffNotes(&track.Staves[0]) + countStaffNotes(&track.Staves[1]); got != 53 {
		t.Errorf("total notes = %d, want 53", got)
	}
	if track.Staves[0].Measures[0].Clef != MeasureClefTreble {
		t.Errorf("staff 1 clef = %v, want treble", track.Staves[0].Measures[0].Clef)
	}
	if track.Staves[1].Measures[0].Clef != MeasureClefBass {
		t.Errorf("staff 2 clef = %v, want bass", track.Staves[1].Measures[0].Clef)
	}
	if track.Staves[1].Measures[0].StaffIndex != 1 {
		t.Errorf("staff 2 measure staff index = %d, want 1", track.Staves[1].Measures[0].StaffIndex)
	}
	if len(track.Measures) != len(track.Staves[0].Measures) {
		t.Errorf("legacy measures = %d, first-staff measures = %d", len(track.Measures), len(track.Staves[0].Measures))
	}
	if &track.Measures[0] != &track.Staves[0].Measures[0] {
		t.Error("legacy measures do not view the first staff")
	}
}

func TestParseGPIFAssignsBarsAcrossStavesBeforeAdvancingTrack(t *testing.T) {
	song, err := parseGPIF([]byte(multiStaffFollowedByTrackGPIF))
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Tracks) != 2 {
		t.Fatalf("tracks = %d, want 2", len(song.Tracks))
	}
	if len(song.Tracks[0].Staves) != 2 || len(song.Tracks[1].Staves) != 1 {
		t.Fatalf("staff counts = [%d %d], want [2 1]", len(song.Tracks[0].Staves), len(song.Tracks[1].Staves))
	}
	if got := song.Tracks[0].Staves[0].Measures[0].Clef; got != MeasureClefTreble {
		t.Errorf("track 1 staff 1 clef = %v, want treble", got)
	}
	if got := song.Tracks[0].Staves[1].Measures[0].Clef; got != MeasureClefBass {
		t.Errorf("track 1 staff 2 clef = %v, want bass", got)
	}
	if got := song.Tracks[1].Staves[0].Measures[0].Clef; got != MeasureClefAlto {
		t.Errorf("track 2 staff 1 clef = %v, want alto", got)
	}
	if measure := song.Tracks[1].Staves[0].Measures[0]; measure.TrackIndex != 1 || measure.StaffIndex != 0 {
		t.Errorf("track 2 measure ownership = track %d staff %d, want track 1 staff 0", measure.TrackIndex, measure.StaffIndex)
	}
	if got := song.Tracks[0].Strings[0].Value; got != 45 {
		t.Errorf("legacy tuning first string = %d, want first staff value 45", got)
	}
	if got := song.Tracks[0].Staves[1].Strings[0].Value; got != 43 {
		t.Errorf("second-staff first string = %d, want 43", got)
	}
	if &song.Tracks[0].Strings[0] != &song.Tracks[0].Staves[0].Strings[0] {
		t.Error("legacy tuning does not view the first staff")
	}
}

func TestGPIFCapoUsesStaffFallbackAndRejectsNarrowing(t *testing.T) {
	t.Run("matching staff overrides", func(t *testing.T) {
		gpif := strings.Replace(multiStaffFollowedByTrackGPIF, "<Name>Piano</Name>",
			`<Name>Piano</Name><Properties><Property name="CapoFret"><Fret>2</Fret></Property></Properties>`, 1)
		gpif = strings.Replace(gpif,
			`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property></Properties></Staff>`,
			`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property><Property name="CapoFret"><Fret>4</Fret></Property></Properties></Staff>`, 1)
		gpif = strings.Replace(gpif,
			`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property></Properties></Staff>`,
			`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property><Property name="CapoFret"><Fret>4</Fret></Property></Properties></Staff>`, 1)
		result, err := ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{Strict: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Diagnostics) != 0 || result.Song.Tracks[0].CapoFret != 4 {
			t.Fatalf("matching staff overrides = capo fret %d, diagnostics %#v", result.Song.Tracks[0].CapoFret, result.Diagnostics)
		}
	})

	t.Run("staff fallback", func(t *testing.T) {
		gpif := strings.Replace(multiStaffFollowedByTrackGPIF,
			`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property></Properties></Staff>`,
			`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property><Property name="CapoFret"><Fret>2</Fret></Property></Properties></Staff>`, 1)
		song, err := parseGPIF([]byte(gpif))
		if err != nil {
			t.Fatal(err)
		}
		if song.Tracks[0].CapoFret != 2 {
			t.Fatalf("staff capo = %d, want 2", song.Tracks[0].CapoFret)
		}
	})

	t.Run("narrowing boundary", func(t *testing.T) {
		gpif := strings.Replace(multiStaffFollowedByTrackGPIF, "<Name>Piano</Name>",
			`<Name>Piano</Name><Properties><Property name="CapoFret"><Fret>2147483648</Fret></Property></Properties>`, 1)
		if _, err := parseGPIF([]byte(gpif)); err == nil {
			t.Fatal("parse accepted a capo fret above int32")
		}
	})

	t.Run("different staff values", func(t *testing.T) {
		gpif := strings.Replace(multiStaffFollowedByTrackGPIF,
			`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property></Properties></Staff>`,
			`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property><Property name="CapoFret"><Fret>2</Fret></Property></Properties></Staff>`, 1)
		gpif = strings.Replace(gpif,
			`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property></Properties></Staff>`,
			`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property><Property name="CapoFret"><Fret>4</Fret></Property></Properties></Staff>`, 1)
		result, err := ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{Strict: true})
		var strictErr *StrictParseError
		if !errors.As(err, &strictErr) || result == nil {
			t.Fatalf("strict parse = %#v, %v, want StrictParseError", result, err)
		}
		if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Code == "GPIF.Track.CapoFret.StaffConflict"
		}) {
			t.Fatalf("diagnostics = %#v, want staff capo conflict", result.Diagnostics)
		}
	})
}

func TestGPIFCapoResolutionTable(t *testing.T) {
	property := func(value int) gpifStaffProperty {
		return gpifStaffProperty{Name: "CapoFret", Fret: &value}
	}
	staff := func(properties ...gpifStaffProperty) gpifStaff {
		return gpifStaff{Properties: properties}
	}
	tests := []struct {
		name   string
		track  gpifTrack
		value  int32
		common bool
	}{
		{name: "neither level", track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{staff()}}}, common: true},
		{name: "track only", track: gpifTrack{Properties: []gpifStaffProperty{property(2)}, Staves: gpifStaves{Staff: []gpifStaff{staff()}}}, value: 2, common: true},
		{name: "staff only", track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{staff(property(4))}}}, value: 4, common: true},
		{name: "matching track and staff", track: gpifTrack{Properties: []gpifStaffProperty{property(2)}, Staves: gpifStaves{Staff: []gpifStaff{staff(property(2))}}}, value: 2, common: true},
		{name: "single staff override", track: gpifTrack{Properties: []gpifStaffProperty{property(2)}, Staves: gpifStaves{Staff: []gpifStaff{staff(property(4))}}}, value: 4, common: true},
		{name: "both staves override", track: gpifTrack{Properties: []gpifStaffProperty{property(2)}, Staves: gpifStaves{Staff: []gpifStaff{staff(property(4)), staff(property(4))}}}, value: 4, common: true},
		{name: "different staff values", track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{staff(property(2)), staff(property(4))}}}, value: 2},
		{name: "missing beside override", track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{staff(), staff(property(4))}}}},
		{name: "explicit zero override", track: gpifTrack{Properties: []gpifStaffProperty{property(2)}, Staves: gpifStaves{Staff: []gpifStaff{staff(property(0)), staff()}}}},
		{name: "last property wins", track: gpifTrack{Properties: []gpifStaffProperty{property(2), property(4)}, Staves: gpifStaves{Staff: []gpifStaff{staff()}}}, value: 4, common: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := gpifResolveCapo(test.track)
			if err != nil {
				t.Fatal(err)
			}
			if got.value != test.value || got.common != test.common {
				t.Fatalf("resolution = %#v, want value %d common %t", got, test.value, test.common)
			}
		})
	}
}

func TestParseGPIFInvalidBarSkipsWholeTrack(t *testing.T) {
	gpif := strings.Replace(multiStaffFollowedByTrackGPIF, "<Bars>0 1 2</Bars>", "<Bars>-1 2</Bars>", 1)
	song, err := parseGPIF([]byte(gpif))
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Tracks[0].Staves[0].Measures) != 1 || len(song.Tracks[0].Staves[1].Measures) != 1 {
		t.Fatalf(
			"skipped track measures = [%d %d], want [1 1]",
			len(song.Tracks[0].Staves[0].Measures),
			len(song.Tracks[0].Staves[1].Measures),
		)
	}
	if got := song.Tracks[0].Staves[1].Measures[0].Clef; got != MeasureClefTreble {
		t.Errorf("skipped staff placeholder clef = %v, want treble", got)
	}
	if got := song.Tracks[1].Staves[0].Measures[0].Clef; got != MeasureClefAlto {
		t.Errorf("following track clef = %v, want alto", got)
	}
}

func TestParseBinaryTrackExposesOneStaff(t *testing.T) {
	song, err := ParseFile("testdata/gp5/notes.gp5")
	if err != nil {
		t.Fatal(err)
	}
	track := &song.Tracks[0]
	if len(track.Staves) != 1 {
		t.Fatalf("staves = %d, want 1", len(track.Staves))
	}
	if len(track.Measures) == 0 || len(track.Strings) == 0 {
		t.Fatal("binary fixture has no legacy measures or strings")
	}
	if &track.Measures[0] != &track.Staves[0].Measures[0] {
		t.Error("legacy measures do not view the binary staff")
	}
	if &track.Strings[0] != &track.Staves[0].Strings[0] {
		t.Error("legacy tuning does not view the binary staff")
	}
}

func countStaffNotes(staff *Staff) int {
	count := 0
	for measureIndex := range staff.Measures {
		for voiceIndex := range staff.Measures[measureIndex].Voices {
			for beatIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
				count += len(staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes)
			}
		}
	}
	return count
}
