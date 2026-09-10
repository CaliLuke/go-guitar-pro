// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func gpifAuditMasterAutomations(automations []gpifAutomation, context *parseContext) {
	for index, automation := range automations {
		switch automation.Type {
		case "Tempo":
			if strings.TrimSpace(automation.Value.Text) == "" {
				context.add(gpifTempoAutomationInvalidSource, ParseDiagnostic{
					SourcePath: fmt.Sprintf("/GPIF/MasterTrack/Automations/Automation[%d]/Value", index),
					Feature:    "tempo",
					Reason:     "tempo automation value is missing",
				})
			}
		case "SyncPoint":
			gpifAuditSyncPointAutomation(automation, index, context)
		default:
			context.add(diagnosticSource("GPIF.MasterTrack.Automation.Type.Unknown", "score-core", ParseDiagnosticUnknownSyntax), ParseDiagnostic{
				SourcePath: fmt.Sprintf("/GPIF/MasterTrack/Automations/Automation[%d]/Type", index),
				ObjectID:   automation.Type,
				Reason:     fmt.Sprintf("master-track automation type %q is not recognized", automation.Type),
			})
		}
	}
}

func gpifAuditSyncPointAutomation(automation gpifAutomation, index int, context *parseContext) {
	path := fmt.Sprintf("/GPIF/MasterTrack/Automations/Automation[%d]", index)
	invalid := func(reason string) {
		context.add(diagnosticSource("GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{
			SourcePath: path + "/Value",
			Reason:     reason,
		})
	}
	frameOffsetText := strings.TrimSpace(automation.Value.FrameOffset)
	if frameOffsetText == "" {
		frameOffsetText = strings.TrimSpace(automation.Value.Text)
	}
	frameOffset, err := strconv.ParseInt(frameOffsetText, 10, 64)
	if err != nil || frameOffset < 0 {
		invalid(fmt.Sprintf("sync-point frame offset %q must be a non-negative integer", frameOffsetText))
		return
	}
	if _, err := NewBarPositionFromFloat64(automation.Position); err != nil {
		invalid(fmt.Sprintf("sync-point position %v must be finite and within 0..1", automation.Position))
		return
	}
	bar := automation.Bar
	if text := strings.TrimSpace(automation.Value.BarIndex); text != "" {
		parsed, parseErr := strconv.Atoi(text)
		if parseErr != nil {
			invalid(fmt.Sprintf("sync-point bar index %q must be a non-negative integer", automation.Value.BarIndex))
			return
		}
		bar = parsed
	}
	if bar < 0 {
		invalid(fmt.Sprintf("sync-point bar index %d must be non-negative", bar))
		return
	}
	if text := strings.TrimSpace(automation.Value.BarOccurrence); text != "" {
		value, parseErr := strconv.Atoi(text)
		if parseErr != nil || value < 0 {
			invalid(fmt.Sprintf("sync-point bar occurrence %q must be a non-negative integer", automation.Value.BarOccurrence))
			return
		}
	}
	for _, value := range []struct{ name, text string }{
		{name: "modified tempo", text: automation.Value.ModifiedTempo},
		{name: "original tempo", text: automation.Value.OriginalTempo},
	} {
		name, text := value.name, value.text
		if strings.TrimSpace(text) == "" {
			continue
		}
		value, parseErr := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if parseErr != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			invalid(fmt.Sprintf("sync-point %s %q must be finite", name, text))
			return
		}
	}
}

