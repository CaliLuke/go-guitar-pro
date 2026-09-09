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
	"sort"
	"strconv"
	"strings"
)

// GPIF XML structures shared by parsing and export.

type gpifDocument struct {
	XMLName      xml.Name          `xml:"GPIF"`
	GPVersion    string            `xml:"GPVersion,omitempty"`
	GPRevision   gpifRevision      `xml:"GPRevision"`
	Encoding     gpifEncoding      `xml:"Encoding"`
	Score        gpifScore         `xml:"Score"`
	MasterTrack  gpifMasterTrack   `xml:"MasterTrack"`
	BackingTrack *gpifBackingTrack `xml:"BackingTrack,omitempty"`
	Tracks       gpifTracks        `xml:"Tracks"`
	MasterBars   gpifMasterBars    `xml:"MasterBars"`
	Bars         gpifBars          `xml:"Bars"`
	Voices       gpifVoices        `xml:"Voices"`
	Beats        gpifBeats         `xml:"Beats"`
	Notes        gpifNotes         `xml:"Notes"`
	Rhythms      gpifRhythms       `xml:"Rhythms"`
	Assets       gpifAssets        `xml:"Assets,omitempty"`
}

type gpifRevision struct {
	Required    string `xml:"required,attr,omitempty"`
	Recommended string `xml:"recommended,attr,omitempty"`
	Value       string `xml:",chardata"`
}

type gpifEncoding struct {
	Description string `xml:"EncodingDescription,omitempty"`
}

type gpifScore struct {
	Title        string `xml:"Title"`
	SubTitle     string `xml:"SubTitle"`
	Artist       string `xml:"Artist"`
	Album        string `xml:"Album"`
	Words        string `xml:"Words"`
	Music        string `xml:"Music"`
	Copyright    string `xml:"Copyright"`
	Tabber       string `xml:"Tabber"`
	Instructions string `xml:"Instructions"`
	Notices      string `xml:"Notices"`
}

type gpifMasterTrack struct {
	Tracks      string          `xml:"Tracks"`
	Anacrusis   *struct{}       `xml:"Anacrusis,omitempty"`
	Automations gpifAutomations `xml:"Automations"`
}

type gpifAutomations struct {
	Automations []gpifAutomation `xml:"Automation"`
}

type gpifAutomation struct {
	Type     string              `xml:"Type"`
	Linear   bool                `xml:"Linear"`
	Value    gpifAutomationValue `xml:"Value"`
	Visible  string              `xml:"Visible,omitempty"`
	Text     string              `xml:"Text,omitempty"`
	Bar      int                 `xml:"Bar"`
	Position float64             `xml:"Position"`
}

type gpifAutomationValue struct {
	Text          string `xml:",chardata"`
	BarIndex      string `xml:"BarIndex,omitempty"`
	BarOccurrence string `xml:"BarOccurrence,omitempty"`
	ModifiedTempo string `xml:"ModifiedTempo,omitempty"`
	OriginalTempo string `xml:"OriginalTempo,omitempty"`
	FrameOffset   string `xml:"FrameOffset,omitempty"`
}

type gpifBackingTrack struct {
	Name         string `xml:"Name"`
	Enabled      bool   `xml:"Enabled"`
	Source       string `xml:"Source"`
	AssetID      string `xml:"AssetId"`
	FramePadding string `xml:"FramePadding"`
}

type gpifAssets struct {
	Assets []gpifAsset `xml:"Asset"`
}

type gpifAsset struct {
	ID               string `xml:"id,attr"`
	OriginalFilePath string `xml:"OriginalFilePath"`
	OriginalFileSHA1 string `xml:"OriginalFileSha1"`
	EmbeddedFilePath string `xml:"EmbeddedFilePath"`
}

type gpifTracks struct {
	Tracks []gpifTrack `xml:"Track"`
}

type gpifTrack struct {
	ID               string              `xml:"id,attr"`
	Name             string              `xml:"Name"`
	Color            string              `xml:"Color,omitempty"`
	Instrument       *gpifInstrument     `xml:"Instrument,omitempty"`
	InstrumentSet    *gpifInstrumentSet  `xml:"InstrumentSet,omitempty"`
	NotationPatch    *gpifInstrumentSet  `xml:"NotationPatch,omitempty"`
	GeneralMidi      *gpifGeneralMidi    `xml:"GeneralMidi,omitempty"`
	Staves           gpifStaves          `xml:"Staves"`
	Properties       []gpifStaffProperty `xml:"Properties>Property"`
	Sounds           gpifSounds          `xml:"Sounds"`
	Automations      gpifAutomations     `xml:"Automations"`
	Transpose        *gpifTranspose      `xml:"Transpose,omitempty"`
	RSE              *gpifTrackRSE       `xml:"RSE,omitempty"`
	MidiConnection   gpifMidiConnection  `xml:"MidiConnection"`
	PlaybackState    string              `xml:"PlaybackState,omitempty"`
	AudioEngineState string              `xml:"AudioEngineState,omitempty"`
	Lyrics           *gpifLyrics         `xml:"Lyrics,omitempty"`
}

type gpifLyrics struct {
	Dispatched bool            `xml:"dispatched,attr"`
	Lines      []gpifLyricLine `xml:"Line"`
}

type gpifLyricLine struct {
	Text   string `xml:"Text"`
	Offset int    `xml:"Offset"`
}

type gpifTrackRSE struct {
	ChannelStrip gpifChannelStrip `xml:"ChannelStrip"`
}

type gpifChannelStrip struct {
	Parameters  string          `xml:"Parameters"`
	Automations gpifAutomations `xml:"Automations"`
}

type gpifMidiConnection struct {
	Port             int `xml:"Port"`
	PrimaryChannel   int `xml:"PrimaryChannel"`
	SecondaryChannel int `xml:"SecondaryChannel"`
}

type gpifInstrument struct {
	Ref string `xml:"ref,attr"`
}

type gpifGeneralMidi struct {
	PrimaryChannel int `xml:"PrimaryChannel"`
}

type gpifStaves struct {
	Staff []gpifStaff `xml:"Staff"`
}

type gpifStaff struct {
	Properties []gpifStaffProperty `xml:"Properties>Property"`
}

type gpifStaffProperty struct {
	Name    string     `xml:"name,attr"`
	Pitches string     `xml:"Pitches,omitempty"`
	Label   string     `xml:"Label,omitempty"`
	Fret    *int       `xml:"Fret,omitempty"`
	Items   *gpifItems `xml:"Items,omitempty"`
}

type gpifItems struct {
	Items []gpifItem `xml:"Item"`
}

type gpifItem struct {
	ID      string       `xml:"id,attr,omitempty"`
	Name    string       `xml:"name,attr,omitempty"`
	Diagram *gpifDiagram `xml:"Diagram,omitempty"`
	Chord   *struct{}    `xml:"Chord,omitempty"`
}

type gpifDiagram struct {
	StringCount int                   `xml:"stringCount,attr,omitempty"`
	FretCount   int                   `xml:"fretCount,attr,omitempty"`
	BaseFret    int                   `xml:"baseFret,attr,omitempty"`
	Frets       []gpifDiagramFret     `xml:"Fret"`
	Fingering   *gpifDiagramFingering `xml:"Fingering,omitempty"`
	Properties  []gpifDiagramProperty `xml:"Property"`
}

type gpifDiagramFret struct {
	String int `xml:"string,attr"`
	Fret   int `xml:"fret,attr"`
}

type gpifDiagramFingering struct {
	Positions []gpifDiagramPosition `xml:"Position"`
}

type gpifDiagramPosition struct {
	Fret   int    `xml:"fret,attr"`
	Finger string `xml:"finger,attr"`
}

type gpifDiagramProperty struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type gpifInstrumentSet struct {
	Name      string       `xml:"Name,omitempty"`
	Type      string       `xml:"Type,omitempty"`
	LineCount int          `xml:"LineCount,omitempty"`
	Elements  gpifElements `xml:"Elements"`
}

type gpifElements struct {
	Elements []gpifElement `xml:"Element"`
}

type gpifElement struct {
	Name          string            `xml:"Name,omitempty"`
	Type          string            `xml:"Type,omitempty"`
	SoundbankName string            `xml:"SoundbankName"`
	Articulations gpifArticulations `xml:"Articulations"`
}

type gpifArticulations struct {
	Articulations []gpifArticulation `xml:"Articulation"`
}

type gpifArticulation struct {
	Name               string `xml:"Name,omitempty"`
	StaffLine          int    `xml:"StaffLine"`
	Noteheads          string `xml:"Noteheads,omitempty"`
	TechniquePlacement string `xml:"TechniquePlacement,omitempty"`
	TechniqueSymbol    string `xml:"TechniqueSymbol"`
	InputMIDINumbers   string `xml:"InputMidiNumbers"`
	OutputRSESound     string `xml:"OutputRSESound"`
	OutputMIDINumber   int    `xml:"OutputMidiNumber"`
}

