// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

const staffNotationCase = "M02-STAFF-NOTATION"
const slashNotationCase = "M02-SLASH-NOTATION"
const staffNotationFixture = "testdata/gp8/staff-notation-settings.gp"

func TestConformanceStaffNotation(t *testing.T) { runConformanceStaffNotation(newConformanceRun(t)) }

func runConformanceStaffNotation(run *conformanceRun) {
	t := run.t
	for flags := byte(0); flags < 16; flags++ {
		source := notationSourceWithFlags(t, flags)
		result, err := ParseWithOptions(source, ParseOptions{Strict: true})
		if err != nil {
			t.Fatal(err)
		}
		song := result.Song
		expectedFlags := flags
		if expectedFlags == 0 {
			expectedFlags = 1
		}
		expected := notationSettingsForFlags(expectedFlags)
		for _, staff := range song.Tracks[0].Staves {
			run.Normalized("Staff.NotationSettings", staff.NotationSettings, &expected)
			recordStaffNotationFields(run, *staff.NotationSettings, expected)
		}
		run.Preserved("Track.Visible", song.Tracks[0].Visible, true)
		run.Preserved("Track.Visible", song.Tracks[1].Visible, false)
		run.Normalized("TrackSettings.Notation", song.Tracks[0].Settings.Notation, expected.Standard)
		run.Normalized("TrackSettings.Tablature", song.Tracks[0].Settings.Tablature, expected.Tablature)
		if !song.Tracks[0].Visible || song.Tracks[1].Visible {
			t.Fatal("track visibility changed with notation selection")
		}
		cleanNotationProvenance(song)
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil {
			t.Fatal(err)
		}
		run.Report(staffNotationCase, reportCodes(report), []string{})
		archive, openErr := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if openErr != nil {
			t.Fatal(openErr)
		}
		configuration := readZipMember(t, archive, "Content/PartConfiguration")
		if configuration[9] != expectedFlags || configuration[10] != 14 {
			t.Fatalf("exact configuration flags=%x", configuration)
		}
		recordStaffNotationFields(run, notationSettingsForFlags(configuration[9]), expected)
	}
	testStaffNotationLosses(run)
	song := notationSong(t)
	source := song.Tracks[0].Staves[0].NotationSettings
	song.Tracks[0].Staves[1].NotationSettings.Numbered = true
	if source.Numbered {
		t.Fatal("notation settings share mutable storage")
	}
	// Nil explicitly restores the compatibility fallback without retaining slash/numbered.
	song.Tracks[0].Staves[1].NotationSettings = nil
	run.Normalized("Staff.NotationSettings", song.Tracks[0].Staves[1].NotationSettings, (*StaffNotationSettings)(nil))
}

func recordStaffNotationFields(run *conformanceRun, got, want StaffNotationSettings) {
	run.Normalized("StaffNotationSettings.Standard", got.Standard, want.Standard)
	run.Normalized("StaffNotationSettings.Tablature", got.Tablature, want.Tablature)
	run.Normalized("StaffNotationSettings.Slash", got.Slash, want.Slash)
	run.Normalized("StaffNotationSettings.Numbered", got.Numbered, want.Numbered)
}

func notationSettingsForFlags(flags byte) StaffNotationSettings {
	return StaffNotationSettings{Standard: flags&1 != 0, Tablature: flags&2 != 0, Slash: flags&4 != 0, Numbered: flags&8 != 0}
}

func notationSourceWithFlags(t *testing.T, flags byte) []byte {
	t.Helper()
	data, err := os.ReadFile(staffNotationFixture)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	configuration := readZipMember(t, archive, "Content/PartConfiguration")
	configuration[9] = flags
	return conformanceBackingArchive(t, string(readZipMember(t, archive, "Content/score.gpif")), map[string][]byte{"Content/PartConfiguration": configuration, "Content/LayoutConfiguration": readZipMember(t, archive, "Content/LayoutConfiguration")})
}

