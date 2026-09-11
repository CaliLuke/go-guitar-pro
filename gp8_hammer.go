// SPDX-License-Identifier: MIT
package goguitarpro

// This projection follows the pinned consumer's Note.findHammerPullDestination.
// It only reports target losses. It never writes inferred links into the score.
type gp8HammerBeat struct {
	notes       []Note
	location    ScoreLocation
	stringCount int
	generated   bool
	next        int
}

func (builder *gp8Builder) recordHammerBeats(staff *Staff, location ScoreLocation, beat *Beat) {
	if builder.hammerLast == nil {
		builder.hammerLast = make(map[[3]int]int)
	}
	key := [3]int{location.Track, location.Staff, location.Voice}
	appendBeat := func(notes []Note, generated bool) {
		index := len(builder.hammerBeats)
		if previous, ok := builder.hammerLast[key]; ok {
			builder.hammerBeats[previous].next = index
		}
		count := len(staff.Strings)
		if staff.PercussionTrack {
			count = 0
		}
		builder.hammerBeats = append(builder.hammerBeats, gp8HammerBeat{notes: notes, location: location, stringCount: count, generated: generated, next: -1})
		builder.hammerLast[key] = index
	}
	for _, sequence := range graceSequences(beat) {
		for _, group := range builder.graceGroups(location.Track, beat, sequence) {
			appendBeat(group.notes, true)
		}
	}
	appendBeat(beat.Notes, false)
}
func (beat *gp8HammerBeat) noteOnString(str int) int {
	if str <= 0 || beat.stringCount == 0 {
		return -1
	}
	// The consumer's string lookup keeps the last note on a repeated string.
	for i := len(beat.notes) - 1; i >= 0; i-- {
		if int(beat.notes[i].String) == str {
			return i
		}
	}
	return -1
}
func (beat *gp8HammerBeat) hammerTarget(str int) int {
	if target := beat.noteOnString(str); target >= 0 {
		return target
	}
	for _, direction := range []int{-1, 1} {
		start := str
		if direction > 0 && start < 1 {
			start = 1
		}
		for candidate := start; candidate > 0 && candidate <= beat.stringCount; candidate += direction {
			if target := beat.noteOnString(candidate); target >= 0 {
				if beat.notes[target].Effect.LeftHandTapped {
					return target
				}
				break
			}
		}
	}
	return -1
}
func (builder *gp8Builder) reportHammerConsumerLimits() {
	incoming := make([][]bool, len(builder.hammerBeats))
	outgoing := make([][]bool, len(builder.hammerBeats))
	for i, beat := range builder.hammerBeats {
		incoming[i] = make([]bool, len(beat.notes))
		outgoing[i] = make([]bool, len(beat.notes))
	}
	for i, beat := range builder.hammerBeats {
		for n, note := range beat.notes {
			if !note.Effect.Hammer {
				continue
			}
			str := int(note.String)
			if beat.stringCount == 0 {
				str = 0
			}
			for j := beat.next; j >= 0 && builder.hammerBeats[j].location.Measure <= beat.location.Measure+1; j = builder.hammerBeats[j].next {
				if target := builder.hammerBeats[j].hammerTarget(str); target >= 0 {
					incoming[j][target] = true
					outgoing[i][n] = true
					break
				}
				// During fresh GPIF import, later bars have not chained their
				// beats yet. Only the next bar's first beat is reachable.
				if builder.hammerBeats[j].location.Measure > beat.location.Measure {
					break
				}
			}
		}
	}
	for i, beat := range builder.hammerBeats {
		if beat.generated {
			continue
		}
		for n, note := range beat.notes {
			location := beat.location
			location.Note = n
			if note.Effect.Hammer && !outgoing[i][n] {
				builder.addReport("gp8.omit.hammer-origin-consumer", "note-and-beat-semantics", ExportDispositionOmitted, location, "the pinned consumer clears an origin without a reachable same-string or left-hand-tap destination during fresh GPIF import")
			}
			if note.Effect.HammerDestination && !incoming[i][n] {
				builder.addReport("gp8.omit.hammer-destination-consumer", "note-and-beat-semantics", ExportDispositionOmitted, location, "GPIF retains the destination marker, but the pinned consumer ignores it without a derived incoming hammer/pull link")
			}
		}
	}
}
