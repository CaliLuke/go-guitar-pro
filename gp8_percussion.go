// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"strconv"
	"strings"
)

func gp8DrumStaffLine(value int16) int {
	switch value {
	case 35:
		return 8
	case 36:
		return 7
	case 37, 38, 40:
		return 3
	case 41:
		return 5
	case 43:
		return 6
	case 45:
		return 5
	case 47:
		return 4
	case 48:
		return 2
	case 50:
		return 1
	case 42, 44, 46:
		return -1
	case 52:
		return -3
	case 55:
		return -2
	case 57:
		return -1
	case 92:
		return -1
	case 99:
		return 1
	case 102:
		return -1
	default:
		return 0
	}
}

func gp8PercussionArticulationIndex(track *Track, note *Note) (int, bool) {
	if note.HasPercussionArticulation {
		return note.PercussionArticulation, note.PercussionArticulation >= 0 && note.PercussionArticulation < len(track.PercussionArticulations)
	}
	for index, articulation := range track.PercussionArticulations {
		for _, input := range articulation.InputMIDINumbers {
			if input == int(note.Value) {
				return index, true
			}
		}
	}
	return 0, false
}

func gp8PercussionElements(track *Track, options GP8ExportOptions) []gpifElement {
	elements := make([]gpifElement, 0)
	for _, source := range track.PercussionArticulations {
		placement := source.TechniquePlacement
		if placement == "" {
			placement = "outside"
		}
		defaultHead := source.NoteheadDefault
		halfHead := source.NoteheadHalf
		if halfHead == "" {
			halfHead = defaultHead
		}
		wholeHead := source.NoteheadWhole
		if wholeHead == "" {
			wholeHead = defaultHead
		}
		noteheads := strings.TrimSpace(strings.Join([]string{defaultHead, halfHead, wholeHead}, " "))
		for _, input := range source.InputMIDINumbers {
			if override := options.PercussionNoteheads[int16(input)]; override != GP8PercussionNoteheadDefault {
				noteheads = gp8PercussionNoteheads(override)
				break
			}
		}
		inputs := make([]string, 0, len(source.InputMIDINumbers))
		for _, input := range source.InputMIDINumbers {
			inputs = append(inputs, strconv.Itoa(input))
		}
		articulation := gpifArticulation{
			Name:               source.Name,
			StaffLine:          source.StaffLine,
			Noteheads:          noteheads,
			TechniquePlacement: placement,
			TechniqueSymbol:    source.TechniqueSymbol,
			InputMIDINumbers:   strings.Join(inputs, " "),
			OutputRSESound:     source.OutputRSESound,
			OutputMIDINumber:   source.OutputMIDINumber,
		}
		if len(elements) == 0 || elements[len(elements)-1].Name != source.ElementName || elements[len(elements)-1].Type != source.ElementType || elements[len(elements)-1].SoundbankName != source.ElementSoundbankName {
			elements = append(elements, gpifElement{Name: source.ElementName, Type: source.ElementType, SoundbankName: source.ElementSoundbankName})
		}
		last := &elements[len(elements)-1]
		last.Articulations.Articulations = append(last.Articulations.Articulations, articulation)
	}

	values := make([]int16, 0)
	seen := make(map[int16]struct{})
	addFallback := func(value int16) {
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, staff := range gp8ExportStaves(track) {
		for measureIndex := range staff.Measures {
			for voiceIndex := range staff.Measures[measureIndex].Voices {
				for beatIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
					for noteIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						note := &staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex]
						if _, ok := gp8PercussionArticulationIndex(track, note); !ok {
							addFallback(note.Value)
						}
						for graceIndex := range note.Effect.Graces {
							graceNote := gp8PercussionGraceNote(note, &note.Effect.Graces[graceIndex])
							if _, ok := gp8PercussionArticulationIndex(track, &graceNote); !ok {
								addFallback(graceNote.Value)
							}
						}
					}
				}
			}
		}
	}
	slices.Sort(values)
	return append(elements, gp8DrumElements(values, options)...)
}

