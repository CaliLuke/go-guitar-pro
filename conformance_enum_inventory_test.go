// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"sort"
	"testing"
)

func TestDiscoverSemanticEnumMembersSemantically(t *testing.T) {
	t.Parallel()
	const base = `package inventory

type NoteType uint8

const NoteTypeDead NoteType = 3
`
	tests := []struct {
		name    string
		sources map[string]string
		want    []string
	}{
		{
			name: "explicit type",
			sources: map[string]string{
				"types.go": base + `const Probe NoteType = 99
`,
			},
			want: []string{"NoteType.NoteTypeDead", "NoteType.Probe"},
		},
		{
			name: "conversion",
			sources: map[string]string{
				"types.go": base + `const Probe = NoteType(99)
`,
			},
			want: []string{"NoteType.NoteTypeDead", "NoteType.Probe"},
		},
		{
			name: "typed arithmetic",
			sources: map[string]string{
				"types.go": base + `const Probe = NoteTypeDead + 1
`,
			},
			want: []string{"NoteType.NoteTypeDead", "NoteType.Probe"},
		},
		{
			name: "same-valued alias is a separate obligation",
			sources: map[string]string{
				"types.go": base + `const Probe = NoteTypeDead
`,
			},
			want: []string{"NoteType.NoteTypeDead", "NoteType.Probe"},
		},
		{
			name: "typed iota with implicit repetition",
			sources: map[string]string{
				"types.go": base + `const (
	IotaStart NoteType = iota + 10
	IotaRepeated
)
`,
			},
			want: []string{"NoteType.IotaRepeated", "NoteType.IotaStart", "NoteType.NoteTypeDead"},
		},
		{
			name: "mixed block resets primitive type",
			sources: map[string]string{
				"types.go": base + `type PublicNoteType = NoteType

const (
	MixedExplicit NoteType = 20
	MixedPrimitive = 21
	MixedPrimitiveRepeated
	MixedConverted = NoteType(22)
	MixedConvertedRepeated
	MixedAlias PublicNoteType = 23
)
`,
			},
			want: []string{
				"NoteType.MixedAlias", "NoteType.MixedConverted", "NoteType.MixedConvertedRepeated",
				"NoteType.MixedExplicit", "NoteType.NoteTypeDead",
			},
		},
		{
			name: "exported type alias uses canonical owner",
			sources: map[string]string{
				"types.go": base + `type PublicNoteType = NoteType

const AliasExplicit PublicNoteType = 23
const AliasConverted = PublicNoteType(24)
`,
			},
			want: []string{"NoteType.AliasConverted", "NoteType.AliasExplicit", "NoteType.NoteTypeDead"},
		},
		{
			name: "string enum",
			sources: map[string]string{
				"types.go": base + `type TextMode string

const TextExplicit TextMode = "explicit"
const TextConverted = TextMode("converted")
const TextSameValueAlias = TextExplicit
`,
			},
			want: []string{
				"NoteType.NoteTypeDead", "TextMode.TextConverted", "TextMode.TextExplicit", "TextMode.TextSameValueAlias",
			},
		},
		{
			name: "moved package file",
			sources: map[string]string{
				"types.go": base,
				"moved.go": `package inventory

const MovedFileMember = NoteType(40)
`,
			},
			want: []string{"NoteType.MovedFileMember", "NoteType.NoteTypeDead"},
		},
		{
			name: "private and primitive controls",
			sources: map[string]string{
				"types.go": base + `type privateType uint8
type PublicFloat float64

const privateMember = NoteType(30)
const ExportedPrivateType privateType = 31
const ExportedPrimitive = 32
const ExportedString = "control"
const ExportedBool = true
const ExportedFloat PublicFloat = 1
`,
			},
			want: []string{"NoteType.NoteTypeDead"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := discoverSemanticEnumFixture(t, test.sources)
			want := slices.Clone(test.want)
			sort.Strings(want)
			if !slices.Equal(got, want) {
				t.Fatalf("semantic enum members = %v, want %v", got, want)
			}
		})
	}
}

func discoverSemanticEnumFixture(t *testing.T, sources map[string]string) []string {
	t.Helper()
	fileSet := token.NewFileSet()
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	files := make([]*ast.File, 0, len(sources))
	for _, name := range names {
		parsed, err := parser.ParseFile(fileSet, name, sources[name], 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, parsed)
	}

	got, err := discoverSemanticEnumMembers(fileSet, files, "example.test/inventory")
	if err != nil {
		t.Fatal(err)
	}
	return got
}
