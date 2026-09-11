// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	gpifMasterBarBeamingDurationID = "1124139010"
	gpifMasterBarBeamingGroupBase  = int64(1124139264)
	gpifMasterBarBeamingGroupLast  = int64(1124139295)
	gpifBeatBeamDirectionInvertID  = "1124204545"
	gpifBeatBeamingModeID          = "1124204546"
	gpifBeatSecondarySplitID       = "1124204552"
)

func gpifXPropertyInt(property gpifXProperty) (int64, bool) {
	if property.Int == nil {
		return 0, false
	}
	raw := strings.TrimSpace(*property.Int)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	return value, err == nil
}

func gpifAuditBeamingMasterBar(context *parseContext, index int, properties *gpifXProperties) {
	if properties == nil {
		return
	}
	basePath := fmt.Sprintf("/GPIF/MasterBars/MasterBar[%d]/XProperties", index)
	durationCount := 0
	groups := make(map[int]int64)
	maxGroupIndex := -1
	invalidGroupSource := diagnosticSource("GPIF.MasterBar.Beaming.Group.InvalidValue", "beaming", ParseDiagnosticInvalidData)
	unknownPropertySource := diagnosticSource("GPIF.MasterBar.XProperty.Unknown", "score-core", ParseDiagnosticUnknownSyntax)
	for propertyIndex, property := range properties.Properties {
		path := fmt.Sprintf("%s/XProperty[%d]", basePath, propertyIndex)
		switch {
		case property.ID == gpifMasterScaleID:
			// The layout reader validates and retains this independent scale.
		case property.ID == gpifMasterBarBeamingDurationID:
			durationCount++
			value, ok := gpifXPropertyInt(property)
			if _, supported := gpifBeamingDuration(value); !ok || !supported {
				gpifAddBeamingDiagnostic(context, diagnosticSource("GPIF.MasterBar.Beaming.Duration.InvalidValue", "beaming", ParseDiagnosticInvalidData), path+"/Int", fmt.Sprintf("beaming duration %q must be one of 1, 2, 4, 8, 16, 32, 64, 128, or 256", gpifXPropertyText(property)))
			}
		case isGPIFBeamingGroupID(property.ID):
			groupIndex, _ := gpifXPropertyGroupIndex(property.ID)
			if _, duplicate := groups[groupIndex]; duplicate {
				gpifAddBeamingDiagnostic(context, diagnosticSource("GPIF.MasterBar.Beaming.Group.Duplicate", "beaming", ParseDiagnosticInvalidData), path, fmt.Sprintf("beaming group index %d is duplicated", groupIndex))
			}
			value, ok := gpifXPropertyInt(property)
			if !ok || value < 0 || value > math.MaxInt32 {
				gpifAddBeamingDiagnostic(context, invalidGroupSource, path+"/Int", fmt.Sprintf("beaming group %q must be within 0..%d; zero is allowed only as trailing padding", gpifXPropertyText(property), math.MaxInt32))
			}
			groups[groupIndex] = value
			maxGroupIndex = max(maxGroupIndex, groupIndex)
		default:
			context.add(unknownPropertySource, ParseDiagnostic{
				SourcePath: path + "/@id", Feature: "score-core",
				Reason: fmt.Sprintf("unknown master-bar XProperty id %q", property.ID),
			})
		}
	}
	if durationCount > 1 {
		gpifAddBeamingDiagnostic(context, diagnosticSource("GPIF.MasterBar.Beaming.Duration.Duplicate", "beaming", ParseDiagnosticInvalidData), basePath, "beaming duration property is duplicated")
	}
	if durationCount > 0 && len(groups) == 0 {
		gpifAddBeamingDiagnostic(context, diagnosticSource("GPIF.MasterBar.Beaming.Groups.Missing", "beaming", ParseDiagnosticInvalidData), basePath, "beaming duration requires at least one group")
	}
	if durationCount == 0 && len(groups) > 0 {
		gpifAddBeamingDiagnostic(context, diagnosticSource("GPIF.MasterBar.Beaming.Duration.Missing", "beaming", ParseDiagnosticInvalidData), basePath, "beaming groups require a duration")
	}
	for index := 0; index <= maxGroupIndex; index++ {
		if _, ok := groups[index]; !ok {
			gpifAddBeamingDiagnostic(context, diagnosticSource("GPIF.MasterBar.Beaming.Group.Gap", "beaming", ParseDiagnosticInvalidData), basePath, fmt.Sprintf("beaming groups must be contiguous from index 0; index %d is missing", index))
		}
	}
	lastPositive := -1
	for index, value := range groups {
		if value > 0 {
			lastPositive = max(lastPositive, index)
		}
	}
	if len(groups) > 0 && lastPositive < 0 {
		gpifAddBeamingDiagnostic(context, invalidGroupSource, basePath, "beaming rules require at least one positive group")
	}
	for index := 0; index <= lastPositive; index++ {
		if groups[index] == 0 {
			gpifAddBeamingDiagnostic(context, invalidGroupSource, basePath, fmt.Sprintf("beaming group index %d is zero before a later positive group", index))
		}
	}
}

