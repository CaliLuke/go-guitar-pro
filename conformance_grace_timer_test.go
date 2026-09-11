// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"encoding/xml"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const graceTimerFixture = "testdata/gp8/grace-timer.gp"

func graceTimerSong(t *testing.T) *Song {
	t.Helper()
	s := parseTestFixture(t, graceTimerFixture)
	s.Version = Version{}
	s.Tracks[0].Settings = TrackSettings{Notation: true}
	return s
}

func TestConformanceGraceTimers(t *testing.T) { runConformanceGraceTimers(newConformanceRun(t)) }
func runConformanceGraceTimers(run *conformanceRun) {
	t := run.t
	s := graceTimerSong(t)
	want := []*BeatTimer{nil, {}, {Milliseconds: ptrTo(int64(0))}, {Milliseconds: ptrTo(int64(99))}, {Milliseconds: ptrTo(maxBeatTimerMilliseconds)}}
	for _, m := range s.Tracks[0].Measures {
		for _, n := range m.Voices[0].Beats[0].Notes {
			for i, g := range n.Effect.Graces {
				run.Preserved("GraceEffect.Timer", g.Timer, want[i])
				if g.Timer != nil {
					run.Field("BeatTimer.Milliseconds", g.Timer.Milliseconds, want[i].Milliseconds)
				}
			}
		}
	}
	data, r := assertConsumerLossPolicy(t, s, []string{})
	run.Report("M15-GRACE-TIMER", reportCodes(r), []string{})
	d := conformanceWireDocument(t, data)
	for i, w := range []*string{nil, ptrTo("-1"), ptrTo("0"), ptrTo("99"), ptrTo("9007199254740991")} {
		run.Wire("gpifBeat.Timer", d.Beats.Beats[i].Timer, w)
		run.Wire("gpifBeat.GraceNotes", d.Beats.Beats[i].GraceNotes, "BeforeBeat")
	}
	run.Wire("gpifBeat.Timer", d.Beats.Beats[5].Timer, ptrTo("25"))
}

func TestAlphaTabGraceTimers(t *testing.T) {
	requireAlphaTabConformance(t)
	var want []beatTimerFact
	readAlphaTabOracleFacts(t, "--beat-timer", graceTimerFixture, &want)
	if len(want) != 18 {
		t.Fatalf("source beat count%d", len(want))
	}
	values := []*int64{nil, nil, ptrTo(int64(0)), ptrTo(int64(99)), ptrTo(maxBeatTimerMilliseconds), ptrTo(int64(25))}
	for _, f := range want {
		if f.Voice == 0 {
			if f.Track != 0 || f.Staff != 0 || f.Bar > 1 || f.Beat > 5 || f.Show != (f.Beat != 0) || !reflect.DeepEqual(f.Milliseconds, values[f.Beat]) || len(f.Notes) != 2 {
				t.Fatalf("source address/value%+v", f)
			}
		}
	}
	s := graceTimerSong(t)
	for generation := 0; generation < 2; generation++ {
		data, _ := assertConsumerLossPolicy(t, s, []string{})
		var got []beatTimerFact
		readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, data), &got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("generation%d\ngot%+v\nwant%+v", generation, got, want)
		}
		var e error
		s, e = Parse(data)
		if e != nil {
			t.Fatal(e)
		}
		s.Version = Version{}
		s.Tracks[0].Settings = TrackSettings{Notation: true}
	}
	// All members of a source grace chord share one timer request by value.
	notes := s.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
	for i := range notes {
		notes[i].Effect.Graces[0].Timer = &BeatTimer{Milliseconds: ptrTo(int64(50))}
		notes[i].Effect.Graces[1].Timer = nil
		notes[i].Effect.Graces[2].Timer = &BeatTimer{}
		*notes[i].Effect.Graces[3].Timer.Milliseconds = 4000
		notes[i].Effect.Graces[4].Timer = nil
	}
	for i := range want {
		f := &want[i]
		if f.Bar == 0 && f.Voice == 0 {
			switch f.Beat {
			case 0:
				f.Show = true
				f.Milliseconds = ptrTo(int64(50))
			case 1, 4:
				f.Show = false
				f.Milliseconds = nil
			case 2:
				f.Milliseconds = nil
			case 3:
				f.Milliseconds = ptrTo(int64(4000))
			}
		}
	}
	data, _ := assertConsumerLossPolicy(t, s, []string{})
	var got []beatTimerFact
	readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, data), &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("edited grace addresses/values differ: got%+v want%+v", got, want)
	}
}