func notationSong(t *testing.T) *Song {
	t.Helper()
	song := parseTestFixture(t, staffNotationFixture)
	cleanNotationProvenance(song)
	return song
}

func cleanNotationProvenance(song *Song) {
	song.Version = Version{}
	for index := range song.Tracks {
		track := &song.Tracks[index]
		track.Settings = TrackSettings{Notation: track.Settings.Notation, Tablature: track.Settings.Tablature}
	}
}

func staffNotationLossCases() []struct {
	code string
	edit func(*StaffNotationSettings)
} {
	return []struct {
		code string
		edit func(*StaffNotationSettings)
	}{
		{"gp8.omit.staff-standard-notation", func(s *StaffNotationSettings) { s.Standard = false }},
		{"gp8.omit.staff-tablature", func(s *StaffNotationSettings) { s.Tablature = true }},
		{"gp8.omit.staff-slash-notation", func(s *StaffNotationSettings) { s.Slash = true }},
		{"gp8.omit.staff-numbered-notation", func(s *StaffNotationSettings) { s.Numbered = true }},
	}
}

func testStaffNotationLosses(run *conformanceRun) {
	t := run.t
	for _, test := range staffNotationLossCases() {
		song := notationSong(t)
		test.edit(song.Tracks[0].Staves[1].NotationSettings)
		before := conformanceContractSnapshot(song, false)
		for _, allowed := range [][]string{nil, {"gp8.normalize.track-view"}, {test.code}} {
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
			run.Report(staffNotationCase, reportCodes(report), []string{test.code})
			if len(report.Entries) != 1 || report.Entries[0].Disposition != ExportDispositionOmitted || report.Entries[0].Location != (ScoreLocation{Track: 0, Staff: 1}) {
				t.Fatalf("staff loss=%#v", report.Entries)
			}
			if len(allowed) == 0 || allowed[0] != test.code {
				var loss *ExportLossError
				if !errors.As(err, &loss) || len(data) != 0 {
					t.Fatalf("strict staff loss=%v", err)
				}
			} else if err != nil || len(data) == 0 {
				t.Fatalf("exact allowance failed: %v", err)
			}
			if conformanceContractSnapshot(song, false) != before {
				t.Fatal("loss policy mutated staff settings")
			}
		}
	}
}

func TestConformanceSlashNotation(t *testing.T) { runConformanceSlashNotation(newConformanceRun(t)) }

func runConformanceSlashNotation(run *conformanceRun) {
	t := run.t
	song := notationSong(t)
	first := &song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0]
	run.Preserved("Beat.Slashed", first.Slashed, true)
	second := &song.Tracks[0].Staves[1].Measures[0].Voices[0].Beats[0]
	run.Preserved("Beat.Slashed", second.Slashed, false)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err)
	}
	run.Report(slashNotationCase, reportCodes(report), []string{})
	wire := conformanceWireDocument(t, data)
	run.Wire("gpifBeat.Slashed", wire.Beats.Beats[0].Slashed != nil, true)
	run.Wire("gpifBeat.Slashed", wire.Beats.Beats[1].Slashed != nil, false)
	before := append([]Note(nil), first.Notes...)
	first.Slashed = false
	second.Slashed = true
	parsed, err := Parse(mustStringNumberExport(t, song))
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Beat.Slashed", parsed.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Slashed, false)
	run.Preserved("Beat.Slashed", parsed.Tracks[0].Staves[1].Measures[0].Voices[0].Beats[0].Slashed, true)
	if !reflect.DeepEqual(first.Notes, before) || parsed.Tracks[1].Staves[0].Measures[0].Voices[0].Beats[0].Slashed {
		t.Fatal("slash edit mutated notes or a reused occurrence")
	}
}

type notationNoteFact struct{ String, Fret, MIDI int }
type notationBeatFact struct {
	Slashed bool
	Notes   []notationNoteFact
}
type notationStaffFact struct {
	Standard, Tablature, Slash, Numbered bool
	Tuning                               []int
	Beats                                []notationBeatFact
}
type notationTrackFact struct {
	Visible bool
	Staves  []notationStaffFact
}

