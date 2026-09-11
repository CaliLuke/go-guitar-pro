// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestConformanceMIDIProgramReferences(t *testing.T) {
	runConformanceMIDIProgramReferences(newConformanceRun(t))
}

func runConformanceMIDIProgramReferences(run *conformanceRun) {
	t := run.t
	for _, fixture := range []struct {
		path           string
		invalidIndices []int
	}{
		{path: "testdata/gp4/fade-to-black.gp4", invalidIndices: []int{25, 41, 57}},
		{path: "references/alphaTab/packages/alphatab/test-data/guitarpro3/beat-harmonics.gp3", invalidIndices: []int{9, 25, 41, 57}},
	} {
		t.Run(fixture.path, func(t *testing.T) {
			data, err := os.ReadFile(fixture.path)
			if errors.Is(err, os.ErrNotExist) && strings.HasPrefix(fixture.path, "references/") {
				t.Skip("pinned AlphaTab source checkout is not present")
			}
			if err != nil {
				t.Fatal(err)
			}
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			selected := make(map[int]bool, len(result.Song.Tracks))
			for _, track := range result.Song.Tracks {
				selected[track.ChannelIndex] = true
			}
			for _, channelIndex := range fixture.invalidIndices {
				if selected[channelIndex] {
					t.Fatalf("channel %d with sentinel program is selected by a track", channelIndex)
				}
				if got := result.Song.Channels[channelIndex].Instrument; got != -1 {
					t.Fatalf("channel %d program = %d, want -1", channelIndex, got)
				}
			}
			if parseDiagnosticWithCode(result.Diagnostics, "score.channel.instrument") != nil {
				t.Fatalf("unused programs produced diagnostics: %#v", result.Diagnostics)
			}
			report := PreflightExport(result.Song, ExportFormatGP8, ExportOptions{})
			if hasExportCode(report, "gp8.reject.score.channel.instrument") {
				t.Fatalf("unused programs produced export rejection: %#v", report.Entries)
			}
			if strings.Contains(strings.ToLower(firstRejectedReason(report)), "midi program") {
				t.Fatalf("unrelated export rejection was masked by MIDI program: %#v", report.Entries)
			}
			selectedSentinel := *result.Song
			selectedSentinel.Tracks = slices.Clone(result.Song.Tracks)
			selectedSentinel.Tracks[0].ChannelIndex = fixture.invalidIndices[0]
			diagnostic := scoreDiagnosticWithCode(ValidateSong(&selectedSentinel), "score.channel.instrument")
			if diagnostic == nil || diagnostic.Location.Track != 0 || !strings.Contains(diagnostic.Reason, "selected channel "+strconv.Itoa(fixture.invalidIndices[0])) {
				t.Fatalf("selected fixture sentinel diagnostic = %#v", diagnostic)
			}
			if !hasExportCode(PreflightExport(&selectedSentinel, ExportFormatGP8, ExportOptions{}), "gp8.reject.score.channel.instrument") {
				t.Fatalf("selected fixture sentinel did not produce export rejection")
			}
			output, exported, exportErr := ExportWithReport(result.Song, ExportFormatGP8, ExportOptions{})
			if !reflect.DeepEqual(exported, report) {
				t.Fatalf("export report differs from preflight: got %#v, want %#v", exported.Entries, report.Entries)
			}
			if fixture.path == "testdata/gp4/fade-to-black.gp4" {
				if len(output) != 0 || exportErr == nil || !hasExportCode(report, "gp8.reject.score.note.kind") || strings.Contains(strings.ToLower(exportErr.Error()), "midi program") {
					t.Fatalf("fade-to-black export = %d bytes, %#v, %v", len(output), report.Entries, exportErr)
				}
			} else if len(output) == 0 || exportErr != nil {
				t.Fatalf("beat-harmonics export = %d bytes, %#v, %v", len(output), report.Entries, exportErr)
			}
		})
	}

	valid := semanticValidPitchedGP8Song(t)
	valid.Tracks[0].Settings.Notation = true
	valid.Channels = append(valid.Channels, MidiChannel{Channel: 4, EffectChannel: 5, Instrument: -1, Volume: 100, Balance: 64})
	before := slices.Clone(valid.Channels)
	if diagnostics := ValidateSong(valid); scoreDiagnosticWithCode(diagnostics, "score.channel.instrument") != nil {
		t.Fatalf("unused invalid program diagnostics = %#v", diagnostics)
	}
	report := PreflightExport(valid, ExportFormatGP8, ExportOptions{})
	if hasExportCode(report, "gp8.reject.score.channel.instrument") {
		t.Fatalf("unused invalid program report = %#v", report.Entries)
	}
	data, exportReport, err := ExportWithReport(valid, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(data) == 0 || len(exportReport.Entries) != 0 {
		t.Fatalf("unused invalid program export = %d bytes, %#v, %v", len(data), exportReport.Entries, err)
	}
	if !reflect.DeepEqual(valid.Channels, before) {
		t.Fatalf("export mutated channel table: got %#v, want %#v", valid.Channels, before)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := roundTrip.Channels[roundTrip.Tracks[0].ChannelIndex].Instrument; got != 25 {
		t.Fatalf("selected program after GP8 reimport = %d, want 25", got)
	}
	for _, tableWide := range []struct {
		name string
		code string
		set  func(*MidiChannel)
	}{
		{name: "bank", code: "score.channel.bank", set: func(channel *MidiChannel) { channel.Bank = -1 }},
		{name: "mixer", code: "score.channel.volume", set: func(channel *MidiChannel) { channel.Volume = -1 }},
	} {
		t.Run("unused invalid "+tableWide.name, func(t *testing.T) {
			probe := semanticValidPitchedGP8Song(t)
			probe.Channels = append(probe.Channels, MidiChannel{Channel: 4, EffectChannel: 5, Instrument: -1, Volume: 100, Balance: 64})
			tableWide.set(&probe.Channels[1])
			if diagnostic := scoreDiagnosticWithCode(ValidateSong(probe), tableWide.code); diagnostic == nil {
				t.Fatalf("unused invalid %s was not validated table-wide", tableWide.name)
			}
		})
	}

	for _, invalid := range []int32{-7, 128} {
		t.Run("selected invalid program "+strconv.Itoa(int(invalid)), func(t *testing.T) {
			probe := semanticValidPitchedGP8Song(t)
			probe.Channels[0].Instrument = invalid
			before := slices.Clone(probe.Channels)
			diagnostic := scoreDiagnosticWithCode(ValidateSong(probe), "score.channel.instrument")
			if diagnostic == nil || diagnostic.Location.Track != 0 || !strings.Contains(diagnostic.Reason, "selected channel 0") || !strings.Contains(diagnostic.Reason, "outside 0..127") {
				t.Fatalf("selected program %d diagnostic = %#v", invalid, diagnostic)
			}
			preflight := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
			entry := exportEntryWithCode(preflight, "gp8.reject.score.channel.instrument")
			if entry == nil || entry.Location.Track != 0 || entry.Reason != diagnostic.Reason {
				t.Fatalf("selected program %d preflight = %#v, diagnostic %#v", invalid, preflight.Entries, diagnostic)
			}
			output, exported, exportErr := ExportWithReport(probe, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if len(output) != 0 || exportErr == nil || !reflect.DeepEqual(exported, preflight) {
				t.Fatalf("selected program %d export = %d bytes, %#v, %v", invalid, len(output), exported.Entries, exportErr)
			}
			if !reflect.DeepEqual(probe.Channels, before) {
				t.Fatalf("selected program %d was mutated: got %#v, want %#v", invalid, probe.Channels, before)
			}
		})
	}

	outOfRange := semanticValidPitchedGP8Song(t)
	outOfRange.Tracks[0].ChannelIndex = len(outOfRange.Channels)
	diagnostics := ValidateSong(outOfRange)
	if scoreDiagnosticWithCode(diagnostics, "score.track.channel-reference") == nil || scoreDiagnosticWithCode(diagnostics, "score.channel.instrument") != nil {
		t.Fatalf("out-of-range channel diagnostics = %#v", diagnostics)
	}
	preflight := PreflightExport(outOfRange, ExportFormatGP8, ExportOptions{})
	if !hasExportCode(preflight, "gp8.reject.score.track.channel-reference") || hasExportCode(preflight, "gp8.reject.score.channel.instrument") {
		t.Fatalf("out-of-range channel preflight = %#v", preflight.Entries)
	}
	assertBinarySelectedMIDIProgramIsNotCoerced(t)
}

func TestBinarySelectedMIDIProgramIsNotCoerced(t *testing.T) {
	assertBinarySelectedMIDIProgramIsNotCoerced(t)
}

func assertBinarySelectedMIDIProgramIsNotCoerced(t *testing.T) {
	t.Helper()
	var data bytes.Buffer
	for _, value := range []int32{1, 2} {
		if err := binary.Write(&data, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	song := Song{Channels: []MidiChannel{{Channel: 0, EffectChannel: 1, Instrument: -7}}}
	track := Track{}
	if err := song.readChannel(newCursor(data.Bytes()), &track); err != nil {
		t.Fatal(err)
	}
	if track.ChannelIndex != 0 || song.Channels[0].Instrument != -7 {
		t.Fatalf("binary channel selection = index %d program %d, want index 0 program -7", track.ChannelIndex, song.Channels[0].Instrument)
	}
}

func TestAlphaTabPreservesSelectedMIDIProgram(t *testing.T) {
	requireAlphaTabConformance(t)
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.Channels[0].Instrument = 73
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("program 73 export = %#v, %v", report.Entries, err)
	}
	var facts []conformanceAlphaTabMIDIBankTrack
	readAlphaTabOracleFacts(t, "--midi-bank", writeConformanceFixture(t, data), &facts)
	if len(facts) != 1 || facts[0].Program != 73 {
		t.Fatalf("AlphaTab selected MIDI facts = %#v, want program 73", facts)
	}
}

func scoreDiagnosticWithCode(diagnostics []ScoreDiagnostic, code string) *ScoreDiagnostic {
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	return nil
}

func parseDiagnosticWithCode(diagnostics []ParseDiagnostic, code string) *ParseDiagnostic {
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	return nil
}

func exportEntryWithCode(report ExportReport, code string) *ExportReportEntry {
	for index := range report.Entries {
		if report.Entries[index].Code == code {
			return &report.Entries[index]
		}
	}
	return nil
}

func firstRejectedReason(report ExportReport) string {
	for _, entry := range report.Entries {
		if entry.Disposition == ExportDispositionRejected {
			return entry.Reason
		}
	}
	return ""
}
