// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestGPIFSemanticPreservingTransformations(t *testing.T) {
	data, err := Export(conformanceExportSong(), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := normalizeGoScore(baseline)
	transforms := []struct {
		name      string
		transform func(string) string
	}{
		{"property order", reorderFirstNoteProperties},
		{"object IDs", renameGPIFObjectIDs},
		{"definition table order", reverseGPIFNoteDefinitions},
		{"reused definition", duplicateFirstGPIFRhythm},
	}
	for _, test := range transforms {
		t.Run(test.name, func(t *testing.T) {
			transformed := rewriteConformanceGPIF(t, data, test.transform)
			got, parseErr := Parse(transformed)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if !reflect.DeepEqual(normalizeGoScore(got), want) {
				t.Fatal("semantic-preserving transformation changed normalized authored output")
			}
		})
	}

	repacked := repackGP8ForMetamorphicTest(t, data)
	got, err := Parse(repacked)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalizeGoScore(got), want) {
		t.Fatal("ZIP entry order or equivalent compression changed normalized authored output")
	}
}

func TestGPIFMetamorphicSensitivity(t *testing.T) {
	data, err := Export(conformanceExportSong(), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	broken := rewriteConformanceGPIF(t, data, func(gpif string) string {
		return strings.Replace(gpif, `<Rhythm ref="0"`, `<Rhythm ref="missing"`, 1)
	})
	result, err := ParseWithOptions(broken, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if findParseDiagnostic(result.Diagnostics, ParseDiagnosticInvalidData, "rhythm") == nil {
		t.Fatal("broken reference mapping was not detected")
	}
}

func TestResilienceAdapterMutationSensitivity(t *testing.T) {
	testResilienceAdapterMutationSensitivity(t)
}

func testResilienceAdapterMutationSensitivity(t *testing.T) {
	t.Helper()
	if goSlide(SlideIntoFromAbove) == goSlide(SlideIntoFromBelow) {
		t.Fatal("slide adapter collapsed distinct enum values")
	}
	if goSlide(SlideType(99)) == goSlide(SlideNone) {
		t.Fatal("slide adapter collapsed an unknown enum into the default")
	}
}

func TestConformanceStructuralResilience(t *testing.T) {
	runConformanceStructuralResilience(newConformanceRun(t))
}

func runConformanceStructuralResilience(run *conformanceRun) {
	t := run.t
	t.Run("adapter sensitivity", testResilienceAdapterMutationSensitivity)
	container := semanticValidPitchedGP8Song(t)
	container.Tracks[0].Settings.Notation = true
	containerData, containerReport, containerErr := ExportWithReport(container, ExportFormatGP8, ExportOptions{})
	if containerErr != nil {
		t.Fatal(containerErr)
	}
	run.ClaimReport(claimSite("container", "export", "M25-STRUCTURAL-RESILIENCE", "GP5 input and GP8 output containers")).Report("M25-STRUCTURAL-RESILIENCE", reportCodes(containerReport), []string{})
	parsedContainer, parseErr := Parse(containerData)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	run.ClaimPrimary(claimSite("container", "import", "M25-STRUCTURAL-RESILIENCE", "GP5 input and GP8 output containers"), claimSite("container", "model", "M25-STRUCTURAL-RESILIENCE", "GP5 input and GP8 output containers"), claimSite("container", "export", "M25-STRUCTURAL-RESILIENCE", "GP5 input and GP8 output containers")).Normalized("Song.Version", parsedContainer.Version, Version{Data: gp8DocumentVersion, Number: [3]byte{8, 1, 3}})
	run.ClaimSerialization(claimSite("container", "export", "M25-STRUCTURAL-RESILIENCE", "GP5 input and GP8 output containers")).Wire("gpifDocument.GPVersion", extractGPIFLeafText(t, containerData)["GPIF/GPVersion"], gp8DocumentVersion)
	data, err := Export(conformanceExportSong(), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	for operation := range 7 {
		t.Run(fmt.Sprintf("malformed GPIF %d", operation), func(t *testing.T) {
			assertResilienceMalformedGPIF(t, data, []byte{byte(operation)})
		})
	}
	for _, shape := range [][4]int{{1, 1, 1, 1}, {2, 2, 2, 2}, {3, 1, 2, 3}} {
		t.Run(fmt.Sprintf("public Song %d-%d-%d-%d", shape[0], shape[1], shape[2], shape[3]), func(t *testing.T) {
			assertResilienceValidPublicSong(t, shape[0], shape[1], shape[2], shape[3])
		})
	}
}

func TestReusedGPIFNotesDoNotShareMutableEffects(t *testing.T) {
	song := conformanceExportSong()
	bend := &BendEffect{Points: []BendPoint{{Position: 0, Value: 1}, {Position: uint8(BendEffectMaxPosition), Value: 2}}}
	for measureIndex := range song.Tracks[0].Measures {
		for voiceIndex := range song.Tracks[0].Measures[measureIndex].Voices {
			for beatIndex := range song.Tracks[0].Measures[measureIndex].Voices[voiceIndex].Beats {
				for noteIndex := range song.Tracks[0].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
					song.Tracks[0].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex].Effect.Graces = nil
				}
			}
		}
	}
	for beatIndex := range song.Tracks[0].Measures[0].Voices[0].Beats[:2] {
		note := &song.Tracks[0].Measures[0].Voices[0].Beats[beatIndex].Notes[0]
		note.Effect.Bend = bend
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	data = rewriteConformanceGPIF(t, data, reuseFirstMainNoteReference)
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	beats := conformanceStaffBeatsWithNotes(&parsed.Tracks[0].Staves[0])
	if len(beats) < 2 || beats[0].Notes[0].Effect.Bend == nil || beats[1].Notes[0].Effect.Bend == nil {
		t.Fatalf("reused note effects = %#v", beats)
	}
	beats[0].Notes[0].Effect.Bend.Points[0].Value++
	if beats[0].Notes[0].Effect.Bend.Points[0].Value == beats[1].Notes[0].Effect.Bend.Points[0].Value {
		t.Fatal("independent note occurrences share mutable bend state")
	}
}

func TestCombinedEffectAndPercussionExportSeeds(t *testing.T) {
	pitched := conformanceExportSong()
	note := &pitched.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.TieOrigin = true
	note.Effect.Bend = &BendEffect{Points: []BendPoint{{Position: 0}, {Position: uint8(BendEffectMaxPosition), Value: 2}}}
	data, err := Export(pitched, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if _, parseErr := Parse(data); parseErr != nil {
		t.Fatal(parseErr)
	}
	pitchedRoundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotPitched := &pitchedRoundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	if !gotPitched.TieOrigin || gotPitched.Effect.Bend == nil || len(gotPitched.Effect.Graces) == 0 {
		t.Fatalf("grace + tie + bend = %#v", gotPitched)
	}

	percussion := parseTestFixture(t, "testdata/gp7/percussion.gp")
	percussionNote := &percussion.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	percussionNote.Kind = NoteTypeDead
	data, err = Export(percussion, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	if !got.HasPercussionArticulation || got.PercussionArticulation != percussionNote.PercussionArticulation || got.Kind != NoteTypeDead {
		t.Fatalf("percussion articulation + mute = %#v", got)
	}
}

func FuzzGPIFSemanticPreservingTransforms(f *testing.F) {
	data, err := Export(conformanceExportSong(), ExportFormatGP8)
	if err != nil {
		f.Fatal(err)
	}
	baseline, err := Parse(data)
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{0})
	f.Add([]byte{1, 0, 3})
	f.Add([]byte{4, 3, 2, 1, 0})
	f.Add([]byte{0x80})
	f.Add([]byte("11"))
	f.Fuzz(func(t *testing.T, plan []byte) {
		if len(plan) > 16 {
			t.Skip()
		}
		step := 0
		transformed := rewriteConformanceGPIF(t, data, func(gpif string) string {
			for _, operation := range plan {
				switch operation % 5 {
				case 0:
					gpif = reorderFirstNoteProperties(gpif)
				case 1:
					gpif = renameGPIFObjectIDs(gpif)
				case 2:
					gpif = reverseGPIFNoteDefinitions(gpif)
				case 3:
					gpif = duplicateFirstGPIFRhythmWithPrefix(gpif, fmt.Sprintf("fuzz-%d-", step))
				case 4:
					gpif = duplicateFirstGPIFNoteWithPrefix(gpif, fmt.Sprintf("fuzz-%d-", step))
				}
				step++
			}
			return gpif
		})
		if len(plan) > 0 && plan[0]&0x80 != 0 {
			transformed = repackGP8ForMetamorphicTest(t, transformed)
		}
		got, err := Parse(transformed)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(normalizeGoScore(got), normalizeGoScore(baseline)) {
			t.Fatal("valid transformed input changed semantics")
		}
	})
}

func FuzzParseMalformedClassified(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("PK\x03\x04"))
	f.Add([]byte("BCFZ"))
	valid, err := Export(conformanceExportSong(), ExportFormatGP8)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add(rewriteConformanceGPIF(f, valid, func(gpif string) string {
		return strings.Replace(gpif, `<Rhythm ref="0"`, `<Rhythm ref="missing"`, 1)
	}))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		result, err := ParseWithOptions(data, ParseOptions{Strict: true})
		if err == nil {
			if result == nil || result.Song == nil {
				t.Fatal("successful parse returned no song")
			}
			return
		}
		var parseErr *ParseError
		var strictErr *StrictParseError
		if !errors.As(err, &parseErr) && !errors.As(err, &strictErr) {
			t.Fatalf("unclassified parse outcome: %T %v", err, err)
		}
	})
}