func gpifReadPercussionArticulations(instrumentSet, notationPatch *gpifInstrumentSet) []PercussionArticulation {
	if instrumentSet == nil {
		return nil
	}
	count := 0
	for _, element := range instrumentSet.Elements.Elements {
		count += len(element.Articulations.Articulations)
	}
	articulations := make([]PercussionArticulation, 0, count)
	type articulationName struct {
		element string
		name    string
	}
	byName := make(map[articulationName]int, count)
	for _, element := range instrumentSet.Elements.Elements {
		for _, source := range element.Articulations.Articulations {
			noteheads := strings.Fields(source.Noteheads)
			articulation := PercussionArticulation{
				ElementName:          element.Name,
				ElementType:          element.Type,
				ElementSoundbankName: element.SoundbankName,
				Name:                 source.Name,
				StaffLine:            source.StaffLine,
				TechniquePlacement:   source.TechniquePlacement,
				TechniqueSymbol:      source.TechniqueSymbol,
				OutputRSESound:       source.OutputRSESound,
				OutputMIDINumber:     source.OutputMIDINumber,
			}
			if articulation.TechniquePlacement == "" {
				articulation.TechniquePlacement = "outside"
			}
			if len(noteheads) > 0 {
				articulation.NoteheadDefault = noteheads[0]
			}
			if len(noteheads) > 1 {
				articulation.NoteheadHalf = noteheads[1]
			}
			if len(noteheads) > 2 {
				articulation.NoteheadWhole = noteheads[2]
			}
			if articulation.NoteheadHalf == "" {
				articulation.NoteheadHalf = articulation.NoteheadDefault
			}
			if articulation.NoteheadWhole == "" {
				articulation.NoteheadWhole = articulation.NoteheadDefault
			}
			for _, value := range strings.Fields(source.InputMIDINumbers) {
				if midi, err := strconv.Atoi(value); err == nil {
					articulation.InputMIDINumbers = append(articulation.InputMIDINumbers, midi)
				}
			}
			byName[articulationName{element: element.Name, name: source.Name}] = len(articulations)
			articulations = append(articulations, articulation)
		}
	}
	if notationPatch != nil {
		for _, element := range notationPatch.Elements.Elements {
			for _, patch := range element.Articulations.Articulations {
				if index, ok := byName[articulationName{element: element.Name, name: patch.Name}]; ok {
					articulations[index].StaffLine = patch.StaffLine
				}
			}
		}
	}
	return articulations
}

func (t *gpifTrack) isPercussionTrack() bool {
	if t.InstrumentSet != nil && (t.InstrumentSet.Type == "drums" || t.InstrumentSet.Type == "percussion" || t.InstrumentSet.Type == "drumKit") {
		return true
	}
	if t.Instrument != nil && t.Instrument.Ref == "drmkt" {
		return true
	}
	if t.GeneralMidi == nil || t.GeneralMidi.PrimaryChannel < 0 || t.GeneralMidi.PrimaryChannel > math.MaxUint8 {
		channel := t.MidiConnection.Port*16 + t.MidiConnection.PrimaryChannel
		if channel < 0 || channel > math.MaxUint8 {
			return false
		}
		midiChannel := MidiChannel{Channel: uint8(channel)}
		return midiChannel.isPercussionChannel()
	}
	channel := MidiChannel{Channel: uint8(t.GeneralMidi.PrimaryChannel)}
	return channel.isPercussionChannel()
}

type gpifTranspose struct {
	Chromatic int `xml:"Chromatic"`
	Octave    int `xml:"Octave"`
}

type gpifSounds struct {
	Sounds []gpifSound `xml:"Sound"`
}

type gpifSound struct {
	Name    string `xml:"Name"`
	Label   string `xml:"Label,omitempty"`
	Path    string `xml:"Path,omitempty"`
	Role    string `xml:"Role,omitempty"`
	Program int    `xml:"MIDI>Program"`
	Channel int    `xml:"MIDI>PrimaryChannel"`
}

type gpifMasterBars struct {
	MasterBars []gpifMasterBar `xml:"MasterBar"`
}

type gpifMasterBar struct {
	Section          *gpifSection `xml:"Section,omitempty"`
	Key              gpifKey      `xml:"Key"`
	Time             string       `xml:"Time"`
	Bars             string       `xml:"Bars"`
	AlternateEndings string       `xml:"AlternateEndings,omitempty"`
	DoubleBar        *struct{}    `xml:"DoubleBar,omitempty"`
	TripletFeel      string       `xml:"TripletFeel,omitempty"`
	Repeat           *gpifRepeat  `xml:"Repeat,omitempty"`
}

type gpifKey struct {
	Mode            string `xml:"Mode"`
	AccidentalCount int    `xml:"AccidentalCount"`
}

type gpifRepeat struct {
	Start string `xml:"start,attr,omitempty"`
	End   string `xml:"end,attr,omitempty"`
	Count int    `xml:"count,attr,omitempty"`
}

type gpifSection struct {
	Letter string `xml:"Letter"`
	Text   string `xml:"Text"`
}

type gpifBars struct {
	Bars []gpifBar `xml:"Bar"`
}

type gpifBar struct {
	ID     string `xml:"id,attr"`
	Voices string `xml:"Voices"`
	Clef   string `xml:"Clef"`
}

type gpifVoices struct {
	Voices []gpifVoice `xml:"Voice"`
}

type gpifVoice struct {
	ID    string `xml:"id,attr"`
	Beats string `xml:"Beats"`
}

type gpifBeats struct {
	Beats []gpifBeat `xml:"Beat"`
}

type gpifBeat struct {
	ID         string         `xml:"id,attr"`
	Rhythm     gpifRhythmRef  `xml:"Rhythm"`
	Notes      string         `xml:"Notes,omitempty"`
	Chord      string         `xml:"Chord,omitempty"`
	Dynamic    string         `xml:"Dynamic,omitempty"`
	GraceNotes string         `xml:"GraceNotes,omitempty"`
	Fadding    string         `xml:"Fadding,omitempty"`
	Tremolo    string         `xml:"Tremolo,omitempty"`
	Arpeggio   string         `xml:"Arpeggio,omitempty"`
	Hairpin    string         `xml:"Hairpin,omitempty"`
	FreeText   string         `xml:"FreeText,omitempty"`
	Ottavia    string         `xml:"Ottavia,omitempty"`
	Wah        string         `xml:"Wah,omitempty"`
	Whammy     *gpifWhammy    `xml:"Whammy,omitempty"`
	Properties gpifProperties `xml:"Properties"`
}

type gpifWhammy struct {
	OriginValue       string `xml:"originValue,attr"`
	MiddleValue       string `xml:"middleValue,attr"`
	DestinationValue  string `xml:"destinationValue,attr"`
	OriginOffset      string `xml:"originOffset,attr"`
	MiddleOffset1     string `xml:"middleOffset1,attr"`
	MiddleOffset2     string `xml:"middleOffset2,attr"`
	DestinationOffset string `xml:"destinationOffset,attr"`
}

type gpifRhythmRef struct {
	Ref string `xml:"ref,attr"`
}

type gpifProperties struct {
	Properties []gpifProperty `xml:"Property"`
}

type gpifProperty struct {
	Fret      *int       `xml:"Fret"`
	String    *float64   `xml:"String"`
	Pitch     *gpifPitch `xml:"Pitch"`
	Float     *string    `xml:"Float"`
	HFret     *string    `xml:"HFret"`
	Enable    *string    `xml:"Enable"`
	Number    *int       `xml:"Number"`
	HType     *string    `xml:"HType"`
	Flags     *string    `xml:"Flags"`
	Direction *string    `xml:"Direction"`
	Strength  *string    `xml:"Strength"`
	Name      string     `xml:"name,attr"`
}

type gpifPitch struct {
	Step       string `xml:"Step"`
	Accidental string `xml:"Accidental"`
	Octave     int    `xml:"Octave"`
}

type gpifNotes struct {
	Notes []gpifNote `xml:"Note"`
}

type gpifNote struct {
	LetRing                *string        `xml:"LetRing,omitempty"`
	Trill                  *gpifTrill     `xml:"Trill,omitempty"`
	Tie                    *gpifTie       `xml:"Tie,omitempty"`
	ID                     string         `xml:"id,attr"`
	InstrumentArticulation *int           `xml:"InstrumentArticulation,omitempty"`
	Vibrato                string         `xml:"Vibrato,omitempty"`
	AntiAccent             string         `xml:"AntiAccent,omitempty"`
	Properties             gpifProperties `xml:"Properties"`
	Accent                 int            `xml:"Accent,omitempty"`
}

type gpifTrill struct {
	Fret int `xml:",chardata"`
}

type gpifTie struct {
	Origin      string `xml:"origin,attr"`
	Destination string `xml:"destination,attr"`
}

type gpifRhythms struct {
	Rhythms []gpifRhythm `xml:"Rhythm"`
}

type gpifRhythm struct {
	AugmentationDot *gpifAugDot `xml:"AugmentationDot"`
	PrimaryTuplet   *gpifTuplet `xml:"PrimaryTuplet"`
	ID              string      `xml:"id,attr"`
	NoteValue       string      `xml:"NoteValue"`
}

type gpifAugDot struct {
	Count int `xml:"count,attr"`
}

type gpifTuplet struct {
	Num int `xml:"num,attr"`
	Den int `xml:"den,attr"`
}

// parseGPIF parses a GPIF (Guitar Pro Interchange Format) XML document into a Song.
func parseGPIF(data []byte) (*Song, error) {
	return parseGPIFWithContext(data, nil)
}

