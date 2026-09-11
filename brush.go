// SPDX-License-Identifier: MIT

package goguitarpro

import "math"

const gpifBeatBrushDurationID = "687935489"

type resolvedBeatStroke struct {
	kind          BeatStrokeKind
	direction     BeatStrokeDirection
	ticks         int64
	emitDuration  bool
	conflict      bool
	targetLimited bool
}

func (stroke BeatStroke) resolved() resolvedBeatStroke {
	result := resolvedBeatStroke{kind: stroke.Kind, direction: stroke.Direction}
	if result.kind == BeatStrokeKindNone && result.direction != BeatStrokeDirectionNone {
		result.kind = BeatStrokeKindArpeggio
	}
	legacyTicks, legacyOK := beatStrokeDurationTicks(stroke.Duration)
	if stroke.ExactDuration != nil {
		exactTicks, exactOK := gp8StrokeTicks(*stroke.ExactDuration)
		legacyEdited := stroke.hasImported && stroke.Duration != stroke.importedDuration
		if legacyEdited {
			result.ticks, result.emitDuration = legacyTicks, legacyOK
			result.conflict = !exactOK || exactTicks != legacyTicks
		} else {
			result.ticks, result.emitDuration = exactTicks, exactOK
			result.targetLimited = !exactOK
		}
		return result
	}
	result.ticks, result.emitDuration = legacyTicks, legacyOK
	if stroke.hasImported && !stroke.importedExact && stroke.Duration == stroke.importedDuration && stroke.importedDuration != 0 {
		result.emitDuration = false
	}
	return result
}

func beatStrokeDurationTicks(value NoteValue) (int64, bool) {
	switch value {
	case 1, 2, 4, 8, 16, 32, 64, 128:
		return DurationQuarterTime * 4 / int64(value), true
	default:
		return 0, false
	}
}

func beatStrokeNoteValue(ticks int64) (NoteValue, bool) {
	if ticks <= 0 || (DurationQuarterTime*4)%ticks != 0 {
		return 0, false
	}
	value := NoteValue(DurationQuarterTime * 4 / ticks)
	_, ok := beatStrokeDurationTicks(value)
	return value, ok
}

func gp8StrokeTicks(value ScoreTime) (int64, bool) {
	if value.Denominator() != 1 || value.Numerator() < 0 || value.Numerator() > math.MaxInt32 {
		return 0, false
	}
	return value.Numerator(), true
}

func gpifApplyBrushDuration(source *gpifBeat, stroke *BeatStroke) {
	if source == nil || source.XProperties == nil {
		return
	}
	for _, property := range source.XProperties.Properties {
		if property.ID != gpifBeatBrushDurationID {
			continue
		}
		value, ok := gpifXPropertyInt(property)
		if !ok || value < 0 || value > math.MaxInt32 {
			continue
		}
		exact, _ := NewScoreTime(value, 1)
		stroke.ExactDuration = &exact
		stroke.importedExact = true
		if duration, represented := beatStrokeNoteValue(value); represented {
			stroke.Duration = duration
		}
	}
}

func gpifAuditBrush(context *parseContext, beat *gpifBeat, path string) {
	if beat == nil {
		return
	}
	brushProperties := 0
	for _, property := range beat.Properties.Properties {
		if property.Name == "Brush" {
			brushProperties++
		}
	}
	if beat.Arpeggio != "" && brushProperties > 0 {
		context.add(diagnosticSource("GPIF.Beat.Brush.Kind.Conflict", "brush", ParseDiagnosticInvalidData), ParseDiagnostic{SourcePath: path + "/Properties", ObjectID: beat.ID, Location: ParseLocation{BeatID: beat.ID}, Reason: "beat contains both an Arpeggio element and a Brush property"})
	}
	if brushProperties > 1 {
		context.add(diagnosticSource("GPIF.Beat.Brush.Property.Duplicate", "brush", ParseDiagnosticInvalidData), ParseDiagnostic{SourcePath: path + "/Properties", ObjectID: beat.ID, Location: ParseLocation{BeatID: beat.ID}, Reason: "beat contains duplicate Brush properties"})
	}
	hasTiming := false
	if beat.XProperties != nil {
		for _, property := range beat.XProperties.Properties {
			hasTiming = hasTiming || property.ID == gpifBeatBrushDurationID
		}
	}
	if hasTiming && beat.Arpeggio == "" && brushProperties == 0 {
		context.add(diagnosticSource("GPIF.Beat.Brush.Duration.Orphan", "brush", ParseDiagnosticInvalidData), ParseDiagnostic{SourcePath: path + "/XProperties", ObjectID: beat.ID, Location: ParseLocation{BeatID: beat.ID}, Reason: "brush duration has no Brush or Arpeggio owner"})
	}
}
