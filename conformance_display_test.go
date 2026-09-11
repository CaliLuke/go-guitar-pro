// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func TestConformancePageSetupImport(t *testing.T) {
	runConformancePageSetupImport(newConformanceRun(t))
}

func runConformancePageSetupImport(run *conformanceRun) {
	song := parseTestFixture(run.t, "testdata/gp5/score-info.gp5")
	page := song.PageSetup
	run.Omitted("Song.PageSetup", song.PageSetup, page)
	run.Omitted("PageSetup.Words", page.Words, "Words by %WORDS%")
	run.Omitted("PageSetup.Music", page.Music, "Music by %MUSIC%")
	run.Omitted("PageSetup.PageNumber", page.PageNumber, "Page %N%/%P%")
	run.Omitted("PageSetup.Copyright", page.Copyright, "Copyright %COPYRIGHT%\nAll Rights Reserved - International Copyright Secured")
	run.Omitted("PageSetup.WordAndMusic", page.WordAndMusic, "Words & Music by %WORDSMUSIC%")
	run.Omitted("PageSetup.Artist", page.Artist, "%ARTIST%")
	run.Omitted("PageSetup.Album", page.Album, "%ALBUM%")
	run.Omitted("PageSetup.Title", page.Title, "%TITLE%")
	run.Omitted("PageSetup.Subtitle", page.Subtitle, "%SUBTITLE%")
	run.Omitted("PageSetup.PageHeight", page.PageHeight, int32(297))
	run.Omitted("PageSetup.ScoreSizeProportion", page.ScoreSizeProportion, float32(1))
	run.Omitted("PageSetup.PageWidth", page.PageWidth, int32(210))
	run.Omitted("PageSetup.MarginBottom", page.MarginBottom, int32(10))
	run.Omitted("PageSetup.MarginTop", page.MarginTop, int32(15))
	run.Omitted("PageSetup.MarginRight", page.MarginRight, int32(10))
	run.Omitted("PageSetup.MarginLeft", page.MarginLeft, int32(10))
	run.Omitted("PageSetup.HeaderAndFooter", page.HeaderAndFooter, uint16(0x1ff))
}

func TestConformanceDisplayExportPolicy(t *testing.T) {
	runConformanceDisplayExportPolicy(newConformanceRun(t))
}