func FuzzGPIFMalformedStructures(f *testing.F) {
	data, err := Export(conformanceExportSong(), ExportFormatGP8)
	if err != nil {
		f.Fatal(err)
	}
	for operation := range 7 {
		f.Add([]byte{byte(operation)})
	}
	f.Add([]byte{0, 2, 5})
	f.Fuzz(func(t *testing.T, plan []byte) {
		if len(plan) == 0 || len(plan) > 8 {
			t.Skip()
		}
		assertResilienceMalformedGPIF(t, data, plan)
	})
}

func FuzzValidPublicSongStructures(f *testing.F) {
	f.Add(uint8(1), uint8(1), uint8(1), uint8(1))
	f.Add(uint8(2), uint8(2), uint8(2), uint8(2))
	f.Add(uint8(3), uint8(1), uint8(2), uint8(3))
	f.Fuzz(func(t *testing.T, trackSeed, staffSeed, voiceSeed, beatSeed uint8) {
		trackCount := 1 + int(trackSeed%3)
		staffCount := 1 + int(staffSeed%3)
		voiceCount := 1 + int(voiceSeed%3)
		beatCount := 1 + int(beatSeed%3)
		assertResilienceValidPublicSong(t, trackCount, staffCount, voiceCount, beatCount)
	})
}

