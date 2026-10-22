// SPDX-License-Identifier: MIT

package integration_test

import (
	"context"
	"encoding/binary"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

const gpxBackwardSectorTestEnv = "GO_GUITAR_PRO_TEST_GPX_BACKWARD_SECTOR"

func TestParseTerminatesOnGPXBackwardSectorPointer(t *testing.T) {
	if os.Getenv(gpxBackwardSectorTestEnv) == "1" {
		if _, err := guitarpro.Parse(gpxWithBackwardSectorPointer()); err == nil || !strings.Contains(err.Error(), "no score.gpif found in GPX container") {
			t.Fatalf("Parse error = %v, want missing score.gpif after bounded container scan", err)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestParseTerminatesOnGPXBackwardSectorPointer$")
	cmd.Env = append(os.Environ(), gpxBackwardSectorTestEnv+"=1")
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("Parse did not return for a backward GPX sector pointer: %s", output)
	}
	if err != nil {
		t.Fatalf("public Parse reproduction failed: %v\n%s", err, output)
	}
}

func TestParseRejectsTruncatedGPXFileEntry(t *testing.T) {
	data := gpxWithTruncatedScoreGPIF()
	if _, err := guitarpro.Parse(data); err == nil || !strings.Contains(err.Error(), `file "score.gpif" contains 4096 of 8192 declared bytes`) {
		t.Fatalf("Parse error = %v, want declared-size mismatch for truncated GPX member", err)
	}
}

func gpxWithTruncatedScoreGPIF() []byte {
	const sectorSize = 0x1000
	const gpif = `<GPIF><GPVersion>8.1.5</GPVersion><Score><Title>Short payload</Title></Score><MasterTrack><Tracks>0</Tracks></MasterTrack><Tracks><Track id="0"><Name>Guitar</Name><Properties><Property name="Tuning"><Pitches>40 45 50 55 59 64</Pitches></Property></Properties><Staves><Staff><Properties><Property name="Tuning"><Pitches>40 45 50 55 59 64</Pitches></Property></Properties></Staff></Staves></Track></Tracks><MasterBars><MasterBar><Time>4/4</Time><Bars>0</Bars></MasterBar></MasterBars><Bars><Bar id="0"><Clef>G2</Clef><Voices>0</Voices></Bar></Bars><Voices><Voice id="0"><Beats>0</Beats></Voice></Voices><Beats><Beat id="0"><Rhythm ref="0"/></Beat></Beats><Notes/><Rhythms><Rhythm id="0"><NoteValue>Whole</NoteValue></Rhythm></Rhythms></GPIF>`
	data := make([]byte, 4+4*sectorSize)
	copy(data, "BCFS")
	block := data[4:]
	entryOffset := sectorSize
	binary.LittleEndian.PutUint32(block[entryOffset:], 2)
	copy(block[entryOffset+4:], "score.gpif")
	binary.LittleEndian.PutUint32(block[entryOffset+0x8c:], 2*sectorSize)
	binary.LittleEndian.PutUint32(block[entryOffset+0x94:], 2)
	dataOffset := 2 * sectorSize
	copy(block[dataOffset:], gpif)
	for index := dataOffset + len(gpif); index < 3*sectorSize; index++ {
		block[index] = ' '
	}
	return data
}

func gpxWithBackwardSectorPointer() []byte {
	const sectorSize = 0x1000
	data := make([]byte, 3*sectorSize+4)
	copy(data, "BCFS")
	block := data[4:]
	entryOffset := 2 * sectorSize
	binary.LittleEndian.PutUint32(block[entryOffset:], 2)
	copy(block[entryOffset+4:], "loop.gpif")
	binary.LittleEndian.PutUint32(block[entryOffset+0x8c:], 1)
	binary.LittleEndian.PutUint32(block[entryOffset+0x94:], 1)
	return data
}
