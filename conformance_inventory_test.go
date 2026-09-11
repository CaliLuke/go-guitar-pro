// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
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
	Oracle   string `json:"oracle"`
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
		CapabilityClaims     []semanticCapabilityClaim    `json:"capabilityClaims"`
		FieldCases           map[string][]string          `json:"fieldCases"`
		WireFieldCases       map[string][]string          `json:"wireFieldCases"`
		DispatchCases        map[string][]string          `json:"dispatchCases"`
		EnumCases            map[string][]string          `json:"enumCases"`
		ReportCases          map[string][]string          `json:"reportCases"`
		StructuralWireFields map[string]string            `json:"structuralWireFields"`
	} `json:"semanticMatrix"`
}

type semanticCapabilityClaim struct {
	Capability          string                       `json:"capability"`
	Stage               string                       `json:"stage"`
	Case                string                       `json:"case"`
	Source              semanticClaimSource          `json:"source"`
	Value               string                       `json:"value"`
	AssertionStage      string                       `json:"assertionStage"`
	Obligation          string                       `json:"obligation"`
	Serialization       string                       `json:"serialization,omitempty"`
	ReportAssertion     string                       `json:"reportAssertion,omitempty"`
	ReportNotApplicable string                       `json:"reportNotApplicable,omitempty"`
	IndependentEvidence *semanticIndependentEvidence `json:"independentEvidence,omitempty"`
	IndependentLimit    *semanticIndependentLimit    `json:"independentLimit,omitempty"`
}

type semanticClaimSource struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type semanticIndependentEvidence struct {
	ID string `json:"id"`
}

type semanticIndependentLimit struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Oracle     string `json:"oracle"`
	Source     string `json:"source"`
	Obligation string `json:"obligation"`
	Reason     string `json:"reason"`
}

type semanticCapabilityCatalog struct {
	Capabilities []struct {
		ID     string            `json:"id"`
		Stages map[string]string `json:"stages"`
	} `json:"capabilities"`
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
			t.Errorf("unknown wire field disposition %q", disposition)
		}
		classifiedWireFields = append(classifiedWireFields, fields...)
	}
	assertExactSemanticSet(t, "wire field dispositions", inventory.wireFields, classifiedWireFields)

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
	classifiedEnumMembers := make([]string, 0, len(ledger.SemanticMatrix.EnumCases))
	for member := range ledger.SemanticMatrix.EnumCases {
		classifiedEnumMembers = append(classifiedEnumMembers, member)
	}
	assertExactSemanticSet(t, "public enum members", inventory.enumMembers, classifiedEnumMembers)
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
	missingWireFields := assertSemanticCaseAssignments(t, "semantic matrix wire fields", inventory.wireFields, ledger.SemanticMatrix.WireFieldCases, cases)
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
			t.Errorf("structural wire field %s is not discovered", field)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("structural wire field %s has no reason", field)
		}
		if !slices.Contains(inventory.structuralWireCandidates, field) {
			t.Errorf("structural wire field %s is a scalar semantic leaf", field)
		}
	}
	assertSemanticMatrixEvidence(t, ledger, cases)
	assertSupportedCapabilityClaims(t, ledger, cases)
	if ledger.SemanticMatrix.Complete && (len(missingFields) != 0 || len(missingWireFields) != 0 || len(missingDispatches) != 0 || len(missingEnumMembers) != 0 || len(missingFieldBehavior) != 0 || len(missingWireBehavior) != 0 || len(missingDispatchBehavior) != 0 || len(missingEnumBehavior) != 0) {
		t.Errorf("complete semantic matrix has %d public fields, %d wire fields, %d source dispatches, and %d enum members without cases", len(missingFields), len(missingWireFields), len(missingDispatches), len(missingEnumMembers))
		t.Logf("missing behavioral obligations: %d public fields, %d wire fields, %d source dispatches, %d enum members", len(missingFieldBehavior), len(missingWireBehavior), len(missingDispatchBehavior), len(missingEnumBehavior))
		if len(missingFields) != 0 {
			t.Logf("missing public fields: %s", strings.Join(missingFields, ", "))
		}
		if len(missingWireFields) != 0 {
			t.Logf("missing wire fields: %s", strings.Join(missingWireFields, ", "))
		}
		if len(missingDispatches) != 0 {
			t.Logf("missing source dispatches: %s", strings.Join(missingDispatches, ", "))
		}
		if len(missingEnumMembers) != 0 {
			t.Logf("missing enum members: %s", strings.Join(missingEnumMembers, ", "))
		}
	}
	t.Logf("semantic matrix coverage: %d/%d public fields, %d/%d wire fields, %d/%d source dispatches, and %d/%d enum members assigned", len(modelFields)-len(missingFields), len(modelFields), len(inventory.wireFields)-len(missingWireFields), len(inventory.wireFields), len(dispatches)-len(missingDispatches), len(dispatches), len(inventory.enumMembers)-len(missingEnumMembers), len(inventory.enumMembers))
}

