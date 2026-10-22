// SPDX-License-Identifier: MIT

package integration_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestValidateSongRejectsOutOfRangeOpenStringTuning(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	song.Tracks[0].Strings[0].Value = -1

	diagnostics := gp.ValidateSong(song)
	if !slices.ContainsFunc(diagnostics, func(diagnostic gp.ScoreDiagnostic) bool {
		return diagnostic.Code == "score.staff.string-tuning" && diagnostic.Location.Track == 0 && diagnostic.Location.Staff == 0
	}) {
		t.Fatalf("validation = %#v, want a scoped tuning-range diagnostic", diagnostics)
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(entry gp.ExportReportEntry) bool {
		return entry.Code == "gp8.reject.score.staff.string-tuning"
	}) {
		t.Fatalf("invalid tuning export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
}

func TestParseRejectsInvalidOpenStringTuning(t *testing.T) {
	const validPitches = "40 45 50 55 59 64"
	const gpif = `<GPIF><GPVersion>8.1.5</GPVersion><Score><Title>Invalid tuning</Title></Score><MasterTrack><Tracks>0</Tracks></MasterTrack><Tracks><Track id="0"><Name>Guitar</Name><Properties><Property name="Tuning"><Pitches>` + validPitches + `</Pitches></Property></Properties><Staves><Staff><Properties><Property name="Tuning"><Pitches>` + validPitches + `</Pitches></Property></Properties></Staff></Staves></Track></Tracks><MasterBars><MasterBar><Time>4/4</Time><Bars>0</Bars></MasterBar></MasterBars><Bars><Bar id="0"><Clef>G2</Clef><Voices>0</Voices></Bar></Bars><Voices><Voice id="0"><Beats>0</Beats></Voice></Voices><Beats><Beat id="0"><Rhythm ref="0"/></Beat></Beats><Notes/><Rhythms><Rhythm id="0"><NoteValue>Whole</NoteValue></Rhythm></Rhythms></GPIF>`
	for _, invalid := range []string{"-1", "128", "forty", "40 nope 64", strings.TrimSpace(strings.Repeat("64 ", 128))} {
		t.Run(invalid, func(t *testing.T) {
			changed := strings.ReplaceAll(gpif, validPitches, invalid)
			if changed == gpif {
				t.Fatal("test tuning replacement missed")
			}
			parsed, err := gp.Parse(buildGP7Archive(t, changed, "", nil))
			if err == nil || parsed != nil {
				t.Fatalf("invalid tuning parse = %#v, %v; want an error and no partial score", parsed, err)
			}
		})
	}
}

func TestValidateSongRejectsUnrepresentableStaffStringCount(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	song.Tracks[0].Strings = make([]gp.GuitarString, 128)
	if !slices.ContainsFunc(gp.ValidateSong(song), func(diagnostic gp.ScoreDiagnostic) bool {
		return diagnostic.Code == "score.staff.string-count" && diagnostic.Location.Track == 0 && diagnostic.Location.Staff == 0
	}) {
		t.Fatalf("validation = %#v, want string-count range diagnostic", gp.ValidateSong(song))
	}
}

func TestValidateSongChecksSoundingPitchWithoutTransposition(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	song.Tracks[0].CapoFret = 64
	song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].Value = 3

	diagnostics := gp.ValidateSong(song)
	if !slices.ContainsFunc(diagnostics, func(diagnostic gp.ScoreDiagnostic) bool {
		return diagnostic.Code == "score.note.sounding-midi" && diagnostic.Location.Track == 0 && diagnostic.Location.Staff == 0
	}) {
		t.Fatalf("validation = %#v, want a scoped sounding-pitch diagnostic", diagnostics)
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(entry gp.ExportReportEntry) bool {
		return entry.Code == "gp8.reject.score.note.sounding-midi"
	}) {
		t.Fatalf("out-of-range sounding pitch export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
}

func TestValidateSongChecksGraceSoundingPitch(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	track := &song.Tracks[0]
	track.Strings[0].Value = 10
	track.Strings[1].Value = 20
	track.CapoFret = 70
	note := &track.Measures[0].Voices[0].Beats[1].Notes[0]
	note.Effect.Graces = []gp.GraceEffect{{Fret: 60, Duration: 32, Velocity: 47}}

	if !slices.ContainsFunc(gp.ValidateSong(song), func(diagnostic gp.ScoreDiagnostic) bool {
		return diagnostic.Code == "score.grace.sounding-midi" && diagnostic.Location.Track == 0 && diagnostic.Location.Staff == 0
	}) {
		t.Fatalf("validation = %#v, want a grace sounding-pitch diagnostic", gp.ValidateSong(song))
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(entry gp.ExportReportEntry) bool {
		return entry.Code == "gp8.reject.score.grace.sounding-midi"
	}) {
		t.Fatalf("out-of-range grace pitch export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
}

func TestValidateSongRejectsNegativeStringAndGraceFrets(t *testing.T) {
	t.Run("string fret", func(t *testing.T) {
		song := dynamicPolicySong(t, 47, false)
		song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].Value = -1
		if !slices.ContainsFunc(gp.ValidateSong(song), func(diagnostic gp.ScoreDiagnostic) bool {
			return diagnostic.Code == "score.note.fret"
		}) {
			t.Fatalf("validation = %#v, want negative-fret diagnostic", gp.ValidateSong(song))
		}
		data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(entry gp.ExportReportEntry) bool {
			return entry.Code == "gp8.reject.score.note.fret"
		}) {
			t.Fatalf("negative string fret export = %d bytes, %#v, %v", len(data), report.Entries, err)
		}
	})
	for _, test := range []struct {
		name  string
		grace gp.GraceEffect
	}{
		{name: "legacy", grace: gp.GraceEffect{Fret: -1, Duration: 32, Velocity: 47}},
		{name: "exact", grace: func() gp.GraceEffect {
			fret := gp.Fret(-1)
			return gp.GraceEffect{Fret: 0, ExactFret: &fret, Duration: 32, Velocity: 47}
		}()},
	} {
		t.Run("grace "+test.name, func(t *testing.T) {
			song := publicGraceTimerSong(t)
			graces := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces
			(*graces)[0] = test.grace
			if !slices.ContainsFunc(gp.ValidateSong(song), func(diagnostic gp.ScoreDiagnostic) bool {
				return diagnostic.Code == "score.grace.fret"
			}) {
				t.Fatalf("validation = %#v, want negative-grace-fret diagnostic", gp.ValidateSong(song))
			}
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(entry gp.ExportReportEntry) bool {
				return entry.Code == "gp8.reject.score.grace.fret"
			}) {
				t.Fatalf("negative grace fret export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
		})
	}
}

func TestValidateSongAllowsLegacyDeadGraceSentinel(t *testing.T) {
	data, err := os.ReadFile("../testdata/gp4/fade-to-black.gp4")
	if err != nil {
		t.Fatal(err)
	}
	song, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	grace := song.Tracks[2].Measures[20].Voices[0].Beats[4].Notes[1].Effect.Graces[0]
	if grace.Fret != -1 || !grace.IsDead || grace.ExactFret != nil || grace.RawFret == nil || *grace.RawFret != -1 {
		t.Fatalf("fixture grace = %#v, want the legacy dead-grace sentinel", grace)
	}
	for _, diagnostic := range gp.ValidateSong(song) {
		if diagnostic.Location == (gp.ScoreLocation{Track: 2, Staff: 0, Measure: 20, Voice: 0, Beat: 4, Note: 1}) &&
			(diagnostic.Code == "score.grace.fret" || diagnostic.Code == "score.grace.sounding-midi") {
			t.Fatalf("dead-grace sentinel validation = %#v", diagnostic)
		}
	}
}

func TestGP8ExportsLegacyDeadGraceSentinel(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
	note.Effect.Graces = []gp.GraceEffect{{Fret: -1, IsDead: true, Duration: 32, Velocity: 47}}

	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err != nil || len(data) == 0 {
		t.Fatalf("dead-grace export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].Effect.Graces
	if len(got) != 1 || !got[0].IsDead {
		t.Fatalf("round-trip graces = %#v, want one dead grace", got)
	}
}

func TestValidateSongRejectsOtherNegativeDeadGraceFrets(t *testing.T) {
	negative := gp.Fret(-1)
	for _, test := range []struct {
		name  string
		grace gp.GraceEffect
	}{
		{name: "legacy minus two", grace: gp.GraceEffect{Fret: -2, IsDead: true}},
		{name: "negative exact fret", grace: gp.GraceEffect{Fret: 0, ExactFret: &negative, IsDead: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := publicGraceTimerSong(t)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[0] = test.grace
			if !slices.ContainsFunc(gp.ValidateSong(song), func(diagnostic gp.ScoreDiagnostic) bool {
				return diagnostic.Code == "score.grace.fret"
			}) {
				t.Fatalf("validation = %#v, want negative-grace-fret diagnostic", gp.ValidateSong(song))
			}
		})
	}
}

func TestGP8PreflightChecksCapoPitchAfterOmittedTransposition(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	track := &song.Tracks[0]
	track.Strings[0].Value = 118
	note := &track.Measures[0].Voices[0].Beats[1].Notes[0]
	note.Value = 1
	zeroFret := gp.Fret(0)
	note.Effect.Graces = []gp.GraceEffect{{Fret: 0, ExactFret: &zeroFret, Duration: 32, Velocity: 47}}
	staff := gp.Staff{
		Measures: track.Measures,
		Strings:  track.Strings,
		PartialCapo: &gp.PartialCapo{
			Offset:  10,
			Strings: []bool{true, false},
		},
		TranspositionPitch: 1,
	}
	track.Staves = []gp.Staff{staff}

	if diagnostics := gp.ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("source sounding pitch should remain valid after transposition: %#v", diagnostics)
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(entry gp.ExportReportEntry) bool {
		return entry.Code == "gp8.reject.score" && entry.Disposition == gp.ExportDispositionRejected
	}) {
		t.Fatalf("out-of-range exported pitch = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
}

func TestValidateSongChannelDiagnosticsHaveStableOrder(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	song.Channels[0].Volume = -1
	song.Channels[0].Balance = -1
	song.Channels[0].Chorus = -1
	song.Channels[0].Reverb = -1
	song.Channels[0].Phaser = -1
	song.Channels[0].Tremolo = -1
	want := []string{
		"score.channel.volume",
		"score.channel.balance",
		"score.channel.chorus",
		"score.channel.reverb",
		"score.channel.phaser",
		"score.channel.tremolo",
	}
	for range 10 {
		var got []string
		for _, diagnostic := range gp.ValidateSong(song) {
			if slices.Contains(want, diagnostic.Code) {
				got = append(got, diagnostic.Code)
			}
		}
		if !slices.Equal(got, want) {
			t.Fatalf("channel diagnostic order = %#v, want %#v", got, want)
		}
	}
}