func TestGraceTimerSourceValidationAndContexts(t *testing.T) {
	source, e := os.ReadFile(graceTimerFixture)
	if e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{"-2", "1.5", "NaN", "9007199254740992", "9223372036854775808"} {
		data := rewriteConformanceGPIF(t, source, func(raw string) string { return strings.Replace(raw, "<Timer>99</Timer>", "<Timer>"+bad+"</Timer>", 1) })
		p, e := ParseWithOptions(data, ParseOptions{Strict: true})
		if e == nil || !slices.ContainsFunc(p.Diagnostics, func(d ParseDiagnostic) bool {
			return d.Code == "GPIF.Beat.Timer.InvalidValue" && d.Location.BeatID == "3"
		}) {
			t.Fatal("invalid grace timer accepted", bad, p.Diagnostics)
		}
	}
	for _, kind := range []string{"OnBeat", "orphan"} {
		data := rewriteConformanceGPIF(t, source, func(raw string) string {
			if kind == "OnBeat" {
				return strings.ReplaceAll(raw, "BeforeBeat", "OnBeat")
			}
			return strings.ReplaceAll(raw, "0 1 2 3 4 5", "0 1 2 3 4")
		})
		p, e := ParseWithOptions(data, ParseOptions{Strict: true})
		if e != nil {
			t.Fatal(kind, e)
		}
		out := mustStringNumberExport(t, p.Song)
		if kind == "OnBeat" {
			if !p.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[3].IsOnBeat {
				t.Fatal("on-beat state lost")
			}
		} else if *p.Song.Tracks[0].Measures[0].Voices[0].Beats[3].Timer.Milliseconds != 99 {
			t.Fatal("orphan timer lost")
		}
		if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
			var before, after []beatTimerFact
			readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, data), &before)
			readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, out), &after)
			if !reflect.DeepEqual(before, after) {
				t.Fatal(kind, "consumer addresses changed", before, after)
			}
		}
	}
}

func TestGraceTimerIndependentVoiceScopes(t *testing.T) {
	s := graceTimerSong(t)
	data := mustStringNumberExport(t, s)
	d := conformanceWireDocument(t, data)
	d.Bars.Bars[0].Voices = "0 0 -1 -1"
	raw, e := xml.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	p, e := Parse(conformanceGPIFArchive(t, string(raw)))
	if e != nil {
		t.Fatal(e)
	}
	voices := p.Tracks[0].Measures[0].Voices
	for i := range voices[0].Beats[0].Notes {
		voices[0].Beats[0].Notes[i].Effect.Graces[3].Timer = &BeatTimer{Milliseconds: ptrTo(int64(50))}
	}
	if *voices[1].Beats[0].Notes[0].Effect.Graces[3].Timer.Milliseconds != 99 {
		t.Fatal("shared voice timer")
	}
	if ds := ValidateSong(p); len(ds) != 0 {
		t.Fatal("independent voices conflict", ds)
	}
}

