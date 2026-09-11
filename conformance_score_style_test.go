// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

const scoreBarlineCase = "M02-SCORE-BARLINES"
const scoreBarlineValue = "global extended barlines and all three number policies"

func TestConformanceScoreBarlines(t *testing.T) { runConformanceScoreBarlines(newConformanceRun(t)) }
func runConformanceScoreBarlines(run *conformanceRun) {
	t := run.t
	for _, test := range []struct {
		name   string
		extend bool
		policy BarNumberPolicy
		member string
	}{{"extended-barlines", true, BarNumberAllBars, "BarNumberAllBars"}, {"barnumbers-all", false, BarNumberAllBars, "BarNumberAllBars"}, {"barnumbers-first", false, BarNumberFirstOfSystem, "BarNumberFirstOfSystem"}, {"barnumbers-hide", false, BarNumberHide, "BarNumberHide"}} {
		song := scoreStyleSong(t, test.name)
		run.Normalized("Song.Style", song.Style != nil, true)
		run.ClaimPrimary(claimAllStages("barlines", scoreBarlineCase, scoreBarlineValue)...).Preserved("ScoreStyle.ExtendedBarLines", song.Style.ExtendedBarLines, ptrTo(test.extend))
		run.Preserved("ScoreStyle.BarNumbers", song.Style.BarNumbers, ptrTo(test.policy))
		run.Enum("BarNumberPolicy."+test.member, *song.Style.BarNumbers, test.policy)
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-name-page-policy"}}})
		if err != nil {
			t.Fatal(err)
		}
		records := scoreStyleWire(t, data)
		extend, number := findStyleRecord(t, records, "System/ExtendedBarLines"), findStyleRecord(t, records, "System/barIndexDrawType")
		run.Dispatch("applyBinaryStylesheet:record.key", []string{extend.key, number.key}, []string{"System/ExtendedBarLines", "System/barIndexDrawType"})
		if extend.kind != 0 || len(extend.value) != 1 || number.kind != 1 || len(number.value) != 4 {
			t.Fatal("style keys have incorrect binary types")
		}
		run.Wire("binaryStyleRecord.key", []string{extend.key, number.key}, []string{"System/ExtendedBarLines", "System/barIndexDrawType"})
		run.Wire("binaryStyleRecord.kind", []byte{extend.kind, number.kind}, []byte{0, 1})
		run.ClaimSerialization(claimSite("barlines", "export", scoreBarlineCase, scoreBarlineValue)).Wire("binaryStyleRecord.value", extend.value, []byte{map[bool]byte{false: 0, true: 1}[test.extend]})
		run.Wire("binaryStyleRecord.value", number.value, []byte{0, 0, 0, byte(test.policy)})
		archive, archiveErr := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if archiveErr != nil {
			t.Fatal(archiveErr)
		}
		raw := readZipMember(t, archive, "Content/BinaryStylesheet")
		for _, record := range []binaryStyleRecord{extend, number} {
			encoded := append([]byte{byte(len(record.key))}, []byte(record.key)...)
			encoded = append(encoded, record.kind)
			encoded = append(encoded, record.value...)
			run.Wire("binaryStyleRecord.value", bytes.Count(raw, encoded), 1)
		}
		run.Preserved("ScoreStyle.BarNumbers", binary.BigEndian.Uint32(number.value), uint32(test.policy))
		run.ClaimReport(claimSite("barlines", "export", scoreBarlineCase, scoreBarlineValue)).Report(scoreBarlineCase, reportCodes(report), []string{"gp8.normalize.track-name-page-policy"})
		if !reflect.DeepEqual(otherStyleRecords(records), otherStyleRecords(song.Style.records)) {
			t.Fatal("barline export replaced other typed stylesheet records")
		}
	}
	song := scoreStyleSong(t, "extended-barlines")
	song.Style.ExtendedBarLines = nil
	song.Style.BarNumbers = nil
	parsed, err := Parse(mustStringNumberExport(t, song))
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("ScoreStyle.ExtendedBarLines", parsed.Style.ExtendedBarLines, (*bool)(nil))
	run.Preserved("ScoreStyle.BarNumbers", parsed.Style.BarNumbers, (*BarNumberPolicy)(nil))
	song.Style = nil
	parsed, err = Parse(mustStringNumberExport(t, song))
	if err != nil {
		t.Fatal(err)
	}
	run.Normalized("Song.Style", parsed.Style != nil, true)
}
func scoreStyleSong(t *testing.T, name string) *Song {
	t.Helper()
	song := parseTestFixture(t, "testdata/gp8/"+name+".gp")
	cleanNotationProvenance(song)
	song.PanAutomations = nil
	song.VolumeAutomations = nil
	for i := range song.Tracks {
		song.Tracks[i].UseRse = false
	}
	return song
}
func scoreStyleWire(t *testing.T, data []byte) []binaryStyleRecord {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	records, err := readBinaryStylesheet(readZipMember(t, archive, "Content/BinaryStylesheet"))
	if err != nil {
		t.Fatal(err)
	}
	return records
}
func findStyleRecord(t *testing.T, records []binaryStyleRecord, key string) binaryStyleRecord {
	t.Helper()
	for _, record := range records {
		if record.key == key {
			return record
		}
	}
	t.Fatalf("missing style key %s", key)
	return binaryStyleRecord{}
}
func otherStyleRecords(records []binaryStyleRecord) []binaryStyleRecord {
	var out []binaryStyleRecord
	for _, record := range records {
		if record.key != "System/ExtendedBarLines" && record.key != "System/barIndexDrawType" {
			out = append(out, record)
		}
	}
	return out
}
func TestBinaryStylesheetMalformedRecords(t *testing.T) {
	for _, data := range [][]byte{nil, {0, 0, 0}, {255, 255, 255, 255}, {0, 0, 0, 1}, {0, 0, 0, 0, 1}, {0, 0, 0, 1, 1, 'a', 8}, {0, 0, 0, 1, 1, 'a', 3, 255, 255}, {0, 0, 0, 1, 1, 'a', 0, 2}, {0, 0, 0, 1, 1, 255, 0, 1}, make([]byte, maxBinaryStylesheetSize+1)} {
		if _, err := readBinaryStylesheet(data); err == nil {
			t.Fatalf("malformed stylesheet accepted (%d bytes)", len(data))
		}
	}
	source := scoreStyleSong(t, "extended-barlines")
	xml := string(conformanceBarreGPIF(t, mustStringNumberExport(t, source)))
	for _, data := range [][]byte{{0, 0, 0, 1, 1, 'a', 1, 0, 0}, {0, 0, 0, 1, 1, 'a', 3, 0, 2, 'a'}} {
		archive := conformanceBackingArchive(t, xml, map[string][]byte{"Content/BinaryStylesheet": data})
		song, err := Parse(archive)
		result, optionsErr := ParseWithOptions(archive, ParseOptions{})
		if err == nil || optionsErr == nil || song != nil || result != nil || !strings.Contains(err.Error(), "reading BinaryStylesheet") {
			t.Fatalf("public malformed stylesheet=%v/%v", err, optionsErr)
		}
	}
}

