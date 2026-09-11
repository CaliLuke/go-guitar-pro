// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"encoding/xml"
	"reflect"
	"slices"
	"testing"
)

func hammerSong(t *testing.T, shape string) (*Song, []string) {
	t.Helper()
	s := semanticExportProbeSong(t)
	beat := s.Tracks[0].Measures[0].Voices[0].Beats[0]
	note := beat.Notes[0]
	note.String = 2
	note.Value = 3
	note.Effect = defaultNoteEffect()
	first, second := beat, beat
	first.Notes = []Note{note}
	second.Notes = []Note{note}
	first.Notes[0].Effect.Hammer = true
	second.Notes[0].Effect.HammerDestination = true
	codes := []string{}
	switch shape {
	case "complete":
	case "destination":
		first.Notes[0].Effect.Hammer = false
		codes = []string{"gp8.omit.hammer-destination-consumer"}
	case "origin":
		second.Notes = nil
		second.Status = BeatStatusRest
		codes = []string{"gp8.omit.hammer-origin-consumer"}
	case "inferred":
		second.Notes[0].Effect.HammerDestination = false
	case "tap":
		second.Notes[0].String = 1
		second.Notes[0].Effect.LeftHandTapped = true
	case "wrong-string":
		second.Notes[0].String = 1
		codes = []string{"gp8.omit.hammer-origin-consumer", "gp8.omit.hammer-destination-consumer"}
	default:
		t.Fatalf("unknown shape %s", shape)
	}
	s.Tracks[0].Measures[0].Voices[0].Beats = []Beat{first, second}
	if e := FinalizeSong(s); e != nil {
		t.Fatal(e)
	}
	if d := ValidateSong(s); len(d) != 0 {
		t.Fatal(d)
	}
	return s, codes
}
func TestConformanceHammerEndpoints(t *testing.T) {
	runConformanceHammerEndpoints(newConformanceRun(t))
}
func runConformanceHammerEndpoints(run *conformanceRun) {
	t := run.t
	for _, shape := range []string{"complete", "destination", "origin", "inferred", "tap", "wrong-string"} {
		s, codes := hammerSong(t, shape)
		data, report := assertConsumerLossPolicy(t, s, codes)
		run.Report("M11-HAMMER-ENDPOINTS", reportCodes(report), codes)
		p, e := Parse(data)
		if e != nil {
			t.Fatal(e)
		}
		source := s.Tracks[0].Measures[0].Voices[0].Beats
		got := p.Tracks[0].Measures[0].Voices[0].Beats
		for b := range source {
			for n, note := range source[b].Notes {
				run.Field("NoteEffect.Hammer", got[b].Notes[n].Effect.Hammer, note.Effect.Hammer)
				run.Preserved("NoteEffect.HammerDestination", got[b].Notes[n].Effect.HammerDestination, note.Effect.HammerDestination)
			}
		}
		doc := conformanceWireDocument(t, data)
		count := 0
		for _, note := range doc.Notes.Notes {
			for _, property := range note.Properties.Properties {
				if property.Name == "HopoDestination" {
					count++
					run.Wire("gpifProperty.Enable", property.Enable != nil, true)
				}
			}
		}
		want := 0
		if shape != "origin" && shape != "inferred" {
			want = 1
		}
		run.Wire("gpifProperty.Name", count, want)
	}
	enable := "false"
	for _, payload := range []*string{nil, &enable} {
		n, e := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: "HopoDestination", Enable: payload}}}}, 6, false)
		if e != nil {
			t.Fatal(e)
		}
		run.Dispatch("gpifNoteToNote:p.Name", n.Effect.HammerDestination, payload != nil)
		ctx := &parseContext{}
		gpifAuditNoteProperty(ctx, "authored", "/GPIF/Notes/Note", gpifProperty{Name: "HopoDestination", Enable: payload})
		run.Dispatch("gpifAuditNoteProperty:property.Name", len(ctx.diagnostics), map[bool]int{true: 0, false: 1}[payload != nil])
	}
}

type hammerRef struct{ Track, Staff, Bar, Voice, Beat, Note int }
type hammerFact struct {
	hammerRef
	Origin, Destination bool
	From, To            *hammerRef
}

