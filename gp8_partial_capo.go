// SPDX-License-Identifier: MIT

package goguitarpro

func buildGP8PartialCapo(target *gpifStaff, staff *Staff) {
	partial := staff.PartialCapo
	if partial == nil {
		return
	}
	offset := int(partial.Offset)
	flags := make([]byte, len(partial.Strings))
	for i, selected := range partial.Strings {
		flags[len(flags)-1-i] = '0'
		if selected {
			flags[len(flags)-1-i] = '1'
		}
	}
	bitset := string(flags)
	target.Properties = append(target.Properties, gpifStaffProperty{Name: "PartialCapoFret", Fret: &offset}, gpifStaffProperty{Name: "PartialCapoStringFlags", Bitset: &bitset})
}
