// SPDX-License-Identifier: MIT

package integration_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestWahLegacySourceExport(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp5/Wah.gp5")
	if err != nil {
		t.Fatal(err)
	}
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name == "Content/score.gpif" {
			r, e := file.Open()
			if e != nil {
				t.Fatal(e)
			}
			wire, e := io.ReadAll(r)
			_ = r.Close()
			if e != nil {
				t.Fatal(e)
			}
			if strings.Count(string(wire), "<Wah>Open</Wah>") != 4 || strings.Count(string(wire), "<Wah>Closed</Wah>") != 4 {
				t.Fatalf("GP5 wah events discarded: open=%d closed=%d first=%#v", strings.Count(string(wire), "<Wah>Open</Wah>"), strings.Count(string(wire), "<Wah>Closed</Wah>"), song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect)
			}
			return
		}
	}
	t.Fatal("missing score")
}

func TestWahPublicStatesAndEdits(t *testing.T) {
	for _, test := range []struct {
		name  string
		edit  func(*gp.BeatEffects)
		want  gp.WahPedal
		codes []string
	}{
		{"unchanged", func(*gp.BeatEffects) {}, gp.WahPedalOpen, []string{}},
		{"typed closed", func(e *gp.BeatEffects) { e.WahPedal = gp.WahPedalClosed }, gp.WahPedalClosed, []string{}},
		{"typed clear", func(e *gp.BeatEffects) { e.WahPedal = gp.WahPedalNone }, gp.WahPedalNone, []string{}},
		{"legacy closed", func(e *gp.BeatEffects) { e.MixTableChange = &gp.MixTableChange{Wah: &gp.WahEffect{Value: 100}} }, gp.WahPedalClosed, []string{"gp8.omit.wah-display"}},
		{"conflict", func(e *gp.BeatEffects) {
			e.WahPedal = gp.WahPedalClosed
			e.MixTableChange = &gp.MixTableChange{Wah: &gp.WahEffect{Value: 0}}
		}, gp.WahPedalClosed, []string{"gp8.normalize.wah-authority", "gp8.omit.wah-display"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := notationPublicSong(t)
			beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
			beat.Effect.WahPedal = gp.WahPedalOpen
			data, e := gp.Export(song, gp.ExportFormatGP8)
			if e != nil {
				t.Fatal(e)
			}
			song, e = gp.Parse(data)
			if e != nil {
				t.Fatal(e)
			}
			song.Version = gp.Version{}
			for i := range song.Tracks {
				song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
			}
			beat = &song.Tracks[0].Measures[0].Voices[0].Beats[0]
			test.edit(&beat.Effect)
			before := beat.Effect
			for mask := 0; mask < (1 << len(test.codes)); mask++ {
				allowed := []string{"unrelated"}
				for i, code := range test.codes {
					if mask&(1<<i) != 0 {
						allowed = append(allowed, code)
					}
				}
				output, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				codes := []string{}
				for _, entry := range report.Entries {
					codes = append(codes, entry.Code)
					if entry.Location.Beat != 0 {
						t.Fatal(entry)
					}
				}
				slices.Sort(codes)
				want := slices.Clone(test.codes)
				slices.Sort(want)
				if !slices.Equal(codes, want) {
					t.Fatal(report)
				}
				if mask != (1<<len(test.codes))-1 {
					if err == nil || len(output) != 0 {
						t.Fatal("strict accepted")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				parsed, err := gp.Parse(output)
				if err != nil {
					t.Fatal(err)
				}
				if parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.WahPedal != test.want || !reflect.DeepEqual(before, beat.Effect) {
					t.Fatal("edit or read-only contract failed")
				}
			}
		})
	}
}

func TestWahLegacyValuesAndPolicy(t *testing.T) {
	for _, value := range []int8{-128, -2, -1, 0, 1, 99, 100, 101, 127} {
		for _, display := range []bool{false, true} {
			song := notationPublicSong(t)
			b := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
			b.Effect.MixTableChange = &gp.MixTableChange{Wah: &gp.WahEffect{Value: value, Display: display}}
			want := gp.WahPedalNone
			codes := []string{}
			if value >= 100 {
				want = gp.WahPedalClosed
			} else if value >= 0 {
				want = gp.WahPedalOpen
			}
			if value < -1 {
				codes = append(codes, "gp8.omit.wah-legacy-state")
			} else if value != -1 && value != 0 && value != 100 {
				codes = append(codes, "gp8.normalize.wah-legacy-value")
			}
			if value != -1 || display {
				codes = append(codes, "gp8.omit.wah-display")
			}
			slices.Sort(codes)
			for mask := 0; mask < (1 << len(codes)); mask++ {
				allowed := []string{"unrelated"}
				for i, code := range codes {
					if mask&(1<<i) != 0 {
						allowed = append(allowed, code)
					}
				}
				data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				gotCodes := []string{}
				for _, entry := range report.Entries {
					gotCodes = append(gotCodes, entry.Code)
				}
				slices.Sort(gotCodes)
				if !slices.Equal(gotCodes, codes) {
					t.Fatalf("value %d display %t: %#v", value, display, report)
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
				parsed, e := gp.Parse(data)
				if e != nil {
					t.Fatal(e)
				}
				if parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.WahPedal != want {
					t.Fatal("wrong state")
				}
				if b.Effect.MixTableChange.Wah.Value != value || b.Effect.MixTableChange.Wah.Display != display {
					t.Fatal("raw legacy mutated")
				}
			}
		}
	}
	song := notationPublicSong(t)
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.WahPedal = gp.WahPedal(255)
	if !slices.ContainsFunc(gp.ValidateSong(song), func(d gp.ScoreDiagnostic) bool { return d.Code == "score.beat.wah" }) {
		t.Fatal("invalid state undiagnosed")
	}
	if data, _, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{}); e == nil || len(data) != 0 {
		t.Fatal("invalid state exported")
	}
}

func TestWahCompleteBinaryBoundaries(t *testing.T) {
	source, e := os.ReadFile("../testdata/gp5/Wah.gp5")
	if e != nil {
		t.Fatal(e)
	}
	if source[1927] != 254 {
		t.Fatal("original final wah byte moved")
	}
	for _, value := range []int8{-128, -2, -1, 0, 1, 99, 100, 101, 127} {
		input := slices.Clone(source)
		input[1927] = byte(value)
		parsed, err := gp.ParseWithOptions(input, gp.ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		beat := parsed.Song.Tracks[0].Measures[1].Voices[0].Beats[0]
		want := gp.WahPedalNone
		if value >= 100 {
			want = gp.WahPedalClosed
		} else if value >= 0 {
			want = gp.WahPedalOpen
		}
		if beat.Effect.WahPedal != want || beat.Effect.MixTableChange.Wah.Value != value {
			t.Fatal("binary value changed")
		}
		found := slices.ContainsFunc(parsed.Diagnostics, func(d gp.ParseDiagnostic) bool {
			return d.Code == "Binary.Beat.Wah.Unsupported" && d.BinaryOffset != nil && *d.BinaryOffset == 1928
		})
		if found != (value < -1) {
			t.Fatalf("value %d diagnostics %#v", value, parsed.Diagnostics)
		}
		_, strictErr := gp.ParseWithOptions(input, gp.ParseOptions{Strict: true})
		if value < -1 {
			var rejection *gp.StrictParseError
			if !errors.As(strictErr, &rejection) || !slices.ContainsFunc(rejection.Diagnostics, func(d gp.ParseDiagnostic) bool { return d.Code == "Binary.Beat.Wah.Unsupported" }) {
				t.Fatal("strict did not reject legacy state")
			}
		}

	}
}

func TestWahImportedLegacyAuthority(t *testing.T) {
	for _, test := range []struct {
		name     string
		edit     func(*gp.BeatEffects)
		want     gp.WahPedal
		conflict bool
	}{
		{"raw value", func(e *gp.BeatEffects) { e.MixTableChange.Wah.Value = 100 }, gp.WahPedalClosed, false},
		{"raw clear", func(e *gp.BeatEffects) { e.MixTableChange.Wah = nil }, gp.WahPedalNone, false},
		{"container clear", func(e *gp.BeatEffects) { e.MixTableChange = nil }, gp.WahPedalNone, false},
		{"typed edit", func(e *gp.BeatEffects) { e.WahPedal = gp.WahPedalClosed }, gp.WahPedalClosed, false},
		{"typed clear", func(e *gp.BeatEffects) { e.WahPedal = gp.WahPedalNone }, gp.WahPedalNone, false},
		{"both incompatible", func(e *gp.BeatEffects) { e.WahPedal = gp.WahPedalClosed; e.MixTableChange.Wah.Value = -1 }, gp.WahPedalClosed, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, e := gp.ParseFile("../testdata/gp5/Wah.gp5")
			if e != nil {
				t.Fatal(e)
			}
			song := notationPublicSong(t)
			b := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
			b.Effect = source.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			raw := *b.Effect.MixTableChange.Wah
			b.Effect.MixTableChange = &gp.MixTableChange{Wah: &raw}
			test.edit(&b.Effect)
			data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if e != nil {
				t.Fatal(e)
			}
			if slices.ContainsFunc(report.Entries, func(e gp.ExportReportEntry) bool { return e.Code == "gp8.normalize.wah-authority" }) != test.conflict {
				t.Fatal(report)
			}
			out, e := gp.Parse(data)
			if e != nil {
				t.Fatal(e)
			}
			if out.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.WahPedal != test.want {
				t.Fatal("wrong legacy edit authority")
			}
		})
	}
}
