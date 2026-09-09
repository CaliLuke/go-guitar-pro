// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type semanticMatrixRun struct {
	t          *testing.T
	assertions map[string]int
}

func newSemanticMatrixRun(t *testing.T) *semanticMatrixRun {
	t.Helper()
	return &semanticMatrixRun{t: t, assertions: make(map[string]int)}
}

func (run *semanticMatrixRun) Field(path string, got, want any) {
	run.equal("field", path, got, want)
}

func (run *semanticMatrixRun) Wire(path string, got, want any) {
	run.equal("wire", path, got, want)
}

func (run *semanticMatrixRun) Dispatch(path string, got, want any) {
	run.equal("dispatch", path, got, want)
}

func (run *semanticMatrixRun) equal(kind, path string, got, want any) {
	run.t.Helper()
	key := kind + ":" + path
	run.assertions[key]++
	if !reflect.DeepEqual(got, want) {
		run.t.Errorf("%s = %#v, want %#v", key, got, want)
	}
}

func (run *semanticMatrixRun) constructs() []string {
	result := make([]string, 0, len(run.assertions))
	for construct, count := range run.assertions {
		if count == 0 {
			panic(fmt.Sprintf("semantic assertion %s did not execute", construct))
		}
		result = append(result, construct)
	}
	sort.Strings(result)
	return result
}

var semanticMatrixExecutors = map[string]func(*semanticMatrixRun){
	"TestSemanticMatrixM01MetadataImport":                     runSemanticMatrixM01MetadataImport,
	"TestSemanticMatrixM01MetadataCorpus":                     runSemanticMatrixM01MetadataCorpus,
	"TestSemanticMatrixM01MetadataExportPolicy":               runSemanticMatrixM01MetadataExportPolicy,
	"TestSemanticMatrixM01BinaryClipboard":                    runSemanticMatrixM01BinaryClipboard,
	"TestSemanticMatrixM01OracleKeepsAuthorAndWriterDistinct": runSemanticMatrixM01OracleKeepsAuthorAndWriterDistinct,
	"TestSemanticMatrixM02DisplayExportPolicy":                runSemanticMatrixM02DisplayExportPolicy,
	"TestSemanticMatrixM02PageSetupImport":                    runSemanticMatrixM02PageSetupImport,
	"TestSemanticMatrixM03OwnershipImport":                    runSemanticMatrixM03OwnershipImport,
	"TestSemanticMatrixM03CompatibilityAuthority":             runSemanticMatrixM03CompatibilityAuthority,
	"TestSemanticMatrixM03BarCardinalityDiagnostics":          runSemanticMatrixM03BarCardinalityDiagnostics,
	"TestSemanticMatrixM04InstrumentContext":                  runSemanticMatrixM04InstrumentContext,
	"TestSemanticMatrixM05PlaybackRouting":                    runSemanticMatrixM05PlaybackRouting,
	"TestSemanticMatrixM06TempoAuthority":                     runSemanticMatrixM06TempoAuthority,
	"TestSemanticMatrixM07MasterBars":                         runSemanticMatrixM07MasterBars,
	"TestSemanticMatrixM07AuthorityAndBoundaries":             runSemanticMatrixM07AuthorityAndBoundaries,
	"TestSemanticMatrixM08Durations":                          runSemanticMatrixM08Durations,
	"TestSemanticMatrixM08ExactTiming":                        runSemanticMatrixM08ExactTiming,
	"TestSemanticMatrixM09NoteRepresentation":                 runSemanticMatrixM09NoteRepresentation,
	"TestSemanticMatrixM09NoteValidation":                     runSemanticMatrixM09NoteValidation,
}

func semanticValidGP8Song(t *testing.T) *Song {
	t.Helper()
	song := syntheticGP8Song()
	for index := range song.Tracks[0].Measures {
		song.Tracks[0].Measures[index].HeaderIndex = index
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("semantic matrix baseline is invalid: %#v", diagnostics)
	}
	return song
}

func semanticValidPitchedGP8Song(t *testing.T) *Song {
	t.Helper()
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.PercussionTrack = false
	track.FretCount = 24
	track.Strings = []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}, {Number: 3, Value: 55}, {Number: 4, Value: 50}, {Number: 5, Value: 45}, {Number: 6, Value: 40}}
	track.PercussionArticulations = nil
	song.Channels[0] = MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}
	for measureIndex := range track.Measures {
		track.Measures[measureIndex].HeaderIndex = measureIndex
		for voiceIndex := range track.Measures[measureIndex].Voices {
			for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
				beat := &track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex]
				for noteIndex := range beat.Notes {
					beat.Notes[noteIndex].Value = int16(noteIndex + 2)
					beat.Notes[noteIndex].String = int8(noteIndex%6 + 1)
					beat.Notes[noteIndex].HasPercussionArticulation = false
					beat.Notes[noteIndex].PercussionArticulation = 0
					for graceIndex := range beat.Notes[noteIndex].Effect.Graces {
						beat.Notes[noteIndex].Effect.Graces[graceIndex].Fret = 1
						fret := Fret(1)
						beat.Notes[noteIndex].Effect.Graces[graceIndex].ExactFret = &fret
						beat.Notes[noteIndex].Effect.Graces[graceIndex].HasPercussionArticulation = false
					}
				}
			}
		}
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("semantic matrix pitched baseline is invalid: %#v", diagnostics)
	}
	return song
}
