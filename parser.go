// SPDX-License-Identifier: MIT

// Package goguitarpro parses Guitar Pro 3 through 8 and exports native Guitar Pro 8 files.
package goguitarpro

import (
	"fmt"
	"slices"
)

// ParseDiagnosticKind classifies a parse diagnostic independently from its semantic feature.
type ParseDiagnosticKind string

const (
	// ParseDiagnosticInvalidData reports malformed values or broken object references.
	ParseDiagnosticInvalidData ParseDiagnosticKind = "invalid-data"
	// ParseDiagnosticUnknownSyntax reports source syntax that the parser does not recognize.
	ParseDiagnosticUnknownSyntax ParseDiagnosticKind = "unknown-syntax"
	// ParseDiagnosticUnsupportedFeature reports recognized source content with no lossless destination.
	ParseDiagnosticUnsupportedFeature ParseDiagnosticKind = "unsupported-feature"
	// ParseDiagnosticLossyProjection reports content represented less precisely by the legacy Song model.
	ParseDiagnosticLossyProjection ParseDiagnosticKind = "lossy-projection"
	// ParseDiagnosticDeliberateDefault reports an intentional default applied to absent source content.
	ParseDiagnosticDeliberateDefault ParseDiagnosticKind = "deliberate-default"
	// ParseDiagnosticDeliberateIgnore records source content that is intentionally not part of Song.
	ParseDiagnosticDeliberateIgnore ParseDiagnosticKind = "deliberate-ignore"
)

// ParseLocation identifies the GPIF objects associated with a diagnostic.
type ParseLocation struct {
	TrackID string
	BarID   string
	VoiceID string
	BeatID  string
	NoteID  string
}

// ParseDiagnostic describes source content that was invalid, unknown, unsupported, or lossy.
type ParseDiagnostic struct {
	// Code identifies the audited source construct.
	Code         string
	Kind         ParseDiagnosticKind
	Format       string
	SourcePath   string
	ObjectID     string
	Location     ParseLocation
	Feature      string
	Reason       string
	BinaryOffset *int64
}

// ParseOptions controls optional parsing diagnostics and policy.
type ParseOptions struct {
	// Strict rejects invalid, unknown, unsupported, and lossy content after returning its diagnostics.
	Strict bool
	// StrictKinds limits strict rejection to the listed diagnostic kinds. An empty slice rejects all loss-bearing kinds.
	StrictKinds []ParseDiagnosticKind
}

// ParseResult contains the parsed song and all diagnostics emitted for the input.
type ParseResult struct {
	Song        *Song
	Diagnostics []ParseDiagnostic
}

// StrictParseError reports diagnostics rejected by strict parsing policy.
type StrictParseError struct {
	Diagnostics []ParseDiagnostic
}

// Error returns the strict parsing failure message.
func (e *StrictParseError) Error() string {
	return fmt.Sprintf("strict parsing rejected %d diagnostic(s)", len(e.Diagnostics))
}

type parseContext struct {
	format      string
	diagnostics []ParseDiagnostic
}

type parseDiagnosticSource struct {
	code    string
	feature string
	kind    ParseDiagnosticKind
}

func diagnosticSource(code, feature string, kind ParseDiagnosticKind) parseDiagnosticSource {
	return parseDiagnosticSource{code: code, feature: feature, kind: kind}
}

func (c *parseContext) add(source parseDiagnosticSource, diagnostic ParseDiagnostic) {
	if c == nil {
		return
	}
	diagnostic.Code = source.code
	diagnostic.Kind = source.kind
	diagnostic.Feature = source.feature
	if diagnostic.Format == "" {
		diagnostic.Format = c.format
	}
	c.diagnostics = append(c.diagnostics, diagnostic)
}

func (c *parseContext) setFormat(format string) {
	if c == nil {
		return
	}
	c.format = format
	for index := range c.diagnostics {
		c.diagnostics[index].Format = format
	}
}

