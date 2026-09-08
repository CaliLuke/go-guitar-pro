// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExportGP8RoundTrip(t *testing.T) {
	song := syntheticGP8Song()
	before, err := json.Marshal(song)
	if err != nil {
		t.Fatal(err)
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(song)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("Export mutated the source Song")
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	version := readZipMember(t, archive, "VERSION")
	if string(version) != gp8ContainerVersion {
		t.Fatalf("VERSION = %q, want %q", version, gp8ContainerVersion)
	}
	gpifData := readZipMember(t, archive, "Content/score.gpif")
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(gpifData, &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if document.GPVersion != gp8DocumentVersion {
		t.Errorf("GPVersion = %q, want %q", document.GPVersion, gp8DocumentVersion)
	}
	if document.Encoding.Description != "GP8" {
		t.Errorf("EncodingDescription = %q, want GP8", document.Encoding.Description)
	}
	if document.GPRevision.Required != gp8RevisionRequired || document.GPRevision.Value != gp8Revision {
		t.Errorf("GPRevision = %#v, want required %q revision %q", document.GPRevision, gp8RevisionRequired, gp8Revision)
	}
	for _, name := range []string{
		"Content/",
		"Content/BinaryStylesheet",
		"Content/PartConfiguration",
		"Content/LayoutConfiguration",
	} {
		readZipMember(t, archive, name)
	}
	for _, file := range archive.File {
		if file.Flags&0x08 != 0 {
			t.Errorf("archive member %q uses a streaming data descriptor", file.Name)
		}
		if file.Name == "Content/" && !file.Mode().IsDir() {
			t.Errorf("Content/ mode = %v, want directory", file.Mode())
		}
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Version.Number != [3]byte{8, 1, 3} {
		t.Errorf("version = %v, want [8 1 3]", roundTrip.Version.Number)
	}
	if roundTrip.Name != song.Name || roundTrip.Artist != song.Artist || roundTrip.Tempo != song.Tempo {
		t.Errorf("score metadata = %q, %q, %d; want %q, %q, %d", roundTrip.Name, roundTrip.Artist, roundTrip.Tempo, song.Name, song.Artist, song.Tempo)
	}
	if len(roundTrip.Notice) != 2 || roundTrip.Notice[0] != "First notice" || roundTrip.Notice[1] != "Second notice" {
		t.Errorf("notices = %q", roundTrip.Notice)
	}
	if !roundTrip.Anacrusis {
		t.Error("Anacrusis = false, want true")
	}
	if len(roundTrip.MeasureHeaders) != 3 || len(roundTrip.Tracks) != 1 || len(roundTrip.Tracks[0].Measures) != 3 {
		t.Fatalf("round-trip graph has %d headers, %d tracks, %d measures", len(roundTrip.MeasureHeaders), len(roundTrip.Tracks), len(roundTrip.Tracks[0].Measures))
	}

	track := roundTrip.Tracks[0]
	if track.Name != "Percussion" || track.Color != 0x336699 || !track.PercussionTrack {
		t.Errorf("track metadata = %#v", track)
	}
	if track.ChannelIndex < 0 {
		t.Fatalf("track channel index = %d, track = %#v", track.ChannelIndex, track)
	}
	if roundTrip.Channels[track.ChannelIndex].Channel != DefaultPercussionChannel {
		t.Errorf("percussion channel = %d, want %d", roundTrip.Channels[track.ChannelIndex].Channel, DefaultPercussionChannel)
	}
	if document.Tracks.Tracks[0].Instrument != nil {
		t.Errorf("percussion track has deprecated instrument reference: %#v", document.Tracks.Tracks[0].Instrument)
	}
	if got := document.Tracks.Tracks[0].InstrumentSet.Type; got != "drumKit" {
		t.Errorf("percussion instrument set type = %q, want drumKit", got)
	}
	if got := document.Tracks.Tracks[0].AudioEngineState; got != "MIDI" {
		t.Errorf("audio engine state = %q, want MIDI", got)
	}
	var articulationMIDI []int
	for _, element := range document.Tracks.Tracks[0].InstrumentSet.Elements.Elements {
		for _, articulation := range element.Articulations.Articulations {
			articulationMIDI = append(articulationMIDI, articulation.OutputMIDINumber)
		}
	}
	if got, want := fmt.Sprint(articulationMIDI), "[36 38 42]"; got != want {
		t.Errorf("percussion articulation MIDI values = %s, want %s", got, want)
	}

	firstHeader := roundTrip.MeasureHeaders[0]
	if firstHeader.Marker == nil || firstHeader.Marker.Title != "Pickup" || !firstHeader.RepeatOpen {
		t.Errorf("first header = %#v", firstHeader)
	}
	secondHeader := roundTrip.MeasureHeaders[1]
	if secondHeader.TimeSignature.Numerator != 3 || secondHeader.TimeSignature.Denominator.Value != 4 || secondHeader.RepeatClose != 1 || secondHeader.RepeatAlternative != 0b11 {
		t.Errorf("second header = %#v", secondHeader)
	}
	if secondHeader.Marker == nil || secondHeader.Marker.Title != "" {
		t.Errorf("empty section marker = %#v, want a present empty marker", secondHeader.Marker)
	}
	if !roundTrip.MeasureHeaders[2].DoubleBar {
		t.Error("last measure lost its double bar")
	}

	pickupBeats := track.Measures[0].Voices[0].Beats
	if len(pickupBeats) != 3 {
		t.Fatalf("pickup beats = %d, want 3", len(pickupBeats))
	}
	graceNote := pickupBeats[0].Notes[0]
	if graceNote.Value != 38 || len(graceNote.Effect.Graces) != 1 || graceNote.Effect.Graces[0].Fret != 38 || !graceNote.Effect.AccentuatedNote {
		t.Errorf("grace snare = %#v", graceNote)
	}
	if pickupBeats[0].Duration.Value != uint16(DurationEighth) || graceNote.Velocity != Forte {
		t.Errorf("grace beat duration/velocity = %d/%d", pickupBeats[0].Duration.Value, graceNote.Velocity)
	}
	if len(pickupBeats[0].Notes) != 2 || len(pickupBeats[0].Notes[1].Effect.Graces) != 0 || pickupBeats[0].Notes[1].String != 0 {
		t.Errorf("partial grace chord = %#v", pickupBeats[0].Notes)
	}
	if len(pickupBeats[1].Notes) != 2 || pickupBeats[1].Effect.Chord == nil || pickupBeats[1].Effect.Chord.Name != "Kick + hat" {
		t.Errorf("percussion chord = %#v", pickupBeats[1])
	}
	if pickupBeats[2].Status != BeatStatusRest || len(pickupBeats[2].Notes) != 0 {
		t.Errorf("rest beat = %#v", pickupBeats[2])
	}
	accent := track.Measures[1].Voices[0].Beats[0]
	if !accent.Notes[0].Effect.HeavyAccentuatedNote || accent.Notes[0].Velocity != MinVelocity+VelocityIncrement*6 || !accent.Duration.Dotted {
		t.Errorf("accent beat = %#v", accent)
	}
}

func TestExportGP8RealPercussionFixture(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/motherload-percussion-grace.gp5")
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(roundTrip.Tracks) != len(song.Tracks) || len(roundTrip.MeasureHeaders) != len(song.MeasureHeaders) {
		t.Fatalf("round trip has %d tracks/%d measures, want %d/%d", len(roundTrip.Tracks), len(roundTrip.MeasureHeaders), len(song.Tracks), len(song.MeasureHeaders))
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	for index, sourceTrack := range song.Tracks {
		instrumentSet := document.Tracks.Tracks[index].InstrumentSet
		if instrumentSet == nil {
			t.Errorf("track %d (%q) has no instrument set", index+1, sourceTrack.Name)
			continue
		}
		if sourceTrack.PercussionTrack {
			if instrumentSet.Type != "drumKit" {
				t.Errorf("percussion track %d instrument type = %q", index+1, instrumentSet.Type)
			}
		} else if instrumentSet.Type == "drumKit" || len(instrumentSet.Elements.Elements) != 1 || instrumentSet.Elements.Elements[0].Type != "pitched" {
			t.Errorf("pitched track %d instrument set = %#v", index+1, instrumentSet)
		}
	}

	var percussion *Track
	for index := range roundTrip.Tracks {
		if roundTrip.Tracks[index].Name == "Percussion" {
			percussion = &roundTrip.Tracks[index]
			break
		}
	}
	if percussion == nil || !percussion.PercussionTrack {
		t.Fatal("round trip lost the percussion track")
	}
	graceCount := 0
	for _, measure := range percussion.Measures {
		for _, voice := range measure.Voices {
			for _, beat := range voice.Beats {
				for _, note := range beat.Notes {
					if len(note.Effect.Graces) > 0 {
						graceCount += len(note.Effect.Graces)
						for _, grace := range note.Effect.Graces {
							if note.Value != 38 || grace.Fret != 38 {
								t.Errorf("grace articulation = note %d grace %d, want 38", note.Value, grace.Fret)
							}
						}
					}
				}
			}
		}
	}
	if graceCount != 8 {
		t.Errorf("percussion grace notes = %d, want 8", graceCount)
	}
}

func TestExportGP8PreservesOrderedMultipleGraceNotes(t *testing.T) {
	song := syntheticGP8Song()
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Effect.Graces = []GraceEffect{
		{Fret: 38, Duration: DurationThirtySecond, Velocity: Forte},
		{Fret: 38, Duration: DurationThirtySecond, Velocity: Forte - VelocityIncrement},
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces
	if len(got) != 2 {
		t.Fatalf("grace notes = %#v, want two ordered grace notes", got)
	}
	if got[0].Fret != 38 || got[0].Velocity != Forte || got[1].Fret != 38 || got[1].Velocity != Forte-VelocityIncrement {
		t.Errorf("grace notes = %#v, want preserved values and order", got)
	}
}

func TestExportGP8WritesRepeatCountAsAttribute(t *testing.T) {
	data, err := Export(syntheticGP8Song(), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	gpif := string(readZipMember(t, archive, "Content/score.gpif"))
	if !strings.Contains(gpif, `<Repeat end="true" count="2">`) {
		t.Errorf("repeat close does not use the GPIF count attribute:\n%s", gpif)
	}
	if strings.Contains(gpif, "<Count>") {
		t.Error("repeat close uses a Count child element that independent consumers ignore")
	}
}

func TestGP8PitchedInstrumentSet(t *testing.T) {
	for _, test := range []struct {
		program        int32
		name           string
		instrumentType string
	}{
		{program: 30, name: "Electric Guitar", instrumentType: "electricGuitar"},
		{program: 33, name: "Electric Bass", instrumentType: "electricBass"},
		{program: 56, name: "Trumpet", instrumentType: "trumpet"},
		{program: 68, name: "Oboe", instrumentType: "oboe"},
		{program: 127, name: "Timpani", instrumentType: "timpani"},
	} {
		name, instrumentType := gp8PitchedInstrumentSet(test.program)
		if name != test.name || instrumentType != test.instrumentType {
			t.Errorf("program %d = %q/%q, want %q/%q", test.program, name, instrumentType, test.name, test.instrumentType)
		}
	}
}

func TestGP8HiHatArticulationMetadata(t *testing.T) {
	for _, test := range []struct {
		midi      int16
		name      string
		staffLine int
		noteheads string
		rseSound  string
	}{
		{midi: 42, name: "Hi-Hat (closed)", staffLine: -1, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "stick.hit.closed"},
		{midi: 44, name: "Pedal Hi-Hat (hit)", staffLine: 9, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "pedal.hit.pedal"},
		{midi: 46, name: "Hi-Hat (open)", staffLine: -1, noteheads: "noteheadCircleX noteheadCircleX noteheadCircleX", rseSound: "stick.hit.open"},
	} {
		element := gp8DrumElement(test.midi, GP8ExportOptions{})
		if element.Name != "Charley" || element.Type != "hiHat" || element.SoundbankName != "Master-Hihat" {
			t.Errorf("MIDI %d element = %#v", test.midi, element)
		}
		if len(element.Articulations.Articulations) != 1 {
			t.Fatalf("MIDI %d articulations = %#v", test.midi, element.Articulations.Articulations)
		}
		articulation := element.Articulations.Articulations[0]
		if articulation.Name != test.name || articulation.StaffLine != test.staffLine || articulation.Noteheads != test.noteheads || articulation.OutputRSESound != test.rseSound {
			t.Errorf("MIDI %d articulation = %#v", test.midi, articulation)
		}
		if articulation.InputMIDINumbers != fmt.Sprint(test.midi) || articulation.OutputMIDINumber != int(test.midi) {
			t.Errorf("MIDI %d routing = %#v", test.midi, articulation)
		}
	}
	elements := gp8DrumElements([]int16{36, 38, 42, 44, 46, 48, 49}, GP8ExportOptions{})
	if len(elements) != 5 {
		t.Fatalf("grouped drum elements = %#v", elements)
	}
	hiHat := elements[2]
	if hiHat.Name != "Charley" || len(hiHat.Articulations.Articulations) != 3 {
		t.Errorf("grouped hi-hat = %#v", hiHat)
	}
	for index, midi := range []int{42, 44, 46} {
		if hiHat.Articulations.Articulations[index].OutputMIDINumber != midi {
			t.Errorf("hi-hat articulation %d = %#v", index, hiHat.Articulations.Articulations[index])
		}
	}

	interleaved := gp8DrumElements([]int16{42, 43, 44, 45, 46}, GP8ExportOptions{})
	if len(interleaved) != 3 || len(interleaved[0].Articulations.Articulations) != 3 {
		t.Fatalf("interleaved hi-hat elements = %#v", interleaved)
	}
	ids := gp8DrumArticulationIDs(interleaved)
	for midi, id := range map[int16]int{42: 0, 44: 1, 46: 2, 43: 3, 45: 4} {
		if ids[midi] != id {
			t.Errorf("MIDI %d articulation ID = %d, want %d", midi, ids[midi], id)
		}
	}
}

func TestExportGP8PercussionNoteheadOverrides(t *testing.T) {
	song := syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[1].Value = 46
	options := ExportOptions{GP8: GP8ExportOptions{PercussionNoteheads: map[int16]GP8PercussionNotehead{
		46: GP8PercussionNoteheadFilled,
	}}}
	data, err := ExportWithOptions(song, ExportFormatGP8, options)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	for _, element := range document.Tracks.Tracks[0].InstrumentSet.Elements.Elements {
		for _, articulation := range element.Articulations.Articulations {
			if articulation.OutputMIDINumber != 46 {
				continue
			}
			if articulation.Noteheads != "noteheadBlack noteheadHalf noteheadWhole" {
				t.Errorf("open hi-hat noteheads = %q", articulation.Noteheads)
			}
			if articulation.OutputRSESound != "stick.hit.open" || element.SoundbankName != "Master-Hihat" {
				t.Errorf("open hi-hat playback metadata = %#v/%#v", element, articulation)
			}
			return
		}
	}
	t.Fatal("exported GP8 has no open hi-hat articulation")
}

func TestGP8NativeDrumKitMetadata(t *testing.T) {
	for _, test := range []struct {
		midi             int16
		elementName      string
		kind             string
		soundbank        string
		articulationName string
		staffLine        int
		noteheads        string
		rseSound         string
		outputMIDI       int
	}{
		{midi: 35, elementName: "Acoustic Kick Drum", kind: "kickDrum", soundbank: "AcousticKick-Percu", articulationName: "Kick (hit)", staffLine: 8, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "pedal.hit.hit"},
		{midi: 36, elementName: "Kick Drum", kind: "kickDrum", soundbank: "Master-Kick", articulationName: "Kick (hit)", staffLine: 7, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "pedal.hit.hit"},
		{midi: 37, elementName: "Snare", kind: "snare", soundbank: "Master-Snare", articulationName: "Snare (side stick)", staffLine: 3, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "stick.hit.sidestick"},
		{midi: 38, elementName: "Snare", kind: "snare", soundbank: "Master-Snare", articulationName: "Snare (hit)", staffLine: 3, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 41, elementName: "Very Low Floor Tom", kind: "tom", soundbank: "LowFloorTom-Percu", articulationName: "Low Floor Tom (hit)", staffLine: 5, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 42, elementName: "Charley", kind: "hiHat", soundbank: "Master-Hihat", articulationName: "Hi-Hat (closed)", staffLine: -1, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "stick.hit.closed"},
		{midi: 43, elementName: "Tom Very Low", kind: "tom", soundbank: "Master-Tom01", articulationName: "Very Low Tom (hit)", staffLine: 6, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 44, elementName: "Charley", kind: "hiHat", soundbank: "Master-Hihat", articulationName: "Pedal Hi-Hat (hit)", staffLine: 9, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "pedal.hit.pedal"},
		{midi: 45, elementName: "Tom Low", kind: "tom", soundbank: "Master-Tom02", articulationName: "Low Tom (hit)", staffLine: 5, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 46, elementName: "Charley", kind: "hiHat", soundbank: "Master-Hihat", articulationName: "Hi-Hat (open)", staffLine: -1, noteheads: "noteheadCircleX noteheadCircleX noteheadCircleX", rseSound: "stick.hit.open"},
		{midi: 47, elementName: "Tom Medium", kind: "tom", soundbank: "Master-Tom03", articulationName: "Mid Tom (hit)", staffLine: 4, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 48, elementName: "Tom High", kind: "tom", soundbank: "Master-Tom04", articulationName: "High Tom (hit)", staffLine: 2, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 49, elementName: "Crash High", kind: "crash", soundbank: "Master-Crash02", articulationName: "Crash high (hit)", staffLine: -2, noteheads: "noteheadHeavyX noteheadHeavyX noteheadHeavyX", rseSound: "stick.hit.hit"},
		{midi: 50, elementName: "Tom Very High", kind: "tom", soundbank: "Master-Tom05", articulationName: "High Floor Tom (hit)", staffLine: 1, noteheads: "noteheadBlack noteheadHalf noteheadWhole", rseSound: "stick.hit.hit"},
		{midi: 51, elementName: "Ride", kind: "ride", soundbank: "Master-Ride", articulationName: "Ride (middle)", staffLine: 0, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "stick.hit.mid"},
		{midi: 52, elementName: "China", kind: "china", soundbank: "Master-China", articulationName: "China (hit)", staffLine: -3, noteheads: "noteheadHeavyXHat noteheadHeavyXHat noteheadHeavyXHat", rseSound: "stick.hit.hit"},
		{midi: 53, elementName: "Ride", kind: "ride", soundbank: "Master-Ride", articulationName: "Ride (bell)", staffLine: 0, noteheads: "noteheadDiamondWhite noteheadDiamondWhite noteheadDiamondWhite", rseSound: "stick.hit.bell"},
		{midi: 55, elementName: "Splash", kind: "splash", soundbank: "Master-Splash", articulationName: "Splash (hit)", staffLine: -2, noteheads: "noteheadXBlack noteheadXBlack noteheadXBlack", rseSound: "stick.hit.hit"},
		{midi: 57, elementName: "Crash Medium", kind: "crash", soundbank: "Master-Crash01", articulationName: "Crash medium (hit)", staffLine: -1, noteheads: "noteheadHeavyX noteheadHeavyX noteheadHeavyX", rseSound: "stick.hit.hit"},
		{midi: 92, elementName: "Charley", kind: "hiHat", soundbank: "Master-Hihat", articulationName: "Hi-Hat (half)", staffLine: -1, noteheads: "noteheadCircleSlash noteheadCircleSlash noteheadCircleSlash", rseSound: "stick.hit.half", outputMIDI: 46},
		{midi: 99, elementName: "Cowbell Low", kind: "cowbell", soundbank: "CowbellBig-Percu", articulationName: "Cowbell low (hit)", staffLine: 1, noteheads: "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole", rseSound: "stick.hit.hit", outputMIDI: 56},
		{midi: 102, elementName: "Cowbell High", kind: "cowbell", soundbank: "CowbellSmall-Percu", articulationName: "Cowbell high (hit)", staffLine: -1, noteheads: "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole", rseSound: "stick.hit.hit", outputMIDI: 56},
	} {
		element := gp8DrumElement(test.midi, GP8ExportOptions{})
		if element.Name != test.elementName || element.Type != test.kind || element.SoundbankName != test.soundbank {
			t.Errorf("MIDI %d element = %#v", test.midi, element)
		}
		articulation := element.Articulations.Articulations[0]
		if articulation.Name != test.articulationName || articulation.StaffLine != test.staffLine || articulation.Noteheads != test.noteheads || articulation.OutputRSESound != test.rseSound {
			t.Errorf("MIDI %d articulation = %#v", test.midi, articulation)
		}
		expectedOutput := int(test.midi)
		if test.outputMIDI != 0 {
			expectedOutput = test.outputMIDI
		}
		if articulation.InputMIDINumbers != fmt.Sprint(test.midi) || articulation.OutputMIDINumber != expectedOutput {
			t.Errorf("MIDI %d routing = %#v", test.midi, articulation)
		}
	}

	snares := gp8DrumElements([]int16{37, 38}, GP8ExportOptions{})
	if len(snares) != 1 || len(snares[0].Articulations.Articulations) != 2 {
		t.Fatalf("grouped snare elements = %#v", snares)
	}
	for index, midi := range []int{37, 38} {
		if snares[0].Articulations.Articulations[index].OutputMIDINumber != midi {
			t.Errorf("snare articulation %d = %#v", index, snares[0].Articulations.Articulations[index])
		}
	}

	hiHats := gp8DrumElements([]int16{42, 44, 46, 92}, GP8ExportOptions{})
	ids := gp8DrumArticulationIDs(hiHats)
	if ids[92] != 3 {
		t.Errorf("MIDI 92 articulation ID = %d, want 3", ids[92])
	}
}

func TestGP8CrashArticulationMetadata(t *testing.T) {
	element := gp8DrumElement(49, GP8ExportOptions{})
	if element.Name != "Crash High" || element.Type != "crash" || element.SoundbankName != "Master-Crash02" {
		t.Errorf("crash element = %#v", element)
	}
	articulation := element.Articulations.Articulations[0]
	if articulation.Name != "Crash high (hit)" || articulation.StaffLine != -2 || articulation.Noteheads != "noteheadHeavyX noteheadHeavyX noteheadHeavyX" || articulation.OutputMIDINumber != 49 {
		t.Errorf("crash articulation = %#v", articulation)
	}
}

func TestExportGP8PitchedNotesHaveConsumerMetadata(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.Name = "Guitar"
	track.PercussionTrack = false
	track.Strings = []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}, {Number: 3, Value: 55}, {Number: 4, Value: 50}, {Number: 5, Value: 45}, {Number: 6, Value: 40}}
	song.Channels[0] = MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 30, Volume: 100, Balance: 64}
	for measureIndex := range track.Measures {
		for voiceIndex := range track.Measures[measureIndex].Voices {
			for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
				for noteIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
					note := &track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex]
					note.String = 1
					note.Value = 5
					for graceIndex := range note.Effect.Graces {
						note.Effect.Graces[graceIndex].Fret = 3
					}
				}
			}
		}
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	note := document.Notes.Notes[0]
	if note.InstrumentArticulation == nil || *note.InstrumentArticulation != 0 {
		t.Errorf("pitched note articulation = %v, want 0", note.InstrumentArticulation)
	}
	properties := make(map[string]gpifProperty)
	for _, property := range note.Properties.Properties {
		properties[property.Name] = property
	}
	for _, name := range []string{"ConcertPitch", "TransposedPitch"} {
		pitch := properties[name].Pitch
		if pitch == nil || *pitch != (gpifPitch{Step: "G", Octave: 4}) {
			t.Errorf("%s = %#v, want G4", name, pitch)
		}
	}
	if number := properties["Midi"].Number; number == nil || *number != 67 {
		t.Errorf("MIDI property = %v, want 67", number)
	}
}

