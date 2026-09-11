// SPDX-License-Identifier: MIT

package integration_test

import (
	"math"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8PanAutomation(t *testing.T) {
	song := doubleBarSong(t, []bool{false, false})
	song.PanAutomations = []gp.PanAutomation{{Track: 0, Bar: 0, Position: 0.25, Value: 0}, {Track: 0, Bar: 1, Position: 0.75, Value: 1, Linear: true}}
	before := append([]gp.PanAutomation(nil), song.PanAutomations...)
	balance := song.Channels[0].Balance
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.pan-automation-consumer"}}})
	if err != nil || len(report.Entries) != 2 {
		t.Fatalf("export %v %#v", err, report)
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed.PanAutomations, before) || parsed.Channels[0].Balance != balance {
		t.Fatalf("events %#v balance %d", parsed.PanAutomations, parsed.Channels[0].Balance)
	}
	if !reflect.DeepEqual(song.PanAutomations, before) || song.Channels[0].Balance != balance {
		t.Fatal("mutated input")
	}
	parsed.PanAutomations = nil
	data, _, err = gp.ExportWithReport(parsed, gp.ExportFormatGP8, gp.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	cleared, err := gp.Parse(data)
	if err != nil || len(cleared.PanAutomations) != 0 {
		t.Fatal("pan clearing failed", err)
	}
}

func TestPanAutomationBoundsAndPolicy(t *testing.T) {
	for _, test := range []struct {
		name   string
		events []gp.PanAutomation
		code   string
	}{
		{"lower", []gp.PanAutomation{{Value: 0}}, ""}, {"upper", []gp.PanAutomation{{Position: 1, Value: 1}}, ""},
		{"negative track", []gp.PanAutomation{{Track: -1}}, "score.pan-automation.value"}, {"missing track", []gp.PanAutomation{{Track: 1}}, "score.pan-automation.value"},
		{"negative bar", []gp.PanAutomation{{Bar: -1}}, "score.pan-automation.value"}, {"missing bar", []gp.PanAutomation{{Bar: 2}}, "score.pan-automation.value"},
		{"negative value", []gp.PanAutomation{{Value: -0.1}}, "score.pan-automation.value"}, {"large value", []gp.PanAutomation{{Value: 1.1}}, "score.pan-automation.value"},
		{"NaN", []gp.PanAutomation{{Value: math.NaN()}}, "score.pan-automation.value"}, {"infinite", []gp.PanAutomation{{Position: math.Inf(1)}}, "score.pan-automation.value"},
		{"negative position", []gp.PanAutomation{{Position: -0.1}}, "score.pan-automation.value"}, {"large position", []gp.PanAutomation{{Position: 1.1}}, "score.pan-automation.value"},
		{"bar order", []gp.PanAutomation{{Bar: 1}, {Bar: 0}}, "score.pan-automation.order"}, {"position order", []gp.PanAutomation{{Position: 0.75}, {Position: 0.25}}, "score.pan-automation.order"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := doubleBarSong(t, []bool{false, false})
			song.PanAutomations = test.events
			ds := gp.ValidateSong(song)
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.pan-automation-consumer"}}})
			if test.code == "" {
				if len(ds) != 0 || err != nil || len(data) == 0 {
					t.Fatalf("valid %v %#v", err, ds)
				}
				data, _, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.volume-automation-consumer"}}})
				if err == nil || len(data) != 0 {
					t.Fatal("unrelated allowance bypassed pan loss")
				}
				return
			}
			if len(ds) != 1 || ds[0].Code != test.code || err == nil || len(data) != 0 || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.reject."+test.code {
				t.Fatalf("invalid %v %#v %#v", err, ds, report)
			}
		})
	}
}

func TestLegacyPanEvents(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp5/motherload-percussion-grace.gp5")
	if err != nil {
		t.Fatal(err)
	}
	want := []gp.PanAutomation{{Track: 2, Bar: 143, Value: 0, Linear: true}, {Track: 3, Bar: 143, Value: 1, Linear: true}}
	if !reflect.DeepEqual(song.PanAutomations, want) {
		t.Fatalf("events %#v", song.PanAutomations)
	}
	before := append([]gp.MidiChannel(nil), song.Channels...)
	song.PanAutomations[0].Value = 0.5
	data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed.PanAutomations, song.PanAutomations) || !reflect.DeepEqual(song.Channels, before) {
		t.Fatal("pan ownership or initial balance changed")
	}
}