type scoreStyleFacts struct {
	Extended bool
	Numbers  string
	Other    map[string]any
	Headers  []map[string]any
}

func TestAlphaTabScoreBarlines(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, name := range []string{"extended-barlines", "barnumbers-all", "barnumbers-first", "barnumbers-hide"} {
		var original, got scoreStyleFacts
		readAlphaTabOracleFacts(t, "--score-style", "testdata/gp8/"+name+".gp", &original)
		song := scoreStyleSong(t, name)
		for _, extend := range []bool{false, true} {
			for _, policy := range []BarNumberPolicy{BarNumberAllBars, BarNumberFirstOfSystem, BarNumberHide} {
				song.Style.ExtendedBarLines = ptrTo(extend)
				song.Style.BarNumbers = ptrTo(policy)
				want := original
				want.Extended = extend
				want.Numbers = []string{"AllBars", "FirstOfSystem", "Hide"}[policy]
				readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("%s style output=%#v want=%#v", name, got, want)
				}
			}
		}
		song.Style.ExtendedBarLines = nil
		song.Style.BarNumbers = nil
		original.Extended = false
		original.Numbers = "AllBars"
		readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if !reflect.DeepEqual(got, original) {
			t.Fatalf("absent policy defaults=%#v", got)
		}
	}
	conformanceIndependentClaim(t, "field:ScoreStyle.ExtendedBarLines", claimAllStages("barlines", scoreBarlineCase, scoreBarlineValue)...)
}
func TestBinaryStylesheetAllRecordKinds(t *testing.T) {
	var data bytes.Buffer
	data.Write([]byte{0, 0, 0, 8})
	payloads := [][]byte{{1}, {128, 0, 0, 0}, {63, 192, 0, 0}, {0, 4, 'A', '&', 195, 169}, {255, 255, 255, 255, 0, 0, 0, 7}, {0, 0, 0, 2, 0, 0, 0, 3}, {0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 4}, {9, 18, 27, 255}}
	for kind, payload := range payloads {
		data.WriteByte(1)
		data.WriteByte(byte('a' + kind))
		data.WriteByte(byte(kind))
		data.Write(payload)
	}
	records, err := readBinaryStylesheet(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	song := &Song{Style: &ScoreStyle{records: records}}
	if !bytes.Equal(buildGP8BinaryStylesheet(song), data.Bytes()) {
		t.Fatal("opaque typed records changed")
	}
	for length := 4; length < len(data.Bytes()); length++ {
		if _, err = readBinaryStylesheet(data.Bytes()[:length]); err == nil {
			t.Fatalf("truncation %d accepted", length)
		}
	}
	// Input storage must not remain reachable after decoding.
	copyBefore := buildGP8BinaryStylesheet(song)
	for i := range data.Bytes() {
		data.Bytes()[i] = 255
	}
	if !bytes.Equal(copyBefore, buildGP8BinaryStylesheet(song)) {
		t.Fatal("parsed records alias input bytes")
	}
}

func TestBinaryStylesheetTypedPolicyBounds(t *testing.T) {
	source := scoreStyleSong(t, "extended-barlines")
	xml := string(conformanceBarreGPIF(t, mustStringNumberExport(t, source)))
	for _, test := range []struct {
		key   string
		kind  byte
		value []byte
	}{
		{"System/ExtendedBarLines", 1, []byte{0, 0, 0, 1}},
		{"System/barIndexDrawType", 0, []byte{1}},
		{"System/barIndexDrawType", 1, []byte{255, 255, 255, 255}},
		{"System/barIndexDrawType", 1, []byte{0, 0, 0, 3}},
		{"System/barIndexDrawType", 1, []byte{127, 255, 255, 255}},
	} {
		data := append([]byte{0, 0, 0, 1, byte(len(test.key))}, []byte(test.key)...)
		data = append(data, test.kind)
		data = append(data, test.value...)
		archive := conformanceBackingArchive(t, xml, map[string][]byte{"Content/BinaryStylesheet": data})
		song, err := Parse(archive)
		result, optionsErr := ParseWithOptions(archive, ParseOptions{})
		if err == nil || optionsErr == nil || song != nil || result != nil || !strings.Contains(err.Error(), test.key) {
			t.Fatalf("bad typed policy accepted: %v/%v", err, optionsErr)
		}
	}
	// Duplicate keys follow the pinned map's last-value rule. Export emits one owned key.
	data := []byte{0, 0, 0, 2}
	for _, value := range []byte{0, 1} {
		key := "System/ExtendedBarLines"
		data = append(data, byte(len(key)))
		data = append(data, key...)
		data = append(data, 0, value)
	}
	if err := applyBinaryStylesheet(source, data); err != nil {
		t.Fatal(err)
	}
	if source.Style.ExtendedBarLines == nil || !*source.Style.ExtendedBarLines || len(scoreStyleRecords(source.Style)) != 1 {
		t.Fatal("duplicate key precedence changed")
	}
}