func TestExportGP8PreservesBendCurves(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.Name = "Bass"
	track.PercussionTrack = false
	track.Strings = []GuitarString{{Number: 1, Value: 43}}
	song.Channels[0] = MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 33, Volume: 100, Balance: 64}

	bends := []*BendEffect{
		{Kind: BendTypeBend, Value: 25, Points: []BendPoint{{Position: 0, Value: 0}, {Position: 3, Value: 1}, {Position: 12, Value: 1}}},
		{Kind: BendTypePrebendRelease, Value: 50, Points: []BendPoint{{Position: 0, Value: 2}, {Position: 3, Value: 2}, {Position: 6, Value: 0}, {Position: 12, Value: 0}}},
		{Kind: BendTypePrebend, Points: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 0}}},
	}
	quarter := defaultDuration()
	track.Measures[0].Voices = []Voice{{Beats: make([]Beat, len(bends))}}
	for index, bend := range bends {
		track.Measures[0].Voices[0].Beats[index] = Beat{
			Duration: quarter,
			Status:   BeatStatusNormal,
			Notes:    []Note{{Value: int16(index + 3), String: 1, Velocity: Forte, Kind: NoteTypeNormal, Effect: NoteEffect{Bend: bend}}},
		}
	}
	for measureIndex := 1; measureIndex < len(track.Measures); measureIndex++ {
		track.Measures[measureIndex].Voices = []Voice{{Beats: []Beat{{Duration: quarter, Status: BeatStatusRest}}}}
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for index, want := range bends {
		got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[index].Notes[0].Effect.Bend
		if got == nil || got.Kind != want.Kind || got.Value != want.Value || !reflect.DeepEqual(got.Points, want.Points) {
			t.Errorf("bend %d = %#v, want %#v", index, got, want)
		}
	}
}

