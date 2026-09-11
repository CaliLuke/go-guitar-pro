// SPDX-License-Identifier: MIT

package goguitarpro

import "testing"

func TestGPIFPopulatesMeasureAndBeatTiming(t *testing.T) {
	t.Run("measure starts", func(t *testing.T) {
		song := parseTestFixture(t, "testdata/gp7/time-signatures.gp")
		want := []int64{960, 2880, 3360}
		for index := range want {
			if song.MeasureHeaders[index].Start != want[index] {
				t.Errorf("header %d start = %d, want %d", index, song.MeasureHeaders[index].Start, want[index])
			}
			if song.Tracks[0].Measures[index].Start != want[index] {
				t.Errorf("measure %d start = %d, want %d", index, song.Tracks[0].Measures[index].Start, want[index])
			}
		}
	})

	t.Run("beat starts", func(t *testing.T) {
		song := parseTestFixture(t, "testdata/gp7/notes.gp")
		beats := song.Tracks[0].Measures[0].Voices[0].Beats
		start := int64(960)
		for index := range beats {
			if beats[index].Start == nil || *beats[index].Start != start {
				t.Errorf("beat %d start = %v, want %d", index, beats[index].Start, start)
			}
			start += int64(beats[index].Duration.time())
		}
	})
}

func TestGPIFTimingStartsVoicesIndependently(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/multi-voice.gp")
	measure := &song.Tracks[0].Measures[0]
	if len(measure.Voices) < 2 {
		t.Fatalf("voices = %d, want at least 2", len(measure.Voices))
	}
	for voiceIndex := range measure.Voices {
		if len(measure.Voices[voiceIndex].Beats) == 0 {
			continue
		}
		if got := measure.Voices[voiceIndex].Beats[0].Start; got == nil || *got != measure.Start {
			t.Errorf("voice %d first beat start = %v, want %d", voiceIndex, got, measure.Start)
		}
	}
}

func TestGPIFTimingUsesTupletPlayedLength(t *testing.T) {
	assertFirstVoiceBeatStarts(t, "testdata/gp7/tuplets.gp", []int64{960, 1280, 1600})
}

func TestGPIFTimingKeepsAttachedGraceOutsideBeatTimeline(t *testing.T) {
	assertFirstVoiceBeatStarts(t, "testdata/gp7/grace.gp", []int64{960, 1920, 2880})
}

func assertFirstVoiceBeatStarts(t *testing.T, fixture string, want []int64) {
	t.Helper()
	song := parseTestFixture(t, fixture)
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	for index := range want {
		if beats[index].Start == nil || *beats[index].Start != want[index] {
			t.Errorf("%s beat %d start = %v, want %d", fixture, index, beats[index].Start, want[index])
		}
	}
}

func TestGPIFTimingUsesPickupContentLength(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/anacrusis.gp")
	want := []int64{960, 2880}
	for index := range want {
		if song.MeasureHeaders[index].Start != want[index] {
			t.Errorf("pickup header %d start = %d, want %d", index, song.MeasureHeaders[index].Start, want[index])
		}
		if song.Tracks[0].Measures[index].Start != want[index] {
			t.Errorf("pickup measure %d start = %d, want %d", index, song.Tracks[0].Measures[index].Start, want[index])
		}
	}
}

func TestTimingAllowsEmptyPickup(t *testing.T) {
	song := Song{
		Anacrusis:      true,
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader(), defaultMeasureHeader()},
		Tracks: []Track{{Measures: []Measure{
			{HeaderIndex: 0, Voices: []Voice{{}}},
			{HeaderIndex: 1, Voices: []Voice{{Beats: []Beat{defaultBeat()}}}},
		}}},
	}
	if err := song.finalizeTiming(); err != nil {
		t.Fatal(err)
	}

	if got := song.MeasureHeaders[1].Start; got != DurationQuarterTime {
		t.Fatalf("measure after empty pickup starts at %d, want %d", got, DurationQuarterTime)
	}
}

func TestTimingDoesNotAdvanceOrphanGraceBeat(t *testing.T) {
	grace := defaultBeat()
	grace.Notes = []Note{{String: 1}}
	target := defaultBeat()
	target.Notes = []Note{{String: 2}}
	orphans := gpifApplyPendingGrace(
		&target,
		[]gpifPendingGrace{{beat: grace}},
		false, nil,
	)
	if len(orphans) != 1 || !orphans[0].isGrace {
		t.Fatalf("orphan grace beats = %#v, want one timing-marked grace beat", orphans)
	}

	regular := defaultBeat()
	beats := append([]Beat{regular}, orphans...)
	beats = append(beats, target)
	song := Song{
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader()},
		Tracks: []Track{{Measures: []Measure{{
			HeaderIndex: 0,
			Voices:      []Voice{{Beats: beats}},
		}}}},
	}
	if err := song.finalizeTiming(); err != nil {
		t.Fatal(err)
	}

	got := song.Tracks[0].Measures[0].Voices[0].Beats
	if *got[1].Start != 1920 || *got[2].Start != 1920 {
		t.Fatalf("orphan grace and following beat starts = %d, %d, want 1920, 1920", *got[1].Start, *got[2].Start)
	}
}

func TestBinaryAndGPIFTimingUseSameOrigin(t *testing.T) {
	for _, fixture := range []string{"testdata/gp5/notes.gp5", "testdata/gp7/notes.gp"} {
		song := parseTestFixture(t, fixture)
		measure := &song.Tracks[0].Measures[0]
		if measure.Start != DurationQuarterTime {
			t.Errorf("%s measure start = %d, want %d", fixture, measure.Start, DurationQuarterTime)
		}
		if beat := measure.Voices[0].Beats[0]; beat.Start == nil || *beat.Start != DurationQuarterTime {
			t.Errorf("%s first beat start = %v, want %d", fixture, beat.Start, DurationQuarterTime)
		}
	}
}
