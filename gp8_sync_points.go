// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"math/big"
	"strconv"
)

func buildGP8SyncPoints(song *Song) []gpifAutomation {
	result := make([]gpifAutomation, 0, len(song.SyncPoints))
	for _, point := range song.SyncPoints {
		ratio := point.BarPosition.Ratio()
		result = append(result, gpifAutomation{
			Type: "SyncPoint", Bar: point.Bar, Position: float64(ratio.Numerator()) / float64(ratio.Denominator()), Linear: point.Linear, Visible: strconv.FormatBool(point.Visible),
			Value: gpifAutomationValue{BarIndex: strconv.Itoa(point.Bar), BarOccurrence: strconv.Itoa(point.BarOccurrence), FrameOffset: strconv.FormatInt(int64(point.AudioFrame), 10), ModifiedTempo: strconv.FormatFloat(point.ModifiedTempo, 'g', -1, 64), OriginalTempo: strconv.FormatFloat(point.OriginalTempo, 'g', -1, 64)},
		})
	}
	return result
}

func gp8SyncIntegerExact(value int64) bool {
	rounded, _ := new(big.Float).SetFloat64(float64(value)).Int(nil)
	return rounded.Cmp(big.NewInt(value)) == 0
}

func gp8SyncPositionExact(point SyncPoint) bool {
	ratio := point.BarPosition.Ratio()
	target, err := NewBarPositionFromFloat64(float64(ratio.Numerator()) / float64(ratio.Denominator()))
	return err == nil && target.Ratio().Compare(ratio) == 0
}
