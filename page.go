// SPDX-License-Identifier: MIT

package goguitarpro

// PageSetup describes how the document is rendered.
type PageSetup struct {
	Words               string
	Music               string
	PageNumber          string
	Copyright           string
	WordAndMusic        string
	Artist              string
	Album               string
	Title               string
	Subtitle            string
	PageHeight          int32
	ScoreSizeProportion float32
	PageWidth           int32
	MarginBottom        int32
	MarginTop           int32
	MarginRight         int32
	MarginLeft          int32
	HeaderAndFooter     uint16
}

func (s *Song) readPageSetup(c *cursor) error {
	var err error
	s.PageSetup.PageWidth, err = c.readInt()
	if err != nil {
		return err
	}
	s.PageSetup.PageHeight, err = c.readInt()
	if err != nil {
		return err
	}
	s.PageSetup.MarginLeft, err = c.readInt()
	if err != nil {
		return err
	}
	s.PageSetup.MarginRight, err = c.readInt()
	if err != nil {
		return err
	}
	s.PageSetup.MarginTop, err = c.readInt()
	if err != nil {
		return err
	}
	s.PageSetup.MarginBottom, err = c.readInt()
	if err != nil {
		return err
	}
	ssp, err := c.readInt()
	if err != nil {
		return err
	}
	s.PageSetup.ScoreSizeProportion = float32(ssp) / 100.0
	hf, err := c.readShort()
	if err != nil {
		return err
	}
	s.PageSetup.HeaderAndFooter = uint16(hf)
	s.PageSetup.Title, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.Subtitle, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.Artist, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.Album, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.Words, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.Music, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.WordAndMusic, err = c.readIntSizeString()
	if err != nil {
		return err
	}
	copy1, err := c.readIntSizeString()
	if err != nil {
		return err
	}
	copy2, err := c.readIntSizeString()
	if err != nil {
		return err
	}
	s.PageSetup.Copyright = copy1 + "\n" + copy2
	s.PageSetup.PageNumber, err = c.readIntSizeString()
	return err
}
