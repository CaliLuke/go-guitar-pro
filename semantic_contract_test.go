// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
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
		Complete             bool                         `json:"complete"`
		ObligationDigest     string                       `json:"obligationDigest"`
		Families             []semanticMatrixFamily       `json:"families"`
		Cases                []semanticMatrixCaseContract `json:"cases"`
		FieldCases           map[string][]string          `json:"fieldCases"`
		WireFieldCases       map[string][]string          `json:"wireFieldCases"`
		DispatchCases        map[string][]string          `json:"dispatchCases"`
		EnumCases            map[string][]string          `json:"enumCases"`
		StructuralWireFields map[string]string            `json:"structuralWireFields"`
	} `json:"semanticMatrix"`
}

type semanticMatrixFamily struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type semanticMatrixCaseContract struct {
	ID              string   `json:"id"`
	Family          string   `json:"family"`
	EvidenceRole    string   `json:"evidenceRole"`
	EvidenceSources []string `json:"evidenceSources"`
	Test            string   `json:"test"`
	Formats         []string `json:"formats"`
	Stages          []string `json:"stages"`
	Values          []string `json:"values"`
	Oracle          string   `json:"oracle"`
	Limitations     []string `json:"limitations"`
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
	modelTypes               map[string][]string
	dispatches               map[string][]string
	enumMembers              []string
	wireFields               []string
	structuralWireCandidates []string
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
	assertSemanticMatrixAssertionsIndependent(t)
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
	allowedFormats := []string{"GP3", "GP4", "GP5", "GP6", "GP7", "GP8", "programmatic"}
	allowedStages := []string{"import", "programmatic", "finalization", "validation", "preflight", "export", "policy", "oracle", "read-only"}
	allowedEvidenceSources := []string{"public-api", "independent-wire", "independent-consumer", "diagnostic-policy", "schema-round-trip"}
	if got := semanticMatrixObligationDigest(ledger.SemanticMatrix.Cases); got != ledger.SemanticMatrix.ObligationDigest {
		t.Errorf("semantic matrix obligation digest changed: got %s, want %s; review every stage, format, value shape, evidence source, oracle, and limitation before updating the digest", got, ledger.SemanticMatrix.ObligationDigest)
	}
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
		if contract.EvidenceRole != "behavior" && contract.EvidenceRole != "structural" {
			t.Errorf("semantic matrix case %s has invalid evidence role %q", contract.ID, contract.EvidenceRole)
		}
		if len(contract.EvidenceSources) == 0 {
			t.Errorf("semantic matrix case %s has no typed evidence source", contract.ID)
		}
		if contract.EvidenceRole == "structural" && !slices.Contains(contract.EvidenceSources, "schema-round-trip") {
			t.Errorf("structural semantic matrix case %s has no schema-round-trip source", contract.ID)
		}
		if contract.EvidenceRole == "behavior" && slices.Equal(contract.EvidenceSources, []string{"schema-round-trip"}) {
			t.Errorf("behavior semantic matrix case %s relies only on structural schema evidence", contract.ID)
		}
		for _, format := range contract.Formats {
			if !slices.Contains(allowedFormats, format) {
				t.Errorf("semantic matrix case %s has unknown format %q", contract.ID, format)
			}
		}
		for _, stage := range contract.Stages {
			if !slices.Contains(allowedStages, stage) {
				t.Errorf("semantic matrix case %s has unknown stage %q", contract.ID, stage)
			}
		}
		for _, source := range contract.EvidenceSources {
			if !slices.Contains(allowedEvidenceSources, source) {
				t.Errorf("semantic matrix case %s has unknown evidence source %q", contract.ID, source)
			}
		}
		if slices.Contains(contract.Stages, "policy") && !slices.Contains(contract.EvidenceSources, "diagnostic-policy") {
			t.Errorf("semantic matrix case %s has a policy stage without diagnostic-policy evidence", contract.ID)
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
	missingEnumMembers := assertSemanticCaseAssignments(t, "semantic matrix enum members", inventory.enumMembers, ledger.SemanticMatrix.EnumCases, cases)
	missingFieldBehavior := missingSemanticBehaviorAssignments(modelFields, ledger.SemanticMatrix.FieldCases, cases, nil)
	missingWireBehavior := missingSemanticBehaviorAssignments(inventory.wireFields, ledger.SemanticMatrix.WireFieldCases, cases, ledger.SemanticMatrix.StructuralWireFields)
	missingDispatchBehavior := missingSemanticBehaviorAssignments(dispatches, ledger.SemanticMatrix.DispatchCases, cases, nil)
	missingEnumBehavior := missingSemanticBehaviorAssignments(inventory.enumMembers, ledger.SemanticMatrix.EnumCases, cases, nil)
	for field, reason := range ledger.SemanticMatrix.StructuralWireFields {
		if !slices.Contains(inventory.wireFields, field) {
			t.Errorf("structural GPIF wire field %s is not discovered", field)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("structural GPIF wire field %s has no reason", field)
		}
		if !slices.Contains(inventory.structuralWireCandidates, field) {
			t.Errorf("structural GPIF wire field %s is a scalar semantic leaf", field)
		}
	}
	assertSemanticMatrixEvidence(t, ledger, cases)
	if ledger.SemanticMatrix.Complete && (len(missingFields) != 0 || len(missingWireFields) != 0 || len(missingDispatches) != 0 || len(missingEnumMembers) != 0 || len(missingFieldBehavior) != 0 || len(missingWireBehavior) != 0 || len(missingDispatchBehavior) != 0 || len(missingEnumBehavior) != 0) {
		t.Errorf("complete semantic matrix has %d public fields, %d GPIF wire fields, %d source dispatches, and %d enum members without cases", len(missingFields), len(missingWireFields), len(missingDispatches), len(missingEnumMembers))
		t.Logf("missing behavioral obligations: %d public fields, %d GPIF wire fields, %d source dispatches, %d enum members", len(missingFieldBehavior), len(missingWireBehavior), len(missingDispatchBehavior), len(missingEnumBehavior))
		if len(missingFields) != 0 {
			t.Logf("missing public fields: %s", strings.Join(missingFields, ", "))
		}
		if len(missingWireFields) != 0 {
			t.Logf("missing GPIF wire fields: %s", strings.Join(missingWireFields, ", "))
		}
		if len(missingDispatches) != 0 {
			t.Logf("missing source dispatches: %s", strings.Join(missingDispatches, ", "))
		}
		if len(missingEnumMembers) != 0 {
			t.Logf("missing enum members: %s", strings.Join(missingEnumMembers, ", "))
		}
	}
	if !ledger.SemanticMatrix.Complete {
		t.Logf("semantic matrix progress: %d/%d public fields, %d/%d GPIF wire fields, and %d/%d source dispatches assigned", len(modelFields)-len(missingFields), len(modelFields), len(inventory.wireFields)-len(missingWireFields), len(inventory.wireFields), len(dispatches)-len(missingDispatches), len(dispatches))
	}
}

func missingSemanticBehaviorAssignments(discovered []string, assignments map[string][]string, cases map[string]semanticMatrixCaseContract, structural map[string]string) []string {
	var missing []string
	for _, construct := range discovered {
		if _, structuralOnly := structural[construct]; structuralOnly {
			continue
		}
		behavior := slices.ContainsFunc(assignments[construct], func(caseID string) bool {
			return cases[caseID].EvidenceRole == "behavior"
		})
		if !behavior {
			missing = append(missing, construct)
		}
	}
	sort.Strings(missing)
	return missing
}

func assertSemanticMatrixAssertionsIndependent(t *testing.T) {
	t.Helper()
	files, err := filepath.Glob("semantic_matrix_m*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		fileSet := token.NewFileSet()
		parsed, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 3 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !slices.Contains([]string{"Field", "Preserved", "Normalized", "Omitted", "Rejected", "Derived", "OutOfScope", "Wire", "Dispatch", "Enum"}, selector.Sel.Name) {
				return true
			}
			var gotSource, wantSource bytes.Buffer
			if format.Node(&gotSource, fileSet, call.Args[1]) != nil || format.Node(&wantSource, fileSet, call.Args[2]) != nil {
				return true
			}
			if gotSource.String() == wantSource.String() {
				t.Errorf("%s:%d semantic %s assertion compares an expression with itself", path, fileSet.Position(call.Pos()).Line, selector.Sel.Name)
			}
			return true
		})
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
	wantFieldDispositions := make(map[string]string)
	for disposition, fields := range ledger.SemanticContracts.FieldDispositions {
		for _, field := range fields {
			wantFieldDispositions[field] = disposition
		}
	}
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
	for construct, caseIDs := range ledger.SemanticMatrix.EnumCases {
		for _, caseID := range caseIDs {
			want[caseID] = append(want[caseID], "enum:"+construct)
		}
	}
	provedFieldDispositions := make(map[string]string)
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
			for field, disposition := range run.fieldDispositions {
				if previous, ok := provedFieldDispositions[field]; ok && previous != disposition {
					t.Errorf("field %s has conflicting executable dispositions %s and %s", field, previous, disposition)
				}
				provedFieldDispositions[field] = disposition
			}
		})
	}
	for field := range ledger.SemanticMatrix.FieldCases {
		got, ok := provedFieldDispositions[field]
		if !ok {
			t.Errorf("field %s lacks an executable disposition proof", field)
			continue
		}
		if want := wantFieldDispositions[field]; got != want {
			t.Errorf("field %s executable evidence proves %s, but the field partition claims %s", field, got, want)
		}
	}
}

