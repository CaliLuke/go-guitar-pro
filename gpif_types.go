// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"
)

// GPIF XML structures shared by parsing and export.

var gpifBackingTrackAssetReferenceSource = diagnosticSource("GPIF.BackingTrack.AssetId.Reference", "score-core", ParseDiagnosticInvalidData)

var gpifSoundBankInvalidSource = diagnosticSource("GPIF.Track.Sound.MIDI.Bank.Invalid", "midi-bank", ParseDiagnosticInvalidData)

var (
	gpifTrackPropertyConflictSource = diagnosticSource("GPIF.Track.Property.ConflictingDuplicate", "staff-ownership", ParseDiagnosticInvalidData)
	gpifStaffPropertyConflictSource = diagnosticSource("GPIF.Staff.Property.ConflictingDuplicate", "staff-ownership", ParseDiagnosticInvalidData)
	gpifBeatPropertyConflictSource  = diagnosticSource("GPIF.Beat.Property.ConflictingDuplicate", "note-and-beat-semantics", ParseDiagnosticInvalidData)
	gpifNotePropertyConflictSource  = diagnosticSource("GPIF.Note.Property.ConflictingDuplicate", "note-and-beat-semantics", ParseDiagnosticInvalidData)
)

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
	Title         string `xml:"Title"`
	SubTitle      string `xml:"SubTitle"`
	Artist        string `xml:"Artist"`
	Album         string `xml:"Album"`
	Words         string `xml:"Words"`
	Music         string `xml:"Music"`
	WordsAndMusic string `xml:"WordsAndMusic"`
	Copyright     string `xml:"Copyright"`
	Tabber        string `xml:"Tabber"`
	Instructions  string `xml:"Instructions"`
	Notices       string `xml:"Notices"`
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
	cdata         bool
}

func (value gpifAutomationValue) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	if value.cdata {
		return encoder.EncodeElement(struct {
			Text string `xml:",cdata"`
		}{Text: value.Text}, start)
	}
	type wireValue gpifAutomationValue
	return encoder.EncodeElement(wireValue(value), start)
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
	GeneralMidi      *gpifGeneralMidi    `xml:"GeneralMidi,omitempty"`
	Staves           gpifStaves          `xml:"Staves"`
	InstrumentSet    *gpifInstrumentSet  `xml:"InstrumentSet,omitempty"`
	NotationPatch    *gpifInstrumentSet  `xml:"NotationPatch,omitempty"`
	Properties       []gpifStaffProperty `xml:"Properties>Property"`
	Sounds           gpifSounds          `xml:"Sounds"`
	Automations      gpifAutomations     `xml:"Automations"`
	PartSounding     *gpifPartSounding   `xml:"PartSounding,omitempty"`
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
	Program          *int `xml:"Program"`
	Port             int  `xml:"Port"`
	PrimaryChannel   int  `xml:"PrimaryChannel"`
	SecondaryChannel int  `xml:"SecondaryChannel"`
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
	Label   *string    `xml:"Label,omitempty"`
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
	String *int   `xml:"string,attr"`
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
	Chromatic int64 `xml:"Chromatic"`
	Octave    int64 `xml:"Octave"`
}

type gpifPartSounding struct {
	NominalKey         string `xml:"NominalKey"`
	TranspositionPitch int64  `xml:"TranspositionPitch"`
}

type gpifSounds struct {
	Sounds []gpifSound `xml:"Sound"`
}

type gpifSound struct {
	Name    string `xml:"Name"`
	Label   string `xml:"Label,omitempty"`
	Path    string `xml:"Path,omitempty"`
	Role    string `xml:"Role,omitempty"`
	LSB     int    `xml:"MIDI>LSB"`
	MSB     int    `xml:"MIDI>MSB"`
	Program int    `xml:"MIDI>Program"`
	Channel *int   `xml:"MIDI>PrimaryChannel"`
}

type gpifMasterBars struct {
	MasterBars []gpifMasterBar `xml:"MasterBar"`
}

type gpifMasterBar struct {
	Section          *gpifSection     `xml:"Section,omitempty"`
	Key              gpifKey          `xml:"Key"`
	Time             string           `xml:"Time"`
	FreeTime         *struct{}        `xml:"FreeTime,omitempty"`
	Fermatas         *gpifFermatas    `xml:"Fermatas,omitempty"`
	Bars             string           `xml:"Bars"`
	AlternateEndings string           `xml:"AlternateEndings,omitempty"`
	DoubleBar        *struct{}        `xml:"DoubleBar,omitempty"`
	TripletFeel      string           `xml:"TripletFeel,omitempty"`
	Repeat           *gpifRepeat      `xml:"Repeat,omitempty"`
	Directions       *gpifDirections  `xml:"Directions,omitempty"`
	XProperties      *gpifXProperties `xml:"XProperties,omitempty"`
}

type gpifFermatas struct {
	Fermatas []gpifFermata `xml:"Fermata"`
}

type gpifFermata struct {
	Type   string `xml:"Type"`
	Offset string `xml:"Offset"`
	Length string `xml:"Length"`
}

type gpifDirections struct {
	Targets []string `xml:"Target"`
	Jumps   []string `xml:"Jump"`
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
	ID         string `xml:"id,attr"`
	Voices     string `xml:"Voices"`
	Clef       string `xml:"Clef"`
	Ottavia    string `xml:"Ottavia,omitempty"`
	SimileMark string `xml:"SimileMark,omitempty"`
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
	ID                                 string           `xml:"id,attr"`
	Rhythm                             gpifRhythmRef    `xml:"Rhythm"`
	Notes                              string           `xml:"Notes,omitempty"`
	Chord                              string           `xml:"Chord,omitempty"`
	Dynamic                            string           `xml:"Dynamic,omitempty"`
	GraceNotes                         string           `xml:"GraceNotes,omitempty"`
	Fadding                            string           `xml:"Fadding,omitempty"`
	Tremolo                            string           `xml:"Tremolo,omitempty"`
	Arpeggio                           string           `xml:"Arpeggio,omitempty"`
	Hairpin                            string           `xml:"Hairpin,omitempty"`
	Legato                             *gpifLegato      `xml:"Legato,omitempty"`
	Lyrics                             *gpifBeatLyrics  `xml:"Lyrics,omitempty"`
	DeadSlapped                        *struct{}        `xml:"DeadSlapped,omitempty"`
	Golpe                              string           `xml:"Golpe,omitempty"`
	FreeText                           string           `xml:"FreeText,omitempty"`
	Ottavia                            string           `xml:"Ottavia,omitempty"`
	TransposedPitchStemOrientation     string           `xml:"TransposedPitchStemOrientation,omitempty"`
	UserTransposedPitchStemOrientation string           `xml:"UserTransposedPitchStemOrientation,omitempty"`
	Wah                                string           `xml:"Wah,omitempty"`
	Whammy                             *gpifWhammy      `xml:"Whammy,omitempty"`
	Properties                         gpifProperties   `xml:"Properties"`
	XProperties                        *gpifXProperties `xml:"XProperties,omitempty"`
}

type gpifBeatLyrics struct {
	Lines []string `xml:"Line"`
}

type gpifXProperties struct {
	Properties []gpifXProperty `xml:"XProperty"`
}

type gpifXProperty struct {
	ID  string  `xml:"id,attr"`
	Int *string `xml:"Int,omitempty"`
}

type gpifLegato struct {
	Origin      string `xml:"origin,attr"`
	Destination string `xml:"destination,attr"`
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
	LeftFingering          *string        `xml:"LeftFingering,omitempty"`
	RightFingering         *string        `xml:"RightFingering,omitempty"`
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