func strictDiagnostics(diagnostics []ParseDiagnostic, kinds []ParseDiagnosticKind) []ParseDiagnostic {
	strict := make([]ParseDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if (diagnostic.Kind == ParseDiagnosticDeliberateDefault || diagnostic.Kind == ParseDiagnosticDeliberateIgnore) && len(kinds) == 0 {
			continue
		}
		if len(kinds) == 0 || slices.Contains(kinds, diagnostic.Kind) {
			strict = append(strict, diagnostic)
		}
	}
	return strict
}

// ParseError reports an unsupported or damaged Guitar Pro file.
type ParseError struct {
	Err error
}

// Error returns the parser error message.
func (e *ParseError) Error() string {
	return "unparseable Guitar Pro file: " + e.Err.Error()
}

// Unwrap returns the underlying parser error.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// Parse detects the file format. Then Parse parses a supported Guitar Pro file.
// Parse supports GP3, GP4, GP5 (binary), GP6/GPX, GP7, and GP8.
// Parse returns a [ParseError] for unsupported or damaged input.
func Parse(data []byte) (*Song, error) {
	result, err := ParseWithOptions(data, ParseOptions{})
	if err != nil {
		return nil, err
	}
	return result.Song, nil
}

// ParseWithOptions detects the file format and returns a song with structured diagnostics.
// The zero-value options preserve the permissive behavior of [Parse].
func ParseWithOptions(data []byte, options ParseOptions) (result *ParseResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result, err = nil, &ParseError{Err: fmt.Errorf("reader failed on malformed input: %v", recovered)}
		}
	}()

	context := &parseContext{}
	song, parseErr := parseWithContext(data, context)
	if parseErr != nil {
		return nil, &ParseError{Err: parseErr}
	}
	for _, diagnostic := range ValidateSong(song) {
		kind := ParseDiagnosticInvalidData
		feature := "score-core"
		if diagnostic.Kind == ScoreDiagnosticTiming {
			feature = "timing"
		}
		context.diagnostics = append(context.diagnostics, ParseDiagnostic{
			Code: diagnostic.Code, Kind: kind, Format: context.format, Feature: feature,
			Reason: fmt.Sprintf("%s at track=%d staff=%d measure=%d voice=%d beat=%d note=%d", diagnostic.Reason, diagnostic.Location.Track, diagnostic.Location.Staff, diagnostic.Location.Measure, diagnostic.Location.Voice, diagnostic.Location.Beat, diagnostic.Location.Note),
		})
	}
	result = &ParseResult{Song: song, Diagnostics: context.diagnostics}
	if options.Strict {
		rejected := strictDiagnostics(result.Diagnostics, options.StrictKinds)
		if len(rejected) != 0 {
			return result, &StrictParseError{Diagnostics: rejected}
		}
	}
	return result, nil
}

func parseWithContext(data []byte, context *parseContext) (*Song, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("data too short to detect format")
	}

	header := string(data[:4])
	switch {
	case header == "BCFZ" || header == "BCFS":
		if context != nil {
			context.format = "GP6"
		}
		return parseGPXWithContext(data, context)
	case data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04:
		if context != nil {
			context.format = "GPIF"
		}
		return parseGP7ZipWithContext(data, context)
	default:
		if context != nil {
			context.format = "GP3-5"
		}
		return parseBinaryGPWithContext(data, context)
	}
}

func parseBinaryGPWithContext(data []byte, context *parseContext) (*Song, error) {
	c := newCursorWithContext(data, context)
	song := &Song{
		Tempo:     120,
		TempoName: "Moderate",
	}
	if err := song.readBinary(c); err != nil {
		return nil, fmt.Errorf("parsing binary GP: %w", err)
	}
	if song.Version.Number[0] >= 3 && song.Version.Number[0] <= 5 {
		context.setFormat(fmt.Sprintf("GP%d", song.Version.Number[0]))
	}
	return song, nil
}
