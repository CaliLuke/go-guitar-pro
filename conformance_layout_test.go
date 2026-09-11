// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"reflect"
	"regexp"
	"testing"
)

const layoutCase = "M02-SYSTEM-LAYOUT"
const layoutValue = "independent system counts and display scales"

func TestConformanceSystemLayout(t *testing.T) { runConformanceSystemLayout(newConformanceRun(t)) }
func runConformanceSystemLayout(run *conformanceRun) {
	t := run.t
	for _, inherit := range []bool{false, true} {
		s := semanticExportProbeSong(t)
		s.SystemLayout = &SystemLayout{DefaultBarsPerSystem: 5, BarsPerSystem: []int{2, 4}}
		s.Tracks[0].SystemLayout = &SystemLayout{DefaultBarsPerSystem: 2, BarsPerSystem: []int{1, 3, 2}}
		want := s.Tracks[0].SystemLayout
		codes := []string{}
		if inherit {
			s.Tracks[0].SystemLayout = nil
			want = s.SystemLayout
			codes = []string{"gp8.normalize.track-layout-inheritance"}
		}
		masterScale, barScale := .5, 2.0
		s.MeasureHeaders[0].DisplayScale = &masterScale
		s.Tracks[0].Measures[0].DisplayScale = &barScale
		data, report := assertConsumerLossPolicy(t, s, codes)
		run.Report(layoutCase, reportCodes(report), codes)
		p, e := Parse(data)
		if e != nil {
			t.Fatal(e)
		}
		run.ClaimPrimary(claimSite("layout", "import", layoutCase, layoutValue), claimSite("layout", "model", layoutCase, layoutValue)).Normalized("Song.SystemLayout", p.SystemLayout, s.SystemLayout)
		run.Normalized("Track.SystemLayout", p.Tracks[0].SystemLayout, want)
		run.Preserved("SystemLayout.DefaultBarsPerSystem", p.SystemLayout.DefaultBarsPerSystem, 5)
		run.Normalized("SystemLayout.BarsPerSystem", p.SystemLayout.BarsPerSystem, []int{2, 4})
		run.Preserved("MeasureHeader.DisplayScale", p.MeasureHeaders[0].DisplayScale, &masterScale)
		run.Preserved("Measure.DisplayScale", p.Tracks[0].Measures[0].DisplayScale, &barScale)
		doc := conformanceWireDocument(t, data)
		run.Wire("gpifScore.SystemLayout", *doc.Score.SystemLayout, "2 4")
		run.Wire("gpifScore.SystemDefault", *doc.Score.SystemDefault, "5")
		def, arr := gp8SystemLayout(want)
		run.Wire("gpifTrack.SystemDefault", doc.Tracks.Tracks[0].SystemDefault, def)
		run.Wire("gpifTrack.SystemLayout", doc.Tracks.Tracks[0].SystemLayout, arr)
		run.Wire("gpifBar.XProperties", len(doc.Bars.Bars[0].XProperties.Properties), 1)
		run.Wire("gpifMasterBar.XProperties", *doc.MasterBars.MasterBars[0].XProperties.Properties[0].Double, "0.5")
		run.Wire("gpifXProperty.ID", []string{doc.MasterBars.MasterBars[0].XProperties.Properties[0].ID, doc.Bars.Bars[0].XProperties.Properties[0].ID}, []string{"1124073984", "1124139520"})
		run.Wire("gpifXProperty.Double", *doc.Bars.Bars[0].XProperties.Properties[0].Double, "2")
		floatData := rewriteConformanceGPIF(t, data, func(source string) string {
			return regexp.MustCompile(`(<XProperty id="1124139520">\s*)<Double>2</Double>`).ReplaceAllString(source, "${1}<Float>2</Float>")
		})
		floatDoc := conformanceWireDocument(t, floatData)
		run.Wire("gpifXProperty.Float", *floatDoc.Bars.Bars[0].XProperties.Properties[0].Float, "2")
		parsed, e := Parse(floatData)
		if e != nil || parsed.Tracks[0].Measures[0].DisplayScale == nil || *parsed.Tracks[0].Measures[0].DisplayScale != 2 {
			t.Fatal("Float bar scale", e)
		}
		if inherit && s.Tracks[0].SystemLayout != nil {
			t.Fatal("export changed authored scope")
		}
	}
}

type layoutConsumerFacts struct {
	Default int       `json:"default"`
	Systems []int     `json:"systems"`
	Masters []float64 `json:"masters"`
	Tracks  []struct {
		Default int         `json:"default"`
		Systems []int       `json:"systems"`
		Scales  [][]float64 `json:"scales"`
	} `json:"tracks"`
}

