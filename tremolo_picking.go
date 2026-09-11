// SPDX-License-Identifier: MIT

package goguitarpro

func cloneTremoloPickingEffect(effect *TremoloPickingEffect) *TremoloPickingEffect {
	if effect == nil {
		return nil
	}
	copy := *effect
	return &copy
}

func sameTremoloPickingEffect(left, right *TremoloPickingEffect) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Duration == right.Duration && left.Style == right.Style
}

// resolvedTremoloPicking uses the beat-wide field when present. Otherwise, it
// falls back to the final legacy note-local field, matching Guitar Pro's
// beat-wide interpretation of its binary note-effect record.
func (beat *Beat) resolvedTremoloPicking() (*TremoloPickingEffect, bool) {
	resolved := beat.Effect.TremoloPicking
	conflict := false
	if resolved != nil {
		for noteIndex := range beat.Notes {
			legacy := beat.Notes[noteIndex].Effect.TremoloPicking
			if legacy != nil && !sameTremoloPickingEffect(resolved, legacy) {
				conflict = true
			}
		}
		return resolved, conflict
	}
	for noteIndex := range beat.Notes {
		legacy := beat.Notes[noteIndex].Effect.TremoloPicking
		if legacy == nil {
			continue
		}
		if resolved != nil && !sameTremoloPickingEffect(resolved, legacy) {
			conflict = true
		}
		resolved = legacy
	}
	return resolved, conflict
}

func (beat *Beat) promoteLegacyTremoloPicking() {
	resolved, _ := beat.resolvedTremoloPicking()
	beat.Effect.TremoloPicking = cloneTremoloPickingEffect(resolved)
}

func gp8TremoloPickingValue(effect *TremoloPickingEffect) (string, bool) {
	if effect == nil {
		return "", false
	}
	duration := effect.Duration
	plainTuplet := (duration.TupletEnters == 0 && duration.TupletTimes == 0) ||
		(duration.TupletEnters == 1 && duration.TupletTimes == 1)
	if duration.Dotted || duration.DoubleDotted || !plainTuplet {
		return "", false
	}
	switch duration.Value {
	case uint16(DurationEighth):
		return "1/2", true
	case uint16(DurationSixteenth):
		return "1/4", true
	case uint16(DurationThirtySecond):
		return "1/8", true
	default:
		return "", false
	}
}
