// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const semanticM03OwnershipGPIF = `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score><Title>Ownership matrix</Title></Score>
  <MasterTrack><Tracks>t0 t1 t2</Tracks></MasterTrack>
  <Tracks>
    <Track id="t0"><Name>Grand before</Name><Staves>
      <Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property></Properties></Staff>
      <Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property></Properties></Staff>
    </Staves><MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection></Track>
    <Track id="t1"><Name>Single middle</Name><Staves>
      <Staff><Properties><Property name="Tuning"><Pitches>50</Pitches></Property></Properties></Staff>
    </Staves><MidiConnection><Port>0</Port><PrimaryChannel>2</PrimaryChannel><SecondaryChannel>3</SecondaryChannel></MidiConnection></Track>
    <Track id="t2"><Name>Grand after</Name><Staves>
      <Staff><Properties><Property name="Tuning"><Pitches>60 64</Pitches></Property></Properties></Staff>
      <Staff><Properties><Property name="Tuning"><Pitches>31 35</Pitches></Property></Properties></Staff>
    </Staves><MidiConnection><Port>0</Port><PrimaryChannel>4</PrimaryChannel><SecondaryChannel>5</SecondaryChannel></MidiConnection></Track>
  </Tracks>
  <MasterBars><MasterBar><Time>4/4</Time><Bars>b0 b1 b2 b3 b4</Bars></MasterBar></MasterBars>
  <Bars>
    <Bar id="b0"><Clef>G2</Clef><Voices>v0 -1 v2 v3</Voices></Bar>
    <Bar id="b1"><Clef>F4</Clef><Voices>v4</Voices></Bar>
    <Bar id="b2"><Clef>C3</Clef><Voices>v5</Voices></Bar>
    <Bar id="b3"><Clef>G2</Clef><Voices>shared</Voices></Bar>
    <Bar id="b4"><Clef>F4</Clef><Voices>shared</Voices></Bar>
  </Bars>
  <Voices>
    <Voice id="v0"><Beats>beat0</Beats></Voice><Voice id="v2"><Beats>beat2</Beats></Voice>
    <Voice id="v3"><Beats>beat3</Beats></Voice><Voice id="v4"><Beats>beat4</Beats></Voice>
    <Voice id="v5"><Beats>beat5</Beats></Voice><Voice id="shared"><Beats>beat6</Beats></Voice>
  </Voices>
  <Beats>
    <Beat id="beat0"><Rhythm ref="r0"/><Notes>n0</Notes></Beat>
    <Beat id="beat2"><Rhythm ref="r2"/><Notes>n2</Notes></Beat>
    <Beat id="beat3"><Rhythm ref="r3"/><Notes>n3</Notes></Beat>
    <Beat id="beat4"><Rhythm ref="r4"/><Notes>n4</Notes></Beat>
    <Beat id="beat5"><Rhythm ref="r5"/><Notes>n5</Notes></Beat>
    <Beat id="beat6"><Rhythm ref="r6"/><Notes>n6</Notes></Beat>
  </Beats>
  <Notes>
    <Note id="n0"><Properties><Property name="Fret"><Fret>1</Fret></Property><Property name="String"><String>0</String></Property></Properties></Note>
    <Note id="n2"><Properties><Property name="Fret"><Fret>2</Fret></Property><Property name="String"><String>1</String></Property></Properties></Note>
    <Note id="n3"><Properties><Property name="Fret"><Fret>3</Fret></Property><Property name="String"><String>0</String></Property></Properties></Note>
    <Note id="n4"><Properties><Property name="Fret"><Fret>4</Fret></Property><Property name="String"><String>0</String></Property></Properties></Note>
    <Note id="n5"><Properties><Property name="Fret"><Fret>5</Fret></Property><Property name="String"><String>1</String></Property></Properties></Note>
    <Note id="n6"><Properties><Property name="Fret"><Fret>6</Fret></Property><Property name="String"><String>0</String></Property></Properties></Note>
  </Notes>
  <Rhythms>
    <Rhythm id="r0"><NoteValue>Quarter</NoteValue></Rhythm><Rhythm id="r2"><NoteValue>Eighth</NoteValue></Rhythm>
    <Rhythm id="r3"><NoteValue>16th</NoteValue></Rhythm><Rhythm id="r4"><NoteValue>Half</NoteValue></Rhythm>
    <Rhythm id="r5"><NoteValue>Whole</NoteValue></Rhythm><Rhythm id="r6"><NoteValue>32nd</NoteValue></Rhythm>
  </Rhythms>