func assertResilienceMalformedGPIF(t *testing.T, data, plan []byte) {
	t.Helper()
	malformed := rewriteConformanceGPIF(t, data, func(gpif string) string {
		for _, operation := range plan {
			switch operation % 7 {
			case 0:
				gpif = strings.Replace(gpif, "<GPIF>", "<GPIF><M25UnknownElement/>", 1)
			case 1:
				gpif = strings.Replace(gpif, "<GPIF>", `<GPIF m25UnknownAttribute="true">`, 1)
			case 2:
				gpif = strings.Replace(gpif, `<Rhythm ref="0"`, `<Rhythm ref="m25-missing"`, 1)
			case 3:
				gpif = replaceFirstGPIFNoteID(gpif, "")
			case 4:
				gpif = duplicateSecondGPIFNoteID(gpif)
			case 5:
				gpif = regexp.MustCompile(`(<Property name="Fret">\s*<Fret>)[^<]*(</Fret>)`).ReplaceAllString(gpif, `${1}not-a-number${2}`)
			case 6:
				gpif = regexp.MustCompile(`<Hairpin>[^<]*</Hairpin>`).ReplaceAllString(gpif, `<Hairpin>M25UnknownHairpin</Hairpin>`)
			}
		}
		return gpif
	})
	result, parseErr := ParseWithOptions(malformed, ParseOptions{Strict: true})
	var strictErr *StrictParseError
	var formatErr *ParseError
	if errors.As(parseErr, &formatErr) {
		if result != nil {
			t.Fatalf("malformed structural plan %v returned a ParseError with result %#v", plan, result)
		}
		return
	}
	if !errors.As(parseErr, &strictErr) || result == nil || len(result.Diagnostics) == 0 {
		t.Fatalf("malformed structural plan %v returned result %#v and error %T %v", plan, result, parseErr, parseErr)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "" || diagnostic.Kind == "" || diagnostic.SourcePath == "" {
			t.Fatalf("malformed structural plan %v produced an unclassified diagnostic: %#v", plan, diagnostic)
		}
	}
}