func parseGPIFWithContext(data []byte, context *parseContext) (*Song, error) {
	if err := gpifAuditXML(data, context); err != nil {
		return nil, fmt.Errorf("parsing GPIF XML: %w", err)
	}
	var doc gpifDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing GPIF XML: %w", err)
	}
	if context != nil && context.format != "GP6" {
		switch {
		case strings.HasPrefix(doc.GPVersion, "7"):
			context.setFormat("GP7")
		case strings.HasPrefix(doc.GPVersion, "8"):
			context.setFormat("GP8")
		}
	}
	gpifAuditDiagnostics(doc, context)

	song := &Song{
		Tempo:     120,
		TempoName: "Moderate",
	}
	song.Version = gpifVersion(doc.GPVersion)
	song.Anacrusis = doc.MasterTrack.Anacrusis != nil

	// Score info
	song.Name = doc.Score.Title
	song.Subtitle = doc.Score.SubTitle
	song.Artist = doc.Score.Artist
	song.Album = doc.Score.Album
	song.Words = doc.Score.Words
	song.Author = doc.Score.Music
	song.Copyright = doc.Score.Copyright
	song.Transcriber = doc.Score.Tabber
	song.Instructions = doc.Score.Instructions
	if doc.Score.Notices != "" {
		song.Notice = strings.Split(doc.Score.Notices, "\n")
	}

	gpifReadBackingTrack(doc, song)
	gpifReadSyncPoints(doc.MasterTrack.Automations.Automations, song)
	gpifReadTempoAutomations(doc.MasterTrack.Automations.Automations, song)

	// Build rhythm lookup
	rhythmMap := make(map[string]*gpifRhythm)
	for i := range doc.Rhythms.Rhythms {
		r := &doc.Rhythms.Rhythms[i]
		rhythmMap[r.ID] = r
	}

	// Build note lookup
	noteMap := make(map[string]*gpifNote)
	for i := range doc.Notes.Notes {
		n := &doc.Notes.Notes[i]
		noteMap[n.ID] = n
	}

	// Build beat lookup
	beatMap := make(map[string]*gpifBeat)
	for i := range doc.Beats.Beats {
		b := &doc.Beats.Beats[i]
		beatMap[b.ID] = b
	}

	// Build voice lookup
	voiceMap := make(map[string]*gpifVoice)
	for i := range doc.Voices.Voices {
		v := &doc.Voices.Voices[i]
		voiceMap[v.ID] = v
	}

	// Build bar lookup
	barMap := make(map[string]*gpifBar)
	for i := range doc.Bars.Bars {
		b := &doc.Bars.Bars[i]
		barMap[b.ID] = b
	}

	// Parse tracks
	trackIDs := splitIDs(doc.MasterTrack.Tracks)
	trackChordMaps := make([]map[string]Chord, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		track := defaultTrack()
		track.Number = int32(len(song.Tracks))
		chordMap := make(map[string]Chord)
		for _, t := range doc.Tracks.Tracks {
			if t.ID == trackID {
				track.Name = t.Name
				track.PercussionTrack = t.isPercussionTrack()
				if track.PercussionTrack {
					track.PercussionArticulations = gpifReadPercussionArticulations(t.InstrumentSet, t.NotationPatch)
				}
				if t.Lyrics != nil {
					for _, line := range t.Lyrics.Lines {
						track.Lyrics = append(track.Lyrics, TrackLyricLine(line))
					}
				}
				for _, sound := range t.Sounds.Sounds {
					track.Sounds = append(track.Sounds, TrackSound{
						Name: sound.Name, Label: sound.Label, Path: sound.Path,
						Role: sound.Role, Program: int32(sound.Program),
					})
				}
				for _, automation := range t.Automations.Automations {
					if automation.Type != "Sound" {
						continue
					}
					for soundIndex, sound := range track.Sounds {
						if automation.Value.Text == sound.Path+";"+sound.Name+";"+sound.Role {
							track.SoundAutomations = append(track.SoundAutomations, SoundAutomation{
								Bar: automation.Bar, Position: automation.Position, Sound: soundIndex,
							})
							break
						}
					}
				}
				staffCount := max(1, len(t.Staves.Staff))
				if t.Instrument != nil && (strings.HasSuffix(t.Instrument.Ref, "-gs") || strings.HasSuffix(t.Instrument.Ref, "GrandStaff")) {
					staffCount = max(2, staffCount)
				}
				lineCount := 5
				if t.InstrumentSet != nil && t.InstrumentSet.LineCount > 0 {
					lineCount = t.InstrumentSet.LineCount
				}
				if t.NotationPatch != nil && t.NotationPatch.LineCount > 0 {
					lineCount = t.NotationPatch.LineCount
				}
				trackStrings := append([]GuitarString(nil), track.Strings...)
				if parsed := gpifReadStaffStrings(gpifStaff{Properties: t.Properties}); parsed != nil {
					trackStrings = parsed
				}
				track.Staves = make([]Staff, staffCount)
				for staffIndex := range track.Staves {
					strings := append([]GuitarString(nil), trackStrings...)
					if staffIndex < len(t.Staves.Staff) {
						if parsed := gpifReadStaffStrings(t.Staves.Staff[staffIndex]); parsed != nil {
							strings = parsed
						}
					}
					track.Staves[staffIndex] = Staff{
						Strings:                   strings,
						PercussionTrack:           track.PercussionTrack,
						StandardNotationLineCount: lineCount,
					}
				}
				track.Strings = track.Staves[0].Strings
				chordMap = gpifReadChordMap(t)
				// Parse color
				if t.Color != "" {
					parts := splitIDs(t.Color)
					if len(parts) >= 3 {
						r, _ := strconv.Atoi(parts[0])
						g, _ := strconv.Atoi(parts[1])
						b, _ := strconv.Atoi(parts[2])
						track.Color = int32(r)*65536 + int32(g)*256 + int32(b)
					}
				}
				ch := defaultMidiChannel()
				if len(t.Sounds.Sounds) > 0 {
					ch.Instrument = int32(t.Sounds.Sounds[0].Program)
				}
				ch.Channel = gpifMIDIChannel(t.MidiConnection.Port, t.MidiConnection.PrimaryChannel)
				ch.EffectChannel = gpifMIDIChannel(t.MidiConnection.Port, t.MidiConnection.SecondaryChannel)
				if t.RSE != nil {
					gpifApplyChannelStrip(t.RSE.ChannelStrip.Parameters, &ch)
					gpifReadVolumeAutomations(
						t.RSE.ChannelStrip.Automations.Automations,
						len(song.Tracks),
						song,
					)
				}
				song.Channels = append(song.Channels, ch)
				track.ChannelIndex = len(song.Channels) - 1
				track.Mute = t.PlaybackState == "Mute"
				track.Solo = t.PlaybackState == "Solo"
				break
			}
		}
		if len(track.Staves) == 0 {
			track.populateSingleStaff()
		}
		song.Tracks = append(song.Tracks, track)
		trackChordMaps = append(trackChordMaps, chordMap)
	}

	// Parse master bars → measure headers + measures
	fallbackArticulations := make([]gpifPercussionFallbacks, len(song.Tracks))
	for trackIndex := range song.Tracks {
		fallbackArticulations[trackIndex].tableLength = len(song.Tracks[trackIndex].PercussionArticulations)
	}
	for mbIdx, mb := range doc.MasterBars.MasterBars {
		mh := defaultMeasureHeader()
		mh.Number = uint16(mbIdx + 1)

		// Time signature
		if mb.Time != "" {
			parts := strings.Split(mb.Time, "/")
			if len(parts) == 2 {
				if num, err := strconv.Atoi(parts[0]); err == nil {
					mh.TimeSignature.Numerator = int8(num)
				}
				if den, err := strconv.Atoi(parts[1]); err == nil {
					mh.TimeSignature.Denominator.Value = uint16(den)
				}
			}
		}

		// Key signature
		mh.KeySignature.Key = int8(mb.Key.AccidentalCount)
		mh.KeySignature.IsMinor = mb.Key.Mode == "Minor"

		if mb.Repeat != nil {
			mh.RepeatOpen = mb.Repeat.Start == "true"
			if mb.Repeat.End == "true" && mb.Repeat.Count > 0 {
				mh.RepeatClose = int8(mb.Repeat.Count - 1)
			}
		}
		for _, ending := range splitIDs(mb.AlternateEndings) {
			number, err := strconv.Atoi(ending)
			if err == nil && number >= 1 && number <= 8 {
				mh.RepeatAlternative |= 1 << (number - 1)
			}
		}

		// Section marker
		if mb.Section != nil {
			title := mb.Section.Text
			if title == "" {
				title = mb.Section.Letter
			}
			mh.Marker = &Marker{Title: title}
		}

		// Double bar
		mh.DoubleBar = mb.DoubleBar != nil

		// Triplet feel
		switch mb.TripletFeel {
		case "Triplet8th":
			mh.TripletFeel = TripletFeelEighth
		case "Triplet16th":
			mh.TripletFeel = TripletFeelSixteenth
		}

		song.MeasureHeaders = append(song.MeasureHeaders, mh)

		// GPIF lists bars vertically by staff, advancing the track only after its final staff.
		barIDs := splitIDs(mb.Bars)
		barIndex := 0
		newMeasure := func(trackIndex, staffIndex int) Measure {
			measure := defaultMeasure()
			measure.Number = mbIdx + 1
			measure.TrackIndex = trackIndex
			measure.StaffIndex = staffIndex
			measure.HeaderIndex = mbIdx
			measure.TimeSignature = mh.TimeSignature
			measure.KeySignature = mh.KeySignature
			measure.HasDoubleBar = mh.DoubleBar
			return measure
		}
		for trackIdx := range song.Tracks {
			track := &song.Tracks[trackIdx]
			for staffIdx := range track.Staves {
				if barIndex >= len(barIDs) {
					break
				}
				barID := barIDs[barIndex]
				barIndex++
				if barID == "-1" {
					for skippedStaffIdx := range track.Staves {
						measure := newMeasure(trackIdx, skippedStaffIdx)
						track.Staves[skippedStaffIdx].Measures = append(track.Staves[skippedStaffIdx].Measures, measure)
					}
					break
				}
				staff := &track.Staves[staffIdx]
				m := newMeasure(trackIdx, staffIdx)

				if bar, ok := barMap[barID]; ok {
					m.Clef = gpifMeasureClef(bar.Clef)
					voiceIDs := splitIDs(bar.Voices)
					for _, voiceID := range voiceIDs {
						if voiceID == "-1" {
							continue
						}
						voice := Voice{}
						var pendingGrace []gpifPendingGrace
						if v, ok := voiceMap[voiceID]; ok {
							beatIDs := splitIDs(v.Beats)
							for _, beatID := range beatIDs {
								if beatID == "-1" {
									continue
								}
								beat := defaultBeat()
								isGrace := false
								graceOnBeat := false
								if b, ok := beatMap[beatID]; ok {
									if r, ok := rhythmMap[b.Rhythm.Ref]; ok {
										beat.Duration = gpifRhythmToDuration(r)
									}

									beat.Effect.FadeIn = b.Fadding == "FadeIn"
									beat.Effect.Hairpin = gpifHairpin(b.Hairpin)
									gpifApplyBeatEffects(b, &beat)
									if chord, ok := trackChordMaps[trackIdx][b.Chord]; ok {
										beat.Effect.Chord = &chord
									}
									switch b.GraceNotes {
									case "OnBeat":
										isGrace = true
										graceOnBeat = true
									case "BeforeBeat":
										isGrace = true
									}

									velocity := gpifDynamicToVelocity(b.Dynamic)
									beat.Dynamics = velocity
									for _, noteID := range splitIDs(b.Notes) {
										if noteID == "-1" {
											continue
										}
										if n, ok := noteMap[noteID]; ok {
											note := gpifNoteToNote(n, len(staff.Strings), staff.PercussionTrack)
											gpifNormalizePercussionArticulation(track, &note, &fallbackArticulations[trackIdx])
											note.Velocity = velocity
											beat.Notes = append(beat.Notes, note)
										}
									}
									gpifApplyTremoloPicking(b.Tremolo, beat.Notes)
									if len(beat.Notes) == 0 {
										beat.Status = BeatStatusRest
									}
								}
								if isGrace {
									pendingGrace = append(pendingGrace, gpifPendingGrace{beat: beat, onBeat: graceOnBeat})
									continue
								}
								voice.Beats = append(
									voice.Beats,
									gpifApplyPendingGrace(&beat, pendingGrace, staff.PercussionTrack)...,
								)
								pendingGrace = nil
								voice.Beats = append(voice.Beats, beat)
							}
							voice.Beats = append(
								voice.Beats,
								gpifApplyPendingGrace(nil, pendingGrace, staff.PercussionTrack)...,
							)
						}
						m.Voices = append(m.Voices, voice)
					}
				}

				staff.Measures = append(staff.Measures, m)
			}
			track.Measures = track.Staves[0].Measures
			track.Strings = track.Staves[0].Strings
		}
	}

	song.finalizeTiming()

	return song, nil
}

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

	for _, track := range doc.Tracks.Tracks {
		path := gpifObjectPath("Tracks/Track", track.ID)
		if track.Transpose != nil {
			context.add(diagnosticSource("GPIF.Track.Transpose", "staff-ownership", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				Kind: ParseDiagnosticUnsupportedFeature, SourcePath: path + "/Transpose",
				ObjectID: track.ID, Location: ParseLocation{TrackID: track.ID}, Feature: "staff-ownership",
				Reason: "track transposition has no destination in Song",
			})
		}
		for propertyIndex, property := range track.Properties {
			gpifAuditTrackProperty(context, track.ID, fmt.Sprintf("%s/Properties/Property[%d]", path, propertyIndex), property)
		}
		for staffIndex, staff := range track.Staves.Staff {
			for propertyIndex, property := range staff.Properties {
				propertyPath := fmt.Sprintf("%s/Staves/Staff[%d]/Properties/Property[%d]", path, staffIndex, propertyIndex)
				gpifAuditStaffProperty(context, track.ID, propertyPath, property)
			}
		}
	}

	for _, note := range doc.Notes.Notes {
		path := gpifObjectPath("Notes/Note", note.ID)
		for _, property := range note.Properties.Properties {
			gpifAuditNoteProperty(context, note.ID, path, property)
		}
		if note.Vibrato != "" && note.Vibrato != "None" {
			context.add(diagnosticSource("GPIF.Note.Vibrato", "note-and-beat-semantics", ParseDiagnosticLossyProjection), ParseDiagnostic{
				Kind: ParseDiagnosticLossyProjection, SourcePath: path + "/Vibrato", ObjectID: note.ID,
				Location: ParseLocation{NoteID: note.ID}, Feature: "note-and-beat-semantics",
				Reason: "Song represents typed note vibrato as a boolean",
			})
		}
	}

	for _, beat := range doc.Beats.Beats {
		path := gpifObjectPath("Beats/Beat", beat.ID)
		for _, property := range beat.Properties.Properties {
			gpifAuditBeatProperty(context, beat.ID, path, property)
		}
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.GraceNotes.InvalidValue", "grace-relationships", ParseDiagnosticUnsupportedFeature), beat.GraceNotes, []string{"", "OnBeat", "BeforeBeat"}, path+"/GraceNotes", beat.ID, "grace-relationships")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Arpeggio.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Arpeggio, []string{"", "Up", "Down"}, path+"/Arpeggio", beat.ID, "note-and-beat-semantics")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Hairpin.InvalidValue", "hairpins", ParseDiagnosticUnsupportedFeature), beat.Hairpin, []string{"", "Crescendo", "Decrescendo", "Diminuendo"}, path+"/Hairpin", beat.ID, "hairpins")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Ottavia.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Ottavia, []string{"", "8va", "8vb", "15ma", "15mb"}, path+"/Ottavia", beat.ID, "note-and-beat-semantics")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Tremolo.InvalidValue", "tremolo-picking", ParseDiagnosticUnsupportedFeature), beat.Tremolo, []string{"", "1/2", "1/4", "1/8"}, path+"/Tremolo", beat.ID, "tremolo-picking")
		gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Dynamic.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Dynamic, []string{"", "PPP", "PP", "P", "MP", "MF", "F", "FF", "FFF"}, path+"/Dynamic", beat.ID, "note-and-beat-semantics")
		if beat.Wah != "" {
			context.add(diagnosticSource("GPIF.Beat.Wah", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				Kind: ParseDiagnosticUnsupportedFeature, SourcePath: path + "/Wah", ObjectID: beat.ID,
				Location: ParseLocation{BeatID: beat.ID}, Feature: "note-and-beat-semantics",
				Reason: "beat wah has no destination in Song",
			})
		}
		switch beat.Fadding {
		case "", "FadeIn":
		case "FadeOut", "VolumeSwell":
			context.add(diagnosticSource("GPIF.Beat.Fadding.Lossy", "note-and-beat-semantics", ParseDiagnosticLossyProjection), ParseDiagnostic{
				Kind: ParseDiagnosticLossyProjection, SourcePath: path + "/Fadding", ObjectID: beat.ID,
				Location: ParseLocation{BeatID: beat.ID}, Feature: "note-and-beat-semantics",
				Reason: "Song combines fade-out and volume-swell values into FadeIn",
			})
		default:
			gpifAuditEnum(context, diagnosticSource("GPIF.Beat.Fadding.InvalidValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beat.Fadding, []string{"", "FadeIn", "FadeOut", "VolumeSwell"}, path+"/Fadding", beat.ID, "note-and-beat-semantics")
		}
	}

	for _, rhythm := range doc.Rhythms.Rhythms {
		gpifAuditEnum(context, diagnosticSource("GPIF.Rhythm.NoteValue.InvalidValue", "rhythm", ParseDiagnosticUnsupportedFeature), rhythm.NoteValue, []string{"Whole", "Half", "Quarter", "Eighth", "16th", "32nd", "64th", "128th"}, gpifObjectPath("Rhythms/Rhythm", rhythm.ID)+"/NoteValue", rhythm.ID, "rhythm")
	}
	for index, masterBar := range doc.MasterBars.MasterBars {
		gpifAuditEnum(context, diagnosticSource("GPIF.MasterBar.TripletFeel.InvalidValue", "rhythm", ParseDiagnosticUnsupportedFeature), masterBar.TripletFeel, []string{"", "NoTripletFeel", "Triplet8th", "Triplet16th"}, fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/TripletFeel", index), "", "rhythm")
	}

	gpifAuditReferences(doc, context)
}
func gpifAuditTrackProperty(context *parseContext, trackID, path string, property gpifStaffProperty) {
	gpifAuditOwnedStaffProperty(
		context, trackID, path, property, "track",
		diagnosticSource("GPIF.Track.Property.Tuning.MissingPitches", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Track.Property.Tuning.Label", "staff-ownership", ParseDiagnosticUnsupportedFeature),
		diagnosticSource("GPIF.Track.Property.CapoFret", "staff-ownership", ParseDiagnosticUnsupportedFeature),
		diagnosticSource("GPIF.Track.Property.Unknown", "staff-ownership", ParseDiagnosticUnknownSyntax),
	)
}

func gpifAuditStaffProperty(context *parseContext, trackID, path string, property gpifStaffProperty) {
	gpifAuditOwnedStaffProperty(
		context, trackID, path, property, "staff",
		diagnosticSource("GPIF.Staff.Property.Tuning.MissingPitches", "staff-ownership", ParseDiagnosticInvalidData),
		diagnosticSource("GPIF.Staff.Property.Tuning.Label", "staff-ownership", ParseDiagnosticUnsupportedFeature),
		diagnosticSource("GPIF.Staff.Property.CapoFret", "staff-ownership", ParseDiagnosticUnsupportedFeature),
		diagnosticSource("GPIF.Staff.Property.Unknown", "staff-ownership", ParseDiagnosticUnknownSyntax),
	)
}

func gpifAuditOwnedStaffProperty(
	context *parseContext,
	trackID, path string,
	property gpifStaffProperty,
	owner string,
	tuningMissingSource, tuningLabelSource, capoSource, unknownSource parseDiagnosticSource,
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
		if property.Label != "" {
			context.add(tuningLabelSource, ParseDiagnostic{
				SourcePath: propertyPath + "/Label", ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
				Reason: "Song has no tuning label destination",
			})
		}
	case "DiagramCollection", "ChordCollection":
		return
	case "CapoFret":
		context.add(capoSource, ParseDiagnostic{
			SourcePath: propertyPath, ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
			Reason: "Staff has no capo destination",
		})
	default:
		context.add(unknownSource, ParseDiagnostic{
			SourcePath: propertyPath, ObjectID: trackID, Location: ParseLocation{TrackID: trackID},
			Reason: fmt.Sprintf("unknown GPIF %s property %q", owner, property.Name),
		})
	}
}

var gpifNotePropertySources = map[string]parseDiagnosticSource{
	"BendOriginOffset":      diagnosticSource("GPIF.Note.Property.BendOriginOffset.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendOriginValue":       diagnosticSource("GPIF.Note.Property.BendOriginValue.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendMiddleOffset1":     diagnosticSource("GPIF.Note.Property.BendMiddleOffset1.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendMiddleOffset2":     diagnosticSource("GPIF.Note.Property.BendMiddleOffset2.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendMiddleValue":       diagnosticSource("GPIF.Note.Property.BendMiddleValue.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendDestinationOffset": diagnosticSource("GPIF.Note.Property.BendDestinationOffset.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"BendDestinationValue":  diagnosticSource("GPIF.Note.Property.BendDestinationValue.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Tapped":                diagnosticSource("GPIF.Note.Property.Tapped", "note-and-beat-semantics", ParseDiagnosticLossyProjection),
	"HopoOrigin":            diagnosticSource("GPIF.Note.Property.HopoOrigin", "note-and-beat-semantics", ParseDiagnosticLossyProjection),
	"HopoDestination":       diagnosticSource("GPIF.Note.Property.HopoDestination", "note-and-beat-semantics", ParseDiagnosticLossyProjection),
	"LeftHandTapped":        diagnosticSource("GPIF.Note.Property.LeftHandTapped", "note-and-beat-semantics", ParseDiagnosticLossyProjection),
	"Element":               diagnosticSource("GPIF.Note.Property.Element", "percussion-articulations", ParseDiagnosticUnsupportedFeature),
	"Variation":             diagnosticSource("GPIF.Note.Property.Variation", "percussion-articulations", ParseDiagnosticUnsupportedFeature),
	"ConcertPitch":          diagnosticSource("GPIF.Note.Property.ConcertPitch", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"TransposedPitch":       diagnosticSource("GPIF.Note.Property.TransposedPitch", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Tone":                  diagnosticSource("GPIF.Note.Property.Tone", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Octave":                diagnosticSource("GPIF.Note.Property.Octave", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
}

var gpifBeatPropertySources = map[string]parseDiagnosticSource{
	"PrimaryPickupVolume":         diagnosticSource("GPIF.Beat.Property.PrimaryPickupVolume", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
	"PrimaryPickupTone":           diagnosticSource("GPIF.Beat.Property.PrimaryPickupTone", "note-and-beat-semantics", ParseDiagnosticDeliberateIgnore),
	"WhammyBar":                   diagnosticSource("GPIF.Beat.Property.WhammyBar", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarExtend":             diagnosticSource("GPIF.Beat.Property.WhammyBarExtend", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarOriginValue":        diagnosticSource("GPIF.Beat.Property.WhammyBarOriginValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarOriginOffset":       diagnosticSource("GPIF.Beat.Property.WhammyBarOriginOffset", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarMiddleValue":        diagnosticSource("GPIF.Beat.Property.WhammyBarMiddleValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarMiddleOffset1":      diagnosticSource("GPIF.Beat.Property.WhammyBarMiddleOffset1", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarMiddleOffset2":      diagnosticSource("GPIF.Beat.Property.WhammyBarMiddleOffset2", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarDestinationValue":   diagnosticSource("GPIF.Beat.Property.WhammyBarDestinationValue", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"WhammyBarDestinationOffset":  diagnosticSource("GPIF.Beat.Property.WhammyBarDestinationOffset", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"BarreFret":                   diagnosticSource("GPIF.Beat.Property.BarreFret", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"BarreString":                 diagnosticSource("GPIF.Beat.Property.BarreString", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Brush.MissingDirection":      diagnosticSource("GPIF.Beat.Property.Brush.MissingDirection", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"PickStroke.MissingDirection": diagnosticSource("GPIF.Beat.Property.PickStroke.MissingDirection", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Brush.InvalidDirection":      diagnosticSource("GPIF.Beat.Property.Brush.InvalidDirection", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"PickStroke.InvalidDirection": diagnosticSource("GPIF.Beat.Property.PickStroke.InvalidDirection", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"Slapped.MissingEnable":       diagnosticSource("GPIF.Beat.Property.Slapped.MissingEnable", "note-and-beat-semantics", ParseDiagnosticInvalidData),
	"Popped.MissingEnable":        diagnosticSource("GPIF.Beat.Property.Popped.MissingEnable", "note-and-beat-semantics", ParseDiagnosticInvalidData),
}

func gpifAuditNoteProperty(context *parseContext, noteID, path string, property gpifProperty) {
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
	case "Slide":
		gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.Slide.MissingPayload", "note-and-beat-semantics", ParseDiagnosticInvalidData), property.Flags != nil, propertyPath, noteID, "note-and-beat-semantics", "Flags")
	case "Muted", "Bended", "PalmMuted", "Harmonic", "ShowStringNumber":
		return
	case "Tapped", "HopoOrigin", "HopoDestination", "LeftHandTapped":
		context.add(gpifNotePropertySources[property.Name], ParseDiagnostic{
			Kind: ParseDiagnosticLossyProjection, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "note-and-beat-semantics",
			Reason: "Song combines authored hammer and tapping relationships into Hammer",
		})
	case "HarmonicType":
		if property.HType == nil {
			gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.HarmonicType.MissingPayload", "harmonics", ParseDiagnosticInvalidData), false, propertyPath, noteID, "harmonics", "HType")
			return
		}
		switch *property.HType {
		case "NoHarmonic", "Natural", "Artificial", "Pinch", "Tap", "Semi":
			return
		case "Feedback":
			context.add(diagnosticSource("GPIF.Note.Property.HarmonicType.Feedback", "harmonics", ParseDiagnosticLossyProjection), ParseDiagnostic{
				Kind: ParseDiagnosticLossyProjection, SourcePath: propertyPath, ObjectID: noteID,
				Location: ParseLocation{NoteID: noteID}, Feature: "harmonics",
				Reason: "Song combines feedback and semi harmonics",
			})
		default:
			context.add(diagnosticSource("GPIF.Note.Property.HarmonicType.Unsupported", "harmonics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				Kind: ParseDiagnosticUnsupportedFeature, SourcePath: propertyPath, ObjectID: noteID,
				Location: ParseLocation{NoteID: noteID}, Feature: "harmonics",
				Reason: fmt.Sprintf("unsupported harmonic type %q", *property.HType),
			})
		}
	case "HarmonicFret":
		gpifAuditPropertyPayload(context, diagnosticSource("GPIF.Note.Property.HarmonicFret.MissingPayload", "harmonics", ParseDiagnosticInvalidData), property.HFret != nil || property.Float != nil, propertyPath, noteID, "harmonics", "HFret or Float")
	case "Element", "Variation":
		context.add(gpifNotePropertySources[property.Name], ParseDiagnostic{
			Kind: ParseDiagnosticUnsupportedFeature, SourcePath: propertyPath, ObjectID: noteID,
			Location: ParseLocation{NoteID: noteID}, Feature: "percussion-articulations",
			Reason: fmt.Sprintf("recognized GPIF percussion property %q has no destination in Song", property.Name),
		})
	case "ConcertPitch", "TransposedPitch", "Tone", "Octave":
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

func gpifAuditPropertyPayload(context *parseContext, source parseDiagnosticSource, present bool, path, objectID, feature, payload string) {
	if present {
		return
	}
	context.add(source, ParseDiagnostic{
		Kind: ParseDiagnosticInvalidData, SourcePath: path, ObjectID: objectID,
		Feature: feature, Reason: fmt.Sprintf("GPIF property has no %s payload", payload),
	})
}

func gpifAuditBeatProperty(context *parseContext, beatID, path string, property gpifProperty) {
	propertyPath := fmt.Sprintf("%s/Properties/Property[@name=%q]", path, property.Name)
	switch property.Name {
	case "Brush", "PickStroke":
		if property.Direction == nil {
			gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingDirection"], false, propertyPath, beatID, "note-and-beat-semantics", "Direction")
			return
		}
		gpifAuditEnum(context, gpifBeatPropertySources[property.Name+".InvalidDirection"], *property.Direction, []string{"Up", "Down"}, propertyPath+"/Direction", beatID, "note-and-beat-semantics")
	case "Slapped", "Popped":
		gpifAuditPropertyPayload(context, gpifBeatPropertySources[property.Name+".MissingEnable"], property.Enable != nil, propertyPath, beatID, "note-and-beat-semantics", "Enable")
	case "VibratoWTremBar":
		context.add(diagnosticSource("GPIF.Beat.Property.VibratoWTremBar", "note-and-beat-semantics", ParseDiagnosticLossyProjection), ParseDiagnostic{
			Kind: ParseDiagnosticLossyProjection, SourcePath: propertyPath, ObjectID: beatID,
			Location: ParseLocation{BeatID: beatID}, Feature: "note-and-beat-semantics",
			Reason: "Song represents typed beat vibrato as a boolean",
		})
	case "PrimaryPickupVolume", "PrimaryPickupTone":
		context.add(gpifBeatPropertySources[property.Name], ParseDiagnostic{
			SourcePath: propertyPath, ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
			Reason: "primary pickup playback metadata is intentionally outside the notation-focused Song model",
		})
		return
	case "WhammyBar", "WhammyBarExtend", "WhammyBarOriginValue", "WhammyBarOriginOffset", "WhammyBarMiddleValue", "WhammyBarMiddleOffset1", "WhammyBarMiddleOffset2", "WhammyBarDestinationValue", "WhammyBarDestinationOffset":
		gpifAuditUnsupportedBeatProperty(context, gpifBeatPropertySources[property.Name], beatID, propertyPath, property.Name)
	case "BarreFret", "BarreString":
		gpifAuditUnsupportedBeatProperty(context, gpifBeatPropertySources[property.Name], beatID, propertyPath, property.Name)
	case "Rasgueado":
		gpifAuditUnsupportedBeatProperty(context, diagnosticSource("GPIF.Beat.Property.Rasgueado", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), beatID, propertyPath, property.Name)
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
	for _, track := range doc.Tracks.Tracks {
		trackPath := gpifObjectPath("Tracks/Track", track.ID)
		gpifAddID(context, diagnosticSource("GPIF.Track.EmptyID", "staff-ownership", ParseDiagnosticInvalidData), diagnosticSource("GPIF.Track.DuplicateID", "staff-ownership", ParseDiagnosticInvalidData), tracks, track.ID, trackPath, "staff-ownership")
		gpifAuditChordIDs(context, chords, track.Properties, trackPath+"/Properties")
		for staffIndex, staff := range track.Staves.Staff {
			path := fmt.Sprintf("%s/Staves/Staff[%d]/Properties", trackPath, staffIndex)
			gpifAuditChordIDs(context, chords, staff.Properties, path)
		}
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
	for index, masterBar := range doc.MasterBars.MasterBars {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.MasterBar.Bars.Reference", "staff-ownership", ParseDiagnosticInvalidData), splitIDs(masterBar.Bars), bars, fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/Bars", index), "", ParseLocation{}, "staff-ownership")
	}
	for _, bar := range doc.Bars.Bars {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Bar.Voices.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), splitIDs(bar.Voices), voices, gpifObjectPath("Bars/Bar", bar.ID)+"/Voices", bar.ID, ParseLocation{BarID: bar.ID}, "note-and-beat-semantics")
	}
	for _, voice := range doc.Voices.Voices {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Voice.Beats.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), splitIDs(voice.Beats), beats, gpifObjectPath("Voices/Voice", voice.ID)+"/Beats", voice.ID, ParseLocation{VoiceID: voice.ID}, "note-and-beat-semantics")
	}
	for _, beat := range doc.Beats.Beats {
		location := ParseLocation{BeatID: beat.ID}
		path := gpifObjectPath("Beats/Beat", beat.ID)
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Beat.Notes.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), splitIDs(beat.Notes), notes, path+"/Notes", beat.ID, location, "note-and-beat-semantics")
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Beat.Rhythm.Reference", "rhythm", ParseDiagnosticInvalidData), []string{beat.Rhythm.Ref}, rhythms, path+"/Rhythm", beat.ID, location, "rhythm")
		gpifAuditReferenceList(context, diagnosticSource("GPIF.Beat.Chord.Reference", "note-and-beat-semantics", ParseDiagnosticInvalidData), []string{beat.Chord}, chords, path+"/Chord", beat.ID, location, "note-and-beat-semantics")
	}
	if doc.BackingTrack != nil {
		gpifAuditReferenceList(context, diagnosticSource("GPIF.BackingTrack.AssetId.Reference", "score-core", ParseDiagnosticInvalidData), []string{doc.BackingTrack.AssetID}, assets, "/GPIF/BackingTrack/AssetId", "", ParseLocation{}, "score-core")
	}
}

var gpifDiagramPropertySources = map[string]parseDiagnosticSource{
	"ShowName":      diagnosticSource("GPIF.Chord.Diagram.Property.ShowName", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"ShowDiagram":   diagnosticSource("GPIF.Chord.Diagram.Property.ShowDiagram", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
	"ShowFingering": diagnosticSource("GPIF.Chord.Diagram.Property.ShowFingering", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature),
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
			if item.Diagram.Fingering != nil {
				context.add(diagnosticSource("GPIF.Chord.Diagram.Fingering", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
					SourcePath: itemPath + "/Diagram/Fingering", ObjectID: item.ID,
					Reason: "Chord does not preserve GPIF diagram finger positions",
				})
			}
			for diagramPropertyIndex, diagramProperty := range item.Diagram.Properties {
				propertyPath := fmt.Sprintf("%s/Diagram/Property[%d][@name=%q]", itemPath, diagramPropertyIndex, diagramProperty.Name)
				source, known := gpifDiagramPropertySources[diagramProperty.Name]
				if !known {
					source = diagnosticSource("GPIF.Chord.Diagram.Property.Unknown", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax)
				}
				context.add(source, ParseDiagnostic{
					SourcePath: propertyPath, ObjectID: item.ID,
					Reason: fmt.Sprintf("Chord does not preserve GPIF diagram property %q", diagramProperty.Name),
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

type gpifPendingGrace struct {
	beat   Beat
	onBeat bool
}

func gpifApplyPendingGrace(target *Beat, pending []gpifPendingGrace, percussion bool) []Beat {
	var orphans []Beat
	for pendingIndex, pendingBeat := range pending {
		orphan := pendingBeat.beat
		orphan.isGrace = true
		orphan.Notes = nil
		for noteIndex := range pendingBeat.beat.Notes {
			graceNote := pendingBeat.beat.Notes[noteIndex]
			effect := gpifGraceEffect(&graceNote, &pendingBeat.beat.Duration, pendingBeat.onBeat, pendingIndex)
			targetIndex := gpifGraceTarget(target, &graceNote, percussion)
			if targetIndex >= 0 {
				target.Notes[targetIndex].Effect.Graces = append(target.Notes[targetIndex].Effect.Graces, effect)
				continue
			}
			graceNote.Effect.Graces = append(graceNote.Effect.Graces, effect)
			orphan.Notes = append(orphan.Notes, graceNote)
		}
		if len(orphan.Notes) > 0 || len(pendingBeat.beat.Notes) == 0 {
			orphans = append(orphans, orphan)
		}
	}
	return orphans
}

func gpifGraceTarget(target *Beat, grace *Note, percussion bool) int {
	if target == nil {
		return -1
	}
	if percussion {
		for index := range target.Notes {
			if target.Notes[index].Value == grace.Value {
				return index
			}
		}
		return -1
	}
	for index := range target.Notes {
		if target.Notes[index].String == grace.String {
			return index
		}
	}
	return -1
}

func gpifGraceEffect(note *Note, duration *Duration, onBeat bool, sequence int) GraceEffect {
	transition := GraceEffectTransitionNone
	switch {
	case note.Effect.Hammer:
		transition = GraceEffectTransitionHammer
	case len(note.Effect.Slides) > 0:
		transition = GraceEffectTransitionSlide
	}
	return GraceEffect{
		Duration:                  uint8(min(uint16(math.MaxUint8), duration.Value)),
		Fret:                      int8(min(int16(math.MaxInt8), max(int16(math.MinInt8), note.Value))),
		PercussionArticulation:    note.PercussionArticulation,
		HasPercussionArticulation: note.HasPercussionArticulation,
		IsDead:                    note.Kind == NoteTypeDead,
		IsOnBeat:                  onBeat,
		Sequence:                  uint8(min(sequence, math.MaxUint8)),
		Transition:                transition,
		Velocity:                  note.Velocity,
	}
}

func gpifVersion(value string) Version {
	version := Version{Data: value}
	parts := strings.Split(value, ".")
	for index := 0; index < len(parts) && index < len(version.Number); index++ {
		number, err := strconv.ParseUint(parts[index], 10, 8)
		if err == nil {
			version.Number[index] = byte(number)
		}
	}
	return version
}

func gpifReadStaffStrings(staff gpifStaff) []GuitarString {
	for _, property := range staff.Properties {
		if property.Name != "Tuning" || property.Pitches == "" {
			continue
		}
		values := splitIDs(property.Pitches)
		strings := make([]GuitarString, 0, len(values))
		for sourceIndex := len(values) - 1; sourceIndex >= 0; sourceIndex-- {
			value, err := strconv.ParseInt(values[sourceIndex], 10, 8)
			if err != nil {
				continue
			}
			strings = append(strings, GuitarString{
				Number: int8(len(strings) + 1),
				Value:  int8(value),
			})
		}
		return strings
	}
	return nil
}

func gpifReadChordMap(track gpifTrack) map[string]Chord {
	chords := make(map[string]Chord)
	gpifReadChordProperties(track.Properties, chords)
	for _, staff := range track.Staves.Staff {
		gpifReadChordProperties(staff.Properties, chords)
	}
	return chords
}

func gpifReadChordProperties(properties []gpifStaffProperty, chords map[string]Chord) {
	for _, property := range properties {
		if (property.Name != "DiagramCollection" && property.Name != "ChordCollection") || property.Items == nil {
			continue
		}
		for _, item := range property.Items.Items {
			if item.ID == "" {
				continue
			}
			chord := Chord{Name: item.Name}
			if item.Diagram != nil {
				firstFret := uint8(min(math.MaxUint8, max(0, item.Diagram.BaseFret+1)))
				chord.FirstFret = &firstFret
				chord.Length = uint8(item.Diagram.StringCount)
				chord.Strings = make([]int8, item.Diagram.StringCount)
				for index := range chord.Strings {
					chord.Strings[index] = -1
				}
				for _, fret := range item.Diagram.Frets {
					index := item.Diagram.StringCount - fret.String - 1
					if index >= 0 && index < len(chord.Strings) {
						chord.Strings[index] = int8(min(math.MaxInt8, max(0, item.Diagram.BaseFret+fret.Fret)))
					}
				}
			}
			chords[item.ID] = chord
		}
	}
}

func gpifMIDIChannel(port, channel int) uint8 {
	value := port*16 + channel
	if value < 0 {
		return 0
	}
	if value > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(value)
}

func gpifApplyChannelStrip(parameters string, channel *MidiChannel) {
	values := strings.Fields(parameters)
	if len(values) <= 12 {
		return
	}
	balance, balanceErr := strconv.ParseFloat(values[11], 64)
	if balanceErr == nil {
		channel.Balance = int8(math.Round(min(1, max(0, balance)) * 127))
	}
	volume, volumeErr := strconv.ParseFloat(values[12], 64)
	if volumeErr == nil {
		channel.Volume = int8(math.Round(min(1, max(0, volume)) * 127))
	}
}

func gpifReadVolumeAutomations(
	automations []gpifAutomation,
	trackIndex int,
	song *Song,
) {
	for _, automation := range automations {
		if automation.Type != "DSPParam_12" {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(automation.Value.Text), 64)
		if err != nil {
			continue
		}
		song.VolumeAutomations = append(song.VolumeAutomations, VolumeAutomation{
			Track:    trackIndex,
			Bar:      automation.Bar,
			Position: min(1, max(0, automation.Position)),
			Value:    min(1, max(0, value)),
			Linear:   automation.Linear,
		})
	}
}

func gpifReadBackingTrack(doc gpifDocument, song *Song) {
	if doc.BackingTrack == nil {
		return
	}
	framePadding, _ := strconv.ParseInt(strings.TrimSpace(doc.BackingTrack.FramePadding), 10, 64)
	backingTrack := &BackingTrack{
		Name:         doc.BackingTrack.Name,
		Source:       doc.BackingTrack.Source,
		AssetID:      doc.BackingTrack.AssetID,
		FramePadding: framePadding,
		Enabled:      doc.BackingTrack.Enabled,
	}
	for _, asset := range doc.Assets.Assets {
		if asset.ID != backingTrack.AssetID {
			continue
		}
		backingTrack.OriginalFilePath = asset.OriginalFilePath
		backingTrack.OriginalFileSHA1 = asset.OriginalFileSHA1
		backingTrack.EmbeddedFilePath = asset.EmbeddedFilePath
		break
	}
	song.BackingTrack = backingTrack
}

func gpifReadSyncPoints(automations []gpifAutomation, song *Song) {
	framePadding := int64(0)
	if song.BackingTrack != nil {
		framePadding = song.BackingTrack.FramePadding
	}
	for _, automation := range automations {
		if automation.Type != "SyncPoint" {
			continue
		}
		frameOffset, err := strconv.ParseInt(strings.TrimSpace(automation.Value.FrameOffset), 10, 64)
		if err != nil {
			continue
		}
		bar := automation.Bar
		if value := strings.TrimSpace(automation.Value.BarIndex); value != "" {
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				bar = parsed
			}
		}
		barOccurrence, _ := strconv.Atoi(strings.TrimSpace(automation.Value.BarOccurrence))
		modifiedTempo, _ := strconv.ParseFloat(strings.TrimSpace(automation.Value.ModifiedTempo), 64)
		originalTempo, _ := strconv.ParseFloat(strings.TrimSpace(automation.Value.OriginalTempo), 64)
		song.SyncPoints = append(song.SyncPoints, SyncPoint{
			Bar:           bar,
			Position:      automation.Position,
			BarOccurrence: barOccurrence,
			FrameOffset:   frameOffset,
			MediaTimeMS:   float64(frameOffset-framePadding) / GPIFBackingTrackSampleRate * 1000,
			ModifiedTempo: modifiedTempo,
			OriginalTempo: originalTempo,
			Linear:        automation.Linear,
			Visible:       automation.Visible == "" || strings.EqualFold(automation.Visible, "true"),
		})
	}
}

func gpifReadTempoAutomations(automations []gpifAutomation, song *Song) {
	earliest := -1
	for _, auto := range automations {
		if auto.Type != "Tempo" {
			continue
		}
		parts := strings.Fields(auto.Value.Text)
		if len(parts) == 0 {
			continue
		}
		tempo, err := strconv.ParseFloat(parts[0], 64)
		if err != nil || tempo <= 0 {
			continue
		}
		tempo *= gpifTempoReferenceFactor(parts)
		change := TempoAutomation{Bar: auto.Bar, Position: auto.Position, Tempo: tempo}
		song.TempoAutomations = append(song.TempoAutomations, change)
		if earliest < 0 || gpifAutomationIsBefore(change, song.TempoAutomations[earliest]) {
			earliest = len(song.TempoAutomations) - 1
			song.Tempo = int16(math.Round(tempo))
			if auto.Text != "" {
				song.TempoName = auto.Text
			}
		}
	}
}

func gpifTempoReferenceFactor(parts []string) float64 {
	reference := 1
	if len(parts) > 1 {
		value := parts[1]
		end := 0
		if value != "" && (value[0] == '+' || value[0] == '-') {
			end++
		}
		digitStart := end
		for end < len(value) && value[end] >= '0' && value[end] <= '9' {
			end++
		}
		parsed, err := strconv.Atoi(value[:end])
		if end == digitStart {
			err = strconv.ErrSyntax
		}
		if err != nil || parsed < 1 || parsed > 5 {
			reference = 2
		} else {
			reference = parsed
		}
	}
	return [...]float64{0, 0.5, 1, 1.5, 2, 3}[reference]
}

func gpifAutomationIsBefore(a, b TempoAutomation) bool {
	if a.Bar != b.Bar {
		return a.Bar < b.Bar
	}
	return a.Position < b.Position
}

func gpifApplyBeatEffects(b *gpifBeat, beat *Beat) {
	if b.Whammy != nil {
		values := []struct {
			position string
			value    string
		}{
			{b.Whammy.OriginOffset, b.Whammy.OriginValue},
			{b.Whammy.MiddleOffset1, b.Whammy.MiddleValue},
			{b.Whammy.MiddleOffset2, b.Whammy.MiddleValue},
			{b.Whammy.DestinationOffset, b.Whammy.DestinationValue},
		}
		bend := &BendEffect{}
		for _, point := range values {
			position, positionErr := strconv.ParseFloat(point.position, 64)
			value, valueErr := strconv.ParseFloat(point.value, 64)
			if positionErr != nil || valueErr != nil {
				continue
			}
			bend.Points = append(bend.Points, BendPoint{
				Position: uint8(math.Round(position * float64(BendEffectMaxPosition) / 100)),
				Value:    int8(math.Round(value / float64(GPBendSemitone))),
			})
		}
		if len(bend.Points) > 0 {
			beat.Effect.TremoloBar = bend
		}
	}

	// Arpeggio / brush stroke
	switch b.Arpeggio {
	case "Up":
		beat.Effect.Stroke.Direction = BeatStrokeDirectionUp
		beat.Effect.Stroke.Value = uint16(DurationEighth)
	case "Down":
		beat.Effect.Stroke.Direction = BeatStrokeDirectionDown
		beat.Effect.Stroke.Value = uint16(DurationEighth)
	}

	// Ottavia
	switch b.Ottavia {
	case "8va":
		beat.Octave = OctaveOttava
	case "8vb":
		beat.Octave = OctaveOttavaBassa
	case "15ma":
		beat.Octave = OctaveQuindicesima
	case "15mb":
		beat.Octave = OctaveQuindicesimaBassa
	}

	// Free text
	if b.FreeText != "" {
		beat.Text = b.FreeText
	}

	// Beat properties
	for _, p := range b.Properties.Properties {
		switch p.Name {
		case "Brush":
			if p.Direction != nil {
				switch *p.Direction {
				case "Up":
					beat.Effect.Stroke.Direction = BeatStrokeDirectionUp
					beat.Effect.Stroke.Value = uint16(DurationEighth)
				case "Down":
					beat.Effect.Stroke.Direction = BeatStrokeDirectionDown
					beat.Effect.Stroke.Value = uint16(DurationEighth)
				}
			}
		case "PickStroke":
			if p.Direction != nil {
				switch *p.Direction {
				case "Up":
					beat.Effect.PickStroke = BeatStrokeDirectionUp
				case "Down":
					beat.Effect.PickStroke = BeatStrokeDirectionDown
				}
			}
		case "Slapped":
			beat.Effect.SlapEffect = SlapEffectSlapping
		case "Popped":
			beat.Effect.SlapEffect = SlapEffectPopping
		case "VibratoWTremBar":
			if p.Strength != nil {
				beat.Effect.Vibrato = true
			}
		}
	}
}

func gpifApplyTremoloPicking(value string, notes []Note) {
	if value == "" {
		return
	}
	effect := TremoloPickingEffect{Duration: defaultDuration()}
	switch value {
	case "1/2":
		effect.Duration.Value = uint16(DurationEighth)
	case "1/4":
		effect.Duration.Value = uint16(DurationSixteenth)
	case "1/8":
		effect.Duration.Value = uint16(DurationThirtySecond)
	}
	for noteIndex := range notes {
		noteEffect := effect
		notes[noteIndex].Effect.TremoloPicking = &noteEffect
	}
}

func gpifHairpin(value string) Hairpin {
	switch value {
	case "Crescendo":
		return HairpinCrescendo
	case "Decrescendo", "Diminuendo":
		return HairpinDiminuendo
	default:
		return HairpinNone
	}
}

// gpifDynamicToVelocity converts a GPIF dynamic string to a velocity value.
func gpifDynamicToVelocity(dynamic string) int16 {
	switch dynamic {
	case "PPP":
		return MinVelocity
	case "PP":
		return MinVelocity + VelocityIncrement
	case "P":
		return MinVelocity + VelocityIncrement*2
	case "MP":
		return MinVelocity + VelocityIncrement*3
	case "MF":
		return MinVelocity + VelocityIncrement*4
	case "F":
		return MinVelocity + VelocityIncrement*5
	case "FF":
		return MinVelocity + VelocityIncrement*6
	case "FFF":
		return MinVelocity + VelocityIncrement*7
	default:
		return DefaultVelocity
	}
}

func splitIDs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

func gpifRhythmToDuration(r *gpifRhythm) Duration {
	d := defaultDuration()
	switch r.NoteValue {
	case "Whole":
		d.Value = 1
	case "Half":
		d.Value = 2
	case "Quarter":
		d.Value = 4
	case "Eighth":
		d.Value = 8
	case "16th":
		d.Value = 16
	case "32nd":
		d.Value = 32
	case "64th":
		d.Value = 64
	case "128th":
		d.Value = 128
	}
	if r.AugmentationDot != nil {
		if r.AugmentationDot.Count >= 1 {
			d.Dotted = true
		}
		if r.AugmentationDot.Count >= 2 {
			d.DoubleDotted = true
		}
	}
	if r.PrimaryTuplet != nil && r.PrimaryTuplet.Num > 0 && r.PrimaryTuplet.Den > 0 {
		d.TupletEnters = uint8(r.PrimaryTuplet.Num)
		d.TupletTimes = uint8(r.PrimaryTuplet.Den)
	}
	return d
}

type gpifPercussionFallbacks struct {
	tableLength int
	ids         map[int]int
}

func gpifNormalizePercussionArticulation(track *Track, note *Note, fallbacks *gpifPercussionFallbacks) {
	if !note.HasPercussionArticulation || note.PercussionArticulation < fallbacks.tableLength {
		return
	}
	sourceID := note.PercussionArticulation
	if sourceID < 29 || sourceID > 127 {
		return
	}
	if fallbacks.ids != nil {
		if index, ok := fallbacks.ids[sourceID]; ok {
			note.PercussionArticulation = index
			return
		}
	} else {
		fallbacks.ids = make(map[int]int)
	}
	element := gp8DrumElement(int16(sourceID), GP8ExportOptions{})
	if element.Type == "percussion" {
		return
	}
	definitions := gpifReadPercussionArticulations(&gpifInstrumentSet{
		Elements: gpifElements{Elements: []gpifElement{element}},
	}, nil)
	if len(definitions) == 0 {
		return
	}
	definition := definitions[0]
	for index, existing := range track.PercussionArticulations {
		if gpifSamePercussionArticulation(existing, definition) {
			fallbacks.ids[sourceID] = index
			note.PercussionArticulation = index
			return
		}
	}
	index := len(track.PercussionArticulations)
	track.PercussionArticulations = append(track.PercussionArticulations, definition)
	fallbacks.ids[sourceID] = index
	note.PercussionArticulation = index
}

func gpifSamePercussionArticulation(a, b PercussionArticulation) bool {
	if a.ElementName != b.ElementName || a.ElementType != b.ElementType || a.ElementSoundbankName != b.ElementSoundbankName ||
		a.Name != b.Name || a.StaffLine != b.StaffLine || a.NoteheadDefault != b.NoteheadDefault ||
		a.NoteheadHalf != b.NoteheadHalf || a.NoteheadWhole != b.NoteheadWhole ||
		a.TechniquePlacement != b.TechniquePlacement || a.TechniqueSymbol != b.TechniqueSymbol ||
		a.OutputRSESound != b.OutputRSESound || a.OutputMIDINumber != b.OutputMIDINumber ||
		len(a.InputMIDINumbers) != len(b.InputMIDINumbers) {
		return false
	}
	for index := range a.InputMIDINumbers {
		if a.InputMIDINumbers[index] != b.InputMIDINumbers[index] {
			return false
		}
	}
	return true
}

func gpifNoteToNote(n *gpifNote, stringCount int, percussion bool) Note {
	note := defaultNote()
	note.Kind = NoteTypeNormal
	if percussion && n.InstrumentArticulation != nil && *n.InstrumentArticulation >= 0 {
		note.PercussionArticulation = *n.InstrumentArticulation
		note.HasPercussionArticulation = true
	}
	hasFret := false
	bend := gpifBendProperties{}
	var harmonicFret *float64

	// Parse properties
	for _, p := range n.Properties.Properties {
		switch p.Name {
		case "Fret":
			if p.Fret != nil {
				note.Value = int16(*p.Fret)
				hasFret = true
			}
		case "String":
			if p.String != nil {
				sourceString := int(*p.String)
				if !percussion && sourceString >= 0 && sourceString < stringCount {
					// GPIF counts from the lowest string while the parser model,
					// like GP3-5, counts from the highest string.
					note.String = int8(stringCount - sourceString)
				} else {
					note.String = int8(sourceString + 1)
				}
			}
		case "Midi":
			if p.Number != nil && !hasFret {
				note.Value = int16(*p.Number)
			}
		case "Muted":
			note.Kind = NoteTypeDead
			note.Effect.DeadNote = true
		case "Bended":
			bend.enabled = true
		case "BendOriginOffset":
			bend.originPosition = gpifBendPosition(p.Float)
		case "BendOriginValue":
			bend.originValue = gpifBendValue(p.Float)
		case "BendMiddleOffset1":
			bend.middlePosition1 = gpifBendPosition(p.Float)
		case "BendMiddleOffset2":
			bend.middlePosition2 = gpifBendPosition(p.Float)
		case "BendMiddleValue":
			bend.middleValue = gpifBendValue(p.Float)
		case "BendDestinationOffset":
			bend.destinationPosition = gpifBendPosition(p.Float)
		case "BendDestinationValue":
			bend.destinationValue = gpifBendValue(p.Float)
		case "PalmMuted":
			note.Effect.PalmMute = true
		case "Tapped":
			note.Effect.Hammer = true
		case "HopoOrigin":
			note.Effect.Hammer = true
		case "LeftHandTapped":
			note.Effect.Hammer = true
		case "HarmonicType":
			if p.HType != nil {
				switch *p.HType {
				case "Natural":
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeNatural}
				case "Artificial":
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial}
				case "Pinch":
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypePinch}
				case "Tap":
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeTapped}
				case "Semi":
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeSemi}
				case "Feedback":
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeSemi}
				}
			}
		case "HarmonicFret":
			value := p.HFret
			if value == nil {
				value = p.Float
			}
			if value != nil {
				if parsed, err := strconv.ParseFloat(*value, 64); err == nil {
					harmonicFret = &parsed
				}
			}
		case "Slide":
			if p.Flags != nil {
				if flags, err := strconv.Atoi(*p.Flags); err == nil {
					if flags&0x01 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlideShiftSlideTo)
					}
					if flags&0x02 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlideLegatoSlideTo)
					}
					if flags&0x04 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlideOutDownwards)
					}
					if flags&0x08 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlideOutUpwards)
					}
					if flags&0x10 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlideIntoFromBelow)
					}
					if flags&0x20 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlideIntoFromAbove)
					}
				}
			}
		}
	}
	if harmonicFret != nil && note.Effect.Harmonic != nil {
		fret := int8(*harmonicFret)
		note.Effect.Harmonic.Fret = &fret
		note.Effect.Harmonic.FretFloat = harmonicFret
	}
	if bend.enabled {
		note.Effect.Bend = bend.effect()
	}

	// Tie
	if n.Tie != nil {
		note.TieOrigin = n.Tie.Origin == "true"
		if n.Tie.Destination == "true" {
			note.Kind = NoteTypeTie
		}
	}

	// Let ring
	if n.LetRing != nil {
		note.Effect.LetRing = true
	}

	// Ghost note (AntiAccent)
	if n.AntiAccent == "Normal" {
		note.Effect.GhostNote = true
	}

	// Accent (bit flags: 0x01=Staccato, 0x04=Heavy, 0x08=Normal, 0x10=Tenuto)
	if n.Accent&0x01 != 0 {
		note.Effect.Staccato = true
	}
	if n.Accent&0x04 != 0 {
		note.Effect.HeavyAccentuatedNote = true
	}
	if n.Accent&0x08 != 0 {
		note.Effect.AccentuatedNote = true
	}

	// Vibrato
	if n.Vibrato != "" && n.Vibrato != "None" {
		note.Effect.Vibrato = true
	}

	// Trill
	if n.Trill != nil {
		note.Effect.Trill = &TrillEffect{
			Fret:     int8(n.Trill.Fret),
			Duration: defaultDuration(),
		}
		note.Effect.Trill.Duration.Value = uint16(DurationSixteenth)
	}

	// Dynamic → velocity
	note.Velocity = DefaultVelocity

	return note
}

