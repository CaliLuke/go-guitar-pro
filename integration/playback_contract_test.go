// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func playbackContractSong(t *testing.T) *guitarpro.Song {
	t.Helper()
	meter := guitarpro.TimeSignature{Beams: [4]uint8{2, 2, 2, 2}, Numerator: 4, Denominator: guitarpro.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}}
	song := &guitarpro.Song{Tempo: 120,
		Channels:       []guitarpro.MidiChannel{{Channel: 1, EffectChannel: 2, Instrument: 27, Volume: 100, Balance: 64}},
		MeasureHeaders: []guitarpro.MeasureHeader{{Number: 1, TimeSignature: meter}},
		Tracks: []guitarpro.Track{{Number: 1, Name: "Guitar", FretCount: 24, Settings: guitarpro.TrackSettings{Notation: true},
			Strings: []guitarpro.GuitarString{{Number: 1, Value: 64}},
			Measures: []guitarpro.Measure{{Number: 1, TimeSignature: meter, Voices: []guitarpro.Voice{{Beats: []guitarpro.Beat{{
				Duration: guitarpro.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}, Status: guitarpro.BeatStatusNormal, Dynamics: guitarpro.Forte,
				Notes: []guitarpro.Note{{String: 1, Value: 3, Velocity: guitarpro.Forte, Kind: guitarpro.NoteTypeNormal, DurationPercent: 1}},
			}}}}}},
		}},
	}
	if err := guitarpro.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := guitarpro.ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("baseline diagnostics: %#v", diagnostics)
	}
	return song
}

func TestMIDIRoutingTargetLimits(t *testing.T) {
	for _, test := range []struct {
		name       string
		effect     uint8
		mute, solo bool
		codes      []string
	}{
		{name: "port", effect: 18, codes: []string{"gp8.normalize.effect-channel-port"}},
		{name: "state", effect: 2, mute: true, solo: true, codes: []string{"gp8.normalize.playback-state"}},
		{name: "both", effect: 18, mute: true, solo: true, codes: []string{"gp8.normalize.playback-state", "gp8.normalize.effect-channel-port"}},
		{name: "default control", effect: 2}, {name: "mute control", effect: 2, mute: true}, {name: "solo control", effect: 2, solo: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := playbackContractSong(t)
			song.Channels[0].EffectChannel = test.effect
			song.Tracks[0].Mute, song.Tracks[0].Solo = test.mute, test.solo
			before, _ := json.Marshal(song)
			preflight := guitarpro.PreflightExport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
			if len(preflight.Entries) != len(test.codes) {
				t.Fatalf("preflight = %#v, want %v", preflight, test.codes)
			}
			for i, entry := range preflight.Entries {
				if entry.Code != test.codes[i] || entry.Location != (guitarpro.ScoreLocation{Track: 0}) || entry.Disposition != guitarpro.ExportDispositionNormalized {
					t.Fatalf("unscoped report: %#v", entry)
				}
			}
			// Enumerate every allowlist subset, including each isolated conflict.
			for mask := 0; mask < 1<<len(test.codes); mask++ {
				var allowed, blocked []string
				for i, code := range test.codes {
					if mask&(1<<i) != 0 {
						allowed = append(allowed, code)
					} else {
						blocked = append(blocked, code)
					}
				}
				data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{LossPolicy: guitarpro.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				if !reflect.DeepEqual(report, preflight) {
					t.Fatalf("preflight/export differ: %#v %#v", preflight, report)
				}
				if len(blocked) > 0 {
					var loss *guitarpro.ExportLossError
					if len(data) != 0 || !errors.As(err, &loss) {
						t.Fatalf("blocked %v export = %d bytes, %v", blocked, len(data), err)
					}
					var got []string
					for _, entry := range loss.Entries {
						got = append(got, entry.Code)
					}
					if !slices.Equal(got, blocked) {
						t.Fatalf("blocked codes = %v, want %v", got, blocked)
					}
					continue
				}
				if err != nil || len(data) == 0 {
					t.Fatalf("allowed export = %d bytes, %v", len(data), err)
				}
				parsed, err := guitarpro.Parse(data)
				if err != nil {
					t.Fatal(err)
				}
				if parsed.Channels[0].Channel != 1 || parsed.Channels[0].EffectChannel != 2 || parsed.Tracks[0].Mute != test.mute || parsed.Tracks[0].Solo != (test.solo && !test.mute) {
					t.Fatalf("target routing = %#v, mute=%v solo=%v", parsed.Channels[0], parsed.Tracks[0].Mute, parsed.Tracks[0].Solo)
				}
			}
			after, _ := json.Marshal(song)
			if string(before) != string(after) {
				t.Fatal("export mutated routing source")
			}
		})
	}
}

