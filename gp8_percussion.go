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
							grace := &note.Effect.Graces[graceIndex]
							graceNote := Note{Value: int16(grace.Fret), HasPercussionArticulation: grace.HasPercussionArticulation, PercussionArticulation: grace.PercussionArticulation}
							if grace.ExactFret != nil {
								graceNote.Value = int16(*grace.ExactFret)
							}
							if graceNote.Value < 27 || graceNote.Value > 87 {
								graceNote.Value = note.Value
							}
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
	switch value {
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
	case 55:
		element.Name = "Splash"
		element.Type = "splash"
		element.SoundbankName = "Master-Splash"
		articulation.Name = "Splash (hit)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.hit"
	case 57:
		element.Name = "Crash Medium"
		element.Type = "crash"
		element.SoundbankName = "Master-Crash01"
		articulation.Name = "Crash medium (hit)"
		articulation.StaffLine = -1
		articulation.Noteheads = "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
		articulation.OutputRSESound = "stick.hit.hit"
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