func runConformanceDisplayExportPolicy(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	wantPageSetup := PageSetup{
		Words: "Words template", Music: "Music template", PageNumber: "Page %N%/%P%",
		Copyright: "Copyright template", WordAndMusic: "Both template", Artist: "Artist template",
		Album: "Album template", Title: "Title template", Subtitle: "Subtitle template",
		PageHeight: 1200, ScoreSizeProportion: 1.25, PageWidth: 900,
		MarginBottom: 41, MarginTop: 42, MarginRight: 43, MarginLeft: 44,
		HeaderAndFooter: 0x155,
	}
	song.PageSetup = wantPageSetup
	marker := song.MeasureHeaders[0].Marker
	marker.Title = "Section & 名"
	marker.Color = 0x123456
	track := &song.Tracks[0]
	track.Visible = false
	track.IndicateTuning = true
	wantSettings := TrackSettings{
		Tablature: true, Notation: true, DiagramAreBelow: true, ShowRhythm: true,
		ForceHorizontal: true, ForceChannels: true, DiagramList: true, DiagramInScore: true,
		AutoLetRing: true, AutoBrush: true, ExtendRhythmic: true,
	}
	track.Settings = wantSettings
	measure := &track.Staves[0].Measures[0]
	measure.LineBreak = LineBreakProtect
	measure.HasDoubleBar = true
	voice := &measure.Voices[0]
	voice.Direction = VoiceDirectionDown
	beat := &voice.Beats[0]
	wantBeatDisplay := BeatDisplay{
		BreakBeam: true, ForceBeam: true, BeamDirection: VoiceDirectionUp,
		TupletBracket: TupletBracketEnd, BreakSecondary: 3,
		BreakSecondaryTuplet: true, ForceBracket: true,
	}
	beat.Display = wantBeatDisplay

	run.Field("Song.PageSetup", song.PageSetup, wantPageSetup)
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
	run.Preserved("MeasureHeader.Marker", song.MeasureHeaders[0].Marker, marker)
	run.Preserved("Marker.Title", marker.Title, "Section & 名")
	run.Omitted("Marker.Color", marker.Color, int32(0x123456))
	run.Preserved("Track.Visible", track.Visible, false)
	run.ClaimPrimary(claimSite("track-identity", "model", "M02-DISPLAY-EXPORT", "non-default track color"), claimSite("track-identity", "export", "M02-DISPLAY-EXPORT", "non-default track color")).Preserved("Track.Color", track.Color, int32(0x336699))
	run.Omitted("Track.IndicateTuning", track.IndicateTuning, true)
	run.Normalized("Track.Settings", track.Settings, wantSettings)
	run.Normalized("TrackSettings.Tablature", track.Settings.Tablature, true)
	run.Normalized("TrackSettings.Notation", track.Settings.Notation, true)
	run.Omitted("TrackSettings.DiagramAreBelow", track.Settings.DiagramAreBelow, true)
	run.Omitted("TrackSettings.ShowRhythm", track.Settings.ShowRhythm, true)
	run.Omitted("TrackSettings.ForceHorizontal", track.Settings.ForceHorizontal, true)
	run.Omitted("TrackSettings.ForceChannels", track.Settings.ForceChannels, true)
	run.Omitted("TrackSettings.DiagramList", track.Settings.DiagramList, true)
	run.Omitted("TrackSettings.DiagramInScore", track.Settings.DiagramInScore, true)
	run.Omitted("TrackSettings.AutoLetRing", track.Settings.AutoLetRing, true)
	run.Omitted("TrackSettings.AutoBrush", track.Settings.AutoBrush, true)
	run.Omitted("TrackSettings.ExtendRhythmic", track.Settings.ExtendRhythmic, true)
	run.Omitted("Measure.LineBreak", measure.LineBreak, LineBreakProtect)
	run.Normalized("Measure.HasDoubleBar", measure.HasDoubleBar, true)
	run.Omitted("Voice.Direction", voice.Direction, VoiceDirectionDown)
	run.Omitted("Beat.Display", beat.Display, wantBeatDisplay)
	run.Omitted("BeatDisplay.BreakBeam", beat.Display.BreakBeam, true)
	run.Omitted("BeatDisplay.ForceBeam", beat.Display.ForceBeam, true)
	run.Omitted("BeatDisplay.BeamDirection", beat.Display.BeamDirection, VoiceDirectionUp)
	run.Omitted("BeatDisplay.TupletBracket", beat.Display.TupletBracket, TupletBracketEnd)
	run.Omitted("BeatDisplay.BreakSecondary", beat.Display.BreakSecondary, uint8(3))
	run.Omitted("BeatDisplay.BreakSecondaryTuplet", beat.Display.BreakSecondaryTuplet, true)
	run.Omitted("BeatDisplay.ForceBracket", beat.Display.ForceBracket, true)
	for index, direction := range []VoiceDirection{VoiceDirectionNone, VoiceDirectionUp, VoiceDirectionDown} {
		run.Field("Voice.Direction", int(direction), index)
		run.Field("BeatDisplay.BeamDirection", int(direction), index)
		probe := semanticValidPitchedGP8Song(t)
		probe.Tracks[0].Measures[0].Voices[0].Direction = direction
		probeReport := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		code := "gp8.omit.voice-direction"
		if direction != VoiceDirectionNone {
			probe.Tracks[0].Measures[0].Voices[0].Beats[0].Display.BeamDirection = direction
			probeReport = PreflightExport(probe, ExportFormatGP8, ExportOptions{})
			code = "gp8.omit.beat-display-beam-direction"
		}
		run.Enum([]string{"VoiceDirection.VoiceDirectionNone", "VoiceDirection.VoiceDirectionUp", "VoiceDirection.VoiceDirectionDown"}[index], hasExportCode(probeReport, code), direction != VoiceDirectionNone)
	}
	for index, bracket := range []TupletBracket{TupletBracketNone, TupletBracketStart, TupletBracketEnd} {
		run.Field("BeatDisplay.TupletBracket", int(bracket), index)
		probe := semanticValidPitchedGP8Song(t)
		probe.Tracks[0].Measures[0].Voices[0].Beats[0].Display.TupletBracket = bracket
		probeReport := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		run.Enum([]string{"TupletBracket.TupletBracketNone", "TupletBracket.TupletBracketStart", "TupletBracket.TupletBracketEnd"}[index], hasExportCode(probeReport, "gp8.omit.beat-display-tuplet-bracket"), bracket != TupletBracketNone)
	}
	for index, lineBreak := range []LineBreak{LineBreakNone, LineBreakBreak, LineBreakProtect} {
		run.Field("Measure.LineBreak", int(lineBreak), index)
		probe := semanticValidPitchedGP8Song(t)
		probe.Tracks[0].Measures[0].LineBreak = lineBreak
		probeReport := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		run.Enum([]string{"LineBreak.LineBreakNone", "LineBreak.LineBreakBreak", "LineBreak.LineBreakProtect"}[index], hasExportCode(probeReport, "gp8.omit.measure-line-break"), lineBreak != LineBreakNone)
	}
	for index, secondary := range []uint8{0, 3, 255} {
		run.Field("BeatDisplay.BreakSecondary", secondary, []uint8{0, 3, 255}[index])
	}

	want := map[string]ScoreLocation{
		"gp8.omit.page-setup":                          {},
		"gp8.omit.marker-color":                        {Measure: 0},
		"gp8.omit.track-display-settings":              {Track: 0},
		"gp8.omit.track-indicate-tuning":               {Track: 0},
		"gp8.omit.measure-line-break":                  {Track: 0, Staff: 0, Measure: 0},
		"gp8.omit.voice-direction":                     {Track: 0, Staff: 0, Measure: 0, Voice: 0},
		"gp8.omit.beat-display-break-beam":             {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
		"gp8.omit.beat-display-force-beam":             {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
		"gp8.omit.beat-display-beam-direction":         {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
		"gp8.omit.beat-display-tuplet-bracket":         {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
		"gp8.omit.beat-display-break-secondary":        {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
		"gp8.omit.beat-display-break-secondary-tuplet": {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
		"gp8.omit.beat-display-force-bracket":          {Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0},
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
	run.ClaimReport(claimSite("track-identity", "export", "M02-DISPLAY-EXPORT", "non-default track color")).Report("M02-DISPLAY-EXPORT", sortedSemanticValues(reportCodes(report)), []string{
		"gp8.normalize.measure-double-bar-authority", "gp8.omit.beat-display-beam-direction", "gp8.omit.beat-display-break-beam",
		"gp8.omit.beat-display-break-secondary", "gp8.omit.beat-display-break-secondary-tuplet", "gp8.omit.beat-display-force-beam",
		"gp8.omit.beat-display-force-bracket", "gp8.omit.beat-display-tuplet-bracket", "gp8.omit.marker-color", "gp8.omit.measure-line-break",
		"gp8.omit.page-setup", "gp8.omit.track-display-settings", "gp8.omit.track-indicate-tuning", "gp8.omit.voice-direction",
	})

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
	run.ClaimSerialization(claimSite("track-identity", "export", "M02-DISPLAY-EXPORT", "non-default track color")).Wire("gpifTrack.Color", values["GPIF/Tracks/Track/Color"], "51 102 153")

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
