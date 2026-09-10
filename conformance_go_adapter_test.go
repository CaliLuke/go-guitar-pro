// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func normalizeGoScore(song *Song) any {
	masterBars := make([]any, 0, len(song.MeasureHeaders))
	for index, header := range song.MeasureHeaders {
		masterBars = append(masterBars, map[string]any{
			"index":            index,
			"start":            header.Start - DurationQuarterTime,
			"timeSignature":    []any{header.TimeSignature.Numerator, header.TimeSignature.Denominator.Value},
			"repeatStart":      header.RepeatStart,
			"repeatCount":      header.RepeatCount,
			"alternateEndings": header.RepeatAlternative,
			"tripletFeel":      goTripletFeel(header.TripletFeel),
			"pickup":           index == 0 && song.Anacrusis,
		})
	}
	tempoAutomations := make([]any, 0, len(song.TempoAutomations)+1)
	if len(song.TempoAutomations) == 0 && song.Tempo > 0 {
		tempoAutomations = append(tempoAutomations, map[string]any{
			"bar": 0, "position": float64(0), "type": "tempo", "value": float64(song.Tempo), "linear": false,
		})
	} else {
		for _, automation := range song.TempoAutomations {
			tempoAutomations = append(tempoAutomations, map[string]any{
				"bar": automation.Bar, "position": automation.Position, "type": "tempo", "value": automation.Tempo, "linear": false,
			})
		}
	}
	tracks := make([]any, 0, len(song.Tracks))
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		program := int32(0)
		primaryChannel := uint8(0)
		if track.ChannelIndex >= 0 && track.ChannelIndex < len(song.Channels) {
			program = song.Channels[track.ChannelIndex].Instrument
			primaryChannel = song.Channels[track.ChannelIndex].Channel
		}
		tracks = append(tracks, map[string]any{
			"index":                   trackIndex,
			"name":                    track.Name,
			"program":                 program,
			"primaryChannel":          primaryChannel,
			"percussionArticulations": normalizeGoPercussionArticulations(track.PercussionArticulations),
			"staves":                  normalizeGoStaves(song, trackIndex),
		})
	}
	return map[string]any{
		"schemaVersion": 1,
		"metadata": map[string]any{
			"title": song.Name, "subtitle": song.Subtitle, "artist": song.Artist, "album": song.Album,
			"words": song.Words, "music": song.Author, "copyright": song.Copyright, "instructions": song.Instructions,
		},
		"masterBars":       masterBars,
		"tempoAutomations": tempoAutomations,
		"tracks":           tracks,
	}
}

func normalizeGoTuning(strings []GuitarString) []any {
	result := make([]any, len(strings))
	for index := range strings {
		result[index] = strings[index].Value
	}
	return result
}

func normalizeGoPercussionArticulations(articulations []PercussionArticulation) []any {
	result := make([]any, 0, len(articulations))
	for _, articulation := range articulations {
		techniqueSymbol := "none"
		if articulation.TechniqueSymbol != "" {
			techniqueSymbol = strings.ToLower(articulation.TechniqueSymbol)
		}
		var inputMIDINumber any
		if len(articulation.InputMIDINumbers) > 0 {
			inputMIDINumber = articulation.InputMIDINumbers[0]
		}
		result = append(result, map[string]any{
			"elementName":        articulation.ElementName,
			"staffLine":          articulation.StaffLine,
			"noteheadDefault":    strings.ToLower(articulation.NoteheadDefault),
			"noteheadHalf":       strings.ToLower(articulation.NoteheadHalf),
			"noteheadWhole":      strings.ToLower(articulation.NoteheadWhole),
			"techniquePlacement": strings.ToLower(articulation.TechniquePlacement),
			"techniqueSymbol":    techniqueSymbol,
			"inputMidiNumber":    inputMIDINumber,
			"outputMidiNumber":   articulation.OutputMIDINumber,
		})
	}
	return result
}

