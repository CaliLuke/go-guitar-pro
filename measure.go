// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

const maxVoices = 2

// Measure represents a measure within a track.
type Measure struct {
	Voices []Voice
	Number int
	// Start matches the owning MeasureHeader display-time start in ticks.
	Start int64
	// ExactStart preserves fractional score ticks before Start is quantized.
	ExactStart ScoreTime
	TrackIndex int
	// StaffIndex is the zero-based staff index within the owning track.
	StaffIndex    int
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

func (s *Song) finalizeTiming() error {
	start, _ := NewScoreTime(DurationQuarterTime, 1)
	for headerIndex := range s.MeasureHeaders {
		s.MeasureHeaders[headerIndex].ExactStart = start
		s.MeasureHeaders[headerIndex].Start = start.FloorTicks()
		contentLength := ScoreTime{}
		for trackIndex := range s.Tracks {
			track := &s.Tracks[trackIndex]
			if len(track.Staves) == 0 {
				trackLength, err := finalizeMeasureTiming(track.Measures, headerIndex, start)
				if err != nil {
					return fmt.Errorf("track %d measure timing: %w", trackIndex, err)
				}
				contentLength = maxScoreTime(contentLength, trackLength)
				continue
			}
			for staffIndex := range track.Staves {
				staffLength, err := finalizeMeasureTiming(track.Staves[staffIndex].Measures, headerIndex, start)
				if err != nil {
					return fmt.Errorf("track %d staff %d measure timing: %w", trackIndex, staffIndex, err)
				}
				contentLength = maxScoreTime(
					contentLength,
					staffLength,
				)
			}
		}
		measureLength, err := s.MeasureHeaders[headerIndex].ExactLength()
		if err != nil {
			return fmt.Errorf("measure header %d length: %w", headerIndex, err)
		}
		if headerIndex == 0 && s.Anacrusis {
			measureLength = contentLength
		}
		next, err := start.Add(measureLength)
		if err != nil {
			return fmt.Errorf("measure header %d end: %w", headerIndex, err)
		}
		start = next
	}
	return nil
}

func finalizeMeasureTiming(measures []Measure, headerIndex int, start ScoreTime) (ScoreTime, error) {
	contentLength := ScoreTime{}
	for measureIndex := range measures {
		measure := &measures[measureIndex]
		if measure.HeaderIndex != headerIndex {
			continue
		}
		measure.ExactStart = start
		measure.Start = start.FloorTicks()
		for voiceIndex := range measure.Voices {
			voiceStart := start
			for beatIndex := range measure.Voices[voiceIndex].Beats {
				beat := &measure.Voices[voiceIndex].Beats[beatIndex]
				beatStart := voiceStart.FloorTicks()
				beat.Start = &beatStart
				exactBeatStart := voiceStart
				beat.ExactStart = &exactBeatStart
				if !beat.isGrace {
					duration, err := beat.Duration.ExactScoreTime()
					if err != nil {
						return ScoreTime{}, fmt.Errorf("measure %d voice %d beat %d duration: %w", measureIndex, voiceIndex, beatIndex, err)
					}
					next, err := voiceStart.Add(duration)
					if err != nil {
						return ScoreTime{}, fmt.Errorf("measure %d voice %d beat %d end: %w", measureIndex, voiceIndex, beatIndex, err)
					}
					voiceStart = next
				}
			}
			voiceLength, err := voiceStart.Subtract(start)
			if err != nil {
				return ScoreTime{}, fmt.Errorf("measure %d voice %d length: %w", measureIndex, voiceIndex, err)
			}
			contentLength = maxScoreTime(contentLength, voiceLength)
		}
	}
	return contentLength, nil
}

func maxScoreTime(left, right ScoreTime) ScoreTime {
	if left.Compare(right) >= 0 {
		return left
	}
	return right
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
