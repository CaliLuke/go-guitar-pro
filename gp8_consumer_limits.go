// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Match the pinned GPIF reader's binary64 division, multiplication, and
// JavaScript ToInt32 conversion. Keep the exact authored rational on the wire.
func gp8ConsumerFermataOffset(offset ScoreTime) int64 {
	parts := strings.Split(gp8FermataOffset(offset), "/")
	numerator, _ := strconv.ParseFloat(parts[0], 64)
	denominator, _ := strconv.ParseFloat(parts[1], 64)
	ticks := float64(numerator/denominator) * float64(DurationQuarterTime)
	target := math.Mod(math.Trunc(ticks), 4294967296)
	if target >= 2147483648 {
		target -= 4294967296
	}
	return int64(target)
}

func (builder *gp8Builder) reportFermataConsumerLimits(header *MeasureHeader, location ScoreLocation) {
	seen := make(map[int64]int)
	for index, fermata := range header.Fermatas {
		target := gp8ConsumerFermataOffset(fermata.Offset)
		exactTarget, _ := NewScoreTime(target, 1)
		if fermata.Offset.Compare(exactTarget) != 0 {
			builder.addReport("gp8.normalize.fermata-consumer-offset", "fermata", ExportDispositionNormalized, location,
				fmt.Sprintf("pinned AlphaTab moves fermata[%d] offset %d/%d to tick %d; GPIF retains the exact offset", index, fermata.Offset.Numerator(), fermata.Offset.Denominator(), target))
		}
		if previous, exists := seen[target]; exists {
			builder.addReport("gp8.omit.fermata-consumer-collision", "fermata", ExportDispositionOmitted, location,
				fmt.Sprintf("pinned AlphaTab overwrites fermata[%d] with fermata[%d] at consumer tick %d; GPIF retains both records", previous, index, target))
		}
		seen[target] = index
	}
}

func (builder *gp8Builder) reportLegatoConsumerLimit(staff *Staff, location ScoreLocation, hasGrace bool) {
	beats := staff.Measures[location.Measure].Voices[location.Voice].Beats
	beat := &beats[location.Beat]
	var previous *Beat
	if location.Beat > 0 {
		previous = &beats[location.Beat-1]
	} else if location.Measure > 0 {
		voices := staff.Measures[location.Measure-1].Voices
		if location.Voice < len(voices) && len(voices[location.Voice].Beats) > 0 {
			previousBeats := voices[location.Voice].Beats
			previous = &previousBeats[len(previousBeats)-1]
		}
	}
	// Exported grace beats immediately precede their owner and have no Legato.
	derived := !hasGrace && previous != nil && previous.Legato != nil && previous.Legato.Origin
	authored := beat.Legato != nil && beat.Legato.Destination
	if authored != derived {
		builder.addReport("gp8.omit.legato-consumer-destination", "legato-slurs", ExportDispositionOmitted, location,
			fmt.Sprintf("pinned AlphaTab derives destination=%t from the preceding beat instead of authored destination=%t; GPIF retains the authored flags", derived, authored))
	}
}

func (builder *gp8Builder) reportChordConsumerLimit(chord *Chord, location ScoreLocation) {
	baseFret := 0
	if chord.FirstFret != nil && *chord.FirstFret > 0 {
		baseFret = int(*chord.FirstFret) - 1
	}
	ringPositions := make(map[int]int)
	for _, position := range gp8ChordPositions(chord, baseFret) {
		if position.Finger != "Ring" {
			continue
		}
		ringPositions[position.Fret]++
		if ringPositions[position.Fret] == 2 {
			builder.addReport("gp8.omit.chord-annular-consumer-barre", "chord-diagram", ExportDispositionOmitted, location,
				fmt.Sprintf("pinned AlphaTab ignores the Ring finger token and loses the annular barre at fret %d; GPIF retains the authored finger meaning", baseFret+position.Fret))
		}
	}
}

func buildGP8ChannelStripAutomations(song *Song, trackIndex int) gpifAutomations {
	var result gpifAutomations
	for _, automation := range song.VolumeAutomations {
		if automation.Track == trackIndex {
			result.Automations = append(result.Automations, gpifAutomation{
				Type: "DSPParam_12", Bar: automation.Bar, Position: automation.Position, Linear: automation.Linear,
				Value: gpifAutomationValue{Text: strconv.FormatFloat(automation.Value, 'g', -1, 64)},
			})
		}
	}
	for _, automation := range song.PanAutomations {
		if automation.Track == trackIndex {
			result.Automations = append(result.Automations, gpifAutomation{Type: "DSPParam_11", Bar: automation.Bar, Position: automation.Position, Linear: automation.Linear, Value: gpifAutomationValue{Text: strconv.FormatFloat(automation.Value, 'g', -1, 64)}})
		}
	}
	return result
}