func normalizeGoStaves(song *Song, trackIndex int) []any {
	track := &song.Tracks[trackIndex]
	staves := track.Staves
	if len(staves) == 0 {
		staves = []Staff{{
			Measures:                  track.Measures,
			Strings:                   track.Strings,
			PercussionTrack:           track.PercussionTrack,
			StandardNotationLineCount: 5,
		}}
	}
	result := make([]any, 0, len(staves))
	for staffIndex := range staves {
		staff := staves[staffIndex]
		staff.PercussionTrack = staff.PercussionTrack || track.PercussionTrack
		tuning := normalizeGoTuning(staff.Strings)
		if staff.PercussionTrack {
			tuning = []any{}
		}
		result = append(result, map[string]any{
			"index":                     staffIndex,
			"capo":                      track.CapoFret,
			"percussion":                staff.PercussionTrack,
			"standardNotationLineCount": staff.StandardNotationLineCount,
			"tuning":                    tuning,
			"bars":                      normalizeGoBars(song, track, &staff),
		})
	}
	return result
}

func normalizeGoBars(song *Song, track *Track, staff *Staff) []any {
	result := make([]any, 0, len(staff.Measures))
	links := goNoteLinkProjections(staff)
	for measureIndex := range staff.Measures {
		measure := &staff.Measures[measureIndex]
		voices := make([]any, 0, len(measure.Voices))
		for voiceIndex := range measure.Voices {
			voice := &measure.Voices[voiceIndex]
			if !goVoiceHasContent(voice) {
				continue
			}
			beats := make([]any, 0, len(voice.Beats))
			pendingGrace := make([]*Beat, 0)
			for beatIndex := range voice.Beats {
				beat := &voice.Beats[beatIndex]
				if beat.isGrace {
					pendingGrace = append(pendingGrace, beat)
					continue
				}
				pendingGrace = pendingGrace[:0]
				beats = append(beats, normalizeGoBeat(song, measureIndex, track, staff, beat, links))
			}
			for _, grace := range pendingGrace {
				beats = append(beats, normalizeGoBeat(song, measureIndex, track, staff, grace, links))
			}
			voices = append(voices, map[string]any{"index": voiceIndex, "beats": beats})
		}
		clef := goClef(measure.Clef)
		if staff.PercussionTrack {
			clef = "neutral"
		}
		result = append(result, map[string]any{"index": measureIndex, "clef": clef, "simileMark": goSimileMark(measure.SimileMark), "voices": voices})
	}
	return result
}

func goSimileMark(mark SimileMark) string {
	switch mark {
	case SimileMarkNone:
		return "none"
	case SimileMarkSimple:
		return "simple"
	case SimileMarkFirstOfDouble:
		return "first-of-double"
	case SimileMarkSecondOfDouble:
		return "second-of-double"
	default:
		return fmt.Sprintf("unknown:%d", mark)
	}
}

func goVoiceHasContent(voice *Voice) bool {
	return slices.ContainsFunc(voice.Beats, func(beat Beat) bool {
		return beat.Status != BeatStatusEmpty
	})
}

func normalizeGoBeat(song *Song, measureIndex int, track *Track, staff *Staff, beat *Beat, links map[*Note]goNoteLinkProjection) any {
	start := any(nil)
	if beat.Start != nil {
		start = *beat.Start - song.MeasureHeaders[measureIndex].Start
	}
	notes := make([]any, 0, len(beat.Notes))
	for noteIndex := range beat.Notes {
		note := &beat.Notes[noteIndex]
		notes = append(notes, normalizeGoNoteWithLinks(track, staff, note, links[note]))
	}
	return map[string]any{
		"start":          start,
		"status":         goBeatStatus(beat.Status),
		"graceRole":      map[bool]string{false: "none", true: "orphan"}[beat.isGrace],
		"duration":       beat.Duration.Value,
		"durationTicks":  beat.Duration.time(),
		"dots":           goDurationDots(beat.Duration),
		"tuplet":         []any{normalizedTuplet(beat.Duration.TupletEnters), normalizedTuplet(beat.Duration.TupletTimes)},
		"dynamic":        goDynamic(beat.Dynamics),
		"text":           beat.Text,
		"octave":         goOctave(beat.Octave),
		"hairpin":        goHairpin(beat.Effect.Hairpin),
		"tremoloPicking": goTremoloPicking(beat.Notes),
		"whammy":         normalizeGoBend(beat.Effect.TremoloBar),
		"notes":          notes,
	}
}

type goNoteLinkProjection struct {
	hammerOrigin bool
	slides       []SlideType
}