func readSemanticContractLedger(t *testing.T) semanticContractLedger {
	t.Helper()
	path := filepath.Join("conformance", "feature-ledger.json")
	if overlay := os.Getenv("SEMANTIC_LEDGER_OVERLAY"); overlay != "" {
		path = overlay
	}
	data, err := os.ReadFile(path)
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

	var enumMembers []string
	for _, parsed := range parsedFiles {
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.CONST {
				continue
			}
			lastType := ""
			for _, specification := range general.Specs {
				value := specification.(*ast.ValueSpec)
				if identifier, ok := value.Type.(*ast.Ident); ok {
					lastType = identifier.Name
				} else if len(value.Values) != 0 {
					lastType = ""
				}
				if _, reachable := seen[lastType]; !reachable {
					continue
				}
				if expression, ok := typeExpressions[lastType]; !ok || !isSemanticEnumUnderlyingType(expression) {
					continue
				}
				for _, name := range value.Names {
					if name.IsExported() {
						enumMembers = append(enumMembers, lastType+"."+name.Name)
					}
				}
			}
		}
	}
	sort.Strings(enumMembers)

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
					selector := semanticSwitchSelector(function.Name.Name, statement.Tag)
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
	return semanticGoInventory{
		modelTypes: modelTypes, dispatches: dispatches, enumMembers: enumMembers, wireFields: wireFields,
		structuralWireCandidates: discoverStructuralWireCandidates(),
	}
}

