// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"encoding/xml"
	"os"
	"reflect"
	"testing"
)

const restVoiceCase = "M10-TIMED-EMPTY-VOICE"

func TestConformanceTimedEmptyVoice(t *testing.T) {
	runConformanceTimedEmptyVoice(newConformanceRun(t))
}
func TestAlphaTabTimedEmptyVoice(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformanceTimedEmptyVoice(newConformanceRun(t))
}

type restVoiceFact struct {
	Index int
	Empty bool
	Beats []restBeatFact
}
type restBeatFact struct {
	Empty, Rest                                                  bool
	Duration, Dots, Numerator, Denominator, Start, Length, Notes int
	Text                                                         string
}

func runConformanceTimedEmptyVoice(run *conformanceRun) {
	t := run.t
	for _, empty := range []bool{false, true} {
		song := semanticExportProbeSong(t)
		first := song.Tracks[0].Measures[0].Voices[0].Beats[0]
		first.Text = "normal"
		middle := Beat{Duration: Duration{Value: 8, Dotted: true, TupletEnters: 3, TupletTimes: 2}, Status: BeatStatusRest, Dynamics: Forte, Text: "timed empty"}
		if empty {
			middle.Status = BeatStatusEmpty
		}
		rest := Beat{Duration: defaultDuration(), Status: BeatStatusRest, Dynamics: Forte, Text: "ordinary rest"}
		song.Tracks[0].Measures[0].Voices = []Voice{{Beats: []Beat{first, middle, rest}}, {}, {Beats: []Beat{first}}}
		if err := FinalizeSong(song); err != nil {
			t.Fatal(err)
		}
		codes := []string{}
		if empty {
			codes = append(codes, "gp8.normalize.empty-beat")
		}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: codes}})
		if err != nil {
			t.Fatal(err)
		}
		run.Report(restVoiceCase, reportCodes(report), codes)
		for _, entry := range report.Entries {
			if entry.Location != (ScoreLocation{Beat: 1}) {
				t.Fatal(entry)
			}
		}
		var wire struct {
			Bars   []struct{ Voices string } `xml:"Bars>Bar"`
			Voices []struct {
				ID    string `xml:"id,attr"`
				Beats string
			} `xml:"Voices>Voice"`
			Beats []struct {
				ID              string `xml:"id,attr"`
				Notes, FreeText string
				Rhythm          struct {
					Ref string `xml:"ref,attr"`
				}
			} `xml:"Beats>Beat"`
		}
		if decodeErr := xml.Unmarshal(conformanceBarreGPIF(t, data), &wire); decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if len(wire.Bars) != 1 || len(wire.Voices) != 2 || len(wire.Beats) != 4 {
			t.Fatalf("wire graph=%#v", wire)
		}
		run.Wire("gpifBar.Voices", wire.Bars[0].Voices, "0 -1 1 -1")
		run.Wire("gpifVoice.Beats", []string{wire.Voices[0].Beats, wire.Voices[1].Beats}, []string{"0 1 2", "3"})
		run.Wire("gpifBeat.Notes", []string{wire.Beats[1].Notes, wire.Beats[2].Notes}, []string{"", ""})
		run.Wire("gpifBeat.FreeText", []string{wire.Beats[1].FreeText, wire.Beats[2].FreeText}, []string{"timed empty", "ordinary rest"})
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		voices := parsed.Tracks[0].Measures[0].Voices
		if len(voices) != 3 {
			t.Fatal("lost voice slot")
		}
		run.Field("Voice.Beats", []int{len(voices[0].Beats), len(voices[1].Beats), len(voices[2].Beats)}, []int{3, 0, 1})
		for i, beat := range voices[0].Beats {
			source := song.Tracks[0].Measures[0].Voices[0].Beats[i]
			status := source.Status
			if status == BeatStatusEmpty {
				status = BeatStatusRest
			}
			run.Field("Beat.Status", beat.Status, status)
			run.Field("Beat.Text", beat.Text, source.Text)
			run.Field("Beat.Duration", beat.Duration, source.Duration)
			run.Field("Beat.ExactStart", beat.ExactStart, source.ExactStart)
			run.Field("Beat.Notes", len(beat.Notes), len(source.Notes))
		}
		if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
			var facts [][][][]restVoiceFact
			readAlphaTabOracleFacts(t, "--voice-rests", writeConformanceFixture(t, data), &facts)
			want := [][][][]restVoiceFact{{{{
				{Index: 0, Beats: []restBeatFact{{Duration: 4, Numerator: -1, Denominator: -1, Start: 0, Length: 960, Text: "normal", Notes: 1}, {Rest: true, Duration: 8, Dots: 1, Numerator: 3, Denominator: 2, Start: 960, Length: 480, Text: "timed empty"}, {Rest: true, Duration: 4, Numerator: -1, Denominator: -1, Start: 1440, Length: 960, Text: "ordinary rest"}}},
				{Index: 1, Empty: true, Beats: []restBeatFact{{Empty: true, Rest: true, Duration: 4, Numerator: -1, Denominator: -1, Start: 0, Length: 960}}},
				{Index: 2, Beats: []restBeatFact{{Duration: 4, Numerator: -1, Denominator: -1, Start: 0, Length: 960, Text: "normal", Notes: 1}}},
				{Index: 3, Empty: true, Beats: []restBeatFact{{Empty: true, Rest: true, Duration: 4, Numerator: -1, Denominator: -1, Length: 960}}},
			}}}}
			if !reflect.DeepEqual(facts, want) {
				t.Fatalf("raw consumer=%#v want %#v", facts, want)
			}
		}
	}
}