type goBeatLinkPosition struct {
	measure int
	beat    *Beat
}

func goNoteLinkProjections(staff *Staff) map[*Note]goNoteLinkProjection {
	result := make(map[*Note]goNoteLinkProjection)
	maxVoices := 0
	for measureIndex := range staff.Measures {
		maxVoices = max(maxVoices, len(staff.Measures[measureIndex].Voices))
	}
	for voiceIndex := 0; voiceIndex < maxVoices; voiceIndex++ {
		var positions []goBeatLinkPosition
		for measureIndex := range staff.Measures {
			measure := &staff.Measures[measureIndex]
			if voiceIndex >= len(measure.Voices) {
				continue
			}
			for beatIndex := range measure.Voices[voiceIndex].Beats {
				positions = append(positions, goBeatLinkPosition{measure: measureIndex, beat: &measure.Voices[voiceIndex].Beats[beatIndex]})
			}
		}
		for positionIndex, position := range positions {
			for noteIndex := range position.beat.Notes {
				note := &position.beat.Notes[noteIndex]
				projection := goNoteLinkProjection{slides: slices.Clone(note.Effect.Slides)}
				projection.hammerOrigin = note.Effect.Hammer && goHasHammerDestination(note, positions[positionIndex+1:], position.measure, len(staff.Strings))
				if !goHasNextNoteOnString(note, positions[positionIndex+1:], position.measure) {
					projection.slides = slices.DeleteFunc(projection.slides, func(slide SlideType) bool {
						return slide == SlideShiftSlideTo || slide == SlideLegatoSlideTo
					})
				}
				result[note] = projection
			}
		}
	}
	return result
}

func goHasNextNoteOnString(note *Note, positions []goBeatLinkPosition, sourceMeasure int) bool {
	for _, position := range positions {
		if position.measure > sourceMeasure+3 {
			break
		}
		if goBeatNoteOnString(position.beat, note.String) != nil {
			return true
		}
	}
	return false
}

func goHasHammerDestination(note *Note, positions []goBeatLinkPosition, sourceMeasure, stringCount int) bool {
	for _, position := range positions {
		if position.measure > sourceMeasure+3 {
			break
		}
		if goBeatNoteOnString(position.beat, note.String) != nil {
			return true
		}
		for sourceString := int(note.String) - 1; sourceString > 0; sourceString-- {
			candidate := goBeatNoteOnString(position.beat, int8(sourceString))
			if candidate != nil {
				if candidate.Effect.LeftHandTapped {
					return true
				}
				break
			}
		}
		for sourceString := int(note.String) + 1; sourceString <= stringCount; sourceString++ {
			candidate := goBeatNoteOnString(position.beat, int8(sourceString))
			if candidate != nil {
				if candidate.Effect.LeftHandTapped {
					return true
				}
				break
			}
		}
	}
	return false
}

func goBeatNoteOnString(beat *Beat, stringNumber int8) *Note {
	for noteIndex := range beat.Notes {
		if beat.Notes[noteIndex].String == stringNumber {
			return &beat.Notes[noteIndex]
		}
	}
	return nil
}

func normalizeGoNote(track *Track, staff *Staff, note *Note) any {
	return normalizeGoNoteWithLinks(track, staff, note, goNoteLinkProjection{slides: note.Effect.Slides, hammerOrigin: note.Effect.Hammer})
}