func TestAlphaTabSystemLayout(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, path := range []string{"testdata/gp7/multi-track-different.gp", "testdata/gp7/resized.gp", "testdata/gp6/file-system-compressed.gpx"} {
		var source, out layoutConsumerFacts
		readAlphaTabOracleFacts(t, "--system-layout", path, &source)
		s := parseTestFixture(t, path)
		data, e := Export(s, ExportFormatGP8)
		if e != nil {
			t.Fatal(e)
		}
		readAlphaTabOracleFacts(t, "--system-layout", writeConformanceFixture(t, data), &out)
		if !reflect.DeepEqual(source, out) {
			t.Fatalf("%s source %#v target %#v", path, source, out)
		}
	}
	s := semanticExportProbeSong(t)
	s.SystemLayout = &SystemLayout{DefaultBarsPerSystem: 7, BarsPerSystem: []int{2, 5}}
	s.Tracks[0].SystemLayout = &SystemLayout{DefaultBarsPerSystem: 3, BarsPerSystem: []int{1, 4}}
	a, b := .125, 1.234567890123
	s.MeasureHeaders[0].DisplayScale = &a
	s.Tracks[0].Measures[0].DisplayScale = &b
	for _, inherit := range []bool{false, true} {
		if inherit {
			s.Tracks[0].SystemLayout = nil
		}
		data, e := Export(s, ExportFormatGP8)
		if e != nil {
			t.Fatal(e)
		}
		var facts layoutConsumerFacts
		readAlphaTabOracleFacts(t, "--system-layout", writeConformanceFixture(t, data), &facts)
		want := []int{1, 4}
		defaultWant := 3
		if inherit {
			want = []int{2, 5}
			defaultWant = 7
		}
		if facts.Default != 7 || !reflect.DeepEqual(facts.Systems, []int{2, 5}) || facts.Tracks[0].Default != defaultWant || !reflect.DeepEqual(facts.Tracks[0].Systems, want) || facts.Masters[0] != a || facts.Tracks[0].Scales[0][0] != b {
			t.Fatal(facts)
		}
	}
	conformanceIndependentClaim(t, "field:Song.SystemLayout", claimSite("layout", "import", layoutCase, layoutValue), claimSite("layout", "model", layoutCase, layoutValue))
}

func TestLayoutSourceValidationAndOwnership(t *testing.T) {
	s := semanticExportProbeSong(t)
	s.SystemLayout = &SystemLayout{DefaultBarsPerSystem: 4, BarsPerSystem: []int{2, 3}}
	s.Tracks[0].SystemLayout = &SystemLayout{DefaultBarsPerSystem: 3, BarsPerSystem: []int{1, 4}}
	scale := .5
	s.Tracks[0].Measures[0].DisplayScale = &scale
	data, e := Export(s, ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	for _, raw := range []string{"", "0", "-1", "2147483648", "1.5", "NaN", "4 bad"} {
		doc := conformanceWireDocument(t, data)
		doc.Score.SystemDefault = &raw
		body, marshalErr := xml.Marshal(doc)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, e = Parse(conformanceGPIFArchive(t, string(body))); e == nil {
			t.Fatalf("invalid default %q accepted", raw)
		}
	}
	for _, raw := range []string{"0", "-1", "2147483648", "1.5", "4 bad"} {
		doc := conformanceWireDocument(t, data)
		doc.Tracks.Tracks[0].SystemLayout = &raw
		body, marshalErr := xml.Marshal(doc)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, e = Parse(conformanceGPIFArchive(t, string(body))); e == nil {
			t.Fatalf("invalid array %q accepted", raw)
		}
	}
	for _, raw := range []string{"0", "-1", "NaN", "Inf", "1e309", "1e-400", "wrong"} {
		doc := conformanceWireDocument(t, data)
		doc.Bars.Bars[0].XProperties.Properties[0].Double = &raw
		body, marshalErr := xml.Marshal(doc)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, e = Parse(conformanceGPIFArchive(t, string(body))); e == nil {
			t.Fatalf("invalid scale %q accepted", raw)
		}
	}
	doc := conformanceWireDocument(t, data)
	doc.MasterBars.MasterBars = append(doc.MasterBars.MasterBars, doc.MasterBars.MasterBars[0])
	body, e := xml.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := Parse(conformanceGPIFArchive(t, string(body)))
	if e != nil {
		t.Fatal(e)
	}
	a := parsed.Tracks[0].Measures[0].DisplayScale
	b := parsed.Tracks[0].Measures[1].DisplayScale
	if a == b || a == nil || b == nil {
		t.Fatal("reused bar shares scale")
	}
	*a = 2
	if *b != .5 {
		t.Fatal("scale edit leaked")
	}
}

func TestLayoutLegacyTerminalRemainder(t *testing.T) {
	for _, c := range []struct {
		counts []int
		bars   int
		valid  bool
	}{{[]int{4, -3}, 1, true}, {[]int{4, -2}, 1, false}, {[]int{4, -3, 1}, 2, false}, {[]int{1, 0}, 1, false}, {[]int{-1}, 1, false}, {[]int{2147483647, -2147483646}, 1, true}} {
		if got := validSystemCounts(c.counts, c.bars); got != c.valid {
			t.Fatalf("counts %v bars %d valid %v", c.counts, c.bars, got)
		}
	}
	source := parseTestFixture(t, "testdata/gp6/file-system-compressed.gpx")
	if !reflect.DeepEqual(source.Tracks[0].SystemLayout.BarsPerSystem, []int{4, -3}) {
		t.Fatal(source.Tracks[0].SystemLayout)
	}
}

func TestAlphaTabEmptyTrackLayoutAuthority(t *testing.T) {
	requireAlphaTabConformance(t)
	s := semanticExportProbeSong(t)
	s.SystemLayout = &SystemLayout{DefaultBarsPerSystem: 5, BarsPerSystem: []int{2, 4}}
	s.Tracks[0].SystemLayout = &SystemLayout{}
	first, _ := assertConsumerLossPolicy(t, s, []string{"gp8.normalize.empty-layout-array"})
	parsed, e := Parse(first)
	if e != nil {
		t.Fatal(e)
	}
	second, e := Export(parsed, ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	for _, data := range [][]byte{first, second} {
		var facts layoutConsumerFacts
		readAlphaTabOracleFacts(t, "--system-layout", writeConformanceFixture(t, data), &facts)
		if facts.Default != 5 || !reflect.DeepEqual(facts.Systems, []int{2, 4}) || facts.Tracks[0].Default != 3 || len(facts.Tracks[0].Systems) != 0 {
			t.Fatalf("explicit track scope lost: %#v", facts)
		}
	}
}
