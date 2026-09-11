// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func gpifApplyBeatEffects(b *gpifBeat, beat *Beat) {
	if b.Legato != nil {
		beat.Legato = &BeatLegato{
			Origin:      b.Legato.Origin == "true",
			Destination: b.Legato.Destination == "true",
		}
	}

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
			position, positionValid := gpifParseBendNumber(point.position, true)
			value, valueValid := gpifParseBendNumber(point.value, false)
			if !positionValid || !valueValid {
				continue
			}
			bend.Points = append(bend.Points, BendPoint{
				Position: uint8(math.Round(position * float64(BendEffectMaxPosition) / 100)),
				Value:    int8(math.Round(value / float64(GPBendSemitone))),
			})
		}
		if len(bend.Points) > 0 {
			bend.Points = canonicalizeStandardWhammyPoints(bend.Points)
			beat.Effect.TremoloBar = bend
		}
	} else if bend := gpifBeatWhammyProperties(b.Properties.Properties); bend != nil {
		beat.Effect.TremoloBar = bend
	}

	// Arpeggio / brush stroke
	switch b.Arpeggio {
	case "Up":
		beat.Effect.Stroke.Direction = BeatStrokeDirectionUp
		beat.Effect.Stroke.Duration = NoteValue(DurationEighth)
	case "Down":
		beat.Effect.Stroke.Direction = BeatStrokeDirectionDown
		beat.Effect.Stroke.Duration = NoteValue(DurationEighth)
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
					beat.Effect.Stroke.Duration = NoteValue(DurationEighth)
				case "Down":
					beat.Effect.Stroke.Direction = BeatStrokeDirectionDown
					beat.Effect.Stroke.Duration = NoteValue(DurationEighth)
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

// gpifBeatWhammyProperties decodes the named-property representation used by
// GP6. Later GPIF revisions use the Whammy element handled above instead.
func gpifBeatWhammyProperties(properties []gpifProperty) *BendEffect {
	enabled := false
	origin := BendPoint{}
	destination := BendPoint{Position: uint8(BendEffectMaxPosition)}
	var middleValue int8
	var middleOffset1, middleOffset2 uint8
	var middleValueSet, middleOffset1Set, middleOffset2Set bool

	for _, property := range properties {
		switch property.Name {
		case "WhammyBar":
			enabled = true
		case "WhammyBarOriginValue":
			origin.Value = gpifBendValue(property.Float)
		case "WhammyBarOriginOffset":
			origin.Position = gpifBendPosition(property.Float)
		case "WhammyBarMiddleValue":
			middleValue = gpifBendValue(property.Float)
			middleValueSet = property.Float != nil
		case "WhammyBarMiddleOffset1":
			middleOffset1 = gpifBendPosition(property.Float)
			middleOffset1Set = property.Float != nil
		case "WhammyBarMiddleOffset2":
			middleOffset2 = gpifBendPosition(property.Float)
			middleOffset2Set = property.Float != nil
		case "WhammyBarDestinationValue":
			destination.Value = gpifBendValue(property.Float)
		case "WhammyBarDestinationOffset":
			destination.Position = gpifBendPosition(property.Float)
		}
	}
	if !enabled {
		return nil
	}

	points := []BendPoint{origin}
	if middleOffset1Set && middleValueSet {
		points = append(points, BendPoint{Position: middleOffset1, Value: middleValue})
	}
	if middleOffset2Set && middleValueSet {
		points = append(points, BendPoint{Position: middleOffset2, Value: middleValue})
	}
	if !middleOffset1Set && !middleOffset2Set && middleValueSet {
		points = append(points, BendPoint{Position: uint8(BendEffectMaxPosition / 2), Value: middleValue})
	}
	points = append(points, destination)
	return &BendEffect{Points: canonicalizeStandardWhammyPoints(points)}
}

func gpifApplyTremoloPicking(value string, beat *Beat) {
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
	beat.Effect.TremoloPicking = cloneTremoloPickingEffect(&effect)
	for noteIndex := range beat.Notes {
		noteEffect := effect
		beat.Notes[noteIndex].Effect.TremoloPicking = &noteEffect
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

func gpifRhythmToDuration(r *gpifRhythm) (Duration, error) {
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
		switch r.AugmentationDot.Count {
		case 0:
		case 1:
			d.Dotted = true
		case 2:
			d.DoubleDotted = true
		default:
			return Duration{}, fmt.Errorf("augmentation-dot count %d is outside 0..2", r.AugmentationDot.Count)
		}
	}
	if r.PrimaryTuplet != nil {
		if r.PrimaryTuplet.Num <= 0 || r.PrimaryTuplet.Num > math.MaxUint8 || r.PrimaryTuplet.Den <= 0 || r.PrimaryTuplet.Den > math.MaxUint8 {
			return Duration{}, fmt.Errorf("tuplet ratio %d:%d is outside legacy boundary 1..255", r.PrimaryTuplet.Num, r.PrimaryTuplet.Den)
		}
		d.TupletEnters = uint8(r.PrimaryTuplet.Num)
		d.TupletTimes = uint8(r.PrimaryTuplet.Den)
	}
	return d, nil
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

func gpifNoteToNote(n *gpifNote, stringCount int, percussion bool) (Note, error) {
	note := defaultNote()
	note.Kind = NoteTypeNormal
	if percussion && n.InstrumentArticulation != nil && *n.InstrumentArticulation >= 0 {
		if _, err := NewPercussionArticulationID(int64(*n.InstrumentArticulation)); err != nil {
			return Note{}, err
		}
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
				// Numbered-notation GPIF can persist a negative derived fret beside
				// its usable absolute MIDI value. Let that MIDI property supply Value.
				if *p.Fret < 0 {
					continue
				}
				fret, err := NewFret(int64(*p.Fret))
				if err != nil {
					return Note{}, err
				}
				note.Value = int16(fret)
				hasFret = true
			}
		case "String":
			if p.String != nil {
				sourceString := int(*p.String)
				if !percussion && sourceString >= 0 && sourceString < stringCount {
					// GPIF counts from the lowest string while the parser model,
					// like GP3-5, counts from the highest string.
					modelString := stringCount - sourceString
					if modelString > math.MaxInt8 {
						return Note{}, fmt.Errorf("string number %d exceeds legacy int8 boundary", modelString)
					}
					note.String = int8(modelString)
				} else {
					if sourceString < -1 || sourceString >= math.MaxInt8 {
						return Note{}, fmt.Errorf("source string index %d exceeds legacy int8 boundary", sourceString)
					}
					note.String = int8(sourceString + 1)
				}
			}
		case "Midi":
			if p.Number != nil && !hasFret {
				midi, err := NewMIDINote(int64(*p.Number))
				if err != nil {
					return Note{}, err
				}
				note.Value = int16(midi)
			}
		case "Muted":
			note.Kind = NoteTypeDead
			note.Effect.DeadNote = true
		case "Bended":
			bend.enabled = true
		case "BendOriginOffset":
			bend.originOffset, _ = gpifBendOffset(p.Float)
		case "BendOriginValue":
			bend.originValue = gpifBendValue(p.Float)
		case "BendMiddleOffset1":
			bend.middleOffset1, bend.hasMiddleOffset1 = gpifBendOffset(p.Float)
		case "BendMiddleOffset2":
			bend.middleOffset2, bend.hasMiddleOffset2 = gpifBendOffset(p.Float)
		case "BendMiddleValue":
			bend.middleValue = gpifBendValue(p.Float)
		case "BendDestinationOffset":
			bend.destinationOffset, bend.hasDestinationOffset = gpifBendOffset(p.Float)
		case "BendDestinationValue":
			bend.destinationValue = gpifBendValue(p.Float)
		case "PalmMuted":
			note.Effect.PalmMute = true
		case "Tapped":
			note.Effect.Tapped = true
		case "HopoOrigin":
			note.Effect.Hammer = true
		case "LeftHandTapped":
			note.Effect.LeftHandTapped = true
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
					note.Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeFeedback}
				}
			}
		case "HarmonicFret":
			value := p.HFret
			if value == nil {
				value = p.Float
			}
			if parsed, valid := gpifHarmonicFret(value); valid {
				harmonicFret = parsed
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
					if flags&0x40 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlidePickSlideDown)
					}
					if flags&0x80 != 0 {
						note.Effect.Slides = append(note.Effect.Slides, SlidePickSlideUp)
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
		note.Effect.Accent = NoteAccentHeavy
	}
	if n.Accent&0x08 != 0 {
		note.Effect.AccentuatedNote = true
		note.Effect.Accent = NoteAccentNormal
	}
	if n.Accent&0x10 != 0 {
		note.Effect.Accent = NoteAccentTenuto
	}

	// Vibrato
	switch n.Vibrato {
	case "Slight":
		note.Effect.VibratoStrength = NoteVibratoSlight
		note.Effect.Vibrato = true
	case "Wide":
		note.Effect.VibratoStrength = NoteVibratoWide
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

	return note, nil
}

