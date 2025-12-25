//go:build !gc.precise && !gc.boehm && !gc.precise2

package gclayout

import "unsafe"

type Layout uintptr

const (
	NoPtrs  = Layout(0)
	Pointer = Layout(0)
	String  = Layout(0)
	Slice   = Layout(0)
)

func (l Layout) AsPtr() unsafe.Pointer { return unsafe.Pointer(l) }