func normalizeGoNoteWithLinks(track *Track, staff *Staff, note *Note, links goNoteLinkProjection) any {
	fret := any(note.Value)
	stringNumber := note.String
	articulation := any(nil)
	midi := int(note.Value)
	if staff.PercussionTrack {
		fret = nil
		stringNumber = -1
		if note.HasPercussionArticulation {
			articulation = note.PercussionArticulation
			if note.PercussionArticulation >= 0 && note.PercussionArticulation < len(track.PercussionArticulations) {
				midi = track.PercussionArticulations[note.PercussionArticulation].OutputMIDINumber
			}
		}
	} else if note.String == 0 {
		fret = nil
	} else if note.String > 0 && int(note.String) <= len(staff.Strings) {
		midi += int(staff.Strings[note.String-1].Value) + int(track.CapoFret)
	}
	graces := make([]any, 0, len(note.Effect.Graces))
	for _, grace := range note.Effect.Graces {
		rawFret := grace.Fret
		if grace.RawFret != nil {
			rawFret = *grace.RawFret
		}
		graces = append(graces, map[string]any{
			"rawFret": rawFret,
			"dead":    grace.IsDead, "onBeat": grace.IsOnBeat, "dynamic": goDynamic(grace.Velocity),
			"transition": goGraceTransition(grace.Transition), "staffPercussion": staff.PercussionTrack,
		})
	}
	percussionInput := any(nil)
	if staff.PercussionTrack {
		percussionInput = note.Value
	}
	return map[string]any{
		"string": stringNumber, "fret": fret, "percussionArticulation": articulation, "percussionInput": percussionInput, "midi": midi,
		"kind": goNoteKind(note.Kind), "dynamic": goDynamic(note.Velocity), "tieOrigin": note.TieOrigin,
		"durationPercent": note.DurationPercent,
		"tieDestination":  note.Kind == NoteTypeTie,
		"effects": map[string]any{
			"accent": goAccent(note.Effect), "ghost": note.Effect.GhostNote, "hammerOrigin": links.hammerOrigin,
			"letRing": note.Effect.LetRing, "palmMute": note.Effect.PalmMute, "staccato": note.Effect.Staccato,
			"vibrato": goVibrato(note.Effect), "harmonic": normalizeGoHarmonic(note.Effect.Harmonic),
			"bend": normalizeGoBend(note.Effect.Bend), "trill": normalizeGoTrill(note.Effect.Trill),
			"slides": normalizeGoSlides(links.slides),
		},
		"graces": graces,
	}
}

func normalizeGoHarmonic(harmonic *HarmonicEffect) any {
	if harmonic == nil {
		return nil
	}
	fret := any(nil)
	if harmonic.FretFloat != nil {
		fret = *harmonic.FretFloat
	} else if harmonic.Fret != nil {
		fret = *harmonic.Fret
	}
	return map[string]any{"kind": goHarmonicKind(harmonic.Kind), "fret": fret}
}

func normalizeGoBend(bend *BendEffect) any {
	if bend == nil || len(bend.Points) == 0 {
		return nil
	}
	points := make([]any, 0, len(bend.Points))
	for _, point := range bend.Points {
		points = append(points, map[string]any{"position": point.Position, "value": point.Value})
	}
	return points
}

func normalizeGoTrill(trill *TrillEffect) any {
	if trill == nil {
		return nil
	}
	return map[string]any{"fret": trill.Fret, "duration": trill.Duration.Value}
}

func normalizeGoSlides(slides []SlideType) []any {
	result := make([]any, 0, len(slides))
	for _, slide := range slides {
		result = append(result, goSlide(slide))
	}
	return result
}

func normalizedTuplet(value uint8) uint8 {
	if value == 0 {
		return 1
	}
	return value
}

func goDurationDots(duration Duration) int {
	if duration.DoubleDotted {
		return 2
	}
	if duration.Dotted {
		return 1
	}
	return 0
}

func goTremoloPicking(notes []Note) any {
	for _, note := range notes {
		if note.Effect.TremoloPicking != nil {
			return note.Effect.TremoloPicking.Duration.Value
		}
	}
	return nil
}

func goDynamic(velocity int16) string {
	delta := velocity - MinVelocity
	if delta < 0 || delta%VelocityIncrement != 0 {
		return strconv.Itoa(int(velocity))
	}
	index := delta / VelocityIncrement
	names := []string{"ppp", "pp", "p", "mp", "mf", "f", "ff", "fff"}
	if index < 0 || int(index) >= len(names) {
		return strconv.Itoa(int(velocity))
	}
	return names[index]
}

func goBeatStatus(status BeatStatus) string {
	switch status {
	case BeatStatusEmpty:
		return "empty"
	case BeatStatusRest:
		return "rest"
	case BeatStatusNormal:
		return "normal"
	default:
		return fmt.Sprintf("unknown:%d", status)
	}
}

