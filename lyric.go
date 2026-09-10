// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// Lyrics holds lyrics data.
type Lyrics struct {
	Lines []LyricLine
	// TrackIndex is the zero-based track that receives these score-level lyrics.
	// A value of -1 means that the source did not select a track.
	TrackIndex int
}

// LyricLine represents a single lyric line.
type LyricLine struct {
	Text string
	// StartMeasureIndex is the zero-based measure where this line starts.
	// A value of -1 means that the source did not place the line.
	StartMeasureIndex int
}

// TrackLyricLine is one GPIF lyric line attached to a track.
type TrackLyricLine struct {
	Text string
	// Offset is the zero-based measure offset for this line within the track.
	Offset int
}

func (s *Song) readLyrics(c *cursor) (Lyrics, error) {
	tc, err := c.readInt()
	if err != nil {
		return Lyrics{}, err
	}
	if tc < 0 {
		return Lyrics{}, fmt.Errorf("lyrics track choice %d must be non-negative", tc)
	}
	lyrics := Lyrics{TrackIndex: int(tc) - 1}
	for lineIndex := range 5 {
		sm, err := c.readInt()
		if err != nil {
			return lyrics, err
		}
		if sm < 0 {
			return lyrics, fmt.Errorf("lyric line %d starting measure %d must be non-negative", lineIndex, sm)
		}
		text, err := c.readIntSizeString()
		if err != nil {
			return lyrics, err
		}
		lyrics.Lines = append(lyrics.Lines, LyricLine{
			StartMeasureIndex: int(sm) - 1,
			Text:              text,
		})
	}
	return lyrics, nil
}
