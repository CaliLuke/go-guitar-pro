// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/xml"
	"errors"
	"reflect"
	"testing"
)

const legacyInstrumentCase = "M04-LEGACY-INSTRUMENT-BOUNDS"
const legacyRSECase = "M05-RSE-BOUNDS"

type legacyOmissionCase struct {
	name, field, code string
	value             any
	edit              func(*Song)
}

func legacyInstrumentOmissions() []legacyOmissionCase {
	return []legacyOmissionCase{
		{"fret count", "Track.FretCount", "gp8.omit.track-fret-count", uint8(31), func(s *Song) { s.Tracks[0].FretCount = 31 }},
		{"twelve string", "Track.TwelveStringedGuitarTrack", "gp8.omit.track-twelve-stringed", true, func(s *Song) { s.Tracks[0].TwelveStringedGuitarTrack = true }},
		{"banjo", "Track.BanjoTrack", "gp8.omit.track-banjo", true, func(s *Song) { s.Tracks[0].BanjoTrack = true }},
	}
}

func legacyRSEOmissions() []legacyOmissionCase {
	return []legacyOmissionCase{
		{"master knobs", "RseEqualizer.Knobs", "gp8.omit.master-rse", []float32{0.25}, func(s *Song) { s.MasterEffect.Equalizer.Knobs = []float32{0.25} }},
		{"master zero knob", "RseEqualizer.Knobs", "gp8.omit.master-rse", []float32{0}, func(s *Song) { s.MasterEffect.Equalizer.Knobs = []float32{0} }},
		{"master gain", "RseEqualizer.Gain", "gp8.omit.master-rse", float32(0.5), func(s *Song) { s.MasterEffect.Equalizer.Gain = float32(0.5) }},
		{"master volume", "RseMasterEffect.Volume", "gp8.omit.master-rse", float32(0.75), func(s *Song) { s.MasterEffect.Volume = float32(0.75) }},
		{"master reverb", "RseMasterEffect.Reverb", "gp8.omit.master-rse", float32(0.5), func(s *Song) { s.MasterEffect.Reverb = float32(0.5) }},
		{"effect category", "RseInstrument.EffectCategory", "gp8.omit.track-rse", "Delay", func(s *Song) { s.Tracks[0].Rse.Instrument.EffectCategory = "Delay" }},
		{"effect", "RseInstrument.Effect", "gp8.omit.track-rse", "Echo", func(s *Song) { s.Tracks[0].Rse.Instrument.Effect = "Echo" }},
		{"instrument", "RseInstrument.Instrument", "gp8.omit.track-rse", int16(-1), func(s *Song) { s.Tracks[0].Rse.Instrument.Instrument = int16(-1) }},
		{"unknown", "RseInstrument.Unknown", "gp8.omit.track-rse", int16(-32768), func(s *Song) { s.Tracks[0].Rse.Instrument.Unknown = int16(-32768) }},
		{"sound bank", "RseInstrument.SoundBank", "gp8.omit.track-rse", int16(32767), func(s *Song) { s.Tracks[0].Rse.Instrument.SoundBank = int16(32767) }},
		{"effect number", "RseInstrument.EffectNumber", "gp8.omit.track-rse", int16(17), func(s *Song) { s.Tracks[0].Rse.Instrument.EffectNumber = int16(17) }},
		{"track knobs", "RseEqualizer.Knobs", "gp8.omit.track-rse", []float32{-0.25}, func(s *Song) { s.Tracks[0].Rse.Equalizer.Knobs = []float32{-0.25} }},
		{"track zero knob", "RseEqualizer.Knobs", "gp8.omit.track-rse", []float32{0}, func(s *Song) { s.Tracks[0].Rse.Equalizer.Knobs = []float32{0} }},
		{"track gain", "RseEqualizer.Gain", "gp8.omit.track-rse", float32(0.5), func(s *Song) { s.Tracks[0].Rse.Equalizer.Gain = float32(0.5) }},
		{"humanize", "TrackRse.Humanize", "gp8.omit.track-rse", uint8(3), func(s *Song) { s.Tracks[0].Rse.Humanize = uint8(3) }},
		{"use rse", "Track.UseRse", "gp8.omit.track-use-rse", true, func(s *Song) { s.Tracks[0].UseRse = true }},
		{"accent VerySoft", "TrackRse.AutoAccentuation", "gp8.omit.track-rse", AccentuationVerySoft, func(s *Song) { s.Tracks[0].Rse.AutoAccentuation = AccentuationVerySoft }},
		{"accent Soft", "TrackRse.AutoAccentuation", "gp8.omit.track-rse", AccentuationSoft, func(s *Song) { s.Tracks[0].Rse.AutoAccentuation = AccentuationSoft }},
		{"accent Medium", "TrackRse.AutoAccentuation", "gp8.omit.track-rse", AccentuationMedium, func(s *Song) { s.Tracks[0].Rse.AutoAccentuation = AccentuationMedium }},
		{"accent Strong", "TrackRse.AutoAccentuation", "gp8.omit.track-rse", AccentuationStrong, func(s *Song) { s.Tracks[0].Rse.AutoAccentuation = AccentuationStrong }},
		{"accent VeryStrong", "TrackRse.AutoAccentuation", "gp8.omit.track-rse", AccentuationVeryStrong, func(s *Song) { s.Tracks[0].Rse.AutoAccentuation = AccentuationVeryStrong }},
	}
}

