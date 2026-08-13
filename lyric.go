// SPDX-License-Identifier: MIT

package goguitarpro

// Lyrics holds lyrics data.
type Lyrics struct {
	Lines       []LyricLine
	TrackChoice uint8
}

// LyricLine represents a single lyric line.
type LyricLine struct {
	Text            string
	StartingMeasure uint16
	Number          uint8
}

func (s *Song) readLyrics(c *cursor) (Lyrics, error) {
	tc, err := c.readInt()
	if err != nil {
		return Lyrics{}, err
	}
	lyrics := Lyrics{TrackChoice: uint8(tc)}
	for i := uint8(0); i < 5; i++ {
		sm, err := c.readInt()
		if err != nil {
			return lyrics, err
		}
		text, err := c.readIntSizeString()
		if err != nil {
			return lyrics, err
		}
		lyrics.Lines = append(lyrics.Lines, LyricLine{
			Number:          i,
			StartingMeasure: uint16(sm),
			Text:            text,
		})
	}
	return lyrics, nil
}
