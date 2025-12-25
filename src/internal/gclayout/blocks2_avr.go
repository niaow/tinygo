//go:build gc.precise2 && avr

package gclayout

import "unsafe"

type Layout uintptr

const (
	minSize = 2

	NoPtrs  = Layout((0b00 << 8) | ((2 - minSize) << 1) | 1)
	Pointer = Layout((0b01 << 8) | ((2 - minSize) << 1) | 1)
	String  = Layout((0b0001 << 8) | ((4 - minSize) << 1) | 1)
	Slice   = Layout((0b000001 << 8) | ((6 - minSize) << 1) | 1)
)

func (l Layout) AsPtr() unsafe.Pointer { return unsafe.Pointer(l) }
