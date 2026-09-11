// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"testing"
)

func TestConformanceTrillSixteenth(t *testing.T) { runConformanceTrillSixteenth(newConformanceRun(t)) }
func TestConformanceTrillThirtySecond(t *testing.T) {
	runConformanceTrillThirtySecond(newConformanceRun(t))
}
func TestConformanceTrillSixtyFourth(t *testing.T) {
	runConformanceTrillSixtyFourth(newConformanceRun(t))
}
func runConformanceTrillSixteenth(run *conformanceRun)    { runConformanceTrillSpeed(run, 16) }
func runConformanceTrillThirtySecond(run *conformanceRun) { runConformanceTrillSpeed(run, 32) }
func runConformanceTrillSixtyFourth(run *conformanceRun)  { runConformanceTrillSpeed(run, 64) }
func runConformanceTrillSpeed(run *conformanceRun, speed uint16) {
	t := run.t
	song := conformanceTrillSpeedSong(t, speed)
	authored := song.Tracks[0].Measures[0].Voices[0].Beats[1]
	run.Normalized("TrillEffect.Duration", authored.Notes[0].Effect.Trill.Duration, Duration{Value: speed, TupletEnters: 1, TupletTimes: 1})
	codes := []string{}
	if speed != 16 {
		codes = append(codes, "gp8.normalize.trill-duration")
	}
	data, report := assertConsumerLossPolicy(t, song, codes)
	run.Report(fmt.Sprintf("M11-TRILL-SPEED-%d", speed), reportCodes(report), codes)
	if speed != 16 && (report.Entries[0].Disposition != ExportDispositionNormalized || report.Entries[0].Location != (ScoreLocation{Beat: 1})) {
		t.Fatalf("trill loss owner %#v", report)
	}
	wire := extractGPIFLeafText(t, data)
	run.Wire("gpifNote.Trill", wire["GPIF/Notes/Note/Trill"], "7")
	run.Wire("gpifTrill.Fret", wire["GPIF/Notes/Note/Trill"], "7")
	assertNoteTargetRhythmWire(run, data)
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	beat := parsed.Tracks[0].Measures[0].Voices[0].Beats[1]
	note := beat.Notes[0]
	run.Preserved("TrillEffect.Fret", note.Effect.Trill.Fret, int8(7))
	run.Field("TrillEffect.Duration", note.Effect.Trill.Duration, Duration{Value: 16, TupletEnters: 1, TupletTimes: 1})
	run.Preserved("Beat.Duration", beat.Duration, authored.Duration)
	run.Preserved("Note.Value", note.Value, int16(3))
	run.Preserved("NoteEffect.LetRing", note.Effect.LetRing, true)
	run.Preserved("NoteEffect.PalmMute", note.Effect.PalmMute, true)
	run.Preserved("NoteEffect.Staccato", note.Effect.Staccato, true)
}
func conformanceTrillSpeedSong(t *testing.T, speed uint16) *Song {
	t.Helper()
	song := conformanceNoteTargetSong(t)
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
	note.Effect.Trill = &TrillEffect{Fret: 7, Duration: Duration{Value: speed, TupletEnters: 1, TupletTimes: 1}}
	note.Effect.PalmMute = true
	note.Effect.Staccato = true
	finalizeNoteTargetSong(t, song)
	return song
}
func TestAlphaTabTrillSpeeds(t *testing.T) {
	for _, speed := range []uint16{16, 32, 64} {
		data, _, err := ExportWithReport(conformanceTrillSpeedSong(t, speed), ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var facts []noteDurationTrillFact
		readAlphaTabOracleFacts(t, "--note-duration-trill", writeConformanceFixture(t, data), &facts)
		want := []noteDurationTrillFact{{Beat: 1, Duration: 8, Dots: 1, Numerator: 3, Denominator: 2, Start: 960, Length: 480, Fret: 3, String: 6, Percent: 1, LetRing: true, PalmMute: true, Staccato: true, TrillFret: 7, TrillDuration: 16}}
		if !reflect.DeepEqual(facts, want) {
			t.Fatalf("speed%d raw consumer %#v want %#v", speed, facts, want)
		}
	}
}