</GPIF>`

func TestSemanticMatrixM03OwnershipImport(t *testing.T) {
	runSemanticMatrixM03OwnershipImport(newSemanticMatrixRun(t))
}

func runSemanticMatrixM03OwnershipImport(run *semanticMatrixRun) {
	song, err := parseGPIF([]byte(semanticM03OwnershipGPIF))
	if err != nil {
		run.t.Fatal(err)
	}
	run.Field("Song.Tracks", len(song.Tracks), 3)
	wantNames := []string{"Grand before", "Single middle", "Grand after"}
	wantStaffCounts := []int{2, 1, 2}
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		run.Field("Track.Name", track.Name, wantNames[trackIndex])
		run.Field("Track.Number", track.Number, int32(trackIndex))
		run.Field("Track.Staves", len(track.Staves), wantStaffCounts[trackIndex])
		run.Field("Track.Measures", len(track.Measures), 1)
		run.Field("Track.Strings", track.Strings, track.Staves[0].Strings)
		if &track.Measures[0] != &track.Staves[0].Measures[0] {
			run.t.Errorf("track %d compatibility measures do not alias staff 0", trackIndex)
		}
		if &track.Strings[0] != &track.Staves[0].Strings[0] {
			run.t.Errorf("track %d compatibility strings do not alias staff 0", trackIndex)
		}
		for staffIndex := range track.Staves {
			staff := &track.Staves[staffIndex]
			run.Field("Staff.Measures", len(staff.Measures), 1)
			run.Field("Staff.Strings", len(staff.Strings) > 0, true)
			measure := &staff.Measures[0]
			run.Field("Measure.Number", measure.Number, 1)
			run.Field("Measure.TrackIndex", measure.TrackIndex, trackIndex)
			run.Field("Measure.StaffIndex", measure.StaffIndex, staffIndex)
			run.Field("Measure.HeaderIndex", measure.HeaderIndex, 0)
			run.Field("Measure.Voices", measure.Voices, measure.Voices)
			for voiceIndex := range measure.Voices {
				voice := &measure.Voices[voiceIndex]
				run.Field("Voice.MeasureIndex", voice.MeasureIndex, int16(0))
				run.Field("Voice.Beats", voice.Beats, voice.Beats)
			}
		}
	}
	if got := song.Tracks[0].Staves[0].Measures[0].Voices; len(got) != 4 || len(got[1].Beats) != 0 || len(got[2].Beats) != 1 {
		run.t.Fatalf("interior voice placeholder = %#v, want four voices with empty voice 1", got)
	}
	if got := song.Tracks[2].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0].Value; got != 6 {
		run.t.Errorf("track 2 staff 0 reused note fret = %d, want 6", got)
	}
	if got := song.Tracks[2].Staves[1].Measures[0].Voices[0].Beats[0].Notes[0].Value; got != 6 {
		run.t.Errorf("track 2 staff 1 reused note fret = %d, want 6", got)
	}

	wire := extractM03OwnershipWire(run.t, semanticM03OwnershipGPIF)
	run.Wire("gpifMasterTrack.Tracks", wire.trackRefs, []string{"t0", "t1", "t2"})
	run.Wire("gpifTrack.ID", wire.trackIDs, []string{"t0", "t1", "t2"})
	run.Wire("gpifTrack.Name", wire.trackNames, wantNames)
	run.Wire("gpifTrack.Staves", wire.staffCounts, wantStaffCounts)
	run.Wire("gpifStaves.Staff", wire.staffCounts, wantStaffCounts)
	run.Wire("gpifMasterBar.Bars", wire.barRefs, []string{"b0", "b1", "b2", "b3", "b4"})
	run.Wire("gpifBar.ID", wire.barIDs, []string{"b0", "b1", "b2", "b3", "b4"})
	run.Wire("gpifBar.Voices", wire.barVoices[0], []string{"v0", "-1", "v2", "v3"})
	run.Wire("gpifVoice.ID", wire.voiceIDs, []string{"v0", "v2", "v3", "v4", "v5", "shared"})
	run.Wire("gpifVoice.Beats", wire.voiceBeats[5], []string{"beat6"})
}

func TestSemanticMatrixM03CompatibilityAuthority(t *testing.T) {
	runSemanticMatrixM03CompatibilityAuthority(newSemanticMatrixRun(t))
}

func runSemanticMatrixM03CompatibilityAuthority(run *semanticMatrixRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	first := track.Staves[0]
	second := first
	second.Measures = slices.Clone(first.Measures)
	second.Strings = []GuitarString{{Number: 1, Value: 72}}
	track.Staves = []Staff{first, second}

	track.Measures = nil
	track.Strings = nil
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	run.Field("Track.Measures", track.Measures, track.Staves[0].Measures)
	run.Field("Track.Strings", track.Strings, track.Staves[0].Strings)

	legacyMeasures := slices.Clone(track.Measures)
	legacyMeasures[0].Clef = MeasureClefTenor
	legacyStrings := []GuitarString{{Number: 1, Value: 67}}
	track.Measures = legacyMeasures
	track.Strings = legacyStrings
	track.Staves[0].Measures[0].Clef = MeasureClefBass
	track.Staves[0].Strings = []GuitarString{{Number: 1, Value: 65}}
	laterBefore := track.Staves[1]
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if track.Staves[0].Measures[0].Clef != MeasureClefTenor || track.Staves[0].Strings[0].Value != 67 {
		t.Fatalf("legacy replacement did not win for staff 0: %#v", track.Staves[0])
	}
	if !reflect.DeepEqual(track.Staves[1], laterBefore) {
		t.Fatal("staff 0 reconciliation changed a later staff")
	}
	run.Field("Track.Staves", track.Staves, track.Staves)
	run.Field("Staff.Measures", track.Staves[0].Measures, legacyMeasures)
	run.Field("Staff.Strings", track.Staves[0].Strings, legacyStrings)

	track.Measures[0].Clef = MeasureClefAlto
	if got := track.Staves[0].Measures[0].Clef; got != MeasureClefAlto {
		t.Fatalf("in-place legacy edit clef = %v, want alto", got)
	}
	track.Staves[0].Measures = slices.Clone(track.Staves[0].Measures)
	track.Staves[0].Measures[0].Clef = MeasureClefBass
	track.Staves[0].Strings = []GuitarString{{Number: 1, Value: 64}}
	track.Measures = nil
	track.Strings = nil
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if track.Measures[0].Clef != MeasureClefBass || track.Strings[0].Value != 64 {
		t.Fatalf("cleared compatibility view did not adopt direct staff 0 replacement")
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("reconciled song diagnostics = %#v", diagnostics)
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(roundTrip.Tracks[0].Staves) != 2 || roundTrip.Tracks[0].Staves[0].Measures[0].Clef != MeasureClefBass || roundTrip.Tracks[0].Staves[1].Strings[0].Value != 72 {
		t.Fatalf("round-trip ownership = %#v", roundTrip.Tracks[0].Staves)
	}
	run.Field("Song.Tracks", len(roundTrip.Tracks), 1)
	run.Field("Measure.Voices", len(roundTrip.Tracks[0].Staves[0].Measures[0].Voices) > 0, true)
	run.Field("Voice.Beats", len(roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats) > 0, true)
}

func TestSemanticMatrixM03BarCardinalityDiagnostics(t *testing.T) {
	runSemanticMatrixM03BarCardinalityDiagnostics(newSemanticMatrixRun(t))
}

func runSemanticMatrixM03BarCardinalityDiagnostics(run *semanticMatrixRun) {
	t := run.t
	tests := []struct {
		name string
		bars string
	}{
		{name: "short", bars: "b0 b1 b2 b3"},
		{name: "long", bars: "b0 b1 b2 b3 b4 b0"},
		{name: "placeholder after staff", bars: "b0 -1 b2 b3 b4"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := replaceM03MasterBarRefs(semanticM03OwnershipGPIF, test.bars)
			result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{})
			if err != nil || result == nil {
				t.Fatalf("permissive parse = %#v, %v", result, err)
			}
			if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
				return diagnostic.Code == "GPIF.MasterBar.Bars.Cardinality" && diagnostic.Kind == ParseDiagnosticInvalidData && diagnostic.SourcePath == "/GPIF/MasterBars/MasterBar[0]/Bars"
			}) {
				t.Fatalf("diagnostics = %#v, want located bar cardinality diagnostic", result.Diagnostics)
			}
			strictResult, strictErr := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true})
			var strict *StrictParseError
			if strictResult == nil || !errors.As(strictErr, &strict) {
				t.Fatalf("strict parse = %#v, %v, want StrictParseError", strictResult, strictErr)
			}
		})
	}
	run.Wire("gpifMasterBar.Bars", strings.Fields("b0 b1 b2 b3 b4"), []string{"b0", "b1", "b2", "b3", "b4"})
}

type semanticM03Wire struct {
	trackRefs   []string
	trackIDs    []string
	trackNames  []string
	staffCounts []int
	barRefs     []string
	barIDs      []string
	barVoices   [][]string
	voiceIDs    []string
	voiceBeats  [][]string
}

func extractM03OwnershipWire(t *testing.T, source string) semanticM03Wire {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(source))
	result := semanticM03Wire{}
	var path []string
	var currentTrack = -1
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			path = append(path, value.Name.Local)
			joined := strings.Join(path, "/")
			if joined == "GPIF/Tracks/Track" {
				currentTrack++
				result.staffCounts = append(result.staffCounts, 0)
				for _, attribute := range value.Attr {
					if attribute.Name.Local == "id" {
						result.trackIDs = append(result.trackIDs, attribute.Value)
					}
				}
			}
			if joined == "GPIF/Tracks/Track/Staves/Staff" {
				result.staffCounts[currentTrack]++
			}
			if joined == "GPIF/Bars/Bar" {
				for _, attribute := range value.Attr {
					if attribute.Name.Local == "id" {
						result.barIDs = append(result.barIDs, attribute.Value)
					}
				}
			}
			if joined == "GPIF/Voices/Voice" {
				for _, attribute := range value.Attr {
					if attribute.Name.Local == "id" {
						result.voiceIDs = append(result.voiceIDs, attribute.Value)
					}
				}
			}
		case xml.CharData:
			text := strings.TrimSpace(string(value))
			if text == "" {
				continue
			}
			switch strings.Join(path, "/") {
			case "GPIF/MasterTrack/Tracks":
				result.trackRefs = strings.Fields(text)
			case "GPIF/Tracks/Track/Name":
				result.trackNames = append(result.trackNames, text)
			case "GPIF/MasterBars/MasterBar/Bars":
				result.barRefs = strings.Fields(text)
			case "GPIF/Bars/Bar/Voices":
				result.barVoices = append(result.barVoices, strings.Fields(text))
			case "GPIF/Voices/Voice/Beats":
				result.voiceBeats = append(result.voiceBeats, strings.Fields(text))
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		}
	}
	return result
}

func replaceM03MasterBarRefs(source, refs string) string {
	return strings.Replace(source, "<Bars>b0 b1 b2 b3 b4</Bars>", "<Bars>"+refs+"</Bars>", 1)
}
