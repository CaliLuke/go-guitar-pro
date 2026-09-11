// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"reflect"
	"strings"
	"testing"
)

const beatTimerCase = "M10-BEAT-TIMER"
const beatTimerFixture = "testdata/gp8/beat-timer.gp"
const beatTimerValue = "absent derived zero and nonzero authored milliseconds"

func TestConformanceBeatTimer(t *testing.T) { runConformanceBeatTimer(newConformanceRun(t)) }
func runConformanceBeatTimer(run *conformanceRun) {
	t := run.t
	song := beatTimerSong(t)
	first := song.Tracks[0].Measures[0].Voices[0].Beats
	run.Preserved("Beat.Timer", first[0].Timer, (*BeatTimer)(nil))
	run.Preserved("Beat.Timer", first[1].Timer, &BeatTimer{})
	for _, n := range []int{2, 3} {
		run.Preserved("BeatTimer.Milliseconds", first[n].Timer.Milliseconds, ptrTo([]int64{0, 12345}[n-2]))
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err)
	}
	run.Report(beatTimerCase, reportCodes(report), []string{})
	wire := conformanceWireDocument(t, data)
	for i, want := range []*string{nil, ptrTo("-1"), ptrTo("0"), ptrTo("12345")} {
		run.Wire("gpifBeat.Timer", wire.Beats.Beats[i].Timer, want)
	}
	for _, value := range []int64{0, 1, maxBeatTimerMilliseconds - 1, maxBeatTimerMilliseconds} {
		first[0].Timer = &BeatTimer{Milliseconds: ptrTo(value)}
		parsed, parseErr := Parse(mustStringNumberExport(t, song))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		run.Preserved("BeatTimer.Milliseconds", parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Timer.Milliseconds, ptrTo(value))
	}
}
func beatTimerSong(t *testing.T) *Song {
	t.Helper()
	song := parseTestFixture(t, beatTimerFixture)
	song.Version = Version{}
	song.Tracks[0].Settings = TrackSettings{Notation: true}
	return song
}
func TestBeatTimerInvalidSource(t *testing.T) {
	raw := string(conformanceBarreGPIF(t, mustStringNumberExport(t, beatTimerSong(t))))
	for _, text := range []string{"-2", "1.5", "NaN", "Infinity", "9007199254740992", "9223372036854775808", "12junk"} {
		source := conformanceGPIFArchive(t, strings.Replace(raw, "<Timer>0</Timer>", "<Timer>"+text+"</Timer>", 1))
		parsed, err := ParseWithOptions(source, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, d := range parsed.Diagnostics {
			if d.Code == "GPIF.Beat.Timer.InvalidValue" {
				found = true
				if d.Kind != ParseDiagnosticInvalidData || d.ObjectID != "2" || d.SourcePath != "/GPIF/Beats/Beat[@id=\"2\"]/Timer" {
					t.Fatal(d)
				}
			}
		}
		if !found {
			t.Fatalf("missing invalid timer diagnostic for %s", text)
		}
		if _, err = ParseWithOptions(source, ParseOptions{Strict: true}); err == nil {
			t.Fatalf("strict accepted %s", text)
		}
		timer := parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[2].Timer
		if timer == nil || timer.Milliseconds != nil {
			t.Fatal("invalid timer fabricated a value")
		}
	}
}

type beatTimerNoteFact struct{ String, Fret, MIDI int }
type beatTimerFact struct {
	Track, Staff, Bar, Voice, Beat int
	Show                           bool
	Milliseconds                   *int64
	Notes                          []beatTimerNoteFact
}

func TestAlphaTabBeatTimer(t *testing.T) {
	requireAlphaTabConformance(t)
	want := []beatTimerFact{}
	for bar := 0; bar < 2; bar++ {
		for voice := 0; voice < 4; voice++ {
			count := 1
			if voice == 0 {
				count = 4
			}
			for beat := 0; beat < count; beat++ {
				fact := beatTimerFact{Bar: bar, Voice: voice, Beat: beat, Notes: []beatTimerNoteFact{}}
				if voice == 0 {
					fact.Notes = []beatTimerNoteFact{{4, 3, 53}}
					fact.Show = bar != 0 || beat != 0
					values := [][]*int64{{nil, nil, ptrTo(int64(0)), ptrTo(int64(12345))}, {nil, ptrTo(maxBeatTimerMilliseconds), ptrTo(int64(25)), ptrTo(int64(12345))}}
					fact.Milliseconds = values[bar][beat]
				}
				want = append(want, fact)
			}
		}
	}
	var got []beatTimerFact
	readAlphaTabOracleFacts(t, "--beat-timer", beatTimerFixture, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("source timer facts=%#v", got)
	}
	song := beatTimerSong(t)
	for _, edited := range []bool{false, true} {
		if edited {
			beats := song.Tracks[0].Measures[0].Voices[0].Beats
			beats[0].Timer = &BeatTimer{Milliseconds: ptrTo(int64(50))}
			beats[1].Timer = nil
			beats[2].Timer = &BeatTimer{}
			*beats[3].Timer.Milliseconds = 4000
			want[0].Show = true
			want[0].Milliseconds = ptrTo(int64(50))
			want[1].Show = false
			want[2].Milliseconds = nil
			want[3].Milliseconds = ptrTo(int64(4000))
		}
		readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("edited=%t timer facts=%#v", edited, got)
		}
	}
	// Existing GP7 authored timer values also retain exact addresses and numbers.
	readAlphaTabOracleFacts(t, "--beat-timer", "testdata/gp7/timer.gp", &want)
	source := parseTestFixture(t, "testdata/gp7/timer.gp")
	readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, mustStringNumberExport(t, source)), &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("existing GP7 timer facts changed")
	}
}
