// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type semanticContractLedger struct {
	Features []struct {
		ID string `json:"id"`
	} `json:"features"`
	SemanticContracts struct {
		FieldDispositions map[string][]string         `json:"fieldDispositions"`
		ModelTypes        []semanticModelTypeContract `json:"modelTypes"`
		SourceDispatches  []semanticDispatchContract  `json:"sourceDispatches"`
	} `json:"semanticContracts"`
}

type semanticModelTypeContract struct {
	Type          string   `json:"type"`
	Feature       string   `json:"feature"`
	Reason        string   `json:"reason"`
	Fields        []string `json:"fields"`
	Compatibility []string `json:"compatibility"`
	Derived       []string `json:"derived"`
	OutOfScope    []string `json:"outOfScope"`
}

type semanticDispatchContract struct {
	Function           string            `json:"function"`
	Selector           string            `json:"selector"`
	Feature            string            `json:"feature"`
	Cases              map[string]string `json:"cases"`
	Evidence           string            `json:"evidence"`
	DefaultDisposition string            `json:"defaultDisposition"`
	Reason             string            `json:"reason"`
}

type semanticGoInventory struct {
	modelTypes map[string][]string
	dispatches map[string][]string
}

func TestSemanticContractInventory(t *testing.T) {
	ledger := readSemanticContractLedger(t)
	inventory := discoverSemanticGoInventory(t)
	features := make(map[string]struct{}, len(ledger.Features))
	for _, feature := range ledger.Features {
		features[feature.ID] = struct{}{}
	}

	modelContracts := make(map[string]semanticModelTypeContract, len(ledger.SemanticContracts.ModelTypes))
	for _, contract := range ledger.SemanticContracts.ModelTypes {
		if _, duplicate := modelContracts[contract.Type]; duplicate {
			t.Errorf("duplicate semantic model contract %s", contract.Type)
			continue
		}
		modelContracts[contract.Type] = contract
		if _, ok := features[contract.Feature]; !ok {
			t.Errorf("semantic model contract %s has unknown feature %q", contract.Type, contract.Feature)
		}
		if contract.Reason == "" {
			t.Errorf("semantic model contract %s has no reason", contract.Type)
		}
		assertExactSemanticSet(t, "model fields for "+contract.Type, inventory.modelTypes[contract.Type], contract.Fields)
		overrides := append([]string(nil), contract.Compatibility...)
		overrides = append(overrides, contract.Derived...)
		overrides = append(overrides, contract.OutOfScope...)
		if duplicate := firstDuplicate(sortedSemanticValues(overrides)); duplicate != "" {
			t.Errorf("model fields for %s classify %s more than once", contract.Type, duplicate)
		}
		for _, field := range overrides {
			if !slices.Contains(contract.Fields, field) {
				t.Errorf("model contract %s classifies unknown field %s", contract.Type, field)
			}
		}
	}
	for typeName, fields := range inventory.modelTypes {
		if _, ok := modelContracts[typeName]; !ok {
			t.Errorf("unclassified semantic model type %s with fields %s", typeName, strings.Join(fields, ", "))
		}
	}
	for typeName := range modelContracts {
		if _, ok := inventory.modelTypes[typeName]; !ok {
			t.Errorf("semantic model contract %s does not resolve from Song", typeName)
		}
	}
	allowedFieldDispositions := []string{"preserved", "normalized", "omitted", "rejected", "derived", "out-of-scope"}
	var inventoriedFields []string
	for typeName, fields := range inventory.modelTypes {
		for _, field := range fields {
			inventoriedFields = append(inventoriedFields, typeName+"."+field)
		}
	}
	var classifiedFields []string
	for disposition, fields := range ledger.SemanticContracts.FieldDispositions {
		if !slices.Contains(allowedFieldDispositions, disposition) {
			t.Errorf("unknown model field disposition %q", disposition)
		}
		classifiedFields = append(classifiedFields, fields...)
	}
	assertExactSemanticSet(t, "model field dispositions", inventoriedFields, classifiedFields)

	dispatchContracts := make(map[string]semanticDispatchContract, len(ledger.SemanticContracts.SourceDispatches))
	for _, contract := range ledger.SemanticContracts.SourceDispatches {
		key := semanticDispatchKey(contract.Function, contract.Selector)
		if _, duplicate := dispatchContracts[key]; duplicate {
			t.Errorf("duplicate source dispatch contract %s", key)
			continue
		}
		dispatchContracts[key] = contract
		if _, ok := features[contract.Feature]; !ok {
			t.Errorf("source dispatch contract %s has unknown feature %q", key, contract.Feature)
		}
		if contract.DefaultDisposition == "" || contract.Reason == "" {
			t.Errorf("source dispatch contract %s has no default disposition or reason", key)
		}
		classifiedCases := make([]string, 0, len(contract.Cases))
		for value, disposition := range contract.Cases {
			classifiedCases = append(classifiedCases, value)
			if disposition == "" {
				t.Errorf("source case %s %q has no disposition", key, value)
			}
		}
		assertExactSemanticSet(t, "source cases for "+key, inventory.dispatches[key], classifiedCases)
	}
	for key, cases := range inventory.dispatches {
		if _, ok := dispatchContracts[key]; !ok {
			t.Errorf("unclassified source dispatch %s with cases %s", key, strings.Join(cases, ", "))
		}
	}
	for key := range dispatchContracts {
		if _, ok := inventory.dispatches[key]; !ok {
			t.Errorf("source dispatch contract %s does not resolve", key)
		}
	}
}

