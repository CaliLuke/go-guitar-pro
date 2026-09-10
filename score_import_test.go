// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"os"
	"testing"
)

func TestGPXPreservesGeneralMIDIProgramAndChannel(t *testing.T) {
	data, err := os.ReadFile("testdata/gp6/full-song.gpx")
	if err != nil {
		t.Fatal(err)
	}

	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Tracks) == 0 || len(song.Channels) == 0 {
		t.Fatalf("parsed %d tracks and %d channels", len(song.Tracks), len(song.Channels))
	}
	files, err := gpxReadFiles(data)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Tracks []struct {
			ID          string `xml:"id,attr"`
			GeneralMidi struct {
				Program          int `xml:"Program"`
				Port             int `xml:"Port"`
				PrimaryChannel   int `xml:"PrimaryChannel"`
				SecondaryChannel int `xml:"SecondaryChannel"`
			} `xml:"GeneralMidi"`
			MidiConnection gpifMidiConnection `xml:"MidiConnection"`
			Sounds         gpifSounds         `xml:"Sounds"`
		} `xml:"Tracks>Track"`
	}
	if err := xml.Unmarshal(files["score.gpif"], &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Tracks[0].GeneralMidi.Program != 73 || wire.Tracks[0].GeneralMidi.PrimaryChannel != 13 {
		t.Fatalf("track 0 GPIF GeneralMidi = %+v, want program 73 and channel 13", wire.Tracks[0].GeneralMidi)
	}

	channel := song.Channels[song.Tracks[0].ChannelIndex]
	if channel.Channel != 13 {
		t.Errorf("primary channel = %d, want GPIF GeneralMidi channel 13", channel.Channel)
	}
	if channel.Instrument != 73 {
		t.Errorf("program = %d, want GPIF GeneralMidi program 73", channel.Instrument)
	}
}

func TestGPIFPreservesAllTripletFeelVariants(t *testing.T) {
	data, err := os.ReadFile("testdata/gp7/triplet-feel.gp")
	if err != nil {
		t.Fatal(err)
	}

	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []TripletFeel{TripletFeel(3), TripletFeel(4), TripletFeel(5), TripletFeel(6)}
	for index, expected := range want {
		got := song.MeasureHeaders[index+2].TripletFeel
		if got != expected {
			t.Errorf("measure %d triplet feel = %d, want %d", index+2, got, expected)
		}
	}

	exported, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(exported)
	if err != nil {
		t.Fatal(err)
	}
	for index, expected := range want {
		if got := roundTrip.MeasureHeaders[index+2].TripletFeel; got != expected {
			t.Errorf("round-trip measure %d triplet feel = %d, want %d", index+2, got, expected)
		}
	}
}

func TestGPIFWordsAndMusicFillsMissingCredits(t *testing.T) {
	data, err := os.ReadFile("testdata/gp7/lyrics-null.gp")
	if err != nil {
		t.Fatal(err)
	}

	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if song.Words != "Septo" || song.Author != "Septo" {
		t.Errorf("words/author = %q/%q, want shared WordsAndMusic credit %q", song.Words, song.Author, "Septo")
	}
}

func TestGP4AlternateEndingDoesNotCrossPreviousRepeat(t *testing.T) {
	data, err := os.ReadFile("testdata/gp4/Repeat.gp4")
	if err != nil {
		t.Fatal(err)
	}

	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := song.MeasureHeaders[3].RepeatAlternative; got != 247 {
		t.Errorf("fourth measure alternate endings = %08b, want %08b", got, uint8(247))
	}
}

func TestGPIFConsolidatesDuplicatePlaybackChannels(t *testing.T) {
	tests := []struct {
		path string
		want map[int]uint8
	}{
		{path: "testdata/gp6/full-song.gpx", want: map[int]uint8{8: 6, 9: 19}},
		{path: "testdata/gp7/canon-audio-track.gp", want: map[int]uint8{6: 11, 7: 13}},
		{path: "testdata/gp7/faulty.gp", want: map[int]uint8{1: 2, 2: 4, 3: 6, 4: 8, 5: 9}},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			song := parseTestFixture(t, test.path)
			for trackIndex, want := range test.want {
				track := song.Tracks[trackIndex]
				if got := song.Channels[track.ChannelIndex].Channel; got != want {
					t.Errorf("track %d primary channel = %d, want %d", trackIndex, got, want)
				}
			}
		})
	}
}
