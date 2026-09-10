// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestConformanceSharedValidation(t *testing.T) {
	runConformanceSharedValidation(newConformanceRun(t))
}

func runConformanceSharedValidation(run *conformanceRun) {
	t := run.t
	tests := []struct {
		name     string
		code     string
		location ScoreLocation
		mutate   func(*Song)
	}{
		{name: "channel reference", code: "score.track.channel-reference", location: ScoreLocation{Track: 0}, mutate: func(song *Song) {
			song.Tracks[0].ChannelIndex = len(song.Channels)
		}},
		{name: "sound reference", code: "score.sound-automation.reference", location: ScoreLocation{Track: 0}, mutate: func(song *Song) {
			song.Tracks[0].Sounds = []TrackSound{{Name: "only"}}
			song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Sound: 1}}
		}},
		{name: "articulation reference", code: "score.note.percussion-reference", location: ScoreLocation{Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0, Note: 0}, mutate: func(song *Song) {
			track := &song.Tracks[0]
			track.PercussionTrack = true
			track.PercussionArticulations = []PercussionArticulation{{OutputMIDINumber: 38}}
			note := &track.Measures[0].Voices[0].Beats[0].Notes[0]
			note.Value = 38
			note.HasPercussionArticulation = true
			note.PercussionArticulation = 1
		}},
		{name: "measure count", code: "score.staff.measure-count", location: ScoreLocation{Track: 0, Staff: 0}, mutate: func(song *Song) {
			song.MeasureHeaders = append(song.MeasureHeaders, defaultMeasureHeader())
		}},
		{name: "duration", code: "score.beat.duration", location: ScoreLocation{Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0}, mutate: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Duration.TupletEnters = 3
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Duration.TupletTimes = 0
		}},
		{name: "legacy tempo range", code: "score.tempo", mutate: func(song *Song) {
			song.Tempo = -1
		}},
		{name: "missing opening tempo", code: "score.tempo", mutate: func(song *Song) {
			song.Tempo = 0
			song.InitialTempo = SourceValue[BPM]{}
			song.TempoAutomations = []TempoAutomation{{Bar: 0, Position: 0.5, Tempo: 132.5}}
		}},
		{name: "semantic tempo NaN", code: "score.tempo", mutate: func(song *Song) {
			song.InitialTempo = KnownSourceValue(BPM(math.NaN()))
		}},
		{name: "semantic tempo infinity", code: "score.tempo", mutate: func(song *Song) {
			song.InitialTempo = KnownSourceValue(BPM(math.Inf(1)))
		}},
		{name: "tempo automation", code: "score.tempo-automation.value", mutate: func(song *Song) {
			song.TempoAutomations = []TempoAutomation{{Bar: 0, Position: 0, Tempo: math.NaN()}}
		}},
		{name: "volume range", code: "score.volume-automation.value", location: ScoreLocation{Track: 0, Measure: 0}, mutate: func(song *Song) {
			song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Position: 0.5, Value: 1.01}}
		}},
		{name: "volume non-finite", code: "score.volume-automation.value", location: ScoreLocation{Track: 0, Measure: 0}, mutate: func(song *Song) {
			song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Position: math.Inf(-1), Value: 0.5}}
		}},
		{name: "sound position", code: "score.sound-automation.location", location: ScoreLocation{Track: 0}, mutate: func(song *Song) {
			song.Tracks[0].Sounds = []TrackSound{{Name: "main"}}
			song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Position: math.NaN(), Sound: 0}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			test.mutate(song)
			diagnostics := ValidateSong(song)
			if !conformanceContractHasScoreDiagnosticAt(diagnostics, test.code, test.location) {
				t.Fatalf("diagnostics = %#v, want %s at %#v", diagnostics, test.code, test.location)
			}
			if len(diagnostics) != 1 {
				t.Fatalf("one invariant mutation produced %d diagnostics: %#v", len(diagnostics), diagnostics)
			}

			wantCode := "gp8.reject." + test.code
			options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{wantCode}}}
			preflight := PreflightExport(song, ExportFormatGP8, options)
			if !conformanceContractHasExportEntryAt(preflight, wantCode, test.location) {
				t.Fatalf("report = %#v, want %s at %#v", preflight.Entries, wantCode, test.location)
			}
			if len(preflight.Entries) != 1 {
				t.Fatalf("one invariant mutation produced %d report entries: %#v", len(preflight.Entries), preflight.Entries)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, options)
			if len(data) != 0 || err == nil {
				t.Fatalf("rejected export = %d bytes, %v", len(data), err)
			}
			if !reflect.DeepEqual(report, preflight) {
				t.Fatalf("export report = %#v, want preflight %#v", report, preflight)
			}
		})
	}
}