func TestExportGP8PreservesHairpins(t *testing.T) {
	song := syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Hairpin = HairpinCrescendo
	song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.Hairpin = HairpinDiminuendo

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	err = xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document)
	if err != nil {
		t.Fatal(err)
	}
	if got := document.Beats.Beats[1].Hairpin; got != "Crescendo" {
		t.Errorf("first beat hairpin = %q, want Crescendo", got)
	}
	if got := document.Beats.Beats[2].Hairpin; got != "Diminuendo" {
		t.Errorf("second beat hairpin = %q, want Diminuendo", got)
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	beats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	if beats[0].Effect.Hairpin != HairpinCrescendo {
		t.Errorf("first round-trip hairpin = %d, want Crescendo", beats[0].Effect.Hairpin)
	}
	if beats[1].Effect.Hairpin != HairpinDiminuendo {
		t.Errorf("second round-trip hairpin = %d, want Diminuendo", beats[1].Effect.Hairpin)
	}
}

func TestExportGP8PreservesTrackLyricsAndSoundAutomations(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.Lyrics = []TrackLyricLine{{Text: "Hel-lo world", Offset: 2}}
	track.Sounds = []TrackSound{
		{Name: "Clean", Label: "Clean", Path: "Stringed/Electric Guitars/Clean Guitar", Role: "Factory", Program: 27},
		{Name: "Distortion", Label: "Distortion", Path: "Stringed/Electric Guitars/Distortion Guitar", Role: "Factory", Program: 30},
	}
	track.SoundAutomations = []SoundAutomation{{Bar: 0, Sound: 0}, {Bar: 1, Position: 0.5, Sound: 1}}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip.Tracks[0].Lyrics, track.Lyrics) {
		t.Errorf("lyrics = %#v, want %#v", roundTrip.Tracks[0].Lyrics, track.Lyrics)
	}
	if !reflect.DeepEqual(roundTrip.Tracks[0].Sounds, track.Sounds) {
		t.Errorf("sounds = %#v, want %#v", roundTrip.Tracks[0].Sounds, track.Sounds)
	}
	if !reflect.DeepEqual(roundTrip.Tracks[0].SoundAutomations, track.SoundAutomations) {
		t.Errorf("sound automations = %#v, want %#v", roundTrip.Tracks[0].SoundAutomations, track.SoundAutomations)
	}
}

