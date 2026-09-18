// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// GP6 identifies a kit piece and playing technique separately. These are
// builtin input identities, not output MIDI numbers or track-local indexes.
// Guitar Pro 8 converts GP6 kicks to input 36 (staff line 7).
// AlphaTab 1.8.4 instead imports this element as input 35.
var gp6PercussionInputs = [17][3]int16{
	{36, 36, 36}, {38, 91, 37}, {99, 36, 36}, {56, 36, 36}, {102, 36, 36},
	{43, 36, 36}, {45, 36, 36}, {47, 36, 36}, {48, 36, 36}, {50, 36, 36},
	{42, 92, 46}, {44, 36, 36}, {57, 36, 36}, {49, 36, 36}, {55, 36, 36},
	{51, 93, 53}, {52, 36, 36},
}

func gpifApplyPercussionElement(properties []gpifProperty, note *Note) error {
	var element, variation *int
	present := false
	for _, property := range properties {
		if property.Name == "Element" {
			element = property.Element
			present = true
		}
		if property.Name == "Variation" {
			variation = property.Variation
			present = true
		}
	}
	if !present {
		return nil
	}
	if element == nil || variation == nil {
		return fmt.Errorf("GP6 percussion requires both Element and Variation")
	}
	if *element < 0 || *element >= len(gp6PercussionInputs) || *variation < 0 || *variation >= len(gp6PercussionInputs[0]) {
		return fmt.Errorf("unsupported GP6 percussion element %d variation %d", *element, *variation)
	}
	note.Value = gp6PercussionInputs[*element][*variation]
	note.PercussionArticulation = int(note.Value)
	note.HasPercussionArticulation = true
	return nil
}
