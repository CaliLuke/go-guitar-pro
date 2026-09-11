// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"strconv"
	"strings"
)

// BeatTimer requests a timer mark on one beat. A nil Milliseconds value requests
// a playback-derived timer; this library does not calculate playback timers.
type BeatTimer struct {
	// Milliseconds is an explicit absolute time, including zero. Direct edits control
	// GP8 output. Values must be in 0..9007199254740991 for exact consumer retention.
	Milliseconds *int64
}

const maxBeatTimerMilliseconds int64 = 1<<53 - 1

func parseGPIFBeatTimer(raw string) (*int64, error) {
	text := strings.TrimSpace(raw)
	if text == "" || text == "-1" {
		return nil, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil || value < 0 || value > maxBeatTimerMilliseconds {
		return nil, fmt.Errorf("timer %q must be empty, -1, or integer milliseconds in 0..%d", raw, maxBeatTimerMilliseconds)
	}
	return &value, nil
}

func gp8BeatTimer(timer *BeatTimer) *string {
	if timer == nil {
		return nil
	}
	text := "-1"
	if timer.Milliseconds != nil {
		text = strconv.FormatInt(*timer.Milliseconds, 10)
	}
	return &text
}

func cloneBeatTimer(timer *BeatTimer) *BeatTimer {
	if timer == nil {
		return nil
	}
	return &BeatTimer{Milliseconds: cloneSemanticPointer(timer.Milliseconds)}
}

func sameBeatTimer(a, b *BeatTimer) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Milliseconds == nil || b.Milliseconds == nil {
		return a.Milliseconds == b.Milliseconds
	}
	return *a.Milliseconds == *b.Milliseconds
}

func validateGraceTimers(beat *Beat, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	timers := make(map[graceBeatKey]*BeatTimer)
	for noteIndex, note := range beat.Notes {
		location.Note = noteIndex
		for graceIndex, grace := range note.Effect.Graces {
			if timer := grace.Timer; timer != nil && timer.Milliseconds != nil && (*timer.Milliseconds < 0 || *timer.Milliseconds > maxBeatTimerMilliseconds) {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.grace.timer", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("grace %d timer milliseconds %d is outside 0..%d", graceIndex, *timer.Milliseconds, maxBeatTimerMilliseconds)})
			}
			key := graceBeatIdentity(&grace)
			if first, seen := timers[key]; seen && !sameBeatTimer(first, grace.Timer) {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.grace.timer-conflict", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("grace %d timer differs from another note in grace beat sequence %d", graceIndex, grace.Sequence)})
			} else if !seen {
				timers[key] = grace.Timer
			}
		}
	}
}
