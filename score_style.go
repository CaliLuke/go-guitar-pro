// SPDX-License-Identifier: MIT

package goguitarpro

// BarNumberPolicy selects the score-wide bar-number display policy.
type BarNumberPolicy uint8

const (
	// BarNumberAllBars displays a number on every bar.
	BarNumberAllBars BarNumberPolicy = iota
	// BarNumberFirstOfSystem displays numbers on the first bar of each system.
	BarNumberFirstOfSystem
	// BarNumberHide hides all bar numbers.
	BarNumberHide
)

// ScoreStyle owns authored score-wide display requests. Nil field pointers mean
// no authored setting. Direct edits control output; clearing a field removes its
// corresponding stylesheet record. Other imported records remain unchanged.
type ScoreStyle struct {
	// ExtendedBarLines controls extended barlines across the score.
	ExtendedBarLines *bool
	// BarNumbers controls score-wide numbering independently from individual bars.
	BarNumbers *BarNumberPolicy
	records    []binaryStyleRecord
}
