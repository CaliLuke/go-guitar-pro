// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func gp8MasterBar(header *MeasureHeader, bars string) gpifMasterBar {
	mode := "Major"
	if header.KeySignature.IsMinor {
		mode = "Minor"
	}
	result := gpifMasterBar{
		Key:  gpifKey{Mode: mode, AccidentalCount: int(header.KeySignature.Key)},
		Time: fmt.Sprintf("%d/%d", header.TimeSignature.Numerator, header.TimeSignature.Denominator.Value),
		Bars: bars,
	}
	if header.Marker != nil {
		result.Section = &gpifSection{Text: header.Marker.Title}
	}
	if header.RepeatStart || header.RepeatCount > 0 {
		result.Repeat = &gpifRepeat{}
	}
	if header.RepeatStart {
		result.Repeat.Start = "true"
	}
	if header.RepeatCount > 0 {
		result.Repeat.End = "true"
		result.Repeat.Count = int(header.RepeatCount)
	}
	if header.RepeatAlternative != 0 {
		endings := make([]string, 0, 8)
		for index := range 8 {
			if header.RepeatAlternative&(1<<index) != 0 {
				endings = append(endings, strconv.Itoa(index+1))
			}
		}
		result.AlternateEndings = strings.Join(endings, " ")
	}
	if header.DoubleBar {
		result.DoubleBar = &struct{}{}
	}
	switch header.TripletFeel {
	case TripletFeelEighth:
		result.TripletFeel = "Triplet8th"
	case TripletFeelSixteenth:
		result.TripletFeel = "Triplet16th"
	case TripletFeelDottedEighth:
		result.TripletFeel = "Dotted8th"
	case TripletFeelDottedSixteenth:
		result.TripletFeel = "Dotted16th"
	case TripletFeelScottishEighth:
		result.TripletFeel = "Scottish8th"
	case TripletFeelScottishSixteenth:
		result.TripletFeel = "Scottish16th"
	}
	return result
}

type gp8GraceGroup struct {
	duration Duration
	velocity int16
	onBeat   bool
	notes    []Note
}

func (builder *gp8Builder) addGraceBeats(trackIndex int, staffStrings []GuitarString, beat *Beat) ([]string, error) {
	var ids []string
	for _, sequence := range graceSequences(beat) {
		groups := builder.graceGroups(trackIndex, beat, sequence)
		groupIDs, err := builder.addGraceGroups(trackIndex, staffStrings, groups)
		if err != nil {
			return nil, err
		}
		ids = append(ids, groupIDs...)
	}
	return ids, nil
}

func graceSequences(beat *Beat) []uint8 {
	seen := make(map[uint8]struct{})
	for noteIndex := range beat.Notes {
		for _, grace := range beat.Notes[noteIndex].Effect.Graces {
			seen[grace.Sequence] = struct{}{}
		}
	}
	result := make([]uint8, 0, len(seen))
	for sequence := range seen {
		result = append(result, sequence)
	}
	slices.Sort(result)
	return result
}

