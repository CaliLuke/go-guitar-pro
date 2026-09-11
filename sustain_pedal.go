// SPDX-License-Identifier: MIT

package goguitarpro

// SustainPedalType identifies the action at a sustain-pedal marker.
type SustainPedalType uint8

const (
	// SustainPedalTypeDown presses the sustain pedal.
	SustainPedalTypeDown SustainPedalType = iota
	// SustainPedalTypeHold continues a pressed pedal through a bar with no authored marker.
	SustainPedalTypeHold
	// SustainPedalTypeRelease releases the sustain pedal.
	SustainPedalTypeRelease
)

// SustainPedalMarker describes one measure-relative sustain-pedal action.
type SustainPedalMarker struct {
	Type     SustainPedalType
	Position float64
}
