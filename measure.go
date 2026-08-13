// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

const maxVoices = 2

// Measure represents a measure within a track.
type Measure struct {
	Voices        []Voice
	Number        int
	Start         int64
	TrackIndex    int
	HeaderIndex   int
	TimeSignature TimeSignature
	KeySignature  KeySignature
	HasDoubleBar  bool
	Clef          MeasureClef
	LineBreak     LineBreak
}

func defaultMeasure() Measure {
	return Measure{
		Number:        1,
		Start:         DurationQuarterTime,
		TimeSignature: defaultTimeSignature(),
	}
}

func (s *Song) readMeasures(c *cursor) error {
	start := DurationQuarterTime
	for h := 0; h < len(s.MeasureHeaders); h++ {
		s.MeasureHeaders[h].Start = start
		for t := 0; t < len(s.Tracks); t++ {
			s.currentTrack = &t
			m := defaultMeasure()
			m.TrackIndex = t
			m.HeaderIndex = h
			m.Start = start

			posBefore := c.pos
			if versionLessThan(s.Version.Number, [3]byte{5, 0, 0}) {
				if err := s.readMeasure(c, &m, t); err != nil {
					return fmt.Errorf("measure %d track %d (pos before=%d after=%d): %w", h+1, t+1, posBefore, c.pos, err)
				}
			} else {
				if err := s.readMeasureV5(c, &m, t); err != nil {
					return fmt.Errorf("measure %d track %d (pos before=%d after=%d): %w", h+1, t+1, posBefore, c.pos, err)
				}
			}
			s.Tracks[t].Measures = append(s.Tracks[t].Measures, m)
		}
		start += s.MeasureHeaders[h].length()
	}
	s.currentTrack = nil
	return nil
}

func (s *Song) readMeasure(c *cursor, measure *Measure, trackIndex int) error {
	voice := Voice{}
	start := measure.Start
	if err := s.readVoice(c, &voice, &start, trackIndex); err != nil {
		return err
	}
	measure.Voices = append(measure.Voices, voice)
	return nil
}

func (s *Song) readMeasureV5(c *cursor, measure *Measure, trackIndex int) error {
	start := measure.Start
	for v := 0; v < maxVoices; v++ {
		voice := Voice{}
		if err := s.readVoice(c, &voice, &start, trackIndex); err != nil {
			return err
		}
		measure.Voices = append(measure.Voices, voice)
	}
	if c.remaining() > 0 {
		lb, err := c.readByte()
		if err != nil {
			return err
		}
		measure.LineBreak = LineBreak(lb)
	}
	return nil
}

func (s *Song) readVoice(c *cursor, voice *Voice, start *int64, trackIndex int) error {
	beatCount, err := c.readCount(2, "beat count")
	if err != nil {
		return err
	}
	var beatPositions []int
	for i := 0; i < beatCount; i++ {
		beatPositions = append(beatPositions, c.pos)
		var duration int64
		if versionLessThan(s.Version.Number, [3]byte{5, 0, 0}) {
			duration, err = s.readBeat(c, voice, *start, trackIndex)
		} else {
			duration, err = s.readBeatV5(c, voice, start, trackIndex)
		}
		if err != nil {
			return fmt.Errorf("beat %d/%d (positions=%v endpos=%d): %w", i+1, beatCount, beatPositions, c.pos, err)
		}
		*start += duration
	}
	return nil
}
