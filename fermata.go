// SPDX-License-Identifier: MIT

package goguitarpro

// FermataType identifies the authored fermata symbol.
type FermataType uint8

const (
	// FermataTypeShort is the triangular short fermata.
	FermataTypeShort FermataType = iota
	// FermataTypeMedium is the round medium fermata.
	FermataTypeMedium
	// FermataTypeLong is the rectangular long fermata.
	FermataTypeLong
)

// Fermata is an authored hold at an exact offset within a measure header.
// Offset is measured in 960-PPQ score ticks from the start of the measure.
// Length preserves GPIF's non-negative duration multiplier.
type Fermata struct {
	Offset ScoreTime
	Type   FermataType
	Length float64
}