func TestAlphaTabHammerEndpoints(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, shape := range []string{"complete", "destination", "origin", "inferred", "tap", "wrong-string"} {
		s, codes := hammerSong(t, shape)
		data, _ := assertConsumerLossPolicy(t, s, codes)
		var facts []hammerFact
		readAlphaTabOracleFacts(t, "--hammer-facts", writeConformanceFixture(t, data), &facts)
		linked := shape == "complete" || shape == "inferred" || shape == "tap"
		if len(facts) < 1 || facts[0].Origin != linked {
			t.Fatalf("%s consumer %#v", shape, facts)
		}
		if shape != "origin" && (len(facts) != 2 || facts[1].Destination != linked) {
			t.Fatalf("%s destination %#v", shape, facts)
		}
		if linked && (!reflect.DeepEqual(facts[0].To, &hammerRef{Beat: 1}) || !reflect.DeepEqual(facts[1].From, &hammerRef{})) {
			t.Fatalf("%s wrong link %#v", shape, facts)
		}
	}
}
func TestHammerSourceMalformedAndOccurrence(t *testing.T) {
	s, _ := hammerSong(t, "destination")
	data, e := Export(s, ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	doc := conformanceWireDocument(t, data)
	for i := range doc.Notes.Notes {
		for j := range doc.Notes.Notes[i].Properties.Properties {
			p := &doc.Notes.Notes[i].Properties.Properties[j]
			if p.Name == "HopoDestination" {
				p.Enable = nil
			}
		}
	}
	raw, e := xml.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	result, e := ParseWithOptions(conformanceGPIFArchive(t, string(raw)), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData}})
	if e == nil || !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
		return d.Code == "GPIF.Note.Property.HopoDestination.MissingPayload" && d.ObjectID == "1"
	}) {
		t.Fatal("missing destination payload not diagnosed", e, result.Diagnostics)
	}
	doc = conformanceWireDocument(t, data)
	doc.Beats.Beats[0].Notes = "not-a-note"
	raw, e = xml.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	result, e = ParseWithOptions(conformanceGPIFArchive(t, string(raw)), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData}})
	if e == nil || !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == "GPIF.Beat.Notes.Reference" && d.Location.BeatID == "0" }) {
		t.Fatal("broken note reference not diagnosed", e, result.Diagnostics)
	}
}

