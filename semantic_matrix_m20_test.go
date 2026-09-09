// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestSemanticMatrixM20SourceAudit(t *testing.T) {
	runSemanticMatrixM20SourceAudit(newSemanticMatrixRun(t))
}

func TestSemanticMatrixM20BehaviorReconciliation(t *testing.T) {
	runSemanticMatrixM20BehaviorReconciliation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM20SourceAudit(run *semanticMatrixRun) {
	t := run.t
	ledger := readSemanticContractLedger(t)
	inventory := discoverSemanticGoInventory(t)

	decoded := m20RoundTripWireDocument(t)
	for _, field := range inventory.wireFields {
		run.Wire(field, slices.Contains(decoded, field), true)
	}
	for _, contract := range ledger.SemanticContracts.SourceDispatches {
		key := semanticDispatchKey(contract.Function, contract.Selector)
		got := append([]string(nil), inventory.dispatches[key]...)
		want := make([]string, 0, len(contract.Cases))
		for value := range contract.Cases {
			want = append(want, value)
		}
		sort.Strings(got)
		sort.Strings(want)
		run.Dispatch(key, got, want)
	}

	tests := []struct {
		name       string
		mutate     func(string) string
		code       string
		kind       ParseDiagnosticKind
		path       string
		objectID   string
		strictKind ParseDiagnosticKind
	}{
		{
			name: "conflicting duplicate note property",
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="Fret"><Fret>31</Fret></Property>`)
			},
			code: "GPIF.Note.Property.ConflictingDuplicate", kind: ParseDiagnosticInvalidData,
			path: "Property[@name=\"Fret\"]", strictKind: ParseDiagnosticInvalidData,
		},
		{
			name: "unknown audio engine state",
			mutate: func(source string) string {
				return m20ReplaceElementTextAfter(t, source, "<Tracks>", "AudioEngineState", "legacy-render-state")
			},
			code: "GPIF.Track.AudioEngineState.InvalidValue", kind: ParseDiagnosticUnknownSyntax,
			path: "/AudioEngineState", strictKind: ParseDiagnosticUnknownSyntax,
		},
		{
			name: "sound channel without public destination",
			mutate: func(source string) string {
				return m20ReplaceElementTextAfter(t, source, "<Sounds>", "PrimaryChannel", "0")
			},
			code: "GPIF.Track.Sound.Channel", kind: ParseDiagnosticUnsupportedFeature,
			path: "/MIDI/PrimaryChannel", strictKind: ParseDiagnosticUnsupportedFeature,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, test.mutate)
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			diagnostic := m20DiagnosticByCode(result.Diagnostics, test.code)
			if diagnostic == nil {
				t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, test.code)
			}
			if diagnostic.Kind != test.kind || diagnostic.Format != "GP8" {
				t.Errorf("diagnostic = %#v, want kind %s and GP8", diagnostic, test.kind)
			}
			if !strings.Contains(diagnostic.SourcePath, test.path) || diagnostic.ObjectID == "" || diagnostic.Reason == "" {
				t.Errorf("diagnostic = %#v, want a useful path, object ID, and reason", diagnostic)
			}

			selected, selectedErr := ParseWithOptions(data, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{test.strictKind}})
			var strictErr *StrictParseError
			if selected == nil || !errors.As(selectedErr, &strictErr) || m20DiagnosticByCode(strictErr.Diagnostics, test.code) == nil {
				t.Fatalf("selected strict parse = %#v, %v, want %s refusal", selected, selectedErr, test.code)
			}
			if test.kind == ParseDiagnosticDeliberateIgnore {
				defaultResult, defaultErr := ParseWithOptions(data, ParseOptions{Strict: true})
				if defaultErr != nil {
					t.Fatalf("default strict mode rejected deliberate ignore: %v; diagnostics = %#v", defaultErr, defaultResult.Diagnostics)
				}
			}
		})
	}

	rseData := diagnosticGP8Fixture(t, func(source string) string {
		return m20ReplaceElementTextAfter(t, source, "<Tracks>", "AudioEngineState", "RSE")
	})
	rseResult, err := ParseWithOptions(rseData, ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if !rseResult.Song.Tracks[0].UseRse || m20DiagnosticByCode(rseResult.Diagnostics, "GPIF.Track.AudioEngineState.InvalidValue") != nil {
		t.Fatalf("RSE engine state = UseRse %t, diagnostics %#v", rseResult.Song.Tracks[0].UseRse, rseResult.Diagnostics)
	}

	missingChannel := diagnosticGP8Fixture(t, func(source string) string {
		return m20RemoveElementAfter(t, source, "<Sounds>", "PrimaryChannel")
	})
	missingResult, err := ParseWithOptions(missingChannel, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic := m20DiagnosticByCode(missingResult.Diagnostics, "GPIF.Track.Sound.Channel"); diagnostic != nil {
		t.Fatalf("missing sound channel was treated as explicit zero: %#v", diagnostic)
	}

	m20AssertMalformedDiagnostics(t)
	m20AssertReusedSourceDefinitions(t)
	m20AssertBinaryDiagnosticLocations(t)
}

func runSemanticMatrixM20BehaviorReconciliation(run *semanticMatrixRun) {
	t := run.t

	tempoSong := &Song{Tempo: 120}
	gpifReadTempoAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: "90 2"}}}, tempoSong, nil)
	run.Wire("gpifAutomationValue.Text", tempoSong.TempoAutomations[0].Tempo, float64(90))

	channel := defaultMidiChannel()
	gpifApplyChannelStrip("0 0 0 0 0 0 0 0 0 0 0 0.25 0.75", &channel)
	run.Wire("gpifChannelStrip.Parameters", []int8{channel.Balance, channel.Volume}, []int8{32, 95})

	direction, strength := "Up", "Wide"
	beat := Beat{}
	gpifApplyBeatEffects(&gpifBeat{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Brush", Direction: &direction}, {Name: "VibratoWTremBar", Strength: &strength}}}}, &beat)
	run.Wire("gpifProperty.Direction", beat.Effect.Stroke.Direction, BeatStrokeDirectionUp)
	run.Wire("gpifProperty.Strength", beat.Effect.Vibrato, true)

	context := &parseContext{format: "GP8"}
	gpifAuditBeatProperty(context, "beat-id", "/GPIF/Beats/Beat", gpifProperty{Name: "FutureBeatProperty"})
	run.Dispatch("gpifAuditBeatProperty:property.Name", m20DiagnosticByCode(context.diagnostics, "GPIF.Beat.Property.Unknown") != nil, true)

	element := "Element"
	staff := gpifStaff{Properties: []gpifStaffProperty{{Name: "Tuning", Pitches: "40 45", Label: "Drop D"}}}
	track := gpifTrack{
		ID: "track-id", AudioEngineState: "FutureEngine",
		Instrument: &gpifInstrument{Ref: "drmkt"},
		Transpose:  &gpifTranspose{Chromatic: 2, Octave: -1},
		Staves:     gpifStaves{Staff: []gpifStaff{staff}},
	}
	doc := gpifDocument{
		Tracks: gpifTracks{Tracks: []gpifTrack{track}},
		Beats: gpifBeats{Beats: []gpifBeat{{
			ID: "beat-id", Tremolo: "future-rate", Wah: "Open", Fadding: "FadeOut",
			Properties: gpifProperties{Properties: []gpifProperty{{Name: "FutureBeatProperty"}}},
		}}},
		Notes: gpifNotes{Notes: []gpifNote{{
			ID: "note-id", Properties: gpifProperties{Properties: []gpifProperty{{Name: element}}},
		}}},
	}
	gpifAuditDiagnostics(doc, context)
	run.Wire("gpifBeat.ID", m20DiagnosticByCode(context.diagnostics, "GPIF.Beat.Wah").ObjectID, "beat-id")
	run.Wire("gpifBeat.Tremolo", m20DiagnosticByCode(context.diagnostics, "GPIF.Beat.Tremolo.InvalidValue") != nil, true)
	run.Wire("gpifBeat.Wah", m20DiagnosticByCode(context.diagnostics, "GPIF.Beat.Wah") != nil, true)
	run.Wire("gpifInstrument.Ref", doc.Tracks.Tracks[0].isPercussionTrack(), true)
	run.Wire("gpifStaffProperty.Label", m20DiagnosticByCode(context.diagnostics, "GPIF.Staff.Property.Tuning.Label") != nil, true)
	run.Wire("gpifTranspose.Chromatic", m20DiagnosticByCode(context.diagnostics, "GPIF.Track.Transpose") != nil, true)
	run.Wire("gpifTranspose.Octave", m20DiagnosticByCode(context.diagnostics, "GPIF.Track.Transpose") != nil, true)
	run.Dispatch("gpifAuditDiagnostics:beat.Fadding", m20DiagnosticByCode(context.diagnostics, "GPIF.Beat.Fadding.Lossy") != nil, true)
	run.Dispatch("gpifAuditDiagnostics:property.Name", m20DiagnosticByCode(context.diagnostics, "GPIF.Note.Property.Element") != nil, true)
	run.Dispatch("gpifAuditDiagnostics:track.AudioEngineState", m20DiagnosticByCode(context.diagnostics, "GPIF.Track.AudioEngineState.InvalidValue") != nil, true)

	xmlContext := &parseContext{format: "GP8"}
	if err := gpifAuditXML([]byte("<GPIF><FutureRootChild/></GPIF>"), xmlContext); err != nil {
		t.Fatal(err)
	}
	run.Dispatch("gpifXMLAuditStart:element.Name.Local", m20DiagnosticByCode(xmlContext.diagnostics, "GPIF.UnknownElement.NoteAndBeat") != nil, true)

	metadataXML := strings.Replace(semanticM03OwnershipGPIF, "<GPVersion>8.0</GPVersion>", `<GPVersion>8.0</GPVersion><GPRevision required="12000" recommended="13000">14000</GPRevision><Encoding><EncodingDescription>producer-x</EncodingDescription></Encoding>`, 1)
	var metadataDoc gpifDocument
	if err := xml.Unmarshal([]byte(metadataXML), &metadataDoc); err != nil {
		t.Fatal(err)
	}
	run.Wire("gpifEncoding.Description", metadataDoc.Encoding.Description, "producer-x")
	run.Wire("gpifRevision.Required", metadataDoc.GPRevision.Required, "12000")
	run.Wire("gpifRevision.Recommended", metadataDoc.GPRevision.Recommended, "13000")
	run.Wire("gpifRevision.Value", strings.TrimSpace(metadataDoc.GPRevision.Value), "14000")

	pickupXML := strings.Replace(semanticM03OwnershipGPIF, "<MasterTrack>", "<MasterTrack><Anacrusis/>", 1)
	pickup, err := parseGPIF([]byte(pickupXML))
	if err != nil {
		t.Fatal(err)
	}
	run.Wire("gpifMasterTrack.Anacrusis", pickup.Anacrusis, true)
	run.Wire("gpifRhythm.ID", pickup.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Duration.Value, uint16(DurationQuarter))
}

func m20AssertMalformedDiagnostics(t *testing.T) {
	t.Helper()
	tests := []struct {
		name     string
		mutate   func(string) string
		code     string
		kind     ParseDiagnosticKind
		path     string
		objectID bool
	}{
		{
			name: "unknown element",
			mutate: func(source string) string {
				return insertFirstGPIFObjectChild(t, source, "<Notes>", "</Note>", "<FutureTechnique/>")
			},
			code: "GPIF.UnknownElement.NoteAndBeat", kind: ParseDiagnosticUnknownSyntax, path: "/FutureTechnique", objectID: true,
		},
		{
			name: "unknown attribute",
			mutate: func(source string) string {
				start := strings.LastIndex(source, "<Notes>")
				return source[:start] + strings.Replace(source[start:], `<Note id="`, `<Note future="m20" id="`, 1)
			},
			code: "GPIF.UnknownAttribute.NoteAndBeat", kind: ParseDiagnosticUnknownSyntax, path: "/@future", objectID: true,
		},
		{
			name: "unknown enum",
			mutate: func(source string) string {
				return insertFirstGPIFObjectChild(t, source, "<Beats>", "</Beat>", "<Dynamic>FutureDynamic</Dynamic>")
			},
			code: "GPIF.Beat.Dynamic.InvalidValue", kind: ParseDiagnosticUnsupportedFeature, path: "/Dynamic", objectID: true,
		},
		{
			name: "unknown flag",
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="Slide"><Flags>256</Flags></Property>`)
			},
			code: "GPIF.Note.Property.Slide.UnknownFlags", kind: ParseDiagnosticUnsupportedFeature, path: "/Flags", objectID: true,
		},
		{
			name: "unknown dispatch type",
			mutate: func(source string) string {
				return strings.Replace(source, "<Type>Tempo</Type>", "<Type>FutureAutomation</Type>", 1)
			},
			code: "GPIF.MasterTrack.Automation.Type.Unknown", kind: ParseDiagnosticUnknownSyntax, path: "/Type",
		},
		{
			name: "missing payload",
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="Muted"/>`)
			},
			code: "GPIF.Note.Property.Muted.MissingPayload", kind: ParseDiagnosticInvalidData, path: "Muted", objectID: true,
		},
		{
			name: "invalid numeric text",
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="Slide"><Flags>not-a-number</Flags></Property>`)
			},
			code: "GPIF.Note.Property.Slide.InvalidFlags", kind: ParseDiagnosticInvalidData, path: "/Flags", objectID: true,
		},
		{
			name: "non-finite number",
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="HarmonicFret"><HFret>NaN</HFret></Property>`)
			},
			code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData, path: "/HFret", objectID: true,
		},
		{
			name: "infinite number",
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="HarmonicFret"><HFret>+Inf</HFret></Property>`)
			},
			code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData, path: "/HFret", objectID: true,
		},
		{
			name:   "duplicate ID",
			mutate: func(source string) string { return m20DuplicateFirstNote(t, source) },
			code:   "GPIF.Note.DuplicateID", kind: ParseDiagnosticInvalidData, path: "/Notes/Note", objectID: true,
		},
		{
			name: "empty ID",
			mutate: func(source string) string {
				start := strings.LastIndex(source, "<Notes>")
				return source[:start] + strings.Replace(source[start:], `<Note id="0"`, `<Note id=""`, 1)
			},
			code: "GPIF.Note.EmptyID", kind: ParseDiagnosticInvalidData, path: "/Notes/Note",
		},
		{
			name: "dangling reference",
			mutate: func(source string) string {
				return replaceFirstGPIFReference(t, source, `<Rhythm ref="`, "missing-rhythm")
			},
			code: "GPIF.Beat.Rhythm.Reference", kind: ParseDiagnosticInvalidData, path: "/Rhythm", objectID: true,
		},
		{
			name:   "recognized element in wrong scope",
			mutate: func(source string) string { return strings.Replace(source, "<Score>", "<Score><Fret>3</Fret>", 1) },
			code:   "GPIF.UnknownElement.ScoreCore", kind: ParseDiagnosticUnknownSyntax, path: "/Score/Fret",
		},
	}

	for _, test := range tests {
		t.Run("malformed/"+test.name, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, test.mutate)
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			diagnostic := m20DiagnosticByCode(result.Diagnostics, test.code)
			if diagnostic == nil {
				t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, test.code)
			}
			if diagnostic.Kind != test.kind || !strings.Contains(diagnostic.SourcePath, test.path) || diagnostic.Reason == "" {
				t.Errorf("diagnostic = %#v, want %s at %s", diagnostic, test.kind, test.path)
			}
			if test.objectID && diagnostic.ObjectID == "" {
				t.Errorf("diagnostic = %#v, want object ID", diagnostic)
			}
			strictResult, strictErr := ParseWithOptions(data, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{test.kind}})
			var policyErr *StrictParseError
			if strictResult == nil || !errors.As(strictErr, &policyErr) {
				t.Fatalf("strict parse = %#v, %v, want selected refusal", strictResult, strictErr)
			}
			for _, rejected := range policyErr.Diagnostics {
				if rejected.Kind != test.kind {
					t.Errorf("selected strict kind %s also rejected %#v", test.kind, rejected)
				}
			}
		})
	}
}