func TestExportGP8PreservesHarmonicsAndWhammyCurves(t *testing.T) {
	song := syntheticGP8Song()
	harmonicFret := 2.4
	note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, FretFloat: &harmonicFret}
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar = &BendEffect{Points: []BendPoint{
		{Position: 0, Value: 0}, {Position: 6, Value: -32}, {Position: 12, Value: 0},
	}}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	foundHFret := false
	for _, rawNote := range document.Notes.Notes {
		for _, property := range rawNote.Properties.Properties {
			if property.Name != "HarmonicFret" {
				continue
			}
			foundHFret = property.HFret != nil && *property.HFret == "2.4" && property.Float == nil
		}
	}
	if !foundHFret {
		t.Error("GPIF fractional harmonic is not encoded as <HFret>2.4</HFret>")
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	beat := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0]
	if beat.Effect.TremoloBar == nil || len(beat.Effect.TremoloBar.Points) != 4 {
		t.Fatalf("round-trip whammy = %#v, want four GPIF control points", beat.Effect.TremoloBar)
	}
	parsedHarmonic := beat.Notes[0].Effect.Harmonic
	if parsedHarmonic == nil || parsedHarmonic.Kind != HarmonicTypeArtificial || parsedHarmonic.FretFloat == nil || *parsedHarmonic.FretFloat != harmonicFret {
		t.Errorf("round-trip harmonic = %#v, want artificial at fret %.1f", parsedHarmonic, harmonicFret)
	}
}

