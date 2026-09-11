// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const multiRestCase = "M02-MULTI-REST"
const multiRestValue = "independent global and conflicting single-track rest preferences"
const multiRestFixture = "testdata/gp8/multi-rest-preferences.gp"

func TestConformanceMultiRest(t *testing.T) { runConformanceMultiRest(newConformanceRun(t)) }
func runConformanceMultiRest(run *conformanceRun) {
	t := run.t
	song := parseTestFixture(t, multiRestFixture)
	run.ClaimPrimary(claimAllStages("multi-rest", multiRestCase, multiRestValue)...).Normalized("ScoreStyle.MultiRest", song.Style.MultiRest, ptrTo(true))
	run.Normalized("Track.MultiRest", []*bool{song.Tracks[0].MultiRest, song.Tracks[1].MultiRest}, []*bool{ptrTo(false), ptrTo(true)})
	if song.Style.MultiRest == song.Tracks[1].MultiRest {
		t.Fatal("global and track payloads alias")
	}
	cleanNotationProvenance(song)
	for _, global := range []*bool{nil, ptrTo(false), ptrTo(true)} {
		for _, first := range []*bool{nil, ptrTo(false), ptrTo(true)} {
			for _, second := range []*bool{nil, ptrTo(false), ptrTo(true)} {
				song.Style.MultiRest = global
				song.Tracks[0].MultiRest = first
				song.Tracks[1].MultiRest = second
				before := conformanceContractSnapshot(song, false)
				data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
				if err != nil {
					t.Fatal(err)
				}
				if conformanceContractSnapshot(song, false) != before {
					t.Fatal("export changed source preferences or score")
				}
				run.ClaimReport(claimSite("multi-rest", "export", multiRestCase, multiRestValue)).Report(multiRestCase, reportCodes(report), []string{})
				want := []bool{boolRequest(global), boolRequest(first), boolRequest(second)}
				archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
				if err != nil {
					t.Fatal(err)
				}
				raw := readZipMember(t, archive, "Content/PartConfiguration")
				run.ClaimSerialization(claimSite("multi-rest", "export", multiRestCase, multiRestValue)).Wire("partConfigurationView.multiRest", []bool{raw[4] == 1, raw[11] == 1, raw[17] == 1}, want)
				run.Wire("partConfigurationView.flags", []byte{raw[9], raw[10], raw[16], raw[22]}, []byte{1, 14, 1, 14})
				parsed, err := Parse(data)
				if err != nil {
					t.Fatal(err)
				}
				run.Normalized("ScoreStyle.MultiRest", parsed.Style.MultiRest, ptrTo(want[0]))
				run.Normalized("Track.MultiRest", []*bool{parsed.Tracks[0].MultiRest, parsed.Tracks[1].MultiRest}, []*bool{ptrTo(want[1]), ptrTo(want[2])})
			}
		}
	}
}
func boolRequest(value *bool) bool { return value != nil && *value }

type multiRestFacts struct {
	Global        bool
	Tracks        []int
	MeasureCounts [][]int
	Notation      []notationTrackFact
}

func TestAlphaTabMultiRest(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, source := range []string{multiRestFixture, "testdata/gp7/grand-staff.gp"} {
		var original, got multiRestFacts
		readAlphaTabOracleFacts(t, "--multi-rest", source, &original)
		song := parseTestFixture(t, source)
		cleanNotationProvenance(song)
		song.PanAutomations = nil
		song.VolumeAutomations = nil
		for i := range song.Tracks {
			song.Tracks[i].UseRse = false
		}
		readAlphaTabOracleFacts(t, "--multi-rest", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if !reflect.DeepEqual(got, original) {
			t.Fatalf("source preferences changed: got=%#v want=%#v", got, original)
		}
		if len(song.Tracks) != 2 {
			continue
		}
		for _, global := range []*bool{nil, ptrTo(false), ptrTo(true)} {
			for _, first := range []*bool{nil, ptrTo(false), ptrTo(true)} {
				for _, second := range []*bool{nil, ptrTo(false), ptrTo(true)} {
					song.Style.MultiRest = global
					song.Tracks[0].MultiRest = first
					song.Tracks[1].MultiRest = second
					want := original
					want.Global = boolRequest(global)
					want.Tracks = nil
					for i, value := range []*bool{first, second} {
						if boolRequest(value) {
							want.Tracks = append(want.Tracks, i)
						}
					}
					readAlphaTabOracleFacts(t, "--multi-rest", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("edited preferences changed: got=%#v want=%#v", got, want)
					}
				}
			}
		}
	}
	conformanceIndependentClaim(t, "field:ScoreStyle.MultiRest", claimAllStages("multi-rest", multiRestCase, multiRestValue)...)
}
func TestMultiRestAbsenceAndMalformed(t *testing.T) {
	song := parseTestFixture(t, multiRestFixture)
	cleanNotationProvenance(song)
	xml := string(conformanceBarreGPIF(t, mustStringNumberExport(t, song)))
	for _, raw := range [][]byte{nil, make([]byte, 8), {0, 0, 0, 1, 1, 0, 0, 0, 2, 1, 14, 0, 0, 0, 0}} {
		members := map[string][]byte{}
		if raw != nil {
			members["Content/PartConfiguration"] = raw
		}
		data := conformanceBackingArchive(t, xml, members)
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Tracks[0].MultiRest != nil || parsed.Tracks[1].MultiRest != nil {
			t.Fatal("absent track view fabricated a preference")
		}
		if len(raw) > 8 {
			if parsed.Style.MultiRest == nil || !*parsed.Style.MultiRest {
				t.Fatal("global preference lost")
			}
		} else if parsed.Style != nil && parsed.Style.MultiRest != nil {
			t.Fatal("absent global view fabricated a preference")
		}
	}
	raw := buildGP8PartConfiguration(song)
	for _, offset := range []int{4, 11, 17} {
		bad := bytes.Clone(raw)
		bad[offset] = 2
		data := conformanceBackingArchive(t, xml, map[string][]byte{"Content/PartConfiguration": bad})
		parsed, err := Parse(data)
		result, optionsErr := ParseWithOptions(data, ParseOptions{})
		if parsed != nil || result != nil || err == nil || optionsErr == nil || !strings.Contains(err.Error(), "invalid multi-rest byte 2") {
			t.Fatalf("invalid preference=%v/%v", err, optionsErr)
		}
	}
	// Pointer edits to one occurrence cannot change another parsed view.
	before, _ := json.Marshal(song.Tracks[1])
	*song.Tracks[0].MultiRest = true
	after, _ := json.Marshal(song.Tracks[1])
	if !bytes.Equal(before, after) {
		t.Fatal("track preference alias")
	}
}