func TestConformanceLegacyInstrumentBounds(t *testing.T) {
	runConformanceLegacyInstrumentBounds(newConformanceRun(t))
}

func runConformanceLegacyInstrumentBounds(run *conformanceRun) {
	assertLegacyIsolatedOmissions(run, legacyInstrumentCase, legacyInstrumentOmissions())
	t := run.t
	for _, frets := range []uint8{0, 24} {
		song := consumerLimitSong(t)
		song.Tracks[0].FretCount = frets
		if report := PreflightExport(song, ExportFormatGP8, ExportOptions{}); len(report.Entries) != 0 {
			t.Fatalf("default fret report = %#v", report.Entries)
		}
	}
	song := consumerLimitSong(t)
	song.Tracks[0].TwelveStringedGuitarTrack, song.Tracks[0].BanjoTrack = true, true
	for _, allowed := range [][]string{{"gp8.omit.track-twelve-stringed"}, {"gp8.omit.track-banjo"}, {"gp8.omit.track-twelve-stringed", "gp8.omit.track-banjo"}} {
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		run.Report(legacyInstrumentCase, reportCodes(report), []string{"gp8.omit.track-twelve-stringed", "gp8.omit.track-banjo"})
		if len(allowed) == 1 {
			if err == nil || len(data) != 0 {
				t.Fatal("one allowance hid the other flag")
			}
		} else if err != nil || len(data) == 0 {
			t.Fatalf("exact allowances failed: %v", err)
		}
	}
}

func TestConformanceRSEBounds(t *testing.T) {
	runConformanceRSEBounds(newConformanceRun(t))
}

func runConformanceRSEBounds(run *conformanceRun) {
	TestRSEInstrumentReaderBounds(run.t)
	TestRSEInstrumentReaderTruncation(run.t)
	assertLegacyIsolatedOmissions(run, legacyRSECase, legacyRSEOmissions())
	for _, knobs := range [][]float32{nil, {}} {
		song := consumerLimitSong(run.t)
		song.MasterEffect = RseMasterEffect{Equalizer: RseEqualizer{Knobs: knobs}}
		song.Tracks[0].Rse = TrackRse{Equalizer: RseEqualizer{Knobs: knobs}}
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		run.Report(legacyRSECase, reportCodes(report), []string{})
	}
}