func TestAlphaTabHammerSearchBoundaries(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range []struct {
		distance int
		late     bool
	}{{1, false}, {1, true}, {2, false}, {3, false}, {4, false}} {
		distance := test.distance
		linked := distance == 1 && !test.late
		s, _ := hammerSong(t, "complete")
		endpoints := s.Tracks[0].Measures[0].Voices[0].Beats
		s.MeasureHeaders = make([]MeasureHeader, distance+1)
		s.Tracks[0].Measures = make([]Measure, distance+1)
		for i := 0; i <= distance; i++ {
			s.MeasureHeaders[i] = defaultMeasureHeader()
			beat := defaultBeat()
			beat.Status = BeatStatusRest
			switch i {
			case 0:
				beat = endpoints[0]
			case distance:
				beat = endpoints[1]
			}
			s.Tracks[0].Measures[i] = Measure{HeaderIndex: i, Voices: []Voice{{Beats: []Beat{beat}}}}
			if i == distance && test.late {
				rest := defaultBeat()
				rest.Status = BeatStatusRest
				s.Tracks[0].Measures[i].Voices[0].Beats = []Beat{rest, beat}
			}
		}
		if e := FinalizeSong(s); e != nil {
			t.Fatal(e)
		}
		codes := []string{}
		if !linked {
			codes = []string{"gp8.omit.hammer-origin-consumer", "gp8.omit.hammer-destination-consumer"}
		}
		data, _ := assertConsumerLossPolicy(t, s, codes)
		var facts []hammerFact
		readAlphaTabOracleFacts(t, "--hammer-facts", writeConformanceFixture(t, data), &facts)
		if len(facts) != 2 || facts[0].Origin != linked || facts[1].Destination != linked {
			t.Fatalf("distance%d facts%#v", distance, facts)
		}
	}
	// A nearer ordinary note blocks a farther left-hand tap in that direction.
	s, _ := hammerSong(t, "tap")
	beats := s.Tracks[0].Measures[0].Voices[0].Beats
	beats[0].Notes[0].String = 3
	blocked := beats[1].Notes[0]
	blocked.String = 2
	blocked.Effect.HammerDestination = false
	blocked.Effect.LeftHandTapped = false
	beats[1].Notes = append(beats[1].Notes, blocked)
	data, _ := assertConsumerLossPolicy(t, s, []string{"gp8.omit.hammer-origin-consumer", "gp8.omit.hammer-destination-consumer"})
	var facts []hammerFact
	readAlphaTabOracleFacts(t, "--hammer-facts", writeConformanceFixture(t, data), &facts)
	if len(facts) != 3 || facts[0].Origin || facts[1].Destination || facts[2].Destination {
		t.Fatal("tap blocker", facts)
	}
}
func TestHammerGraceAndOccurrenceIsolation(t *testing.T) {
	s, _ := hammerSong(t, "destination")
	beats := s.Tracks[0].Measures[0].Voices[0].Beats
	beats[0].Notes[0].Effect.HammerDestination = true
	beats[1].Notes[0].Effect.HammerDestination = false
	data, e := Export(s, ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	doc := conformanceWireDocument(t, data)
	doc.Beats.Beats[0].GraceNotes = "BeforeBeat"
	raw, e := xml.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	result, e := ParseWithOptions(conformanceGPIFArchive(t, string(raw)), ParseOptions{})
	if e != nil || !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
		return d.Code == "GPIF.Note.HammerDestination.Grace" && d.Location.NoteID == "0"
	}) {
		t.Fatal("grace loss missing", e, result.Diagnostics)
	}
	doc = conformanceWireDocument(t, data)
	doc.Beats.Beats[1].Notes = doc.Beats.Beats[0].Notes
	raw, e = xml.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	p, e := Parse(conformanceGPIFArchive(t, string(raw)))
	if e != nil {
		t.Fatal(e)
	}
	notes := p.Tracks[0].Measures[0].Voices[0].Beats
	notes[0].Notes[0].Effect.HammerDestination = false
	if !notes[1].Notes[0].Effect.HammerDestination {
		t.Fatal("reused definition marker alias")
	}
}
func TestAlphaTabHammerGraceIncoming(t *testing.T) {
	requireAlphaTabConformance(t)
	s, _ := hammerSong(t, "destination")
	owner := s.Tracks[0].Measures[0].Voices[0].Beats[1]
	owner.Notes[0].Effect.Graces = []GraceEffect{{Fret: 2, Duration: 32, Velocity: Forte, Transition: GraceEffectTransitionHammer}}
	s.Tracks[0].Measures[0].Voices[0].Beats = []Beat{owner}
	if e := FinalizeSong(s); e != nil {
		t.Fatal(e)
	}
	data, _ := assertConsumerLossPolicy(t, s, []string{})
	var facts []hammerFact
	readAlphaTabOracleFacts(t, "--hammer-facts", writeConformanceFixture(t, data), &facts)
	if len(facts) != 2 || !facts[0].Origin || !facts[1].Destination {
		t.Fatal("grace origin must satisfy owner destination", facts)
	}
}

func TestAlphaTabHammerVoiceIsolation(t *testing.T) {
	requireAlphaTabConformance(t)
	s, _ := hammerSong(t, "complete")
	beats := s.Tracks[0].Measures[0].Voices[0].Beats
	s.Tracks[0].Measures[0].Voices = []Voice{{Beats: []Beat{beats[0]}}, {Beats: []Beat{beats[1]}}}
	if e := FinalizeSong(s); e != nil {
		t.Fatal(e)
	}
	data, _ := assertConsumerLossPolicy(t, s, []string{"gp8.omit.hammer-origin-consumer", "gp8.omit.hammer-destination-consumer"})
	var facts []hammerFact
	readAlphaTabOracleFacts(t, "--hammer-facts", writeConformanceFixture(t, data), &facts)
	if len(facts) != 2 || facts[0].Origin || facts[1].Destination || facts[1].Voice != 1 {
		t.Fatal("crossed voice scope", facts)
	}
}
