// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func semanticDifferences(goValue, alphaValue any) []semanticDifference {
	goJSON, _ := json.Marshal(goValue)
	alphaJSON, _ := json.Marshal(alphaValue)
	var left, right any
	_ = json.Unmarshal(goJSON, &left)
	_ = json.Unmarshal(alphaJSON, &right)
	differences := make([]semanticDifference, 0)
	collectSemanticDifferences("", left, right, &differences)
	return differences
}

func collectSemanticDifferences(path string, goValue, alphaValue any, differences *[]semanticDifference) {
	if reflect.DeepEqual(goValue, alphaValue) {
		return
	}
	leftMap, leftIsMap := goValue.(map[string]any)
	rightMap, rightIsMap := alphaValue.(map[string]any)
	if leftIsMap && rightIsMap {
		keys := make([]string, 0, len(leftMap)+len(rightMap))
		seen := make(map[string]bool)
		for key := range leftMap {
			keys = append(keys, key)
			seen[key] = true
		}
		for key := range rightMap {
			if !seen[key] {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			collectSemanticDifferences(path+"/"+key, leftMap[key], rightMap[key], differences)
		}
		return
	}
	leftSlice, leftIsSlice := goValue.([]any)
	rightSlice, rightIsSlice := alphaValue.([]any)
	if leftIsSlice && rightIsSlice {
		shared := min(len(leftSlice), len(rightSlice))
		for index := range shared {
			collectSemanticDifferences(path+"/"+strconv.Itoa(index), leftSlice[index], rightSlice[index], differences)
		}
		for index := shared; index < len(leftSlice); index++ {
			*differences = append(*differences, semanticDifference{Path: path + "/" + strconv.Itoa(index), Go: leftSlice[index]})
		}
		for index := shared; index < len(rightSlice); index++ {
			*differences = append(*differences, semanticDifference{Path: path + "/" + strconv.Itoa(index), AlphaTab: rightSlice[index]})
		}
		return
	}
	*differences = append(*differences, semanticDifference{Path: path, Go: goValue, AlphaTab: alphaValue})
}

func conformanceFeature(path string) string {
	for _, feature := range []string{
		"rhythm", "timing", "grace-relationships", "tremolo-picking", "harmonics", "hairpins",
		"tempo-automations", "staff-ownership", "percussion-articulations", "note-and-beat-semantics", "score-core",
	} {
		if strings.HasPrefix(path, "/"+feature) {
			return feature
		}
	}
	classifications := []struct {
		fragments []string
		feature   string
	}{
		{[]string{"/graces"}, "grace-relationships"},
		{[]string{"/tremoloPicking"}, "tremolo-picking"},
		{[]string{"/harmonic"}, "harmonics"},
		{[]string{"/hairpin"}, "hairpins"},
		{[]string{"/tempoAutomations"}, "tempo-automations"},
		{[]string{"/staff-ownership", "/staffCount", "/noteCount", "/clefs", "/tuning"}, "staff-ownership"},
		{[]string{"/timing", "/start", "/durationTicks"}, "timing"},
		{[]string{"/rhythm", "/duration", "/tuplet", "/timeSignature", "/pickup"}, "rhythm"},
		{[]string{"/percussion", "/percussionArticulation", "/percussionInput", "/midi"}, "percussion-articulations"},
		{[]string{"/effects", "/kind", "/dynamic", "/status", "/whammy", "/notes", "/voices", "/bars"}, "note-and-beat-semantics"},
		{[]string{"/metadata", "/program", "/primaryChannel", "/name", "/index", "/repeat", "/alternateEndings", "/tripletFeel", "/text", "/schemaVersion", "/string", "/fret"}, "score-core"},
	}
	for _, classification := range classifications {
		for _, fragment := range classification.fragments {
			if strings.Contains(path, fragment) {
				return classification.feature
			}
		}
	}
	return ""
}

func selectConformanceFeatures(score any, features []string) any {
	data, _ := json.Marshal(score)
	var canonical any
	_ = json.Unmarshal(data, &canonical)
	selected := make(map[string]any, len(features))
	for _, feature := range features {
		switch feature {
		case "rhythm":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{
				"timeSignature": true, "pickup": true, "status": true, "duration": true, "durationTicks": true,
				"dots": true, "tuplet": true,
			}, nil)
		case "timing":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"start": true}, nil)
		case "grace-relationships":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"graces": true}, func(value any) bool {
				items, ok := value.([]any)
				return ok && len(items) > 0
			})
		case "tremolo-picking":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"tremoloPicking": true}, nonNilConformanceFact)
		case "harmonics":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"harmonic": true}, nonNilConformanceFact)
		case "hairpins":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"hairpin": true}, func(value any) bool {
				return value != nil && value != "none"
			})
		case "tempo-automations":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"tempoAutomations": true}, nil)
		case "staff-ownership":
			selected[feature] = conformanceStaffOwnership(canonical)
		case "percussion-articulations":
			selected[feature] = conformancePercussionFacts(canonical)
		case "score-core":
			selected[feature] = conformanceScoreCore(canonical)
		case "note-and-beat-semantics":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{
				"status": true, "dynamic": true, "text": true, "string": true, "fret": true,
				"kind": true, "durationPercent": true, "tieOrigin": true, "tieDestination": true, "effects": true,
				"octave": true, "whammy": true,
			}, nil)
		default:
			panic("unhandled conformance feature " + feature)
		}
	}
	return selected
}

