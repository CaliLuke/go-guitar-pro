// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestGP8PercussionUsesEveryResolvedStaff(t *testing.T) {
	tests := []struct {
		name       string
		regular    [][]int16
		grace      map[int]int16
		wantMIDIs  []int
		wantInputs []int
	}{
		{name: "canonical staff without legacy measures", regular: [][]int16{{38}}, grace: map[int]int16{0: 42}, wantMIDIs: []int{38, 42}, wantInputs: []int{38, 42}},
		{name: "two staves", regular: [][]int16{{38}, {36}}, wantMIDIs: []int{36, 38}, wantInputs: []int{36, 38}},
		{name: "three staves and last-only drum", regular: [][]int16{{38}, {42}, {36}}, wantMIDIs: []int{36, 38, 42}, wantInputs: []int{36, 38, 42}},
		{name: "native fallbacks across staves and grace", regular: [][]int16{{69}, {85}}, grace: map[int]int16{1: 87}, wantMIDIs: []int{69, 85, 87}, wantInputs: []int{69, 85, 87}},
		{name: "empty first staff", regular: [][]int16{nil, {36}}, wantMIDIs: []int{36}, wantInputs: []int{36}},
		{name: "later grace-only drum", regular: [][]int16{{38}, {42}}, grace: map[int]int16{1: 46}, wantMIDIs: []int{38, 42, 46}, wantInputs: []int{38, 42, 46}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := percussionExportPercussionSong(t, test.regular, test.grace, percussionExportRepeatedLineCounts(len(test.regular), 5), nil)
			if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
				t.Fatalf("validation diagnostics = %#v", diagnostics)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if err != nil || len(report.Entries) != 0 || len(data) == 0 {
				t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}

			gpif := readCurveGPIF(t, data)
			stavesAt := strings.Index(string(gpif), "<Staves>")
			instrumentAt := strings.Index(string(gpif), "<InstrumentSet>")
			if stavesAt < 0 || instrumentAt < 0 || stavesAt > instrumentAt {
				t.Fatalf("track resource order has Staves at %d and InstrumentSet at %d", stavesAt, instrumentAt)
			}
			wire := percussionExportWireDocument(t, gpif)
			if got := percussionExportWireOutputs(wire); !slices.Equal(got, test.wantMIDIs) {
				t.Fatalf("resolved wire output MIDI = %v, want %v", got, test.wantMIDIs)
			}
			if got := percussionExportWireInputs(wire); !slices.Equal(got, test.wantInputs) {
				t.Fatalf("wire articulation inputs = %v, want %v", got, test.wantInputs)
			}

			if os.Getenv("ALPHATAB_CONFORMANCE") != "1" {
				return
			}
			alpha := readAlphaTabScore(t, writeConformanceFixture(t, data))
			staffs := percussionExportAlphaStaves(t, alpha, 0)
			if len(staffs) != len(test.regular) {
				t.Fatalf("AlphaTab staves = %d, want %d", len(staffs), len(test.regular))
			}
			for staffIndex, staff := range staffs {
				if staff["percussion"] != true || staff["standardNotationLineCount"] != float64(5) {
					t.Fatalf("AlphaTab staff %d context = %#v, want percussion with five lines", staffIndex, staff)
				}
			}
			if got := percussionExportAlphaMainMIDIs(staffs); !slices.Equal(got, percussionExportSortedRegularMIDIs(test.regular)) {
				t.Fatalf("AlphaTab main-note MIDI = %v, want %v", got, percussionExportSortedRegularMIDIs(test.regular))
			}
		})
	}
}

