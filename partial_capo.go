// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"strings"
)

// PartialCapo describes a capo on selected strings. It changes open notes only;
// nonzero frets retain their ordinary pitch, including frets below the capo.
type PartialCapo struct {
	// Offset is the non-negative fret offset above the staff's whole capo.
	// Guitar Pro displays the absolute position as CapoFret + Offset.
	Offset int32
	// Strings selects strings in public tuning order, highest string first.
	// Its length must equal Staff.Strings. Each parsed staff owns its slice.
	Strings []bool
}

func gpifReadPartialCapo(properties []gpifStaffProperty, stringCount int) (*PartialCapo, error) {
	var fret *int
	var bitset *string
	presentFret, presentFlags := false, false
	for _, property := range properties {
		switch property.Name {
		case "PartialCapoFret":
			if presentFret {
				return nil, fmt.Errorf("duplicate partial capo fret")
			}
			presentFret, fret = true, property.Fret
		case "PartialCapoStringFlags":
			if presentFlags {
				return nil, fmt.Errorf("duplicate partial capo string flags")
			}
			presentFlags, bitset = true, property.Bitset
			if property.Flags != nil {
				if bitset != nil {
					return nil, fmt.Errorf("partial capo has conflicting Flags and Bitset payloads")
				}
				if strings.TrimSpace(*property.Flags) != "0" {
					return nil, fmt.Errorf("nonzero legacy partial capo Flags %q has no verified string-order mapping", *property.Flags)
				}
				value := strings.Repeat("0", stringCount)
				bitset = &value
			}
		}
	}
	if !presentFret && !presentFlags {
		return nil, nil
	}
	if !presentFret || !presentFlags || fret == nil || bitset == nil {
		return nil, fmt.Errorf("partial capo requires both Fret and Bitset payloads")
	}
	if *fret < 0 || *fret > math.MaxInt32 {
		return nil, fmt.Errorf("partial capo offset %d is outside 0..%d", *fret, math.MaxInt32)
	}
	if len(*bitset) != stringCount {
		// Guitar Pro can leave its six-string inactive placeholder on other tunings.
		if *fret == 0 && strings.Trim(*bitset, "0") == "" {
			return &PartialCapo{Strings: make([]bool, stringCount)}, nil
		}
		return nil, fmt.Errorf("partial capo has %d flags for %d strings", len(*bitset), stringCount)
	}
	result := &PartialCapo{Offset: int32(*fret), Strings: make([]bool, stringCount)}
	for index, value := range []byte(*bitset) {
		if value != '0' && value != '1' {
			return nil, fmt.Errorf("partial capo flag %d is not 0 or 1", index)
		}
		result.Strings[stringCount-index-1] = value == '1'
	}
	return result, nil
}

func partialCapoOffset(staff *Staff, note *Note) int64 {
	partial := staff.PartialCapo
	if partial == nil || note.Value != 0 || note.String <= 0 || int(note.String) > len(partial.Strings) || !partial.Strings[note.String-1] {
		return 0
	}
	return int64(partial.Offset)
}

func gpifReadPartialCapoWithContext(properties []gpifStaffProperty, stringCount int, context *parseContext, trackID string, staffIndex int) (*PartialCapo, error) {
	partial, err := gpifReadPartialCapo(properties, stringCount)
	if err == nil && partial != nil {
		for _, property := range properties {
			if property.Name == "PartialCapoStringFlags" && property.Bitset != nil && len(*property.Bitset) != stringCount {
				base := gpifObjectPath("Tracks/Track", trackID)
				if staffIndex >= 0 {
					base += fmt.Sprintf("/Staves/Staff[%d]", staffIndex)
				}
				context.add(diagnosticSource("GPIF.Staff.PartialCapo.InactiveFlags.Normalized", "staff-ownership", ParseDiagnosticLossyProjection), ParseDiagnostic{SourcePath: base + "/Properties/Property[@name=\"PartialCapoStringFlags\"]/Bitset", ObjectID: trackID, Location: ParseLocation{TrackID: trackID}, Reason: fmt.Sprintf("inactive zero-offset partial capo has %d zero flags for %d strings; selection normalized to the staff tuning", len(*property.Bitset), stringCount)})
			}
		}
	}
	return partial, err
}
