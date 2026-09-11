// SPDX-License-Identifier: MIT

package integration_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

const protectionEvidence = "../conformance/capabilities/evidence/protection/"

type protectionFixture struct {
	File        string `json:"file"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
	Size        int    `json:"size"`
	PrefixHex   string `json:"prefix_hex"`
	IsZIP       bool   `json:"is_zip"`
	ScorePrefix string `json:"score_gpif_prefix_hex"`
	Members     []struct {
		Name        string `json:"name"`
		Flags       uint16 `json:"flags"`
		Compression uint16 `json:"compression"`
		Size        uint64 `json:"size"`
		Readable    bool   `json:"readable"`
		SHA256      string `json:"sha256"`
	} `json:"members"`
}

type protectionOutcome struct {
	File            string  `json:"file"`
	API             string  `json:"api"`
	ScoreReturned   bool    `json:"score_returned"`
	ErrorType       *string `json:"error_type"`
	ErrorMessage    *string `json:"error_message"`
	ErrorMessageHex string  `json:"error_message_hex,omitempty"`
}

func TestGP8ProtectionEvidence(t *testing.T) {
	var authoring struct {
		Files []protectionFixture `json:"files"`
	}
	readProtectionReceipt(t, "authoring-receipt.json", &authoring)
	var outcomes []protectionOutcome
	readProtectionReceipt(t, "go-results.json", &outcomes)
	if len(authoring.Files) != 4 || len(outcomes) != 8 {
		t.Fatal("protection package requires four fixtures and both Go API outcomes for each")
	}
	for _, fixture := range authoring.Files {
		t.Run(fixture.File, func(t *testing.T) {
			data := mustReadFixture(t, "../"+fixture.Path)
			verifyProtectionContainer(t, fixture, data)
			matched := 0
			for _, want := range outcomes {
				if want.File != fixture.Path {
					continue
				}
				matched++
				t.Run(want.API, func(t *testing.T) {
					var song *guitarpro.Song
					var err error
					switch want.API {
					case "Parse":
						song, err = guitarpro.Parse(data)
					case "ParseWithOptions":
						var result *guitarpro.ParseResult
						result, err = guitarpro.ParseWithOptions(data, guitarpro.ParseOptions{})
						if result != nil {
							song = result.Song
						}
					default:
						t.Fatalf("unknown public API %q", want.API)
					}
					got := protectionOutcome{File: want.File, API: want.API, ScoreReturned: song != nil}
					if err != nil {
						kind, message := fmt.Sprintf("%T", err), string([]rune(err.Error()))
						got.ErrorMessageHex = fmt.Sprintf("%x", err.Error())
						got.ErrorType, got.ErrorMessage = &kind, &message
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("public outcome = %#v, want %#v (error %v)", got, want, err)
					}
					if song != nil {
						verifyProtectionControl(t, song)
					}
				})
			}
			if matched != 2 {
				t.Fatalf("fixture has %d public outcomes, want 2", matched)
			}
		})
	}
	t.Run("malformed-control-provenance", func(t *testing.T) {
		plain := mustReadFixture(t, "../testdata/gp8/protection-unprotected.gp")
		broken := mustReadFixture(t, "../testdata/gp8/protection-truncated.gp")
		if !bytes.Equal(broken, plain[:64]) {
			t.Fatal("malformed control is not the first 64 bytes of the original control")
		}
	})
	t.Run("pinned-consumer", verifyProtectionAlphaTab)
}

func readProtectionReceipt(t *testing.T, file string, target any) {
	t.Helper()
	if err := json.Unmarshal(mustReadFixture(t, protectionEvidence+file), target); err != nil {
		t.Fatal(err)
	}
}

func verifyProtectionContainer(t *testing.T, fixture protectionFixture, data []byte) {
	t.Helper()
	if fmt.Sprintf("%x", sha256.Sum256(data)) != fixture.SHA256 || len(data) != fixture.Size || fmt.Sprintf("%x", data[:64]) != fixture.PrefixHex {
		t.Fatal("hash-bound authoring evidence no longer matches the fixture")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if !fixture.IsZIP {
		if err == nil {
			t.Fatal("malformed control unexpectedly forms a ZIP")
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != len(fixture.Members) {
		t.Fatal("ZIP member inventory changed")
	}
	for i, member := range archive.File {
		want := fixture.Members[i]
		if member.Name != want.Name || member.Flags != want.Flags || member.Method != want.Compression || member.UncompressedSize64 != want.Size || !want.Readable {
			t.Fatalf("ZIP member %d metadata changed", i)
		}
		reader, openErr := member.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		payload, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("reading %s: %v / %v", member.Name, readErr, closeErr)
		}
		if fmt.Sprintf("%x", sha256.Sum256(payload)) != want.SHA256 {
			t.Fatalf("member %s bytes changed", member.Name)
		}
		if member.Name == "Content/score.gpif" && fmt.Sprintf("%x", payload[:64]) != fixture.ScorePrefix {
			t.Fatal("score member prefix changed")
		}
	}
}

func verifyProtectionControl(t *testing.T, song *guitarpro.Song) {
	t.Helper()
	if song.Tempo != 120 || len(song.Tracks) != 1 || len(song.MeasureHeaders) != 1 {
		t.Fatal("control score tempo or structure changed")
	}
	header := song.MeasureHeaders[0]
	if header.TimeSignature.Numerator != 4 || header.TimeSignature.Denominator.Value != 4 {
		t.Fatal("control meter changed")
	}
	staff := song.Tracks[0].Staves[0]
	if len(staff.Measures) != 1 || len(staff.Measures[0].Voices) != 1 || len(staff.Measures[0].Voices[0].Beats) != 1 {
		t.Fatal("control rhythm changed")
	}
	beat := staff.Measures[0].Voices[0].Beats[0]
	if beat.Duration.Value != 1 || len(beat.Notes) != 6 {
		t.Fatal("control whole-note chord changed")
	}
	for i, note := range beat.Notes {
		if int(note.String) != i+1 || note.Value != 0 {
			t.Fatalf("control string/fret at %d changed", i)
		}
	}
}

func verifyProtectionAlphaTab(t *testing.T) {
	if os.Getenv("ALPHATAB_CONFORMANCE") != "1" {
		t.Skip("set ALPHATAB_CONFORMANCE=1 for the pinned independent consumer")
	}
	var expected struct {
		Results []protectionOutcome `json:"results"`
	}
	readProtectionReceipt(t, "alphatab-results.json", &expected)
	if len(expected.Results) != 4 {
		t.Fatal("independent receipt requires all four fixtures")
	}
	for _, want := range expected.Results {
		output, err := exec.Command("node", "../conformance/oracle.mjs", "--batch-receipts", "../"+want.File).CombinedOutput()
		if err != nil {
			t.Fatalf("AlphaTab receipt: %v\n%s", err, output)
		}
		var rows []struct {
			Score json.RawMessage `json:"score"`
			Error *struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(output, &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 {
			t.Fatal("expected one independent result")
		}
		row := rows[0]
		got := protectionOutcome{File: want.File, ScoreReturned: len(row.Score) > 0 && string(row.Score) != "null"}
		if row.Error != nil {
			got.ErrorType, got.ErrorMessage = &row.Error.Type, &row.Error.Message
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("AlphaTab outcome changed for %s: %s", want.File, output)
		}
	}
}