func TestGP8PercussionKeepsCustomIdentityAndTrackOwnership(t *testing.T) {
	custom := []PercussionArticulation{
		{ElementName: "Metal", ElementType: "cymbal", Name: "Edge", InputMIDINumbers: []int{91, 92}, OutputMIDINumber: 56},
		{ElementName: "Metal", ElementType: "cymbal", Name: "Bell", InputMIDINumbers: []int{93}, OutputMIDINumber: 56},
	}
	song := percussionExportPercussionSong(t, [][]int16{{93}, {92}, {36}}, nil, []int{5, 5, 5}, custom)
	song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0].HasPercussionArticulation = true
	song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0].PercussionArticulation = 1
	percussionExportAddFollowingPitchedTrack(t, song)
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 || len(data) == 0 {
		t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
	wire := percussionExportWireDocument(t, readCurveGPIF(t, data))
	flat := percussionExportFlatArticulations(wire.Tracks.Tracks[0].InstrumentSet.Elements.Elements)
	if len(flat) != 3 || flat[0].Name != "Edge" || flat[1].Name != "Bell" || flat[0].OutputMIDINumber != 56 || flat[1].OutputMIDINumber != 56 || flat[2].OutputMIDINumber != 36 {
		t.Fatalf("ordered custom and fallback articulations = %#v", flat)
	}
	identities := []int{*wire.Notes.Notes[0].InstrumentArticulation, *wire.Notes.Notes[1].InstrumentArticulation, *wire.Notes.Notes[2].InstrumentArticulation}
	if !slices.Equal(identities, []int{1, 0, 2}) {
		t.Fatalf("wire articulation identities = %v, want [1 0 2]", identities)
	}
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		alpha := readAlphaTabScore(t, writeConformanceFixture(t, data))
		tracks := alpha.(map[string]any)["tracks"].([]any)
		if len(tracks) != 2 || len(percussionExportAlphaStaves(t, alpha, 0)) != 3 || percussionExportAlphaStaves(t, alpha, 1)[0]["percussion"] != false {
			t.Fatalf("AlphaTab track ownership = %#v", tracks)
		}
		if got := percussionExportAlphaMainMIDIs(percussionExportAlphaStaves(t, alpha, 0)); !slices.Equal(got, []int{36, 56, 56}) {
			t.Fatalf("AlphaTab resolved custom/fallback MIDI = %v, want [36 56 56]", got)
		}
	}
}

func TestGP8PercussionKeepsReorderedCustomTables(t *testing.T) {
	definitions := []PercussionArticulation{
		{ElementName: "Metal", ElementType: "cymbal", Name: "Edge", InputMIDINumbers: []int{91, 92}, OutputMIDINumber: 56},
		{ElementName: "Metal", ElementType: "cymbal", Name: "Bell", InputMIDINumbers: []int{93}, OutputMIDINumber: 56},
	}
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		name := definitions[order[0]].Name + "-then-" + definitions[order[1]].Name
		t.Run(name, func(t *testing.T) {
			custom := []PercussionArticulation{definitions[order[0]], definitions[order[1]]}
			song := percussionExportPercussionSong(t, [][]int16{{int16(custom[0].InputMIDINumbers[0])}, {int16(custom[1].InputMIDINumbers[0])}, {36}}, nil, []int{5, 5, 5}, custom)
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if err != nil || len(report.Entries) != 0 || len(data) == 0 {
				t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
			wire := percussionExportWireDocument(t, readCurveGPIF(t, data))
			flat := percussionExportFlatArticulations(wire.Tracks.Tracks[0].InstrumentSet.Elements.Elements)
			if len(flat) != 3 || flat[0].Name != custom[0].Name || flat[1].Name != custom[1].Name || flat[2].OutputMIDINumber != 36 {
				t.Fatalf("ordered articulations = %#v", flat)
			}
			identities := []int{*wire.Notes.Notes[0].InstrumentArticulation, *wire.Notes.Notes[1].InstrumentArticulation, *wire.Notes.Notes[2].InstrumentArticulation}
			if !slices.Equal(identities, []int{0, 1, 2}) {
				t.Fatalf("wire articulation identities = %v, want [0 1 2]", identities)
			}
		})
	}
}