func assertSupportedCapabilityClaims(t *testing.T, ledger semanticContractLedger, cases map[string]semanticMatrixCaseContract) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("conformance", "capabilities", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog semanticCapabilityCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	supported := make(map[string]struct{})
	for _, capability := range catalog.Capabilities {
		for stage, status := range capability.Stages {
			if status == "supported" {
				supported[capability.ID+":"+stage] = struct{}{}
			}
		}
	}
	receipts := make(map[string]conformanceClaimReceipt)
	for caseID, contract := range cases {
		executor := conformanceExecutors[contract.Test]
		if executor == nil {
			continue
		}
		t.Run("claim-receipts/"+caseID, func(t *testing.T) {
			run := newConformanceRun(t)
			executor(run)
			if run.pendingComponent != "" {
				t.Errorf("case %s left claim component %s without an assertion", caseID, run.pendingComponent)
			}
			for key, receipt := range run.claimReceipts {
				if receipt.Site.Case != caseID {
					t.Errorf("case %s emitted claim %s for case %s", caseID, key, receipt.Site.Case)
				}
				if _, duplicate := receipts[key]; duplicate {
					t.Errorf("duplicate assertion-site claim receipt %s", key)
				}
				receipts[key] = *receipt
			}
		})
	}
	actual := make(map[string]int, len(ledger.SemanticMatrix.CapabilityClaims))
	for _, claim := range ledger.SemanticMatrix.CapabilityClaims {
		key := claim.Capability + ":" + claim.Stage
		actual[key]++
		contract, caseExists := cases[claim.Case]
		if !caseExists {
			t.Errorf("capability closure claim %s names unknown case %s", key, claim.Case)
			continue
		}
		receipt, ok := receipts[key]
		if !ok {
			t.Errorf("capability closure claim %s has no assertion-site receipt", key)
			continue
		}
		wantSite := conformanceClaimSite{Capability: claim.Capability, Stage: claim.Stage, Case: claim.Case, Source: claim.Source, Value: claim.Value, AssertionStage: claim.AssertionStage}
		if receipt.Site != wantSite || receipt.Obligation != claim.Obligation || receipt.Serialization != claim.Serialization || receipt.ReportAssertion != claim.ReportAssertion {
			t.Errorf("capability closure claim %s does not match assertion-site receipt %#v", key, receipt)
		}
		if claim.Source.Kind != "matrix-scenario" || !strings.HasPrefix(claim.Source.ID, claim.Case+"/"+claim.Capability+"/") || !slices.Contains(contract.Values, claim.Value) || !slices.Contains(contract.Stages, claim.AssertionStage) {
			t.Errorf("capability closure claim %s has an invalid source, value, or assertion stage", key)
		}
		validObligation := strings.HasPrefix(claim.Obligation, "field:") || strings.HasPrefix(claim.Obligation, "wire:") || strings.HasPrefix(claim.Obligation, "dispatch:")
		if !validObligation {
			t.Errorf("capability closure claim %s has invalid primary obligation %q", key, claim.Obligation)
		}
		if claim.Stage == "export" {
			if !strings.HasPrefix(claim.Serialization, "wire:") || !strings.HasPrefix(claim.ReportAssertion, "report:") || claim.ReportNotApplicable != "" {
				t.Errorf("export capability closure claim %s lacks serialization or report evidence", key)
			}
		} else if claim.Serialization != "" || claim.ReportAssertion != "" || claim.ReportNotApplicable != "not-applicable-non-export-stage" {
			t.Errorf("non-export capability closure claim %s has invalid report disposition", key)
		}
		hasIndependent := claim.IndependentEvidence != nil
		hasLimit := claim.IndependentLimit != nil
		if hasIndependent == hasLimit {
			t.Errorf("capability closure claim %s must record independent evidence or exactly one justified limit", key)
		} else if hasIndependent {
			if _, exists := conformanceIndependentEvidenceExecutors[claim.IndependentEvidence.ID]; !exists {
				t.Errorf("capability closure claim %s cites unknown independent evidence %q", key, claim.IndependentEvidence.ID)
			}
		} else {
			limit := claim.IndependentLimit
			if limit.ID != "limit-"+claim.Capability+"-"+claim.Stage || limit.Kind != "no-claim-specific-consumer" || limit.Oracle != ledger.Oracle || limit.Source != contract.Test || limit.Obligation != claim.Obligation || len(limit.Reason) < 80 || !strings.Contains(limit.Reason, claim.Capability) || !strings.Contains(limit.Reason, claim.Obligation) {
				t.Errorf("capability closure claim %s has invalid independent limit %#v", key, limit)
			}
		}
	}
	for key := range supported {
		if actual[key] != 1 {
			t.Errorf("supported capability stage %s has %d executable closure claims, want 1", key, actual[key])
		}
	}
	for key := range receipts {
		if _, ok := supported[key]; !ok {
			t.Errorf("assertion-site receipt %s does not name a supported catalog stage", key)
		}
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
	files, err := filepath.Glob("*_test.go")
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
			if !ok || !slices.Contains([]string{"Field", "Preserved", "Normalized", "Omitted", "Rejected", "Derived", "OutOfScope", "Wire", "Report", "Dispatch", "Enum"}, selector.Sel.Name) {
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
	for construct, caseIDs := range ledger.SemanticMatrix.ReportCases {
		for _, caseID := range caseIDs {
			want[caseID] = append(want[caseID], "report:"+construct)
		}
	}
	provedFieldDispositions := make(map[string]string)
	for caseID, contract := range cases {
		executor, ok := conformanceExecutors[contract.Test]
		if !ok {
			t.Errorf("semantic matrix case %s names unregistered test %s", caseID, contract.Test)
			continue
		}
		t.Run("evidence/"+caseID, func(t *testing.T) {
			run := newConformanceRun(t)
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
	parsedFileList := make([]*ast.File, 0, len(files))
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
		parsedFileList = append(parsedFileList, parsed)
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
		structure, ok := expression.(*ast.StructType)
		if !ok {
			continue
		}
		for _, field := range structure.Fields.List {
			if field.Tag == nil {
				continue
			}
			binaryWire := strings.Contains(field.Tag.Value, `wire:"`)
			for _, name := range field.Names {
				if binaryWire || (strings.HasPrefix(typeName, "gpif") && name.IsExported()) {
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
			exported := len(field.Names) == 0 // Embedded fields remain traversable.
			for _, name := range field.Names {
				if name.IsExported() {
					exported = true
					fields = append(fields, name.Name)
				}
			}
			if !exported {
				continue
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

	enumMembers, err := discoverSemanticEnumMembers(fileSet, parsedFileList, "github.com/CaliLuke/go-guitar-pro")
	if err != nil {
		t.Fatal(err)
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

func discoverSemanticEnumMembers(fileSet *token.FileSet, files []*ast.File, packagePath string) ([]string, error) {
	info := types.Info{Defs: make(map[*ast.Ident]types.Object)}
	configuration := types.Config{Importer: semanticInventoryImporter{delegate: importer.Default()}, IgnoreFuncBodies: true}
	checkedPackage, err := configuration.Check(packagePath, fileSet, files, &info)
	if err != nil {
		return nil, fmt.Errorf("type-checking semantic enum inventory: %w", err)
	}

	// Every exported constant name is a distinct semantic obligation, including
	// same-valued aliases. Resolve aliases to the local defined type so the
	// inventory has one stable owner regardless of declaration syntax.
	members := make([]string, 0)
	for _, object := range info.Defs {
		constant, ok := object.(*types.Const)
		if !ok || !constant.Exported() {
			continue
		}
		named, ok := types.Unalias(constant.Type()).(*types.Named)
		if !ok || named.Obj().Pkg() != checkedPackage || !named.Obj().Exported() {
			continue
		}
		underlying, ok := named.Underlying().(*types.Basic)
		if !ok || underlying.Info()&(types.IsInteger|types.IsString) == 0 {
			continue
		}
		members = append(members, named.Obj().Name()+"."+constant.Name())
	}
	sort.Strings(members)
	return members, nil
}

type semanticInventoryImporter struct {
	delegate types.Importer
}

func (i semanticInventoryImporter) Import(importPath string) (*types.Package, error) {
	imported, err := i.delegate.Import(importPath)
	if err == nil {
		return imported, nil
	}
	// The package under test has already been compiled by go test. A dependency
	// used only inside ignored function bodies does not need export data for the
	// declaration-only inventory pass.
	name := importPath[strings.LastIndex(importPath, "/")+1:]
	placeholder := types.NewPackage(importPath, name)
	placeholder.MarkComplete()
	return placeholder, nil
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