func m20AssertReusedSourceDefinitions(t *testing.T) {
	t.Helper()
	result, err := ParseWithOptions(conformanceGPIFArchive(t, semanticM03OwnershipGPIF), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	first := &result.Song.Tracks[2].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	second := &result.Song.Tracks[2].Staves[1].Measures[0].Voices[0].Beats[0].Notes[0]
	first.Value = 17
	if second.Value != 6 {
		t.Fatalf("reused source note shares mutable occurrences: second value = %d", second.Value)
	}
}

func m20AssertBinaryDiagnosticLocations(t *testing.T) {
	t.Helper()
	result, err := ParseFileWithOptions("testdata/gp5/other-effects.gp5", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code != "" && diagnostic.Format == "GP5" && diagnostic.SourcePath == "" && diagnostic.BinaryOffset != nil && *diagnostic.BinaryOffset >= 0
	}) {
		t.Fatalf("binary diagnostics = %#v, want stable code and non-negative offset", result.Diagnostics)
	}
}

func m20RoundTripWireDocument(t *testing.T) []string {
	t.Helper()
	original := reflect.New(reflect.TypeOf(gpifDocument{})).Elem()
	sequence := int64(0)
	m20FillWireValue(original, &sequence)
	original.FieldByName("XMLName").Set(reflect.ValueOf(xml.Name{Local: "GPIF"}))
	data, err := xml.Marshal(original.Interface())
	if err != nil {
		t.Fatal(err)
	}
	var decoded gpifDocument
	if err := xml.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original.Interface(), reflect.ValueOf(decoded).Interface()) {
		t.Fatalf("GPIF wire round trip changed decoded content\noriginal: %#v\ndecoded:  %#v", original.Interface(), decoded)
	}
	var fields []string
	m20CollectDecodedWireFields(reflect.ValueOf(decoded), &fields)
	sort.Strings(fields)
	return fields
}