func TestConformanceAuthoredAndDerivedDistinction(t *testing.T) {
	runConformanceAuthoredAndDerivedDistinction(newConformanceRun(t))
}

func runConformanceAuthoredAndDerivedDistinction(run *conformanceRun) {
	t := run.t
	tests := []struct {
		name   string
		code   string
		mutate func(*Song)
	}{
		{name: "header alignment", code: "score.measure.header-alignment", mutate: func(song *Song) {
			song.MeasureHeaders = append(song.MeasureHeaders, defaultMeasureHeader())
			song.Tracks[0].Measures = append(song.Tracks[0].Measures, Measure{HeaderIndex: 1})
			song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
			if err := FinalizeSong(song); err != nil {
				t.Fatal(err)
			}
			song.Tracks[0].Measures[0].HeaderIndex = 1
		}},
		{name: "track ownership", code: "score.measure.track-ownership", mutate: func(song *Song) {
			song.Tracks[0].Measures[0].TrackIndex = 9
		}},
		{name: "staff ownership", code: "score.measure.staff-ownership", mutate: func(song *Song) {
			song.Tracks[0].Measures[0].StaffIndex = 9
		}},
		{name: "voice ownership", code: "score.voice.measure-ownership", mutate: func(song *Song) {
			song.Tracks[0].Measures[0].Voices[0].MeasureIndex = 9
		}},
		{name: "legacy beat start", code: "score.beat.start", mutate: func(song *Song) {
			wrong := int64(17)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Start = &wrong
		}},
		{name: "exact beat start", code: "score.beat.exact-start", mutate: func(song *Song) {
			wrong := mustScoreTime(t, 17, 3)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].ExactStart = &wrong
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			test.mutate(song)
			if diagnostics := ValidateSong(song); !hasScoreDiagnostic(diagnostics, test.code) {
				t.Fatalf("diagnostics = %#v, want derived %s", diagnostics, test.code)
			}
			if report := PreflightExport(song, ExportFormatGP8, ExportOptions{}); hasExportCode(report, "gp8.reject."+test.code) {
				t.Fatalf("derived diagnostic was treated as authored: %#v", report.Entries)
			}
			if _, err := Export(song, ExportFormatGP8); err != nil {
				t.Fatalf("export rejected documented derived disagreement: %v", err)
			}
		})
	}

	omitted := semanticExportProbeSong(t)
	omitted.Tracks[0].Measures[0].Voices[0].Beats[0].Start = nil
	omitted.Tracks[0].Measures[0].Voices[0].Beats[0].ExactStart = nil
	if diagnostics := ValidateSong(omitted); hasScoreDiagnostic(diagnostics, "score.beat.start") || hasScoreDiagnostic(diagnostics, "score.beat.exact-start") {
		t.Fatalf("omitted derived beat fields diagnosed: %#v", diagnostics)
	}
}

func TestConformanceFinalizeContracts(t *testing.T) {
	runConformanceFinalizeContracts(newConformanceRun(t))
}

func runConformanceFinalizeContracts(run *conformanceRun) {
	t := run.t
	for _, pickup := range []bool{false, true} {
		t.Run(fmt.Sprintf("pickup=%t", pickup), func(t *testing.T) {
			song := conformanceContractTimingSong(pickup)
			run.Field("Song.Anacrusis", song.Anacrusis, pickup)
			authored := conformanceContractSnapshot(song, true)
			if err := FinalizeSong(song); err != nil {
				t.Fatal(err)
			}
			if got := conformanceContractSnapshot(song, true); got != authored {
				t.Fatal("finalization changed authored state")
			}

			wantSecondStart := DurationQuarterTime + 4*DurationQuarterTime
			if pickup {
				wantSecondStart = DurationQuarterTime + 2*DurationQuarterTime
			}
			if got := song.MeasureHeaders[1].Start; got != wantSecondStart {
				t.Fatalf("second header start = %d, want %d", got, wantSecondStart)
			}
			run.Field("MeasureHeader.Start", song.MeasureHeaders[1].Start, wantSecondStart)
			run.Field("MeasureHeader.ExactStart", [2]int64{song.MeasureHeaders[1].ExactStart.Numerator(), song.MeasureHeaders[1].ExactStart.Denominator()}, [2]int64{wantSecondStart, 1})
			conformanceContractAssertFinalizedPaths(run, song)
			if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
				t.Fatalf("finalized diagnostics = %#v", diagnostics)
			}

			first := conformanceContractSnapshot(song, false)
			if !strings.Contains(first, "numerator") || !strings.Contains(first, "denominator") {
				t.Fatal("deep snapshot did not include private ScoreTime terms")
			}
			if err := FinalizeSong(song); err != nil {
				t.Fatal(err)
			}
			if second := conformanceContractSnapshot(song, false); second != first {
				t.Fatal("second finalization changed the deep score snapshot")
			}
		})
	}
}