func readSemanticContractLedger(t *testing.T) semanticContractLedger {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("conformance", "feature-ledger.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ledger semanticContractLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	return ledger
}

func discoverSemanticGoInventory(t *testing.T) semanticGoInventory {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fileSet := token.NewFileSet()
	typeExpressions := make(map[string]ast.Expr)
	parsedFiles := make(map[string]*ast.File)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		var source any
		if overlay := os.Getenv("SEMANTIC_INVENTORY_OVERLAY"); overlay != "" && os.Getenv("SEMANTIC_INVENTORY_FILE") == file {
			contents, readErr := os.ReadFile(overlay)
			if readErr != nil {
				t.Fatal(readErr)
			}
			source = contents
		}
		parsed, parseErr := parser.ParseFile(fileSet, file, source, 0)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		parsedFiles[file] = parsed
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec := specification.(*ast.TypeSpec)
				typeExpressions[typeSpec.Name.Name] = typeSpec.Type
			}
		}
	}

	modelTypes := make(map[string][]string)
	queued := []string{"Song"}
	seen := make(map[string]struct{})
	for len(queued) > 0 {
		typeName := queued[0]
		queued = queued[1:]
		if _, ok := seen[typeName]; ok {
			continue
		}
		seen[typeName] = struct{}{}
		expression, ok := typeExpressions[typeName]
		if !ok {
			continue
		}
		structure, ok := expression.(*ast.StructType)
		if !ok {
			continue
		}
		var fields []string
		for _, field := range structure.Fields.List {
			for _, name := range field.Names {
				if name.IsExported() {
					fields = append(fields, name.Name)
				}
			}
			ast.Inspect(field.Type, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if ok {
					if _, local := typeExpressions[identifier.Name]; local {
						queued = append(queued, identifier.Name)
					}
				}
				return true
			})
		}
		sort.Strings(fields)
		modelTypes[typeName] = fields
	}

	dispatchSets := make(map[string]map[string]struct{})
	for file, parsed := range parsedFiles {
		if file != "gpif.go" {
			continue
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				switch statement := node.(type) {
				case *ast.SwitchStmt:
					selector := semanticSelectorName(statement.Tag)
					if selector == "" {
						return true
					}
					for _, bodyStatement := range statement.Body.List {
						clause := bodyStatement.(*ast.CaseClause)
						for _, expression := range clause.List {
							addSemanticDispatchLiteral(t, dispatchSets, function.Name.Name, selector, expression)
						}
					}
				case *ast.BinaryExpr:
					if statement.Op != token.EQL && statement.Op != token.NEQ {
						return true
					}
					if selector := semanticSelectorName(statement.X); selector != "" {
						addSemanticDispatchLiteral(t, dispatchSets, function.Name.Name, selector, statement.Y)
					}
					if selector := semanticSelectorName(statement.Y); selector != "" {
						addSemanticDispatchLiteral(t, dispatchSets, function.Name.Name, selector, statement.X)
					}
				}
				return true
			})
		}
	}
	dispatches := make(map[string][]string, len(dispatchSets))
	for key, values := range dispatchSets {
		for value := range values {
			dispatches[key] = append(dispatches[key], value)
		}
		sort.Strings(dispatches[key])
	}
	return semanticGoInventory{modelTypes: modelTypes, dispatches: dispatches}
}

func semanticSelectorName(expression ast.Expr) string {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		return semanticSelectorName(pointer.X)
	}
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok || (selector.Sel.Name != "Name" && selector.Sel.Name != "Type" && selector.Sel.Name != "HType") {
		return ""
	}
	prefix := semanticExpressionName(selector.X)
	if prefix == "" {
		return ""
	}
	return prefix + "." + selector.Sel.Name
}

func semanticExpressionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		prefix := semanticExpressionName(value.X)
		if prefix != "" {
			return prefix + "." + value.Sel.Name
		}
	}
	return ""
}

func addSemanticDispatchLiteral(t *testing.T, dispatches map[string]map[string]struct{}, function, selector string, expression ast.Expr) {
	t.Helper()
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		t.Fatal(err)
	}
	key := semanticDispatchKey(function, selector)
	if dispatches[key] == nil {
		dispatches[key] = make(map[string]struct{})
	}
	dispatches[key][value] = struct{}{}
}

func semanticDispatchKey(function, selector string) string {
	return function + ":" + selector
}

func assertExactSemanticSet(t *testing.T, label string, actual, classified []string) {
	t.Helper()
	actual = slices.Clone(actual)
	classified = slices.Clone(classified)
	sort.Strings(actual)
	sort.Strings(classified)
	if duplicate := firstDuplicate(classified); duplicate != "" {
		t.Errorf("%s classifies %s more than once", label, duplicate)
		return
	}
	if !slices.Equal(actual, classified) {
		t.Errorf("%s mismatch\nactual: %s\nclassified: %s", label, fmt.Sprint(actual), fmt.Sprint(classified))
	}
}

func firstDuplicate(values []string) string {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return values[index]
		}
	}
	return ""
}

func sortedSemanticValues(values []string) []string {
	values = slices.Clone(values)
	sort.Strings(values)
	return values
}