func discoverStructuralWireCandidates() []string {
	seen := make(map[reflect.Type]bool)
	var candidates []string
	var walk func(reflect.Type)
	walk = func(valueType reflect.Type) {
		for valueType.Kind() == reflect.Pointer || valueType.Kind() == reflect.Slice {
			valueType = valueType.Elem()
		}
		if valueType.Kind() != reflect.Struct || seen[valueType] {
			return
		}
		seen[valueType] = true
		for index := range valueType.NumField() {
			field := valueType.Field(index)
			fieldType := field.Type
			for fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct || fieldType.Kind() == reflect.Slice {
				candidates = append(candidates, valueType.Name()+"."+field.Name)
			}
			walk(field.Type)
		}
	}
	walk(reflect.TypeOf(gpifDocument{}))
	sort.Strings(candidates)
	return candidates
}

func isSemanticEnumUnderlyingType(expression ast.Expr) bool {
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return false
	}
	switch identifier.Name {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "string":
		return true
	default:
		return false
	}
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

func semanticSwitchSelector(function string, expression ast.Expr) string {
	if selector := semanticSelectorName(expression); selector != "" {
		return selector
	}
	if !strings.HasPrefix(function, "read") {
		return ""
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok {
		return ""
	}
	return identifier.Name
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
	if !ok {
		return
	}
	var value string
	switch literal.Kind {
	case token.STRING:
		unquoted, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		value = unquoted
	case token.INT:
		integer, err := strconv.ParseInt(literal.Value, 0, 64)
		if err != nil {
			t.Fatal(err)
		}
		value = strconv.FormatInt(integer, 10)
	default:
		return
	}
	key := semanticDispatchKey(function, selector)
	if dispatches[key] == nil {
		dispatches[key] = make(map[string]struct{})
	}
	dispatches[key][value] = struct{}{}
}

func semanticMatrixObligationDigest(cases []semanticMatrixCaseContract) string {
	type obligation semanticMatrixCaseContract
	values := make([]obligation, 0, len(cases))
	for _, contract := range cases {
		values = append(values, obligation(contract))
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
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