func TestConformanceReadOnlyAndFileContracts(t *testing.T) {
	runConformanceReadOnlyAndFileContracts(newConformanceRun(t))
}

func runConformanceReadOnlyAndFileContracts(run *conformanceRun) {
	t := run.t
	song := conformanceContractReadOnlySong(t)
	options := ExportOptions{
		GP8:        GP8ExportOptions{PercussionNoteheads: map[int16]GP8PercussionNotehead{38: GP8PercussionNoteheadX}},
		LossPolicy: ExportLossPolicy{AllowedCodes: []string{"sentinel.allowed"}},
	}
	wantSong := conformanceContractSnapshot(song, false)
	wantOptions := conformanceContractSnapshot(options, false)
	assertReadOnly := func(operation string) {
		t.Helper()
		if got := conformanceContractSnapshot(song, false); got != wantSong {
			t.Fatalf("%s mutated the song", operation)
		}
		if got := conformanceContractSnapshot(options, false); got != wantOptions {
			t.Fatalf("%s mutated export options", operation)
		}
	}

	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("read-only baseline diagnostics = %#v", diagnostics)
	}
	assertReadOnly("ValidateSong")
	preflight := PreflightExport(song, ExportFormatGP8, options)
	assertReadOnly("PreflightExport")
	data, report, exportErr := ExportWithReport(song, ExportFormatGP8, options)
	if exportErr != nil || len(data) == 0 {
		t.Fatalf("export = %d bytes, %v", len(data), exportErr)
	}
	assertReadOnly("ExportWithReport")
	if !reflect.DeepEqual(report, preflight) {
		t.Fatalf("export report = %#v, want preflight %#v", report, preflight)
	}

	strictSong := semanticExportProbeSong(t)
	strictSong.Writer = "must be reported"
	strict := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	wantStrictSong := conformanceContractSnapshot(strictSong, false)
	wantStrictOptions := conformanceContractSnapshot(strict, false)
	directory := t.TempDir()
	existing := filepath.Join(directory, "existing.gp")
	wantContents := []byte("existing sentinel")
	if writeErr := os.WriteFile(existing, wantContents, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	if fileErr := ExportFileWithOptions(existing, strictSong, ExportFormatGP8, strict); fileErr == nil {
		t.Fatal("strict file export unexpectedly succeeded")
	} else {
		var lossErr *ExportLossError
		if !errors.As(fileErr, &lossErr) {
			t.Fatalf("strict file export error = %T, want ExportLossError", fileErr)
		}
	}
	gotContents, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotContents, wantContents) {
		t.Fatalf("failed strict export overwrote destination: %q", gotContents)
	}

	missing := filepath.Join(directory, "missing.gp")
	if err := ExportFileWithOptions(missing, strictSong, ExportFormatGP8, strict); err == nil {
		t.Fatal("strict file export unexpectedly succeeded")
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed strict export created destination: %v", err)
	}
	if got := conformanceContractSnapshot(strictSong, false); got != wantStrictSong {
		t.Fatal("failed strict file export mutated the song")
	}
	if got := conformanceContractSnapshot(strict, false); got != wantStrictOptions {
		t.Fatal("failed strict file export mutated options")
	}
}

