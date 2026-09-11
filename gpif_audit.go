// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type gpifXMLSchemaNode struct {
	children   map[string]*gpifXMLSchemaNode
	attributes map[string]struct{}
}

var gpifXMLSchema = gpifBuildXMLSchema()

func gpifBuildXMLSchema() *gpifXMLSchemaNode {
	root := &gpifXMLSchemaNode{children: make(map[string]*gpifXMLSchemaNode)}
	document := &gpifXMLSchemaNode{}
	root.children["GPIF"] = document
	gpifPopulateXMLSchema(document, reflect.TypeOf(gpifDocument{}))
	return root
}

func gpifPopulateXMLSchema(node *gpifXMLSchemaNode, valueType reflect.Type) {
	for valueType.Kind() == reflect.Pointer || valueType.Kind() == reflect.Slice {
		valueType = valueType.Elem()
	}
	if valueType.Kind() != reflect.Struct || valueType == reflect.TypeOf(xml.Name{}) {
		return
	}
	for fieldIndex := range valueType.NumField() {
		field := valueType.Field(fieldIndex)
		tag := field.Tag.Get("xml")
		if tag == "-" || field.Type == reflect.TypeOf(xml.Name{}) {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = field.Name
		}
		if slices.Contains(parts[1:], "attr") {
			if node.attributes == nil {
				node.attributes = make(map[string]struct{})
			}
			node.attributes[name] = struct{}{}
			continue
		}
		if slices.Contains(parts[1:], "chardata") || slices.Contains(parts[1:], "innerxml") {
			continue
		}
		current := node
		path := strings.Split(name, ">")
		for _, element := range path {
			if current.children == nil {
				current.children = make(map[string]*gpifXMLSchemaNode)
			}
			child := current.children[element]
			if child == nil {
				child = &gpifXMLSchemaNode{}
				current.children[element] = child
			}
			current = child
		}
		gpifPopulateXMLSchema(current, field.Type)
	}
}

type gpifXMLAuditFrame struct {
	schema   *gpifXMLSchemaNode
	path     string
	location ParseLocation
	objectID string
	unknown  bool
}

