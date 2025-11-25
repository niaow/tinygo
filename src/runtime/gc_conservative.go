//go:build gc.conservative

// This implements the block-based heap as a fully conservative GC. No tracking
// of pointers is done, every word in an object is considered live if it looks
// like a pointer.

package runtime

import "unsafe"

const preciseHeap = false

type gcLayout struct {
}

func (gcl *gcLayout) set(ptr unsafe.Pointer) {
}

func (gcl gcLayout) scan(start, end uintptr) {
	// Treat every possible address as a pointer.
	markRoots(start, end)
}

func (gcl gcLayout) scanner() gcObjectScanner {
	return gcObjectScanner{}
}

type gcObjectScanner struct {
}

func (scanner *gcObjectScanner) pointerFree() bool {
	// We don't know whether this object contains pointers, so conservatively
	// return false.
	return false
}

// nextIsPointer returns whether this could be a pointer. Because the GC is
// conservative, we can't do much more than check whether the object lies
// somewhere in the heap.
func (scanner gcObjectScanner) nextIsPointer(ptr, parent, addrOfWord uintptr) bool {
	return isOnHeap(ptr)
}
