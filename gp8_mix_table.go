// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

func (builder *gp8Builder) reportMixTable(change *MixTableChange, location ScoreLocation) {
	if change == nil {
		return
	}
	if *change == (MixTableChange{}) {
		builder.addReport("gp8.omit.beat-mix-table-change", "mix-table", ExportDispositionOmitted, location, "an empty legacy mix-table record has no equivalent GPIF event")
		return
	}
	for _, controller := range []struct {
		name string
		item *MixTableItem
	}{{"chorus", change.Chorus}, {"reverb", change.Reverb}, {"phaser", change.Phaser}, {"tremolo", change.Tremolo}} {
		if controller.item != nil {
			builder.addReport("gp8.omit.mix-table-"+controller.name, "mix-table", ExportDispositionOmitted, location, fmt.Sprintf("legacy %s value=%d duration=%d allTracks=%t has no verified GPIF event mapping", controller.name, controller.item.Value, controller.item.Duration, controller.item.AllTracks))
		}
	}
	for _, controller := range []struct {
		name string
		item *MixTableItem
	}{{"tempo", change.Tempo}, {"instrument", change.Instrument}, {"volume", change.Volume}, {"balance", change.Balance}} {
		if controller.item != nil && controller.item.Duration != 0 {
			builder.addReport("gp8.omit.mix-table-"+controller.name+"-transition", "mix-table", ExportDispositionOmitted, location, fmt.Sprintf("GPIF has no duration field for the legacy %s transition of %d beats", controller.name, controller.item.Duration))
		}
	}
	if change.Tempo == nil && change.TempoName != "" {
		builder.addReport("gp8.omit.mix-table-tempo-label", "mix-table", ExportDispositionOmitted, location, "a legacy tempo label without a tempo value has no target event owner")
	}
	for _, field := range []struct {
		name  string
		value int16
	}{{"instrument", change.Rse.Instrument}, {"unknown", change.Rse.Unknown}, {"sound-bank", change.Rse.SoundBank}, {"effect-number", change.Rse.EffectNumber}} {
		if field.value != 0 {
			builder.addReport("gp8.omit.mix-table-rse-"+field.name, "mix-table", ExportDispositionOmitted, location, fmt.Sprintf("legacy mix-table RSE %s=%d has no target event field", field.name, field.value))
		}
	}
	if change.Rse.EffectCategory != "" {
		builder.addReport("gp8.omit.mix-table-rse-effect-category", "mix-table", ExportDispositionOmitted, location, "legacy mix-table RSE effect category has no target event field")
	}
	if change.Rse.Effect != "" {
		builder.addReport("gp8.omit.mix-table-rse-effect", "mix-table", ExportDispositionOmitted, location, "legacy mix-table RSE effect name has no target event field")
	}
	if change.UseRse {
		builder.addReport("gp8.omit.mix-table-use-rse", "mix-table", ExportDispositionOmitted, location, "the legacy mix-table RSE engine switch has no target event field")
	}
}