func TestExportGP8EmitsEnabledMutedAndHammerOnProperties(t *testing.T) {
	song := syntheticGP8Song()
	mutedTie := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	mutedTie.Kind = NoteTypeTie
	mutedTie.Effect.DeadNote = true
	hammerOn := &song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
	hammerOn.Effect.Hammer = true

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	err = xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Muted", "HopoOrigin"} {
		foundEnabled := false
		for _, rawNote := range document.Notes.Notes {
			for _, property := range rawNote.Properties.Properties {
				if property.Name == name && property.Enable != nil {
					foundEnabled = true
				}
			}
		}
		if !foundEnabled {
			t.Errorf("GPIF %s property is missing its required Enable element", name)
		}
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	parsedMutedTie := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	if parsedMutedTie.Kind != NoteTypeTie || !parsedMutedTie.Effect.DeadNote {
		t.Errorf("round-trip tied muted note = %#v, want both tie and muted semantics", parsedMutedTie)
	}
	parsedHammerOn := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
	if !parsedHammerOn.Effect.Hammer {
		t.Error("round trip dropped hammer-on semantics")
	}
}

func TestExportGP8RejectsInvalidSoundAutomation(t *testing.T) {
	song := syntheticGP8Song()
	song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Sound: 1}}
	if _, err := Export(song, ExportFormatGP8); err == nil || !strings.Contains(err.Error(), "sound index") {
		t.Fatalf("Export error = %v, want invalid sound index", err)
	}
}