func TestGraceTimerPartialMatchRetainsSourceBeats(t *testing.T) {
	source, e := os.ReadFile(graceTimerFixture)
	if e != nil {
		t.Fatal(e)
	}
	data := rewriteConformanceGPIF(t, source, func(raw string) string { return strings.Replace(raw, "<Notes>2 3</Notes>", "<Notes>2</Notes>", 1) })
	p, e := ParseWithOptions(data, ParseOptions{Strict: true})
	if e != nil {
		t.Fatal(e)
	}
	var original []beatTimerFact
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, data), &original)
	}
	for generation := 0; generation < 2; generation++ {
		for _, m := range p.Song.Tracks[0].Measures {
			beats := m.Voices[0].Beats
			if len(beats) != 6 || len(beats[5].Notes) != 1 || len(beats[5].Notes[0].Effect.Graces) != 0 {
				t.Fatal("partial chord split", beats)
			}
			for i := 0; i < 5; i++ {
				if !beats[i].isGrace || len(beats[i].Notes) != 2 {
					t.Fatal("source grace membership changed", i, beats[i])
				}
			}
			if *beats[3].Timer.Milliseconds != 99 || beats[5].Timer == nil || *beats[5].Timer.Milliseconds != 25 {
				t.Fatal("timer duplicated or moved")
			}
		}
		before, marshalErr := xml.Marshal(p.Song)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if e = FinalizeSong(p.Song); e != nil {
			t.Fatal(e)
		}
		if e = FinalizeSong(p.Song); e != nil {
			t.Fatal(e)
		}
		after, marshalErr := xml.Marshal(p.Song)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if string(before) != string(after) {
			t.Fatal("finalization changed retained grace group")
		}
		out := mustStringNumberExport(t, p.Song)
		if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
			var facts []beatTimerFact
			readAlphaTabOracleFacts(t, "--beat-timer", writeConformanceFixture(t, out), &facts)
			if !reflect.DeepEqual(facts, original) {
				t.Fatalf("generation%d source grace addresses changed: got%+v want%+v", generation, facts, original)
			}
		}
		p, e = ParseWithOptions(out, ParseOptions{Strict: true})
		if e != nil {
			t.Fatal(e)
		}
	}
	// The same partial group without timer requests keeps its existing attachment.
	untimed := rewriteConformanceGPIF(t, data, func(raw string) string {
		for _, value := range []string{"-1", "0", "99", "9007199254740991"} {
			raw = strings.Replace(raw, "<Timer>"+value+"</Timer>", "", 1)
		}
		return raw
	})
	q, e := Parse(untimed)
	if e != nil {
		t.Fatal(e)
	}
	beats := q.Tracks[0].Measures[0].Voices[0].Beats
	if len(beats) != 6 || len(beats[0].Notes) != 1 || len(beats[5].Notes[0].Effect.Graces) != 5 {
		t.Fatal("untimed attachment changed")
	}
}

func TestGraceTimerIndependentStaffScopes(t *testing.T) {
	s := graceTimerSong(t)
	d := conformanceWireDocument(t, mustStringNumberExport(t, s))
	d.Tracks.Tracks[0].Staves.Staff = append(d.Tracks.Tracks[0].Staves.Staff, d.Tracks.Tracks[0].Staves.Staff[0])
	for i := range d.MasterBars.MasterBars {
		d.MasterBars.MasterBars[i].Bars += " " + d.MasterBars.MasterBars[i].Bars
	}
	raw, e := xml.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	p, e := Parse(conformanceGPIFArchive(t, string(raw)))
	if e != nil {
		t.Fatal(e)
	}
	staves := p.Tracks[0].Staves
	for i := range staves[0].Measures[0].Voices[0].Beats[0].Notes {
		staves[0].Measures[0].Voices[0].Beats[0].Notes[i].Effect.Graces[3].Timer = &BeatTimer{Milliseconds: ptrTo(int64(50))}
	}
	if *staves[1].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[3].Timer.Milliseconds != 99 {
		t.Fatal("shared staff timer")
	}
	if ds := ValidateSong(p); len(ds) != 0 {
		t.Fatal("independent staff requests conflict", ds)
	}
}
