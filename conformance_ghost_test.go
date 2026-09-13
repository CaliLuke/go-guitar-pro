// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGPIFGhostNoteSpellings(t *testing.T) {
	runGhostNoteSpellings(newConformanceRun(t))
}

func runGhostNoteSpellings(run *conformanceRun) {
	t := run.t
	source, err := Export(conformanceTechniqueSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Normal", "normal", "NORMAL", "none"} {
		t.Run(value, func(t *testing.T) {
			data := rewriteConformanceGPIF(t, source, func(xml string) string {
				return strings.Replace(xml, "</Note>", "<AntiAccent>"+value+"</AntiAccent></Note>", 1)
			})
			parsed, parseErr := Parse(data)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			want := value != "none"
			run.Field("NoteEffect.GhostNote", parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.GhostNote, want)
			output, exportErr := Export(parsed, ExportFormatGP8)
			if exportErr != nil {
				t.Fatal(exportErr)
			}
			wire := conformanceWireDocument(t, output).Notes.Notes[0].AntiAccent
			expectedWire := ""
			if want {
				expectedWire = "Normal"
			}
			run.Wire("gpifNote.AntiAccent", wire, expectedWire)
		})
	}
}

func TestAlphaTabGhostNoteExport(t *testing.T) {
	requireAlphaTabConformance(t)
	source, err := Export(conformanceTechniqueSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	data := rewriteConformanceGPIF(t, source, func(xml string) string {
		return strings.Replace(xml, "</Note>", "<AntiAccent>normal</AntiAccent></Note>", 1)
	})
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	output, err := Export(parsed, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	script := `import fs from 'node:fs'; import {importer,Settings} from './conformance/node_modules/@coderline/alphatab/dist/alphaTab.core.mjs'; for(const path of process.argv.slice(1)){const score=importer.ScoreLoader.loadScoreFromBytes(new Uint8Array(fs.readFileSync(path)),new Settings());if(!score.tracks[0].staves[0].bars[0].voices[0].beats[0].notes[0].isGhost)throw Error('Lost ghost note');}`
	command := exec.Command("node", "--input-type=module", "-e", script, writeConformanceFixture(t, data), writeConformanceFixture(t, output))
	if result, err := command.CombinedOutput(); err != nil {
		t.Fatalf("ghost consumer: %v\n%s", err, result)
	}
}