func TestExportGP8UsesInstrumentClefs(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.Name = "Bass"
	track.PercussionTrack = false
	track.Strings = []GuitarString{{Number: 1, Value: 43}}
	song.Channels[0] = MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 33, Volume: 100, Balance: 64}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	for index, bar := range document.Bars.Bars {
		if bar.Clef != "F4" {
			t.Errorf("bass bar %d clef = %q, want F4", index+1, bar.Clef)
		}
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for index, measure := range roundTrip.Tracks[0].Measures {
		if measure.Clef != MeasureClefBass {
			t.Errorf("round-trip bass measure %d clef = %d, want bass", index+1, measure.Clef)
		}
	}

	song.Tracks[0].PercussionTrack = true
	data, err = Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err = zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	document = gpifDocument{}
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	for index, bar := range document.Bars.Bars {
		if bar.Clef != "Neutral" {
			t.Errorf("percussion bar %d clef = %q, want Neutral", index+1, bar.Clef)
		}
	}
}

func TestExportGP8WritesCompleteTieChain(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.PercussionTrack = false
	track.Strings = []GuitarString{{Number: 1, Value: 64}}
	song.Channels[0] = MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 30, Volume: 100, Balance: 64}
	quarter := defaultDuration()
	track.Measures[0].Voices = []Voice{{Beats: []Beat{
		{Duration: quarter, Status: BeatStatusNormal, Notes: []Note{{Value: 3, String: 1, Velocity: Forte, Kind: NoteTypeNormal, TieOrigin: true}}},
		{Duration: quarter, Status: BeatStatusNormal, Notes: []Note{{Value: 3, String: 1, Velocity: Forte, Kind: NoteTypeTie, TieOrigin: true}}},
		{Duration: quarter, Status: BeatStatusNormal, Notes: []Note{{Value: 3, String: 1, Velocity: Forte, Kind: NoteTypeTie}}},
	}}}
	for measureIndex := 1; measureIndex < len(track.Measures); measureIndex++ {
		track.Measures[measureIndex].Voices = []Voice{{Beats: []Beat{{Duration: quarter, Status: BeatStatusRest}}}}
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if len(document.Notes.Notes) != 3 {
		t.Fatalf("notes = %d, want 3", len(document.Notes.Notes))
	}
	want := []gpifTie{
		{Origin: "true", Destination: "false"},
		{Origin: "true", Destination: "true"},
		{Origin: "false", Destination: "true"},
	}
	for index, expected := range want {
		if tie := document.Notes.Notes[index].Tie; tie == nil || *tie != expected {
			t.Errorf("note %d tie = %#v, want %#v", index, tie, expected)
		}
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	parsedNotes := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	if !parsedNotes[0].Notes[0].TieOrigin || parsedNotes[0].Notes[0].Kind != NoteTypeNormal {
		t.Errorf("first parsed tie note = %#v", parsedNotes[0].Notes[0])
	}
	if !parsedNotes[1].Notes[0].TieOrigin || parsedNotes[1].Notes[0].Kind != NoteTypeTie {
		t.Errorf("middle parsed tie note = %#v", parsedNotes[1].Notes[0])
	}
	if parsedNotes[2].Notes[0].TieOrigin || parsedNotes[2].Notes[0].Kind != NoteTypeTie {
		t.Errorf("last parsed tie note = %#v", parsedNotes[2].Notes[0])
	}
}

func TestExportFileGP8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "score.gp")
	if err := ExportFile(path, syntheticGP8Song(), ExportFormatGP8); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestExportRejectsUnsupportedInput(t *testing.T) {
	if _, err := Export(nil, ExportFormatGP8); err == nil {
		t.Fatal("Export(nil) succeeded")
	}
	if _, err := Export(syntheticGP8Song(), ExportFormat(7)); err == nil {
		t.Fatal("Export accepted an unsupported target")
	}
	song := syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Value = 128
	if _, err := Export(song, ExportFormatGP8); err == nil {
		t.Fatal("Export accepted an out-of-range percussion MIDI value")
	}
	song = syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Value = 40
	if _, err := Export(song, ExportFormatGP8); err == nil {
		t.Fatal("Export accepted percussion without native Guitar Pro drum-kit metadata")
	} else if !strings.Contains(err.Error(), "percussion MIDI value 40 without a native Guitar Pro drum-kit articulation") {
		t.Fatalf("unsupported percussion error = %v", err)
	}
	song = syntheticGP8Song()
	song.Tracks[0].PercussionTrack = false
	song.Channels[0].Instrument = 128
	if _, err := Export(song, ExportFormatGP8); err == nil {
		t.Fatal("Export accepted an out-of-range MIDI program")
	}
	if _, err := ExportWithOptions(syntheticGP8Song(), ExportFormatGP8, ExportOptions{GP8: GP8ExportOptions{PercussionNoteheads: map[int16]GP8PercussionNotehead{46: 99}}}); err == nil {
		t.Fatal("Export accepted an unsupported percussion notehead")
	}
}

