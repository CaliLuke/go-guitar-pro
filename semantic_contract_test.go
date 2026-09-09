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
		FieldDispositions     map[string][]string         `json:"fieldDispositions"`
		ModelTypes            []semanticModelTypeContract `json:"modelTypes"`
		SourceDispatches      []semanticDispatchContract  `json:"sourceDispatches"`
		WireFieldDispositions map[string][]string         `json:"wireFieldDispositions"`
	} `json:"semanticContracts"`
	SemanticMatrix struct {
		Complete       bool                         `json:"complete"`
		Families       []semanticMatrixFamily       `json:"families"`
		Cases          []semanticMatrixCaseContract `json:"cases"`
		FieldCases     map[string][]string          `json:"fieldCases"`
		WireFieldCases map[string][]string          `json:"wireFieldCases"`
		DispatchCases  map[string][]string          `json:"dispatchCases"`
	} `json:"semanticMatrix"`
}

type semanticMatrixFamily struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type semanticMatrixCaseContract struct {
	ID          string   `json:"id"`
	Family      string   `json:"family"`
	Test        string   `json:"test"`
	Formats     []string `json:"formats"`
	Stages      []string `json:"stages"`
	Values      []string `json:"values"`
	Oracle      string   `json:"oracle"`
	Limitations []string `json:"limitations"`
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
	wireFields []string
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

	allowedWireDispositions := []string{"preserved", "normalized", "omitted", "rejected", "schema"}
	var classifiedWireFields []string
	for disposition, fields := range ledger.SemanticContracts.WireFieldDispositions {
		if !slices.Contains(allowedWireDispositions, disposition) {
			t.Errorf("unknown GPIF wire field disposition %q", disposition)
		}
		classifiedWireFields = append(classifiedWireFields, fields...)
	}
	assertExactSemanticSet(t, "GPIF wire field dispositions", inventory.wireFields, classifiedWireFields)

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

func TestSemanticMatrixInventory(t *testing.T) {
	ledger := readSemanticContractLedger(t)
	inventory := discoverSemanticGoInventory(t)
	wantFamilies := make([]string, 25)
	for index := range wantFamilies {
		wantFamilies[index] = fmt.Sprintf("M%02d", index+1)
	}
	gotFamilies := make([]string, 0, len(ledger.SemanticMatrix.Families))
	for _, family := range ledger.SemanticMatrix.Families {
		gotFamilies = append(gotFamilies, family.ID)
		if family.Title == "" || family.Reason == "" {
			t.Errorf("semantic matrix family %s has no title or reason", family.ID)
		}
	}
	assertExactSemanticSet(t, "semantic matrix families", wantFamilies, gotFamilies)

	cases := make(map[string]semanticMatrixCaseContract, len(ledger.SemanticMatrix.Cases))
	for _, contract := range ledger.SemanticMatrix.Cases {
		if _, duplicate := cases[contract.ID]; duplicate {
			t.Errorf("duplicate semantic matrix case %s", contract.ID)
		}
		cases[contract.ID] = contract
		if !slices.Contains(gotFamilies, contract.Family) {
			t.Errorf("semantic matrix case %s has unknown family %s", contract.ID, contract.Family)
		}
		if contract.Test == "" || len(contract.Formats) == 0 || len(contract.Stages) == 0 || len(contract.Values) == 0 || contract.Oracle == "" {
			t.Errorf("semantic matrix case %s has incomplete executable evidence", contract.ID)
		}
	}

	var modelFields []string
	for typeName, fields := range inventory.modelTypes {
		for _, field := range fields {
			modelFields = append(modelFields, typeName+"."+field)
		}
	}
	missingFields := assertSemanticCaseAssignments(t, "semantic matrix public fields", modelFields, ledger.SemanticMatrix.FieldCases, cases)
	missingWireFields := assertSemanticCaseAssignments(t, "semantic matrix GPIF wire fields", inventory.wireFields, ledger.SemanticMatrix.WireFieldCases, cases)
	var dispatches []string
	for key := range inventory.dispatches {
		dispatches = append(dispatches, key)
	}
	missingDispatches := assertSemanticCaseAssignments(t, "semantic matrix source dispatches", dispatches, ledger.SemanticMatrix.DispatchCases, cases)
	assertSemanticMatrixEvidence(t, ledger, cases)
	if ledger.SemanticMatrix.Complete && (len(missingFields) != 0 || len(missingWireFields) != 0 || len(missingDispatches) != 0) {
		t.Errorf("complete semantic matrix has %d public fields, %d GPIF wire fields, and %d source dispatches without cases", len(missingFields), len(missingWireFields), len(missingDispatches))
	}
	if !ledger.SemanticMatrix.Complete {
		t.Logf("semantic matrix progress: %d/%d public fields, %d/%d GPIF wire fields, and %d/%d source dispatches assigned", len(modelFields)-len(missingFields), len(modelFields), len(inventory.wireFields)-len(missingWireFields), len(inventory.wireFields), len(dispatches)-len(missingDispatches), len(dispatches))
	}
}

func assertSemanticCaseAssignments(
	t *testing.T,
	label string,
	discovered []string,
	assignments map[string][]string,
	cases map[string]semanticMatrixCaseContract,
) []string {
	t.Helper()
	discoveredSet := make(map[string]struct{}, len(discovered))
	for _, construct := range discovered {
		discoveredSet[construct] = struct{}{}
	}
	for construct, caseIDs := range assignments {
		if _, ok := discoveredSet[construct]; !ok {
			t.Errorf("%s classifies unknown construct %s", label, construct)
		}
		if len(caseIDs) == 0 {
			t.Errorf("%s %s has no executable case", label, construct)
		}
		for _, caseID := range caseIDs {
			if _, ok := cases[caseID]; !ok {
				t.Errorf("%s %s references unknown case %s", label, construct, caseID)
			}
		}
	}
	var missing []string
	for _, construct := range discovered {
		if _, ok := assignments[construct]; !ok {
			missing = append(missing, construct)
		}
	}
	sort.Strings(missing)
	return missing
}

func assertSemanticMatrixEvidence(t *testing.T, ledger semanticContractLedger, cases map[string]semanticMatrixCaseContract) {
	t.Helper()
	want := make(map[string][]string, len(cases))
	for construct, caseIDs := range ledger.SemanticMatrix.FieldCases {
		for _, caseID := range caseIDs {
			want[caseID] = append(want[caseID], "field:"+construct)
		}
	}
	for construct, caseIDs := range ledger.SemanticMatrix.WireFieldCases {
		for _, caseID := range caseIDs {
			want[caseID] = append(want[caseID], "wire:"+construct)
		}
	}
	for construct, caseIDs := range ledger.SemanticMatrix.DispatchCases {
		for _, caseID := range caseIDs {
			want[caseID] = append(want[caseID], "dispatch:"+construct)
		}
	}
	for caseID, contract := range cases {
		executor, ok := semanticMatrixExecutors[contract.Test]
		if !ok {
			t.Errorf("semantic matrix case %s names unregistered test %s", caseID, contract.Test)
			continue
		}
		t.Run("evidence/"+caseID, func(t *testing.T) {
			run := newSemanticMatrixRun(t)
			executor(run)
			assertExactSemanticSet(t, "semantic assertions for "+caseID, run.constructs(), want[caseID])
		})
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
	var wireFields []string
	for typeName, expression := range typeExpressions {
		if !strings.HasPrefix(typeName, "gpif") {
			continue
		}
		structure, ok := expression.(*ast.StructType)
		if !ok {
			continue
		}
		for _, field := range structure.Fields.List {
			if field.Tag == nil {
				continue
			}
			for _, name := range field.Names {
				if name.IsExported() {
					wireFields = append(wireFields, typeName+"."+name.Name)
				}
			}
		}
	}
	sort.Strings(wireFields)
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
	for _, parsed := range parsedFiles {
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
					if selector := semanticBinarySelectorName(statement.X); selector != "" {
						addSemanticDispatchLiteral(t, dispatchSets, function.Name.Name, selector, statement.Y)
					}
					if selector := semanticBinarySelectorName(statement.Y); selector != "" {
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
	return semanticGoInventory{modelTypes: modelTypes, dispatches: dispatches, wireFields: wireFields}
}

func semanticSelectorName(expression ast.Expr) string {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		return semanticSelectorName(pointer.X)
	}
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	prefix := semanticExpressionName(selector.X)
	if prefix == "" {
		return ""
	}
	return prefix + "." + selector.Sel.Name
}

func semanticBinarySelectorName(expression ast.Expr) string {
	selector := semanticSelectorName(expression)
	if selector == "" {
		return ""
	}
	name := selector[strings.LastIndex(selector, ".")+1:]
	if name != "Name" && name != "Type" && name != "HType" {
		return ""
	}
	return selector
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