func assertResilienceValidPublicSong(t *testing.T, trackCount, staffCount, voiceCount, beatCount int) {
	t.Helper()
	song := conformanceResilienceValidPublicSong(trackCount, staffCount, voiceCount, beatCount)
	if err := FinalizeSong(song); err != nil {
		t.Fatalf("finalizing %d/%d/%d/%d structure: %v", trackCount, staffCount, voiceCount, beatCount, err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("valid %d/%d/%d/%d structure diagnostics: %#v", trackCount, staffCount, voiceCount, beatCount, diagnostics)
	}
	assertResiliencePublicSongShape(t, song, trackCount, staffCount, voiceCount, beatCount)
	preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	allowedCodes := make([]string, 0, len(preflight.Entries))
	for _, entry := range preflight.Entries {
		if !slices.Contains(allowedCodes, entry.Code) {
			allowedCodes = append(allowedCodes, entry.Code)
		}
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        allowedCodes,
	}})
	if err != nil {
		t.Fatalf("exporting %d/%d/%d/%d structure: %v; report %#v", trackCount, staffCount, voiceCount, beatCount, err, report.Entries)
	}
	if !reflect.DeepEqual(report, preflight) {
		t.Fatalf("export report = %#v, want preflight %#v", report, preflight)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatalf("reimporting %d/%d/%d/%d structure: %v", trackCount, staffCount, voiceCount, beatCount, err)
	}
	assertResiliencePublicSongShape(t, roundTrip, trackCount, staffCount, voiceCount, beatCount)
}

func assertResiliencePublicSongShape(t *testing.T, song *Song, trackCount, staffCount, voiceCount, beatCount int) {
	t.Helper()
	if len(song.Tracks) != trackCount {
		t.Fatalf("track count = %d, want %d", len(song.Tracks), trackCount)
	}
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		if len(track.Staves) != staffCount {
			t.Fatalf("track %d staff count = %d, want %d", trackIndex, len(track.Staves), staffCount)
		}
		for staffIndex := range track.Staves {
			if len(track.Staves[staffIndex].Measures) != 1 {
				t.Fatalf("track %d staff %d measure count = %d, want 1", trackIndex, staffIndex, len(track.Staves[staffIndex].Measures))
			}
			measure := &track.Staves[staffIndex].Measures[0]
			if len(measure.Voices) != voiceCount {
				t.Fatalf("track %d staff %d voice count = %d, want %d", trackIndex, staffIndex, len(measure.Voices), voiceCount)
			}
			for voiceIndex := range measure.Voices {
				if len(measure.Voices[voiceIndex].Beats) != beatCount {
					t.Fatalf("track %d staff %d voice %d beat count = %d, want %d", trackIndex, staffIndex, voiceIndex, len(measure.Voices[voiceIndex].Beats), beatCount)
				}
				for beatIndex := range measure.Voices[voiceIndex].Beats {
					notes := measure.Voices[voiceIndex].Beats[beatIndex].Notes
					wantValue := int16((trackIndex + staffIndex + voiceIndex + beatIndex) % 12)
					if len(notes) != 1 || notes[0].Value != wantValue {
						t.Fatalf("track %d staff %d voice %d beat %d notes = %#v, want value %d", trackIndex, staffIndex, voiceIndex, beatIndex, notes, wantValue)
					}
				}
			}
		}
	}
}