func gpifAuditBeamingBeat(context *parseContext, beatID, path string, beat *gpifBeat) {
	for _, orientation := range []struct {
		name  string
		value string
	}{
		{"TransposedPitchStemOrientation", beat.TransposedPitchStemOrientation},
		{"UserTransposedPitchStemOrientation", beat.UserTransposedPitchStemOrientation},
	} {
		if orientation.value != "" && orientation.value != "Undefined" && orientation.value != "Upward" && orientation.value != "Downward" {
			source := diagnosticSource("GPIF.Beat.TransposedPitchStemOrientation.InvalidValue", "beaming", ParseDiagnosticInvalidData)
			if orientation.name == "UserTransposedPitchStemOrientation" {
				source = diagnosticSource("GPIF.Beat.UserTransposedPitchStemOrientation.InvalidValue", "beaming", ParseDiagnosticInvalidData)
			}
			context.add(source, ParseDiagnostic{
				SourcePath: path + "/" + orientation.name, ObjectID: beatID, Location: ParseLocation{BeatID: beatID},
				Feature: "beaming", Reason: fmt.Sprintf("unsupported stem orientation %q", orientation.value),
			})
		}
	}
	if beat.XProperties == nil {
		return
	}
	seen := make(map[string]struct{})
	unknownPropertySource := diagnosticSource("GPIF.Beat.XProperty.Unknown", "note-and-beat-semantics", ParseDiagnosticUnknownSyntax)
	brushSeen := false
	for propertyIndex, property := range beat.XProperties.Properties {
		allowed := map[string]map[int64]struct{}{
			gpifBeatBeamDirectionInvertID: {0: {}, 1: {}},
			gpifBeatBeamingModeID:         {0: {}, 1: {}, 2: {}},
			gpifBeatSecondarySplitID:      {0: {}, 1: {}},
		}[property.ID]
		propertyPath := fmt.Sprintf("%s/XProperties/XProperty[%d]", path, propertyIndex)
		if property.ID == gpifBeatBrushDurationID {
			if brushSeen {
				context.add(diagnosticSource("GPIF.Beat.Brush.Duration.Duplicate", "brush", ParseDiagnosticInvalidData), ParseDiagnostic{SourcePath: propertyPath, ObjectID: beatID, Location: ParseLocation{BeatID: beatID}, Reason: "brush duration XProperty is duplicated"})
			}
			brushSeen = true
			value, ok := gpifXPropertyInt(property)
			if !ok || value < 0 || value > math.MaxInt32 {
				context.add(diagnosticSource("GPIF.Beat.Brush.Duration.Invalid", "brush", ParseDiagnosticInvalidData), ParseDiagnostic{SourcePath: propertyPath + "/Int", ObjectID: beatID, Location: ParseLocation{BeatID: beatID}, Reason: fmt.Sprintf("brush duration %q must be an integer within 0..%d", gpifXPropertyText(property), math.MaxInt32)})
			}
			continue
		}
		if allowed == nil {
			context.add(unknownPropertySource, ParseDiagnostic{
				SourcePath: fmt.Sprintf("%s/XProperties/XProperty[%d]/@id", path, propertyIndex),
				ObjectID:   beatID, Location: ParseLocation{BeatID: beatID},
				Reason: fmt.Sprintf("unhandled beat XProperty id %q", property.ID),
			})
			continue
		}
		if _, duplicate := seen[property.ID]; duplicate {
			context.add(diagnosticSource("GPIF.Beat.Beaming.XProperty.Duplicate", "beaming", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: propertyPath, ObjectID: beatID, Location: ParseLocation{BeatID: beatID}, Feature: "beaming",
				Reason: fmt.Sprintf("beaming XProperty %q is duplicated", property.ID),
			})
		}
		seen[property.ID] = struct{}{}
		value, ok := gpifXPropertyInt(property)
		if _, valid := allowed[value]; !ok || !valid {
			context.add(diagnosticSource("GPIF.Beat.Beaming.XProperty.InvalidValue", "beaming", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: propertyPath + "/Int", ObjectID: beatID, Location: ParseLocation{BeatID: beatID}, Feature: "beaming",
				Reason: fmt.Sprintf("beaming XProperty %q has unsupported integer %q", property.ID, gpifXPropertyText(property)),
			})
		}
	}
}

