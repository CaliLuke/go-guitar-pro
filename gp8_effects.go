// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"strconv"
)

func gp8Whammy(bend *BendEffect) *gpifWhammy {
	return gp8ConvertWhammy(bend).whammy
}

type gp8WhammyConversion struct {
	whammy     *gpifWhammy
	omitted    bool
	normalized bool
}

func gp8ConvertWhammy(bend *BendEffect) gp8WhammyConversion {
	if bend == nil {
		return gp8WhammyConversion{}
	}
	points := simplifyNoteBendPoints(slices.Clone(bend.Points))
	if len(points) < 2 || len(points) > 4 {
		return gp8WhammyConversion{omitted: len(points) != 0}
	}
	origin := points[0]
	destination := points[len(points)-1]
	middle1, middle2 := origin, destination
	switch len(points) {
	case 4:
		middle1, middle2 = points[1], points[2]
	case 3:
		middle1, middle2 = points[1], points[1]
	case 2:
		middle1 = bendPointWithOffset((resolvedBendOffset(origin)+resolvedBendOffset(destination))/2, int8((int16(origin.Value)+int16(destination.Value))/2))
		middle2 = middle1
	}
	whammy := &gpifWhammy{
		OriginValue:       gp8WhammyValue(origin.Value),
		MiddleValue:       gp8WhammyValue(middle1.Value),
		DestinationValue:  gp8WhammyValue(destination.Value),
		OriginOffset:      gp8WhammyOffset(origin),
		MiddleOffset1:     gp8WhammyOffset(middle1),
		MiddleOffset2:     gp8WhammyOffset(middle2),
		DestinationOffset: gp8WhammyOffset(destination),
	}
	encoded := []BendPoint{cloneBendPoint(origin), cloneBendPoint(middle1), cloneBendPoint(middle2), cloneBendPoint(destination)}
	encoded[2].Value = middle1.Value
	for index := range encoded {
		encoded[index].Vibrato = false
	}

	return gp8WhammyConversion{
		whammy:     whammy,
		normalized: !sameBendPoints(simplifyCurvePoints(canonicalizeStandardWhammyPoints(encoded), true), simplifyCurvePoints(points, true)),
	}
}

func gp8WhammyOffset(point BendPoint) string {
	offset := resolvedBendOffset(point)
	text := strconv.FormatFloat(offset, 'f', 6, 64)
	value, _ := strconv.ParseFloat(text, 64)
	if value != offset {
		return strconv.FormatFloat(offset, 'f', -1, 64)
	}
	return text
}

func gp8WhammyValue(value int8) string {
	return strconv.FormatFloat(float64(value)*float64(GPBendSemitone), 'f', 6, 64)
}

type gp8BendConversion struct {
	properties []gpifProperty
	omitted    bool
	normalized bool
}

func gp8ConvertBend(bend *BendEffect) gp8BendConversion {
	if bend == nil {
		return gp8BendConversion{}
	}
	points := simplifyNoteBendPoints(slices.Clone(bend.Points))
	if len(points) < 2 || len(points) > 4 {
		return gp8BendConversion{omitted: len(points) != 0}
	}
	origin := points[0]
	destination := points[len(points)-1]
	var middle1, middle2 BendPoint
	switch len(points) {
	case 4:
		if !nonmonotonicBendControlRoles(points) && points[0].Value == points[1].Value && points[2].Value == points[3].Value {
			// GPIF keeps the destination value through the end of the note. Encode
			// an initial hold and release by ending the explicit curve where the
			// final value is reached.
			middle1 = points[1]
			middle2 = cloneBendPoint(points[1])
			destination = points[2]
		} else {
			middle1 = points[1]
			middle2 = points[2]
		}
	case 3:
		middle1 = points[1]
		if points[1].Value == points[2].Value {
			// A destination before the end denotes a bend followed by a hold.
			destination = cloneBendPoint(points[1])
			// The final hold is implicit after the destination. Do not create a
			// nonmonotonic role tuple from the redundant terminal hold point.
			middle2 = cloneBendPoint(points[1])
		} else {
			middle2 = cloneBendPoint(points[1])
		}
	default:
		middle1 = bendPointWithOffset((resolvedBendOffset(origin)+resolvedBendOffset(destination))/2, int8((int16(origin.Value)+int16(destination.Value))/2))
		middle2 = cloneBendPoint(middle1)
	}
	originOffset := gp8BendOffset(origin)
	middleOffset1 := gp8BendOffset(middle1)
	middleOffset2 := gp8BendOffset(middle2)
	destinationOffset := gp8BendOffset(destination)
	enable := ""
	properties := []gpifProperty{
		{Name: "Bended", Enable: &enable},
		{Name: "BendDestinationOffset", Float: destinationOffset.text},
		{Name: "BendDestinationValue", Float: gp8BendValue(destination.Value)},
		{Name: "BendMiddleOffset1", Float: middleOffset1.text},
		{Name: "BendMiddleOffset2", Float: middleOffset2.text},
		{Name: "BendMiddleValue", Float: gp8BendValue(middle1.Value)},
		{Name: "BendOriginOffset", Float: originOffset.text},
		{Name: "BendOriginValue", Float: gp8BendValue(origin.Value)},
	}
	encoded := gpifBendProperties{
		enabled:              true,
		originOffset:         originOffset.value,
		originValue:          origin.Value,
		middleOffset1:        middleOffset1.value,
		hasMiddleOffset1:     true,
		middleOffset2:        middleOffset2.value,
		hasMiddleOffset2:     true,
		middleValue:          middle1.Value,
		destinationOffset:    destinationOffset.value,
		hasDestinationOffset: true,
		destinationValue:     destination.Value,
	}
	return gp8BendConversion{
		properties: properties,
		normalized: !sameBendPoints(simplifyNoteBendPoints(encoded.effect().Points), simplifyNoteBendPoints(points)),
	}
}