func gpifHarmonicFret(raw *string) (*float64, bool) {
	if raw == nil {
		return nil, false
	}
	value, err := strconv.ParseFloat(*raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > math.MaxInt8 {
		return nil, false
	}
	return &value, true
}

type gpifBendProperties struct {
	enabled              bool
	originOffset         float64
	originValue          int8
	middleOffset1        float64
	hasMiddleOffset1     bool
	middleOffset2        float64
	hasMiddleOffset2     bool
	middleValue          int8
	destinationOffset    float64
	hasDestinationOffset bool
	destinationValue     int8
}

func (bend gpifBendProperties) effect() *BendEffect {
	destinationOffset := bend.destinationOffset
	if !bend.hasDestinationOffset {
		destinationOffset = 100
	}
	points := []BendPoint{importedBendPoint(bend.originOffset, bend.originValue, false)}
	if bend.middleValue != 0 {
		switch {
		case !bend.hasMiddleOffset1 && !bend.hasMiddleOffset2:
			points = append(points, importedBendPoint(50, bend.middleValue, false))
		default:
			if bend.hasMiddleOffset1 {
				points = append(points, importedBendPoint(bend.middleOffset1, bend.middleValue, false))
			}
			if bend.hasMiddleOffset2 {
				points = append(points, importedBendPoint(bend.middleOffset2, bend.middleValue, false))
			}
		}
	}
	points = append(points, importedBendPoint(destinationOffset, bend.destinationValue, false))
	points = canonicalizeStandardBendPoints(points)
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
	return simplifyCurvePoints(points, false)
}

func simplifyNoteBendPoints(points []BendPoint) []BendPoint {
	return simplifyCurvePoints(points, true)
}

func simplifyCurvePoints(points []BendPoint, exactOffsets bool) []BendPoint {
	result := make([]BendPoint, 0, len(points))
	for _, point := range points {
		if len(result) > 0 && ((exactOffsets && sameBendPoint(result[len(result)-1], point)) || (!exactOffsets && result[len(result)-1] == point)) {
			continue
		}
		result = append(result, point)
	}
	for index := 1; index+1 < len(result); {
		left := result[index-1]
		middle := result[index]
		right := result[index+1]
		leftOffset := curvePointOffset(left, exactOffsets)
		middleOffset := curvePointOffset(middle, exactOffsets)
		rightOffset := curvePointOffset(right, exactOffsets)
		leftSpan := rightOffset - leftOffset
		if leftSpan > 0 && float64(int(middle.Value)-int(left.Value))*leftSpan == float64(int(right.Value)-int(left.Value))*(middleOffset-leftOffset) {
			result = append(result[:index], result[index+1:]...)
			continue
		}
		index++
	}
	return result
}

func curvePointOffset(point BendPoint, exact bool) float64 {
	if exact {
		return resolvedBendOffset(point)
	}
	return float64(point.Position)
}

func gpifBendPosition(value *string) uint8 {
	parsed, valid := gpifParseBendNumberPointer(value, true)
	if !valid {
		return 0
	}
	return uint8(math.Round(parsed * float64(BendEffectMaxPosition) / 100))
}

func gpifBendOffset(value *string) (float64, bool) {
	return gpifParseBendNumberPointer(value, true)
}

func gpifBendValue(value *string) int8 {
	parsed, valid := gpifParseBendNumberPointer(value, false)
	if !valid {
		return 0
	}
	return int8(math.Round(parsed / float64(GPBendSemitone)))
}

func gpifParseBendNumberPointer(value *string, offset bool) (float64, bool) {
	if value == nil {
		return 0, false
	}
	return gpifParseBendNumber(*value, offset)
}

func gpifParseBendNumber(value string, offset bool) (float64, bool) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	minimum, maximum := gpifBendNumberBounds(offset)
	if parsed < minimum || parsed > maximum {
		return 0, false
	}
	return parsed, true
}

func gpifBendNumberBounds(offset bool) (float64, float64) {
	if offset {
		return 0, 100
	}
	return float64(math.MinInt8) * float64(GPBendSemitone), float64(math.MaxInt8) * float64(GPBendSemitone)
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

func gpifSimileMark(value string) SimileMark {
	switch value {
	case "Simple":
		return SimileMarkSimple
	case "FirstOfDouble":
		return SimileMarkFirstOfDouble
	case "SecondOfDouble":
		return SimileMarkSecondOfDouble
	default:
		return SimileMarkNone
	}
}

func gp8SimileMark(value SimileMark) string {
	switch value {
	case SimileMarkSimple:
		return "Simple"
	case SimileMarkFirstOfDouble:
		return "FirstOfDouble"
	case SimileMarkSecondOfDouble:
		return "SecondOfDouble"
	default:
		return ""
	}
}
