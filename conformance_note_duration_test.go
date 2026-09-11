// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"reflect"
	"testing"
)

type noteDurationTrillFact struct {
	Track, Staff, Bar, Voice, Beat, Note                   int
	Duration, Dots, Numerator, Denominator, Start, Length  int
	Fret, String                                           int
	Percent                                                float64
	TieOrigin, TieDestination, LetRing, PalmMute, Staccato bool
	TrillFret, TrillDuration                               int
}

func TestConformanceNoteDurationPercentLoss(t *testing.T) {
	runConformanceNoteDurationPercentLoss(newConformanceRun(t))
}
func runConformanceNoteDurationPercentLoss(run *conformanceRun) {
	t := run.t
	for _, value := range []float32{.5, .75, 1} {
		song := conformanceDurationPercentSong(t, value)
		source := song.Tracks[0].Measures[0].Voices[0].Beats
		run.Omitted("Note.DurationPercent", source[1].Notes[0].DurationPercent, value)
		codes := []string{}
		if value != 1 {
			codes = append(codes, "gp8.omit.note-duration-percent")
		}
		data, report := assertConsumerLossPolicy(t, song, codes)
		run.Report("M09-DURATION-PERCENT-LOSS", reportCodes(report), codes)
		if value != 1 && (report.Entries[0].Disposition != ExportDispositionOmitted || report.Entries[0].Location != (ScoreLocation{Beat: 1})) {
			t.Fatalf("loss owner %#v", report)
		}
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		beats := parsed.Tracks[0].Measures[0].Voices[0].Beats
		for i := 1; i <= 2; i++ {
			note := beats[i].Notes[0]
			run.Field("Note.DurationPercent", note.DurationPercent, float32(1))
			run.Preserved("Beat.Duration", beats[i].Duration, source[i].Duration)
			run.Derived("Beat.ExactStart", beats[i].ExactStart, source[i].ExactStart)
			run.Preserved("Note.Kind", note.Kind, source[i].Notes[0].Kind)
			run.Preserved("Note.TieOrigin", note.TieOrigin, source[i].Notes[0].TieOrigin)
			run.Preserved("NoteEffect.LetRing", note.Effect.LetRing, true)
		}
		assertNoteTargetRhythmWire(run, data)
	}
}
func conformanceNoteTargetSong(t *testing.T) *Song {
	t.Helper()
	song := semanticExportProbeSong(t)
	beat := song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Duration = Duration{Value: 8, Dotted: true, TupletEnters: 3, TupletTimes: 2}
	beat.Notes[0].Effect.LetRing = true
	song.Tracks[0].Measures[0].Voices[0].Beats = []Beat{{Duration: defaultDuration(), Status: BeatStatusRest, Dynamics: Forte}, beat}
	return song
}
func conformanceDurationPercentSong(t *testing.T, value float32) *Song {
	t.Helper()
	song := conformanceNoteTargetSong(t)
	voice := &song.Tracks[0].Measures[0].Voices[0]
	voice.Beats[1].Notes[0].DurationPercent = value
	voice.Beats[1].Notes[0].TieOrigin = true
	voice.Beats = append(voice.Beats, Beat{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{{Value: 3, String: 1, Kind: NoteTypeTie, Velocity: Forte, DurationPercent: 1, Effect: NoteEffect{LetRing: true}}}})
	finalizeNoteTargetSong(t, song)
	return song
}
func finalizeNoteTargetSong(t *testing.T, song *Song) {
	t.Helper()
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if ds := ValidateSong(song); len(ds) != 0 {
		t.Fatalf("invalid target-limit input %#v", ds)
	}
}
func assertNoteTargetRhythmWire(run *conformanceRun, data []byte) {
	t := run.t
	var doc struct {
		Beats []struct {
			Rhythm struct {
				Ref string `xml:"ref,attr"`
			}
		} `xml:"Beats>Beat"`
		Rhythms []struct {
			ID        string `xml:"id,attr"`
			NoteValue string
			Dot       *struct {
				Count int `xml:"count,attr"`
			} `xml:"AugmentationDot"`
			Tuplet *struct {
				Num int `xml:"num,attr"`
				Den int `xml:"den,attr"`
			} `xml:"PrimaryTuplet"`
		} `xml:"Rhythms>Rhythm"`
	}
	if err := xml.Unmarshal(conformanceBarreGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	ref := doc.Beats[1].Rhythm.Ref
	for _, rhythm := range doc.Rhythms {
		if rhythm.ID != ref {
			continue
		}
		run.Wire("gpifRhythm.NoteValue", rhythm.NoteValue, "Eighth")
		if rhythm.Dot == nil || rhythm.Tuplet == nil {
			t.Fatal("lost dotted tuplet rhythm")
		}
		run.Wire("gpifAugDot.Count", rhythm.Dot.Count, 1)
		run.Wire("gpifTuplet.Num", rhythm.Tuplet.Num, 3)
		run.Wire("gpifTuplet.Den", rhythm.Tuplet.Den, 2)
		return
	}
	t.Fatal("missing target rhythm")
}
func TestAlphaTabNoteDurationPercentLoss(t *testing.T) {
	for _, value := range []float32{.5, .75, 1} {
		song := conformanceDurationPercentSong(t, value)
		data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var facts []noteDurationTrillFact
		readAlphaTabOracleFacts(t, "--note-duration-trill", writeConformanceFixture(t, data), &facts)
		want := []noteDurationTrillFact{
			{Beat: 1, Duration: 8, Dots: 1, Numerator: 3, Denominator: 2, Start: 960, Length: 480, Fret: 3, String: 6, Percent: 1, TieOrigin: true, LetRing: true, TrillFret: -1, TrillDuration: 32},
			{Beat: 2, Duration: 4, Numerator: -1, Denominator: -1, Start: 1440, Length: 960, Fret: 3, String: 6, Percent: 1, TieDestination: true, LetRing: true, TrillFret: -1, TrillDuration: 32},
		}
		if !reflect.DeepEqual(facts, want) {
			t.Fatalf("fraction %v raw GPIF consumer %#v want %#v", value, facts, want)
		}
	}
}