type gp8BendOffsetEncoding struct {
	text  *string
	value float64
}

func gp8BendOffset(point BendPoint) gp8BendOffsetEncoding {
	text := strconv.FormatFloat(resolvedBendOffset(point), 'f', -1, 64)
	value, _ := strconv.ParseFloat(text, 64)
	return gp8BendOffsetEncoding{text: &text, value: value}
}

func gp8BendValue(value int8) *string {
	encoded := strconv.FormatFloat(float64(value)*float64(GPBendSemitone), 'f', 6, 64)
	return &encoded
}

func gp8NoteMIDI(track *Track, staffStrings []GuitarString, note *Note) int {
	midi := int(note.Value)
	if !track.PercussionTrack && note.String > 0 && int(note.String) <= len(staffStrings) {
		midi += int(staffStrings[int(note.String)-1].Value)
	}
	return midi
}

func gp8Pitch(midi int) gpifPitch {
	steps := [...]string{"C", "C", "D", "D", "E", "F", "F", "G", "G", "A", "A", "B"}
	accidental := ""
	switch midi % 12 {
	case 1, 3, 6, 8, 10:
		accidental = "#"
	}
	return gpifPitch{Step: steps[midi%12], Accidental: accidental, Octave: midi/12 - 1}
}

func gp8NoteValue(value uint16) (string, bool) {
	switch value {
	case 1:
		return "Whole", true
	case 2:
		return "Half", true
	case 4:
		return "Quarter", true
	case 8:
		return "Eighth", true
	case 16:
		return "16th", true
	case 32:
		return "32nd", true
	case 64:
		return "64th", true
	case 128:
		return "128th", true
	default:
		return "", false
	}
}

func gp8VelocityToDynamic(velocity int16) string {
	index := int((velocity-MinVelocity+VelocityIncrement/2)/VelocityIncrement) + 1
	switch min(8, max(1, index)) {
	case 1:
		return "PPP"
	case 2:
		return "PP"
	case 3:
		return "P"
	case 4:
		return "MP"
	case 5:
		return "MF"
	case 6:
		return "F"
	case 7:
		return "FF"
	default:
		return "FFF"
	}
}

func gp8ResolveNoteVibrato(effect NoteEffect) string {
	strength := effect.VibratoStrength
	if strength == NoteVibratoNone && effect.Vibrato {
		strength = NoteVibratoSlight
	}
	switch strength {
	case NoteVibratoSlight:
		return "Slight"
	case NoteVibratoWide:
		return "Wide"
	default:
		return ""
	}
}

func gp8ResolveNoteAccent(effect NoteEffect) (int, bool) {
	legacyFlags := 0
	if effect.HeavyAccentuatedNote {
		legacyFlags |= 0x04
	}
	if effect.AccentuatedNote {
		legacyFlags |= 0x08
	}
	if effect.Accent == NoteAccentNone {
		return legacyFlags, false
	}
	typedFlags := 0
	switch effect.Accent {
	case NoteAccentNormal:
		typedFlags = 0x08
	case NoteAccentHeavy:
		typedFlags = 0x04
	case NoteAccentTenuto:
		typedFlags = 0x10
	}
	return typedFlags, legacyFlags != 0 && legacyFlags != typedFlags
}

type gp8BeatDynamicConversion struct {
	marking                  string
	velocity                 int16
	authoredNormalized       bool
	noteVelocitiesNormalized bool
}

func gp8ConvertBeatDynamic(beat *Beat) gp8BeatDynamicConversion {
	source := beat.Dynamics
	if source == 0 && len(beat.Notes) > 0 {
		source = beat.Notes[0].Velocity
	}
	if source == 0 {
		return gp8BeatDynamicConversion{}
	}
	marking := gp8VelocityToDynamic(source)
	target := gpifDynamicToVelocity(marking)
	conversion := gp8BeatDynamicConversion{
		marking:            marking,
		velocity:           target,
		authoredNormalized: beat.Dynamics != 0 && beat.Dynamics != target,
	}
	conversion.noteVelocitiesNormalized = slices.ContainsFunc(beat.Notes, func(note Note) bool {
		return note.Velocity != conversion.velocity
	})
	return conversion
}
