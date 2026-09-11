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
