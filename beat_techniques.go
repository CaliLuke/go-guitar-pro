// SPDX-License-Identifier: MIT

package goguitarpro

type beatTechniques struct{ tap, slap, pop bool }

func techniquesFromLegacy(value SlapEffect) beatTechniques {
	return beatTechniques{tap: value == SlapEffectTapping, slap: value == SlapEffectSlapping, pop: value == SlapEffectPopping}
}

func (value beatTechniques) legacy() SlapEffect {
	switch {
	case value.pop:
		return SlapEffectPopping
	case value.slap:
		return SlapEffectSlapping
	case value.tap:
		return SlapEffectTapping
	default:
		return SlapEffectNone
	}
}

func (effect *BeatEffects) rememberTechniques() {
	effect.importedTechniques = beatTechniques{effect.Tap, effect.Slap, effect.Pop}
	effect.importedSlapEffect = effect.SlapEffect
	effect.hasImportedTechniques = true
}

func (effect BeatEffects) resolvedTechniques() (beatTechniques, bool) {
	typed := beatTechniques{effect.Tap, effect.Slap, effect.Pop}
	legacy := techniquesFromLegacy(effect.SlapEffect)
	if effect.hasImportedTechniques {
		typedEdited := typed != effect.importedTechniques
		legacyEdited := effect.SlapEffect != effect.importedSlapEffect
		if typedEdited {
			return typed, legacyEdited && legacy != typed
		}
		if legacyEdited {
			return legacy, false
		}
		return typed, false
	}
	if typed != (beatTechniques{}) {
		return typed, effect.SlapEffect != SlapEffectNone && legacy != typed
	}
	return legacy, false
}

func (beat *Beat) importLegacyTechniques() {
	value := techniquesFromLegacy(beat.Effect.SlapEffect)
	beat.Effect.Tap, beat.Effect.Slap, beat.Effect.Pop = value.tap, value.slap, value.pop
	beat.Effect.rememberTechniques()
}

func (beat *Beat) importGPIFTechniques() {
	beat.Effect.Tap = beat.hasTappedNote()
	beat.Effect.SlapEffect = (beatTechniques{beat.Effect.Tap, beat.Effect.Slap, beat.Effect.Pop}).legacy()
	beat.Effect.rememberTechniques()
}

func (beat *Beat) hasTappedNote() bool {
	for i := range beat.Notes {
		if beat.Notes[i].Effect.Tapped {
			return true
		}
	}
	return false
}

func (builder *gp8Builder) reportBeatTechniques(beat *Beat, location ScoreLocation) {
	value, conflict := beat.Effect.resolvedTechniques()
	if conflict {
		builder.addReport("gp8.normalize.beat-technique-authority", "tap-slap-pop", ExportDispositionNormalized, location, "independent tap, slap and pop states take precedence over incompatible legacy enum edits")
	}
	switch {
	case value.tap && len(beat.Notes) == 0:
		builder.addReport("gp8.omit.beat-tap", "tap-slap-pop", ExportDispositionOmitted, location, "the pinned GPIF consumer requires a note Tapped property to retain beat tap; no note is added to a rest")
	case value.tap && !beat.hasTappedNote():
		location.Note = 0
		builder.addReport("gp8.normalize.beat-tap-note-flag", "tap-slap-pop", ExportDispositionNormalized, location, "the first exported note receives a Tapped flag so the pinned GPIF consumer retains beat tap; the authored note is unchanged")
	case !value.tap && beat.hasTappedNote():
		builder.addReport("gp8.normalize.beat-tap-note-authority", "tap-slap-pop", ExportDispositionNormalized, location, "authored note Tapped flags are retained and cause the pinned GPIF consumer to enable beat tap despite the cleared beat view")
	}
}
