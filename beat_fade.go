// SPDX-License-Identifier: MIT

package goguitarpro

func (effect *BeatEffects) setImportedFade(fade BeatFade) {
	effect.Fade = fade
	effect.FadeIn = fade == BeatFadeIn
	effect.importedFade = fade
	effect.importedFadeIn = effect.FadeIn
	effect.hasImportedFade = true
}

// resolvedFade returns the fade used at the GP8 boundary. The conflict result
// is limited to incompatible edits of both views after import.
func (effect BeatEffects) resolvedFade() (BeatFade, bool) {
	legacy := BeatFadeNone
	if effect.FadeIn {
		legacy = BeatFadeIn
	}
	return resolveImportedEffectValue(effect.Fade, legacy, effect.importedFade, effect.importedFadeIn, effect.FadeIn, effect.hasImportedFade)
}

func resolveImportedEffectValue[T comparable](typed, legacy, importedTyped T, importedLegacy, legacyPresence bool, hasImported bool) (T, bool) {
	var zero T
	if !hasImported {
		if typed != zero {
			return typed, false
		}
		return legacy, false
	}
	typedChanged := typed != importedTyped
	legacyChanged := legacyPresence != importedLegacy
	switch {
	case typedChanged && !legacyChanged:
		return typed, false
	case legacyChanged && !typedChanged:
		return legacy, false
	case typedChanged && legacyChanged:
		return typed, typed != legacy
	default:
		return importedTyped, false
	}
}

func gp8BeatFade(fade BeatFade) string {
	switch fade {
	case BeatFadeIn:
		return "FadeIn"
	case BeatFadeOut:
		return "FadeOut"
	case BeatFadeVolumeSwell:
		return "VolumeSwell"
	default:
		return ""
	}
}

func gpifBeatFade(value string) BeatFade {
	switch value {
	case "FadeIn":
		return BeatFadeIn
	case "FadeOut":
		return BeatFadeOut
	case "VolumeSwell":
		return BeatFadeVolumeSwell
	default:
		return BeatFadeNone
	}
}

func gp8BeatFadeValue(effect BeatEffects) string {
	fade, _ := effect.resolvedFade()
	return gp8BeatFade(fade)
}
