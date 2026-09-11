// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// WahPedal identifies a beat-local pedal event, independent from sound settings.
type WahPedal uint8

const (
	// WahPedalNone means no pedal event; it does not reset a previous event.
	WahPedalNone WahPedal = iota
	// WahPedalOpen opens the wah pedal.
	WahPedalOpen
	// WahPedalClosed closes the wah pedal.
	WahPedalClosed
)

func legacyWah(value int8) WahPedal {
	switch {
	case value >= 100:
		return WahPedalClosed
	case value >= 0:
		return WahPedalOpen
	default:
		return WahPedalNone
	}
}

func gpifWah(value string) WahPedal {
	switch value {
	case "Open":
		return WahPedalOpen
	case "Closed":
		return WahPedalClosed
	default:
		return WahPedalNone
	}
}

func gp8Wah(value WahPedal) string {
	switch value {
	case WahPedalOpen:
		return "Open"
	case WahPedalClosed:
		return "Closed"
	default:
		return ""
	}
}

func (effect BeatEffects) legacyWahValue() (int8, bool) {
	if effect.MixTableChange != nil && effect.MixTableChange.Wah != nil {
		return effect.MixTableChange.Wah.Value, true
	}
	return -1, false
}

func (effect *BeatEffects) rememberWah() {
	effect.importedWah = effect.WahPedal
	effect.importedWahValue, effect.importedWahPresent = effect.legacyWahValue()
	effect.hasImportedWah = true
}

type wahResolution struct {
	state                WahPedal
	conflict, legacyUsed bool
}

func (effect BeatEffects) resolvedWah() wahResolution {
	value, present := effect.legacyWahValue()
	legacy := legacyWah(value)
	if effect.hasImportedWah {
		typedEdited := effect.WahPedal != effect.importedWah
		legacyEdited := value != effect.importedWahValue || present != effect.importedWahPresent
		if typedEdited {
			return wahResolution{state: effect.WahPedal, conflict: legacyEdited && effect.WahPedal != legacy}
		}
		if legacyEdited {
			return wahResolution{state: legacy, legacyUsed: present}
		}
		return wahResolution{state: effect.WahPedal, legacyUsed: present}
	}
	if effect.WahPedal != WahPedalNone {
		return wahResolution{state: effect.WahPedal, conflict: present && effect.WahPedal != legacy}
	}
	return wahResolution{state: legacy, legacyUsed: present}
}

func (builder *gp8Builder) reportWah(beat *Beat, location ScoreLocation) {
	resolved := beat.Effect.resolvedWah()
	if resolved.conflict {
		builder.addReport("gp8.normalize.wah-authority", "wah", ExportDispositionNormalized, location, "the edited pedal state takes precedence over incompatible legacy mix-table wah edits")
	}
	raw := beat.Effect.MixTableChange
	if raw == nil || raw.Wah == nil {
		return
	}
	value := raw.Wah.Value
	switch {
	case value < -1:
		builder.addReport("gp8.omit.wah-legacy-state", "wah", ExportDispositionOmitted, location, fmt.Sprintf("legacy wah value %d has no supported pedal event", value))
	case resolved.legacyUsed && value != -1 && value != 0 && value != 100:
		builder.addReport("gp8.normalize.wah-legacy-value", "wah", ExportDispositionNormalized, location, fmt.Sprintf("legacy wah value %d retains only the pinned consumer %s pedal state", value, gp8Wah(resolved.state)))
	}
	if value != -1 || raw.Wah.Display {
		builder.addReport("gp8.omit.wah-display", "wah", ExportDispositionOmitted, location, fmt.Sprintf("GPIF has no field for the legacy wah display flag %t", raw.Wah.Display))
	}
}