func gpifAuditXML(data []byte, context *parseContext) error {
	if context == nil {
		return nil
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	stack := []gpifXMLAuditFrame{{schema: gpifXMLSchema}}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			parent := stack[len(stack)-1]
			frame := gpifXMLAuditStart(context, parent, value)
			stack = append(stack, frame)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
}

func gpifXMLAuditStart(context *parseContext, parent gpifXMLAuditFrame, element xml.StartElement) gpifXMLAuditFrame {
	path := parent.path + "/" + element.Name.Local
	location := parent.location
	objectID := parent.objectID
	for _, attribute := range element.Attr {
		if attribute.Name.Local != "id" {
			continue
		}
		objectID = attribute.Value
		switch element.Name.Local {
		case "Track":
			location.TrackID = attribute.Value
		case "Bar":
			location.BarID = attribute.Value
		case "Voice":
			location.VoiceID = attribute.Value
		case "Beat":
			location.BeatID = attribute.Value
		case "Note":
			location.NoteID = attribute.Value
		}
		path += fmt.Sprintf("[@id=%q]", attribute.Value)
		break
	}
	if element.Name.Local == "Property" {
		for _, attribute := range element.Attr {
			if attribute.Name.Local == "name" {
				path += fmt.Sprintf("[@name=%q]", attribute.Value)
				break
			}
		}
	}

	frame := gpifXMLAuditFrame{path: path, location: location, objectID: objectID, unknown: parent.unknown}
	if !parent.unknown && parent.schema != nil {
		frame.schema = parent.schema.children[element.Name.Local]
		if frame.schema == nil {
			frame.unknown = true
			feature := gpifDiagnosticFeature(path)
			source := diagnosticSource("GPIF.UnknownElement.NoteAndBeat", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax)
			switch feature {
			case "rhythm":
				source = diagnosticSource("GPIF.UnknownElement.Rhythm", "rhythm", ParseDiagnosticUnknownSyntax)
			case "staff-ownership":
				source = diagnosticSource("GPIF.UnknownElement.StaffOwnership", "staff-ownership", ParseDiagnosticUnknownSyntax)
			case "score-core":
				source = diagnosticSource("GPIF.UnknownElement.ScoreCore", "score-core", ParseDiagnosticUnknownSyntax)
			}
			context.add(source, ParseDiagnostic{
				Kind: ParseDiagnosticUnknownSyntax, SourcePath: path, ObjectID: objectID,
				Location: location, Feature: feature,
				Reason: fmt.Sprintf("unknown GPIF element %q", element.Name.Local),
			})
		}
	}
	if frame.schema != nil && !frame.unknown {
		for _, attribute := range element.Attr {
			if attribute.Name.Space == "xmlns" || attribute.Name.Local == "xmlns" {
				continue
			}
			if _, exists := frame.schema.attributes[attribute.Name.Local]; !exists {
				feature := gpifDiagnosticFeature(path)
				source := diagnosticSource("GPIF.UnknownAttribute.NoteAndBeat", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax)
				switch feature {
				case "rhythm":
					source = diagnosticSource("GPIF.UnknownAttribute.Rhythm", "rhythm", ParseDiagnosticUnknownSyntax)
				case "staff-ownership":
					source = diagnosticSource("GPIF.UnknownAttribute.StaffOwnership", "staff-ownership", ParseDiagnosticUnknownSyntax)
				case "score-core":
					source = diagnosticSource("GPIF.UnknownAttribute.ScoreCore", "score-core", ParseDiagnosticUnknownSyntax)
				}
				context.add(source, ParseDiagnostic{
					Kind: ParseDiagnosticUnknownSyntax, SourcePath: path + "/@" + attribute.Name.Local,
					ObjectID: objectID, Location: location, Feature: feature,
					Reason: fmt.Sprintf("unknown GPIF attribute %q", attribute.Name.Local),
				})
			}
		}
	}
	return frame
}

func gpifDiagnosticFeature(path string) string {
	switch {
	case strings.Contains(path, "/Rhythms/") || strings.HasSuffix(path, "/Rhythm"):
		return "rhythm"
	case strings.Contains(path, "/Diagram") || strings.Contains(path, "/Chord"):
		return "note-and-beat-semantics"
	case strings.Contains(path, "/Tracks/") || strings.Contains(path, "/Bars/") || strings.Contains(path, "/MasterBars/"):
		return "staff-ownership"
	case path == "/GPIF" || strings.Contains(path, "/Score") || strings.Contains(path, "/MasterTrack") || strings.Contains(path, "/Assets") || strings.Contains(path, "/BackingTrack"):
		return "score-core"
	default:
		return "note-and-beat-semantics"
	}
}

func gpifAuditDiagnostics(doc gpifDocument, context *parseContext) {
	if context == nil {
		return
	}
	if doc.BackingTrack != nil && doc.BackingTrack.Enabled && strings.EqualFold(doc.BackingTrack.Source, "Local") {
		if strings.TrimSpace(doc.BackingTrack.AssetID) == "" {
			context.add(gpifBackingTrackAssetReferenceSource, ParseDiagnostic{
				SourcePath: "/GPIF/BackingTrack/AssetId",
				Reason:     "backing track has no asset reference",
			})
		}
	}

	for _, track := range doc.Tracks.Tracks {
		path := gpifObjectPath("Tracks/Track", track.ID)
		switch track.AudioEngineState {
		case "", "MIDI", "RSE":
		default:
			context.add(diagnosticSource("GPIF.Track.AudioEngineState.InvalidValue", "score-core", ParseDiagnosticUnknownSyntax), ParseDiagnostic{
				SourcePath: path + "/AudioEngineState", ObjectID: track.ID,
				Location: ParseLocation{TrackID: track.ID},
				Reason:   fmt.Sprintf("audio-engine state %q is not recognized", track.AudioEngineState),
			})
		}
		if track.Lyrics != nil && !track.Lyrics.Dispatched {
			context.add(diagnosticSource("GPIF.Track.Lyrics.Undispatched", "score-core", ParseDiagnosticLossyProjection), ParseDiagnostic{
				SourcePath: path + "/Lyrics/@dispatched", ObjectID: track.ID,
				Location: ParseLocation{TrackID: track.ID}, Feature: "score-core",
				Reason: "the public lyric model does not retain the source dispatch state",
			})
		}
		gpifAuditTrackTransposition(context, track, path)
		for soundIndex, sound := range track.Sounds.Sounds {
			if sound.MSB < 0 || sound.MSB > 127 || sound.LSB < 0 || sound.LSB > 127 {
				context.add(gpifSoundBankInvalidSource, ParseDiagnostic{
					SourcePath: fmt.Sprintf("%s/Sounds/Sound[%d]/MIDI", path, soundIndex),
					ObjectID:   track.ID, Location: ParseLocation{TrackID: track.ID}, Feature: "midi-bank",
					Reason: fmt.Sprintf("MIDI bank MSB %d and LSB %d must each be within 0..127", sound.MSB, sound.LSB),
				})
			}
			if sound.Channel == nil || *sound.Channel == track.MidiConnection.PrimaryChannel {
				continue
			}
			context.add(diagnosticSource("GPIF.Track.Sound.Channel", "score-core", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				SourcePath: fmt.Sprintf("%s/Sounds/Sound[%d]/MIDI/PrimaryChannel", path, soundIndex),
				ObjectID:   track.ID, Location: ParseLocation{TrackID: track.ID},
				Reason: "TrackSound does not retain a per-sound MIDI channel",
			})
		}
		gpifAuditStaffPropertyConflicts(context, track.Properties, path+"/Properties", track.ID, ParseLocation{TrackID: track.ID}, gpifTrackPropertyConflictSource)
		for propertyIndex, property := range track.Properties {
			gpifAuditTrackProperty(context, track.ID, fmt.Sprintf("%s/Properties/Property[%d]", path, propertyIndex), property)
		}
		for staffIndex, staff := range track.Staves.Staff {
			gpifAuditStaffPropertyConflicts(
				context, staff.Properties, fmt.Sprintf("%s/Staves/Staff[%d]/Properties", path, staffIndex),
				track.ID, ParseLocation{TrackID: track.ID}, gpifStaffPropertyConflictSource,
			)
			for propertyIndex, property := range staff.Properties {
				propertyPath := fmt.Sprintf("%s/Staves/Staff[%d]/Properties/Property[%d]", path, staffIndex, propertyIndex)
				gpifAuditStaffProperty(context, track.ID, propertyPath, property)
			}
		}
	}

	for _, note := range doc.Notes.Notes {
		path := gpifObjectPath("Notes/Note", note.ID)
		gpifAuditPropertyConflicts(context, note.Properties.Properties, path+"/Properties", note.ID, ParseLocation{NoteID: note.ID}, gpifNotePropertyConflictSource)
		if note.InstrumentArticulation != nil && *note.InstrumentArticulation < 0 {
			context.add(diagnosticSource("GPIF.Note.InstrumentArticulation.Invalid", "percussion-articulations", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: path + "/InstrumentArticulation", ObjectID: note.ID,
				Location: ParseLocation{NoteID: note.ID}, Feature: "percussion-articulations",
				Reason: fmt.Sprintf("percussion articulation identity %d must be non-negative", *note.InstrumentArticulation),
			})
		}
		var mappedMIDI *int
		for _, property := range note.Properties.Properties {
			if property.Name == "Midi" && property.Number != nil && *property.Number >= 0 && *property.Number <= 127 {
				value := *property.Number
				mappedMIDI = &value
			}
		}
		for _, property := range note.Properties.Properties {
			gpifAuditNoteProperty(context, note.ID, path, property, mappedMIDI)
		}
		gpifAuditEnum(context, diagnosticSource("GPIF.Note.Ornament.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), note.Ornament, []string{"", "Turn", "InvertedTurn", "UpperMordent", "LowerMordent"}, path+"/Ornament", note.ID, "ornaments")
		gpifAuditEnum(context, diagnosticSource("GPIF.Note.Vibrato.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), note.Vibrato, []string{"", "None", "Slight", "Wide"}, path+"/Vibrato", note.ID, "note-and-beat-semantics")
		if note.LeftFingering != nil {
			gpifAuditEnum(context, diagnosticSource("GPIF.Note.LeftFingering.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), *note.LeftFingering, []string{"P", "I", "M", "A", "C"}, path+"/LeftFingering", note.ID, "note-and-beat-semantics")
		}
		if note.RightFingering != nil {
			gpifAuditEnum(context, diagnosticSource("GPIF.Note.RightFingering.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), *note.RightFingering, []string{"P", "I", "M", "A", "C"}, path+"/RightFingering", note.ID, "note-and-beat-semantics")
		}
	}

	for _, beat := range doc.Beats.Beats {
		path := gpifObjectPath("Beats/Beat", beat.ID)
		gpifAuditBeamingBeat(context, beat.ID, path, &beat)
		gpifAuditBrush(context, &beat, path)
		gpifAuditPropertyConflicts(context, beat.Properties.Properties, path+"/Properties", beat.ID, ParseLocation{BeatID: beat.ID}, gpifBeatPropertyConflictSource)
		gpifAuditBarrePair(context, beat.ID, path+"/Properties", beat.Properties.Properties)
		for _, property := range beat.Properties.Properties {
			gpifAuditBeatProperty(context, beat.ID, path, property)
		}
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.GraceNotes.InvalidValue", "grace-relationships", ParseDiagnosticUnsupportedFeature), beat.GraceNotes, []string{"", "OnBeat", "BeforeBeat"}, path+"/GraceNotes", beat.ID, "grace-relationships")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Arpeggio.InvalidValue", "brush", ParseDiagnosticUnsupportedFeature), beat.Arpeggio, []string{"", "Up", "Down"}, path+"/Arpeggio", beat.ID, "brush")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Hairpin.InvalidValue", "hairpins", ParseDiagnosticUnsupportedFeature), beat.Hairpin, []string{"", "Crescendo", "Decrescendo", "Diminuendo"}, path+"/Hairpin", beat.ID, "hairpins")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Ottavia.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Ottavia, []string{"", "8va", "8vb", "15ma", "15mb"}, path+"/Ottavia", beat.ID, "note-and-beat-semantics")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Tremolo.InvalidValue", "tremolo-picking", ParseDiagnosticUnsupportedFeature), beat.Tremolo, []string{"", "1/2", "1/4", "1/8"}, path+"/Tremolo", beat.ID, "tremolo-picking")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Dynamic.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Dynamic, []string{"", "PPP", "PP", "P", "MP", "MF", "F", "FF", "FFF"}, path+"/Dynamic", beat.ID, "note-and-beat-semantics")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Golpe.InvalidValue", "golpe", ParseDiagnosticUnsupportedFeature), beat.Golpe, []string{"", "Thumb", "Finger"}, path+"/Golpe", beat.ID, "golpe")
		if beat.Legato != nil {
			gpifAuditLegato(context, beat.ID, path+"/Legato", beat.Legato)
		}
		if beat.Wah != "" && beat.Wah != "Open" && beat.Wah != "Closed" {
			context.add(diagnosticSource("GPIF.Beat.Wah", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				Kind: ParseDiagnosticUnsupportedFeature, SourcePath: path + "/Wah", ObjectID: beat.ID,
				Location: ParseLocation{BeatID: beat.ID}, Feature: "note-and-beat-semantics",
				Reason: fmt.Sprintf("unsupported GPIF wah state %q", beat.Wah),
			})
		}
		switch beat.Fadding {
		case "", "FadeIn", "FadeOut", "VolumeSwell":
		default:
			gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Fadding.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Fadding, []string{"", "FadeIn", "FadeOut", "VolumeSwell"}, path+"/Fadding", beat.ID, "note-and-beat-semantics")
		}
		if beat.Whammy != nil {
			gpifAuditWhammy(context, beat.ID, path+"/Whammy", beat.Whammy)
		}
	}

	for _, rhythm := range doc.Rhythms.Rhythms {
		gpifAuditEnum(context, diagnosticSource("GPIF.Rhythm.NoteValue.InvalidValue", "rhythm", ParseDiagnosticUnsupportedFeature), rhythm.NoteValue, []string{"Whole", "Half", "Quarter", "Eighth", "16th", "32nd", "64th", "128th"}, gpifObjectPath("Rhythms/Rhythm", rhythm.ID)+"/NoteValue", rhythm.ID, "rhythm")
	}
	for _, bar := range doc.Bars.Bars {
		gpifAuditEnum(context, diagnosticSource("GPIF.Bar.Ottavia.InvalidValue", "clef-octave", ParseDiagnosticUnsupportedFeature), bar.Ottavia, []string{"", "8va", "8vb", "15ma", "15mb"}, gpifObjectPath("Bars/Bar", bar.ID)+"/Ottavia", bar.ID, "clef-octave")
		gpifAuditEnum(context, diagnosticSource("GPIF.Bar.SimileMark.InvalidValue", "rhythm", ParseDiagnosticUnsupportedFeature), bar.SimileMark, []string{"", "Simple", "FirstOfDouble", "SecondOfDouble"}, gpifObjectPath("Bars/Bar", bar.ID)+"/SimileMark", bar.ID, "rhythm")
	}
	for index, masterBar := range doc.MasterBars.MasterBars {
		gpifAuditBeamingMasterBar(context, index, masterBar.XProperties)
		gpifAuditEnum(context, diagnosticSource("GPIF.MasterBar.Key.Mode.InvalidValue", "score-core", ParseDiagnosticUnsupportedFeature), masterBar.Key.Mode, []string{"", "Major", "major", "Minor", "minor"}, fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/Key/Mode", index), "", "score-core")
		gpifAuditEnum(context, diagnosticSource("GPIF.MasterBar.TripletFeel.InvalidValue", "rhythm", ParseDiagnosticUnsupportedFeature), masterBar.TripletFeel, []string{"", "NoTripletFeel", "Triplet8th", "Triplet16th", "Dotted8th", "Dotted16th", "Scottish8th", "Scottish16th"}, fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/TripletFeel", index), "", "rhythm")
		if masterBar.Fermatas != nil {
			path := fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/Fermatas", index)
			validOffsets := make([]ScoreTime, 0, len(masterBar.Fermatas.Fermatas))
			for fermataIndex, fermata := range masterBar.Fermatas.Fermatas {
				fermataPath := fmt.Sprintf("%s/Fermata[%d]", path, fermataIndex)
				gpifAuditEnum(context, diagnosticSource("GPIF.MasterBar.Fermata.Type.InvalidValue", "fermata", ParseDiagnosticUnsupportedFeature), fermata.Type, []string{"Short", "Medium", "Long"}, fermataPath+"/Type", "", "fermata")
				length, err := strconv.ParseFloat(fermata.Length, 64)
				if err != nil || math.IsNaN(length) || math.IsInf(length, 0) || length < 0 {
					context.add(diagnosticSource("GPIF.MasterBar.Fermata.Length.InvalidValue", "fermata", ParseDiagnosticInvalidData), ParseDiagnostic{
						SourcePath: fermataPath + "/Length",
						Reason:     fmt.Sprintf("fermata length %q must be finite and non-negative", fermata.Length),
					})
				}
				offset, ok := gpifFermataOffset(fermata.Offset)
				if !ok {
					context.add(diagnosticSource("GPIF.MasterBar.Fermata.Offset.InvalidValue", "fermata", ParseDiagnosticInvalidData), ParseDiagnostic{
						SourcePath: fermataPath + "/Offset",
						Reason:     fmt.Sprintf("fermata offset %q must be a non-negative rational score position", fermata.Offset),
					})
					continue
				}
				for _, earlier := range validOffsets {
					if offset.Compare(earlier) == 0 {
						context.add(diagnosticSource("GPIF.MasterBar.Fermata.Offset.Duplicate", "fermata", ParseDiagnosticInvalidData), ParseDiagnostic{
							SourcePath: fermataPath + "/Offset",
							Reason:     fmt.Sprintf("fermata offset %q duplicates an earlier record", fermata.Offset),
						})
						break
					}
				}
				validOffsets = append(validOffsets, offset)
			}
		}
		if masterBar.Directions != nil {
			path := fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/Directions", index)
			for targetIndex, target := range masterBar.Directions.Targets {
				gpifAuditEnum(context, diagnosticSource("GPIF.MasterBar.Directions.Target.InvalidValue", "score-core", ParseDiagnosticUnsupportedFeature), target, []string{"Coda", "DoubleCoda", "Segno", "SegnoSegno", "Fine"}, fmt.Sprintf("%s/Target[%d]", path, targetIndex), "", "score-core")
			}
			for jumpIndex, jump := range masterBar.Directions.Jumps {
				gpifAuditEnum(context, diagnosticSource("GPIF.MasterBar.Directions.Jump.InvalidValue", "score-core", ParseDiagnosticUnsupportedFeature), jump, []string{"DaCapo", "DaCapoAlCoda", "DaCapoAlDoubleCoda", "DaCapoAlFine", "DaSegno", "DaSegnoAlCoda", "DaSegnoAlDoubleCoda", "DaSegnoAlFine", "DaSegnoSegno", "DaSegnoSegnoAlCoda", "DaSegnoSegnoAlDoubleCoda", "DaSegnoSegnoAlFine", "DaCoda", "DaDoubleCoda"}, fmt.Sprintf("%s/Jump[%d]", path, jumpIndex), "", "score-core")
			}
		}
	}
	gpifAuditMasterBarCardinality(doc, context)

	gpifAuditReferences(doc, context)
}

var (
	gpifLegatoOriginInvalidSource      = diagnosticSource("GPIF.Beat.Legato.InvalidOrigin", "legato-slurs", ParseDiagnosticInvalidData)
	gpifLegatoDestinationInvalidSource = diagnosticSource("GPIF.Beat.Legato.InvalidDestination", "legato-slurs", ParseDiagnosticInvalidData)
)

func gpifAuditLegato(context *parseContext, beatID, path string, legato *gpifLegato) {
	for _, value := range []struct {
		name   string
		text   string
		source parseDiagnosticSource
	}{
		{name: "origin", text: legato.Origin, source: gpifLegatoOriginInvalidSource},
		{name: "destination", text: legato.Destination, source: gpifLegatoDestinationInvalidSource},
	} {
		if value.text == "true" || value.text == "false" {
			continue
		}
		context.add(value.source, ParseDiagnostic{
			SourcePath: path + "/@" + value.name,
			ObjectID:   beatID,
			Location:   ParseLocation{BeatID: beatID},
			Feature:    "legato-slurs",
			Reason:     fmt.Sprintf("legato %s %q must be the exact boolean true or false", value.name, value.text),
		})
	}
}

func gpifAuditPropertyConflicts(context *parseContext, properties []gpifProperty, path, objectID string, location ParseLocation, source parseDiagnosticSource) {
	gpifAuditPropertyConflictValues(context, properties, path, objectID, location, source, func(property gpifProperty) string { return property.Name })
}

func gpifAuditStaffPropertyConflicts(context *parseContext, properties []gpifStaffProperty, path, objectID string, location ParseLocation, source parseDiagnosticSource) {
	gpifAuditPropertyConflictValues(context, properties, path, objectID, location, source, func(property gpifStaffProperty) string { return property.Name })
}

func gpifAuditPropertyConflictValues[T any](context *parseContext, properties []T, path, objectID string, location ParseLocation, source parseDiagnosticSource, propertyName func(T) string) {
	seen := make(map[string]T, len(properties))
	for _, property := range properties {
		name := propertyName(property)
		previous, exists := seen[name]
		if exists && !reflect.DeepEqual(previous, property) {
			context.add(source, ParseDiagnostic{
				SourcePath: fmt.Sprintf("%s/Property[@name=%q]", path, name),
				ObjectID:   objectID, Location: location,
				Reason: fmt.Sprintf("GPIF property %q has conflicting duplicate values", name),
			})
		}
		seen[name] = property
	}
}

func gpifAuditMasterBarCardinality(doc gpifDocument, context *parseContext) {
	staffCounts := make(map[string]int, len(doc.Tracks.Tracks))
	for _, track := range doc.Tracks.Tracks {
		staffCounts[track.ID] = max(1, len(track.Staves.Staff))
	}
	trackIDs := splitIDs(doc.MasterTrack.Tracks)
	for masterBarIndex, masterBar := range doc.MasterBars.MasterBars {
		trackIndex := 0
		staffIndex := 0
		invalid := false
		for _, barID := range splitIDs(masterBar.Bars) {
			if trackIndex >= len(trackIDs) {
				invalid = true
				break
			}
			if barID == "-1" {
				if staffIndex != 0 {
					invalid = true
				}
				trackIndex++
				staffIndex = 0
				continue
			}
			staffIndex++
			if staffIndex >= staffCounts[trackIDs[trackIndex]] {
				trackIndex++
				staffIndex = 0
			}
		}
		if trackIndex != len(trackIDs) || staffIndex != 0 {
			invalid = true
		}
		if invalid {
			path := fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/Bars", masterBarIndex)
			context.add(diagnosticSource("GPIF.MasterBar.Bars.Cardinality", "staff-ownership", ParseDiagnosticInvalidData), ParseDiagnostic{
				Kind: ParseDiagnosticInvalidData, SourcePath: path, Feature: "staff-ownership",
				Reason: "bar references do not match the ordered track and staff layout",
			})
		}
	}
}
func gpifAuditTrackProperty(context *parseContext, trackID, path string, property gpifStaffProperty) {
	gpifAuditOwnedStaffProperty(
		context, trackID, path, property, "track",
		diagnosticSource("GPIF.Track.Property.Tuning.MissingPitches", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Track.Property.CapoFret.MissingFret", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Track.Property.CapoFret.Negative", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Track.Property.Unknown", "staff-ownership", ParseDiagnosticUnknownSyntax),
	)
}

func gpifAuditStaffProperty(context *parseContext, trackID, path string, property gpifStaffProperty) {
	gpifAuditOwnedStaffProperty(
		context, trackID, path, property, "staff",
		diagnosticSource("GPIF.Staff.Property.Tuning.MissingPitches", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Staff.Property.CapoFret.MissingFret", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Staff.Property.CapoFret.Negative", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Staff.Property.Unknown", "staff-ownership", ParseDiagnosticUnknownSyntax),
	)
}

func gpifAuditOwnedStaffProperty(
	context *parseContext,
	trackID, path string,
	property gpifStaffProperty,
	owner string,
	tuningMissingSource, capoMissingSource, capoNegativeSource, unknownSource parseDiagnosticSource,
) {
	propertyPath := fmt.Sprintf("%s[@name=%q]", path, property.Name)
	switch property.Name {
	case "Tuning":
		if property.Pitches == "" {
			context.add(tuningMissingSource, ParseDiagnostic{
				SourcePath: propertyPath, ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
				Reason: owner + " tuning property has no pitches",
			})
		}
	case "DiagramCollection", "ChordCollection":
		return
	case "CapoFret":
		if property.Fret == nil {
			context.add(capoMissingSource, ParseDiagnostic{
				SourcePath: propertyPath, ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
				Reason: owner + " capo property has no fret",
			})
		} else if *property.Fret < 0 {
			context.add(capoNegativeSource, ParseDiagnostic{
				SourcePath: propertyPath + "/Fret", ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
				Reason: owner + " capo fret is negative",
			})
		}
	default:
		context.add(unknownSource, ParseDiagnostic{
			SourcePath: propertyPath, ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
			Reason: fmt.Sprintf("unknown GPIF %s property %q", owner, property.Name),
		})
	}
}

func gpifReadCapo(properties []gpifStaffProperty) (int32, bool, error) {
	var value int32
	found := false
	for _, property := range properties {
		if property.Name == "CapoFret" && property.Fret != nil {
			wide := int64(*property.Fret)
			if wide < math.MinInt32 || wide > math.MaxInt32 {
				return 0, false, fmt.Errorf("fret %d is outside %d..%d", wide, math.MinInt32, math.MaxInt32)
			}
			value = int32(wide)
			found = true
		}
	}
	return value, found, nil
}

type gpifCapoResolution struct {
	value  int32
	common bool
}

func gpifResolveCapo(track gpifTrack) (gpifCapoResolution, error) {
	values, err := gpifResolveStaffCapos(track, max(1, len(track.Staves.Staff)))
	if err != nil {
		return gpifCapoResolution{}, err
	}
	common := true
	for _, value := range values[1:] {
		if value != values[0] {
			common = false
			break
		}
	}
	return gpifCapoResolution{value: values[0], common: common}, nil
}

func gpifResolveStaffCapos(track gpifTrack, staffCount int) ([]int32, error) {
	base, _, err := gpifReadCapo(track.Properties)
	if err != nil {
		return nil, err
	}
	values := make([]int32, staffCount)
	for index := range values {
		values[index] = base
	}
	for staffIndex, staff := range track.Staves.Staff {
		if staffIndex >= len(values) {
			break
		}
		if override, found, readErr := gpifReadCapo(staff.Properties); readErr != nil {
			return nil, fmt.Errorf("staff %d: %w", staffIndex, readErr)
		} else if found {
			values[staffIndex] = override
		}
	}
	return values, nil
}

var gpifNotePropertySources = map[string]parseDiagnosticSource{
	"BendOriginOffset":      diagnosticSource("GPIF.Note.Property.BendOriginOffset.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendOriginValue":       diagnosticSource("GPIF.Note.Property.BendOriginValue.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendMiddleOffset1":     diagnosticSource("GPIF.Note.Property.BendMiddleOffset1.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendMiddleOffset2":     diagnosticSource("GPIF.Note.Property.BendMiddleOffset2.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendMiddleValue":       diagnosticSource("GPIF.Note.Property.BendMiddleValue.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendDestinationOffset": diagnosticSource("GPIF.Note.Property.BendDestinationOffset.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendDestinationValue":  diagnosticSource("GPIF.Note.Property.BendDestinationValue.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"HopoDestination":       diagnosticSource("GPIF.Note.Property.HopoDestination", "note-and-beat-semantics", ParseDiagnosticLossyProjection),
	"Element":               diagnosticSource("GPIF.Note.Property.Element", "percussion-articulations", ParseDiagnosticUnsupportedFeature),
	"Variation":             diagnosticSource("GPIF.Note.Property.Variation", "percussion-articulations", ParseDiagnosticUnsupportedFeature),
	"ConcertPitch":          diagnosticSource("GPIF.Note.Property.ConcertPitch", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"TransposedPitch":       diagnosticSource("GPIF.Note.Property.TransposedPitch", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Tone":                  diagnosticSource("GPIF.Note.Property.Tone", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Octave":                diagnosticSource("GPIF.Note.Property.Octave", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
}

var gpifTechniqueMissingPayloadSources = map[string]parseDiagnosticSource{
	"Muted":           diagnosticSource("GPIF.Note.Property.Muted.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"PalmMuted":       diagnosticSource("GPIF.Note.Property.PalmMuted.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Tapped":          diagnosticSource("GPIF.Note.Property.Tapped.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"HopoOrigin":      diagnosticSource("GPIF.Note.Property.HopoOrigin.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"HopoDestination": diagnosticSource("GPIF.Note.Property.HopoDestination.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"LeftHandTapped":  diagnosticSource("GPIF.Note.Property.LeftHandTapped.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
}

var (
	gpifNoteBendInvalidSource        = diagnosticSource("GPIF.Note.Property.BendNumber.Invalid", "note-and-beat-semantics", ParseDiagnosticInvalidData)
	gpifNoteBendQuantizedSource      = diagnosticSource("GPIF.Note.Property.BendNumber.Quantized", "note-and-beat-semantics", ParseDiagnosticLossyProjection)
	gpifWhammyInvalidSource          = diagnosticSource("GPIF.Beat.Whammy.Invalid", "note-and-beat-semantics", ParseDiagnosticInvalidData)
	gpifWhammyQuantizedSource        = diagnosticSource("GPIF.Beat.Whammy.Quantized", "note-and-beat-semantics", ParseDiagnosticLossyProjection)
	gpifTempoAutomationInvalidSource = diagnosticSource("GPIF.MasterTrack.Automation.Tempo.Invalid", "tempo-automations", ParseDiagnosticInvalidData)
)

var gpifRedundantPitchSources = map[string]parseDiagnosticSource{
	"ConcertPitch":    diagnosticSource("GPIF.Note.Property.ConcertPitch.Redundant", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
	"TransposedPitch": diagnosticSource("GPIF.Note.Property.TransposedPitch.Redundant", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
}

var gpifBeatPropertySources = map[string]parseDiagnosticSource{
	"PrimaryPickupVolume":             diagnosticSource("GPIF.Beat.Property.PrimaryPickupVolume", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
	"PrimaryPickupTone":               diagnosticSource("GPIF.Beat.Property.PrimaryPickupTone", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
	"WhammyBarExtend":                 diagnosticSource("GPIF.Beat.Property.WhammyBarExtend", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
	"BarreFret.MissingPayload":        diagnosticSource("GPIF.Beat.Property.BarreFret.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BarreFret.InvalidValue":          diagnosticSource("GPIF.Beat.Property.BarreFret.InvalidValue", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BarreString.MissingPayload":      diagnosticSource("GPIF.Beat.Property.BarreString.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BarreString.InvalidValue":        diagnosticSource("GPIF.Beat.Property.BarreString.InvalidValue", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Brush.MissingDirection":          diagnosticSource("GPIF.Beat.Property.Brush.MissingDirection", "brush", ParseDiagnosticInvalidData),
	"PickStroke.MissingDirection":     diagnosticSource("GPIF.Beat.Property.PickStroke.MissingDirection", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Brush.InvalidDirection":          diagnosticSource("GPIF.Beat.Property.Brush.InvalidDirection", "brush", ParseDiagnosticUnsupportedFeature),
	"PickStroke.InvalidDirection":     diagnosticSource("GPIF.Beat.Property.PickStroke.InvalidDirection", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Slapped.MissingEnable":           diagnosticSource("GPIF.Beat.Property.Slapped.MissingEnable", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Popped.MissingEnable":            diagnosticSource("GPIF.Beat.Property.Popped.MissingEnable", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"VibratoWTremBar.MissingStrength": diagnosticSource("GPIF.Beat.Property.VibratoWTremBar.MissingStrength", "beat-vibrato", ParseDiagnosticInvalidData),
	"VibratoWTremBar.InvalidStrength": diagnosticSource("GPIF.Beat.Property.VibratoWTremBar.InvalidStrength", "beat-vibrato", ParseDiagnosticUnsupportedFeature),
}

func gpifAuditBarrePair(context *parseContext, beatID, path string, properties []gpifProperty) {
	hasFret := false
	hasShape := false
	for _, property := range properties {
		switch property.Name {
		case "BarreFret":
			hasFret = true
		case "BarreString":
			hasShape = true
		}
	}
	if hasFret == hasShape {
		return
	}
	missing := "BarreString"
	if hasShape {
		missing = "BarreFret"
	}
	context.add(diagnosticSource("GPIF.Beat.Property.Barre.Incomplete", "note-and-beat-semantics", ParseDiagnosticInvalidData), ParseDiagnostic{
		SourcePath: path, ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
		Reason: fmt.Sprintf("beat-level barre is missing its %s property", missing),
	})
}

func gpifAuditNoteProperty(context *parseContext, noteID, path string, property gpifProperty, mappedMIDI *int) {
	propertyPath := fmt.Sprintf("%s/Properties/Property[@name=%q]", path, property.Name)
	switch property.Name {
	case "Fret":
		gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.Fret.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData), property.Fret != nil, propertyPath, noteID, "note-and-beat-semantics", "Fret")
	case "String":
		gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.String.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData), property.String != nil, propertyPath, noteID, "note-and-beat-semantics", "String")
	case "Midi":
		gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.Midi.MissingPayload", "percussion-articulations", ParseDiagnosticInvalidData), property.Number != nil, propertyPath, noteID, "percussion-articulations", "Number")
	case "BendOriginOffset", "BendOriginValue", "BendMiddleOffset1", "BendMiddleOffset2", "BendMiddleValue", "BendDestinationOffset", "BendDestinationValue":
		gpifAuditPropertyPayload(context, gpifNotePropertySources[property.Name], property.Float != nil, propertyPath, noteID, "note-and-beat-semantics", "Float")
		if property.Float != nil {
			offset := strings.Contains(property.Name, "Offset")
			quantizedSource := gpifNoteBendQuantizedSource
			if offset {
				quantizedSource = parseDiagnosticSource{}
			}
			gpifAuditBendNumber(context, *property.Float, offset, propertyPath+"/Float", ParseLocation{NoteID: noteID}, noteID, gpifNoteBendInvalidSource, quantizedSource)
		}
	case "Slide":
		if property.Flags == nil {
			gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.Slide.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData), false, propertyPath, noteID, "note-and-beat-semantics", "Flags")
			return
		}
		flags, err := strconv.Atoi(*property.Flags)
		if err != nil || flags < 0 {
			context.add(diagnosticSource("GPIF.Note.Property.Slide.InvalidFlags", "note-and-beat-semantics", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: propertyPath + "/Flags", ObjectID: noteID, Location: ParseLocation{NoteID: noteID},
				Reason: fmt.Sprintf("slide flags %q are not a non-negative integer", *property.Flags),
			})
			return
		}
		if flags & ^0xff != 0 {
			context.add(diagnosticSource("GPIF.Note.Property.Slide.UnknownFlags", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				SourcePath: propertyPath + "/Flags", ObjectID: noteID, Location: ParseLocation{NoteID: noteID},
				Reason: fmt.Sprintf("slide flags %d contain unknown bits", flags),
			})
		}
	case "Muted", "PalmMuted":
		gpifAuditPropertyPayload(context, gpifTechniqueMissingPayloadSources[property.Name], property.Enable != nil, propertyPath, noteID, "note-and-beat-semantics", "Enable")
	case "Bended", "Harmonic", "ShowStringNumber":
		return
	case "Tapped", "HopoOrigin", "LeftHandTapped":
		gpifAuditPropertyPayload(context, gpifTechniqueMissingPayloadSources[property.Name], property.Enable != nil, propertyPath, noteID, "note-and-beat-semantics", "Enable")
	case "HopoDestination":
		gpifAuditPropertyPayload(context, gpifTechniqueMissingPayloadSources[property.Name], property.Enable != nil, propertyPath, noteID, "note-and-beat-semantics", "Enable")
		context.add(gpifNotePropertySources[property.Name], ParseDiagnostic{
			Kind: ParseDiagnosticLossyProjection, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "note-and-beat-semantics",
			Reason: "hammer/pull destinations are derived and have no authored destination field in Song",
		})
	case "HarmonicType":
		if property.HType == nil {
			gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.HarmonicType.MissingPayload", "harmonics", ParseDiagnosticInvalidData), false, propertyPath, noteID, "harmonics", "HType")
			return
		}
		switch *property.HType {
		case "NoHarmonic", "Natural", "Artificial", "Pinch", "Tap", "Semi", "Feedback":
			return
		default:
			context.add(diagnosticSource("GPIF.Note.Property.HarmonicType.Unsupported", "harmonics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				Kind: ParseDiagnosticUnsupportedFeature, SourcePath: propertyPath, ObjectID: noteID,
				Location: ParseLocation{NoteID: noteID}, Feature: "harmonics",
				Reason: fmt.Sprintf("unsupported harmonic type %q", *property.HType),
			})
		}
	case "HarmonicFret":
		gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.HarmonicFret.MissingPayload", "harmonics", ParseDiagnosticInvalidData), property.HFret != nil || property.Float != nil, propertyPath, noteID, "harmonics", "HFret or Float")
		raw, element := property.HFret, "HFret"
		if raw == nil {
			raw, element = property.Float, "Float"
		}
		if raw != nil {
			if _, valid := gpifHarmonicFret(raw); !valid {
				context.add(diagnosticSource("GPIF.Note.Property.HarmonicFret.Invalid", "harmonics", ParseDiagnosticInvalidData), ParseDiagnostic{
					SourcePath: propertyPath + "/" + element, ObjectID: noteID, Location: ParseLocation{NoteID: noteID},
					Reason: fmt.Sprintf("harmonic fret %q is not finite or outside 0..%d", *raw, math.MaxInt8),
				})
			}
		}
	case "Element", "Variation":
		context.add(gpifNotePropertySources[property.Name], ParseDiagnostic{
			Kind: ParseDiagnosticUnsupportedFeature, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "percussion-articulations",
			Reason: fmt.Sprintf("recognized GPIF percussion property %q has no destination in Song", property.Name),
		})
	case "ConcertPitch", "TransposedPitch":
		if mappedMIDI != nil && gpifPitchMatchesMIDI(property.Pitch, *mappedMIDI) {
			context.add(gpifRedundantPitchSources[property.Name], ParseDiagnostic{
				SourcePath: propertyPath, ObjectID: noteID, Location: ParseLocation{NoteID: noteID},
				Reason: fmt.Sprintf("GPIF pitch property %q is redundant with the mapped fret or MIDI value", property.Name),
			})
			return
		}
		context.add(gpifNotePropertySources[property.Name], ParseDiagnostic{
			Kind: ParseDiagnosticUnsupportedFeature, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "note-and-beat-semantics",
			Reason: fmt.Sprintf("recognized GPIF pitch property %q has no destination in Song", property.Name),
		})
	case "Tone", "Octave":
		context.add(gpifNotePropertySources[property.Name], ParseDiagnostic{
			Kind: ParseDiagnosticUnsupportedFeature, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "note-and-beat-semantics",
			Reason: fmt.Sprintf("recognized GPIF pitch property %q has no destination in Song", property.Name),
		})
	default:
		context.add(diagnosticSource("GPIF.Note.Property.Unknown", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax), ParseDiagnostic{
			Kind: ParseDiagnosticUnknownSyntax, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "note-and-beat-semantics",
			Reason: fmt.Sprintf("unknown GPIF note property %q", property.Name),
		})
	}
}

func gpifPitchMatchesMIDI(pitch *gpifPitch, midi int) bool {
	if pitch == nil {
		return false
	}
	steps := map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
	step, ok := steps[pitch.Step]
	if !ok {
		return false
	}
	alterations := map[string]int{"": 0, "#": 1, "##": 2, "b": -1, "bb": -2}
	alteration, ok := alterations[pitch.Accidental]
	if !ok {
		return false
	}
	return (pitch.Octave+1)*12+step+alteration == midi
}

func gpifAuditPropertyPayload(context *parseContext, source parseDiagnosticSource, present bool, path, objectID, feature, payload string) {
	if present {
		return
	}
	context.add(source, ParseDiagnostic{
		Kind: ParseDiagnosticInvalidData, SourcePath: path, ObjectID: objectID,
		Feature: feature, Reason: fmt.Sprintf("GPIF property has no %s payload", payload),
	})
}

func gpifAuditWhammy(context *parseContext, beatID, path string, whammy *gpifWhammy) {
	values := []struct {
		name   string
		value  string
		offset bool
	}{
		{name: "originValue", value: whammy.OriginValue},
		{name: "middleValue", value: whammy.MiddleValue},
		{name: "destinationValue", value: whammy.DestinationValue},
		{name: "originOffset", value: whammy.OriginOffset, offset: true},
		{name: "middleOffset1", value: whammy.MiddleOffset1, offset: true},
		{name: "middleOffset2", value: whammy.MiddleOffset2, offset: true},
		{name: "destinationOffset", value: whammy.DestinationOffset, offset: true},
	}
	for _, value := range values {
		gpifAuditBendNumber(context, value.value, value.offset, path+"/@"+value.name, ParseLocation{BeatID: beatID}, beatID, gpifWhammyInvalidSource, gpifWhammyQuantizedSource)
	}
}

func gpifAuditBendNumber(context *parseContext, raw string, offset bool, path string, location ParseLocation, objectID string, invalidSource, quantizedSource parseDiagnosticSource) {
	parsed, valid := gpifParseBendNumber(raw, offset)
	if !valid {
		minimum, maximum := gpifBendNumberBounds(offset)
		context.add(invalidSource, ParseDiagnostic{
			SourcePath: path, ObjectID: objectID, Location: location,
			Reason: fmt.Sprintf("bend number %q must be finite and inside %g..%g", raw, minimum, maximum),
		})
		return
	}
	// Exact offsets retain the source value; only heights require projection.
	if offset {
		return
	}
	projected := parsed / float64(GPBendSemitone)
	if quantizedSource.code != "" && math.Abs(projected-math.Round(projected)) > 0.000001 {
		context.add(quantizedSource, ParseDiagnostic{
			SourcePath: path, ObjectID: objectID, Location: location,
			Reason: fmt.Sprintf("bend number %q requires quantization to the public curve scale", raw),
		})
	}
}

func gpifAuditBeatProperty(context *parseContext, beatID, path string, property gpifProperty) {
	propertyPath := fmt.Sprintf("%s/Properties/Property[@name=%q]", path, property.Name)
	switch property.Name {
	case "BarreFret":
		if property.Fret == nil {
			gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingPayload"], false, propertyPath, beatID, "note-and-beat-semantics", "Fret")
			return
		}
		if _, err := NewFret(int64(*property.Fret)); err != nil {
			context.add(gpifBeatPropertySources[property.Name+".InvalidValue"], ParseDiagnostic{
				SourcePath: propertyPath + "/Fret", ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
				Reason: err.Error(),
			})
		}
	case "BarreString":
		if property.String == nil {
			gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingPayload"], false, propertyPath, beatID, "note-and-beat-semantics", "String")
			return
		}
		if *property.String != 0 && *property.String != 1 {
			context.add(gpifBeatPropertySources[property.Name+".InvalidValue"], ParseDiagnostic{
				SourcePath: propertyPath + "/String", ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
				Reason: fmt.Sprintf("barre string %v must be 0 (full) or 1 (half)", *property.String),
			})
		}
	case "Brush", "PickStroke":
		if property.Direction == nil {
			gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingDirection"], false, propertyPath, beatID, "note-and-beat-semantics", "Direction")
			return
		}
		gpifAuditEnum(context, gpifBeatPropertySources[property.Name+".InvalidDirection"], *property.Direction, []string{"Up", "Down"}, propertyPath+"/Direction", beatID, "note-and-beat-semantics")
	case "Slapped", "Popped":
		gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingEnable"], property.Enable != nil, propertyPath, beatID, "note-and-beat-semantics", "Enable")
	case "VibratoWTremBar":
		if property.Strength == nil {
			gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingStrength"], false, propertyPath, beatID, "beat-vibrato", "Strength")
			return
		}
		gpifAuditEnum(context, gpifBeatPropertySources[property.Name+".InvalidStrength"], *property.Strength, []string{"Slight", "Wide"}, propertyPath+"/Strength", beatID, "beat-vibrato")
	case "PrimaryPickupVolume", "PrimaryPickupTone":
		context.add(gpifBeatPropertySources[property.Name], ParseDiagnostic{
			SourcePath: propertyPath, ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
			Reason: "primary pickup playback metadata is intentionally outside the notation-focused Song model",
		})
		return
	case "WhammyBar":
		return
	case "WhammyBarOriginValue", "WhammyBarOriginOffset", "WhammyBarMiddleValue", "WhammyBarMiddleOffset1", "WhammyBarMiddleOffset2", "WhammyBarDestinationValue", "WhammyBarDestinationOffset":
		gpifAuditPropertyPayload(context, gpifWhammyInvalidSource, property.Float != nil, propertyPath, beatID, "note-and-beat-semantics", "Float")
		if property.Float != nil {
			gpifAuditBendNumber(context, *property.Float, strings.Contains(property.Name, "Offset"), propertyPath+"/Float", ParseLocation{BeatID: beatID}, beatID, gpifWhammyInvalidSource, gpifWhammyQuantizedSource)
		}
	case "WhammyBarExtend":
		context.add(gpifBeatPropertySources[property.Name], ParseDiagnostic{
			SourcePath: propertyPath, ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
			Reason: "the GPIF whammy extension marker has no documented playback or notation effect",
		})
	case "Rasgueado":
		if property.Rasgueado == nil {
			gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Beat.Property.Rasgueado.MissingPattern", "note-and-beat-semantics", ParseDiagnosticInvalidData), false, propertyPath, beatID, "note-and-beat-semantics", "Rasgueado")
		} else if gpifRasgueado(*property.Rasgueado) == RasgueadoNone {
			gpifAuditUnsupportedBeatProperty(context, diagnosticSource("GPIF.Beat.Property.Rasgueado", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beatID, propertyPath, property.Name+"="+*property.Rasgueado)
		}
	default:
		context.add(diagnosticSource("GPIF.Beat.Property.Unknown", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax), ParseDiagnostic{
			Kind: ParseDiagnosticUnknownSyntax, SourcePath: propertyPath, ObjectID: beatID,
			Location: ParseLocation{BeatID: beatID}, Feature: "note-and-beat-semantics",
			Reason: fmt.Sprintf("unknown GPIF beat property %q", property.Name),
		})
	}
}

func gpifAuditUnsupportedBeatProperty(context *parseContext, source parseDiagnosticSource, beatID, path, name string) {
	context.add(source, ParseDiagnostic{
		Kind: ParseDiagnosticUnsupportedFeature, SourcePath: path, ObjectID: beatID,
		Location: ParseLocation{BeatID: beatID}, Feature: "note-and-beat-semantics",
		Reason: fmt.Sprintf("recognized GPIF beat property %q has no lossless Song destination", name),
	})
}
func gpifAuditEnum(context *parseContext, source parseDiagnosticSource, value string, allowed []string, path, objectID, feature string) {
	for _, candidate := range allowed {
		if value == candidate {
			return
		}
	}
	context.add(source, ParseDiagnostic{
		Kind: ParseDiagnosticUnsupportedFeature, SourcePath: path, ObjectID: objectID,
		Feature: feature, Reason: fmt.Sprintf("unsupported GPIF value %q", value),
	})
}
func gpifObjectPath(collection, id string) string {
	return fmt.Sprintf("/GPIF/%s[@id=%q]", collection, id)
}

func gpifAuditReferences(doc gpifDocument, context *parseContext) {
	tracks := make(map[string]struct{}, len(doc.Tracks.Tracks))
	chords := make(map[string]struct{})
	type chordIDScope struct {
		track  map[string]struct{}
		staves []map[string]struct{}
	}
	chordScopes := make(map[string]chordIDScope, len(doc.Tracks.Tracks))
	for _, track := range doc.Tracks.Tracks {
		trackPath := gpifObjectPath("Tracks/Track", track.ID)
		gpifAddID(context, diagnosticSource("GPIF.Track.EmptyID", "staff-ownership", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Track.DuplicateID", "staff-ownership", ParseDiagnosticInvalidData), tracks, track.ID, trackPath, "staff-ownership")
		trackChords := make(map[string]struct{})
		gpifAuditChordIDs(context, trackChords, track.Properties, trackPath+"/Properties")
		for id := range trackChords {
			chords[id] = struct{}{}
		}
		scope := chordIDScope{track: trackChords, staves: make([]map[string]struct{}, len(track.Staves.Staff))}
		for staffIndex, staff := range track.Staves.Staff {
			path := fmt.Sprintf("%s/Staves/Staff[%d]/Properties", trackPath, staffIndex)
			staffChords := make(map[string]struct{})
			scope.staves[staffIndex] = staffChords
			gpifAuditChordIDs(context, staffChords, staff.Properties, path)
			for id := range staffChords {
				chords[id] = struct{}{}
			}
		}
		chordScopes[track.ID] = scope
	}
	bars := make(map[string]struct{}, len(doc.Bars.Bars))
	for _, bar := range doc.Bars.Bars {
		gpifAddID(context, diagnosticSource("GPIF.Bar.EmptyID", "staff-ownership", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Bar.DuplicateID", "staff-ownership", ParseDiagnosticInvalidData), bars, bar.ID, gpifObjectPath("Bars/Bar", bar.ID), "staff-ownership")
	}
	voices := make(map[string]struct{}, len(doc.Voices.Voices))
	for _, voice := range doc.Voices.Voices {
		gpifAddID(context, diagnosticSource("GPIF.Voice.EmptyID", "note-and-beat-semantics", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Voice.DuplicateID", "note-and-beat-semantics", ParseDiagnosticInvalidData), voices, voice.ID, gpifObjectPath("Voices/Voice", voice.ID), "note-and-beat-semantics")
	}
	beats := make(map[string]struct{}, len(doc.Beats.Beats))
	for _, beat := range doc.Beats.Beats {
		gpifAddID(context, diagnosticSource("GPIF.Beat.EmptyID", "note-and-beat-semantics", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Beat.DuplicateID", "note-and-beat-semantics", ParseDiagnosticInvalidData), beats, beat.ID, gpifObjectPath("Beats/Beat", beat.ID), "note-and-beat-semantics")
	}
	notes := make(map[string]struct{}, len(doc.Notes.Notes))
	for _, note := range doc.Notes.Notes {
		gpifAddID(context, diagnosticSource("GPIF.Note.EmptyID", "note-and-beat-semantics", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Note.DuplicateID", "note-and-beat-semantics", ParseDiagnosticInvalidData), notes, note.ID, gpifObjectPath("Notes/Note", note.ID), "note-and-beat-semantics")
	}
	rhythms := make(map[string]struct{}, len(doc.Rhythms.Rhythms))
	for _, rhythm := range doc.Rhythms.Rhythms {
		gpifAddID(context, diagnosticSource("GPIF.Rhythm.EmptyID", "rhythm", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Rhythm.DuplicateID", "rhythm", ParseDiagnosticInvalidData), rhythms, rhythm.ID, gpifObjectPath("Rhythms/Rhythm", rhythm.ID), "rhythm")
	}
	assets := make(map[string]struct{}, len(doc.Assets.Assets))
	for _, asset := range doc.Assets.Assets {
		gpifAddID(context, diagnosticSource("GPIF.Asset.EmptyID", "score-core", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Asset.DuplicateID", "score-core", ParseDiagnosticInvalidData), assets, asset.ID, gpifObjectPath("Assets/Asset", asset.ID), "score-core")
	}

	gpifAuditReferenceList(context, diagnosticSource("GPIF.MasterTrack.Tracks.Reference", "staff-ownership", ParseDiagnosticInvalidData), splitIDs(doc.MasterTrack.Tracks), tracks, "/GPIF/MasterTrack/Tracks", "", ParseLocation{}, "staff-ownership")
	barChordScopes := make(map[string]map[string]struct{})
	trackByID := make(map[string]gpifTrack, len(doc.Tracks.Tracks))
	for _, track := range doc.Tracks.Tracks {
		trackByID[track.ID] = track
	}
	for index, masterBar := range doc.MasterBars.MasterBars {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.MasterBar.Bars.Reference", "staff-ownership", ParseDiagnosticInvalidData), splitIDs(masterBar.Bars), bars, fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/Bars", index), "", ParseLocation{}, "staff-ownership")
		barIDs := splitIDs(masterBar.Bars)
		barIndex := 0
		for _, trackID := range splitIDs(doc.MasterTrack.Tracks) {
			track, exists := trackByID[trackID]
			if !exists {
				continue
			}
			staffCount := max(1, len(track.Staves.Staff))
			for staffIndex := 0; staffIndex < staffCount && barIndex < len(barIDs); staffIndex++ {
				barID := barIDs[barIndex]
				barIndex++
				if barID == "-1" {
					break
				}
				scope := make(map[string]struct{})
				for id := range chordScopes[trackID].track {
					scope[id] = struct{}{}
				}
				if staffIndex < len(chordScopes[trackID].staves) {
					for id := range chordScopes[trackID].staves[staffIndex] {
						scope[id] = struct{}{}
					}
				}
				barChordScopes[barID] = scope
			}
		}
	}
	voiceChordScopes := make(map[string][]map[string]struct{})
	for _, bar := range doc.Bars.Bars {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Bar.Voices.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), splitIDs(bar.Voices), voices, gpifObjectPath("Bars/Bar", bar.ID)+"/Voices", bar.ID, ParseLocation{BarID: bar.ID}, "note-and-beat-semantics")
		for _, voiceID := range splitIDs(bar.Voices) {
			if scope, ok := barChordScopes[bar.ID]; ok {
				voiceChordScopes[voiceID] = append(voiceChordScopes[voiceID], scope)
			}
		}
	}
	beatChordScopes := make(map[string][]map[string]struct{})
	for _, voice := range doc.Voices.Voices {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Voice.Beats.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), splitIDs(voice.Beats), beats, gpifObjectPath("Voices/Voice", voice.ID)+"/Beats", voice.ID, ParseLocation{VoiceID: voice.ID}, "note-and-beat-semantics")
		for _, beatID := range splitIDs(voice.Beats) {
			beatChordScopes[beatID] = append(beatChordScopes[beatID], voiceChordScopes[voice.ID]...)
		}
	}
	for _, beat := range doc.Beats.Beats {
		location := ParseLocation{BeatID: beat.ID}
		path := gpifObjectPath("Beats/Beat", beat.ID)
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Beat.Notes.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), splitIDs(beat.Notes), notes, path+"/Notes", beat.ID, location, "note-and-beat-semantics")
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Beat.Rhythm.Reference", "rhythm", ParseDiagnosticInvalidData), []string{beat.Rhythm.Ref}, rhythms, path+"/Rhythm", beat.ID, location, "rhythm")
		targetScopes := beatChordScopes[beat.ID]
		if len(targetScopes) == 0 {
			targetScopes = []map[string]struct{}{chords}
		}
		for _, targets := range targetScopes {
			if beat.Chord == "" || beat.Chord == "-1" {
				break
			}
			if _, exists := targets[beat.Chord]; !exists {
				gpifAuditReferenceList(context, diagnosticSource("GPIF.Beat.Chord.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), []string{beat.Chord}, targets, path+"/Chord", beat.ID, location, "note-and-beat-semantics")
				break
			}
		}
	}
	if doc.BackingTrack != nil {
		if doc.BackingTrack.Enabled && strings.EqualFold(doc.BackingTrack.Source, "Local") {
			gpifAuditReferenceList(context, gpifBackingTrackAssetReferenceSource, []string{doc.BackingTrack.AssetID}, assets, "/GPIF/BackingTrack/AssetId", "", ParseLocation{}, "score-core")
		}
		framePadding := strings.TrimSpace(doc.BackingTrack.FramePadding)
		if _, err := strconv.ParseInt(framePadding, 10, 64); err != nil {
			context.add(diagnosticSource("GPIF.BackingTrack.FramePadding.Invalid", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: "/GPIF/BackingTrack/FramePadding",
				Reason:     fmt.Sprintf("backing-track frame padding %q is not a signed 64-bit frame count", framePadding),
			})
		}
	}
}

var gpifDiagramPropertySources = map[string]parseDiagnosticSource{
	"ShowName":      diagnosticSource("GPIF.Chord.Diagram.Property.ShowName", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"ShowDiagram":   diagnosticSource("GPIF.Chord.Diagram.Property.ShowDiagram", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"ShowFingering": diagnosticSource("GPIF.Chord.Diagram.Property.ShowFingering", "note-and-beat-semantics", ParseDiagnosticInvalidData),
}

func gpifAuditChordIDs(context *parseContext, chords map[string]struct{}, properties []gpifStaffProperty, path string) {
	for propertyIndex, property := range properties {
		if property.Items == nil || (property.Name != "DiagramCollection" && property.Name != "ChordCollection") {
			continue
		}
		for itemIndex, item := range property.Items.Items {
			itemPath := fmt.Sprintf("%s/Property[%d]/Items/Item[%d]", path, propertyIndex, itemIndex)
			gpifAddID(context,
				diagnosticSource("GPIF.ChordDefinition.EmptyID", "note-and-beat-semantics", ParseDiagnosticInvalidData),
				diagnosticSource("GPIF.ChordDefinition.DuplicateID", "note-and-beat-semantics", ParseDiagnosticInvalidData),
				chords, item.ID, itemPath, "note-and-beat-semantics")
			if item.Diagram == nil {
				continue
			}
			for diagramPropertyIndex, diagramProperty := range item.Diagram.Properties {
				propertyPath := fmt.Sprintf("%s/Diagram/Property[%d][@name=%q]", itemPath, diagramPropertyIndex, diagramProperty.Name)
				source, known := gpifDiagramPropertySources[diagramProperty.Name]
				if known && (diagramProperty.Value == "true" || diagramProperty.Value == "false") {
					continue
				}
				if !known {
					source = diagnosticSource("GPIF.Chord.Diagram.Property.Unknown", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax)
				}
				context.add(source, ParseDiagnostic{
					SourcePath: propertyPath, ObjectID: item.ID,
					Reason: fmt.Sprintf("GPIF diagram property %q has unsupported value %q", diagramProperty.Name, diagramProperty.Value),
				})
			}
		}
	}
}
func gpifAddID(context *parseContext, emptySource, duplicateSource parseDiagnosticSource, ids map[string]struct{}, id, path, feature string) {
	if id == "" {
		context.add(emptySource, ParseDiagnostic{Kind: ParseDiagnosticInvalidData, SourcePath: path, Feature: feature, Reason: "GPIF object has an empty ID"})
		return
	}
	if _, exists := ids[id]; exists {
		context.add(duplicateSource, ParseDiagnostic{Kind: ParseDiagnosticInvalidData, SourcePath: path, ObjectID: id, Feature: feature, Reason: fmt.Sprintf("duplicate GPIF object ID %q", id)})
	}
	ids[id] = struct{}{}
}

func gpifAuditReferenceList(context *parseContext, source parseDiagnosticSource, references []string, targets map[string]struct{}, path, objectID string, location ParseLocation, feature string) {
	for _, reference := range references {
		if reference == "" || reference == "-1" {
			continue
		}
		if _, exists := targets[reference]; !exists {
			context.add(source, ParseDiagnostic{
				Kind: ParseDiagnosticInvalidData, SourcePath: path, ObjectID: objectID, Location: location,
				Feature: feature, Reason: fmt.Sprintf("GPIF reference %q does not exist", reference),
			})
		}
	}
}