func syntheticGP8Song() *Song {
	quarter := defaultDuration()
	eighth := defaultDuration()
	eighth.Value = uint16(DurationEighth)
	dottedQuarter := defaultDuration()
	dottedQuarter.Dotted = true

	headers := make([]MeasureHeader, 3)
	for index := range headers {
		headers[index] = defaultMeasureHeader()
		headers[index].Number = uint16(index + 1)
	}
	headers[0].Marker = &Marker{Title: "Pickup"}
	headers[0].RepeatOpen = true
	headers[1].TimeSignature.Numerator = 3
	headers[1].Marker = &Marker{}
	headers[1].RepeatClose = 1
	headers[1].RepeatAlternative = 0b11
	headers[2].DoubleBar = true

	chord := &Chord{Name: "Kick + hat", Strings: []int8{0}}
	grace := GraceEffect{Fret: 38, Duration: 1, Velocity: Forte}
	measures := []Measure{
		{
			Number: 1,
			Voices: []Voice{{Beats: []Beat{
				{Duration: eighth, Status: BeatStatusNormal, Notes: []Note{{Value: 38, String: 1, Velocity: Forte, Kind: NoteTypeNormal, Effect: NoteEffect{Graces: []GraceEffect{grace}, AccentuatedNote: true}}, {Value: 42, Velocity: Forte, Kind: NoteTypeNormal}}},
				{Duration: eighth, Status: BeatStatusNormal, Effect: BeatEffects{Chord: chord}, Notes: []Note{{Value: 36, String: 1, Velocity: Forte, Kind: NoteTypeNormal}, {Value: 42, String: 1, Velocity: Forte, Kind: NoteTypeNormal}}},
				{Duration: quarter, Status: BeatStatusRest},
			}}},
		},
		{
			Number: 2,
			Voices: []Voice{{Beats: []Beat{{Duration: dottedQuarter, Status: BeatStatusNormal, Notes: []Note{{Value: 38, String: 1, Velocity: MinVelocity + VelocityIncrement*6, Kind: NoteTypeNormal, Effect: NoteEffect{HeavyAccentuatedNote: true}}}}}}},
		},
		{
			Number: 3,
			Voices: []Voice{{Beats: []Beat{{Duration: quarter, Status: BeatStatusRest}}}},
		},
	}

	return &Song{
		Name:           "Exporter fixture",
		Artist:         "go-guitar-pro",
		Notice:         []string{"First notice", "Second notice"},
		TempoName:      "Bright",
		Tempo:          132,
		Anacrusis:      true,
		MeasureHeaders: headers,
		Channels:       []MidiChannel{{Channel: DefaultPercussionChannel, EffectChannel: DefaultPercussionChannel, Instrument: 0, Volume: 100, Balance: 64}},
		Tracks: []Track{{
			Name:            "Percussion",
			Number:          0,
			Color:           0x336699,
			ChannelIndex:    0,
			PercussionTrack: true,
			Visible:         true,
			Strings:         []GuitarString{{Number: 1}},
			Measures:        measures,
		}},
	}
}

func readZipMember(t *testing.T, archive *zip.Reader, name string) []byte {
	t.Helper()
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if closeErr := reader.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	t.Fatalf("archive member %q not found", name)
	return nil
}
