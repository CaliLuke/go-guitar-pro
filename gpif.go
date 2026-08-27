// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"fmt"
	"math"
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
	ID               string             `xml:"id,attr"`
	Name             string             `xml:"Name"`
	Color            string             `xml:"Color,omitempty"`
	Instrument       *gpifInstrument    `xml:"Instrument,omitempty"`
	InstrumentSet    *gpifInstrumentSet `xml:"InstrumentSet,omitempty"`
	GeneralMidi      *gpifGeneralMidi   `xml:"GeneralMidi,omitempty"`
	Staves           gpifStaves         `xml:"Staves"`
	Sounds           gpifSounds         `xml:"Sounds"`
	Transpose        *gpifTranspose     `xml:"Transpose,omitempty"`
	RSE              *gpifTrackRSE      `xml:"RSE,omitempty"`
	MidiConnection   gpifMidiConnection `xml:"MidiConnection"`
	PlaybackState    string             `xml:"PlaybackState,omitempty"`
	AudioEngineState string             `xml:"AudioEngineState,omitempty"`
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
	StringCount int               `xml:"stringCount,attr,omitempty"`
	FretCount   int               `xml:"fretCount,attr,omitempty"`
	BaseFret    int               `xml:"baseFret,attr,omitempty"`
	Frets       []gpifDiagramFret `xml:"Fret"`
}

type gpifDiagramFret struct {
	String int `xml:"string,attr"`
	Fret   int `xml:"fret,attr"`
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
	Count int    `xml:"Count,omitempty"`
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
	Properties gpifProperties `xml:"Properties"`
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
	var doc gpifDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing GPIF XML: %w", err)
	}

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
				// Parse string tuning from staves
				for _, staff := range t.Staves.Staff {
					for _, prop := range staff.Properties {
						if prop.Name == "Tuning" && prop.Pitches != "" {
							vals := splitIDs(prop.Pitches)
							track.Strings = nil
							for sourceIndex := len(vals) - 1; sourceIndex >= 0; sourceIndex-- {
								if v, err := strconv.ParseInt(vals[sourceIndex], 10, 8); err == nil {
									track.Strings = append(track.Strings, GuitarString{
										Number: int8(len(track.Strings) + 1),
										Value:  int8(v),
									})
								}
							}
						}
					}
				}
				chordMap = gpifReadChordMap(t.Staves)
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
		song.Tracks = append(song.Tracks, track)
		trackChordMaps = append(trackChordMaps, chordMap)
	}

	// Parse master bars → measure headers + measures
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

		// Parse bars for each track
		barIDs := splitIDs(mb.Bars)
		for trackIdx := range song.Tracks {
			m := defaultMeasure()
			m.Number = mbIdx + 1
			m.TrackIndex = trackIdx
			m.HeaderIndex = mbIdx
			m.TimeSignature = mh.TimeSignature
			m.KeySignature = mh.KeySignature
			m.HasDoubleBar = mh.DoubleBar

			if trackIdx < len(barIDs) {
				barID := barIDs[trackIdx]
				if bar, ok := barMap[barID]; ok {
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
									// Rhythm → duration
									if r, ok := rhythmMap[b.Rhythm.Ref]; ok {
										beat.Duration = gpifRhythmToDuration(r)
									}

									// Beat effects
									beat.Effect.FadeIn = b.Fadding == "FadeIn"
									gpifApplyBeatEffects(b, &beat)
									if trackIdx < len(trackChordMaps) {
										if chord, ok := trackChordMaps[trackIdx][b.Chord]; ok {
											beat.Effect.Chord = &chord
										}
									}
									switch b.GraceNotes {
									case "OnBeat":
										isGrace = true
										graceOnBeat = true
									case "BeforeBeat":
										isGrace = true
									}

									// Notes
									velocity := gpifDynamicToVelocity(b.Dynamic)
									noteIDs := splitIDs(b.Notes)
									for _, noteID := range noteIDs {
										if noteID == "-1" {
											continue
										}
										if n, ok := noteMap[noteID]; ok {
											note := gpifNoteToNote(
												n,
												len(song.Tracks[trackIdx].Strings),
												song.Tracks[trackIdx].PercussionTrack,
											)
											note.Velocity = velocity
											beat.Notes = append(beat.Notes, note)
										}
									}
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
									gpifApplyPendingGrace(&beat, pendingGrace, song.Tracks[trackIdx].PercussionTrack)...,
								)
								pendingGrace = nil
								voice.Beats = append(voice.Beats, beat)
							}
							voice.Beats = append(
								voice.Beats,
								gpifApplyPendingGrace(nil, pendingGrace, song.Tracks[trackIdx].PercussionTrack)...,
							)
						}
						m.Voices = append(m.Voices, voice)
					}
				}
			}

			song.Tracks[trackIdx].Measures = append(song.Tracks[trackIdx].Measures, m)
		}
	}

	// Set measure header starts
	start := DurationQuarterTime
	for i := range song.MeasureHeaders {
		song.MeasureHeaders[i].Start = start
		start += song.MeasureHeaders[i].length()
	}

	return song, nil
}

