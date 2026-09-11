// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

// HeaderFooterStyle owns an optional template and visibility request. Nil leaves
// use the consumer default. Template text is literal; placeholders are not expanded.
type HeaderFooterStyle struct {
	// Template is the authored text, including an explicitly empty string.
	Template *string
	// Visible requests display independently of the template.
	Visible *bool
}

// HeaderFooterSettings contains independent score header and footer elements.
// Nil elements have no authored template or visibility. Imported alignment records
// remain unchanged privately; this API does not author alignment.
type HeaderFooterSettings struct {
	// Title owns this element's independent template and visibility.
	Title *HeaderFooterStyle
	// Subtitle owns this element's independent template and visibility.
	Subtitle *HeaderFooterStyle
	// Artist owns this element's independent template and visibility.
	Artist *HeaderFooterStyle
	// Album owns this element's independent template and visibility.
	Album *HeaderFooterStyle
	// Words owns this element's independent template and visibility.
	Words *HeaderFooterStyle
	// Music owns this element's independent template and visibility.
	Music *HeaderFooterStyle
	// WordsAndMusic owns this element's independent template and visibility.
	WordsAndMusic *HeaderFooterStyle
	// Tabber owns this element's independent template and visibility.
	Tabber *HeaderFooterStyle
	// Copyright owns this element's independent template and visibility.
	Copyright *HeaderFooterStyle
	// Copyright2 owns this element's independent template and visibility.
	Copyright2 *HeaderFooterStyle
}

var headerFooterNames = [10]string{"Title", "Subtitle", "Artist", "Album", "Words", "Music", "WordsAndMusic", "Tabber", "Copyright", "Copyright2"}