func gpifAddBeamingDiagnostic(context *parseContext, source parseDiagnosticSource, path, reason string) {
	context.add(source, ParseDiagnostic{
		SourcePath: path, Feature: "beaming", Reason: reason,
	})
}

func gpifXPropertyText(property gpifXProperty) string {
	if property.Int == nil {
		return "<missing>"
	}
	return strings.TrimSpace(*property.Int)
}

func isGPIFBeamingGroupID(id string) bool {
	_, ok := gpifXPropertyGroupIndex(id)
	return ok
}

func gpifBeamingDuration(value int64) (NoteValue, bool) {
	switch value {
	case 1, 2, 4, 8, 16, 32, 64, 128, 256:
		return NoteValue(value), true
	default:
		return 0, false
	}
}

func gpifXPropertyGroupIndex(id string) (int, bool) {
	numeric, err := strconv.ParseInt(id, 10, 64)
	if err != nil || numeric < gpifMasterBarBeamingGroupBase || numeric > gpifMasterBarBeamingGroupLast {
		return 0, false
	}
	return int(numeric - gpifMasterBarBeamingGroupBase), true
}

func gpifParseBeamingRules(properties *gpifXProperties) *BeamingRules {
	if properties == nil {
		return nil
	}
	var duration NoteValue
	durationSet := false
	groups := make(map[int]int)
	maxIndex := -1
	for _, property := range properties.Properties {
		if property.ID == gpifMasterBarBeamingDurationID {
			value, ok := gpifXPropertyInt(property)
			if !ok {
				return nil
			}
			duration, ok = gpifBeamingDuration(value)
			if !ok || durationSet {
				return nil
			}
			durationSet = true
			continue
		}
		index, ok := gpifXPropertyGroupIndex(property.ID)
		if !ok {
			continue
		}
		value, valid := gpifXPropertyInt(property)
		if !valid || value < 0 || value > math.MaxInt32 {
			return nil
		}
		if _, duplicate := groups[index]; duplicate {
			return nil
		}
		groups[index] = int(value)
		maxIndex = max(maxIndex, index)
	}
	if !durationSet || maxIndex < 0 {
		return nil
	}
	for maxIndex >= 0 && groups[maxIndex] == 0 {
		maxIndex--
	}
	if maxIndex < 0 {
		return nil
	}
	result := &BeamingRules{Duration: duration, Groups: make([]int, maxIndex+1)}
	for index := range result.Groups {
		value, ok := groups[index]
		if !ok || value <= 0 {
			return nil
		}
		result.Groups[index] = value
	}
	return result
}

func gpifApplyBeaming(b *gpifBeat, beat *Beat) {
	if b == nil || beat == nil {
		return
	}
	if b.XProperties != nil {
		for _, property := range b.XProperties.Properties {
			value, ok := gpifXPropertyInt(property)
			if !ok {
				continue
			}
			switch property.ID {
			case gpifBeatBeamingModeID:
				switch value {
				case 1:
					beat.BeamingMode = BeatBeamingForceMerge
				case 2:
					beat.BeamingMode = BeatBeamingForceSplit
				}
			case gpifBeatSecondarySplitID:
				if value == 1 && beat.BeamingMode != BeatBeamingForceSplit {
					beat.BeamingMode = BeatBeamingForceSplitSecondary
				}
			case gpifBeatBeamDirectionInvertID:
				beat.InvertBeamDirection = value == 1
			}
		}
	}
	beat.PreferredBeamDirection = gpifStemDirection(b.TransposedPitchStemOrientation)
	if b.UserTransposedPitchStemOrientation != "" {
		// AlphaTab treats Undefined and unknown user overrides as absent, so a
		// valid transposed-pitch orientation remains authoritative.
		if direction := gpifStemDirection(b.UserTransposedPitchStemOrientation); direction != VoiceDirectionNone {
			beat.PreferredBeamDirection = direction
		}
	}
}

func gpifStemDirection(value string) VoiceDirection {
	switch value {
	case "Upward":
		return VoiceDirectionUp
	case "Downward":
		return VoiceDirectionDown
	default:
		return VoiceDirectionNone
	}
}

