//go:build gc.precise2 && !avr

package gclayout

import (
	"math/bits"
	"unsafe"
)

type Layout uintptr

const (
	minSize  = 1
	maskBits = bits.UintSize

	NoPtrs  = Layout((((0b0 * maskBits) + (1 - minSize)) << 1) | 1)
	Pointer = Layout((((0b1 * maskBits) + (1 - minSize)) << 1) | 1)
	String  = Layout((((0b10 * maskBits) + (2 - minSize)) << 1) | 1)
	Slice   = Layout((((0b100 * maskBits) + (3 - minSize)) << 1) | 1)
)

func (l Layout) AsPtr() unsafe.Pointer { return unsafe.Pointer(l) }
