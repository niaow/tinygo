//go:build gc.conservative2

package blocks

// layout is used to hold the pointer locations within a heap object.
// No information is required when using a conservative collector.
type layout struct{}

// set the layout of the heap object.
// This does nothing when using a conservative collector.
func (l *layout) set(v uintptr) {
}

// hasPointers checks if the layout may contain pointers.
// This is always true for a conservative collector.
func (l layout) hasPointers() bool {
	return true
}

// scanWithLayout scans memory with the provided layout.
// The conservative collector scans conservatively.
func (h Heap) scanWithLayout(dst *ScanStack, mem []byte, layout layout) {
	h.scanConservative(dst, mem)
}
