// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"fmt"
)

const maxPartConfigurationSize = 1 << 20

// StaffNotationSettings contains independently authored staff notation requests.
// GP8 has one shared configuration per track. Export reports differing later
// staff values instead of changing these settings in the public score.
type StaffNotationSettings struct {
	// Standard requests standard notation.
	Standard bool
	// Tablature requests tablature independently from the other displays.
	Tablature bool
	// Slash requests staff-wide slash notation; it does not set Beat.Slashed.
	Slash bool
	// Numbered requests numbered notation independently from other displays.
	Numbered bool
}

func (track *Track) resolvedStaffNotation(staffIndex int) StaffNotationSettings {
	value := StaffNotationSettings{Standard: track.Settings.Notation, Tablature: track.Settings.Tablature && !track.PercussionTrack}
	if track.PercussionTrack {
		value.Standard = true
	}
	if staffIndex < len(track.Staves) && track.Staves[staffIndex].NotationSettings != nil {
		value = *track.Staves[staffIndex].NotationSettings
	}
	if track.notationCompatibilitySet {
		if track.Settings.Notation != track.notationCompatibility[0] {
			value.Standard = track.Settings.Notation
		}
		if track.Settings.Tablature != track.notationCompatibility[1] {
			value.Tablature = track.Settings.Tablature
		}
	}
	return value
}

func (track *Track) markNotationCompatibility() {
	track.notationCompatibility = [2]bool{track.Settings.Notation, track.Settings.Tablature}
	track.notationCompatibilitySet = true
}

func (settings StaffNotationSettings) flags() byte {
	var flags byte
	if settings.Standard {
		flags |= 1
	}
	if settings.Tablature {
		flags |= 2
	}
	if settings.Slash {
		flags |= 4
	}
	if settings.Numbered {
		flags |= 8
	}
	return flags
}

func applyPartConfiguration(song *Song, data []byte) error {
	flags, err := readPartConfiguration(data)
	if err != nil {
		return fmt.Errorf("reading PartConfiguration: %w", err)
	}
	for index, flag := range flags {
		if index >= len(song.Tracks) {
			break
		}
		if flag == 0 {
			flag = 1
		} // The pinned consumer enables standard notation for zero.
		track := &song.Tracks[index]
		track.Settings.Notation = flag&1 != 0
		track.Settings.Tablature = flag&2 != 0 && !track.PercussionTrack
		for staffIndex := range track.Staves {
			staff := &track.Staves[staffIndex]
			staff.NotationSettings = &StaffNotationSettings{Standard: flag&1 != 0, Tablature: flag&2 != 0 && !staff.PercussionTrack, Slash: flag&4 != 0, Numbered: flag&8 != 0}
		}
		track.markNotationCompatibility()
	}
	return nil
}

func readPartConfiguration(data []byte) ([]byte, error) {
	if len(data) > maxPartConfigurationSize {
		return nil, fmt.Errorf("size %d exceeds %d-byte limit", len(data), maxPartConfigurationSize)
	}
	position := 0
	readCount := func(label string) (int, error) {
		if len(data)-position < 4 {
			return 0, fmt.Errorf("%s is truncated at offset %d", label, position)
		}
		value := int64(int32(binary.BigEndian.Uint32(data[position:])))
		position += 4
		if value < 0 || value > int64(len(data)-position) {
			return 0, fmt.Errorf("%s %d exceeds remaining configuration length %d", label, value, len(data)-position)
		}
		return int(value), nil
	}
	views, err := readCount("score view count")
	if err != nil {
		return nil, err
	}
	var first []byte
	for view := 0; view < views; view++ {
		if position >= len(data) {
			return nil, fmt.Errorf("score view %d is truncated at offset %d", view, position)
		}
		position++ // Multi-rest preferences are separate from notation selection.
		count, readErr := readCount(fmt.Sprintf("score view %d track group count", view))
		if readErr != nil {
			return nil, readErr
		}
		if count > len(data)-position {
			return nil, fmt.Errorf("score view %d flags are truncated", view)
		}
		for group, flags := range data[position : position+count] {
			if flags&0xf0 != 0 {
				return nil, fmt.Errorf("score view %d track group %d has unsupported notation flags %#x", view, group, flags)
			}
		}
		if view == 0 {
			first = append([]byte(nil), data[position:position+count]...)
		}
		position += count
	}
	if len(data)-position != 4 {
		return nil, fmt.Errorf("active view requires 4 bytes at offset %d, found %d", position, len(data)-position)
	}
	return first, nil
}

func (builder *gp8Builder) reportStaffNotation(trackIndex int) {
	track := &builder.song.Tracks[trackIndex]
	first := track.resolvedStaffNotation(0)
	shared := first
	if shared.flags() == 0 {
		shared.Standard = true
		builder.addReport("gp8.normalize.track-view", "score-core", ExportDispositionNormalized, ScoreLocation{Track: trackIndex}, "GP8 requires a track view and defaults to standard notation")
	}
	for staffIndex := 0; staffIndex < max(1, len(track.Staves)); staffIndex++ {
		value := track.resolvedStaffNotation(staffIndex)
		location := ScoreLocation{Track: trackIndex, Staff: staffIndex}
		percussion := track.PercussionTrack
		if staffIndex < len(track.Staves) {
			percussion = track.Staves[staffIndex].PercussionTrack
		}
		if percussion && value.Tablature {
			builder.addReport("gp8.omit.percussion-tablature", "score-core", ExportDispositionOmitted, location, "the pinned consumer suppresses tablature on percussion staves")
		}
		if staffIndex == 0 {
			continue
		}
		for _, field := range []struct {
			code         string
			actual, want bool
		}{
			{"gp8.omit.staff-standard-notation", value.Standard, shared.Standard},
			{"gp8.omit.staff-tablature", value.Tablature, shared.Tablature},
			{"gp8.omit.staff-slash-notation", value.Slash, shared.Slash},
			{"gp8.omit.staff-numbered-notation", value.Numbered, shared.Numbered},
		} {
			if field.actual != field.want {
				builder.addReport(field.code, "score-core", ExportDispositionOmitted, location, "GP8 applies the first staff's notation configuration to every staff in this track")
			}
		}
	}
}