func (builder *gp8Builder) graceGroups(trackIndex int, beat *Beat, sequence uint8) []gp8GraceGroup {
	var groups []gp8GraceGroup
	for noteIndex := range beat.Notes {
		note := &beat.Notes[noteIndex]
		for graceIndex := range note.Effect.Graces {
			grace := &note.Effect.Graces[graceIndex]
			if grace.Sequence != sequence {
				continue
			}
			duration := defaultDuration()
			duration.Value = uint16(grace.Duration)
			if _, supported := gp8NoteValue(duration.Value); !supported {
				duration.Value = uint16(DurationThirtySecond)
			}
			velocity := grace.Velocity
			if velocity == 0 {
				velocity = DefaultVelocity
			}
			groupIndex := -1
			for index := range groups {
				if groups[index].duration == duration && groups[index].velocity == velocity && groups[index].onBeat == grace.IsOnBeat {
					groupIndex = index
					break
				}
			}
			if groupIndex < 0 {
				groups = append(groups, gp8GraceGroup{duration: duration, velocity: velocity, onBeat: grace.IsOnBeat})
				groupIndex = len(groups) - 1
			}
			graceNote := *note
			graceNote.Value = int16(grace.Fret)
			if grace.ExactFret != nil {
				graceNote.Value = int16(*grace.ExactFret)
			}
			graceNote.PercussionArticulation = grace.PercussionArticulation
			graceNote.HasPercussionArticulation = grace.HasPercussionArticulation
			if builder.song.Tracks[trackIndex].PercussionTrack && (graceNote.Value < 27 || graceNote.Value > 87) {
				graceNote.Value = note.Value
			}
			graceNote.Velocity = velocity
			graceNote.Kind = NoteTypeNormal
			graceNote.Effect = defaultNoteEffect()
			if grace.IsDead {
				graceNote.Kind = NoteTypeDead
			}
			switch grace.Transition {
			case GraceEffectTransitionSlide:
				graceNote.Effect.Slides = []SlideType{SlideLegatoSlideTo}
			case GraceEffectTransitionHammer:
				graceNote.Effect.Hammer = true
			}
			groups[groupIndex].notes = append(groups[groupIndex].notes, graceNote)
		}
	}
	return groups
}

func (builder *gp8Builder) addGraceGroups(trackIndex int, staffStrings []GuitarString, groups []gp8GraceGroup) ([]string, error) {
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		rhythmID, err := builder.addRhythm(group.duration)
		if err != nil {
			return nil, err
		}
		id := strconv.Itoa(len(builder.doc.Beats.Beats))
		graceKind := "BeforeBeat"
		if group.onBeat {
			graceKind = "OnBeat"
		}
		result := gpifBeat{
			ID:         id,
			Rhythm:     gpifRhythmRef{Ref: rhythmID},
			Dynamic:    gp8VelocityToDynamic(group.velocity),
			GraceNotes: graceKind,
		}
		noteIDs := make([]string, 0, len(group.notes))
		for noteIndex := range group.notes {
			noteID, err := builder.addNote(trackIndex, staffStrings, &group.notes[noteIndex])
			if err != nil {
				return nil, err
			}
			noteIDs = append(noteIDs, noteID)
		}
		result.Notes = strings.Join(noteIDs, " ")
		builder.doc.Beats.Beats = append(builder.doc.Beats.Beats, result)
		ids = append(ids, id)
	}
	return ids, nil
}

func (builder *gp8Builder) addBeat(trackIndex int, staffStrings []GuitarString, beat *Beat) (string, error) {
	rhythmID, err := builder.addRhythm(beat.Duration)
	if err != nil {
		return "", err
	}
	beatID := strconv.Itoa(len(builder.doc.Beats.Beats))
	result := gpifBeat{ID: beatID, Rhythm: gpifRhythmRef{Ref: rhythmID}, FreeText: beat.Text}
	if beat.isGrace {
		result.GraceNotes = "BeforeBeat"
		if beat.graceOnBeat {
			result.GraceNotes = "OnBeat"
		}
	}
	result.Dynamic = gp8ConvertBeatDynamic(beat).marking
	if beat.Effect.Chord != nil {
		result.Chord = builder.chordIDs[trackIndex][beat.Effect.Chord]
	}
	if beat.Effect.FadeIn {
		result.Fadding = "FadeIn"
	}
	result.Whammy = gp8Whammy(beat.Effect.TremoloBar)
	switch beat.Effect.Hairpin {
	case HairpinCrescendo:
		result.Hairpin = "Crescendo"
	case HairpinDiminuendo:
		result.Hairpin = "Decrescendo"
	}
	switch beat.Effect.Stroke.Direction {
	case BeatStrokeDirectionUp:
		result.Arpeggio = "Up"
	case BeatStrokeDirectionDown:
		result.Arpeggio = "Down"
	}
	switch beat.Octave {
	case OctaveOttava:
		result.Ottavia = "8va"
	case OctaveOttavaBassa:
		result.Ottavia = "8vb"
	case OctaveQuindicesima:
		result.Ottavia = "15ma"
	case OctaveQuindicesimaBassa:
		result.Ottavia = "15mb"
	}

	noteIDs := make([]string, 0, len(beat.Notes))
	for noteIndex := range beat.Notes {
		noteID, err := builder.addNote(trackIndex, staffStrings, &beat.Notes[noteIndex])
		if err != nil {
			return "", err
		}
		noteIDs = append(noteIDs, noteID)
	}
	result.Notes = strings.Join(noteIDs, " ")
	builder.doc.Beats.Beats = append(builder.doc.Beats.Beats, result)
	return beatID, nil
}