func gp8BeamingRules(rules *BeamingRules) *gpifXProperties {
	if rules == nil {
		return nil
	}
	properties := []gpifXProperty{gp8IntXProperty(gpifMasterBarBeamingDurationID, int64(rules.Duration))}
	for index, group := range rules.Groups {
		properties = append(properties, gp8IntXProperty(strconv.FormatInt(gpifMasterBarBeamingGroupBase+int64(index), 10), int64(group)))
	}
	return &gpifXProperties{Properties: properties}
}

func gp8BeatBeaming(beat *Beat, target *gpifBeat) {
	if beat.PreferredBeamDirection != VoiceDirectionNone {
		orientation := "Upward"
		if beat.PreferredBeamDirection == VoiceDirectionDown {
			orientation = "Downward"
		}
		target.TransposedPitchStemOrientation = orientation
		target.UserTransposedPitchStemOrientation = orientation
	}
	var properties []gpifXProperty
	switch beat.BeamingMode {
	case BeatBeamingForceSplit:
		properties = append(properties, gp8IntXProperty(gpifBeatBeamingModeID, 2))
	case BeatBeamingForceMerge:
		properties = append(properties, gp8IntXProperty(gpifBeatBeamingModeID, 1))
	case BeatBeamingForceSplitSecondary:
		properties = append(properties, gp8IntXProperty(gpifBeatSecondarySplitID, 1))
	}
	if beat.InvertBeamDirection {
		properties = append(properties, gp8IntXProperty(gpifBeatBeamDirectionInvertID, 1))
	}
	if len(properties) > 0 {
		if target.XProperties == nil {
			target.XProperties = &gpifXProperties{}
		}
		target.XProperties.Properties = append(target.XProperties.Properties, properties...)
	}
}

func gp8IntXProperty(id string, value int64) gpifXProperty {
	text := strconv.FormatInt(value, 10)
	return gpifXProperty{ID: id, Int: &text}
}

func validateBeamingRules(rules *BeamingRules) error {
	if rules == nil {
		return nil
	}
	if _, ok := gpifBeamingDuration(int64(rules.Duration)); !ok {
		return fmt.Errorf("duration %d is not a supported note-value denominator", rules.Duration)
	}
	if len(rules.Groups) == 0 || len(rules.Groups) > int(gpifMasterBarBeamingGroupLast-gpifMasterBarBeamingGroupBase+1) {
		return fmt.Errorf("group count %d is outside 1..32", len(rules.Groups))
	}
	for index, group := range rules.Groups {
		if group <= 0 || int64(group) > math.MaxInt32 {
			return fmt.Errorf("group %d value %d is outside 1..%d", index, group, math.MaxInt32)
		}
	}
	return nil
}

func (builder *gp8Builder) reportLegacyBeatDisplay(display BeatDisplay, location ScoreLocation) {
	values := []struct {
		nonDefault bool
		code       string
		reason     string
	}{
		{display.BreakBeam, "gp8.omit.beat-display-break-beam", "legacy GP5 break-beam flags are source-adjacent compatibility data; edit Beat.BeamingMode to author the connection"},
		{display.ForceBeam, "gp8.omit.beat-display-force-beam", "legacy GP5 force-beam flags are source-adjacent compatibility data; edit Beat.BeamingMode to author the connection"},
		{display.BeamDirection != VoiceDirectionNone, "gp8.omit.beat-display-beam-direction", "legacy GP5 beam direction is source-compatible raw data; edit Beat.PreferredBeamDirection to author the stem direction"},
		{display.TupletBracket != TupletBracketNone, "gp8.omit.beat-display-tuplet-bracket", "GP8 GPIF does not represent the legacy tuplet bracket boundary"},
		{display.BreakSecondary != 0, "gp8.omit.beat-display-break-secondary", "legacy GP5 secondary-break bytes are source-adjacent compatibility data; edit Beat.BeamingMode to author the connection"},
		{display.BreakSecondaryTuplet, "gp8.omit.beat-display-break-secondary-tuplet", "GP8 GPIF does not represent the legacy secondary tuplet break flag"},
		{display.ForceBracket, "gp8.omit.beat-display-force-bracket", "GP8 GPIF does not represent the legacy force-bracket flag"},
	}
	for _, value := range values {
		if value.nonDefault {
			builder.addReport(value.code, "beaming", ExportDispositionOmitted, location, value.reason)
		}
	}
}
