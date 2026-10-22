// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
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

// ExportReportSummary counts report entries with the same code, feature, and disposition.
// It omits occurrence locations and reasons, which remain available in ExportReport.Entries.
type ExportReportSummary struct {
	Code        string
	Feature     string
	Disposition ExportDisposition
	Count       int
}

// Summary returns a fresh count sorted lexically by Code, Feature, then Disposition.
// Entries are grouped by their exact Code, Feature, and Disposition values.
// Location and Reason are omitted; use Entries when occurrence details matter.
func (report ExportReport) Summary() []ExportReportSummary {
	type key struct {
		code        string
		feature     string
		disposition ExportDisposition
	}
	counts := make(map[key]int)
	for _, entry := range report.Entries {
		counts[key{entry.Code, entry.Feature, entry.Disposition}]++
	}
	summary := make([]ExportReportSummary, 0, len(counts))
	for item, count := range counts {
		summary = append(summary, ExportReportSummary{
			Code: item.code, Feature: item.feature, Disposition: item.disposition, Count: count,
		})
	}
	slices.SortFunc(summary, func(a, b ExportReportSummary) int {
		if order := cmp.Compare(a.Code, b.Code); order != 0 {
			return order
		}
		if order := cmp.Compare(a.Feature, b.Feature); order != 0 {
			return order
		}
		return cmp.Compare(a.Disposition, b.Disposition)
	})
	return summary
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
	prefix := fmt.Sprintf("export loss policy rejected %d conversion(s)", len(e.Entries))
	if len(e.Entries) == 0 {
		return prefix
	}
	counts := make(map[string]int)
	for _, entry := range e.Entries {
		counts[entry.Code]++
	}
	codes := make([]string, 0, len(counts))
	for code := range counts {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	parts := make([]string, 0, len(codes))
	for _, code := range codes {
		parts = append(parts, fmt.Sprintf("%s=%d", code, counts[code]))
	}
	return prefix + " (codes: " + strings.Join(parts, ", ") + ")"
}

// PreflightExport reports target-specific rejections, normalizations, and omissions.
// It does not mutate the song and does not produce an output artifact.
func PreflightExport(song *Song, target ExportFormat, options ExportOptions) ExportReport {
	report, _ := planExport(song, target, options)
	return report
}

type gp8ExportPlan struct {
	document          gpifDocument
	backingTrackAsset *gp8BackingTrackAsset
}

type gp8BackingTrackAsset struct {
	name string
	data []byte
}

func planExport(song *Song, target ExportFormat, options ExportOptions) (ExportReport, *gp8ExportPlan) {
	report := ExportReport{Target: target}
	add := func(code, feature string, disposition ExportDisposition, location ScoreLocation, reason string) {
		report.Entries = append(report.Entries, ExportReportEntry{Code: code, Feature: feature, Disposition: disposition, Location: location, Reason: reason})
	}
	if song == nil {
		add("export.reject.nil-song", "score-core", ExportDispositionRejected, ScoreLocation{}, "song is nil")
		return report, nil
	}
	if target != ExportFormatGP8 {
		add("export.reject.target", "score-core", ExportDispositionRejected, ScoreLocation{}, fmt.Sprintf("unsupported target %d", target))
		return report, nil
	}
	for _, diagnostic := range authoredScoreDiagnostics(song) {
		feature := "score-core"
		if diagnostic.Kind == ScoreDiagnosticTiming {
			feature = "rhythm"
		}
		add("gp8.reject."+diagnostic.Code, feature, ExportDispositionRejected, diagnostic.Location, diagnostic.Reason)
	}
	if len(report.Entries) != 0 {
		return report, nil
	}
	originalSong := song
	var mixConflicts []legacyMixConflict
	song, mixConflicts = projectMixTableAutomations(song)
	if song != originalSong {
		for _, diagnostic := range authoredScoreDiagnostics(song) {
			add("gp8.reject."+diagnostic.Code, "mix-table", ExportDispositionRejected, diagnostic.Location, diagnostic.Reason)
		}
	}
	if len(report.Entries) != 0 {
		return report, nil
	}
	for _, conflict := range mixConflicts {
		add("gp8.normalize.mix-table-"+conflict.controller+"-authority", "mix-table", ExportDispositionNormalized, conflict.location, fmt.Sprintf("the explicit %s automation collection (target track %d; -1 means score-wide) takes precedence over the conflicting legacy event at this source beat", conflict.controller, conflict.target))
	}
	if err := validateGP8Song(song); err != nil {
		add("gp8.reject.score", "score-core", ExportDispositionRejected, ScoreLocation{}, err.Error())
		return report, nil
	}
	if err := validateGP8ExportOptions(options.GP8); err != nil {
		add("gp8.reject.options", "score-core", ExportDispositionRejected, ScoreLocation{}, err.Error())
		return report, nil
	}
	if _, conflict, err := gp8ResolvedFieldTempo(song); err == nil && conflict {
		add("gp8.normalize.tempo-compatibility", "tempo-automations", ExportDispositionNormalized, ScoreLocation{}, "the edited score tempo representation takes precedence over its stale compatibility value")
	}
	if fieldTempo, _, err := gp8ResolvedFieldTempo(song); err == nil && fieldTempo > 0 {
		for _, automation := range song.TempoAutomations {
			if automation.Bar == 0 && automation.Position == 0 && automation.Tempo != fieldTempo {
				add("gp8.normalize.tempo-automation-authority", "tempo-automations", ExportDispositionNormalized, ScoreLocation{}, "the authored opening automation takes precedence over conflicting score tempo fields")
				break
			}
		}
	}
	if song.BackingTrack != nil && !gp8EmbedsBackingTrack(song.BackingTrack) {
		add("gp8.omit.backing-track", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer does not emit backing-track assets")
	}
	if reason := gp8ScoreLyricsLoss(song); reason != "" {
		location := ScoreLocation{}
		if song.Lyrics.TrackIndex >= 0 {
			location.Track = song.Lyrics.TrackIndex
		}
		add("gp8.omit.score-lyrics", "score-core", ExportDispositionOmitted, location, reason)
	}

	for _, point := range song.SyncPoints {
		location := ScoreLocation{Measure: point.Bar}
		if point.AudioFrame != AudioFrame(point.FrameOffset) {
			add("gp8.normalize.sync-point-frame-authority", "timing", ExportDispositionNormalized, location, "the checked audio frame takes precedence over its conflicting legacy frame offset")
		}
		if legacyPosition, err := NewBarPositionFromFloat64(point.Position); err == nil && point.BarPosition.Ratio().Compare(legacyPosition.Ratio()) != 0 {
			add("gp8.normalize.sync-point-position-authority", "timing", ExportDispositionNormalized, location, "the exact bar position takes precedence over its conflicting legacy position")
		}
		if song.BackingTrack != nil && !gp8EmbedsBackingTrack(song.BackingTrack) && song.BackingTrack.FramePadding != 0 {
			add("gp8.normalize.sync-point-consumer-padding", "sync-points", ExportDispositionNormalized, location, "the omitted backing track cannot supply frame padding; sync-point frame offsets remain exact but target media time changes")
		}
		if !gp8SyncPositionExact(point) {
			add("gp8.reject.sync-point-position-precision", "sync-points", ExportDispositionRejected, location, "the checked sync-point position cannot be retained exactly by the GPIF numeric position")
		}
		if !gp8SyncIntegerExact(int64(point.AudioFrame)) || !gp8SyncIntegerExact(int64(point.BarOccurrence)) {
			add("gp8.normalize.sync-point-consumer-integer", "sync-points", ExportDispositionNormalized, location, "GPIF retains exact frame and occurrence integers; the pinned consumer rounds these integers to binary64")
		}
	}
	for index, automation := range song.PanAutomations {
		add("gp8.omit.pan-automation-consumer", "pan-automation", ExportDispositionOmitted, ScoreLocation{Track: automation.Track, Measure: automation.Bar}, fmt.Sprintf("pinned AlphaTab ignores channel-strip pan automation[%d] at position %g; GPIF retains the event", index, automation.Position))
	}
	for index, automation := range song.VolumeAutomations {
		add("gp8.omit.volume-automation-consumer", "volume-automation", ExportDispositionOmitted, ScoreLocation{Track: automation.Track, Measure: automation.Bar}, fmt.Sprintf("pinned AlphaTab ignores channel-strip volume automation[%d] at position %g with value %g and linear=%t; GPIF retains the event", index, automation.Position, automation.Value, automation.Linear))
	}
	for trackIndex := range song.Tracks {
		for index, automation := range song.Tracks[trackIndex].SoundAutomations {
			if automation.Position > 0 {
				add("gp8.normalize.sound-automation-consumer-position", "sound-automation", ExportDispositionNormalized, ScoreLocation{Track: trackIndex, Measure: automation.Bar}, fmt.Sprintf("pinned AlphaTab applies sound automation[%d] at position %g on the first beat of its bar; GPIF and Go retain the authored position", index, automation.Position))
			}
			if automation.Hidden {
				add("gp8.omit.sound-automation-visibility", "automation-detail", ExportDispositionOmitted, ScoreLocation{Track: trackIndex, Measure: automation.Bar}, "the pinned GP8 consumer does not retain hidden visibility on an instrument-change automation")
			}
		}
	}
	document, err := buildGP8DocumentWithReport(song, options.GP8, &report)
	if err != nil {
		add("gp8.reject.conversion", "score-core", ExportDispositionRejected, ScoreLocation{}, err.Error())
		return report, nil
	}
	plan := &gp8ExportPlan{document: document}
	if gp8EmbedsBackingTrack(song.BackingTrack) {
		plan.backingTrackAsset = &gp8BackingTrackAsset{
			name: song.BackingTrack.EmbeddedFilePath,
			data: song.BackingTrack.AudioData,
		}
	}
	return report, plan
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
