// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"testing"
)

func TestConformancePublicAPIEnums(t *testing.T) {
	runConformancePublicAPIEnums(newConformanceRun(t))
}

func runConformancePublicAPIEnums(run *conformanceRun) {
	conformanceResilienceAssertExportEnums(run)
	conformanceResilienceAssertParseDiagnosticEnums(run)
	conformanceResilienceAssertScoreDiagnosticEnums(run)
}

func conformanceResilienceAssertExportEnums(run *conformanceRun) {
	t := run.t
	valid := semanticExportProbeSong(t)
	data, report, err := ExportWithReport(valid, ExportFormatGP8, ExportOptions{})
	if err != nil || len(data) == 0 {
		t.Fatalf("GP8 export = %d bytes, %v", len(data), err)
	}
	if _, err := Parse(data); err != nil {
		t.Fatalf("parse exported GP8: %v", err)
	}
	run.Enum("ExportFormat.ExportFormatGP8", report.Target, ExportFormatGP8)

	normalized := semanticExportProbeSong(t)
	normalized.Tracks[0].Measures[0].Voices[0].Beats[0].Dynamics = 16
	run.Enum(
		"ExportDisposition.ExportDispositionNormalized",
		conformanceResilienceExportEntry(t, PreflightExport(normalized, ExportFormatGP8, ExportOptions{}), "gp8.normalize.beat-dynamic").Disposition,
		ExportDispositionNormalized,
	)

	omitted := semanticExportProbeSong(t)
	omitted.Writer = "writer omitted by GP8"
	run.Enum(
		"ExportDisposition.ExportDispositionOmitted",
		conformanceResilienceExportEntry(t, PreflightExport(omitted, ExportFormatGP8, ExportOptions{}), "gp8.omit.writer").Disposition,
		ExportDispositionOmitted,
	)

	run.Enum(
		"ExportDisposition.ExportDispositionRejected",
		conformanceResilienceExportEntry(t, PreflightExport(valid, ExportFormat(99), ExportOptions{}), "export.reject.target").Disposition,
		ExportDispositionRejected,
	)
}

