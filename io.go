// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"fmt"
	"math"

	"golang.org/x/text/encoding/charmap"
)

// cursor provides sequential binary reading from a byte slice.
type cursor struct {
	data []byte
	pos  int
}

func newCursor(data []byte) *cursor {
	return &cursor{data: data, pos: 0}
}

func (c *cursor) remaining() int {
	return len(c.data) - c.pos
}

func (c *cursor) readByte() (byte, error) {
	if c.pos >= len(c.data) {
		return 0, fmt.Errorf("unexpected end of data at offset %d", c.pos)
	}
	b := c.data[c.pos]
	c.pos++
	return b, nil
}

func (c *cursor) readSignedByte() (int8, error) {
	b, err := c.readByte()
	return int8(b), err
}

func (c *cursor) readBool() (bool, error) {
	b, err := c.readByte()
	return b != 0, err
}

func (c *cursor) readShort() (int16, error) {
	if c.pos+2 > len(c.data) {
		return 0, fmt.Errorf("unexpected end of data at offset %d", c.pos)
	}
	n := int16(binary.LittleEndian.Uint16(c.data[c.pos : c.pos+2]))
	c.pos += 2
	return n, nil
}

func (c *cursor) readInt() (int32, error) {
	if c.pos+4 > len(c.data) {
		return 0, fmt.Errorf("unexpected end of data at offset %d", c.pos)
	}
	n := int32(binary.LittleEndian.Uint32(c.data[c.pos : c.pos+4]))
	c.pos += 4
	return n, nil
}

func (c *cursor) readDouble() (float64, error) {
	if c.pos+8 > len(c.data) {
		return 0, fmt.Errorf("unexpected end of data at offset %d", c.pos)
	}
	bits := binary.LittleEndian.Uint64(c.data[c.pos : c.pos+8])
	c.pos += 8
	return math.Float64frombits(bits), nil
}

func (c *cursor) skip(n int) error {
	if c.pos+n > len(c.data) {
		return fmt.Errorf("unexpected end of data at offset %d, skip %d", c.pos, n)
	}
	c.pos += n
	return nil
}

// readString reads a string of 'size' bytes, using only 'length' of them.
func (c *cursor) readString(size int, length int) (string, error) {
	if c.pos+size > len(c.data) {
		return "", fmt.Errorf("unexpected end of data at offset %d, string size %d", c.pos, size)
	}
	if length > size {
		length = size
	}
	raw := c.data[c.pos : c.pos+length]
	// Try Windows-1252 decoding
	decoder := charmap.Windows1252.NewDecoder()
	decoded, err := decoder.Bytes(raw)
	if err != nil {
		// Fall back to raw bytes as UTF-8
		decoded = raw
	}
	c.pos += size
	return string(decoded), nil
}

// readIntSizeString reads a string whose length is stored as an int32.
func (c *cursor) readIntSizeString() (string, error) {
	n, err := c.readInt()
	if err != nil {
		return "", err
	}
	size := int(n)
	if size < 0 {
		return "", fmt.Errorf("negative string size %d at offset %d", size, c.pos)
	}
	return c.readString(size, size)
}

// readIntByteSizeString reads a string whose length+1 is stored as int32,
// followed by the actual length as a byte, then the character bytes.
func (c *cursor) readIntByteSizeString() (string, error) {
	n, err := c.readInt()
	if err != nil {
		return "", err
	}
	size := int(n) - 1
	if size < 0 {
		size = 0
	}
	return c.readByteSizeString(size)
}

// readByteSizeString reads a byte for actual length, then reads 'size' bytes using 'length' of them.
func (c *cursor) readByteSizeString(size int) (string, error) {
	length, err := c.readByte()
	if err != nil {
		return "", err
	}
	return c.readString(size, int(length))
}

// readColor reads 3 bytes (R, G, B) + 1 padding byte and returns packed int.
func (c *cursor) readColor() (int32, error) {
	r, err := c.readByte()
	if err != nil {
		return 0, err
	}
	g, err := c.readByte()
	if err != nil {
		return 0, err
	}
	b, err := c.readByte()
	if err != nil {
		return 0, err
	}
	if err := c.skip(1); err != nil {
		return 0, err
	}
	return int32(r)*65536 + int32(g)*256 + int32(b), nil
}

// Version info

type versionEntry struct {
	str       string
	number    [3]byte
	clipboard bool
}

var versions = []versionEntry{
	{number: [3]byte{3, 0, 0}, clipboard: false, str: "FICHIER GUITAR PRO v3.00"},
	{number: [3]byte{4, 0, 0}, clipboard: false, str: "FICHIER GUITAR PRO v4.00"},
	{number: [3]byte{4, 0, 6}, clipboard: false, str: "FICHIER GUITAR PRO v4.06"},
	{number: [3]byte{4, 0, 6}, clipboard: true, str: "CLIPBOARD GUITAR PRO 4.0 [c6]"},
	{number: [3]byte{5, 0, 0}, clipboard: false, str: "FICHIER GUITAR PRO v5.00"},
	{number: [3]byte{5, 1, 0}, clipboard: false, str: "FICHIER GUITAR PRO v5.10"},
	{number: [3]byte{5, 2, 0}, clipboard: false, str: "FICHIER GUITAR PRO v5.10"}, // sic
	{number: [3]byte{5, 0, 0}, clipboard: true, str: "CLIPBOARD GP 5.0"},
	{number: [3]byte{5, 1, 0}, clipboard: true, str: "CLIPBOARD GP 5.1"},
	{number: [3]byte{5, 2, 0}, clipboard: true, str: "CLIPBOARD GP 5.2"},
}

func (c *cursor) readVersionString() (Version, error) {
	s, err := c.readByteSizeString(30)
	if err != nil {
		return Version{}, fmt.Errorf("reading version: %w", err)
	}
	v := Version{Data: s, Number: [3]byte{5, 2, 0}}
	for _, entry := range versions {
		if s == entry.str {
			v.Number = entry.number
			v.Clipboard = entry.clipboard
			break
		}
	}
	return v, nil
}