func gpifAuditTrackAutomations(track gpifTrack, context *parseContext) {
	for index, automation := range track.Automations.Automations {
		path := fmt.Sprintf("/GPIF/Tracks/Track[@id=%q]/Automations/Automation[%d]", track.ID, index)
		switch automation.Type {
		case "Sound":
			resolved := false
			for _, sound := range track.Sounds.Sounds {
				if automation.Value.Text == sound.Path+";"+sound.Name+";"+sound.Role {
					resolved = true
					break
				}
			}
			if !resolved {
				context.add(diagnosticSource("GPIF.Track.Automation.Sound.Reference", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{
					SourcePath: path + "/Value", ObjectID: track.ID,
					Reason: fmt.Sprintf("sound automation reference %q does not resolve in its track", automation.Value.Text),
				})
			}
		case "SustainPedal":
			context.add(diagnosticSource("GPIF.Track.Automation.SustainPedal", "score-core", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				SourcePath: path + "/Type/SustainPedal", ObjectID: track.ID,
				Reason: "sustain-pedal automation has no Song destination",
			})
		default:
			context.add(diagnosticSource("GPIF.Track.Automation.Type.Unknown", "score-core", ParseDiagnosticUnknownSyntax), ParseDiagnostic{
				SourcePath: path + "/Type", ObjectID: track.ID,
				Reason: fmt.Sprintf("track automation type %q is not recognized", automation.Type),
			})
		}
	}
}

func gpifAuditChannelStripAutomations(automations []gpifAutomation, trackID string, context *parseContext) {
	for index, automation := range automations {
		path := fmt.Sprintf("/GPIF/Tracks/Track[@id=%q]/RSE/ChannelStrip/Automations/Automation[%d]", trackID, index)
		switch automation.Type {
		case "DSPParam_12":
			value, err := strconv.ParseFloat(strings.TrimSpace(automation.Value.Text), 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				context.add(diagnosticSource("GPIF.ChannelStrip.Automation.Volume.Value.Invalid", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{
					SourcePath: path + "/Value", ObjectID: trackID,
					Reason: fmt.Sprintf("volume automation value %q must be finite", automation.Value.Text),
				})
			} else if automation.Position < 0 || automation.Position > 1 || value < 0 || value > 1 {
				context.add(diagnosticSource("GPIF.ChannelStrip.Automation.Volume.Range.Invalid", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{
					SourcePath: path, ObjectID: trackID,
					Reason: fmt.Sprintf("volume automation position %v and value %v must be within 0..1", automation.Position, value),
				})
			}
		case "DSPParam_00", "DSPParam_01", "DSPParam_11":
			context.add(diagnosticSource("GPIF.ChannelStrip.Automation.Unsupported", "score-core", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
				SourcePath: path + "/Type", ObjectID: trackID,
				Reason: fmt.Sprintf("channel-strip automation type %q has no Song destination", automation.Type),
			})
		default:
			context.add(diagnosticSource("GPIF.ChannelStrip.Automation.Type.Unknown", "score-core", ParseDiagnosticUnknownSyntax), ParseDiagnostic{
				SourcePath: path + "/Type", ObjectID: trackID,
				Reason: fmt.Sprintf("channel-strip automation type %q is not recognized", automation.Type),
			})
		}
	}
}

func gpifMIDIChannel(port, channel int) uint8 {
	value := port*16 + channel
	if value < 0 {
		return 0
	}
	if value > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(value)
}

func gpifApplyChannelStrip(parameters string, channel *MidiChannel) {
	values := strings.Fields(parameters)
	if len(values) <= 12 {
		return
	}
	balance, balanceErr := strconv.ParseFloat(values[11], 64)
	if balanceErr == nil {
		channel.Balance = int8(math.Round(min(1, max(0, balance)) * 127))
	}
	volume, volumeErr := strconv.ParseFloat(values[12], 64)
	if volumeErr == nil {
		channel.Volume = int8(math.Round(min(1, max(0, volume)) * 127))
	}
}

func gpifReadVolumeAutomations(
	automations []gpifAutomation,
	trackIndex int,
	song *Song,
) {
	for _, automation := range automations {
		if automation.Type != "DSPParam_12" {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(automation.Value.Text), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || automation.Position < 0 || automation.Position > 1 || value < 0 || value > 1 {
			continue
		}
		song.VolumeAutomations = append(song.VolumeAutomations, VolumeAutomation{
			Track:    trackIndex,
			Bar:      automation.Bar,
			Position: automation.Position,
			Value:    value,
			Linear:   automation.Linear,
		})
	}
}

func gpifReadBackingTrack(doc gpifDocument, song *Song) {
	if doc.BackingTrack == nil {
		return
	}
	framePadding, _ := strconv.ParseInt(strings.TrimSpace(doc.BackingTrack.FramePadding), 10, 64)
	backingTrack := &BackingTrack{
		Name:         doc.BackingTrack.Name,
		Source:       doc.BackingTrack.Source,
		AssetID:      doc.BackingTrack.AssetID,
		FramePadding: framePadding,
		Enabled:      doc.BackingTrack.Enabled,
	}
	for _, asset := range doc.Assets.Assets {
		if asset.ID != backingTrack.AssetID {
			continue
		}
		backingTrack.OriginalFilePath = asset.OriginalFilePath
		backingTrack.OriginalFileSHA1 = asset.OriginalFileSHA1
		backingTrack.EmbeddedFilePath = asset.EmbeddedFilePath
		break
	}
	song.BackingTrack = backingTrack
}

func gpifReadSyncPoints(automations []gpifAutomation, song *Song) {
	framePadding := int64(0)
	if song.BackingTrack != nil {
		framePadding = song.BackingTrack.FramePadding
	}
	for _, automation := range automations {
		if automation.Type != "SyncPoint" {
			continue
		}
		frameOffsetText := strings.TrimSpace(automation.Value.FrameOffset)
		if frameOffsetText == "" {
			frameOffsetText = strings.TrimSpace(automation.Value.Text)
		}
		frameOffset, err := strconv.ParseInt(frameOffsetText, 10, 64)
		if err != nil {
			continue
		}
		audioFrame, err := NewAudioFrame(frameOffset)
		if err != nil {
			continue
		}
		barPosition, err := NewBarPositionFromFloat64(automation.Position)
		if err != nil {
			continue
		}
		bar := automation.Bar
		if value := strings.TrimSpace(automation.Value.BarIndex); value != "" {
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				bar = parsed
			}
		}
		barOccurrence, _ := strconv.Atoi(strings.TrimSpace(automation.Value.BarOccurrence))
		modifiedTempo, _ := strconv.ParseFloat(strings.TrimSpace(automation.Value.ModifiedTempo), 64)
		originalTempo, _ := strconv.ParseFloat(strings.TrimSpace(automation.Value.OriginalTempo), 64)
		song.SyncPoints = append(song.SyncPoints, SyncPoint{
			Bar:           bar,
			Position:      automation.Position,
			BarPosition:   barPosition,
			BarOccurrence: barOccurrence,
			FrameOffset:   frameOffset,
			AudioFrame:    audioFrame,
			MediaTimeMS:   (float64(frameOffset) - float64(framePadding)) / GPIFBackingTrackSampleRate * 1000,
			ModifiedTempo: modifiedTempo,
			OriginalTempo: originalTempo,
			Linear:        automation.Linear,
			Visible:       automation.Visible == "" || strings.EqualFold(automation.Visible, "true"),
		})
	}
}

func gpifReadTempoAutomations(automations []gpifAutomation, song *Song, context *parseContext) {
	earliest := -1
	for index, auto := range automations {
		if auto.Type != "Tempo" {
			continue
		}
		parts := strings.Fields(auto.Value.Text)
		if len(parts) == 0 {
			continue
		}
		tempo, err := strconv.ParseFloat(parts[0], 64)
		bpm, bpmErr := NewBPM(tempo)
		if err != nil || bpmErr != nil {
			if song.InitialTempo.State == SourceValueMissing {
				song.InitialTempo = UnknownSourceValue[BPM](parts[0])
			}
			context.add(gpifTempoAutomationInvalidSource, ParseDiagnostic{
				SourcePath: fmt.Sprintf("/GPIF/MasterTrack/Automations/Automation[%d]/Value", index),
				Feature:    "tempo",
				Reason:     fmt.Sprintf("tempo %q must be finite and positive", parts[0]),
			})
			continue
		}
		tempo = float64(bpm) * gpifTempoReferenceFactor(parts)
		bpm, bpmErr = NewBPM(tempo)
		if bpmErr != nil {
			context.add(diagnosticSource("GPIF.MasterTrack.Automation.Tempo.Reference.Invalid", "tempo-automations", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: fmt.Sprintf("/GPIF/MasterTrack/Automations/Automation[%d]/Value", index),
				Feature:    "tempo",
				Reason:     bpmErr.Error(),
			})
			continue
		}
		change := TempoAutomation{
			Bar: auto.Bar, Position: auto.Position, Tempo: tempo,
			Linear: auto.Linear, Text: auto.Text, Hidden: !gpifAutomationVisible(auto.Visible),
		}
		song.TempoAutomations = append(song.TempoAutomations, change)
		if earliest < 0 || gpifAutomationIsBefore(change, song.TempoAutomations[earliest]) {
			earliest = len(song.TempoAutomations) - 1
			song.InitialTempo = KnownSourceValue(bpm)
			legacyTempo, legacyErr := bpm.LegacyTempo()
			if legacyErr != nil {
				context.add(diagnosticSource("GPIF.MasterTrack.Automation.Tempo.LegacyOverflow", "tempo-automations", ParseDiagnosticLossyProjection), ParseDiagnostic{
					SourcePath: fmt.Sprintf("/GPIF/MasterTrack/Automations/Automation[%d]/Value", index),
					Feature:    "tempo",
					Reason:     legacyErr.Error(),
				})
			} else {
				song.Tempo = legacyTempo
			}
			song.TempoName = auto.Text
		}
	}
}

func gpifAutomationVisible(value string) bool {
	return value == "" || strings.EqualFold(value, "true")
}

func gpifTempoReferenceFactor(parts []string) float64 {
	reference := 1
	if len(parts) > 1 {
		value := parts[1]
		end := 0
		if value != "" && (value[0] == '+' || value[0] == '-') {
			end++
		}
		digitStart := end
		for end < len(value) && value[end] >= '0' && value[end] <= '9' {
			end++
		}
		parsed, err := strconv.Atoi(value[:end])
		if end == digitStart {
			err = strconv.ErrSyntax
		}
		if err != nil || parsed < 1 || parsed > 5 {
			reference = 2
		} else {
			reference = parsed
		}
	}
	return [...]float64{0, 0.5, 1, 1.5, 2, 3}[reference]
}

func gpifAutomationIsBefore(a, b TempoAutomation) bool {
	if a.Bar != b.Bar {
		return a.Bar < b.Bar
	}
	return a.Position < b.Position
}
