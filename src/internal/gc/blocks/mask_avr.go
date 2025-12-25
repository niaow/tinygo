//go:build avr

package blocks

import "math/bits"

type mask uint8

// TODO: rewrite in assembly

// skipForwards shifts off the least significant bit in the mask.
// It returns the length of the shift.
// The mask must be nonzero.
func (m *mask) skipForwards() int {
	old := *m
	tz := bits.TrailingZeros8(uint8(old))
	*m = (old >> tz) >> 1
	return tz + 1
}

// skipBackwards shifts off the most significant bit in the mask.
// It returns the length of the shift.
// The mask must be nonzero.
func (m *mask) skipBackwards() int {
	old := *m
	lz := bits.LeadingZeros8(uint8(old))
	*m = (old << lz) << 1
	return lz + 1
}

// count the set bits in the mask.
func (m mask) count() int {
	return bits.OnesCount8(uint8(m))
}
