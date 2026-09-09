// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"reflect"
	"regexp"
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
	note.Effect.Graces[0].Transition = GraceEffectTransitionBend
	data, err := Export(pitched, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if _, parseErr := Parse(data); parseErr != nil {
		t.Fatal(parseErr)
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
	f.Add(byte(0))
	f.Add(byte(1))
	f.Add(byte(2))
	f.Add(byte(3))
	f.Fuzz(func(t *testing.T, selector byte) {
		data, err := Export(conformanceExportSong(), ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		baseline, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		transforms := []func(string) string{
			reorderFirstNoteProperties, renameGPIFObjectIDs, reverseGPIFNoteDefinitions, duplicateFirstGPIFRhythm,
		}
		transformed := rewriteConformanceGPIF(t, data, transforms[int(selector)%len(transforms)])
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
	rhythm := regexp.MustCompile(`(?s)<Rhythm id="([^"]+)">.*?</Rhythm>`)
	match := rhythm.FindStringSubmatch(gpif)
	if len(match) == 0 {
		return gpif
	}
	duplicateID := "duplicate-" + match[1]
	duplicate := strings.Replace(match[0], `id="`+match[1]+`"`, `id="`+duplicateID+`"`, 1)
	gpif = strings.Replace(gpif, "</Rhythms>", duplicate+"</Rhythms>", 1)
	return strings.Replace(gpif, `<Rhythm ref="`+match[1]+`"`, `<Rhythm ref="`+duplicateID+`"`, 1)
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
