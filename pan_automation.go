// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// PanAutomation is an authored track pan event, separate from initial MIDI balance.
// The slice in Song is authoritative after import, including edits and clearing.
// Events must be chronological within each track; equal positions keep their order.
type PanAutomation struct {
	// Track is the zero-based owning track index.
	Track int
	// Bar is the zero-based master-bar index.
	Bar int
	// Position is the fraction of the bar in the inclusive range 0 through 1.
	Position float64
	// Value is normalized pan: 0 is left, 0.5 is center, and 1 is right.
	// GPIF DSPParam_11 uses these units; legacy mix-table balance is divided by 16.
	Value float64
	// Linear means interpolation from the preceding point to this point is linear.
	Linear bool
}

func gpifReadPanAutomations(automations []gpifAutomation, track int, song *Song) {
	for _, a := range automations {
		if a.Type != "DSPParam_11" {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(a.Value.Text), 64)
		if err != nil {
			continue
		}
		song.PanAutomations = append(song.PanAutomations, PanAutomation{Track: track, Bar: a.Bar, Position: a.Position, Value: value, Linear: a.Linear})
	}
}

func gpifAuditPanAutomation(a gpifAutomation, path, trackID string, context *parseContext) {
	if _, err := strconv.ParseFloat(strings.TrimSpace(a.Value.Text), 64); err != nil {
		context.add(diagnosticSource("GPIF.ChannelStrip.Automation.Pan.Value.Invalid", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{SourcePath: path + "/Value", ObjectID: trackID, Reason: fmt.Sprintf("pan automation value %q is not a number", a.Value.Text)})
	}
}

func panAutomationUnitValuesValid(a PanAutomation) bool {
	for _, value := range []float64{a.Position, a.Value} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			return false
		}
	}
	return true
}

func validatePanAutomations(song *Song, diagnostics *[]ScoreDiagnostic) {
	previous := make(map[int]PanAutomation)
	for i, a := range song.PanAutomations {
		location := ScoreLocation{Track: a.Track, Measure: a.Bar}
		if a.Track < 0 || a.Track >= len(song.Tracks) || a.Bar < 0 || a.Bar >= len(song.MeasureHeaders) || !panAutomationUnitValuesValid(a) {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.pan-automation.value", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("pan automation %d has track %d bar %d position %g value %g", i, a.Track, a.Bar, a.Position, a.Value)})
		}
		if p, ok := previous[a.Track]; ok && (a.Bar < p.Bar || a.Bar == p.Bar && a.Position < p.Position) {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.pan-automation.order", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("pan automation %d precedes its previous track event", i)})
		}
		previous[a.Track] = a
	}
}

func readBinaryPanAutomations(song *Song) {
	for ti, track := range song.Tracks {
		for bi, measure := range track.Measures {
			header := song.MeasureHeaders[bi]
			length := float64(header.TimeSignature.Numerator) * float64(DurationQuarterTime) * 4 / float64(header.TimeSignature.Denominator.Value)
			for _, voice := range measure.Voices {
				for _, beat := range voice.Beats {
					mt := beat.Effect.MixTableChange
					if mt == nil || mt.Balance == nil || beat.Start == nil {
						continue
					}
					offset := float64(*beat.Start - header.Start)
					if beat.ExactStart != nil {
						if exact, err := beat.ExactStart.Subtract(header.ExactStart); err == nil {
							offset = float64(exact.Numerator()) / float64(exact.Denominator())
						}
					}
					a := PanAutomation{Track: ti, Bar: bi, Position: offset / length, Value: float64(mt.Balance.Value) / 16, Linear: true}
					if mt.Balance.AllTracks {
						for target := range song.Tracks {
							a.Track = target
							song.PanAutomations = append(song.PanAutomations, a)
						}
					} else {
						song.PanAutomations = append(song.PanAutomations, a)
					}
				}
			}
		}
	}
	slices.SortStableFunc(song.PanAutomations, func(a, b PanAutomation) int {
		if a.Track != b.Track {
			return a.Track - b.Track
		}
		if a.Bar != b.Bar {
			return a.Bar - b.Bar
		}
		if a.Position < b.Position {
			return -1
		}
		if a.Position > b.Position {
			return 1
		}
		return 0
	})
}
