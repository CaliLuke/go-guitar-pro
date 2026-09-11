// SPDX-License-Identifier: MIT
package integration_test

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"os"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

type layoutWire struct {
	Score struct {
		Default string `xml:"ScoreSystemsDefaultLayout"`
		Systems string `xml:"ScoreSystemsLayout"`
	} `xml:"Score"`
	Tracks []struct {
		Default string `xml:"SystemsDefautLayout"`
		Systems string `xml:"SystemsLayout"`
	} `xml:"Tracks>Track"`
}

func publicLayoutWire(t *testing.T, data []byte) layoutWire {
	t.Helper()
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range z.File {
		if f.Name != "Content/score.gpif" {
			continue
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		_ = r.Close()
		if e != nil {
			t.Fatal(e)
		}
		var v layoutWire
		if e = xml.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	t.Fatal("missing GPIF")
	return layoutWire{}
}
func TestPublicLayoutPreservesSourceFields(t *testing.T) {
	for _, name := range []string{"multi-track-different.gp", "resized.gp"} {
		t.Run(name, func(t *testing.T) {
			data, e := os.ReadFile("../testdata/gp7/" + name)
			if e != nil {
				t.Fatal(e)
			}
			song, e := gp.Parse(data)
			if e != nil {
				t.Fatal(e)
			}
			out, e := gp.Export(song, gp.ExportFormatGP8)
			if e != nil {
				t.Fatal(e)
			}
			if got, want := publicLayoutWire(t, out), publicLayoutWire(t, data); !reflect.DeepEqual(got, want) {
				t.Fatalf("layout=%#v want %#v", got, want)
			}
		})
	}
}

func layoutPublicSong(t *testing.T) *gp.Song {
	t.Helper()
	s, e := gp.ParseFile("../testdata/gp8/section-track-names.gp")
	if e != nil {
		t.Fatal(e)
	}
	s.Version = gp.Version{}
	for i := range s.Tracks {
		s.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
	}
	return s
}
func TestPublicLayoutEditsAndInheritance(t *testing.T) {
	s := layoutPublicSong(t)
	s.SystemLayout = &gp.SystemLayout{DefaultBarsPerSystem: 5, BarsPerSystem: []int{2, 4}}
	s.Tracks[0].SystemLayout = &gp.SystemLayout{DefaultBarsPerSystem: 2, BarsPerSystem: []int{1, 3, 2}}
	s.Tracks[1].SystemLayout = &gp.SystemLayout{DefaultBarsPerSystem: 6, BarsPerSystem: []int{4, 2}}
	a, b, c := .5, 2.0, .75
	s.MeasureHeaders[0].DisplayScale = &a
	s.Tracks[0].Measures[0].DisplayScale = &b
	s.Tracks[1].Measures[0].DisplayScale = &c
	data, r, e := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if e != nil || len(r.Entries) != 0 {
		t.Fatal(e, r)
	}
	p, e := gp.Parse(data)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(p.SystemLayout, s.SystemLayout) || !reflect.DeepEqual(p.Tracks[0].SystemLayout, s.Tracks[0].SystemLayout) || *p.MeasureHeaders[0].DisplayScale != a || *p.Tracks[0].Measures[0].DisplayScale != b || *p.Tracks[1].Measures[0].DisplayScale != c {
		t.Fatal("authored layout changed")
	}
	p.Tracks[0].SystemLayout.BarsPerSystem[0] = 4
	*p.Tracks[0].Measures[0].DisplayScale = 1.25
	if s.Tracks[0].SystemLayout.BarsPerSystem[0] != 1 || *s.Tracks[0].Measures[0].DisplayScale != 2 || p.Tracks[1].SystemLayout.BarsPerSystem[0] != 4 {
		t.Fatal("layout ownership alias")
	}
	p.Tracks[1].SystemLayout = nil
	p.Version = gp.Version{}
	for i := range p.Tracks {
		p.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
	}
	for _, allowed := range [][]string{nil, {"unrelated"}, {"gp8.normalize.track-layout-inheritance"}} {
		out, report, err := gp.ExportWithReport(p, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if len(allowed) == 1 && allowed[0] == "gp8.normalize.track-layout-inheritance" {
			if err != nil {
				t.Fatal(err, report)
			}
			q, e := gp.Parse(out)
			if e != nil || !reflect.DeepEqual(q.Tracks[1].SystemLayout, p.SystemLayout) {
				t.Fatal("inheritance", e)
			}
			if p.Tracks[1].SystemLayout != nil {
				t.Fatal("overwrote authored absence")
			}
		} else if err == nil || len(out) != 0 {
			t.Fatal("unallowed inheritance accepted")
		}
	}
}
func TestPublicLayoutRejectsInvalidValues(t *testing.T) {
	for _, count := range []int{-1, 0, 2147483648} {
		s := layoutPublicSong(t)
		s.SystemLayout = &gp.SystemLayout{BarsPerSystem: []int{count}}
		out, _, e := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{})
		if e == nil || len(out) != 0 {
			t.Fatalf("invalid count %d accepted", count)
		}
	}
	for _, scale := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		s := layoutPublicSong(t)
		s.Tracks[0].Measures[0].DisplayScale = &scale
		out, _, e := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{})
		if e == nil || len(out) != 0 {
			t.Fatalf("invalid scale %v accepted", scale)
		}
	}
}