func assertLegacyIsolatedOmissions(run *conformanceRun, caseID string, cases []legacyOmissionCase) {
	t := run.t
	baseline := consumerLimitSong(t)
	baselineData, baselineReport, err := ExportWithReport(baseline, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(baselineReport.Entries) != 0 {
		t.Fatalf("unclean omission control: %v %#v", err, baselineReport.Entries)
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			song := consumerLimitSong(t)
			test.edit(song)
			before := conformanceContractSnapshot(song, false)
			run.Omitted(test.field, legacyOmissionValue(song, test), test.value)
			for _, allowed := range [][]string{nil, {"gp8.omit.track-port"}, {test.code}} {
				data, report, exportErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				run.Report(caseID, reportCodes(report), []string{test.code})
				if len(report.Entries) != 1 || report.Entries[0].Disposition != ExportDispositionOmitted || report.Entries[0].Location != (ScoreLocation{}) {
					t.Fatalf("loss location/disposition = %#v", report.Entries)
				}
				if len(allowed) == 0 || allowed[0] != test.code {
					var loss *ExportLossError
					if !errors.As(exportErr, &loss) || len(data) != 0 {
						t.Fatalf("strict result=%d bytes %v", len(data), exportErr)
					}
				} else {
					if exportErr != nil || len(data) == 0 {
						t.Fatalf("exact allowance failed: %v", exportErr)
					}
					var wire gpifDocument
					if decodeErr := xml.Unmarshal(conformanceBarreGPIF(t, data), &wire); decodeErr != nil {
						t.Fatal(decodeErr)
					}
					run.Wire("gpifChannelStrip.Parameters", wire.Tracks.Tracks[0].RSE.ChannelStrip.Parameters, "0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0.5 0 0.5 0.503937 0.787402 0.5 0.5 0.5")
					if !bytes.Equal(conformanceBarreGPIF(t, data), conformanceBarreGPIF(t, baselineData)) {
						t.Fatal("legacy omission changed unrelated GPIF, tuning or MIDI")
					}
				}
				if after := conformanceContractSnapshot(song, false); after != before {
					t.Fatal("export changed public input")
				}
			}
		})
	}
}

func legacyOmissionValue(song *Song, test legacyOmissionCase) any {
	// Each case sets one field; use its public type and member name to read the
	// value independently of the edit closure for the assertion recorder.
	switch test.field {
	case "Track.FretCount":
		return song.Tracks[0].FretCount
	case "Track.TwelveStringedGuitarTrack":
		return song.Tracks[0].TwelveStringedGuitarTrack
	case "Track.BanjoTrack":
		return song.Tracks[0].BanjoTrack
	case "Track.UseRse":
		return song.Tracks[0].UseRse
	case "RseMasterEffect.Volume":
		return song.MasterEffect.Volume
	case "RseMasterEffect.Reverb":
		return song.MasterEffect.Reverb
	case "RseEqualizer.Knobs":
		if song.MasterEffect.Equalizer.Knobs != nil {
			return song.MasterEffect.Equalizer.Knobs
		}
		return song.Tracks[0].Rse.Equalizer.Knobs
	case "RseEqualizer.Gain":
		if song.MasterEffect.Equalizer.Gain != 0 {
			return song.MasterEffect.Equalizer.Gain
		}
		return song.Tracks[0].Rse.Equalizer.Gain
	}
	if len(test.field) > len("RseInstrument.") && test.field[:len("RseInstrument.")] == "RseInstrument." {
		return reflect.ValueOf(song.Tracks[0].Rse.Instrument).FieldByName(test.field[len("RseInstrument."):]).Interface()
	}
	return reflect.ValueOf(song.Tracks[0].Rse).FieldByName(test.field[len("TrackRse."):]).Interface()
}

func TestAlphaTabLegacyOmissionBoundaries(t *testing.T) {
	requireAlphaTabConformance(t)
	baseline := consumerLimitSong(t)
	data, _, err := ExportWithReport(baseline, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var want []legacyInstrumentConsumerFact
	readAlphaTabOracleFacts(t, "--legacy-instrument", writeConformanceFixture(t, data), &want)
	expected := []legacyInstrumentConsumerFact{{Program: 25, Volume: 12, Balance: 8, PrimaryChannel: 0, SecondaryChannel: 1, Staves: []legacyStaffConsumerFact{{Tuning: []int{64, 59, 55, 50, 45, 40}, Capo: 0}}}}
	if !reflect.DeepEqual(want, expected) {
		t.Fatalf("baseline consumer=%#v", want)
	}
	for _, test := range append(legacyInstrumentOmissions(), legacyRSEOmissions()...) {
		t.Run(test.name, func(t *testing.T) {
			song := consumerLimitSong(t)
			test.edit(song)
			output, report, exportErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if exportErr != nil || len(report.Entries) != 1 || report.Entries[0].Code != test.code {
				t.Fatalf("omission export=%v %#v", exportErr, report.Entries)
			}
			var got []legacyInstrumentConsumerFact
			readAlphaTabOracleFacts(t, "--legacy-instrument", writeConformanceFixture(t, output), &got)
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("legacy omission changed final consumer: %#v", got)
			}
		})
	}
}

type legacyInstrumentConsumerFact struct {
	Program, Volume, Balance, PrimaryChannel, SecondaryChannel int
	Staves                                                     []legacyStaffConsumerFact
}
type legacyStaffConsumerFact struct {
	Tuning []int
	Capo   int
}