func m20FillWireValue(value reflect.Value, sequence *int64) {
	if !value.CanSet() {
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(xml.Name{}) {
			value.Set(reflect.ValueOf(xml.Name{Local: "M20"}))
			return
		}
		for index := range value.NumField() {
			m20FillWireValue(value.Field(index), sequence)
		}
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		m20FillWireValue(value.Elem(), sequence)
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
		m20FillWireValue(value.Index(0), sequence)
	case reflect.String:
		*sequence++
		value.SetString(fmt.Sprintf("M20-%d", *sequence))
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*sequence++
		value.SetInt(*sequence)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		*sequence++
		value.SetUint(uint64(*sequence))
	case reflect.Float32, reflect.Float64:
		*sequence++
		value.SetFloat(float64(*sequence) + 0.25)
	}
}

func m20CollectDecodedWireFields(value reflect.Value, fields *[]string) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.Slice {
		if value.Len() != 0 {
			m20CollectDecodedWireFields(value.Index(0), fields)
		}
		return
	}
	if value.Kind() != reflect.Struct || value.Type() == reflect.TypeOf(xml.Name{}) {
		return
	}
	for index := range value.NumField() {
		field := value.Type().Field(index)
		fieldValue := value.Field(index)
		name := value.Type().Name() + "." + field.Name
		if !fieldValue.IsZero() {
			*fields = append(*fields, name)
		}
		m20CollectDecodedWireFields(fieldValue, fields)
	}
}

