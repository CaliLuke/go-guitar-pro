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
	return resolveImportedEffectValue(effect.VibratoStrength, legacy, effect.importedVibratoStrength, effect.importedVibrato, effect.Vibrato, effect.hasImportedVibrato)
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