func TestParseGP8RejectsOversizedGPIFBeforeInflatingIt(t *testing.T) {
	const limit = 16 << 20
	archive, err := writeGP8Archive([]struct {
		name   string
		method uint16
		data   []byte
	}{{
		name: "Content/score.gpif", method: zip.Deflate,
		data: append([]byte(strings.Repeat(" ", limit)), []byte(staffScopedChordGPIF)...),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(archive); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized GPIF error = %v, want explicit size-limit rejection", err)
	}
}

func reorderFirstNoteProperties(gpif string) string {
	start := strings.LastIndex(gpif, "<Notes>")
	if start < 0 {
		return gpif
	}
	properties := regexp.MustCompile(`(?s)(<Properties>\s*)(<Property.*?</Property>)(\s*)(<Property.*?</Property>)`)
	suffix := properties.ReplaceAllString(gpif[start:], `${1}${4}${3}${2}`)
	return gpif[:start] + suffix
}

func renameGPIFObjectIDs(gpif string) string {
	attribute := regexp.MustCompile(`\b(id|ref)="([^"]+)"`)
	gpif = attribute.ReplaceAllStringFunc(gpif, func(value string) string {
		parts := strings.SplitN(value, `="`, 2)
		id := strings.TrimSuffix(parts[1], `"`)
		if id == "-1" {
			return value
		}
		return parts[0] + `="x` + id + `"`
	})
	references := regexp.MustCompile(`<(Tracks|Bars|Voices|Beats|Notes|Chord)>([^<]*)</(Tracks|Bars|Voices|Beats|Notes|Chord)>`)
	return references.ReplaceAllStringFunc(gpif, func(value string) string {
		openEnd := strings.IndexByte(value, '>')
		closeStart := strings.LastIndex(value, "</")
		ids := strings.Fields(value[openEnd+1 : closeStart])
		for index, id := range ids {
			if id != "-1" {
				ids[index] = "x" + id
			}
		}
		return value[:openEnd+1] + strings.Join(ids, " ") + value[closeStart:]
	})
}

func reverseGPIFNoteDefinitions(gpif string) string {
	start := strings.LastIndex(gpif, "<Notes>")
	if start < 0 {
		return gpif
	}
	end := strings.Index(gpif[start:], "</Notes>")
	if end < 0 {
		return gpif
	}
	end += start
	note := regexp.MustCompile(`(?s)<Note id="[^"]+">.*?</Note>`)
	items := note.FindAllString(gpif[start:end], -1)
	index := len(items) - 1
	reordered := note.ReplaceAllStringFunc(gpif[start:end], func(string) string {
		item := items[index]
		index--
		return item
	})
	return gpif[:start] + reordered + gpif[end:]
}

func duplicateFirstGPIFRhythm(gpif string) string {
	return duplicateFirstGPIFRhythmWithPrefix(gpif, "duplicate-")
}

func duplicateFirstGPIFRhythmWithPrefix(gpif, prefix string) string {
	rhythm := regexp.MustCompile(`(?s)<Rhythm id="([^"]+)">.*?</Rhythm>`)
	match := rhythm.FindStringSubmatch(gpif)
	if len(match) == 0 {
		return gpif
	}
	duplicateID := prefix + match[1]
	duplicate := strings.Replace(match[0], `id="`+match[1]+`"`, `id="`+duplicateID+`"`, 1)
	gpif = strings.Replace(gpif, "</Rhythms>", duplicate+"</Rhythms>", 1)
	return strings.Replace(gpif, `<Rhythm ref="`+match[1]+`"`, `<Rhythm ref="`+duplicateID+`"`, 1)
}

func duplicateFirstGPIFNoteWithPrefix(gpif, prefix string) string {
	note := regexp.MustCompile(`(?s)<Note id="([^"]+)">.*?</Note>`)
	match := note.FindStringSubmatch(gpif)
	if len(match) == 0 {
		return gpif
	}
	duplicateID := prefix + match[1]
	duplicate := strings.Replace(match[0], `id="`+match[1]+`"`, `id="`+duplicateID+`"`, 1)
	definitionsEnd := strings.LastIndex(gpif, "</Notes>")
	if definitionsEnd < 0 {
		return gpif
	}
	gpif = gpif[:definitionsEnd] + duplicate + gpif[definitionsEnd:]
	references := regexp.MustCompile(`<Notes>([^<]*)</Notes>`)
	return references.ReplaceAllStringFunc(gpif, func(value string) string {
		openEnd := strings.IndexByte(value, '>')
		closeStart := strings.LastIndex(value, "</")
		ids := strings.Fields(value[openEnd+1 : closeStart])
		for index, id := range ids {
			if id == match[1] {
				ids[index] = duplicateID
			}
		}
		return value[:openEnd+1] + strings.Join(ids, " ") + value[closeStart:]
	})
}

func replaceFirstGPIFNoteID(gpif, replacement string) string {
	note := regexp.MustCompile(`<Note id="[^"]+">`)
	location := note.FindStringIndex(gpif)
	if location == nil {
		return gpif
	}
	return gpif[:location[0]] + `<Note id="` + replacement + `">` + gpif[location[1]:]
}

func duplicateSecondGPIFNoteID(gpif string) string {
	note := regexp.MustCompile(`<Note id="([^"]+)">`)
	matches := note.FindAllStringSubmatchIndex(gpif, 2)
	if len(matches) < 2 {
		return gpif
	}
	firstID := gpif[matches[0][2]:matches[0][3]]
	secondIDStart, secondIDEnd := matches[1][2], matches[1][3]
	return gpif[:secondIDStart] + firstID + gpif[secondIDEnd:]
}

func conformanceResilienceValidPublicSong(trackCount, staffCount, voiceCount, beatCount int) *Song {
	song := &Song{Tempo: 120, MeasureHeaders: []MeasureHeader{defaultMeasureHeader()}}
	for trackIndex := range trackCount {
		track := defaultTrack()
		track.Number = int32(trackIndex + 1)
		track.Name = fmt.Sprintf("Track %d", trackIndex+1)
		track.Measures = nil
		track.Staves = make([]Staff, staffCount)
		for staffIndex := range staffCount {
			measure := Measure{HeaderIndex: 0, Voices: make([]Voice, voiceCount)}
			for voiceIndex := range voiceCount {
				beats := make([]Beat, beatCount)
				for beatIndex := range beatCount {
					beat := defaultBeat()
					note := defaultNote()
					note.Kind = NoteTypeNormal
					note.String = 1
					note.Value = int16((trackIndex + staffIndex + voiceIndex + beatIndex) % 12)
					beat.Notes = []Note{note}
					beats[beatIndex] = beat
				}
				measure.Voices[voiceIndex] = Voice{Beats: beats}
			}
			track.Staves[staffIndex] = Staff{
				Strings:                   append([]GuitarString(nil), track.Strings...),
				Measures:                  []Measure{measure},
				StandardNotationLineCount: 5,
			}
		}
		song.Tracks = append(song.Tracks, track)
	}
	return song
}

func reuseFirstMainNoteReference(gpif string) string {
	reference := regexp.MustCompile(`<Notes>([^< ]+)([^<]*)</Notes>`)
	matches := reference.FindAllStringSubmatchIndex(gpif, -1)
	if len(matches) < 2 {
		return gpif
	}
	firstID := gpif[matches[0][2]:matches[0][3]]
	secondStart, secondEnd := matches[1][2], matches[1][3]
	return gpif[:secondStart] + firstID + gpif[secondEnd:]
}

func repackGP8ForMetamorphicTest(t *testing.T, data []byte) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entries := make([]struct {
		name   string
		method uint16
		data   []byte
	}, 0, len(reader.File))
	for index := len(reader.File) - 1; index >= 0; index-- {
		file := reader.File[index]
		input, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		method := zip.Store
		if index%2 == 0 && !strings.HasSuffix(file.Name, "/") {
			method = zip.Deflate
		}
		entries = append(entries, struct {
			name   string
			method uint16
			data   []byte
		}{file.Name, method, contents})
	}
	result, err := writeGP8Archive(entries)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