type gpifBendProperties struct {
	enabled             bool
	originPosition      uint8
	originValue         int8
	middlePosition1     uint8
	middlePosition2     uint8
	middleValue         int8
	destinationPosition uint8
	destinationValue    int8
}

func (bend gpifBendProperties) effect() *BendEffect {
	points := []BendPoint{
		{Position: bend.originPosition, Value: bend.originValue},
		{Position: bend.middlePosition1, Value: bend.middleValue},
		{Position: bend.middlePosition2, Value: bend.middleValue},
		{Position: bend.destinationPosition, Value: bend.destinationValue},
	}
	sort.SliceStable(points, func(left, right int) bool {
		return points[left].Position < points[right].Position
	})
	if bend.destinationPosition < uint8(BendEffectMaxPosition) {
		points = append(points, BendPoint{Position: uint8(BendEffectMaxPosition), Value: bend.destinationValue})
	}
	points = simplifyBendPoints(points)
	maximum := int8(0)
	for _, point := range points {
		if point.Value > maximum {
			maximum = point.Value
		}
	}
	kind := BendTypePrebend
	origin := points[0].Value
	destination := points[len(points)-1].Value
	switch {
	case destination > origin:
		kind = BendTypeBend
	case destination < origin:
		kind = BendTypePrebendRelease
	case maximum > origin:
		kind = BendTypeBendRelease
	}
	return &BendEffect{Points: points, Value: int16(maximum) * int16(GPBendSemitone), Kind: kind}
}

