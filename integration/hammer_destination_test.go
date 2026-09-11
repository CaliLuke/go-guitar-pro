// SPDX-License-Identifier: MIT
package integration_test

import (
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestPublicHammerDestinationPreservation(t *testing.T) {
	for _, shape := range []string{"origin", "destination", "complete"} {
		t.Run(shape, func(t *testing.T) {
			s := dynamicPolicySong(t, 95, false)
			beats := s.Tracks[0].Measures[0].Voices[0].Beats
			n := beats[1].Notes[0]
			beats[0] = beats[1]
			beats[0].Notes = []gp.Note{n}
			beats[1].Notes = []gp.Note{n}
			beats[0].Notes[0].Effect.Hammer = shape != "destination"
			beats[1].Notes[0].Effect.HammerDestination = shape != "origin"
			if shape == "origin" {
				beats[1].Notes = nil
				beats[1].Status = gp.BeatStatusRest
			}
			if err := gp.FinalizeSong(s); err != nil {
				t.Fatal(err)
			}
			data, err := gp.Export(s, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			p, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := p.Tracks[0].Measures[0].Voices[0].Beats
			if got[0].Notes[0].Effect.Hammer != (shape != "destination") {
				t.Fatal("origin marker changed")
			}
			if shape != "origin" && !got[1].Notes[0].Effect.HammerDestination {
				t.Fatal("authored destination lost")
			}
			if shape == "destination" && got[0].Notes[0].Effect.Hammer {
				t.Fatal("invented origin")
			}
		})
	}
}

func TestPublicHammerEndpointEditsAndPolicy(t *testing.T) {
	s := dynamicPolicySong(t, 95, false)
	beats := s.Tracks[0].Measures[0].Voices[0].Beats
	first := beats[1]
	first.Notes = []gp.Note{beats[1].Notes[0]}
	first.Notes[0].Velocity = 95
	second := first
	second.Notes = append([]gp.Note(nil), first.Notes...)
	first.Notes[0].Effect.Hammer = true
	second.Notes[0].String = 2
	second.Notes[0].Effect.HammerDestination = true
	s.Tracks[0].Measures[0].Voices[0].Beats = []gp.Beat{first, second}
	if e := gp.FinalizeSong(s); e != nil {
		t.Fatal(e)
	}
	codes := []string{"gp8.omit.hammer-origin-consumer", "gp8.omit.hammer-destination-consumer"}
	for mask := 0; mask < 4; mask++ {
		allowed := []string{}
		for i, code := range codes {
			if mask&(1<<i) != 0 {
				allowed = append(allowed, code)
			}
		}
		data, r, e := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if len(r.Entries) != 2 || r.Entries[0].Code != codes[0] || r.Entries[1].Code != codes[1] || r.Entries[0].Location.Beat != 0 || r.Entries[1].Location.Beat != 1 {
			t.Fatal("wrong endpoint losses", r)
		}
		if (e == nil) != (mask == 3) || ((mask != 3) && len(data) != 0) {
			t.Fatalf("mask%d data%d err%v", mask, len(data), e)
		}
	}
	if !first.Notes[0].Effect.Hammer || !second.Notes[0].Effect.HammerDestination {
		t.Fatal("export mutated authored endpoints")
	}
	// Both edits are authoritative after parsing. Finalization cannot invent a chain.
	data, e := gp.Export(s, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	p, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	got := p.Tracks[0].Measures[0].Voices[0].Beats
	got[0].Notes[0].Effect.Hammer = false
	got[1].Notes[0].Effect.HammerDestination = false
	if e = gp.FinalizeSong(p); e != nil {
		t.Fatal(e)
	}
	data, e = gp.Export(p, gp.ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	q, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	for _, beat := range q.Tracks[0].Measures[0].Voices[0].Beats {
		for _, n := range beat.Notes {
			if n.Effect.Hammer || n.Effect.HammerDestination {
				t.Fatal("cleared marker reappeared")
			}
		}
	}
}
