// SPDX-License-Identifier: MIT

package goguitarpro

import "slices"

var directionSignOrder = []DirectionSign{
	DirectionSignCoda,
	DirectionSignDoubleCoda,
	DirectionSignSegno,
	DirectionSignSegnoSegno,
	DirectionSignFine,
}

var directionJumpOrder = []DirectionSign{
	DirectionSignDaCapo,
	DirectionSignDaCapoAlCoda,
	DirectionSignDaCapoAlDoubleCoda,
	DirectionSignDaCapoAlFine,
	DirectionSignDaSegno,
	DirectionSignDaSegnoAlCoda,
	DirectionSignDaSegnoAlDoubleCoda,
	DirectionSignDaSegnoAlFine,
	DirectionSignDaSegnoSegno,
	DirectionSignDaSegnoSegnoAlCoda,
	DirectionSignDaSegnoSegnoAlDoubleCoda,
	DirectionSignDaSegnoSegnoAlFine,
	DirectionSignDaCoda,
	DirectionSignDaDoubleCoda,
}

func canonicalDirections(values []DirectionSign) []DirectionSign {
	if len(values) == 0 {
		return nil
	}
	result := slices.Clone(values)
	slices.Sort(result)
	return slices.Compact(result)
}

func cloneDirection(value *DirectionSign) *DirectionSign {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func equalDirection(left, right *DirectionSign) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func (header *MeasureHeader) markDirectionCompatibility() {
	header.Directions = canonicalDirections(header.Directions)
	if len(header.Directions) > 0 {
		header.Direction = cloneDirection(&header.Directions[len(header.Directions)-1])
	} else {
		header.Direction = nil
	}
	header.directionsCompatibility = slices.Clone(header.Directions)
	header.directionCompatibility = cloneDirection(header.Direction)
	header.directionCompatibilitySet = true
}

func (header *MeasureHeader) resolvedDirections() []DirectionSign {
	if !header.directionCompatibilitySet {
		if header.Directions != nil {
			return canonicalDirections(header.Directions)
		}
		if header.Direction != nil {
			return []DirectionSign{*header.Direction}
		}
		return nil
	}
	if !equalDirection(header.Direction, header.directionCompatibility) {
		if header.Direction == nil {
			return nil
		}
		return []DirectionSign{*header.Direction}
	}
	return canonicalDirections(header.Directions)
}

func directionTargetToken(direction DirectionSign) (string, bool) {
	switch direction {
	case DirectionSignCoda:
		return "Coda", true
	case DirectionSignDoubleCoda:
		return "DoubleCoda", true
	case DirectionSignSegno:
		return "Segno", true
	case DirectionSignSegnoSegno:
		return "SegnoSegno", true
	case DirectionSignFine:
		return "Fine", true
	default:
		return "", false
	}
}

func directionJumpToken(direction DirectionSign) (string, bool) {
	switch direction {
	case DirectionSignDaCapo:
		return "DaCapo", true
	case DirectionSignDaCapoAlCoda:
		return "DaCapoAlCoda", true
	case DirectionSignDaCapoAlDoubleCoda:
		return "DaCapoAlDoubleCoda", true
	case DirectionSignDaCapoAlFine:
		return "DaCapoAlFine", true
	case DirectionSignDaSegno:
		return "DaSegno", true
	case DirectionSignDaSegnoAlCoda:
		return "DaSegnoAlCoda", true
	case DirectionSignDaSegnoAlDoubleCoda:
		return "DaSegnoAlDoubleCoda", true
	case DirectionSignDaSegnoAlFine:
		return "DaSegnoAlFine", true
	case DirectionSignDaSegnoSegno:
		return "DaSegnoSegno", true
	case DirectionSignDaSegnoSegnoAlCoda:
		return "DaSegnoSegnoAlCoda", true
	case DirectionSignDaSegnoSegnoAlDoubleCoda:
		return "DaSegnoSegnoAlDoubleCoda", true
	case DirectionSignDaSegnoSegnoAlFine:
		return "DaSegnoSegnoAlFine", true
	case DirectionSignDaCoda:
		return "DaCoda", true
	case DirectionSignDaDoubleCoda:
		return "DaDoubleCoda", true
	default:
		return "", false
	}
}

func directionFromTargetToken(value string) (DirectionSign, bool) {
	for _, direction := range directionSignOrder {
		if token, _ := directionTargetToken(direction); token == value {
			return direction, true
		}
	}
	return 0, false
}

func directionFromJumpToken(value string) (DirectionSign, bool) {
	for _, direction := range directionJumpOrder {
		if token, _ := directionJumpToken(direction); token == value {
			return direction, true
		}
	}
	return 0, false
}