func (builder *gp8Builder) addRhythm(duration Duration) (string, error) {
	if id, ok := builder.rhythmIDs[duration]; ok {
		return id, nil
	}
	noteValue, ok := gp8NoteValue(duration.Value)
	if !ok {
		return "", fmt.Errorf("unsupported duration value %d", duration.Value)
	}
	id := strconv.Itoa(len(builder.doc.Rhythms.Rhythms))
	rhythm := gpifRhythm{ID: id, NoteValue: noteValue}
	switch {
	case duration.DoubleDotted:
		rhythm.AugmentationDot = &gpifAugDot{Count: 2}
	case duration.Dotted:
		rhythm.AugmentationDot = &gpifAugDot{Count: 1}
	}
	if duration.TupletEnters > 0 && duration.TupletTimes > 0 && (duration.TupletEnters != 1 || duration.TupletTimes != 1) {
		rhythm.PrimaryTuplet = &gpifTuplet{Num: int(duration.TupletEnters), Den: int(duration.TupletTimes)}
	}
	builder.rhythmIDs[duration] = id
	builder.doc.Rhythms.Rhythms = append(builder.doc.Rhythms.Rhythms, rhythm)
	return id, nil
}

func (builder *gp8Builder) addNote(trackIndex int, staffStrings []GuitarString, note *Note) (string, error) {
	track := &builder.song.Tracks[trackIndex]
	noteID := strconv.Itoa(len(builder.doc.Notes.Notes))
	fret := int(note.Value)
	midi := gp8NoteMIDI(track, staffStrings, note)
	properties := []gpifProperty{
		{Name: "Fret", Fret: &fret},
		{Name: "Midi", Number: &midi},
	}
	if note.String > 0 {
		stringValue := float64(int(note.String) - 1)
		if !track.PercussionTrack && int(note.String) <= len(staffStrings) {
			stringValue = float64(len(staffStrings) - int(note.String))
		}
		properties = append(properties, gpifProperty{Name: "String", String: &stringValue})
	}
	if !track.PercussionTrack {
		pitch := gp8Pitch(midi)
		properties = append([]gpifProperty{{Name: "ConcertPitch", Pitch: &pitch}, {Name: "TransposedPitch", Pitch: &pitch}}, properties...)
	}
	articulation := 0
	result := gpifNote{ID: noteID, InstrumentArticulation: &articulation, Properties: gpifProperties{Properties: properties}}
	result.Properties.Properties = append(result.Properties.Properties, gp8ConvertBend(note.Effect.Bend).properties...)
	if note.Effect.Harmonic != nil {
		harmonicType := gp8HarmonicType(note.Effect.Harmonic.Kind)
		if harmonicType != "" {
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HarmonicType", HType: &harmonicType})
		}
		var harmonicFret *float64
		if note.Effect.Harmonic.FretFloat != nil {
			harmonicFret = note.Effect.Harmonic.FretFloat
		} else if note.Effect.Harmonic.Fret != nil {
			value := float64(*note.Effect.Harmonic.Fret)
			harmonicFret = &value
		}
		if harmonicFret != nil {
			value := strconv.FormatFloat(*harmonicFret, 'f', -1, 64)
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HarmonicFret", HFret: &value})
		}
	}
	if track.PercussionTrack {
		if index, ok := gp8PercussionArticulationIndex(track, note); ok {
			articulation = index
		} else {
			var ok bool
			articulation, ok = builder.articulationIDs[trackIndex][note.Value]
			if !ok {
				return "", fmt.Errorf("percussion MIDI value %d has no exported articulation resource", note.Value)
			}
		}
		result.InstrumentArticulation = &articulation
	}
	if note.TieOrigin || note.Kind == NoteTypeTie {
		result.Tie = &gpifTie{
			Origin:      strconv.FormatBool(note.TieOrigin),
			Destination: strconv.FormatBool(note.Kind == NoteTypeTie),
		}
	}
	if note.Kind == NoteTypeDead || note.Effect.DeadNote || note.Effect.GhostNote {
		if note.Kind == NoteTypeDead || note.Effect.DeadNote {
			enable := ""
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Muted", Enable: &enable})
		}
		if note.Effect.GhostNote {
			result.AntiAccent = "Normal"
		}
	}
	if note.Effect.LetRing {
		value := ""
		result.LetRing = &value
	}
	if note.Effect.PalmMute {
		value := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "PalmMuted", Enable: &value})
	}
	if note.Effect.Hammer {
		enable := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HopoOrigin", Enable: &enable})
	}
	if note.Effect.Tapped {
		enable := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Tapped", Enable: &enable})
	}
	if note.Effect.LeftHandTapped {
		enable := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "LeftHandTapped", Enable: &enable})
	}
	if len(note.Effect.Slides) > 0 {
		flags := 0
		for _, slide := range note.Effect.Slides {
			switch slide {
			case SlideShiftSlideTo:
				flags |= 0x01
			case SlideLegatoSlideTo:
				flags |= 0x02
			case SlideOutDownwards:
				flags |= 0x04
			case SlideOutUpwards:
				flags |= 0x08
			case SlideIntoFromBelow:
				flags |= 0x10
			case SlideIntoFromAbove:
				flags |= 0x20
			case SlidePickSlideDown:
				flags |= 0x40
			case SlidePickSlideUp:
				flags |= 0x80
			}
		}
		value := strconv.Itoa(flags)
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Slide", Flags: &value})
	}
	result.Vibrato = gp8ResolveNoteVibrato(note.Effect)
	if note.Effect.Trill != nil {
		result.Trill = &gpifTrill{Fret: int(note.Effect.Trill.Fret)}
	}
	if note.Effect.Staccato {
		result.Accent |= 0x01
	}
	accentFlags, _ := gp8ResolveNoteAccent(note.Effect)
	result.Accent |= accentFlags
	builder.doc.Notes.Notes = append(builder.doc.Notes.Notes, result)
	return noteID, nil
}

func gp8BarClef(song *Song, trackIndex int, measure *Measure) string {
	if song.Tracks[trackIndex].PercussionTrack {
		return "Neutral"
	}
	switch measure.Clef {
	case MeasureClefBass:
		return "F4"
	case MeasureClefTenor:
		return "C4"
	case MeasureClefAlto:
		return "C3"
	}
	track := &song.Tracks[trackIndex]
	if track.ChannelIndex >= 0 && track.ChannelIndex < len(song.Channels) {
		program := song.Channels[track.ChannelIndex].Instrument
		if program >= 32 && program <= 39 {
			return "F4"
		}
	}
	return "G2"
}

func gp8HarmonicType(kind HarmonicType) string {
	switch kind {
	case HarmonicTypeNatural:
		return "Natural"
	case HarmonicTypeArtificial:
		return "Artificial"
	case HarmonicTypePinch:
		return "Pinch"
	case HarmonicTypeTapped:
		return "Tap"
	case HarmonicTypeSemi:
		return "Semi"
	case HarmonicTypeFeedback:
		return "Feedback"
	default:
		return ""
	}
}
