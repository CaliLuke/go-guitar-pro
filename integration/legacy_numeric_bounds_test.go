// SPDX-License-Identifier: MIT

package integration_test

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

type legacyNumericFixture struct {
	path, sha256 string
	fret, rse    int
	effectShort  bool
}

// Offsets were located by tracing the reader cursor on the complete, hash-bound
// source files. Fret count follows the two channel int32s and precedes capo.
// RSE is the first track instrument, after humanize and six metadata int32s.
var legacyNumericFixtures = []legacyNumericFixture{
	{"../testdata/gp3/notes.gp3", "a9ae6bb06ed259016e7a138a1b17d850491e9e2e8510c235df28a9ec81a706f1", 949, 0, false},
	{"../testdata/gp4/notes.gp4", "4f918d90aed2bdd12d229e612637b6f470a409771c805cc648c0682028f9125e", 996, 0, false},
	{"../testdata/gp5/serenade.gp5", "ce8090c55e47c68c06f68f9795dbea787aead29e3d0773df7255ef7752b635ef", 1681, 1722, true},
	{"../testdata/gp5/RSE.gp5", "8e9e712724de18f2f0d8235cb1aaade49c21eccc48a6cefa5858f45042738d31", 1345, 1386, false},
}

func readLegacyNumericFixture(t *testing.T, fixture legacyNumericFixture) []byte {
	t.Helper()
	data, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != fixture.sha256 {
		t.Fatalf("fixture hash = %s", got)
	}
	if binary.LittleEndian.Uint32(data[fixture.fret:]) != 24 {
		t.Fatal("located fret count changed")
	}
	return data
}

func assertLegacyNumericParse(t *testing.T, data []byte, expected *gp.Song, wantError string) {
	t.Helper()
	song, err := gp.Parse(data)
	result, optionsErr := gp.ParseWithOptions(data, gp.ParseOptions{})
	if wantError != "" {
		for _, got := range []error{err, optionsErr} {
			var parseError *gp.ParseError
			if !errors.As(got, &parseError) || got.Error() != wantError {
				t.Fatalf("error = %v, want %q", got, wantError)
			}
		}
		if song != nil || result != nil {
			t.Fatal("range error returned a partial score")
		}
		return
	}
	if err != nil || optionsErr != nil {
		t.Fatalf("valid sibling: %v / %v", err, optionsErr)
	}
	// Compare all fields, including following tracks, tuning, capo, channels,
	// flags and parsed measures. Only the located authored value may differ.
	if !reflect.DeepEqual(song, expected) || !reflect.DeepEqual(result.Song, expected) {
		t.Fatal("located numeric edit changed unrelated score data")
	}
}

func TestLegacyFretCountPublicBounds(t *testing.T) {
	for _, fixture := range legacyNumericFixtures {
		for _, value := range []int32{0, 31, 255, -1, 256, 280, 2147483647} {
			t.Run(fmt.Sprintf("%s/%d", fixture.path, value), func(t *testing.T) {
				data := readLegacyNumericFixture(t, fixture)
				expected, err := gp.Parse(data)
				if err != nil {
					t.Fatal(err)
				}
				binary.LittleEndian.PutUint32(data[fixture.fret:], uint32(value))
				wantError := ""
				if value < 0 || value > 255 {
					wantError = fmt.Sprintf("unparseable Guitar Pro file: parsing binary GP: reading tracks: reading track 1 fret count: value %d is outside public uint8 range 0..255", value)
				} else {
					expected.Tracks[0].FretCount = uint8(value)
				}
				assertLegacyNumericParse(t, data, expected, wantError)
			})
		}
	}
}

func TestRSEPublicSignedBounds(t *testing.T) {
	for _, fixture := range legacyNumericFixtures[2:] {
		for index, field := range []string{"Instrument", "Unknown", "SoundBank", "EffectNumber"} {
			for _, value := range []int32{-32768, -1, 32767, -32769, 32768, 65536} {
				if fixture.effectShort && index == 3 && (value < -32768 || value > 32767) {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%d", fixture.path, field, value), func(t *testing.T) {
					data := readLegacyNumericFixture(t, fixture)
					expected, err := gp.Parse(data)
					if err != nil {
						t.Fatal(err)
					}
					offset := fixture.rse + 4*index
					if fixture.effectShort && index == 3 {
						binary.LittleEndian.PutUint16(data[offset:], uint16(value))
					} else {
						binary.LittleEndian.PutUint32(data[offset:], uint32(value))
					}
					wantError := ""
					if value < -32768 || value > 32767 {
						wantError = fmt.Sprintf("unparseable Guitar Pro file: parsing binary GP: reading tracks: reading RSE %s: value %d is outside public int16 range -32768..32767", field, value)
					} else {
						instrument := &expected.Tracks[0].Rse.Instrument
						fields := []*int16{&instrument.Instrument, &instrument.Unknown, &instrument.SoundBank, &instrument.EffectNumber}
						*fields[index] = int16(value)
					}
					assertLegacyNumericParse(t, data, expected, wantError)
				})
			}
		}
	}
}