func TestAlphaTabStaffNotation(t *testing.T) {
	requireAlphaTabConformance(t)
	ordinary := notationStaffFact{Standard: true, Tuning: []int{64, 59, 55, 50, 45, 40}, Beats: []notationBeatFact{{Notes: []notationNoteFact{{4, 3, 58}}}, {Notes: []notationNoteFact{}}, {Notes: []notationNoteFact{}}, {Notes: []notationNoteFact{}}}}
	first := ordinary
	first.Beats = []notationBeatFact{{Slashed: true, Notes: []notationNoteFact{{4, 3, 58}}}, {Notes: []notationNoteFact{}}, {Notes: []notationNoteFact{}}, {Notes: []notationNoteFact{}}}
	alternate := ordinary
	alternate.Standard = false
	alternate.Tablature = true
	alternate.Slash = true
	alternate.Numbered = true
	expected := []notationTrackFact{{Visible: true, Staves: []notationStaffFact{first, ordinary}}, {Visible: false, Staves: []notationStaffFact{alternate}}}
	var got []notationTrackFact
	readAlphaTabOracleFacts(t, "--staff-notation", staffNotationFixture, &got)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("source consumer=%#v", got)
	}
	for flags := byte(1); flags < 16; flags++ {
		song := notationSong(t)
		settings := notationSettingsForFlags(flags)
		for i := range song.Tracks[0].Staves {
			copy := settings
			song.Tracks[0].Staves[i].NotationSettings = &copy
			expected[0].Staves[i].Standard = settings.Standard
			expected[0].Staves[i].Tablature = settings.Tablature
			expected[0].Staves[i].Slash = settings.Slash
			expected[0].Staves[i].Numbered = settings.Numbered
		}
		data := mustStringNumberExport(t, song)
		readAlphaTabOracleFacts(t, "--staff-notation", writeConformanceFixture(t, data), &got)
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("flags=%x consumer=%#v", flags, got)
		}
	}
	// Divergent same-track settings are a retaining-consumer limit, not a support claim.
	for _, test := range staffNotationLossCases() {
		song := notationSong(t)
		test.edit(song.Tracks[0].Staves[1].NotationSettings)
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Entries) != 1 || report.Entries[0].Code != test.code {
			t.Fatalf("residual report=%#v", report.Entries)
		}
		readAlphaTabOracleFacts(t, "--staff-notation", writeConformanceFixture(t, data), &got)
		if !reflect.DeepEqual(got[0].Staves, []notationStaffFact{first, ordinary}) {
			t.Fatalf("residual consumer=%#v", got)
		}
	}
	song := notationSong(t)
	song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Slashed = false
	song.Tracks[0].Staves[1].Measures[0].Voices[0].Beats[0].Slashed = true
	readAlphaTabOracleFacts(t, "--staff-notation", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
	if got[0].Staves[0].Beats[0].Slashed || !got[0].Staves[1].Beats[0].Slashed || got[1].Staves[0].Beats[0].Slashed {
		t.Fatal("consumer conflated staff and beat slash flags")
	}
}