func headerFooterSlots(settings *HeaderFooterSettings) [10]**HeaderFooterStyle {
	return [10]**HeaderFooterStyle{&settings.Title, &settings.Subtitle, &settings.Artist, &settings.Album, &settings.Words, &settings.Music, &settings.WordsAndMusic, &settings.Tabber, &settings.Copyright, &settings.Copyright2}
}
func headerFooterKey(index int, visible bool) string {
	prefix := "Header/"
	if index >= 8 {
		prefix = "Footer/"
	}
	if visible {
		prefix += "draw"
	}
	return prefix + headerFooterNames[index]
}
func headerFooterOwnedKey(key string) (int, bool, bool) {
	for index := range headerFooterNames {
		if key == headerFooterKey(index, false) {
			return index, false, true
		}
		if key == headerFooterKey(index, true) {
			return index, true, true
		}
	}
	return 0, false, false
}
func applyHeaderFooterRecord(style *ScoreStyle, record binaryStyleRecord) error {
	index, visible, owned := headerFooterOwnedKey(record.key)
	if !owned {
		return nil
	}
	kind := byte(3)
	if visible {
		kind = 0
	}
	if record.kind != kind {
		return fmt.Errorf("reading BinaryStylesheet: key %q requires type %d, got %d", record.key, kind, record.kind)
	}
	if style.HeaderFooter == nil {
		style.HeaderFooter = &HeaderFooterSettings{}
	}
	slot := headerFooterSlots(style.HeaderFooter)[index]
	if *slot == nil {
		*slot = &HeaderFooterStyle{}
	}
	if visible {
		v := record.value[0] == 1
		(*slot).Visible = &v
	} else {
		v := string(record.value[2:])
		(*slot).Template = &v
	}
	return nil
}
func pageTemplatePointers(page *PageSetup) [7]*string {
	return [7]*string{&page.Title, &page.Subtitle, &page.Artist, &page.Album, &page.Words, &page.Music, &page.WordAndMusic}
}
func pageHeaderBit(index int) uint16 {
	if index >= 8 {
		return 1 << 7
	}
	if index == 7 {
		return 0
	}
	return 1 << index
}
func legacyHeaderFooter(page PageSetup, copyLines *[2]string) HeaderFooterSettings {
	var out HeaderFooterSettings
	slots := headerFooterSlots(&out)
	for i, value := range pageTemplatePointers(&page) {
		text := *value
		visible := page.HeaderAndFooter&pageHeaderBit(i) != 0
		*slots[i] = &HeaderFooterStyle{Template: &text, Visible: &visible}
	}
	lines := [2]string{}
	if copyLines != nil {
		lines = *copyLines
	} else {
		pieces := strings.SplitN(page.Copyright, "\n", 2)
		copy(lines[:], pieces)
	}
	for offset, text := range lines {
		visible := page.HeaderAndFooter&(1<<7) != 0
		*slots[8+offset] = &HeaderFooterStyle{Template: &text, Visible: &visible}
	}
	return out
}
func (song *Song) importLegacyHeaderFooter(copy1, copy2 string) {
	if song.Style == nil {
		song.Style = &ScoreStyle{}
	}
	settings := legacyHeaderFooter(song.PageSetup, &[2]string{copy1, copy2})
	song.Style.HeaderFooter = &settings
	song.pageCompatibility = song.PageSetup
	song.pageCompatibilitySet = true
}
func (song *Song) projectHeaderFooterCompatibility() {
	if song.Style == nil || song.Style.HeaderFooter == nil {
		return
	}
	slots := headerFooterSlots(song.Style.HeaderFooter)
	templates := pageTemplatePointers(&song.PageSetup)
	for i, slot := range slots {
		if *slot == nil {
			continue
		}
		if i < 7 && (*slot).Template != nil {
			*templates[i] = *(*slot).Template
		}
		if (*slot).Visible != nil && i != 7 && i != 9 {
			if *(*slot).Visible {
				song.PageSetup.HeaderAndFooter |= pageHeaderBit(i)
			} else {
				song.PageSetup.HeaderAndFooter &^= pageHeaderBit(i)
			}
		}
	}
	var lines []string
	for _, slot := range slots[8:] {
		if *slot != nil && (*slot).Template != nil {
			lines = append(lines, *(*slot).Template)
		} else if len(lines) > 0 {
			lines = append(lines, "")
		}
	}
	song.PageSetup.Copyright = strings.Join(lines, "\n")
	song.pageCompatibility = song.PageSetup
	song.pageCompatibilitySet = true
}