func conformanceContractTimingSong(pickup bool) *Song {
	quarter := defaultBeat()
	quarter.Status = BeatStatusRest
	half := quarter
	half.Duration.Value = 2
	dottedQuarter := quarter
	dottedQuarter.Duration.Dotted = true
	makeMeasures := func(first []Voice) []Measure {
		return []Measure{
			{HeaderIndex: 0, Voices: first},
			{HeaderIndex: 1, Voices: []Voice{{Beats: []Beat{quarter}}, {Beats: []Beat{half}}}},
		}
	}
	firstStaffMeasures := makeMeasures([]Voice{{Beats: []Beat{quarter}}, {Beats: []Beat{half}}})
	secondStaffMeasures := makeMeasures([]Voice{{Beats: []Beat{dottedQuarter}}, {}})
	secondTrackMeasures := makeMeasures([]Voice{{Beats: []Beat{quarter, quarter}}, {Beats: []Beat{quarter}}})
	return &Song{
		Anacrusis: pickup,
		Tempo:     120,
		MeasureHeaders: []MeasureHeader{
			defaultMeasureHeader(),
			defaultMeasureHeader(),
		},
		Tracks: []Track{
			{ChannelIndex: -1, Measures: firstStaffMeasures, Staves: []Staff{
				{Measures: firstStaffMeasures},
				{Measures: secondStaffMeasures},
			}},
			{ChannelIndex: -1, Measures: secondTrackMeasures, Staves: []Staff{{Measures: secondTrackMeasures}}},
		},
	}
}

func conformanceContractAssertFinalizedPaths(run *conformanceRun, song *Song) {
	t := run.t
	t.Helper()
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			measures := song.Tracks[trackIndex].Staves[staffIndex].Measures
			for measureIndex := range measures {
				measure := &measures[measureIndex]
				if measure.TrackIndex != trackIndex || measure.StaffIndex != staffIndex || measure.Start != song.MeasureHeaders[measureIndex].Start || measure.ExactStart.Compare(song.MeasureHeaders[measureIndex].ExactStart) != 0 {
					t.Fatalf("measure path %d/%d/%d not finalized: %#v", trackIndex, staffIndex, measureIndex, measure)
				}
				run.Field("Measure.TrackIndex", measure.TrackIndex, trackIndex)
				run.Field("Measure.StaffIndex", measure.StaffIndex, staffIndex)
				run.Field("Measure.Start", measure.Start, song.MeasureHeaders[measureIndex].Start)
				run.Field("Measure.ExactStart", [2]int64{measure.ExactStart.Numerator(), measure.ExactStart.Denominator()}, [2]int64{song.MeasureHeaders[measureIndex].ExactStart.Numerator(), song.MeasureHeaders[measureIndex].ExactStart.Denominator()})
				for voiceIndex := range measure.Voices {
					voice := &measure.Voices[voiceIndex]
					if int(voice.MeasureIndex) != measureIndex {
						t.Fatalf("voice path %d/%d/%d/%d ownership = %d", trackIndex, staffIndex, measureIndex, voiceIndex, voice.MeasureIndex)
					}
					run.Field("Voice.MeasureIndex", voice.MeasureIndex, int16(measureIndex))
					if len(voice.Beats) != 0 {
						beat := voice.Beats[0]
						if beat.Start == nil || beat.ExactStart == nil || *beat.Start != measure.Start || beat.ExactStart.Compare(measure.ExactStart) != 0 {
							t.Fatalf("voice path %d/%d/%d/%d did not start at its measure", trackIndex, staffIndex, measureIndex, voiceIndex)
						}
						run.Field("Beat.Start", *beat.Start, measure.Start)
						run.Field("Beat.ExactStart", [2]int64{beat.ExactStart.Numerator(), beat.ExactStart.Denominator()}, [2]int64{measure.ExactStart.Numerator(), measure.ExactStart.Denominator()})
					}
				}
			}
		}
	}
}

func conformanceContractReadOnlySong(t *testing.T) *Song {
	t.Helper()
	song := semanticExportProbeSong(t)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	second := *beat
	second.Notes = append([]Note(nil), beat.Notes...)
	beat.Duration.TupletEnters, beat.Duration.TupletTimes = 7, 4
	second.Duration.TupletEnters, second.Duration.TupletTimes = 7, 4
	song.Tracks[0].Measures[0].Voices[0].Beats = []Beat{*beat, second}
	beat = &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	firstFret := uint8(1)
	root := PitchClass{Note: "C", Value: 0, Sharp: true}
	beat.Effect.Chord = &Chord{Name: "C", Length: 6, FirstFret: &firstFret, Root: &root, Strings: []int8{0, 3, 2, 0, 1, 0}, Barres: []Barre{{Fret: 1, Start: 1, End: 2}}}
	beat.Notes[0].Effect.Bend = &BendEffect{Kind: BendTypeBend, Points: []BendPoint{{Position: 0}, {Position: 12, Value: 2}}}
	exactFret := Fret(2)
	rawFret := int8(2)
	beat.Notes[0].Effect.Graces = []GraceEffect{{Duration: DurationThirtySecond, Fret: 2, ExactFret: &exactFret, RawFret: &rawFret, Velocity: Forte}}
	harmonicFret := int8(12)
	harmonicFloat := 12.0
	beat.Notes[0].Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, Fret: &harmonicFret, FretFloat: &harmonicFloat}
	direction := DirectionSignSegno
	song.MeasureHeaders[0].Direction = &direction
	song.MeasureHeaders[0].Marker = &Marker{Title: "nested", Color: 0x010203}
	song.Notice = []string{"first", "second"}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	secondExact := song.Tracks[0].Measures[0].Voices[0].Beats[1].ExactStart
	if secondExact == nil || secondExact.Denominator() == 1 {
		t.Fatalf("read-only baseline lacks fractional ScoreTime: %#v", secondExact)
	}
	return song
}