func goNoteKind(kind NoteType) string {
	switch kind {
	case NoteTypeDead:
		return "dead"
	case NoteTypeTie:
		return "tie"
	case NoteTypeRest:
		return "rest"
	case NoteTypeNormal:
		return "normal"
	default:
		return fmt.Sprintf("unknown:%d", kind)
	}
}

func goHairpin(hairpin Hairpin) string {
	switch hairpin {
	case HairpinCrescendo:
		return "crescendo"
	case HairpinDiminuendo:
		return "decrescendo"
	case HairpinNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", hairpin)
	}
}

func goTripletFeel(feel TripletFeel) string {
	switch feel {
	case TripletFeelEighth:
		return "triplet-8th"
	case TripletFeelSixteenth:
		return "triplet-16th"
	case TripletFeelDottedEighth:
		return "dotted-8th"
	case TripletFeelDottedSixteenth:
		return "dotted-16th"
	case TripletFeelScottishEighth:
		return "scottish-8th"
	case TripletFeelScottishSixteenth:
		return "scottish-16th"
	case TripletFeelNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", feel)
	}
}

func goClef(clef MeasureClef) string {
	switch clef {
	case MeasureClefBass:
		return "bass"
	case MeasureClefTenor:
		return "tenor"
	case MeasureClefAlto:
		return "alto"
	case MeasureClefTreble:
		return "treble"
	default:
		return fmt.Sprintf("unknown:%d", clef)
	}
}

func goOctave(octave Octave) string {
	switch octave {
	case OctaveNone:
		return "none"
	case OctaveOttava:
		return "8va"
	case OctaveQuindicesima:
		return "15ma"
	case OctaveOttavaBassa:
		return "8vb"
	case OctaveQuindicesimaBassa:
		return "15mb"
	default:
		return fmt.Sprintf("unknown:%d", octave)
	}
}

func TestGoClefNormalizationContract(t *testing.T) {
	tests := []struct {
		clef MeasureClef
		want string
	}{
		{MeasureClefTreble, "treble"},
		{MeasureClefBass, "bass"},
		{MeasureClefTenor, "tenor"},
		{MeasureClefAlto, "alto"},
	}
	for _, test := range tests {
		if got := goClef(test.clef); got != test.want {
			t.Errorf("goClef(%d) = %q, want %q", test.clef, got, test.want)
		}
	}
}

func goAccent(effect NoteEffect) string {
	switch effect.Accent {
	case NoteAccentNormal:
		return "normal"
	case NoteAccentHeavy:
		return "heavy"
	case NoteAccentTenuto:
		return "tenuto"
	}
	if effect.HeavyAccentuatedNote {
		return "heavy"
	}
	if effect.AccentuatedNote {
		return "normal"
	}
	return "none"
}

func goVibrato(effect NoteEffect) string {
	switch effect.VibratoStrength {
	case NoteVibratoSlight:
		return "slight"
	case NoteVibratoWide:
		return "wide"
	}
	if effect.Vibrato {
		return "slight"
	}
	return "none"
}

func goHarmonicKind(kind HarmonicType) string {
	switch kind {
	case HarmonicTypeNatural:
		return "natural"
	case HarmonicTypeArtificial:
		return "artificial"
	case HarmonicTypeTapped:
		return "tap"
	case HarmonicTypePinch:
		return "pinch"
	case HarmonicTypeSemi:
		return "semi"
	case HarmonicTypeFeedback:
		return "feedback"
	case 0:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", kind)
	}
}

func goGraceTransition(transition GraceEffectTransition) string {
	switch transition {
	case GraceEffectTransitionSlide:
		return "slide"
	case GraceEffectTransitionBend:
		return "bend"
	case GraceEffectTransitionHammer:
		return "hammer"
	case GraceEffectTransitionNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", transition)
	}
}

func goSlide(slide SlideType) string {
	switch slide {
	case SlideIntoFromAbove:
		return "into-from-above"
	case SlideIntoFromBelow:
		return "into-from-below"
	case SlideShiftSlideTo:
		return "shift"
	case SlideLegatoSlideTo:
		return "legato"
	case SlideOutDownwards:
		return "out-down"
	case SlideOutUpwards:
		return "out-up"
	case SlidePickSlideDown:
		return "pick-slide-down"
	case SlidePickSlideUp:
		return "pick-slide-up"
	case SlideNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", slide)
	}
}
