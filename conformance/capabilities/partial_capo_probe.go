//go:build ignore

// SPDX-License-Identifier: MIT

// This acquisition probe records public source fields and diagnostics.
// LegacyModelMIDI intentionally omits partial capo for comparison with the baseline.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	gp "github.com/CaliLuke/go-guitar-pro"
)

type Note struct {
	Track, Staff, Measure, Voice, Beat, Note int
	String                                   int8
	Fret                                     int16
	LegacyModelMIDI                          int
	Mode                                     gp.NoteAccidentalMode
}
type Staff struct {
	Track, Staff int
	Capo         int32
	Tuning       []gp.GuitarString
	PartialCapo  any
}
type Result struct {
	File         string
	Error        string
	StrictError  string
	Diagnostics  []gp.ParseDiagnostic
	Staves       []Staff
	Notes        []Note
	ExportError  string
	ExportReport gp.ExportReport
}

func main() {
	var results []Result
	for _, p := range os.Args[1:] {
		out := Result{File: p}
		data, e := os.ReadFile(p)
		if e != nil {
			panic(e)
		}
		r, e := gp.ParseWithOptions(data, gp.ParseOptions{})
		if e != nil {
			out.Error = e.Error()
		}
		_, se := gp.ParseWithOptions(data, gp.ParseOptions{Strict: true, StrictKinds: []gp.ParseDiagnosticKind{gp.ParseDiagnosticInvalidData}})
		if se != nil {
			out.StrictError = se.Error()
		}
		if r != nil {
			out.Diagnostics = r.Diagnostics
			for ti, tr := range r.Song.Tracks {
				for si, st := range tr.Staves {
					raw, _ := json.Marshal(st)
					var fields map[string]any
					_ = json.Unmarshal(raw, &fields)
					out.Staves = append(out.Staves, Staff{ti, si, st.CapoFret, st.Strings, fields["PartialCapo"]})
					for mi, m := range st.Measures {
						for vi, v := range m.Voices {
							for bi, b := range v.Beats {
								for ni, n := range b.Notes {
									midi := int(n.Value)
									if n.String > 0 && int(n.String) <= len(st.Strings) {
										midi += int(st.Strings[n.String-1].Value) + int(st.CapoFret) - int(st.TranspositionPitch)
									}
									out.Notes = append(out.Notes, Note{ti, si, mi, vi, bi, ni, n.String, n.Value, midi, n.AccidentalMode})
								}
							}
						}
					}
				}
			}
			_, report, e := gp.ExportWithReport(r.Song, gp.ExportFormatGP8, gp.ExportOptions{})
			out.ExportReport = report
			if e != nil {
				out.ExportError = e.Error()
			}
		}
		results = append(results, out)
	}
	b, e := json.MarshalIndent(results, "", "  ")
	if e != nil {
		panic(e)
	}
	fmt.Println(string(b))
}
