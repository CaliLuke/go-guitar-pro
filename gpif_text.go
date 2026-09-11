// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/xml"
	"strings"
	"unicode"
)

// CDATA keeps authored text intact in the pinned consumer when possible.
type gpifCDATA string

func (text gpifCDATA) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	if gpifTextNeedsEscaping(string(text)) {
		// The pinned consumer reads only the last CDATA chunk. Its numeric
		// entity parser accepts decimal references but misreads hex references.
		// Character references also prevent XML carriage-return normalization.
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(text)); err != nil {
			return err
		}
		value := strings.NewReplacer("&#x9;", "&#9;", "&#xA;", "&#10;", "&#xD;", "&#13;").Replace(escaped.String())
		return encoder.EncodeElement(struct {
			Text string `xml:",innerxml"`
		}{Text: value}, start)
	}
	return encoder.EncodeElement(struct {
		Text string `xml:",cdata"`
	}{Text: string(text)}, start)
}

func (section gpifSection) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	return encoder.EncodeElement(struct {
		Letter gpifCDATA `xml:"Letter"`
		Text   gpifCDATA `xml:"Text"`
	}{Letter: gpifCDATA(section.Letter), Text: gpifCDATA(section.Text)}, start)
}

func (track gpifTrack) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	type plainTrack gpifTrack
	var shortName *gpifCDATA
	if track.ShortName != nil {
		text := gpifCDATA(*track.ShortName)
		shortName = &text
	}
	return encoder.EncodeElement(struct {
		*plainTrack
		ShortName *gpifCDATA `xml:"ShortName,omitempty"`
	}{plainTrack: (*plainTrack)(&track), ShortName: shortName}, start)
}

func gpifTextConsumerTrims(text string) bool {
	return gpifTextNeedsEscaping(text) && strings.TrimFunc(text, func(r rune) bool {
		return unicode.Is(unicode.Zs, r) || strings.ContainsRune("\t\n\v\f\r\ufeff\u2028\u2029", r)
	}) != text
}

func gpifTextNeedsEscaping(text string) bool {
	return strings.Contains(text, "]]>") || strings.ContainsRune(text, '\r')
}

func (score gpifScore) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	return encoder.EncodeElement(struct {
		SystemDefault *string   `xml:"ScoreSystemsDefaultLayout,omitempty"`
		SystemLayout  *string   `xml:"ScoreSystemsLayout,omitempty"`
		Title         gpifCDATA `xml:"Title"`
		SubTitle      gpifCDATA `xml:"SubTitle"`
		Artist        gpifCDATA `xml:"Artist"`
		Album         gpifCDATA `xml:"Album"`
		Words         gpifCDATA `xml:"Words"`
		Music         gpifCDATA `xml:"Music"`
		WordsAndMusic gpifCDATA `xml:"WordsAndMusic"`
		Copyright     gpifCDATA `xml:"Copyright"`
		Tabber        gpifCDATA `xml:"Tabber"`
		Instructions  gpifCDATA `xml:"Instructions"`
		Notices       gpifCDATA `xml:"Notices"`
	}{
		score.SystemDefault, score.SystemLayout,
		gpifCDATA(score.Title), gpifCDATA(score.SubTitle), gpifCDATA(score.Artist),
		gpifCDATA(score.Album), gpifCDATA(score.Words), gpifCDATA(score.Music),
		gpifCDATA(score.WordsAndMusic), gpifCDATA(score.Copyright), gpifCDATA(score.Tabber),
		gpifCDATA(score.Instructions), gpifCDATA(score.Notices),
	}, start)
}

func (line gpifLyricLine) MarshalXML(encoder *xml.Encoder, start xml.StartElement) error {
	return encoder.EncodeElement(struct {
		Text   gpifCDATA `xml:"Text"`
		Offset int       `xml:"Offset"`
	}{gpifCDATA(line.Text), line.Offset}, start)
}