func TestGP8PercussionUsesCompatibilityAuthorityForResources(t *testing.T) {
	for _, test := range []struct {
		name       string
		configure  func(*Track)
		wantOutput []int
	}{
		{name: "absent legacy slice", wantOutput: []int{36, 38}},
		{name: "shared legacy view", configure: func(track *Track) {
			track.Measures = track.Staves[0].Measures
		}, wantOutput: []int{36, 38}},
		{name: "replaced legacy slice", configure: func(track *Track) {
			track.Measures = percussionExportMeasuresWithMIDIs(track.Staves[0].Measures, []int16{42})
		}, wantOutput: []int{36, 42}},
		{name: "replaced canonical staff after clearing legacy", configure: func(track *Track) {
			track.Measures = nil
			track.Staves[0].Measures = percussionExportMeasuresWithMIDIs(track.Staves[0].Measures, []int16{46})
		}, wantOutput: []int{36, 46}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := percussionExportPercussionSong(t, [][]int16{{38}, {36}}, nil, []int{5, 5}, nil)
			if test.configure != nil {
				test.configure(&song.Tracks[0])
			}
			if err := FinalizeSong(song); err != nil {
				t.Fatal(err)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if err != nil || len(report.Entries) != 0 || len(data) == 0 {
				t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
			if got := percussionExportWireOutputs(percussionExportWireDocument(t, readCurveGPIF(t, data))); !slices.Equal(got, test.wantOutput) {
				t.Fatalf("resolved wire output MIDI = %v, want %v", got, test.wantOutput)
			}
		})
	}
}

func TestGP8PercussionRejectsMissingArticulationResource(t *testing.T) {
	song := percussionExportPercussionSong(t, [][]int16{{38}}, nil, []int{5}, nil)
	builder := gp8Builder{
		song:            song,
		articulationIDs: []map[int16]int{{}},
	}
	note := &song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	if _, err := builder.addNote(0, nil, note); err == nil || !strings.Contains(err.Error(), "no exported articulation resource") {
		t.Fatalf("missing articulation resource error = %v", err)
	}
}

func TestGP8PercussionLineCountPolicy(t *testing.T) {
	for _, lines := range [][]int{{1, 1}, {5, 5}} {
		t.Run(strings.Join([]string{"equal", string(rune('0' + lines[0]))}, "-"), func(t *testing.T) {
			song := percussionExportPercussionSong(t, [][]int16{{38}, {36}}, nil, lines, nil)
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if err != nil || len(report.Entries) != 0 || len(data) == 0 {
				t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
			if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
				for staffIndex, staff := range percussionExportAlphaStaves(t, readAlphaTabScore(t, writeConformanceFixture(t, data)), 0) {
					if staff["percussion"] != true || staff["standardNotationLineCount"] != float64(lines[staffIndex]) {
						t.Fatalf("AlphaTab staff %d = %#v, want percussion with %d lines", staffIndex, staff, lines[staffIndex])
					}
				}
			}
		})
	}

	for _, lines := range [][]int{{5, 1}, {1, 5}} {
		t.Run(strings.Join([]string{"different", string(rune('0' + lines[0])), string(rune('0' + lines[1]))}, "-"), func(t *testing.T) {
			song := percussionExportPercussionSong(t, [][]int16{{38}, {36}}, nil, lines, nil)
			beforeSong, _ := json.Marshal(song)
			options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
			beforeOptions, _ := json.Marshal(options)
			preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if len(preflight.Entries) != 1 || preflight.Entries[0].Code != "gp8.normalize.percussion-line-count" || preflight.Entries[0].Location.Staff != 1 {
				t.Fatalf("preflight = %#v, want one staff-1 line-count normalization", preflight.Entries)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, options)
			var lossErr *ExportLossError
			if len(data) != 0 || !errors.As(err, &lossErr) || !reflect.DeepEqual(report, preflight) {
				t.Fatalf("strict export = %d bytes, %#v, %v; preflight %#v", len(data), report.Entries, err, preflight.Entries)
			}
			unrelated := options
			unrelated.LossPolicy.AllowedCodes = []string{"gp8.omit.staff-line-count"}
			if deniedData, _, deniedErr := ExportWithReport(song, ExportFormatGP8, unrelated); len(deniedData) != 0 || deniedErr == nil {
				t.Fatalf("unrelated allowlist export = %d bytes, %v", len(deniedData), deniedErr)
			}
			allowed := options
			allowed.LossPolicy.AllowedCodes = []string{"gp8.normalize.percussion-line-count"}
			data, _, err = ExportWithReport(song, ExportFormatGP8, allowed)
			if err != nil || len(data) == 0 {
				t.Fatalf("allowed export = %d bytes, %v", len(data), err)
			}
			if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
				for _, staff := range percussionExportAlphaStaves(t, readAlphaTabScore(t, writeConformanceFixture(t, data)), 0) {
					if staff["percussion"] != true || staff["standardNotationLineCount"] != float64(lines[0]) {
						t.Fatalf("normalized AlphaTab staff = %#v, want percussion with %d lines", staff, lines[0])
					}
				}
			}
			afterSong, _ := json.Marshal(song)
			afterOptions, _ := json.Marshal(options)
			if !slices.Equal(beforeSong, afterSong) || !slices.Equal(beforeOptions, afterOptions) {
				t.Fatal("preflight or export mutated its input or options")
			}
		})
	}
}

func percussionExportMeasuresWithMIDIs(source []Measure, midis []int16) []Measure {
	measure := source[0]
	notes := make([]Note, len(midis))
	for index, midi := range midis {
		notes[index] = Note{Value: midi, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}
	}
	measure.Voices = []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: notes}}}}
	return []Measure{measure}
}

