# go-guitar-pro

`go-guitar-pro` parses Guitar Pro files into Go data structures and exports songs to native Guitar Pro 8 files. It reads Guitar Pro 3 through 8.

## Install

```sh
go get github.com/CaliLuke/go-guitar-pro
```

## Use

Parse data that is already in memory:

```go
package main

import (
	"errors"
	"fmt"
	"os"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func main() {
	data, err := os.ReadFile("song.gp5")
	if err != nil {
		panic(err)
	}

	song, err := guitarpro.Parse(data)
	if err != nil {
		var parseErr *guitarpro.ParseError
		if errors.As(err, &parseErr) {
			fmt.Println("The file is not a supported Guitar Pro file.")
			return
		}
		panic(err)
	}

	fmt.Printf("%s — %s\n", song.Artist, song.Name)
}
```

If the parser must read the file, use `guitarpro.ParseFile(path)`.

## Export

Export a `Song` to native Guitar Pro 8 `.gp` bytes:

```go
data, err := guitarpro.Export(song, guitarpro.ExportFormatGP8)
if err != nil {
	panic(err)
}
if err := os.WriteFile("song.gp", data, 0o644); err != nil {
	panic(err)
}
```

Use `guitarpro.ExportFile(path, song, guitarpro.ExportFormatGP8)` when the
library should write the file. Export does not mutate the source `Song`.

Guitar Pro's native percussion notation is the default. Override noteheads by
MIDI value without changing playback routing:

```go
data, err := guitarpro.ExportWithOptions(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{
	GP8: guitarpro.GP8ExportOptions{
		PercussionNoteheads: map[int16]guitarpro.GP8PercussionNotehead{
			46: guitarpro.GP8PercussionNoteheadFilled,
		},
	},
})
```

`ExportFileWithOptions` provides the corresponding file-based API. Supported
families are filled, X, circle-X, and heavy-X. Playback soundbanks, RSE
selectors, and MIDI values remain canonical regardless of the visual override.

GP8 export preserves score metadata, tempo changes, track metadata and MIDI
channels, measure structure, time and key signatures, sections, repeats,
alternate endings, double bars, pickup status, voices, rests, chord diagrams,
note durations and tuplets, percussion articulations, beat dynamics,
grace-note pitch, duration, velocity and placement, accents, ties, and the
supported note effects represented in GPIF.

Fields not listed above are intentionally not serialized. In particular, GP8
export currently omits backing audio and sync points, volume automations,
lyrics, page and RSE settings, bend/tremolo/mix-table effects, detailed chord
harmony and fingering metadata, trill subdivisions, and grace-note bend
transitions. GPIF stores dynamics at beat level, so differing note velocities
within one chord normalize to the first note's dynamic. Explicit empty beats
normalize to rests.

## Format support

| Guitar Pro version | Extension | Container | Read | Export |
| --- | --- | --- | --- | --- |
| 3–5 | `.gp3`, `.gp4`, `.gp5` | Binary | Yes | No |
| 6 | `.gpx` | BCFZ or BCFS | Yes | No |
| 7 | `.gp` | ZIP with GPIF XML | Yes | No |
| 8 | `.gp` | ZIP with GPIF XML | Yes | Yes |

The test suite reads every supported file in the compatibility corpus under `testdata/`.

For GP6, GP7, and GP8 files, `Song.TempoAutomations` contains the complete tempo map.
For GP7 and GP8 files, `Song.SyncPoints` contains the score-to-audio anchors and
`Song.BackingTrack` contains the referenced asset metadata plus its embedded audio
bytes when the asset is present in the project archive.

## Develop

```sh
go test ./...
go vet ./...
golangci-lint run
```

Before you submit a change, run `go fmt ./...`.

## Provenance

This project is an independent Go implementation of the Guitar Pro formats. Research included public files and the alphaTab implementation because no complete format specification exists.

The production code does not import or include alphaTab code. The compatibility fixtures include third-party test data. See `THIRD_PARTY_NOTICES.md` for details.

## License

The Go source code is available under the MIT License. Third-party fixtures keep their applicable upstream terms.