func conformanceResilienceAssertParseDiagnosticEnums(run *conformanceRun) {
	t := run.t
	tests := []struct {
		member        string
		kind          ParseDiagnosticKind
		code          string
		defaultReject bool
		mutate        func(string) string
	}{
		{
			member: "ParseDiagnosticInvalidData", kind: ParseDiagnosticInvalidData,
			code: "GPIF.Note.Property.ConflictingDuplicate", defaultReject: true,
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="Fret"><Fret>31</Fret></Property>`)
			},
		},
		{
			member: "ParseDiagnosticUnknownSyntax", kind: ParseDiagnosticUnknownSyntax,
			code: "GPIF.UnknownElement.NoteAndBeat", defaultReject: true,
			mutate: func(source string) string {
				return insertFirstGPIFObjectChild(t, source, "<Notes>", "</Note>", "<FutureTechnique/>")
			},
		},
		{
			member: "ParseDiagnosticUnsupportedFeature", kind: ParseDiagnosticUnsupportedFeature,
			code: "GPIF.Track.Sound.Channel", defaultReject: true,
			mutate: func(source string) string {
				return conformanceSourceAuditReplaceElementTextAfter(t, source, "<Sounds>", "PrimaryChannel", "0")
			},
		},
		{
			member: "ParseDiagnosticLossyProjection", kind: ParseDiagnosticLossyProjection,
			code: "GPIF.Note.Property.HopoDestination", defaultReject: true,
			mutate: func(source string) string {
				return insertFirstNoteProperty(t, source, `<Property name="HopoDestination"><Enable>true</Enable></Property>`)
			},
		},
		{
			member: "ParseDiagnosticDeliberateIgnore", kind: ParseDiagnosticDeliberateIgnore,
			code: "GPIF.Beat.Property.PrimaryPickupVolume", defaultReject: false,
			mutate: func(source string) string {
				return insertFirstGPIFObjectChild(t, source, "<Beats>", "</Beat>", `<Properties><Property name="PrimaryPickupVolume"><Float>0.5</Float></Property></Properties>`)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.member, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, test.mutate)
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			diagnostic := conformanceResilienceParseDiagnostic(t, result.Diagnostics, test.code)
			run.Enum("ParseDiagnosticKind."+test.member, diagnostic.Kind, test.kind)

			_, defaultErr := ParseWithOptions(data, ParseOptions{Strict: true})
			if (defaultErr != nil) != test.defaultReject {
				t.Fatalf("default strict error = %v, want reject %t", defaultErr, test.defaultReject)
			}
			_, selectedErr := ParseWithOptions(data, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{test.kind}})
			var strictErr *StrictParseError
			if !errors.As(selectedErr, &strictErr) || conformanceResilienceParseDiagnostic(t, strictErr.Diagnostics, test.code).Kind != test.kind {
				t.Fatalf("selected strict error = %v, want %s refusal", selectedErr, test.kind)
			}
		})
	}

	// No source construct currently emits a deliberate-default diagnostic. Its
	// public behavior is the strict-policy distinction: default strict mode
	// accepts it, while explicitly selecting the kind rejects it.
	deliberateDefault := ParseDiagnostic{Code: "review.deliberate-default", Kind: ParseDiagnosticDeliberateDefault}
	if got := strictDiagnostics([]ParseDiagnostic{deliberateDefault}, nil); len(got) != 0 {
		t.Fatalf("default strict policy rejected deliberate default: %#v", got)
	}
	selected := strictDiagnostics([]ParseDiagnostic{deliberateDefault}, []ParseDiagnosticKind{ParseDiagnosticDeliberateDefault})
	if len(selected) != 1 {
		t.Fatalf("selected deliberate-default diagnostics = %#v, want one", selected)
	}
	run.Enum("ParseDiagnosticKind.ParseDiagnosticDeliberateDefault", selected[0].Kind, ParseDiagnosticDeliberateDefault)
}

func conformanceResilienceAssertScoreDiagnosticEnums(run *conformanceRun) {
	tests := []struct {
		member string
		kind   ScoreDiagnosticKind
		code   string
		mutate func(*Song)
	}{
		{
			member: "ScoreDiagnosticStructural", kind: ScoreDiagnosticStructural, code: "score.track.channel-reference",
			mutate: func(song *Song) { song.Tracks[0].ChannelIndex = len(song.Channels) },
		},
		{
			member: "ScoreDiagnosticTiming", kind: ScoreDiagnosticTiming, code: "score.beat.duration",
			mutate: func(song *Song) {
				beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
				beat.Duration.TupletEnters = 3
				beat.Duration.TupletTimes = 0
			},
		},
		{
			member: "ScoreDiagnosticValue", kind: ScoreDiagnosticValue, code: "score.channel.bank",
			mutate: func(song *Song) { song.Channels[0].Bank = 16384 },
		},
	}
	for _, test := range tests {
		run.t.Run(test.member, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			test.mutate(song)
			diagnostic := conformanceResilienceScoreDiagnostic(t, ValidateSong(song), test.code)
			run.Enum("ScoreDiagnosticKind."+test.member, diagnostic.Kind, test.kind)
		})
	}
}

func conformanceResilienceExportEntry(t *testing.T, report ExportReport, code string) *ExportReportEntry {
	t.Helper()
	for index := range report.Entries {
		if report.Entries[index].Code == code {
			return &report.Entries[index]
		}
	}
	t.Fatalf("export report = %#v, want %s", report.Entries, code)
	return nil
}

func conformanceResilienceParseDiagnostic(t *testing.T, diagnostics []ParseDiagnostic, code string) *ParseDiagnostic {
	t.Helper()
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	t.Fatalf("parse diagnostics = %#v, want %s", diagnostics, code)
	return nil
}

func conformanceResilienceScoreDiagnostic(t *testing.T, diagnostics []ScoreDiagnostic, code string) *ScoreDiagnostic {
	t.Helper()
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	t.Fatalf("score diagnostics = %#v, want %s", diagnostics, code)
	return nil
}