func TestSoundSelectionIdentityAndEdits(t *testing.T) {
	song := playbackContractSong(t)
	song.Tracks[0].Sounds = []guitarpro.TrackSound{
		{Name: "Clean", Label: "Clean label", Path: "factory/clean", Role: "main", Program: 27},
		{Name: "Lead", Label: "Lead label", Path: "user/lead", Role: "solo", Program: 27},
	}
	song.Tracks[0].SoundAutomations = []guitarpro.SoundAutomation{{Sound: 1, Text: "opening"}, {Position: 0.75, Sound: 0, Text: "later"}, {Position: 0.25, Sound: 1, Text: "first at quarter"}, {Position: 0.25, Sound: 0, Text: "second at quarter", Linear: true}}
	assertSoundSelectionRoundTrip(t, song)
	// Public edits change identity; indices remain caller-owned references.
	song.Tracks[0].Sounds[1].Name = "Edited lead"
	song.Tracks[0].SoundAutomations[2].Sound = 0
	assertSoundSelectionRoundTrip(t, song)
	before := selectedSoundIdentities(song)
	slices.Reverse(song.Tracks[0].Sounds)
	for i := range song.Tracks[0].SoundAutomations {
		song.Tracks[0].SoundAutomations[i].Sound = 1 - song.Tracks[0].SoundAutomations[i].Sound
	}
	assertSoundSelectionRoundTrip(t, song)
	if !slices.Equal(selectedSoundIdentities(song), before) {
		t.Fatal("definition reorder changed selected identities")
	}
	for _, broken := range []int{-1, 2} {
		song.Tracks[0].SoundAutomations[0].Sound = broken
		if !slices.ContainsFunc(guitarpro.ValidateSong(song), func(d guitarpro.ScoreDiagnostic) bool { return d.Code == "score.sound-automation.reference" }) {
			t.Fatal("broken reference passed validation")
		}
		data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
		if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(e guitarpro.ExportReportEntry) bool {
			return e.Code == "gp8.reject.score.sound-automation.reference"
		}) {
			t.Fatalf("broken reference export = %d bytes, %#v, %v", len(data), report, err)
		}
	}
}

func assertSoundSelectionRoundTrip(t *testing.T, song *guitarpro.Song) {
	t.Helper()
	before, _ := json.Marshal(song)
	const positionLoss = "gp8.normalize.sound-automation-consumer-position"
	data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{LossPolicy: guitarpro.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{positionLoss}}})
	if err != nil || len(report.Entries) != 3 {
		t.Fatalf("sound export = %#v, %v", report, err)
	}
	for _, entry := range report.Entries {
		if entry.Code != positionLoss {
			t.Fatalf("unexpected sound loss: %#v", entry)
		}
	}
	parsed, err := guitarpro.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(parsed.Tracks[0].Sounds, song.Tracks[0].Sounds) || !slices.Equal(parsed.Tracks[0].SoundAutomations, song.Tracks[0].SoundAutomations) {
		t.Fatalf("sound identities or ordered events changed: %#v %#v", parsed.Tracks[0].Sounds, parsed.Tracks[0].SoundAutomations)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("sound export mutated source")
	}
}

func selectedSoundIdentities(song *guitarpro.Song) []guitarpro.TrackSound {
	var result []guitarpro.TrackSound
	for _, event := range song.Tracks[0].SoundAutomations {
		result = append(result, song.Tracks[0].Sounds[event.Sound])
	}
	return result
}

func TestMIDIRoutingReportsOwningTrack(t *testing.T) {
	song := playbackContractSong(t)
	second := playbackContractSong(t)
	song.Tracks = append(song.Tracks, second.Tracks[0])
	song.Channels = append(song.Channels, second.Channels[0])
	song.Tracks[1].Number, song.Tracks[1].ChannelIndex = 2, 1
	song.Tracks[1].Mute, song.Tracks[1].Solo = true, true
	song.Channels[1].EffectChannel = 18
	if err := guitarpro.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	report := guitarpro.PreflightExport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
	if len(report.Entries) != 2 {
		t.Fatalf("track-scoped reports = %#v", report)
	}
	for _, entry := range report.Entries {
		if entry.Location != (guitarpro.ScoreLocation{Track: 1}) || entry.Disposition != guitarpro.ExportDispositionNormalized {
			t.Fatalf("report targets wrong track: %#v", entry)
		}
	}
}

func TestSoundOpeningSelectionKeepsBaseProgram(t *testing.T) {
	song := playbackContractSong(t)
	song.Channels[0].Instrument = 99
	song.Tracks[0].Sounds = []guitarpro.TrackSound{{Name: "Base", Path: "factory/base", Role: "main", Program: 27}, {Name: "Opening", Path: "user/opening", Role: "solo", Program: 81}}
	song.Tracks[0].SoundAutomations = []guitarpro.SoundAutomation{{Bar: 0, Position: 0, Sound: 1}}
	before, _ := json.Marshal(song)
	options := guitarpro.ExportOptions{LossPolicy: guitarpro.ExportLossPolicy{RequirePreservation: true}}
	data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, options)
	var loss *guitarpro.ExportLossError
	if len(data) != 0 || !errors.As(err, &loss) || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.normalize.sound-authority" {
		t.Fatalf("sound authority policy = %d bytes, %#v, %v", len(data), report, err)
	}
	options.LossPolicy.AllowedCodes = []string{"gp8.normalize.sound-authority"}
	data, _, err = guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, options)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := guitarpro.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Channels[0].Instrument != 27 || parsed.Tracks[0].Sounds[0].Program != 27 || !slices.Equal(parsed.Tracks[0].SoundAutomations, song.Tracks[0].SoundAutomations) || parsed.Tracks[0].Sounds[parsed.Tracks[0].SoundAutomations[0].Sound].Program != 81 {
		t.Fatalf("base and opening authority = %#v, %#v, %#v", parsed.Channels, parsed.Tracks[0].Sounds, parsed.Tracks[0].SoundAutomations)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("authority reconciliation mutated source")
	}
}