// conformanceContractSnapshot is a deterministic, non-JSON observation of the complete value
// graph. It dereferences pointers, walks slices and maps, and sees unexported
// ScoreTime terms through their primitive fields.
func conformanceContractSnapshot(value any, authoredOnly bool) string {
	var output strings.Builder
	conformanceContractWriteSnapshot(&output, reflect.ValueOf(value), authoredOnly)
	return output.String()
}

func conformanceContractWriteSnapshot(output *strings.Builder, value reflect.Value, authoredOnly bool) {
	if !value.IsValid() {
		output.WriteString("invalid")
		return
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			output.WriteString("nil")
			return
		}
		output.WriteByte('&')
		conformanceContractWriteSnapshot(output, value.Elem(), authoredOnly)
	case reflect.Struct:
		output.WriteString(value.Type().String())
		output.WriteByte('{')
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if authoredOnly && conformanceContractDerivedField(value.Type(), field.Name) {
				continue
			}
			output.WriteString(field.Name)
			output.WriteByte(':')
			conformanceContractWriteSnapshot(output, value.Field(index), authoredOnly)
			output.WriteByte(';')
		}
		output.WriteByte('}')
	case reflect.Slice:
		if value.IsNil() {
			output.WriteString("nil-slice")
			return
		}
		output.WriteByte('[')
		for index := 0; index < value.Len(); index++ {
			conformanceContractWriteSnapshot(output, value.Index(index), authoredOnly)
			output.WriteByte(';')
		}
		output.WriteByte(']')
	case reflect.Array:
		output.WriteByte('[')
		for index := 0; index < value.Len(); index++ {
			conformanceContractWriteSnapshot(output, value.Index(index), authoredOnly)
			output.WriteByte(';')
		}
		output.WriteByte(']')
	case reflect.Map:
		if value.IsNil() {
			output.WriteString("nil-map")
			return
		}
		items := make([]string, 0, value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			var item strings.Builder
			conformanceContractWriteSnapshot(&item, iterator.Key(), authoredOnly)
			item.WriteByte(':')
			conformanceContractWriteSnapshot(&item, iterator.Value(), authoredOnly)
			items = append(items, item.String())
		}
		sort.Strings(items)
		output.WriteByte('{')
		output.WriteString(strings.Join(items, ";"))
		output.WriteByte('}')
	case reflect.String:
		output.WriteString(strconv.Quote(value.String()))
	case reflect.Bool:
		output.WriteString(strconv.FormatBool(value.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		output.WriteString(strconv.FormatInt(value.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		output.WriteString(strconv.FormatUint(value.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		output.WriteString(strconv.FormatUint(math.Float64bits(value.Convert(reflect.TypeOf(float64(0))).Float()), 16))
	default:
		output.WriteString(value.Type().String())
	}
}

func conformanceContractDerivedField(owner reflect.Type, name string) bool {
	switch owner {
	case reflect.TypeOf(MeasureHeader{}):
		return name == "Start" || name == "ExactStart"
	case reflect.TypeOf(Measure{}):
		return name == "Start" || name == "ExactStart" || name == "TrackIndex" || name == "StaffIndex"
	case reflect.TypeOf(Voice{}):
		return name == "MeasureIndex"
	case reflect.TypeOf(Beat{}):
		return name == "Start" || name == "ExactStart"
	default:
		return false
	}
}

func conformanceContractHasScoreDiagnosticAt(diagnostics []ScoreDiagnostic, code string, location ScoreLocation) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Location == location {
			return true
		}
	}
	return false
}

func conformanceContractHasExportEntryAt(report ExportReport, code string, location ScoreLocation) bool {
	for _, entry := range report.Entries {
		if entry.Code == code && entry.Location == location {
			return true
		}
	}
	return false
}
