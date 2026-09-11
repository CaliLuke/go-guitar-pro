// SPDX-License-Identifier: MIT

package goguitarpro

func (effect *BeatEffects) setImportedVibrato(strength BeatVibrato) {
	effect.VibratoStrength = strength
	effect.Vibrato = strength != BeatVibratoNone
	effect.importedVibratoStrength = strength
	effect.importedVibrato = effect.Vibrato
	effect.hasImportedVibrato = true
}

// resolvedVibrato returns the beat-wide value used at the GP8 boundary. The
// conflict result is limited to incompatible edits of both views after import.
func (effect BeatEffects) resolvedVibrato() (BeatVibrato, bool) {
	legacy := BeatVibratoNone
	if effect.Vibrato {
		legacy = BeatVibratoSlight
	}
	if !effect.hasImportedVibrato {
		if effect.VibratoStrength != BeatVibratoNone {
			return effect.VibratoStrength, false
		}
		return legacy, false
	}

	typedChanged := effect.VibratoStrength != effect.importedVibratoStrength
	legacyChanged := effect.Vibrato != effect.importedVibrato
	switch {
	case typedChanged && !legacyChanged:
		return effect.VibratoStrength, false
	case legacyChanged && !typedChanged:
		return legacy, false
	case typedChanged && legacyChanged:
		return effect.VibratoStrength, effect.VibratoStrength != legacy
	default:
		return effect.importedVibratoStrength, false
	}
}

func gp8BeatVibratoStrength(strength BeatVibrato) string {
	switch strength {
	case BeatVibratoSlight:
		return "Slight"
	case BeatVibratoWide:
		return "Wide"
	default:
		return ""
	}
}
