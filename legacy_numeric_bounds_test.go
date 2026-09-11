// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestRSEInstrumentReaderBounds(t *testing.T) {
	for _, version := range [][3]byte{{5, 0, 0}, {5, 1, 0}} {
		for index, field := range []string{"Instrument", "Unknown", "SoundBank", "EffectNumber"} {
			for _, value := range []int32{-32768, -1, 32767, -32769, 32768, 65536} {
				short := version == [3]byte{5, 0, 0} && index == 3
				invalid := value < -32768 || value > 32767
				if short && invalid {
					continue
				}
				t.Run(fmt.Sprintf("%v/%s/%d", version, field, value), func(t *testing.T) {
					data := make([]byte, 17)
					if short {
						binary.LittleEndian.PutUint16(data[12:], uint16(value))
					} else {
						binary.LittleEndian.PutUint32(data[index*4:], uint32(value))
					}
					end := 16
					if version == [3]byte{5, 0, 0} {
						end = 15
						data[14] = 0xa5
					}
					data[end] = 0x6e
					song := Song{Version: Version{Number: version}}
					c := newCursor(data)
					got, err := song.readRseInstrument(c)
					if invalid {
						want := fmt.Sprintf("reading RSE %s: value %d is outside public int16 range -32768..32767", field, value)
						if err == nil || err.Error() != want {
							t.Fatalf("range error=%v, want %q", err, want)
						}
						return
					}
					fields := []int16{got.Instrument, got.Unknown, got.SoundBank, got.EffectNumber}
					for i, actual := range fields {
						want := int16(0)
						if i == index {
							want = int16(value)
						}
						if actual != want {
							t.Fatalf("field %d=%d, want %d", i, actual, want)
						}
					}
					sentinel, readErr := c.readByte()
					if err != nil || readErr != nil || sentinel != 0x6e || c.pos != end+1 {
						t.Fatalf("record alignment pos=%d sentinel=%x error=%v/%v", c.pos, sentinel, err, readErr)
					}
				})
			}
		}
	}
}

func TestRSEInstrumentReaderTruncation(t *testing.T) {
	for _, version := range [][3]byte{{5, 0, 0}, {5, 1, 0}} {
		end := 16
		if version == [3]byte{5, 0, 0} {
			end = 15
		}
		for length := 0; length < end; length++ {
			t.Run(fmt.Sprintf("%v/%d", version, length), func(t *testing.T) {
				song := Song{Version: Version{Number: version}}
				_, err := song.readRseInstrument(newCursor(make([]byte, length)))
				field := []string{"Instrument", "Unknown", "SoundBank", "EffectNumber"}[min(length/4, 3)]
				if version == [3]byte{5, 0, 0} && length == 14 {
					field += " padding"
				}
				if err == nil || !strings.HasPrefix(err.Error(), "reading RSE "+field+": unexpected end of data at offset ") {
					t.Fatalf("truncation error=%v", err)
				}
			})
		}
	}
}

func TestTrackFretCountReaderTruncation(t *testing.T) {
	for length := 0; length < 4; length++ {
		_, err := readTrackFretCount(newCursor(make([]byte, length)), 2)
		if err == nil || err.Error() != "reading track 2 fret count: unexpected end of data at offset 0" {
			t.Fatalf("truncation error=%v", err)
		}
	}
}
