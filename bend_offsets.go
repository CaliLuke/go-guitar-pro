// SPDX-License-Identifier: MIT

package goguitarpro

import "math"

func legacyBendOffset(position uint8) float64 {
	return float64(position) * 100 / float64(BendEffectMaxPosition)
}

func importedBendPoint(offset float64, value int8, vibrato bool) BendPoint {
	position := uint8(math.Round(offset * float64(BendEffectMaxPosition) / 100))
	point := BendPoint{Position: position, Value: value, Vibrato: vibrato}
	if offset == legacyBendOffset(position) {
		return point
	}
	point.ExactOffset = &offset
	point.importedPosition = position
	point.hasImportedExactOffset = true
	return point
}

func bendPointWithOffset(offset float64, value int8) BendPoint {
	position := uint8(math.Round(offset * float64(BendEffectMaxPosition) / 100))
	point := BendPoint{Position: position, Value: value}
	if offset != legacyBendOffset(position) {
		point.ExactOffset = &offset
	}
	return point
}

func cloneBendPoint(point BendPoint) BendPoint {
	if point.ExactOffset != nil {
		offset := *point.ExactOffset
		point.ExactOffset = &offset
	}
	return point
}

func resolvedBendOffset(point BendPoint) float64 {
	if point.ExactOffset == nil {
		return legacyBendOffset(point.Position)
	}
	if point.hasImportedExactOffset && point.Position != point.importedPosition {
		return legacyBendOffset(point.Position)
	}
	return *point.ExactOffset
}

func resolvedBendRawOffset(point BendPoint) float64 {
	raw := resolvedBendOffset(point) * float64(GPBendPosition) / 100
	return math.Round(raw*1_000_000_000) / 1_000_000_000
}

func bendPointOffsetAuthorityConflict(point BendPoint) bool {
	return point.ExactOffset != nil && point.hasImportedExactOffset && point.Position != point.importedPosition && *point.ExactOffset != legacyBendOffset(point.Position)
}

func sameBendPoint(left, right BendPoint) bool {
	return resolvedBendOffset(left) == resolvedBendOffset(right) && left.Value == right.Value && left.Vibrato == right.Vibrato
}

func sameBendPoints(left, right []BendPoint) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !sameBendPoint(left[index], right[index]) {
			return false
		}
	}
	return true
}

// nonmonotonicBendControlRoles recognizes the bounded GPIF role tuple whose
// destination precedes a middle control. The two middle roles share one value.
func nonmonotonicBendControlRoles(points []BendPoint) bool {
	if len(points) != 4 || points[1].Value != points[2].Value {
		return false
	}
	origin, middle1 := resolvedBendOffset(points[0]), resolvedBendOffset(points[1])
	middle2, destination := resolvedBendOffset(points[2]), resolvedBendOffset(points[3])
	return origin <= middle1 && middle1 <= middle2 && origin <= destination && destination < middle2
}