func TestPartConfigurationReaderBounds(t *testing.T) {
	for _, data := range [][]byte{
		make([]byte, maxPartConfigurationSize+1),
		{255, 255, 255, 255},
		{0, 0, 0, 2, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0, 1, 16, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	} {
		if _, err := readPartConfiguration(data); err == nil {
			t.Fatalf("malformed configuration accepted: %d bytes", len(data))
		}
	}
	if flags, err := readPartConfiguration(make([]byte, 8)); err != nil || len(flags) != 0 {
		t.Fatalf("empty configuration=%v %v", flags, err)
	}
	// The pinned source applies a short first view only to corresponding tracks.
	song := notationSong(t)
	song.Tracks[1].Staves[0].NotationSettings = nil
	partial := []byte{0, 0, 0, 1, 0, 0, 0, 0, 1, 5, 0, 0, 0, 0}
	if err := applyPartConfiguration(song, partial); err != nil {
		t.Fatal(err)
	}
	if !song.Tracks[0].Staves[0].NotationSettings.Slash || song.Tracks[1].Staves[0].NotationSettings != nil {
		t.Fatal("short view fabricated missing track settings")
	}
}

func TestAlphaTabLegacyPartConfiguration(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, path := range []string{"testdata/gp6/slash.gpx", "testdata/gp7/slash.gp", "testdata/gp7/colors.gp"} {
		t.Run(path, func(t *testing.T) {
			var source, output []notationTrackFact
			readAlphaTabOracleFacts(t, "--staff-notation", path, &source)
			song := parseTestFixture(t, path)
			readAlphaTabOracleFacts(t, "--staff-notation", writeConformanceFixture(t, mustStringNumberExport(t, song)), &output)
			for ti, track := range source {
				for si, staff := range track.Staves {
					want := StaffNotationSettings{Standard: staff.Standard, Tablature: staff.Tablature, Slash: staff.Slash, Numbered: staff.Numbered}
					got := output[ti].Staves[si]
					actual := StaffNotationSettings{Standard: got.Standard, Tablature: got.Tablature, Slash: got.Slash, Numbered: got.Numbered}
					if actual != want || song.Tracks[ti].Staves[si].NotationSettings == nil || *song.Tracks[ti].Staves[si].NotationSettings != want {
						t.Fatalf("track=%d staff=%d source=%#v output=%#v", ti, si, want, actual)
					}
				}
			}
		})
	}
}

func TestStaffNotationTargetDefaults(t *testing.T) {
	for _, test := range []struct {
		name, code string
		edit       func(*Song)
	}{
		{"empty flags", "gp8.normalize.track-view", func(song *Song) {
			for i := range song.Tracks[0].Staves {
				*song.Tracks[0].Staves[i].NotationSettings = StaffNotationSettings{}
			}
			song.Tracks[0].Staves = song.Tracks[0].Staves[:1]
		}},
		{"percussion tablature", "gp8.omit.percussion-tablature", func(song *Song) {
			song.Tracks[0].PercussionTrack = true
			song.Tracks[0].Staves = song.Tracks[0].Staves[:1]
			song.Tracks[0].Staves[0].PercussionTrack = true
			song.Tracks[0].Staves[0].NotationSettings.Tablature = true
			song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0].Value = 38
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := notationSong(t)
			test.edit(song)
			before := conformanceContractSnapshot(song, false)
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			var loss *ExportLossError
			if !errors.As(err, &loss) || len(data) != 0 {
				t.Fatalf("strict result: bytes=%d error=%v", len(data), err)
			}
			found := false
			for _, entry := range report.Entries {
				if entry.Code == test.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing %s: %#v", test.code, report)
			}
			if conformanceContractSnapshot(song, false) != before {
				t.Fatal("notation target handling mutated source")
			}
		})
	}
}

func TestSlashedElementPresence(t *testing.T) {
	data, err := os.ReadFile(staffNotationFixture)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(readZipMember(t, archive, "Content/score.gpif"))
	for _, spelling := range []string{"<Slashed />", "<Slashed>false</Slashed>"} {
		t.Run(spelling, func(t *testing.T) {
			source := conformanceBackingArchive(t, strings.Replace(raw, "<Slashed />", spelling, 1), nil)
			result, parseErr := ParseWithOptions(source, ParseOptions{Strict: true})
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			first := result.Song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0]
			second := result.Song.Tracks[0].Staves[1].Measures[0].Voices[0].Beats[0]
			if !first.Slashed || second.Slashed || first.Status != BeatStatusNormal || len(first.Notes) != 1 {
				t.Fatal("presence changed beat content or absent control")
			}
		})
	}
}