func gp8PercussionGraceNote(parent *Note, grace *GraceEffect) Note {
	value := int16(grace.Fret)
	if grace.ExactFret != nil {
		value = int16(*grace.ExactFret)
	} else if grace.Fret == 0 {
		value = parent.Value
	}
	return Note{
		Value:                     value,
		HasPercussionArticulation: grace.HasPercussionArticulation,
		PercussionArticulation:    grace.PercussionArticulation,
	}
}

func gp8DrumElements(values []int16, options GP8ExportOptions) []gpifElement {
	elements := make([]gpifElement, 0, len(values))
	type elementKey struct {
		name          string
		kind          string
		soundbankName string
	}
	elementIndices := make(map[elementKey]int, len(values))
	for _, value := range values {
		element := gp8DrumElement(value, options)
		key := elementKey{name: element.Name, kind: element.Type, soundbankName: element.SoundbankName}
		if index, exists := elementIndices[key]; exists {
			existing := &elements[index]
			existing.Articulations.Articulations = append(existing.Articulations.Articulations, element.Articulations.Articulations...)
			continue
		}
		elements = append(elements, element)
		elementIndices[key] = len(elements) - 1
	}
	return elements
}

func gp8DrumArticulationIDs(elements []gpifElement) map[int16]int {
	ids := make(map[int16]int)
	index := 0
	for _, element := range elements {
		for _, articulation := range element.Articulations.Articulations {
			for _, input := range strings.Fields(articulation.InputMIDINumbers) {
				midi, err := strconv.ParseInt(input, 10, 16)
				if err == nil {
					ids[int16(midi)] = index
				}
			}
			index++
		}
	}
	return ids
}

