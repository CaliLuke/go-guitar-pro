// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// SystemLayout contains authored system counts for one score or track scope.
// A non-nil track layout takes precedence over the score layout. Export copies
// the score layout into an unspecified track with a normalization report.
// It never modifies either authored scope.
type SystemLayout struct {
	// DefaultBarsPerSystem supplies counts after BarsPerSystem is exhausted.
	// Zero leaves the source field absent; the pinned consumer then defaults to three.
	DefaultBarsPerSystem int
	// BarsPerSystem contains counts per explicitly laid-out system. A terminal
	// negative GP6 remainder is allowed only when it adjusts prior counts to the
	// score length; all reachable system counts must be positive.
	// Nil omits the source array unless both fields are empty. Export then emits
	// an empty array with a normalization report to retain the explicit scope.
	BarsPerSystem []int
}

const gpifMasterScaleID = "1124073984"
const gpifBarScaleID = "1124139520"

func readGPIFSystemLayout(defaultValue, array *string, measureCount int) (*SystemLayout, error) {
	if defaultValue == nil && array == nil {
		return nil, nil
	}
	layout := &SystemLayout{}
	parse := func(raw string) (int, error) {
		v, e := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
		if e != nil {
			return 0, fmt.Errorf("count %q must fit a signed 32-bit integer (maximum %d)", raw, math.MaxInt32)
		}
		return int(v), nil
	}
	if defaultValue != nil {
		v, e := parse(*defaultValue)
		if e != nil {
			return nil, e
		}
		if v < 1 {
			return nil, fmt.Errorf("default count %d must be positive", v)
		}
		layout.DefaultBarsPerSystem = v
	}
	if array != nil {
		layout.BarsPerSystem = make([]int, 0)
		for _, token := range strings.Fields(*array) {
			v, e := parse(token)
			if e != nil {
				return nil, e
			}
			layout.BarsPerSystem = append(layout.BarsPerSystem, v)
		}
	}
	if !validSystemCounts(layout.BarsPerSystem, measureCount) {
		return nil, fmt.Errorf("invalid system counts %v for %d measures", layout.BarsPerSystem, measureCount)
	}
	return layout, nil
}

func gp8SystemLayout(layout *SystemLayout) (defaultValue, array *string) {
	if layout == nil {
		return nil, nil
	}
	if layout.DefaultBarsPerSystem != 0 {
		value := strconv.Itoa(layout.DefaultBarsPerSystem)
		defaultValue = &value
	}
	if layout.BarsPerSystem != nil || layout.DefaultBarsPerSystem == 0 {
		values := make([]string, len(layout.BarsPerSystem))
		for i, v := range layout.BarsPerSystem {
			values[i] = strconv.Itoa(v)
		}
		value := strings.Join(values, " ")
		array = &value
	}
	return
}

func readGPIFDisplayScale(properties *gpifXProperties, id string, allowFloat bool) (*float64, error) {
	if properties == nil {
		return nil, nil
	}
	var result *float64
	for _, property := range properties.Properties {
		if property.ID != id {
			continue
		}
		if result != nil {
			return nil, fmt.Errorf("duplicate scale property %s", id)
		}
		raw := property.Double
		if raw == nil && allowFloat {
			raw = property.Float
		}
		if raw == nil {
			return nil, fmt.Errorf("scale property %s has no numeric payload", id)
		}
		value, e := strconv.ParseFloat(strings.TrimSpace(*raw), 64)
		if e != nil || !validDisplayScale(value) {
			return nil, fmt.Errorf("scale %q must be finite and positive", *raw)
		}
		result = &value
	}
	return result, nil
}

func gp8DisplayScale(properties *gpifXProperties, id string, value *float64) *gpifXProperties {
	if value == nil {
		return properties
	}
	if properties == nil {
		properties = &gpifXProperties{}
	}
	text := strconv.FormatFloat(*value, 'g', -1, 64)
	properties.Properties = append(properties.Properties, gpifXProperty{ID: id, Double: &text})
	return properties
}
func validDisplayScale(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
func validateDisplayScale(value *float64, code string, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	if value != nil && !validDisplayScale(*value) {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: code, Kind: ScoreDiagnosticValue, Location: location, Reason: "display scale must be finite and positive"})
	}
}
func validateSystemLayout(layout *SystemLayout, code string, location ScoreLocation, measureCount int, diagnostics *[]ScoreDiagnostic) {
	if layout == nil {
		return
	}
	invalid := layout.DefaultBarsPerSystem < 0 || layout.DefaultBarsPerSystem > math.MaxInt32
	invalid = invalid || !validSystemCounts(layout.BarsPerSystem, measureCount)
	if invalid {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: code, Kind: ScoreDiagnosticValue, Location: location, Reason: "system counts must be positive signed 32-bit values or a valid terminal GP6 remainder; default zero means absent"})
	}
}

// Legacy GP6 stores a negative terminal remainder after an oversized final system.
// It is retained only when the preceding counts plus that remainder equal the
// actual measure count; no later system can be reached in that representation.
func validSystemCounts(values []int, measureCount int) bool {
	var total int64
	for i, value := range values {
		if value < math.MinInt32 || value > math.MaxInt32 {
			return false
		}
		if value <= 0 {
			return value < 0 && i == len(values)-1 && total > int64(measureCount) && total+int64(value) == int64(measureCount)
		}
		if total > math.MaxInt64-int64(value) {
			return false
		}
		total += int64(value)
	}
	return true
}

func (builder *gp8Builder) reportEmptySystemLayout(layout *SystemLayout, location ScoreLocation) {
	if layout != nil && layout.DefaultBarsPerSystem == 0 && layout.BarsPerSystem == nil {
		builder.addReport("gp8.normalize.empty-layout-array", "score-core", ExportDispositionNormalized, location, "an explicit empty layout requires an empty GPIF array to retain its scope; reimport exposes an empty array instead of nil")
	}
}