type gpifPendingGrace struct {
	beat   Beat
	onBeat bool
}

func gpifApplyPendingGrace(target *Beat, pending []gpifPendingGrace, percussion bool) []Beat {
	var orphans []Beat
	for _, pendingBeat := range pending {
		orphan := pendingBeat.beat
		orphan.Notes = nil
		for noteIndex := range pendingBeat.beat.Notes {
			graceNote := pendingBeat.beat.Notes[noteIndex]
			effect := gpifGraceEffect(&graceNote, &pendingBeat.beat.Duration, pendingBeat.onBeat)
			targetIndex := gpifGraceTarget(target, &graceNote, percussion)
			if targetIndex >= 0 {
				target.Notes[targetIndex].Effect.Grace = &effect
				continue
			}
			graceNote.Effect.Grace = &effect
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
			if target.Notes[index].Effect.Grace == nil && target.Notes[index].Value == grace.Value {
				return index
			}
		}
	}
	for index := range target.Notes {
		if target.Notes[index].Effect.Grace == nil && target.Notes[index].String == grace.String {
			return index
		}
	}
	return -1
}

func gpifGraceEffect(note *Note, duration *Duration, onBeat bool) GraceEffect {
	transition := GraceEffectTransitionNone
	switch {
	case note.Effect.Hammer:
		transition = GraceEffectTransitionHammer
	case len(note.Effect.Slides) > 0:
		transition = GraceEffectTransitionSlide
	}
	return GraceEffect{
		Duration:   uint8(min(uint16(math.MaxUint8), duration.Value)),
		Fret:       int8(min(int16(math.MaxInt8), max(int16(math.MinInt8), note.Value))),
		IsDead:     note.Kind == NoteTypeDead,
		IsOnBeat:   onBeat,
		Transition: transition,
		Velocity:   note.Velocity,
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

func gpifReadChordMap(staves gpifStaves) map[string]Chord {
	chords := make(map[string]Chord)
	for _, staff := range staves.Staff {
		for _, property := range staff.Properties {
			if property.Name != "DiagramCollection" || property.Items == nil {
				continue
			}
			for _, item := range property.Items.Items {
				if item.ID == "" {
					continue
				}
				chord := Chord{Name: item.Name}
				if item.Diagram != nil {
					chord.Length = uint8(item.Diagram.StringCount)
					chord.Strings = make([]int8, item.Diagram.StringCount)
					for index := range chord.Strings {
						chord.Strings[index] = -1
					}
					for _, fret := range item.Diagram.Frets {
						index := item.Diagram.StringCount - fret.String - 1
						if index >= 0 && index < len(chord.Strings) {
							chord.Strings[index] = int8(fret.Fret)
						}
					}
				}
				chords[item.ID] = chord
			}
		}
	}
	return chords
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

func gpifAutomationIsBefore(a, b TempoAutomation) bool {
	if a.Bar != b.Bar {
		return a.Bar < b.Bar
	}
	return a.Position < b.Position
}

func gpifApplyBeatEffects(b *gpifBeat, beat *Beat) {
	// Tremolo picking
	if b.Tremolo != "" {
		tp := TremoloPickingEffect{Duration: defaultDuration()}
		switch b.Tremolo {
		case "1/2":
			tp.Duration.Value = uint16(DurationEighth)
		case "1/4":
			tp.Duration.Value = uint16(DurationSixteenth)
		case "1/8":
			tp.Duration.Value = uint16(DurationThirtySecond)
		}
		// Apply to all notes in the beat (set after notes are parsed)
		_ = tp // stored for later use on notes
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

func gpifNoteToNote(n *gpifNote, stringCount int, percussion bool) Note {
	note := defaultNote()
	note.Kind = NoteTypeNormal
	hasFret := false

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
			if p.Float != nil && note.Effect.Harmonic != nil {
				if v, err := strconv.ParseFloat(*p.Float, 64); err == nil {
					fret := int8(v)
					note.Effect.Harmonic.Fret = &fret
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
