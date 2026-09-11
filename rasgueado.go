// SPDX-License-Identifier: MIT

package goguitarpro

// RasgueadoPattern identifies an authored Guitar Pro finger pattern.
type RasgueadoPattern uint8

const (
	// RasgueadoNone indicates no named pattern.
	RasgueadoNone RasgueadoPattern = 0
	// RasgueadoIi identifies the ii_1 finger pattern.
	RasgueadoIi RasgueadoPattern = 1
	// RasgueadoMi identifies the mi_1 finger pattern.
	RasgueadoMi RasgueadoPattern = 2
	// RasgueadoMiiTriplet identifies the mii_1 finger pattern.
	RasgueadoMiiTriplet RasgueadoPattern = 3
	// RasgueadoMiiAnapaest identifies the mii_2 finger pattern.
	RasgueadoMiiAnapaest RasgueadoPattern = 4
	// RasgueadoPmpTriplet identifies the pmp_1 finger pattern.
	RasgueadoPmpTriplet RasgueadoPattern = 5
	// RasgueadoPmpAnapaest identifies the pmp_2 finger pattern.
	RasgueadoPmpAnapaest RasgueadoPattern = 6
	// RasgueadoPeiTriplet identifies the pei_1 finger pattern.
	RasgueadoPeiTriplet RasgueadoPattern = 7
	// RasgueadoPeiAnapaest identifies the pei_2 finger pattern.
	RasgueadoPeiAnapaest RasgueadoPattern = 8
	// RasgueadoPaiTriplet identifies the pai_1 finger pattern.
	RasgueadoPaiTriplet RasgueadoPattern = 9
	// RasgueadoPaiAnapaest identifies the pai_2 finger pattern.
	RasgueadoPaiAnapaest RasgueadoPattern = 10
	// RasgueadoAmiTriplet identifies the ami_1 finger pattern.
	RasgueadoAmiTriplet RasgueadoPattern = 11
	// RasgueadoAmiAnapaest identifies the ami_2 finger pattern.
	RasgueadoAmiAnapaest RasgueadoPattern = 12
	// RasgueadoPpp identifies the ppp_1 finger pattern.
	RasgueadoPpp RasgueadoPattern = 13
	// RasgueadoAmii identifies the amii_1 finger pattern.
	RasgueadoAmii RasgueadoPattern = 14
	// RasgueadoAmip identifies the amip_1 finger pattern.
	RasgueadoAmip RasgueadoPattern = 15
	// RasgueadoEami identifies the eami_1 finger pattern.
	RasgueadoEami RasgueadoPattern = 16
	// RasgueadoEamii identifies the eamii_1 finger pattern.
	RasgueadoEamii RasgueadoPattern = 17
	// RasgueadoPeami identifies the peami_1 finger pattern.
	RasgueadoPeami RasgueadoPattern = 18
)

var rasgueadoTokens = [...]string{"", "ii_1", "mi_1", "mii_1", "mii_2", "pmp_1", "pmp_2", "pei_1", "pei_2", "pai_1", "pai_2", "ami_1", "ami_2", "ppp_1", "amii_1", "amip_1", "eami_1", "eamii_1", "peami_1"}

func gpifRasgueado(value string) RasgueadoPattern {
	for index, token := range rasgueadoTokens {
		if token == value {
			return RasgueadoPattern(index)
		}
	}
	return RasgueadoNone
}
func gp8Rasgueado(pattern RasgueadoPattern) string {
	if int(pattern) >= len(rasgueadoTokens) {
		return ""
	}
	return rasgueadoTokens[pattern]
}
func (effect *BeatEffects) rememberRasgueado() {
	effect.importedRasgueado = effect.RasgueadoPattern
	effect.importedHasRasgueado = effect.HasRasgueado
	effect.hasImportedRasgueado = true
}

func (effect BeatEffects) resolvedRasgueado() (pattern RasgueadoPattern, conflict, unspecified bool) {
	typedChanged := effect.hasImportedRasgueado && effect.RasgueadoPattern != effect.importedRasgueado
	legacyChanged := effect.hasImportedRasgueado && effect.HasRasgueado != effect.importedHasRasgueado
	if typedChanged {
		return effect.RasgueadoPattern, legacyChanged && (effect.RasgueadoPattern != RasgueadoNone) != effect.HasRasgueado, false
	}
	if legacyChanged {
		if effect.HasRasgueado {
			return RasgueadoIi, false, true
		}
		return RasgueadoNone, false, false
	}
	if effect.RasgueadoPattern != RasgueadoNone {
		return effect.RasgueadoPattern, false, false
	}
	if effect.HasRasgueado {
		return RasgueadoIi, false, true
	}
	return RasgueadoNone, false, false
}
