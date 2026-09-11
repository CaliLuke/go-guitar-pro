// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

var gp8InflaterFixtureNames = []string{
	"a",
	"ab",
	"b",
	"bb",
	"c",
	"cb",
	"csharp",
	"d",
	"db",
	"e",
	"eb",
	"f",
	"fsharp",
	"g",
	"gb",
}

func TestGP8ExportUsesConsumerCompatibleArchiveMembers(t *testing.T) {
	song := conformanceBackingProgrammaticSong(t)
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{
		"VERSION",
		"meta.json",
		"Content/",
		"Content/BinaryStylesheet",
		"Content/PartConfiguration",
		"Content/LayoutConfiguration",
		"Content/score.gpif",
		song.BackingTrack.EmbeddedFilePath,
	}
	if len(archive.File) != len(wantNames) {
		t.Fatalf("archive members = %d, want %d", len(archive.File), len(wantNames))
	}
	for index, file := range archive.File {
		if file.Name != wantNames[index] {
			t.Fatalf("archive member %d = %q, want %q", index, file.Name, wantNames[index])
		}
		if file.Method != zip.Store {
			t.Errorf("archive member %q method = %d, want Store", file.Name, file.Method)
		}
		if file.Flags != 0x0800 {
			t.Errorf("archive member %q flags = %#x, want UTF-8 without data descriptor", file.Name, file.Flags)
		}
		contents := readZipMember(t, archive, file.Name)
		if file.CRC32 != crc32.ChecksumIEEE(contents) {
			t.Errorf("archive member %q CRC = %#x, want %#x", file.Name, file.CRC32, crc32.ChecksumIEEE(contents))
		}
		if file.CompressedSize64 != file.UncompressedSize64 || file.UncompressedSize64 != uint64(len(contents)) {
			t.Errorf("archive member %q sizes = %d/%d, want stored size %d", file.Name, file.CompressedSize64, file.UncompressedSize64, len(contents))
		}
		if strings.HasSuffix(file.Name, "/") {
			if !file.Mode().IsDir() || file.Mode().Perm() != 0o755 {
				t.Errorf("archive member %q mode = %v, want directory 0755", file.Name, file.Mode())
			}
		} else if file.Mode().Perm() != 0o644 {
			t.Errorf("archive member %q mode = %v, want file 0644", file.Name, file.Mode())
		}
	}
	if got := readZipMember(t, archive, song.BackingTrack.EmbeddedFilePath); !bytes.Equal(got, song.BackingTrack.AudioData) {
		t.Fatalf("backing-track bytes = %v, want %v", got, song.BackingTrack.AudioData)
	}
	assertExternalZipIntegrity(t, data)

	legacy := gp8LegacyDeflateArchive(t, data)
	assertGP8ArchivePayloadsEqual(t, data, legacy)
}

func TestAlphaTabLoadsGP8StoredArchiveMembers(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, name := range gp8InflaterFixtureNames {
		t.Run(name, func(t *testing.T) {
			fixture := fmt.Sprintf("references/alphaTab/packages/alphatab/test-data/guitarpro8/transposition-tonality-%s.gp", name)
			sourceData, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			var sourceFacts []conformanceAlphaTabTranspositionStaff
			readAlphaTabOracleFacts(t, "--transposition", fixture, &sourceFacts)

			song, err := Parse(sourceData)
			if err != nil {
				t.Fatalf("Go source parse: %v", err)
			}
			exported, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if err != nil {
				t.Fatalf("Go export: %v", err)
			}
			if _, err := Parse(exported); err != nil {
				t.Fatalf("Go target parse: %v", err)
			}
			legacy := gp8LegacyDeflateArchive(t, exported)
			if failure := alphaTabTranspositionFailure(t, legacy); !strings.Contains(failure, "Invalid huffman") {
				t.Fatalf("legacy raw-Deflate target failure = %q, want Invalid huffman", failure)
			}

			var outputFacts []conformanceAlphaTabTranspositionStaff
			readAlphaTabOracleFacts(t, "--transposition", writeConformanceFixture(t, exported), &outputFacts)
			if !reflect.DeepEqual(outputFacts, sourceFacts) {
				t.Fatalf("AlphaTab transposition facts changed:\nsource = %#v\noutput = %#v", sourceFacts, outputFacts)
			}
			assertGP8ArchivePayloadsEqual(t, exported, legacy)
		})
	}
}

func gp8LegacyDeflateArchive(t *testing.T, data []byte) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	legacyDeflate := map[string]bool{
		"meta.json":                   true,
		"Content/BinaryStylesheet":    true,
		"Content/PartConfiguration":   true,
		"Content/LayoutConfiguration": true,
		"Content/score.gpif":          true,
	}
	entries := make([]gp8ArchiveEntry, 0, len(reader.File))
	for _, file := range reader.File {
		input, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		entry := gp8ArchiveEntry{name: file.Name, method: zip.Store, data: contents}
		if legacyDeflate[file.Name] {
			entry.method = zip.Deflate
		}
		entries = append(entries, entry)
	}
	result, err := writeGP8Archive(entries)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func alphaTabTranspositionFailure(t *testing.T, data []byte) string {
	t.Helper()
	fixture := writeConformanceFixture(t, data)
	command := exec.Command("node", alphaTabOracleScript(), "--transposition", fixture)
	output, err := command.CombinedOutput()
	if err == nil {
		var facts []conformanceAlphaTabTranspositionStaff
		if decodeErr := json.Unmarshal(output, &facts); decodeErr != nil {
			t.Fatalf("decoding unexpected AlphaTab success: %v\n%s", decodeErr, output)
		}
		return ""
	}
	return string(output)
}

func assertExternalZipIntegrity(t *testing.T, data []byte) {
	t.Helper()
	unzip, err := exec.LookPath("unzip")
	if err != nil {
		t.Log("unzip is unavailable; Go archive/zip integrity checks still passed")
		return
	}
	fixture := writeConformanceFixture(t, data)
	if output, err := exec.Command(unzip, "-t", fixture).CombinedOutput(); err != nil {
		t.Fatalf("unzip -t: %v\n%s", err, output)
	}
}

func assertGP8ArchivePayloadsEqual(t *testing.T, left, right []byte) {
	t.Helper()
	leftReader, err := zip.NewReader(bytes.NewReader(left), int64(len(left)))
	if err != nil {
		t.Fatal(err)
	}
	rightReader, err := zip.NewReader(bytes.NewReader(right), int64(len(right)))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftReader.File) != len(rightReader.File) {
		t.Fatalf("archive member count = %d/%d", len(leftReader.File), len(rightReader.File))
	}
	for index := range leftReader.File {
		leftFile := leftReader.File[index]
		rightFile := rightReader.File[index]
		if leftFile.Name != rightFile.Name || leftFile.CRC32 != rightFile.CRC32 || leftFile.Flags != rightFile.Flags || leftFile.Mode() != rightFile.Mode() {
			t.Fatalf("archive member %d metadata changed: %#v / %#v", index, leftFile.FileHeader, rightFile.FileHeader)
		}
		leftData := readZipMember(t, leftReader, leftFile.Name)
		rightData := readZipMember(t, rightReader, rightFile.Name)
		if !bytes.Equal(leftData, rightData) {
			t.Fatalf("archive member %q payload changed", leftFile.Name)
		}
	}
}