func gp8DrumElement(value int16, options GP8ExportOptions) gpifElement {
	midi := strconv.Itoa(int(value))
	element := gpifElement{
		Name: "MIDI " + midi,
		Type: "percussion",
	}
	articulation := gpifArticulation{
		Name:               element.Name,
		StaffLine:          gp8DrumStaffLine(value),
		Noteheads:          "noteheadBlack noteheadHalf noteheadWhole",
		TechniquePlacement: "outside",
		InputMIDINumbers:   midi,
		OutputMIDINumber:   int(value),
	}
	setNative := func(elementName, kind, soundbank, articulationName string, staffLine int, noteheads, rseSound string, outputMIDI int) {
		element.Name = elementName
		element.Type = kind
		element.SoundbankName = soundbank
		articulation.Name = articulationName
		articulation.StaffLine = staffLine
		articulation.Noteheads = noteheads
		articulation.OutputRSESound = rseSound
		articulation.OutputMIDINumber = outputMIDI
	}
	black := "noteheadBlack noteheadHalf noteheadWhole"
	x := "noteheadXBlack noteheadXBlack noteheadXBlack"
	triangle := "noteheadTriangleUpBlack noteheadTriangleUpBlack noteheadTriangleUpBlack"
	switch value {
	case 29:
		setNative("Ride Cymbal 2", "ride", "Ride-Percu", "Ride (choke)", 2, x, "stick.hit.choke", 59)
		articulation.TechniqueSymbol = "articStaccatoAbove"
	case 30:
		setNative("Reverse Cymbal", "crash", "Reverse-Cymbal", "Reverse Cymbal (hit)", -3, x, "stick.hit.hit", 49)
	case 31:
		setNative("Sticks", "snare", "Stick-Percu", "Snare (side stick)", 3, "noteheadSlashedBlack2 noteheadSlashedBlack2 noteheadSlashedBlack2", "stick.hit.sidestick", 40)
	case 33:
		setNative("Metronome", "snare", "Metronome-Percu", "Metronome (hit)", 3, x, "stick.hit.sidestick", 37)
	case 34:
		setNative("Metronome", "snare", "Metronome-Percu", "Metronome (bell)", 3, "noteheadBlack noteheadBlack noteheadBlack", "stick.hit.hit", 38)
	case 35:
		element.Name = "Acoustic Kick Drum"
		element.Type = "kickDrum"
		element.SoundbankName = "AcousticKick-Percu"
		articulation.Name = "Kick (hit)"
		articulation.OutputRSESound = "pedal.hit.hit"
	case 36:
		element.Name = "Kick Drum"
		element.Type = "kickDrum"
		element.SoundbankName = "Master-Kick"
		articulation.Name = "Kick (hit)"
		articulation.OutputRSESound = "pedal.hit.hit"
	case 37:
		element.Name = "Snare"
		element.Type = "snare"
		element.SoundbankName = "Master-Snare"
		articulation.Name = "Snare (side stick)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.sidestick"
	case 38:
		element.Name = "Snare"
		element.Type = "snare"
		element.SoundbankName = "Master-Snare"
		articulation.Name = "Snare (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 39:
		setNative("Hand Clap", "handClap", "GroupHandClap-Percu", "Hand Clap (hit)", 3, black, "hand.hit.hit", 39)
	case 40:
		setNative("Electric Snare", "snare", "ElectricSnare-Percu", "Electric Snare (hit)", 3, black, "stick.hit.hit", 40)
	case 41:
		element.Name = "Very Low Floor Tom"
		element.Type = "tom"
		element.SoundbankName = "LowFloorTom-Percu"
		articulation.Name = "Low Floor Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 42:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (closed)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.closed"
	case 44:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Pedal Hi-Hat (hit)"
		articulation.StaffLine = 9
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "pedal.hit.pedal"
	case 45:
		element.Name = "Tom Low"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom02"
		articulation.Name = "Low Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 46:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (open)"
		articulation.Noteheads = "noteheadCircleX noteheadCircleX noteheadCircleX"
		articulation.OutputRSESound = "stick.hit.open"
	case 43:
		element.Name = "Tom Very Low"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom01"
		articulation.Name = "Very Low Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 47:
		element.Name = "Tom Medium"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom03"
		articulation.Name = "Mid Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 48:
		element.Name = "Tom High"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom04"
		articulation.Name = "High Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 49:
		element.Name = "Crash High"
		element.Type = "crash"
		element.SoundbankName = "Master-Crash02"
		articulation.Name = "Crash high (hit)"
		articulation.StaffLine = -2
		articulation.Noteheads = "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
		articulation.OutputRSESound = "stick.hit.hit"
	case 50:
		element.Name = "Tom Very High"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom05"
		articulation.Name = "High Floor Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 51:
		element.Name = "Ride"
		element.Type = "ride"
		element.SoundbankName = "Master-Ride"
		articulation.Name = "Ride (middle)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.mid"
	case 52:
		element.Name = "China"
		element.Type = "china"
		element.SoundbankName = "Master-China"
		articulation.Name = "China (hit)"
		articulation.Noteheads = "noteheadHeavyXHat noteheadHeavyXHat noteheadHeavyXHat"
		articulation.OutputRSESound = "stick.hit.hit"
	case 53:
		element.Name = "Ride"
		element.Type = "ride"
		element.SoundbankName = "Master-Ride"
		articulation.Name = "Ride (bell)"
		articulation.Noteheads = "noteheadDiamondWhite noteheadDiamondWhite noteheadDiamondWhite"
		articulation.OutputRSESound = "stick.hit.bell"
	case 54:
		setNative("Tambourine", "tambourine", "Tambourine-Percu", "Tambourine (hit)", 3, triangle, "hand.hit.hit", 54)
	case 55:
		element.Name = "Splash"
		element.Type = "splash"
		element.SoundbankName = "Master-Splash"
		articulation.Name = "Splash (hit)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.hit"
	case 56:
		setNative("Cowbell Medium", "cowbell", "CowbellMid-Percu", "Cowbell medium (hit)", 0, "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole", "stick.hit.hit", 56)
	case 57:
		element.Name = "Crash Medium"
		element.Type = "crash"
		element.SoundbankName = "Master-Crash01"
		articulation.Name = "Crash medium (hit)"
		articulation.StaffLine = -1
		articulation.Noteheads = "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
		articulation.OutputRSESound = "stick.hit.hit"
	case 58:
		setNative("Vibraslap", "vibraslap", "Vibraslap-Percu", "Vibraslap (hit)", 28, black, "hand.hit.hit", 58)
	case 59:
		setNative("Ride Cymbal 2", "ride", "Ride-Percu", "Ride (edge)", 2, x, "stick.hit.edge", 59)
		articulation.TechniquePlacement = "above"
		articulation.TechniqueSymbol = "pictEdgeOfCymbal"
	case 60:
		setNative("Bongo High", "bongo", "BongoHigh-Percu", "Bongo High (hit)", -4, black, "hand.hit.hit", 60)
	case 61:
		setNative("Bongo Low", "bongo", "BongoLow-Percu", "Bongo Low (hit)", -7, black, "hand.hit.hit", 61)
	case 62:
		setNative("Conga High", "conga", "CongaHigh-Percu", "Conga high (mute)", 19, black, "hand.hit.mute", 62)
		articulation.TechniquePlacement = "inside"
		articulation.TechniqueSymbol = "noteheadParenthesis"
	case 63:
		setNative("Conga High", "conga", "CongaHigh-Percu", "Conga high (hit)", 14, black, "hand.hit.hit", 63)
	case 64:
		setNative("Conga Low", "conga", "CongaLow-Percu", "Conga low (hit)", 17, black, "hand.hit.hit", 64)
	case 65:
		setNative("Timbale High", "timbale", "TimbaleHigh-Percu", "Timbale high (hit)", 9, black, "stick.hit.hit", 65)
	case 66:
		setNative("Timbale Low", "timbale", "TimbaleLow-Percu", "Timbale low (hit)", 10, black, "stick.hit.hit", 66)
	case 67:
		setNative("Agogo High", "agogo", "AgogoHigh-Percu", "Agogo high (hit)", 11, black, "stick.hit.hit", 67)
	case 68:
		setNative("Agogo Low", "agogo", "AgogoLow-Percu", "Agogo low (hit)", 12, black, "stick.hit.hit", 68)
	case 69:
		setNative("Cabasa", "cabasa", "Cabasa-Percu", "Cabasa (hit)", 23, black, "hand.hit.hit", 69)
	case 70:
		setNative("Left Maraca", "maraca", "Maracas-Percu", "Left Maraca (hit)", -12, black, "hand.hit.hit", 70)
	case 71:
		setNative("Whistle High", "whistle", "WhistleHigh-Percu", "Whistle high (hit)", -17, black, "blow.hit.hit", 71)
	case 72:
		setNative("Whistle Low", "whistle", "WhistleLow-Percu", "Whistle low (hit)", -11, black, "blow.hit.hit", 72)
	case 73:
		setNative("Guiro", "guiro", "Guiro-Percu", "Guiro (hit)", 38, black, "stick.hit.hit", 73)
	case 74:
		setNative("Guiro", "guiro", "Guiro-Percu", "Guiro (scrap-return)", 37, black, "stick.scrape.return", 74)
	case 75:
		setNative("Claves", "claves", "Claves-Percu", "Claves (hit)", 20, black, "stick.hit.hit", 75)
	case 76:
		setNative("Woodblock High", "woodblock", "WoodblockHigh-Percu", "Woodblock high (hit)", -10, triangle, "stick.hit.hit", 76)
	case 77:
		setNative("Woodblock Low", "woodblock", "WoodblockLow-Percu", "Woodblock low (hit)", -9, triangle, "stick.hit.hit", 77)
	case 78:
		setNative("Cuica", "cuica", "Cuica-Percu", "Cuica (mute)", 29, x, "hand.hit.mute", 78)
	case 79:
		setNative("Cuica", "cuica", "Cuica-Percu", "Cuica (open)", 30, black, "hand.hit.hit", 79)
	case 80:
		setNative("Triangle", "triangle", "Triangle-Percu", "Triangle (mute)", 26, x, "stick.hit.mute", 80)
		articulation.TechniquePlacement = "inside"
		articulation.TechniqueSymbol = "noteheadParenthesis"
	case 81:
		setNative("Triangle", "triangle", "Triangle-Percu", "Triangle (hit)", 27, black, "stick.hit.hit", 81)
	case 82:
		setNative("Shaker", "shaker", "ShakerStudio-Percu", "Shaker (hit)", -23, black, "hand.hit.hit", 82)
	case 83:
		setNative("Tinkle Bell", "jingleBell", "JingleBell-Percu", "Tinkle Bell (hit)", -20, black, "stick.hit.hit", 53)
	case 84:
		setNative("Bell Tree", "bellTree", "BellTree-Percu", "Bell Tree (hit)", -18, black, "stick.hit.hit", 53)
	case 85:
		setNative("Castanets", "castanets", "Castanets-Percu", "Castanets (hit)", 21, black, "hand.hit.hit", 85)
	case 86:
		setNative("Surdo", "surdo", "Surdo-Percu", "Surdo (hit)", 36, black, "brush.hit.hit", 86)
	case 87:
		setNative("Surdo", "surdo", "Surdo-Percu", "Surdo (mute)", 35, x, "brush.hit.mute", 87)
		articulation.TechniquePlacement = "inside"
		articulation.TechniqueSymbol = "noteheadParenthesis"
	case 92:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (half)"
		articulation.Noteheads = "noteheadCircleSlash noteheadCircleSlash noteheadCircleSlash"
		articulation.OutputMIDINumber = 46
		articulation.OutputRSESound = "stick.hit.half"
	case 99:
		element.Name = "Cowbell Low"
		element.Type = "cowbell"
		element.SoundbankName = "CowbellBig-Percu"
		articulation.Name = "Cowbell low (hit)"
		articulation.StaffLine = 1
		articulation.Noteheads = "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole"
		articulation.OutputMIDINumber = 56
		articulation.OutputRSESound = "stick.hit.hit"
	case 102:
		element.Name = "Cowbell High"
		element.Type = "cowbell"
		element.SoundbankName = "CowbellSmall-Percu"
		articulation.Name = "Cowbell high (hit)"
		articulation.StaffLine = -1
		articulation.Noteheads = "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole"
		articulation.OutputMIDINumber = 56
		articulation.OutputRSESound = "stick.hit.hit"
	}
	if notehead := options.PercussionNoteheads[value]; notehead != GP8PercussionNoteheadDefault {
		articulation.Noteheads = gp8PercussionNoteheads(notehead)
	}
	element.Articulations.Articulations = []gpifArticulation{articulation}
	return element
}

func gp8PercussionNoteheads(notehead GP8PercussionNotehead) string {
	switch notehead {
	case GP8PercussionNoteheadFilled:
		return "noteheadBlack noteheadHalf noteheadWhole"
	case GP8PercussionNoteheadX:
		return "noteheadXBlack noteheadXBlack noteheadXBlack"
	case GP8PercussionNoteheadCircleX:
		return "noteheadCircleX noteheadCircleX noteheadCircleX"
	case GP8PercussionNoteheadHeavyX:
		return "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
	default:
		return ""
	}
}

func sequentialIDs(count int) string {
	ids := make([]string, count)
	for index := range count {
		ids[index] = strconv.Itoa(index)
	}
	return strings.Join(ids, " ")
}
