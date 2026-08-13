// SPDX-License-Identifier: MIT

package goguitarpro

var sharpNotes = [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// PitchClass represents a pitch class.
type PitchClass struct {
	Note       string
	Just       int8
	Accidental int8
	Value      int8
	Sharp      bool
}

func pitchClassFrom(just int8, accidental *int8, sharp *bool) PitchClass {
	p := PitchClass{Just: just, Sharp: true}
	var pitch, acc int8

	if accidental != nil {
		pitch = just
		acc = *accidental
	} else {
		value := just % 12
		if value < 0 {
			value += 12
		}
		p.Note = sharpNotes[value]
		if len(p.Note) > 1 && p.Note[len(p.Note)-1] == 'b' {
			acc = -1
			p.Sharp = false
		} else if len(p.Note) > 1 && p.Note[len(p.Note)-1] == '#' {
			acc = 1
		}
		pitch = value - acc
	}
	p.Just = pitch % 12
	p.Accidental = acc
	p.Value = p.Just + acc
	if sharp == nil {
		p.Sharp = p.Accidental >= 0
	} else {
		p.Sharp = *sharp
	}
	return p
}

// Chord represents a chord annotation for beats.
type Chord struct {
	FirstFret  *uint8
	Ninth      *ChordAlteration
	Root       *PitchClass
	Fifth      *ChordAlteration
	Extension  *ChordExtension
	Bass       *PitchClass
	Tonality   *ChordAlteration
	Add        *bool
	Sharp      *bool
	NewFormat  *bool
	Kind       *ChordType
	Eleventh   *ChordAlteration
	Show       *bool
	Name       string
	Strings    []int8
	Barres     []Barre
	Omissions  []bool
	Fingerings []Fingering
	Length     uint8
}

// Barre represents a single barre.
type Barre struct {
	Fret  int8
	Start int8
	End   int8
}

// readChord reads a chord diagram.
func (s *Song) readChord(c *cursor, stringCount uint8) (Chord, error) {
	ch := Chord{Length: stringCount}
	ch.Strings = make([]int8, stringCount)
	for i := range ch.Strings {
		ch.Strings[i] = -1
	}

	isNew, err := c.readBool()
	if err != nil {
		return ch, err
	}
	ch.NewFormat = &isNew

	if isNew {
		if s.Version.Number[0] == 3 {
			return s.readNewFormatChordV3(c, ch)
		}
		return s.readNewFormatChordV4(c, ch)
	}
	return s.readOldFormatChord(c, ch)
}

func (s *Song) readOldFormatChord(c *cursor, ch Chord) (Chord, error) {
	name, err := c.readIntSizeString()
	if err != nil {
		return ch, err
	}
	ch.Name = name
	ff32, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ff := uint8(ff32)
	ch.FirstFret = &ff
	if ff > 0 {
		for i := 0; i < 6; i++ {
			fret32, err := c.readInt()
			if err != nil {
				return ch, err
			}
			if i < len(ch.Strings) {
				ch.Strings[i] = int8(fret32)
			}
		}
	}
	return ch, nil
}

func (s *Song) readNewFormatChordV3(c *cursor, ch Chord) (Chord, error) {
	sharp, err := c.readBool()
	if err != nil {
		return ch, err
	}
	ch.Sharp = &sharp
	if skipErr := c.skip(3); skipErr != nil {
		return ch, skipErr
	}
	rootVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	root := pitchClassFrom(int8(rootVal), nil, ch.Sharp)
	ch.Root = &root
	kindVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	kind := ChordType(kindVal)
	ch.Kind = &kind
	extVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ext := ChordExtension(extVal)
	ch.Extension = &ext
	bassVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	bass := pitchClassFrom(int8(bassVal), nil, ch.Sharp)
	ch.Bass = &bass
	tonVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ton := ChordAlteration(tonVal)
	ch.Tonality = &ton
	add, err := c.readBool()
	if err != nil {
		return ch, err
	}
	ch.Add = &add
	name, err := c.readByteSizeString(22)
	if err != nil {
		return ch, err
	}
	ch.Name = name
	fifthVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	fifth := ChordAlteration(fifthVal)
	ch.Fifth = &fifth
	ninthVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ninth := ChordAlteration(ninthVal)
	ch.Ninth = &ninth
	eleventhVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	eleventh := ChordAlteration(eleventhVal)
	ch.Eleventh = &eleventh
	ffVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ffU := uint8(ffVal)
	ch.FirstFret = &ffU
	for i := 0; i < 6; i++ {
		fret, fretErr := c.readInt()
		if fretErr != nil {
			return ch, fretErr
		}
		if i < len(ch.Strings) {
			ch.Strings[i] = int8(fret)
		}
	}
	barreCount, err := c.readInt()
	if err != nil {
		return ch, err
	}
	barreFrets := make([]int32, 2)
	barreStarts := make([]int32, 2)
	barreEnds := make([]int32, 2)
	for i := 0; i < 2; i++ {
		barreFrets[i], err = c.readInt()
		if err != nil {
			return ch, err
		}
	}
	for i := 0; i < 2; i++ {
		barreStarts[i], err = c.readInt()
		if err != nil {
			return ch, err
		}
	}
	for i := 0; i < 2; i++ {
		barreEnds[i], err = c.readInt()
		if err != nil {
			return ch, err
		}
	}
	for i := 0; i < int(barreCount) && i < 2; i++ {
		ch.Barres = append(ch.Barres, Barre{
			Fret:  int8(barreFrets[i]),
			Start: int8(barreStarts[i]),
			End:   int8(barreEnds[i]),
		})
	}
	for i := 0; i < 7; i++ {
		om, omissionErr := c.readBool()
		if omissionErr != nil {
			return ch, omissionErr
		}
		ch.Omissions = append(ch.Omissions, om)
	}
	if skipErr := c.skip(1); skipErr != nil {
		return ch, skipErr
	}
	return ch, nil
}

func (s *Song) readNewFormatChordV4(c *cursor, ch Chord) (Chord, error) {
	sharp, err := c.readBool()
	if err != nil {
		return ch, err
	}
	ch.Sharp = &sharp
	if skipErr := c.skip(3); skipErr != nil {
		return ch, skipErr
	}
	rootByte, err := c.readByte()
	if err != nil {
		return ch, err
	}
	root := pitchClassFrom(int8(rootByte), nil, ch.Sharp)
	ch.Root = &root
	kindByte, err := c.readByte()
	if err != nil {
		return ch, err
	}
	kind := ChordType(kindByte)
	ch.Kind = &kind
	extByte, err := c.readByte()
	if err != nil {
		return ch, err
	}
	ext := ChordExtension(extByte)
	ch.Extension = &ext
	bassVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	bass := pitchClassFrom(int8(bassVal), nil, ch.Sharp)
	ch.Bass = &bass
	tonVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ton := ChordAlteration(tonVal)
	ch.Tonality = &ton
	add, err := c.readBool()
	if err != nil {
		return ch, err
	}
	ch.Add = &add
	name, err := c.readByteSizeString(22)
	if err != nil {
		return ch, err
	}
	ch.Name = name
	fifthByte, err := c.readByte()
	if err != nil {
		return ch, err
	}
	fifth := ChordAlteration(fifthByte)
	ch.Fifth = &fifth
	ninthByte, err := c.readByte()
	if err != nil {
		return ch, err
	}
	ninth := ChordAlteration(ninthByte)
	ch.Ninth = &ninth
	eleventhByte, err := c.readByte()
	if err != nil {
		return ch, err
	}
	eleventh := ChordAlteration(eleventhByte)
	ch.Eleventh = &eleventh
	ffVal, err := c.readInt()
	if err != nil {
		return ch, err
	}
	ffU := uint8(ffVal)
	ch.FirstFret = &ffU
	for i := 0; i < 7; i++ {
		fret, fretErr := c.readInt()
		if fretErr != nil {
			return ch, fretErr
		}
		if i < len(ch.Strings) {
			ch.Strings[i] = int8(fret)
		}
	}
	barreCount, err := c.readByte()
	if err != nil {
		return ch, err
	}
	barreFrets := make([]byte, 5)
	barreStarts := make([]byte, 5)
	barreEnds := make([]byte, 5)
	for i := 0; i < 5; i++ {
		barreFrets[i], err = c.readByte()
		if err != nil {
			return ch, err
		}
	}
	for i := 0; i < 5; i++ {
		barreStarts[i], err = c.readByte()
		if err != nil {
			return ch, err
		}
	}
	for i := 0; i < 5; i++ {
		barreEnds[i], err = c.readByte()
		if err != nil {
			return ch, err
		}
	}
	for i := 0; i < int(barreCount) && i < 5; i++ {
		ch.Barres = append(ch.Barres, Barre{
			Fret:  int8(barreFrets[i]),
			Start: int8(barreStarts[i]),
			End:   int8(barreEnds[i]),
		})
	}
	for i := 0; i < 7; i++ {
		om, omissionErr := c.readBool()
		if omissionErr != nil {
			return ch, omissionErr
		}
		ch.Omissions = append(ch.Omissions, om)
	}
	if skipErr := c.skip(1); skipErr != nil {
		return ch, skipErr
	}
	for i := 0; i < 7; i++ {
		f, fingeringErr := c.readSignedByte()
		if fingeringErr != nil {
			return ch, fingeringErr
		}
		ch.Fingerings = append(ch.Fingerings, Fingering(f))
	}
	show, err := c.readBool()
	if err != nil {
		return ch, err
	}
	ch.Show = &show
	return ch, nil
}
