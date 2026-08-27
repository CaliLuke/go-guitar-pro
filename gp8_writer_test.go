// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"path/filepath"
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
	if err := xml.Unmarshal(gpifData, &document); err != nil {
		t.Fatal(err)
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
	if graceNote.Value != 38 || graceNote.Effect.Grace == nil || graceNote.Effect.Grace.Fret != 38 || !graceNote.Effect.AccentuatedNote {
		t.Errorf("grace snare = %#v", graceNote)
	}
	if pickupBeats[0].Duration.Value != uint16(DurationEighth) || graceNote.Velocity != Forte {
		t.Errorf("grace beat duration/velocity = %d/%d", pickupBeats[0].Duration.Value, graceNote.Velocity)
	}
	if len(pickupBeats[0].Notes) != 2 || pickupBeats[0].Notes[1].Effect.Grace != nil || pickupBeats[0].Notes[1].String != 0 {
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
					if note.Effect.Grace != nil {
						graceCount++
						if note.Value != 38 || note.Effect.Grace.Fret != 38 {
							t.Errorf("grace articulation = note %d grace %d, want 38", note.Value, note.Effect.Grace.Fret)
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
	grace := &GraceEffect{Fret: 38, Duration: 1, Velocity: Forte}
	measures := []Measure{
		{
			Number: 1,
			Voices: []Voice{{Beats: []Beat{
				{Duration: eighth, Status: BeatStatusNormal, Notes: []Note{{Value: 38, String: 1, Velocity: Forte, Kind: NoteTypeNormal, Effect: NoteEffect{Grace: grace, AccentuatedNote: true}}, {Value: 42, Velocity: Forte, Kind: NoteTypeNormal}}},
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
