# go-guitar-pro

`go-guitar-pro` parses Guitar Pro files into Go data structures. It supports the binary and container formats of Guitar Pro 3 through 8.

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

## Format support

| Guitar Pro version | Extension | Container |
| --- | --- | --- |
| 3–5 | `.gp3`, `.gp4`, `.gp5` | Binary |
| 6 | `.gpx` | BCFZ or BCFS |
| 7–8 | `.gp` | ZIP with GPIF XML |

The test suite reads every supported file in the compatibility corpus under `testdata/`.

For GP6, GP7, and GP8 files, `Song.TempoAutomations` contains the complete tempo map.

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