func percussionExportPercussionSong(t *testing.T, regular [][]int16, grace map[int]int16, lines []int, custom []PercussionArticulation) *Song {
	t.Helper()
	song := conformanceTechniqueSong(t)
	track := &song.Tracks[0]
	track.PercussionTrack = true
	track.Strings = nil
	track.PercussionArticulations = slices.Clone(custom)
	song.Channels[0] = MidiChannel{Channel: 9, EffectChannel: 9, Instrument: 0, Volume: 100, Balance: 64}
	base := track.Measures[0]
	track.Measures = nil
	track.Staves = make([]Staff, len(regular))
	for staffIndex, midis := range regular {
		measure := base
		measure.Voices = []Voice{{}}
		if len(midis) > 0 {
			notes := make([]Note, len(midis))
			for noteIndex, midi := range midis {
				notes[noteIndex] = Note{Value: midi, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}
			}
			if graceMIDI, ok := grace[staffIndex]; ok {
				exact, err := NewFret(int64(graceMIDI))
				if err != nil {
					t.Fatal(err)
				}
				notes[0].Effect.Graces = []GraceEffect{{Fret: int8(graceMIDI), ExactFret: &exact, Duration: DurationThirtySecond, Velocity: Forte}}
			}
			measure.Voices[0].Beats = []Beat{{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: notes}}
		}
		track.Staves[staffIndex] = Staff{Measures: []Measure{measure}, PercussionTrack: true, StandardNotationLineCount: lines[staffIndex]}
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

func percussionExportAddFollowingPitchedTrack(t *testing.T, song *Song) {
	t.Helper()
	pitchedSong := conformanceTechniqueSong(t)
	pitched := pitchedSong.Tracks[0]
	pitched.Settings.Notation = true
	pitched.ChannelIndex = len(song.Channels)
	song.Channels = append(song.Channels, MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64})
	song.Tracks = append(song.Tracks, pitched)
}

func percussionExportWireDocument(t *testing.T, data []byte) gpifDocument {
	t.Helper()
	var document gpifDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func percussionExportFlatArticulations(elements []gpifElement) []gpifArticulation {
	var result []gpifArticulation
	for _, element := range elements {
		result = append(result, element.Articulations.Articulations...)
	}
	return result
}

func percussionExportWireOutputs(document gpifDocument) []int {
	flat := percussionExportFlatArticulations(document.Tracks.Tracks[0].InstrumentSet.Elements.Elements)
	result := make([]int, 0, len(document.Notes.Notes))
	for _, note := range document.Notes.Notes {
		if note.InstrumentArticulation == nil || *note.InstrumentArticulation < 0 || *note.InstrumentArticulation >= len(flat) {
			continue
		}
		result = append(result, flat[*note.InstrumentArticulation].OutputMIDINumber)
	}
	sort.Ints(result)
	return result
}

func percussionExportWireInputs(document gpifDocument) []int {
	flat := percussionExportFlatArticulations(document.Tracks.Tracks[0].InstrumentSet.Elements.Elements)
	result := make([]int, 0, len(flat))
	for _, articulation := range flat {
		for _, input := range strings.Fields(articulation.InputMIDINumbers) {
			var value int
			if _, err := fmt.Sscan(input, &value); err == nil {
				result = append(result, value)
			}
		}
	}
	sort.Ints(result)
	return result
}

func percussionExportAlphaStaves(t *testing.T, score any, trackIndex int) []map[string]any {
	t.Helper()
	tracks := score.(map[string]any)["tracks"].([]any)
	items := tracks[trackIndex].(map[string]any)["staves"].([]any)
	result := make([]map[string]any, len(items))
	for index := range items {
		result[index] = items[index].(map[string]any)
	}
	return result
}

func percussionExportAlphaMainMIDIs(staves []map[string]any) []int {
	var result []int
	for _, staff := range staves {
		for _, barItem := range staff["bars"].([]any) {
			for _, voiceItem := range barItem.(map[string]any)["voices"].([]any) {
				for _, beatItem := range voiceItem.(map[string]any)["beats"].([]any) {
					for _, noteItem := range beatItem.(map[string]any)["notes"].([]any) {
						result = append(result, int(noteItem.(map[string]any)["midi"].(float64)))
					}
				}
			}
		}
	}
	sort.Ints(result)
	return result
}

func percussionExportSortedRegularMIDIs(staves [][]int16) []int {
	var result []int
	for _, midis := range staves {
		for _, midi := range midis {
			result = append(result, int(midi))
		}
	}
	sort.Ints(result)
	return result
}

func percussionExportRepeatedLineCounts(count, value int) []int {
	result := make([]int, count)
	for index := range result {
		result[index] = value
	}
	return result
}
