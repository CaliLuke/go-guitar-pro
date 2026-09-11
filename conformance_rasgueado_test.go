// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

const rasgueadoCase = "M10-RASGUEADO"

type beatTechniqueFact struct {
	Track, Staff, Bar, Voice, Beat int
	Rasgueado                      int
	Tap, Slap, Pop                 bool
	Wah                            int
	LeftHandTapped                 []bool
}

var rasgueadoCases = []struct {
	pattern     RasgueadoPattern
	name, token string
}{
	{RasgueadoNone, "RasgueadoPattern.RasgueadoNone", ""},
	{RasgueadoIi, "RasgueadoPattern.RasgueadoIi", "ii_1"},
	{RasgueadoMi, "RasgueadoPattern.RasgueadoMi", "mi_1"},
	{RasgueadoMiiTriplet, "RasgueadoPattern.RasgueadoMiiTriplet", "mii_1"},
	{RasgueadoMiiAnapaest, "RasgueadoPattern.RasgueadoMiiAnapaest", "mii_2"},
	{RasgueadoPmpTriplet, "RasgueadoPattern.RasgueadoPmpTriplet", "pmp_1"},
	{RasgueadoPmpAnapaest, "RasgueadoPattern.RasgueadoPmpAnapaest", "pmp_2"},
	{RasgueadoPeiTriplet, "RasgueadoPattern.RasgueadoPeiTriplet", "pei_1"},
	{RasgueadoPeiAnapaest, "RasgueadoPattern.RasgueadoPeiAnapaest", "pei_2"},
	{RasgueadoPaiTriplet, "RasgueadoPattern.RasgueadoPaiTriplet", "pai_1"},
	{RasgueadoPaiAnapaest, "RasgueadoPattern.RasgueadoPaiAnapaest", "pai_2"},
	{RasgueadoAmiTriplet, "RasgueadoPattern.RasgueadoAmiTriplet", "ami_1"},
	{RasgueadoAmiAnapaest, "RasgueadoPattern.RasgueadoAmiAnapaest", "ami_2"},
	{RasgueadoPpp, "RasgueadoPattern.RasgueadoPpp", "ppp_1"},
	{RasgueadoAmii, "RasgueadoPattern.RasgueadoAmii", "amii_1"},
	{RasgueadoAmip, "RasgueadoPattern.RasgueadoAmip", "amip_1"},
	{RasgueadoEami, "RasgueadoPattern.RasgueadoEami", "eami_1"},
	{RasgueadoEamii, "RasgueadoPattern.RasgueadoEamii", "eamii_1"},
	{RasgueadoPeami, "RasgueadoPattern.RasgueadoPeami", "peami_1"},
}

func TestConformanceRasgueado(t *testing.T) { runConformanceRasgueado(newConformanceRun(t)) }
func runConformanceRasgueado(run *conformanceRun) {
	for _, test := range rasgueadoCases {
		song := consumerLimitSong(run.t)
		effect := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
		effect.RasgueadoPattern = test.pattern
		before := conformanceContractSnapshot(song, false)
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil {
			run.t.Fatal(err, report)
		}
		run.Report(rasgueadoCase, reportCodes(report), []string{})
		document := conformanceWireDocument(run.t, data)
		token := ""
		for _, p := range document.Beats.Beats[0].Properties.Properties {
			if p.Name == "Rasgueado" && p.Rasgueado != nil {
				token = *p.Rasgueado
			}
		}
		run.Wire("gpifProperty.Rasgueado", token, test.token)
		parsed, err := ParseWithOptions(data, ParseOptions{Strict: true})
		if err != nil {
			run.t.Fatal(err)
		}
		got := parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
		run.Enum(test.name, got.RasgueadoPattern, test.pattern)
		run.Preserved("BeatEffects.RasgueadoPattern", got.RasgueadoPattern, test.pattern)
		run.Normalized("BeatEffects.HasRasgueado", got.HasRasgueado, test.pattern != RasgueadoNone)
		if !reflect.DeepEqual(before, conformanceContractSnapshot(song, false)) {
			run.t.Fatal("export changed input")
		}
	}
	source := parseTestFixture(run.t, "testdata/gp7/rasgueado.gp")
	beats := source.Tracks[0].Measures[0].Voices[0].Beats
	run.Preserved("BeatEffects.RasgueadoPattern", []RasgueadoPattern{beats[0].Effect.RasgueadoPattern, beats[2].Effect.RasgueadoPattern}, []RasgueadoPattern{RasgueadoIi, RasgueadoPmpAnapaest})
}

func TestRasgueadoUnknownSource(t *testing.T) {
	song := consumerLimitSong(t)
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.RasgueadoPattern = RasgueadoIi
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range []string{"<Rasgueado>unknown-pattern</Rasgueado>", ""} {
		source := strings.Replace(string(conformanceBarreGPIF(t, data)), "<Rasgueado>ii_1</Rasgueado>", replacement, 1)
		archive := conformanceGPIFArchive(t, source)
		parsed, e := ParseWithOptions(archive, ParseOptions{})
		if e != nil {
			t.Fatal(e)
		}
		code := "GPIF.Beat.Property.Rasgueado"
		if replacement == "" {
			code += ".MissingPattern"
		}
		if !slices.ContainsFunc(parsed.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == code && d.ObjectID == "0" }) {
			t.Fatal(parsed.Diagnostics)
		}
		effect := parsed.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
		if effect.HasRasgueado || effect.RasgueadoPattern != RasgueadoNone {
			t.Fatal("unknown pattern substituted")
		}
		if _, e = ParseWithOptions(archive, ParseOptions{Strict: true}); e == nil {
			t.Fatal("strict unknown source accepted")
		}
	}
}

func TestAlphaTabRasgueado(t *testing.T) {
	requireAlphaTabConformance(t)
	baseline, err := Export(consumerLimitSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var expected []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, baseline), &expected)
	if len(expected) == 0 || expected[0].Track != 0 || expected[0].Bar != 0 || expected[0].Beat != 0 || expected[0].Rasgueado != 0 {
		t.Fatal(expected)
	}
	for index, test := range rasgueadoCases {
		song := consumerLimitSong(t)
		song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.RasgueadoPattern = test.pattern
		output, exportErr := Export(song, ExportFormatGP8)
		if exportErr != nil {
			t.Fatal(exportErr)
		}
		var facts []beatTechniqueFact
		readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &facts)
		expected[0].Rasgueado = index
		if !reflect.DeepEqual(facts, expected) {
			t.Fatalf("%s consumer=%#v", test.name, facts)
		}
	}
	var source []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", "testdata/gp7/rasgueado.gp", &source)
	song := parseTestFixture(t, "testdata/gp7/rasgueado.gp")
	output, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var target []beatTechniqueFact
	readAlphaTabOracleFacts(t, "--beat-techniques", writeConformanceFixture(t, output), &target)
	if !reflect.DeepEqual(source, target) {
		t.Fatalf("source=%#v target=%#v", source, target)
	}
}
