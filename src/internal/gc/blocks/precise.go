//go:build !gc.conservative2

package blocks

// layout is used to hold the pointer locations within a heap object.
// The precise collector uses the format accepted by scanPrecise (see scan_*.go).
type layout uintptr

// set the layout of the heap object.
// The precise collector just saves the provided value.
func (l *layout) set(v uintptr) {
	*l = layout(v)
}

// hasPointers checks if the layout may contain pointers.
func (l layout) hasPointers() bool {
	// The layout differs between AVR and non-AVR, but 1 always corresponds to:
	// - inline mask
	// - ptr length
	// - no pointers
	return l != 1
}

func (h Heap) scanWithLayout(dst *ScanStack, mem []byte, layout layout) {
	h.scanPrecise(dst, mem, uintptr(layout))
}