func m20DiagnosticByCode(diagnostics []ParseDiagnostic, code string) *ParseDiagnostic {
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	return nil
}

func m20ReplaceElementTextAfter(t *testing.T, source, anchor, element, replacement string) string {
	t.Helper()
	anchorIndex := strings.Index(source, anchor)
	if anchorIndex < 0 {
		t.Fatalf("GPIF has no %s", anchor)
	}
	open := "<" + element + ">"
	close := "</" + element + ">"
	start := strings.Index(source[anchorIndex:], open)
	if start < 0 {
		t.Fatalf("GPIF after %s has no %s", anchor, element)
	}
	start += anchorIndex + len(open)
	end := strings.Index(source[start:], close)
	if end < 0 {
		t.Fatalf("GPIF %s has no closing element", element)
	}
	end += start
	return source[:start] + replacement + source[end:]
}

func m20DuplicateFirstNote(t *testing.T, source string) string {
	t.Helper()
	collection := strings.LastIndex(source, "<Notes>")
	if collection < 0 {
		t.Fatal("GPIF has no Notes collection")
	}
	start := strings.Index(source[collection:], "<Note ")
	if start < 0 {
		t.Fatal("GPIF has no Note")
	}
	start += collection
	end := strings.Index(source[start:], "</Note>")
	if end < 0 {
		t.Fatal("GPIF Note has no closing element")
	}
	end += start + len("</Note>")
	return source[:end] + source[start:end] + source[end:]
}

func m20RemoveElementAfter(t *testing.T, source, anchor, element string) string {
	t.Helper()
	anchorIndex := strings.Index(source, anchor)
	if anchorIndex < 0 {
		t.Fatalf("GPIF has no %s", anchor)
	}
	open := "<" + element + ">"
	close := "</" + element + ">"
	start := strings.Index(source[anchorIndex:], open)
	if start < 0 {
		t.Fatalf("GPIF after %s has no %s", anchor, element)
	}
	start += anchorIndex
	end := strings.Index(source[start:], close)
	if end < 0 {
		t.Fatalf("GPIF %s has no closing element", element)
	}
	end += start + len(close)
	return source[:start] + source[end:]
}
