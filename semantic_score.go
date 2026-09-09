// SPDX-License-Identifier: MIT

package goguitarpro

// Score is the semantic name for Song. It is an alias, not a second model.
// Existing Song callers and new Score callers therefore use the same object.
type Score = Song

// ParseScore parses a Guitar Pro file into the semantic score model.
// It is an additive name for [Parse] and returns the same underlying type.
func ParseScore(data []byte) (*Score, error) {
	return Parse(data)
}