func simplifyBendPoints(points []BendPoint) []BendPoint {
	result := make([]BendPoint, 0, len(points))
	for _, point := range points {
		if len(result) > 0 && result[len(result)-1] == point {
			continue
		}
		result = append(result, point)
	}
	for index := 1; index+1 < len(result); {
		left := result[index-1]
		middle := result[index]
		right := result[index+1]
		leftSpan := int(right.Position) - int(left.Position)
		if leftSpan > 0 && (int(middle.Value)-int(left.Value))*leftSpan == (int(right.Value)-int(left.Value))*(int(middle.Position)-int(left.Position)) {
			result = append(result[:index], result[index+1:]...)
			continue
		}
		index++
	}
	return result
}

func gpifBendPosition(value *string) uint8 {
	if value == nil {
		return 0
	}
	parsed, err := strconv.ParseFloat(*value, 64)
	if err != nil {
		return 0
	}
	return uint8(math.Round(parsed * float64(BendEffectMaxPosition) / 100))
}

func gpifBendValue(value *string) int8 {
	if value == nil {
		return 0
	}
	parsed, err := strconv.ParseFloat(*value, 64)
	if err != nil {
		return 0
	}
	return int8(math.Round(parsed / float64(GPBendSemitone)))
}

func gpifMeasureClef(value string) MeasureClef {
	switch value {
	case "F4":
		return MeasureClefBass
	case "C4":
		return MeasureClefTenor
	case "C3":
		return MeasureClefAlto
	default:
		return MeasureClefTreble
	}
}