// resolvedHeaderFooter returns copies so reconciliation cannot mutate source data.
func (song *Song) resolvedHeaderFooter() HeaderFooterSettings {
	var out HeaderFooterSettings
	if song.Style != nil && song.Style.HeaderFooter != nil {
		for i, slot := range headerFooterSlots(song.Style.HeaderFooter) {
			if *slot != nil {
				value := **slot
				*headerFooterSlots(&out)[i] = &value
			}
		}
	}
	if song.Style == nil || song.Style.HeaderFooter == nil {
		if !song.pageCompatibilitySet && song.PageSetup != (PageSetup{}) {
			return legacyHeaderFooter(song.PageSetup, nil)
		}
	}
	slots := headerFooterSlots(&out)
	if !song.pageCompatibilitySet {
		return out
	}
	legacy := legacyHeaderFooter(song.PageSetup, nil)
	legacySlots := headerFooterSlots(&legacy)
	current, original := pageTemplatePointers(&song.PageSetup), pageTemplatePointers(&song.pageCompatibility)
	for i := range slots {
		if i == 7 {
			continue
		}
		templateChanged := false
		if i < 7 {
			templateChanged = *current[i] != *original[i]
		} else {
			templateChanged = song.PageSetup.Copyright != song.pageCompatibility.Copyright
		}
		visibleChanged := (song.PageSetup.HeaderAndFooter^song.pageCompatibility.HeaderAndFooter)&pageHeaderBit(i) != 0
		if !templateChanged && !visibleChanged {
			continue
		}
		if *slots[i] == nil {
			*slots[i] = &HeaderFooterStyle{}
		}
		if templateChanged {
			(*slots[i]).Template = (*legacySlots[i]).Template
		}
		if visibleChanged {
			(*slots[i]).Visible = (*legacySlots[i]).Visible
		}
	}
	return out
}
func headerFooterRecords(song *Song) []binaryStyleRecord {
	settings := song.resolvedHeaderFooter()
	var records []binaryStyleRecord
	for index, slot := range headerFooterSlots(&settings) {
		if *slot == nil {
			continue
		}
		value := *slot
		if value.Template != nil && len(*value.Template) <= 32767 && utf8.ValidString(*value.Template) {
			data := make([]byte, 2+len(*value.Template))
			binary.BigEndian.PutUint16(data, uint16(len(*value.Template)))
			copy(data[2:], *value.Template)
			records = append(records, binaryStyleRecord{key: headerFooterKey(index, false), kind: 3, value: data})
		}
		if value.Visible != nil {
			b := byte(0)
			if *value.Visible {
				b = 1
			}
			records = append(records, binaryStyleRecord{key: headerFooterKey(index, true), kind: 0, value: []byte{b}})
		}
	}
	return records
}
func validateHeaderFooter(song *Song, diagnostics *[]ScoreDiagnostic) {
	size := 4
	for _, record := range mergeHeaderFooterRecords(song, scoreStyleRecords(song.Style)) {
		size += len(record.key) + len(record.value) + 2
	}
	if size > maxBinaryStylesheetSize {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.style.size", Kind: ScoreDiagnosticValue, Reason: fmt.Sprintf("stylesheet size %d exceeds %d-byte limit", size, maxBinaryStylesheetSize)})
	}

	settings := song.resolvedHeaderFooter()
	for index, slot := range headerFooterSlots(&settings) {
		if *slot == nil || (*slot).Template == nil {
			continue
		}
		value := *(*slot).Template
		if len(value) > 32767 || !utf8.ValidString(value) {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.style.header-template", Kind: ScoreDiagnosticValue, Reason: fmt.Sprintf("%s template requires valid UTF-8 and at most 32767 bytes", headerFooterNames[index])})
		}
	}
}
func (builder *gp8Builder) reportPageSetup() {
	page := builder.song.PageSetup
	if page.PageWidth != 0 || page.PageHeight != 0 || page.MarginLeft != 0 || page.MarginRight != 0 || page.MarginTop != 0 || page.MarginBottom != 0 || page.ScoreSizeProportion != 0 || page.PageNumber != "" || page.HeaderAndFooter & ^uint16(0xff) != 0 {
		builder.addReport("gp8.omit.page-setup", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 does not retain physical page dimensions, margins, score scale, page-number templates or their selection bits")
	}
	if (!builder.song.pageCompatibilitySet || page.Copyright != builder.song.pageCompatibility.Copyright) && strings.Count(page.Copyright, "\n") > 1 {
		builder.addReport("gp8.normalize.page-copyright-lines", "score-core", ExportDispositionNormalized, ScoreLocation{}, "the legacy copyright string has multiple line separators; the first separates footer fields and remaining text stays in the second field")
	}
}

// Existing owned records keep their source position; unrelated records are untouched.
func mergeHeaderFooterRecords(song *Song, source []binaryStyleRecord) []binaryStyleRecord {
	authored := headerFooterRecords(song)
	byKey := make(map[string]binaryStyleRecord, len(authored))
	for _, record := range authored {
		byKey[record.key] = record
	}
	var output []binaryStyleRecord
	for _, record := range source {
		if _, _, owned := headerFooterOwnedKey(record.key); !owned {
			output = append(output, record)
			continue
		}
		if value, exists := byKey[record.key]; exists {
			output = append(output, value)
			delete(byKey, record.key)
		}
	}
	for _, record := range authored {
		if _, exists := byKey[record.key]; exists {
			output = append(output, record)
		}
	}
	return output
}
