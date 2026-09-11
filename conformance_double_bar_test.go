// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"reflect"
	"testing"
)

const doubleBarCase = "M07-TERMINAL-DOUBLE-BAR"
const doubleBarConsumerCode = "gp8.omit.double-bar-consumer-terminal"

type doubleBarLineFact struct{ Requested, Actual string }
type doubleBarFacts struct {
	Flags  []bool
	Tracks [][][]doubleBarLineFact
}

func TestConformanceTerminalDoubleBar(t *testing.T) {
	runConformanceTerminalDoubleBar(newConformanceRun(t))
}
func TestAlphaTabTerminalDoubleBar(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformanceTerminalDoubleBar(newConformanceRun(t))
}

func runConformanceTerminalDoubleBar(run *conformanceRun) {
	t := run.t
	for _, flags := range [][]bool{{false}, {true}, {true, false}, {true, false, true}} {
		song := semanticExportProbeSong(t)
		header := song.MeasureHeaders[0]
		measure := song.Tracks[0].Measures[0]
		song.MeasureHeaders = nil
		song.Tracks[0].Measures = nil
		for i, flag := range flags {
			h := header
			h.DoubleBar = flag
			m := measure
			m.HeaderIndex = i
			m.HasDoubleBar = flag
			song.MeasureHeaders = append(song.MeasureHeaders, h)
			song.Tracks[0].Measures = append(song.Tracks[0].Measures, m)
		}
		if err := FinalizeSong(song); err != nil {
			t.Fatal(err)
		}
		wantCodes := []string{}
		if flags[len(flags)-1] {
			wantCodes = append(wantCodes, doubleBarConsumerCode)
		}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: wantCodes}})
		if err != nil {
			t.Fatal(err)
		}
		run.Report(doubleBarCase, reportCodes(report), wantCodes)
		for _, entry := range report.Entries {
			if entry.Location != (ScoreLocation{Measure: len(flags) - 1}) {
				t.Fatal(entry)
			}
		}
		assertDoubleBarWireAndGo(run, data, flags)
		if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
			var facts doubleBarFacts
			readAlphaTabOracleFacts(t, "--double-bars", writeConformanceFixture(t, data), &facts)
			want := append([]bool(nil), flags...)
			want[len(want)-1] = false
			run.Field("MeasureHeader.DoubleBar", facts.Flags, want)
			lines := make([]doubleBarLineFact, len(flags))
			for i, flag := range flags {
				lines[i] = doubleBarLineFact{"Automatic", "Regular"}
				if flag {
					lines[i] = doubleBarLineFact{"LightLight", "LightLight"}
				}
				if i == len(flags)-1 {
					lines[i] = doubleBarLineFact{"Automatic", "LightHeavy"}
				}
			}
			if !reflect.DeepEqual(facts.Tracks, [][][]doubleBarLineFact{{lines}}) {
				t.Fatalf("consumer line styles = %#v, want %#v", facts.Tracks, lines)
			}
		}
	}
	source := parseTestFixture(t, "testdata/gp4/Slides.gp4")
	flags := make([]bool, len(source.MeasureHeaders))
	for i, h := range source.MeasureHeaders {
		flags[i] = h.DoubleBar
	}
	run.Field("MeasureHeader.DoubleBar", flags[len(flags)-1], true)
	data, _, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertDoubleBarWireAndGo(run, data, flags)
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		for _, test := range []struct {
			path string
			flag bool
			line doubleBarLineFact
		}{{"testdata/gp4/Slides.gp4", true, doubleBarLineFact{"LightLight", "LightLight"}}, {writeConformanceFixture(t, data), false, doubleBarLineFact{"Automatic", "LightHeavy"}}} {
			var facts doubleBarFacts
			readAlphaTabOracleFacts(t, "--double-bars", test.path, &facts)
			last := len(facts.Flags) - 1
			run.Field("MeasureHeader.DoubleBar", facts.Flags[last], test.flag)
			for _, track := range facts.Tracks {
				for _, staff := range track {
					if staff[last] != test.line {
						t.Fatalf("%s terminal line = %#v, want %#v", test.path, staff[last], test.line)
					}
				}
			}
		}
	}
}

func assertDoubleBarWireAndGo(run *conformanceRun, data []byte, want []bool) {
	run.t.Helper()
	wires := extractMeasureWire(run.t, data)
	got := make([]bool, len(wires.masterBars))
	for i, bar := range wires.masterBars {
		got[i] = bar.doubleBar
	}
	run.Wire("gpifMasterBar.DoubleBar", got, want)
	parsed, err := Parse(data)
	if err != nil {
		run.t.Fatal(err)
	}
	got = make([]bool, len(parsed.MeasureHeaders))
	for i, h := range parsed.MeasureHeaders {
		got[i] = h.DoubleBar
	}
	run.Field("MeasureHeader.DoubleBar", got, want)
}
