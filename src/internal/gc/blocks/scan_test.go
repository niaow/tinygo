//go:build !avr

package blocks

import (
	"testing"
	"unsafe"
)

func TestMarkRoots(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Create a mix of pointers and non-pointers.
	a := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	b := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	c := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	roots := [...]uintptr{
		0,
		uintptr(a),
		^uintptr(0),
		uintptr(b),
		0,
	}

	// Mark the roots.
	heap.MarkRoots(&ScanStack{}, unsafe.Slice((*byte)(unsafe.Pointer(&roots)), unsafe.Sizeof(roots)))

	// Verify that the correct objects were marked.
	heap.verifyMarked([]unsafe.Pointer{a, b}, []unsafe.Pointer{c}, t)
}

func TestMarkRootsUnaligned(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Create a mix of pointers and non-pointers.
	a := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	b := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	c := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	roots := [...]uintptr{
		uintptr(a),
		^uintptr(0),
		uintptr(b),
		uintptr(c),
	}

	// Mark the roots.
	rootSlice := unsafe.Slice((*byte)(unsafe.Pointer(&roots)), unsafe.Sizeof(roots))
	rootSlice = rootSlice[1:]
	rootSlice = rootSlice[:len(rootSlice)-1]
	heap.MarkRoots(&ScanStack{}, rootSlice)

	// Verify that the correct objects were marked.
	heap.verifyMarked([]unsafe.Pointer{b}, []unsafe.Pointer{a, c}, t)
}

func TestScanPreciseNoPtr(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Create memory containing pointers.
	ptrs := [...]unsafe.Pointer{
		heap.forceAlloc(&free, BytesPerBlock, 0, t),
		heap.forceAlloc(&free, BytesPerBlock, 0, t),
		heap.forceAlloc(&free, BytesPerBlock, 0, t),
	}

	// Scan it with a no-pointer pattern.
	memBytes := unsafe.Slice((*byte)(unsafe.Pointer(&ptrs)), unsafe.Sizeof(ptrs))
	heap.scanPrecise(&ScanStack{}, memBytes, 1)

	// Verify that no objects were marked.
	heap.verifyMarked(nil, ptrs[:], t)
}

func TestScanPreciseSlice(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Create memory containing pointers.
	a := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	b := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	c := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	d := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	e := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	f := heap.forceAlloc(&free, BytesPerBlock, 0, t)
	ptrs := [...]unsafe.Pointer{
		a, b, c,
		d, e, f,
	}

	// Scan it with a slice-shaped pattern.
	memBytes := unsafe.Slice((*byte)(unsafe.Pointer(&ptrs)), unsafe.Sizeof(ptrs))
	const layout uintptr = (((0b100 * maskBits) | (3 - 1)) << 1) | 1
	heap.scanPrecise(&ScanStack{}, memBytes, layout)

	// Verify that the correct objects were marked.
	heap.verifyMarked([]unsafe.Pointer{a, d}, []unsafe.Pointer{b, c, e, f}, t)
}

// TODO: test complex layout

func (h Heap) forceAlloc(free *FreeList, size uintptr, layout uintptr, t *testing.T) unsafe.Pointer {
	t.Helper()

	ptr := h.TryAlloc(free, size, layout)
	if ptr == nil {
		t.Fatalf("allocation of size %d failed", size)
	}

	return ptr
}

func (h Heap) verifyMarked(marked []unsafe.Pointer, unmarked []unsafe.Pointer, t *testing.T) {
	t.Helper()

	for i, v := range marked {
		if !isMarked(v) {
			t.Errorf("object at %p (idx %d) should have been marked", v, i)
		}
	}
	for i, v := range unmarked {
		if isMarked(v) {
			t.Errorf("object at %p (idx %d) should not have been marked", v, i)
		}
	}
}

func isMarked(base unsafe.Pointer) bool {
	return (*objHeader)(unsafe.Add(base, -int(unsafe.Sizeof(objHeader{})))).next != 0
}
