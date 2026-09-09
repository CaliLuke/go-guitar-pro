// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func TestSemanticMatrixM02PageSetupImport(t *testing.T) {
	runSemanticMatrixM02PageSetupImport(newSemanticMatrixRun(t))
}

func runSemanticMatrixM02PageSetupImport(run *semanticMatrixRun) {
	song := parseTestFixture(run.t, "testdata/gp5/score-info.gp5")
	page := song.PageSetup
	run.Field("Song.PageSetup", song.PageSetup, page)
	run.Field("PageSetup.Words", page.Words, "Words by %WORDS%")
	run.Field("PageSetup.Music", page.Music, "Music by %MUSIC%")
	run.Field("PageSetup.PageNumber", page.PageNumber, "Page %N%/%P%")
	run.Field("PageSetup.Copyright", page.Copyright, "Copyright %COPYRIGHT%\nAll Rights Reserved - International Copyright Secured")
	run.Field("PageSetup.WordAndMusic", page.WordAndMusic, "Words & Music by %WORDSMUSIC%")
	run.Field("PageSetup.Artist", page.Artist, "%ARTIST%")
	run.Field("PageSetup.Album", page.Album, "%ALBUM%")
	run.Field("PageSetup.Title", page.Title, "%TITLE%")
	run.Field("PageSetup.Subtitle", page.Subtitle, "%SUBTITLE%")
	run.Field("PageSetup.PageHeight", page.PageHeight, int32(297))
	run.Field("PageSetup.ScoreSizeProportion", page.ScoreSizeProportion, float32(1))
	run.Field("PageSetup.PageWidth", page.PageWidth, int32(210))
	run.Field("PageSetup.MarginBottom", page.MarginBottom, int32(10))
	run.Field("PageSetup.MarginTop", page.MarginTop, int32(15))
	run.Field("PageSetup.MarginRight", page.MarginRight, int32(10))
	run.Field("PageSetup.MarginLeft", page.MarginLeft, int32(10))
	run.Field("PageSetup.HeaderAndFooter", page.HeaderAndFooter, uint16(0x1ff))
}

func TestSemanticMatrixM02DisplayExportPolicy(t *testing.T) {
	runSemanticMatrixM02DisplayExportPolicy(newSemanticMatrixRun(t))
}

