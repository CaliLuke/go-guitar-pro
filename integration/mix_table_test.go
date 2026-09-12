// SPDX-License-Identifier: MIT

package integration_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestLegacyMixTableSimultaneousEvents(t *testing.T) {
	source, e := os.ReadFile("../testdata/gp3/other-effects.gp3")
	if e != nil {
		t.Fatal(e)
	}
	// The independently parsed original's second beat begins at 977; its notes
	// begin at 979. GP3 mix-table data precedes those notes.
	if source[977] != 0 {
		t.Fatal("source beat changed")
	}
	record := []byte{73, 12, 5, 255, 255, 255, 255}
	record = binary.LittleEndian.AppendUint32(record, 173)
	record = append(record, 0, 0, 0)
	input := append([]byte{}, source[:979]...)
	input = append(input, record...)
	input = append(input, source[979:]...)
	input[977] |= 0x10
	song, e := gp.Parse(input)
	if e != nil {
		t.Fatal(e)
	}
	change := song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.MixTableChange
	if change == nil || change.Tempo.Value != 173 || change.Instrument.Value != 73 || change.Volume.Value != 12 || change.Balance.Value != 5 {
		t.Fatal("source changes lost")
	}
	data, e := gp.Export(song, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	out, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	tempo, program, volume, pan := false, false, false, false
	for _, a := range out.TempoAutomations {
		tempo = tempo || (a.Bar == 0 && a.Position == .25 && a.Tempo == 173 && !a.Linear)
	}
	for _, a := range out.Tracks[0].SoundAutomations {
		program = program || (a.Bar == 0 && a.Position == .25 && out.Tracks[0].Sounds[a.Sound].Program == 73 && !a.Linear)
	}
	for _, a := range out.VolumeAutomations {
		volume = volume || (a.Track == 0 && a.Bar == 0 && a.Position == .25 && a.Value == .56 && !a.Linear)
	}
	for _, a := range out.PanAutomations {
		pan = pan || (a.Track == 0 && a.Bar == 0 && a.Position == .25 && a.Value == .3125 && !a.Linear)
	}
	if !tempo || !program || !volume || !pan {
		t.Fatalf("events lost: tempo=%t program=%t volume=%t pan=%t", tempo, program, volume, pan)
	}
}

func TestLegacyMixTableCollectionsOwnEdits(t *testing.T) {
	song, e := gp.ParseFile("../testdata/gp3/mix-table-events.gp3")
	if e != nil {
		t.Fatal(e)
	}
	if len(song.VolumeAutomations) != 1 || len(song.PanAutomations) != 1 || len(song.TempoAutomations) != 2 || len(song.Tracks[0].SoundAutomations) != 2 {
		t.Fatal("legacy events not promoted")
	}
	song.TempoAutomations = nil
	song.VolumeAutomations = nil
	song.PanAutomations = nil
	song.Tracks[0].SoundAutomations = nil
	data, e := gp.Export(song, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	out, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.TempoAutomations) != 1 || len(out.VolumeAutomations) != 0 || len(out.PanAutomations) != 0 || len(out.Tracks[0].SoundAutomations) != 0 {
		t.Fatal("cleared legacy events replayed")
	}
}

func TestMixTableProgrammaticProjectionIsReadOnly(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
	beat.Notes[1].Velocity = 47
	beat.Effect.MixTableChange = &gp.MixTableChange{Tempo: &gp.MixTableItem{Value: 173}, Instrument: &gp.MixTableItem{Value: 73}, Volume: &gp.MixTableItem{Value: 12}, Balance: &gp.MixTableItem{Value: 5}}
	before, _ := json.Marshal(song)
	data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.pan-automation-consumer", "gp8.omit.volume-automation-consumer", "gp8.normalize.sound-automation-consumer-position"}}})
	if e != nil {
		t.Fatal(e, report)
	}
	if len(report.Entries) != 3 {
		t.Fatal(report)
	}
	after, _ := json.Marshal(song)
	if !bytes.Equal(before, after) {
		t.Fatal("export mutated source")
	}
	out, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.VolumeAutomations) != 1 || len(out.PanAutomations) != 1 || len(out.Tracks[0].SoundAutomations) != 1 {
		t.Fatal("projection missing")
	}
	// Explicit collections duplicate the raw values once; they do not create
	// additional target events on the next export.
	song.TempoAutomations = out.TempoAutomations
	song.VolumeAutomations = out.VolumeAutomations
	song.PanAutomations = out.PanAutomations
	song.Tracks[0].Sounds = out.Tracks[0].Sounds
	song.Tracks[0].SoundAutomations = out.Tracks[0].SoundAutomations
	data, e = gp.Export(song, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	again, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	if len(again.TempoAutomations) != len(out.TempoAutomations) || len(again.VolumeAutomations) != 1 || len(again.PanAutomations) != 1 || len(again.Tracks[0].SoundAutomations) != 1 {
		t.Fatal("duplicate events")
	}
}

func TestMixTableControllerLossPolicies(t *testing.T) {
	for _, test := range []struct {
		code string
		set  func(*gp.MixTableChange)
	}{
		{"chorus", func(c *gp.MixTableChange) { c.Chorus = &gp.MixTableItem{Value: 7, Duration: 2, AllTracks: true} }},
		{"reverb", func(c *gp.MixTableChange) { c.Reverb = &gp.MixTableItem{Value: 8, Duration: 3} }},
		{"phaser", func(c *gp.MixTableChange) { c.Phaser = &gp.MixTableItem{Value: 9, Duration: 4, AllTracks: true} }},
		{"tremolo", func(c *gp.MixTableChange) { c.Tremolo = &gp.MixTableItem{Value: 10, Duration: 5} }},
		{"tempo-transition", func(c *gp.MixTableChange) { c.Tempo = &gp.MixTableItem{Value: 120, Duration: 3} }},
		{"instrument-transition", func(c *gp.MixTableChange) { c.Instrument = &gp.MixTableItem{Value: 73, Duration: 2} }},
		{"rse-instrument", func(c *gp.MixTableChange) { c.Rse.Instrument = 3 }},
		{"rse-unknown", func(c *gp.MixTableChange) { c.Rse.Unknown = -1 }},
		{"rse-sound-bank", func(c *gp.MixTableChange) { c.Rse.SoundBank = 5 }},
		{"rse-effect-number", func(c *gp.MixTableChange) { c.Rse.EffectNumber = 6 }},
		{"rse-effect-category", func(c *gp.MixTableChange) { c.Rse.EffectCategory = "category" }},
		{"rse-effect", func(c *gp.MixTableChange) { c.Rse.Effect = "effect" }},
		{"use-rse", func(c *gp.MixTableChange) { c.UseRse = true }},
		{"tempo-label", func(c *gp.MixTableChange) { c.TempoName = "orphan label" }},
	} {
		t.Run(test.code, func(t *testing.T) {
			song := notationPublicSong(t)
			song.TempoAutomations = nil
			c := &gp.MixTableChange{}
			test.set(c)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.MixTableChange = c
			code := "gp8.omit.mix-table-" + test.code
			for _, allowed := range [][]string{nil, {"unrelated"}, {code}} {
				data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				if len(report.Entries) != 1 || report.Entries[0].Code != code {
					t.Fatal(report)
				}
				if slices.Contains(allowed, code) {
					if e != nil {
						t.Fatal(e)
					}
				} else if e == nil || len(data) != 0 {
					t.Fatal("unallowed loss accepted")
				}
			}
		})
	}
}

func TestMixTableExactTimingAndBounds(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	beats[1].Notes[1].Velocity = 47
	beats[0].Duration.TupletEnters = 3
	beats[0].Duration.TupletTimes = 2
	beats[1].Effect.MixTableChange = &gp.MixTableChange{Volume: &gp.MixTableItem{Value: 12}}
	data, e := gp.Export(song, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	out, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.VolumeAutomations) != 1 || out.VolumeAutomations[0].Position != 1.0/6 {
		t.Fatal("stale Start cache supplied event position", out.VolumeAutomations)
	}
	for _, test := range []struct {
		name string
		item func(*gp.MixTableChange) *gp.MixTableItem
		bad  int32
	}{
		{"tempo", func(c *gp.MixTableChange) *gp.MixTableItem { c.Tempo = &gp.MixTableItem{}; return c.Tempo }, 0},
		{"instrument", func(c *gp.MixTableChange) *gp.MixTableItem { c.Instrument = &gp.MixTableItem{}; return c.Instrument }, 128},
		{"volume", func(c *gp.MixTableChange) *gp.MixTableItem { c.Volume = &gp.MixTableItem{}; return c.Volume }, 17},
		{"balance", func(c *gp.MixTableChange) *gp.MixTableItem { c.Balance = &gp.MixTableItem{}; return c.Balance }, -1},
	} {
		s := notationPublicSong(t)
		c := &gp.MixTableChange{}
		test.item(c).Value = test.bad
		s.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.MixTableChange = c
		code := "score.mix-table." + test.name
		if !slices.ContainsFunc(gp.ValidateSong(s), func(d gp.ScoreDiagnostic) bool { return d.Code == code }) {
			t.Fatal("invalid raw value undiagnosed")
		}
		data, _, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 {
			t.Fatal("invalid raw value exported")
		}
	}
}

func TestMixTableCollisionPolicies(t *testing.T) {
	for _, controller := range []string{"tempo", "instrument", "volume", "balance"} {
		t.Run(controller, func(t *testing.T) {
			song := dynamicPolicySong(t, 47, false)
			beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
			beat.Notes[1].Velocity = 47
			c := &gp.MixTableChange{}
			beat.Effect.MixTableChange = c
			codes := []string{"gp8.normalize.mix-table-" + controller + "-authority"}
			switch controller {
			case "tempo":
				c.Tempo = &gp.MixTableItem{Value: 173}
				song.TempoAutomations = []gp.TempoAutomation{{Bar: 0, Position: .25, Tempo: 181}}
			case "instrument":
				c.Instrument = &gp.MixTableItem{Value: 73}
				song.Tracks[0].Sounds = []gp.TrackSound{{Name: "initial", Program: song.Channels[0].Instrument}, {Name: "explicit", Program: 81}}
				song.Tracks[0].SoundAutomations = []gp.SoundAutomation{{Position: .25, Sound: 1}}
				codes = append(codes, "gp8.normalize.sound-automation-consumer-position")
			case "volume":
				c.Volume = &gp.MixTableItem{Value: 12}
				song.VolumeAutomations = []gp.VolumeAutomation{{Position: .25, Value: .5}}
				codes = append(codes, "gp8.omit.volume-automation-consumer")
			case "balance":
				c.Balance = &gp.MixTableItem{Value: 5}
				song.PanAutomations = []gp.PanAutomation{{Position: .25, Value: .75}}
				codes = append(codes, "gp8.omit.pan-automation-consumer")
			}
			for mask := 0; mask < (1 << len(codes)); mask++ {
				allowed := []string{"unrelated"}
				for i, code := range codes {
					if mask&(1<<i) != 0 {
						allowed = append(allowed, code)
					}
				}
				data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				got := []string{}
				for _, entry := range report.Entries {
					got = append(got, entry.Code)
				}
				want := slices.Clone(codes)
				slices.Sort(got)
				slices.Sort(want)
				if !slices.Equal(got, want) {
					t.Fatal(report)
				}
				if mask != (1<<len(codes))-1 {
					if e == nil || len(data) != 0 {
						t.Fatal("partial allowance accepted")
					}
					continue
				}
				if e != nil {
					t.Fatal(e)
				}
				out, e := gp.Parse(data)
				if e != nil {
					t.Fatal(e)
				}
				switch controller {
				case "tempo":
					if out.TempoAutomations[1].Tempo != 181 {
						t.Fatal("tempo override lost")
					}
				case "instrument":
					a := out.Tracks[0].SoundAutomations
					if len(a) != 1 || out.Tracks[0].Sounds[a[0].Sound].Program != 81 {
						t.Fatal("program override lost")
					}
				case "volume":
					if len(out.VolumeAutomations) != 1 || out.VolumeAutomations[0].Value != .5 {
						t.Fatal("volume override lost")
					}
				case "balance":
					if len(out.PanAutomations) != 1 || out.PanAutomations[0].Value != .75 {
						t.Fatal("balance override lost")
					}
				}
			}
		})
	}
}

func TestMixTableProgramKeepsSelectedBank(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
	beat.Notes[1].Velocity = 47
	song.Channels[0].Bank = 77
	song.Tracks[0].Sounds = []gp.TrackSound{{Name: "initial", Program: song.Channels[0].Instrument, Bank: 77}, {Name: "selected", Program: 26, Bank: 256}}
	song.Tracks[0].SoundAutomations = []gp.SoundAutomation{{Sound: 1}}
	beat.Effect.MixTableChange = &gp.MixTableChange{Instrument: &gp.MixTableItem{Value: 73}}
	data, e := gp.Export(song, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	out, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	events := out.Tracks[0].SoundAutomations
	if len(events) != 2 || events[1].Position != .25 || out.Tracks[0].Sounds[events[1].Sound].Program != 73 || out.Tracks[0].Sounds[events[1].Sound].Bank != 256 {
		t.Fatal("program reset selected bank", events)
	}
	if len(song.Tracks[0].Sounds) != 2 || len(song.Tracks[0].SoundAutomations) != 1 {
		t.Fatal("generated sound changed input")
	}
}
