// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// GPIF XML structures for unmarshalling.

type gpifDocument struct {
	XMLName     xml.Name        `xml:"GPIF"`
	Score       gpifScore       `xml:"Score"`
	MasterTrack gpifMasterTrack `xml:"MasterTrack"`
	Tracks      gpifTracks      `xml:"Tracks"`
	MasterBars  gpifMasterBars  `xml:"MasterBars"`
	Bars        gpifBars        `xml:"Bars"`
	Voices      gpifVoices      `xml:"Voices"`
	Beats       gpifBeats       `xml:"Beats"`
	Notes       gpifNotes       `xml:"Notes"`
	Rhythms     gpifRhythms     `xml:"Rhythms"`
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
	Automations gpifAutomations `xml:"Automations"`
}

type gpifAutomations struct {
	Automations []gpifAutomation `xml:"Automation"`
}

type gpifAutomation struct {
	Type     string  `xml:"Type"`
	Value    string  `xml:"Value"`
	Visible  string  `xml:"Visible"`
	Text     string  `xml:"Text"`
	Bar      int     `xml:"Bar"`
	Position float64 `xml:"Position"`
}

type gpifTracks struct {
	Tracks []gpifTrack `xml:"Track"`
}

type gpifTrack struct {
	ID            string            `xml:"id,attr"`
	Name          string            `xml:"Name"`
	Color         string            `xml:"Color"`
	Instrument    gpifInstrument    `xml:"Instrument"`
	InstrumentSet gpifInstrumentSet `xml:"InstrumentSet"`
	GeneralMidi   *gpifGeneralMidi  `xml:"GeneralMidi"`
	Staves        gpifStaves        `xml:"Staves"`
	Sounds        gpifSounds        `xml:"Sounds"`
	Transpose     gpifTranspose     `xml:"Transpose"`
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
	Properties []gpifStaffProperty `xml:"Property"`
}

type gpifStaffProperty struct {
	Name   string     `xml:"name,attr"`
	Tuning gpifTuning `xml:"Tuning"`
}

type gpifTuning struct {
	Values string `xml:"Values,attr"`
}

type gpifInstrumentSet struct {
	Type string `xml:"Type"`
}

func (t *gpifTrack) isPercussionTrack() bool {
	if t.InstrumentSet.Type == "drums" || t.InstrumentSet.Type == "percussion" || t.Instrument.Ref == "drmkt" {
		return true
	}
	if t.GeneralMidi == nil || t.GeneralMidi.PrimaryChannel < 0 || t.GeneralMidi.PrimaryChannel > math.MaxUint8 {
		return false
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
	Section          gpifSection `xml:"Section"`
	Key              gpifKey     `xml:"Key"`
	Time             string      `xml:"Time"`
	Bars             string      `xml:"Bars"`
	AlternateEndings string      `xml:"AlternateEndings"`
	DoubleBar        string      `xml:"DoubleBar"`
	TripletFeel      string      `xml:"TripletFeel"`
	Repeat           gpifRepeat  `xml:"Repeat"`
}

type gpifKey struct {
	Mode            string `xml:"Mode"`
	AccidentalCount int    `xml:"AccidentalCount"`
}

type gpifRepeat struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
	Count int    `xml:"Count"`
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
	Notes      string         `xml:"Notes"`
	Dynamic    string         `xml:"Dynamic"`
	GraceNotes string         `xml:"GraceNotes"`
	Fadding    string         `xml:"Fadding"`
	Tremolo    string         `xml:"Tremolo"`
	Arpeggio   string         `xml:"Arpeggio"`
	Hairpin    string         `xml:"Hairpin"`
	FreeText   string         `xml:"FreeText"`
	Ottavia    string         `xml:"Ottavia"`
	Wah        string         `xml:"Wah"`
	Properties gpifProperties `xml:"Properties"`
}

type gpifRhythmRef struct {
	Ref string `xml:"ref,attr"`
}

type gpifProperties struct {
	Properties []gpifProperty `xml:"Property"`
}

type gpifProperty struct {
	Fret      *int     `xml:"Fret"`
	String    *float64 `xml:"String"`
	Float     *string  `xml:"Float"`
	Enable    *string  `xml:"Enable"`
	Number    *int     `xml:"Number"`
	HType     *string  `xml:"HType"`
	Flags     *string  `xml:"Flags"`
	Direction *string  `xml:"Direction"`
	Strength  *string  `xml:"Strength"`
	Name      string   `xml:"name,attr"`
}

type gpifNotes struct {
	Notes []gpifNote `xml:"Note"`
}

