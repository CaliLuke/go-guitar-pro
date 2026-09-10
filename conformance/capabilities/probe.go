//go:build ignore

// SPDX-License-Identifier: MIT

// Development-only public API bridge for probe.mjs. Not part of the Go package.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

type request struct {
	Paths  []string
	Output string
}

type receipt struct {
	Path        string
	Output      string
	ParseError  string
	ExportError string
	Diagnostics []guitarpro.ParseDiagnostic
	Report      guitarpro.ExportReport
}

func run() error {
	var input request
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	for index, path := range input.Paths {
		result := receipt{Path: path}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parsed, err := guitarpro.ParseWithOptions(data, guitarpro.ParseOptions{})
		if parsed != nil {
			result.Diagnostics = parsed.Diagnostics
		}
		if err != nil {
			result.ParseError = err.Error()
		} else {
			exported, report, exportErr := guitarpro.ExportWithReport(parsed.Song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
			result.Report = report
			if exportErr != nil {
				result.ExportError = exportErr.Error()
			} else {
				result.Output = filepath.Join(input.Output, fmt.Sprintf("%d.gp", index))
				if writeErr := os.WriteFile(result.Output, exported, 0o600); writeErr != nil {
					return writeErr
				}
			}
		}
		if err := encoder.Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
