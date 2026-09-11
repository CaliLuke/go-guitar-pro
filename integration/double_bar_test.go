// SPDX-License-Identifier: MIT

package integration_test

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

const terminalDoubleBarCode = "gp8.omit.double-bar-consumer-terminal"

func TestGP8TerminalDoubleBarPolicy(t *testing.T) {
	for _, flags := range [][]bool{{false}, {true}, {true, false}, {true, false, true}} {
		song := doubleBarSong(t, flags)
		before := slices.Clone(song.MeasureHeaders)
		report := gp.PreflightExport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		terminal := flags[len(flags)-1]
		want := 0
		if terminal {
			want = 1
		}
		if len(report.Entries) != want {
			t.Fatalf("flags %v: report %#v", flags, report)
		}
		if terminal {
			e := report.Entries[0]
			if e.Code != terminalDoubleBarCode || e.Disposition != gp.ExportDispositionOmitted || e.Location != (gp.ScoreLocation{Measure: len(flags) - 1}) || !strings.Contains(e.Reason, "GPIF") || !strings.Contains(e.Reason, "AlphaTab") {
				t.Fatalf("wrong scoped loss: %#v", e)
			}
		}
		for _, allowed := range [][]string{nil, {"gp8.normalize.measure-double-bar-authority"}, {terminalDoubleBarCode}} {
			data, got, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
			if !reflect.DeepEqual(got, report) {
				t.Fatal("preflight and export reports differ")
			}
			if terminal && !slices.Contains(allowed, terminalDoubleBarCode) {
				var loss *gp.ExportLossError
				if len(data) != 0 || !errors.As(err, &loss) || len(loss.Entries) != 1 || loss.Entries[0].Code != terminalDoubleBarCode {
					t.Fatalf("strict export: %d bytes, %v", len(data), err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				assertDoubleBarOutput(t, data, flags)
			}
		}
		if !reflect.DeepEqual(song.MeasureHeaders, before) {
			t.Fatal("export mutated headers")
		}
	}
}

func TestGP4TerminalDoubleBarPreservation(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp4/Slides.gp4")
	if err != nil {
		t.Fatal(err)
	}
	flags := make([]bool, len(song.MeasureHeaders))
	for i, h := range song.MeasureHeaders {
		flags[i] = h.DoubleBar
	}
	if !flags[len(flags)-1] {
		t.Fatal("fixture lost authored final DoubleBar")
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range report.Entries {
		if e.Code == terminalDoubleBarCode {
			count++
			if e.Location != (gp.ScoreLocation{Measure: len(flags) - 1}) {
				t.Fatal(e)
			}
		}
	}
	if count != 1 {
		t.Fatalf("terminal reports=%d", count)
	}
	assertDoubleBarOutput(t, data, flags)
}

func doubleBarSong(t *testing.T, flags []bool) *gp.Song {
	t.Helper()
	duration := gp.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}
	song := &gp.Song{Tempo: 120, Channels: []gp.MidiChannel{{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}}, Tracks: []gp.Track{{Name: "Guitar", Visible: true, Settings: gp.TrackSettings{Notation: true}, Strings: []gp.GuitarString{{Number: 1, Value: 64}}}}}
	for index, flag := range flags {
		song.MeasureHeaders = append(song.MeasureHeaders, gp.MeasureHeader{DoubleBar: flag, TimeSignature: gp.TimeSignature{Numerator: 4, Denominator: duration, Beams: [4]uint8{2, 2, 2, 2}}})
		song.Tracks[0].Measures = append(song.Tracks[0].Measures, gp.Measure{HeaderIndex: index, HasDoubleBar: flag, Voices: []gp.Voice{{Beats: []gp.Beat{{Status: gp.BeatStatusRest, Duration: duration, Dynamics: 95}}}}})
	}
	if err := gp.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if d := gp.ValidateSong(song); len(d) != 0 {
		t.Fatal(d)
	}
	return song
}

func assertDoubleBarOutput(t *testing.T, data []byte, want []bool) {
	t.Helper()
	song, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]bool, len(song.MeasureHeaders))
	for i, h := range song.MeasureHeaders {
		got[i] = h.DoubleBar
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Go flags=%v want %v", got, want)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	file, err := archive.Open("Content/score.gpif")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := xml.NewDecoder(file)
	var flags []bool
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if start, ok := token.(xml.StartElement); ok {
			switch start.Name.Local {
			case "MasterBar":
				flags = append(flags, false)
			case "DoubleBar":
				flags[len(flags)-1] = true
			}
		}
	}
	if !slices.Equal(flags, want) {
		t.Fatalf("XML flags=%v want %v", flags, want)
	}
}
