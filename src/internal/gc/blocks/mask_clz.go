//go:build !avr

package blocks

// TODO: seperate non-CLZ platforms with inline asm

import "math/bits"

type mask uint

// skipForwards shifts off the least significant bit in the mask.
// It returns the length of the shift.
// The mask must be nonzero.
func (m *mask) skipForwards() int {
	old := *m
	tz := bits.TrailingZeros(uint(old))
	*m = (old >> tz) >> 1
	return tz + 1
}

// skipBackwards shifts off the most significant bit in the mask.
// It returns the length of the shift.
// The mask must be nonzero.
func (m *mask) skipBackwards() int {
	old := *m
	lz := bits.LeadingZeros(uint(old))
	*m = (old << lz) << 1
	return lz + 1
}

// count the set bits in the mask.
func (m mask) count() int {
	return bits.OnesCount(uint(m))
}
