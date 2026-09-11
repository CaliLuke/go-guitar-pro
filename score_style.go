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
// corresponding configuration record. Other imported records remain unchanged.
type ScoreStyle struct {
	// Brackets selects score-wide brackets and braces.
	Brackets *BracketMode
	// SystemSeparators requests separators between systems.
	SystemSeparators *bool
	// HideDynamics hides dynamic markings independently of their values.
	HideDynamics *bool
	// DisplayTuning requests the global tuning display.
	DisplayTuning *bool
	// ChordDiagramsOnTop requests chord diagrams above the score.
	ChordDiagramsOnTop *bool
	// ChordDiagramsInScore requests chord diagrams within the score.
	ChordDiagramsInScore *bool
	// SingleTrackNamesVisible controls names in a single-track view; false overrides its system mode.
	SingleTrackNamesVisible *bool
	// MultiTrackNamesVisible controls names in a multi-track view; false overrides its system mode.
	MultiTrackNamesVisible *bool
	// SingleTrackNameMode selects named systems in a single-track view.
	SingleTrackNameMode *TrackNameSystemMode
	// MultiTrackNameMode selects named systems in a multi-track view.
	MultiTrackNameMode *TrackNameSystemMode
	// FirstSystemShortNames selects short rather than full names on the first system.
	FirstSystemShortNames *bool
	// OtherSystemsShortNames selects short rather than full names on later systems.
	OtherSystemsShortNames *bool
	// FirstSystemHorizontalNames selects horizontal rather than vertical names on the first system.
	FirstSystemHorizontalNames *bool
	// OtherSystemsHorizontalNames selects horizontal rather than vertical names on later systems.
	OtherSystemsHorizontalNames *bool

	// HeaderFooter owns template and visibility requests. Changed parsed PageSetup
	// leaves override only their matching template or visibility bit on export.
	HeaderFooter *HeaderFooterSettings
	// MultiRest requests combined rests in the multi-track score view. Nil uses
	// false. This preference does not control any single-track view.
	MultiRest *bool
	// ExtendedBarLines controls extended barlines across the score.
	ExtendedBarLines *bool
	// BarNumbers controls score-wide numbering independently from individual bars.
	BarNumbers *BarNumberPolicy
	records    []binaryStyleRecord
}
