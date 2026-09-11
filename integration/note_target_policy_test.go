// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func noteTargetPolicySong(t *testing.T) *gp.Song {
	t.Helper()
	s := dynamicPolicySong(t, 95, false)
	b := &s.Tracks[0].Measures[0].Voices[0].Beats[1]
	b.Duration = gp.Duration{Value: 8, Dotted: true, TupletEnters: 3, TupletTimes: 2}
	b.Notes = b.Notes[:1:1]
	b.Notes[0].Velocity = 95
	b.Notes[0].Effect.LetRing = true
	return s
}
func assertNoteTargetPolicy(t *testing.T, s *gp.Song, code string, disposition gp.ExportDisposition) *gp.Song {
	t.Helper()
	if e := gp.FinalizeSong(s); e != nil {
		t.Fatal(e)
	}
	if ds := gp.ValidateSong(s); len(ds) != 0 {
		t.Fatalf("invalid public score %#v", ds)
	}
	before, e := json.Marshal(s)
	if e != nil {
		t.Fatal(e)
	}
	pre := gp.PreflightExport(s, gp.ExportFormatGP8, gp.ExportOptions{})
	want := 0
	if code != "" {
		want = 1
	}
	if len(pre.Entries) != want {
		t.Fatalf("report %#v", pre)
	}
	if code != "" {
		entry := pre.Entries[0]
		if entry.Code != code || entry.Disposition != disposition || entry.Location != (gp.ScoreLocation{Beat: 1}) {
			t.Fatalf("report owner %#v", entry)
		}
	}
	var parsed *gp.Song
	for _, allow := range [][]string{nil, {"gp8.omit.writer"}, {code}} {
		opts := gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allow}}
		data, report, err := gp.ExportWithReport(s, gp.ExportFormatGP8, opts)
		if !reflect.DeepEqual(pre, report) {
			t.Fatal("preflight and export disagree")
		}
		if code != "" && (len(allow) == 0 || allow[0] != code) {
			var loss *gp.ExportLossError
			if len(data) != 0 || !errors.As(err, &loss) || len(loss.Entries) != 1 {
				t.Fatalf("unallowed loss %d bytes %v", len(data), err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		parsed, err = gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
	}
	after, e := json.Marshal(s)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("policy mutated public score: %v", e)
	}
	return parsed
}