func conformanceScoreCore(score any) any {
	root, _ := score.(map[string]any)
	masterBars, _ := root["masterBars"].([]any)
	barFacts := make([]any, 0, len(masterBars))
	for _, item := range masterBars {
		bar, _ := item.(map[string]any)
		barFacts = append(barFacts, map[string]any{
			"index": bar["index"], "repeatStart": bar["repeatStart"], "repeatCount": bar["repeatCount"],
			"alternateEndings": bar["alternateEndings"], "tripletFeel": bar["tripletFeel"],
		})
	}
	tracks, _ := root["tracks"].([]any)
	trackFacts := make([]any, 0, len(tracks))
	for _, item := range tracks {
		track, _ := item.(map[string]any)
		trackFacts = append(trackFacts, map[string]any{
			"index": track["index"], "name": track["name"], "program": track["program"],
			"primaryChannel": track["primaryChannel"],
		})
	}
	return map[string]any{"metadata": root["metadata"], "masterBars": barFacts, "tracks": trackFacts}
}

func nonNilConformanceFact(value any) bool {
	return value != nil
}

func collectConformanceFacts(value any, keys map[string]bool, include func(any) bool) []any {
	var facts []any
	collectConformanceFactsAt("", value, keys, include, &facts)
	return facts
}

func collectConformanceFactsAt(path string, value any, keys map[string]bool, include func(any) bool, facts *[]any) {
	switch typed := value.(type) {
	case map[string]any:
		mapKeys := make([]string, 0, len(typed))
		for key := range typed {
			mapKeys = append(mapKeys, key)
		}
		sort.Strings(mapKeys)
		for _, key := range mapKeys {
			child := typed[key]
			childPath := path + "/" + key
			if key == "graces" && !keys[key] {
				continue
			}
			if keys[key] && (include == nil || include(child)) {
				*facts = append(*facts, map[string]any{"path": childPath, "value": child})
				continue
			}
			collectConformanceFactsAt(childPath, child, keys, include, facts)
		}
	case []any:
		for index, child := range typed {
			collectConformanceFactsAt(path+"/"+strconv.Itoa(index), child, keys, include, facts)
		}
	}
}

func conformanceStaffOwnership(score any) []any {
	root, _ := score.(map[string]any)
	tracks, _ := root["tracks"].([]any)
	result := make([]any, 0, len(tracks))
	for _, item := range tracks {
		track, _ := item.(map[string]any)
		result = append(result, map[string]any{
			"index": track["index"], "name": track["name"], "staves": track["staves"],
		})
	}
	return result
}

func conformancePercussionFacts(score any) []any {
	root, _ := score.(map[string]any)
	tracks, _ := root["tracks"].([]any)
	var facts []any
	for trackIndex, trackItem := range tracks {
		track, _ := trackItem.(map[string]any)
		if articulations, ok := track["percussionArticulations"].([]any); ok && len(articulations) > 0 {
			facts = append(facts, map[string]any{
				"path": fmt.Sprintf("/tracks/%d/percussionArticulations", trackIndex), "value": articulations,
			})
		}
		staves, _ := track["staves"].([]any)
		for staffIndex, staffItem := range staves {
			staff, _ := staffItem.(map[string]any)
			if staff["percussion"] != true {
				continue
			}
			facts = append(facts, map[string]any{
				"path": fmt.Sprintf("/tracks/%d/staves/%d/percussion", trackIndex, staffIndex), "value": true,
			})
			collectConformanceFactsAt(
				fmt.Sprintf("/tracks/%d/staves/%d", trackIndex, staffIndex),
				staff,
				map[string]bool{"standardNotationLineCount": true, "percussionArticulation": true, "percussionInput": true, "midi": true},
				nil,
				&facts,
			)
		}
	}
	return facts
}

func differenceContains(differences []semanticDifference, suffix string) bool {
	return slices.ContainsFunc(differences, func(difference semanticDifference) bool {
		return strings.HasSuffix(difference.Path, suffix)
	})
}
