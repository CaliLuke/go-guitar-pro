// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestConformanceTuningLabels(t *testing.T) {
	runConformanceTuningLabels(newConformanceRun(t))
}

func runConformanceTuningLabels(run *conformanceRun) {
	t := run.t
	song := conformanceTuningLabelSong(t)
	first := &song.Tracks[0].Staves[0]
	second := &song.Tracks[0].Staves[1]
	wantStrings := []GuitarString{{Number: 1, Value: 46}, {Number: 2, Value: 41}}

	run.Preserved("Staff.TuningName", first.TuningName, "Author's tuning")
	run.Preserved("Staff.TuningName", second.TuningName, "")
	if !reflect.DeepEqual(first.Strings, wantStrings) || !reflect.DeepEqual(second.Strings, wantStrings) {
		t.Fatalf("identical staff tunings = %#v / %#v, want %#v", first.Strings, second.Strings, wantStrings)
	}

	beforeStrings := append([]GuitarString(nil), second.Strings...)
	beforeCapo := second.CapoFret
	second.TuningName = "Label-only edit"
	if !reflect.DeepEqual(second.Strings, beforeStrings) || second.CapoFret != beforeCapo {
		t.Fatalf("label edit changed pitches/order/capo: %#v capo %d", second.Strings, second.CapoFret)
	}
	second.TuningName = ""

	fallbackSource := strings.Replace(conformanceOwnershipGPIF,
		`<Track id="t0"><Name>Grand before</Name><Staves>`,
		`<Track id="t0"><Name>Grand before</Name><Properties><Property name="Tuning"><Pitches>41 46</Pitches><Label>Track fallback</Label></Property></Properties><Staves>`, 1)
	fallbackSource = strings.Replace(fallbackSource,
		`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property></Properties></Staff>`,
		`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches><Label></Label></Property></Properties></Staff>`, 1)
	fallbackSong, err := parseGPIF([]byte(fallbackSource))
	if err != nil {
		t.Fatal(err)
	}
	if got := fallbackSong.Tracks[0].Staves[0].TuningName; got != "Track fallback" {
		t.Errorf("pitches-only staff label = %q, want inherited track label", got)
	}
	if got := fallbackSong.Tracks[0].Staves[1].TuningName; got != "" {
		t.Errorf("explicit empty staff label = %q, want empty override", got)
	}

	name, found := gpifReadTuningName(gpifStaff{Properties: []gpifStaffProperty{{Name: "Tuning", Label: stringPointer("Dispatch label")}}})
	run.Dispatch("gpifReadTuningName:property.Name", []any{name, found}, []any{"Dispatch label", true})

	exportSong := conformanceTuningLabelExportSong(t)
	exportStrings := exportSong.Tracks[0].Staves[0].Strings
	data, report, err := ExportWithReport(exportSong, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("tuning label export = %v, %#v", err, report.Entries)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	gpifData := readZipMember(t, archive, "Content/score.gpif")
	if !strings.Contains(string(gpifData), "<Label></Label>") {
		t.Fatalf("GPIF does not contain an explicit empty tuning label:\n%s", gpifData)
	}
	var document gpifDocument
	if unmarshalErr := xml.Unmarshal(gpifData, &document); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	properties := document.Tracks.Tracks[0].Staves.Staff
	if len(properties) != 2 {
		t.Fatalf("exported staves = %d, want 2", len(properties))
	}
	for staffIndex, wantLabel := range []string{"Author's tuning", ""} {
		tuning := conformanceTuningProperty(t, properties[staffIndex])
		if tuning.Label == nil {
			t.Fatalf("staff %d tuning label is absent", staffIndex)
		}
		run.Wire("gpifStaffProperty.Pitches", tuning.Pitches, "41 46 51 56 60 65")
		run.Wire("gpifStaffProperty.Label", *tuning.Label, wantLabel)
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for staffIndex, wantLabel := range []string{"Author's tuning", ""} {
		staff := roundTrip.Tracks[0].Staves[staffIndex]
		if staff.TuningName != wantLabel || !reflect.DeepEqual(staff.Strings, exportStrings) || staff.CapoFret != 0 {
			t.Errorf("round-trip staff %d = %#v, want label %q tuning %#v capo 0", staffIndex, staff, wantLabel, exportStrings)
		}
	}
}

func conformanceTuningLabelSong(t *testing.T) *Song {
	t.Helper()
	source := strings.Replace(conformanceOwnershipGPIF,
		`<Pitches>40 45</Pitches>`,
		`<Pitches>41 46</Pitches><Label>Author's tuning</Label>`, 1)
	source = strings.Replace(source,
		`<Pitches>36 43</Pitches>`,
		`<Pitches>41 46</Pitches><Label></Label>`, 1)
	source = strings.Replace(source, `<Pitches>50</Pitches>`, `<Pitches>45 50</Pitches>`, 1)
	song, err := parseGPIF([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return song
}

func conformanceTuningLabelExportSong(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	track.Settings.Notation = true
	track.Settings.Tablature = true
	strings := []GuitarString{
		{Number: 1, Value: 65}, {Number: 2, Value: 60}, {Number: 3, Value: 56},
		{Number: 4, Value: 51}, {Number: 5, Value: 46}, {Number: 6, Value: 41},
	}
	track.Strings = append([]GuitarString(nil), strings...)
	first := track.Staves[0]
	first.Strings = append([]GuitarString(nil), strings...)
	first.TuningName = "Author's tuning"
	second := first
	second.Strings = append([]GuitarString(nil), strings...)
	second.TuningName = ""
	track.Staves = []Staff{first, second}
	return song
}

func conformanceTuningProperty(t *testing.T, staff gpifStaff) gpifStaffProperty {
	t.Helper()
	for _, property := range staff.Properties {
		if property.Name == "Tuning" {
			return property
		}
	}
	t.Fatal("staff has no tuning property")
	return gpifStaffProperty{}
}

func stringPointer(value string) *string {
	return &value
}