func runSemanticMatrixM02DisplayExportPolicy(run *semanticMatrixRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.PageSetup = PageSetup{
		Words: "Words template", Music: "Music template", PageNumber: "Page %N%/%P%",
		Copyright: "Copyright template", WordAndMusic: "Both template", Artist: "Artist template",
		Album: "Album template", Title: "Title template", Subtitle: "Subtitle template",
		PageHeight: 1200, ScoreSizeProportion: 1.25, PageWidth: 900,
		MarginBottom: 41, MarginTop: 42, MarginRight: 43, MarginLeft: 44,
		HeaderAndFooter: 0x155,
	}
	marker := song.MeasureHeaders[0].Marker
	marker.Title = "Section & 名"
	marker.Color = 0x123456
	track := &song.Tracks[0]
	track.Visible = false
	track.IndicateTuning = true
	track.Settings = TrackSettings{
		Tablature: true, Notation: true, DiagramAreBelow: true, ShowRhythm: true,
		ForceHorizontal: true, ForceChannels: true, DiagramList: true, DiagramInScore: true,
		AutoLetRing: true, AutoBrush: true, ExtendRhythmic: true,
	}
	measure := &track.Staves[0].Measures[0]
	measure.LineBreak = LineBreakProtect
	measure.HasDoubleBar = true
	voice := &measure.Voices[0]
	voice.Direction = VoiceDirectionDown
	beat := &voice.Beats[0]
	beat.Display = BeatDisplay{
		BreakBeam: true, ForceBeam: true, BeamDirection: VoiceDirectionUp,
		TupletBracket: TupletBracketEnd, BreakSecondary: 3,
		BreakSecondaryTuplet: true, ForceBracket: true,
	}

	run.Field("Song.PageSetup", song.PageSetup, song.PageSetup)
	run.Field("PageSetup.Words", song.PageSetup.Words, "Words template")
	run.Field("PageSetup.Music", song.PageSetup.Music, "Music template")
	run.Field("PageSetup.PageNumber", song.PageSetup.PageNumber, "Page %N%/%P%")
	run.Field("PageSetup.Copyright", song.PageSetup.Copyright, "Copyright template")
	run.Field("PageSetup.WordAndMusic", song.PageSetup.WordAndMusic, "Both template")
	run.Field("PageSetup.Artist", song.PageSetup.Artist, "Artist template")
	run.Field("PageSetup.Album", song.PageSetup.Album, "Album template")
	run.Field("PageSetup.Title", song.PageSetup.Title, "Title template")
	run.Field("PageSetup.Subtitle", song.PageSetup.Subtitle, "Subtitle template")
	run.Field("PageSetup.PageHeight", song.PageSetup.PageHeight, int32(1200))
	run.Field("PageSetup.ScoreSizeProportion", song.PageSetup.ScoreSizeProportion, float32(1.25))
	run.Field("PageSetup.PageWidth", song.PageSetup.PageWidth, int32(900))
	run.Field("PageSetup.MarginBottom", song.PageSetup.MarginBottom, int32(41))
	run.Field("PageSetup.MarginTop", song.PageSetup.MarginTop, int32(42))
	run.Field("PageSetup.MarginRight", song.PageSetup.MarginRight, int32(43))
	run.Field("PageSetup.MarginLeft", song.PageSetup.MarginLeft, int32(44))
	run.Field("PageSetup.HeaderAndFooter", song.PageSetup.HeaderAndFooter, uint16(0x155))
	run.Field("MeasureHeader.Marker", song.MeasureHeaders[0].Marker, marker)
	run.Field("Marker.Title", marker.Title, "Section & 名")
	run.Field("Marker.Color", marker.Color, int32(0x123456))
	run.Field("Track.Visible", track.Visible, false)
	run.Field("Track.Color", track.Color, int32(0x336699))
	run.Field("Track.IndicateTuning", track.IndicateTuning, true)
	run.Field("Track.Settings", track.Settings, track.Settings)
	run.Field("TrackSettings.Tablature", track.Settings.Tablature, true)
	run.Field("TrackSettings.Notation", track.Settings.Notation, true)
	run.Field("TrackSettings.DiagramAreBelow", track.Settings.DiagramAreBelow, true)
	run.Field("TrackSettings.ShowRhythm", track.Settings.ShowRhythm, true)
	run.Field("TrackSettings.ForceHorizontal", track.Settings.ForceHorizontal, true)
	run.Field("TrackSettings.ForceChannels", track.Settings.ForceChannels, true)
	run.Field("TrackSettings.DiagramList", track.Settings.DiagramList, true)
	run.Field("TrackSettings.DiagramInScore", track.Settings.DiagramInScore, true)
	run.Field("TrackSettings.AutoLetRing", track.Settings.AutoLetRing, true)
	run.Field("TrackSettings.AutoBrush", track.Settings.AutoBrush, true)
	run.Field("TrackSettings.ExtendRhythmic", track.Settings.ExtendRhythmic, true)
	run.Field("Measure.LineBreak", measure.LineBreak, LineBreakProtect)
	run.Field("Measure.HasDoubleBar", measure.HasDoubleBar, true)
	run.Field("Voice.Direction", voice.Direction, VoiceDirectionDown)
	run.Field("Beat.Display", beat.Display, beat.Display)
	run.Field("BeatDisplay.BreakBeam", beat.Display.BreakBeam, true)
	run.Field("BeatDisplay.ForceBeam", beat.Display.ForceBeam, true)
	run.Field("BeatDisplay.BeamDirection", beat.Display.BeamDirection, VoiceDirectionUp)
	run.Field("BeatDisplay.TupletBracket", beat.Display.TupletBracket, TupletBracketEnd)
	run.Field("BeatDisplay.BreakSecondary", beat.Display.BreakSecondary, uint8(3))
	run.Field("BeatDisplay.BreakSecondaryTuplet", beat.Display.BreakSecondaryTuplet, true)
	run.Field("BeatDisplay.ForceBracket", beat.Display.ForceBracket, true)
	for _, direction := range []VoiceDirection{VoiceDirectionNone, VoiceDirectionUp, VoiceDirectionDown} {
		run.Field("Voice.Direction", direction, direction)
		run.Field("BeatDisplay.BeamDirection", direction, direction)
	}
	for _, bracket := range []TupletBracket{TupletBracketNone, TupletBracketStart, TupletBracketEnd} {
		run.Field("BeatDisplay.TupletBracket", bracket, bracket)
	}
	for _, lineBreak := range []LineBreak{LineBreakNone, LineBreakBreak, LineBreakProtect} {
		run.Field("Measure.LineBreak", lineBreak, lineBreak)
	}
	for _, secondary := range []uint8{0, 3, 255} {
		run.Field("BeatDisplay.BreakSecondary", secondary, secondary)
	}

	want := map[string]ScoreLocation{
		"gp8.omit.page-setup":             {},
		"gp8.omit.marker-color":           {Measure: 0},
		"gp8.omit.track-display-settings": {Track: 0},
		"gp8.omit.track-indicate-tuning":  {Track: 0},
		"gp8.omit.measure-line-break":     {Track: 0, Staff: 0, Measure: 0},
		"gp8.omit.voice-direction":        {Track: 0, Staff: 0, Measure: 0, Voice: 0},
		"gp8.omit.beat-display":           {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
	}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for code, location := range want {
		entry := exportReportEntry(report, code)
		if entry == nil || entry.Location != location {
			t.Errorf("report = %#v, want %s at %#v", report.Entries, code, location)
		}
	}
	var doubleBarLocations []ScoreLocation
	for _, entry := range report.Entries {
		if entry.Code == "gp8.normalize.measure-double-bar-authority" {
			doubleBarLocations = append(doubleBarLocations, entry.Location)
		}
	}
	wantDoubleBarLocations := []ScoreLocation{{Track: 0, Staff: 0, Measure: 0}}
	if !reflect.DeepEqual(doubleBarLocations, wantDoubleBarLocations) {
		t.Errorf("double-bar locations = %#v, want %#v", doubleBarLocations, wantDoubleBarLocations)
	}
	if len(report.Entries) != len(want)+len(wantDoubleBarLocations) {
		t.Errorf("report has %d entries, want %d: %#v", len(report.Entries), len(want)+len(wantDoubleBarLocations), report.Entries)
	}

	data, exported, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil || !reflect.DeepEqual(exported, report) {
		t.Fatalf("export = %d bytes, %#v, %v", len(data), exported, err)
	}
	values := extractGPIFLeafText(t, data)
	roundTrip, parseErr := Parse(data)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	run.Field("Track.Visible", roundTrip.Tracks[0].Visible, false)
	run.Wire("gpifSection.Text", values["GPIF/MasterBars/MasterBar/Section/Text"], marker.Title)
	run.Wire("gpifTrack.Color", values["GPIF/Tracks/Track/Color"], "51 102 153")

	layout := buildGP8LayoutConfiguration(song)
	if got := binary.BigEndian.Uint32(layout[:4]); got != 4 {
		t.Errorf("layout record count = %d, want 4", got)
	}
	if got := layout[6]; got != 0 {
		t.Errorf("track visible byte = %#x, want 0", got)
	}
	track.Visible = true
	if got := buildGP8LayoutConfiguration(song)[6]; got != 0xff {
		t.Errorf("track visible byte = %#x, want 0xff", got)
	}
	run.Field("Track.Visible", track.Visible, true)

	if got := gp8TrackViewFlags(track); got != 0x03 {
		t.Errorf("notation and tablature flags = %#x, want 0x03", got)
	}
	track.Settings = TrackSettings{}
	run.Field("Track.Settings", track.Settings, TrackSettings{})
	for _, field := range []struct {
		path string
		got  bool
	}{
		{"TrackSettings.Tablature", track.Settings.Tablature},
		{"TrackSettings.Notation", track.Settings.Notation},
		{"TrackSettings.DiagramAreBelow", track.Settings.DiagramAreBelow},
		{"TrackSettings.ShowRhythm", track.Settings.ShowRhythm},
		{"TrackSettings.ForceHorizontal", track.Settings.ForceHorizontal},
		{"TrackSettings.ForceChannels", track.Settings.ForceChannels},
		{"TrackSettings.DiagramList", track.Settings.DiagramList},
		{"TrackSettings.DiagramInScore", track.Settings.DiagramInScore},
		{"TrackSettings.AutoLetRing", track.Settings.AutoLetRing},
		{"TrackSettings.AutoBrush", track.Settings.AutoBrush},
		{"TrackSettings.ExtendRhythmic", track.Settings.ExtendRhythmic},
	} {
		run.Field(field.path, field.got, false)
	}
	zeroReport := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	entry := exportReportEntry(zeroReport, "gp8.normalize.track-view")
	if entry == nil || entry.Location != (ScoreLocation{Track: 0}) {
		t.Errorf("zero track-view report = %#v", zeroReport.Entries)
	}

	strict := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, strict)
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict display export = %d bytes, %v", len(strictData), strictErr)
	}
}