type gpifNote struct {
	LetRing    *string        `xml:"LetRing"`
	Trill      *gpifTrill     `xml:"Trill"`
	Tie        gpifTie        `xml:"Tie"`
	ID         string         `xml:"id,attr"`
	Vibrato    string         `xml:"Vibrato"`
	AntiAccent string         `xml:"AntiAccent"`
	Properties gpifProperties `xml:"Properties"`
	Accent     int            `xml:"Accent"`
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
	for _, trackID := range trackIDs {
		track := defaultTrack()
		track.Number = int32(len(song.Tracks))
		for _, t := range doc.Tracks.Tracks {
			if t.ID == trackID {
				track.Name = t.Name
				track.PercussionTrack = t.isPercussionTrack()
				// Parse string tuning from staves
				for _, staff := range t.Staves.Staff {
					for _, prop := range staff.Properties {
						if prop.Name == "Tuning" && prop.Tuning.Values != "" {
							vals := splitIDs(prop.Tuning.Values)
							track.Strings = nil
							for si, sv := range vals {
								if v, err := strconv.Atoi(sv); err == nil {
									track.Strings = append(track.Strings, GuitarString{
										Number: int8(si + 1),
										Value:  int8(v),
									})
								}
							}
						}
					}
				}
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
				// MIDI info from sounds
				if len(t.Sounds.Sounds) > 0 {
					s := t.Sounds.Sounds[0]
					ch := defaultMidiChannel()
					ch.Instrument = int32(s.Program)
					ch.Channel = uint8(s.Channel)
					song.Channels = append(song.Channels, ch)
					track.ChannelIndex = len(song.Channels) - 1
				}
				break
			}
		}
		song.Tracks = append(song.Tracks, track)
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

		// Repeat
		mh.RepeatOpen = mb.Repeat.Start == "true"
		if mb.Repeat.End == "true" && mb.Repeat.Count > 0 {
			mh.RepeatClose = int8(mb.Repeat.Count - 1)
		}

		// Section marker
		if mb.Section.Text != "" || mb.Section.Letter != "" {
			title := mb.Section.Text
			if title == "" {
				title = mb.Section.Letter
			}
			mh.Marker = &Marker{Title: title}
		}

		// Double bar
		mh.DoubleBar = mb.DoubleBar != ""

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
		for trackIdx := 0; trackIdx < len(song.Tracks); trackIdx++ {
			m := defaultMeasure()
			m.Number = mbIdx + 1
			m.TrackIndex = trackIdx
			m.HeaderIndex = mbIdx

			if trackIdx < len(barIDs) {
				barID := barIDs[trackIdx]
				if bar, ok := barMap[barID]; ok {
					voiceIDs := splitIDs(bar.Voices)
					for _, voiceID := range voiceIDs {
						if voiceID == "-1" {
							continue
						}
						voice := Voice{}
						if v, ok := voiceMap[voiceID]; ok {
							beatIDs := splitIDs(v.Beats)
							for _, beatID := range beatIDs {
								if beatID == "-1" {
									continue
								}
								beat := defaultBeat()
								if b, ok := beatMap[beatID]; ok {
									// Rhythm → duration
									if r, ok := rhythmMap[b.Rhythm.Ref]; ok {
										beat.Duration = gpifRhythmToDuration(r)
									}

									// Beat effects
									beat.Effect.FadeIn = b.Fadding == "FadeIn"
									gpifApplyBeatEffects(b, &beat)

									// Grace notes
									var graceEffect *GraceEffect
									switch b.GraceNotes {
									case "OnBeat":
										graceEffect = &GraceEffect{IsOnBeat: true, Duration: 1, Velocity: DefaultVelocity, Transition: GraceEffectTransitionNone}
									case "BeforeBeat":
										graceEffect = &GraceEffect{IsOnBeat: false, Duration: 1, Velocity: DefaultVelocity, Transition: GraceEffectTransitionNone}
									}

									// Notes
									velocity := gpifDynamicToVelocity(b.Dynamic)
									noteIDs := splitIDs(b.Notes)
									for _, noteID := range noteIDs {
										if noteID == "-1" {
											continue
										}
										if n, ok := noteMap[noteID]; ok {
											note := gpifNoteToNote(n)
											note.Velocity = velocity
											if graceEffect != nil {
												ge := *graceEffect
												ge.Fret = int8(note.Value)
												note.Effect.Grace = &ge
											}
											beat.Notes = append(beat.Notes, note)
										}
									}
								}
								voice.Beats = append(voice.Beats, beat)
							}
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

func gpifReadTempoAutomations(automations []gpifAutomation, song *Song) {
	earliest := -1
	for _, auto := range automations {
		if auto.Type != "Tempo" {
			continue
		}
		parts := strings.Fields(auto.Value)
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

func gpifNoteToNote(n *gpifNote) Note {
	note := defaultNote()
	note.Kind = NoteTypeNormal

	// Parse properties
	for _, p := range n.Properties.Properties {
		switch p.Name {
		case "Fret":
			if p.Fret != nil {
				note.Value = int16(*p.Fret)
			}
		case "String":
			if p.String != nil {
				note.String = int8(int(*p.String) + 1) // GPIF is 0-based, our model is 1-based
			}
		case "Midi":
			if p.Number != nil && note.Value == 0 {
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
	if n.Tie.Destination == "true" {
		note.Kind = NoteTypeTie
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
