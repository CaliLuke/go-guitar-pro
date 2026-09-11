// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"slices"
	"strings"
)

// The explicit track list wins; score lyrics are a fallback for its selected track.
func gp8ResolvedTrackLyrics(song *Song, trackIndex int) []TrackLyricLine {
	lines := song.Tracks[trackIndex].Lyrics
	if len(lines) != 0 || song.Lyrics.TrackIndex != trackIndex {
		return lines
	}
	return gp8LegacyLyricLines(song.Lyrics)
}

func gp8LegacyLyricLines(lyrics Lyrics) []TrackLyricLine {
	lines := make([]TrackLyricLine, len(lyrics.Lines))
	for i, line := range lyrics.Lines {
		lines[i] = TrackLyricLine{Text: line.Text, Offset: line.StartMeasureIndex}
	}
	return lines
}

func gp8ScoreLyricsLoss(song *Song) string {
	if len(song.Lyrics.Lines) == 0 {
		return ""
	}
	index := song.Lyrics.TrackIndex
	if index < 0 || index >= len(song.Tracks) {
		return "GP8 cannot assign score lyrics without a valid selected track"
	}
	explicit := song.Tracks[index].Lyrics
	if len(explicit) > 0 && !slices.Equal(explicit, gp8LegacyLyricLines(song.Lyrics)) {
		return "explicit track lyrics take precedence over conflicting score lyrics"
	}
	return ""
}

func (builder *gp8Builder) reportLyricConsumerLimits() {
	beatLyrics := slices.ContainsFunc(builder.doc.Beats.Beats, func(beat gpifBeat) bool { return beat.Lyrics != nil })
	for trackIndex, track := range builder.doc.Tracks.Tracks {
		if track.Lyrics == nil || len(track.Lyrics.Lines) == 0 {
			continue
		}
		location := ScoreLocation{Track: trackIndex}
		if beatLyrics {
			builder.addReport("gp8.omit.track-lyrics-consumer-dispatch", "score-core", ExportDispositionOmitted, location, "pinned AlphaTab skips all track lyric dispatch when any beat lyric element is present; GPIF retains the independent track and beat lyrics")
		}
		var trimmed []string
		for index, line := range track.Lyrics.Lines {
			if gpifTextConsumerTrims(line.Text) {
				trimmed = append(trimmed, fmt.Sprint(index))
			}
		}
		if len(trimmed) > 0 {
			builder.addReport("gp8.normalize.track-lyrics-text-consumer", "score-core", ExportDispositionNormalized, location, "pinned AlphaTab trims boundary whitespace from escaped lyric text on lines "+strings.Join(trimmed, ", ")+"; GPIF retains the full text")
		}
	}
}
