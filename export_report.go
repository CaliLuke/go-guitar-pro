// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"slices"
)

// ExportDisposition describes how one source value is handled by a target writer.
type ExportDisposition string

const (
	// ExportDispositionNormalized means the writer emits a documented equivalent value.
	ExportDispositionNormalized ExportDisposition = "normalized"
	// ExportDispositionOmitted means the writer does not emit the source value.
	ExportDispositionOmitted ExportDisposition = "omitted"
	// ExportDispositionRejected means the source cannot be serialized.
	ExportDispositionRejected ExportDisposition = "rejected"
)

// ExportReportEntry describes one target-specific conversion decision.
type ExportReportEntry struct {
	Code        string
	Feature     string
	Disposition ExportDisposition
	Location    ScoreLocation
	Reason      string
}

// ExportReport contains the conversion decisions for one target.
type ExportReport struct {
	Target  ExportFormat
	Entries []ExportReportEntry
}

// ExportLossPolicy controls opt-in refusal of normalized or omitted content.
type ExportLossPolicy struct {
	RequirePreservation bool
	// AllowedCodes permits only the named stable report entries when preservation is required.
	AllowedCodes []string
}

// ExportLossError reports entries refused by an export loss policy.
type ExportLossError struct {
	Entries []ExportReportEntry
}

// Error returns a concise export-policy failure message.
func (e *ExportLossError) Error() string {
	return fmt.Sprintf("export loss policy rejected %d conversion(s)", len(e.Entries))
}

// PreflightExport reports target-specific rejections, normalizations, and omissions.
// It does not mutate the song and does not produce an output artifact.
func PreflightExport(song *Song, target ExportFormat, options ExportOptions) ExportReport {
	report := ExportReport{Target: target}
	add := func(code, feature string, disposition ExportDisposition, location ScoreLocation, reason string) {
		report.Entries = append(report.Entries, ExportReportEntry{Code: code, Feature: feature, Disposition: disposition, Location: location, Reason: reason})
	}
	if song == nil {
		add("export.reject.nil-song", "score-core", ExportDispositionRejected, ScoreLocation{}, "song is nil")
		return report
	}
	if target != ExportFormatGP8 {
		add("export.reject.target", "score-core", ExportDispositionRejected, ScoreLocation{}, fmt.Sprintf("unsupported target %d", target))
		return report
	}
	if err := validateGP8Song(song); err != nil {
		add("gp8.reject.score", "score-core", ExportDispositionRejected, ScoreLocation{}, err.Error())
		return report
	}
	if err := validateGP8ExportOptions(options.GP8); err != nil {
		add("gp8.reject.options", "score-core", ExportDispositionRejected, ScoreLocation{}, err.Error())
		return report
	}
	if song.BackingTrack != nil {
		add("gp8.omit.backing-track", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer does not emit backing-track assets")
	}
	for _, point := range song.SyncPoints {
		add("gp8.omit.sync-points", "score-core", ExportDispositionOmitted, ScoreLocation{Measure: point.Bar}, "GP8 writer does not emit this backing-track sync point")
	}
	for _, automation := range song.VolumeAutomations {
		add("gp8.omit.volume-automations", "score-core", ExportDispositionOmitted, ScoreLocation{Track: automation.Track, Measure: automation.Bar}, "GP8 writer does not emit this track volume automation")
	}
	for trackIndex := range song.Tracks {
		for staffIndex, staff := range gp8ExportStaves(&song.Tracks[trackIndex]) {
			for measureIndex := range staff.Measures {
				for voiceIndex := range staff.Measures[measureIndex].Voices {
					for beatIndex, beat := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
						location := ScoreLocation{Track: trackIndex, Staff: staffIndex, Measure: measureIndex, Voice: voiceIndex, Beat: beatIndex}
						if beat.Status == BeatStatusEmpty {
							add("gp8.normalize.empty-beat", "note-and-beat-semantics", ExportDispositionNormalized, location, "GP8 writer emits an explicit empty beat as a rest")
						}
						if len(beat.Notes) > 0 {
							velocity := gpifDynamicToVelocity(gp8VelocityToDynamic(beat.Notes[0].Velocity))
							for _, note := range beat.Notes {
								if note.Velocity != velocity {
									add("gp8.normalize.note-velocity", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF stores one quantized dynamic for all notes in a beat")
									break
								}
							}
						}
					}
				}
			}
		}
	}
	return report
}

func refusedExportEntries(report ExportReport, policy ExportLossPolicy) []ExportReportEntry {
	if !policy.RequirePreservation {
		return nil
	}
	var refused []ExportReportEntry
	for _, entry := range report.Entries {
		if entry.Disposition != ExportDispositionRejected && !slices.Contains(policy.AllowedCodes, entry.Code) {
			refused = append(refused, entry)
		}
	}
	return refused
}
