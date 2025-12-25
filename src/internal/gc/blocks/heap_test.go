package blocks

import (
	"slices"
	"testing"
	"unsafe"
)

const testHeapGroups = 12

type testHeap struct {
	blocks     [testHeapGroups][BlocksPerGroup][BytesPerBlock]byte
	startMasks [testHeapGroups]mask
	skipBacks  [testHeapGroups]*uintptr
	end        uint8
}

func (th *testHeap) toHeap() Heap {
	return HeapFromRange(unsafe.Pointer(th), unsafe.Pointer(&th.end))
}

// TODO: test HeapFromRange rounding?

// TestHeapLayout ensures that the component slice addresses are calculated correctly.
func TestHeapLayout(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()

	// Check that the blocks have the expected position and length.
	if expected, got := th.blocks[:], heap.blocks(); !sliceRefEqual(expected, got) {
		t.Errorf("expected blocks at %p:%d:%d but got %p:%d:%d",
			expected, len(expected), cap(expected),
			got, len(got), cap(got),
		)
	}
	if expected, got := th.startMasks[:], heap.startMasks(); !sliceRefEqual(expected, got) {
		t.Errorf("expected startMasks at %p:%d:%d but got %p:%d:%d",
			expected, len(expected), cap(expected),
			got, len(got), cap(got),
		)
	}
	if expected, got := th.skipBacks[:], heap.skipBacks(); !sliceRefEqual(expected, got) {
		t.Errorf("expected skipBacks at %p:%d:%d but got %p:%d:%d",
			expected, len(expected), cap(expected),
			got, len(got), cap(got),
		)
	}
}

// sliceRefEqual checks if two slices refer to the same memory.
func sliceRefEqual[T any](a, b []T) bool {
	return unsafe.SliceData(a) == unsafe.SliceData(b) &&
		len(a) == len(b) &&
		cap(a) == cap(b)
}

// TestHeapInitEmpty ensures that a zero-length heap can be initialized.
func TestHeapInitEmpty(t *testing.T) {
	t.Parallel()

	(Heap{}).Init(nil)
}

// TestHeapInit tests initialization of a normal heap.
func TestHeapInit(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Pop the free memory.
	ptr := free.pop(testHeapGroups * BlocksPerGroup * BytesPerBlock)
	if ptr != unsafe.Pointer(&th.blocks) {
		t.Errorf("expected pop %p but got %p", unsafe.Pointer(&th.blocks), ptr)
	}
}

// TODO: test Heap.Grow somehow?

// TODO: test tryAlloc

func TestTryAllocZero(t *testing.T) {
	t.Parallel()

	var free FreeList
	if (Heap{}).TryAlloc(&free, 1<<8, 0) != nil {
		t.Error("tryAlloc succeeded on a zero heap")
	}
}

// TestTryAllocFull tests allocation of the full heap size.
// NOTE: This used to trigger an index panic because endMask == h.len.
func TestTryAllocFull(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Allocate the whole heap size.
	gotPtr := heap.TryAlloc(&free, testHeapGroups*BlocksPerGroup*BytesPerBlock, 0)
	expectPtr := unsafe.Pointer(&th.blocks[0][0][unsafe.Sizeof(objHeader{})])
	if gotPtr != expectPtr {
		t.Errorf("expected %p but got %p", expectPtr, gotPtr)
	}

	// The first start mask should be 1 and the rest should be 0.
	expectStartMasks := [testHeapGroups]mask{1}
	if th.startMasks != expectStartMasks {
		t.Errorf("expected startMasks 0b%b but got 0b%b", expectStartMasks, th.startMasks)
	}

	// The first skipBack should be unset, and the rest should be the heap start.
	if th.skipBacks[0] != nil {
		t.Error("first skipBack is set")
	}
	for i, v := range th.skipBacks[1:] {
		if v == nil {
			t.Errorf("skipBack %d should not be set", i)
		}
	}
}

// TestTryAlloc3 allocates 3-block objects.
// This sometmes requires a skipBack and sometimes does not.
func TestTryAlloc3(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Allocate 3-block objects until we run out of memory.
	expectPtr := unsafe.Pointer(&th.blocks[0][0][unsafe.Sizeof(objHeader{})])
	var n uintptr
	for {
		gotPtr := heap.TryAlloc(&free, 3*BytesPerBlock, 0)
		if gotPtr == nil {
			break
		}
		n++
		if gotPtr != expectPtr {
			t.Errorf("expected %p but got %p", expectPtr, gotPtr)
		}
		expectPtr = unsafe.Add(expectPtr, 3*BytesPerBlock)
	}
	expectN := (testHeapGroups * BlocksPerGroup) / 3
	if n != expectN {
		t.Errorf("expected %d allocations but got %d", expectN, n)
	}

	// Every third bit should be set in startMasks, excluding the leftovers.
	var expectStartMasks [testHeapGroups]mask
	for i := uintptr(0); i < expectN*3; i += 3 {
		expectStartMasks[i/BlocksPerGroup] |= 1 << (i % BlocksPerGroup)
	}
	if th.startMasks != expectStartMasks {
		t.Errorf("expected startMasks 0b%b but got 0b%b", expectStartMasks, th.startMasks)
	}

	// skipBack should be set except on every third group.
	var expectSkipBacks [testHeapGroups]*uintptr
	for i := uintptr(1); i < testHeapGroups; i++ {
		mod := i % 3
		if mod == 0 {
			continue
		}

		expectSkipBacks[i] = (*uintptr)(unsafe.Pointer(&th.blocks[i-1][BlocksPerGroup-mod]))
	}
	if th.skipBacks != expectSkipBacks {
		t.Errorf("expected skipBacks %v but got %v", expectSkipBacks, th.skipBacks)
	}
}

// TestFindHeadZero tests findHead on a zero-lenth heap.
func TestFindHeadZero(t *testing.T) {
	t.Parallel()

	cases := []uintptr{
		0,
		2129,
		^uintptr(1),
	}
	for _, addr := range cases {
		got := (Heap{}).findHead(addr)
		if got != nil {
			t.Errorf("found head from addr 0x%x on an empty heap", addr)
		}
	}
}

func TestFindHead(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// OOB pointers should return nil.
	cases := []uintptr{
		0,
		1, ^uintptr(0),
		uintptr(unsafe.Pointer(&th.startMasks)),
		uintptr(unsafe.Pointer(&th.skipBacks)),
	}
	for _, addr := range cases {
		got := heap.findHead(addr)
		if got != nil {
			t.Errorf("found head from OOB addr 0x%x on heap at %p", addr, &th)
		}
	}

	// The heap base pointer is a free range.
	// The free range is considered a head, but is not marked.
	if got := heap.findHead(uintptr(unsafe.Pointer(&th.blocks))); got == nil {
		t.Error("findHead == nil for pointer to heap start (free)")
	} else if *got == 0 {
		t.Error("heap start head (free) is unmarked")
	}

	// An interior pointer to a free range should match the range if it is in the same block group.
	if got := heap.findHead(uintptr(unsafe.Pointer(&th.blocks[0][7]))); got == nil {
		t.Error("findHead == nil for interior pointer to free in group")
	} else if *got == 0 {
		t.Error("head of interior pointer to free in group is unmarked")
	}

	// A pointer to a free range should not find a head from a different group because skipBack is not set.
	if got := heap.findHead(uintptr(unsafe.Pointer(&th.blocks[1][7]))); got != nil {
		t.Error("findHead != nil for interior pointer to free in a different group")
	}

	// TODO: test with alloc
	// TODO: test skipBack set
}

func TestMark(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Allocate a single block.
	ptr := heap.TryAlloc(&free, BytesPerBlock, 0)
	if ptr == nil {
		t.Error("allocation failed")
	}

	// Mark it by the base address.
	var scanStack ScanStack
	heap.Mark(&scanStack, uintptr(ptr), uintptr(unsafe.Pointer(ptr)))
	if expected, got := (*objHeader)(unsafe.Add(ptr, -int(unsafe.Sizeof(objHeader{})))), scanStack.pop(); got != expected {
		t.Errorf("expected scan stack head %p but got %p", expected, got)
	}

	// Mark it again.
	// A repeat mark should not do anything.
	heap.Mark(&scanStack, uintptr(ptr), uintptr(unsafe.Pointer(ptr)))
	if got := scanStack.pop(); got != nil {
		t.Errorf("scan stack should be empty, but got %p", got)
	}
}

// TestSweepZero checks that a zero-length heap can be sweeped.
func TestSweepZero(t *testing.T) {
	t.Parallel()

	var free FreeList
	(Heap{}).sweepAndCheck(t, &free, nil)
}

// TestSweepEmpty checks that an empty heap can be sweeped.
func TestSweepEmpty(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Sweep the heap.
	heap.sweepAndCheck(t, &free, []expectFreeRange{
		{
			size:   testHeapGroups * BlocksPerGroup * BytesPerBlock,
			offset: 0,
		},
	})
}

// TestSweepAll fills a heap with small objects and sweeps them into a single free range.
func TestSweepAll(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Fill it with 3-block allocations.
	for {
		ptr := heap.TryAlloc(&free, 3*BytesPerBlock, 0)
		if ptr == nil {
			break
		}
	}

	// Sweep the heap.
	// This should merge everything back into a single free range.
	heap.sweepAndCheck(t, &free, []expectFreeRange{
		{
			size:   testHeapGroups * BlocksPerGroup * BytesPerBlock,
			offset: 0,
		},
	})

	// Ensure that all skipBacks are cleared.
	for i, v := range th.skipBacks {
		if v != nil {
			t.Errorf("unexpected skipBack at %d: %p", i, v)
		}
	}
}

// TestMarkSweep marks an object and checks that it is not sweeped.
func TestMarkSweep(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Allocate an object at the start of the heap.
	const allocSize = BytesPerBlock
	ptr := heap.TryAlloc(&free, allocSize, 0)
	if ptr == nil {
		t.Error("alloc failed")
	}

	// Mark the object.
	heap.Mark(&ScanStack{}, uintptr(ptr), uintptr(unsafe.Pointer(&ptr)))

	// Sweep the heap.
	// The space occupied by the object should not be reclaimed.
	heap.sweepAndCheck(t, &free, []expectFreeRange{
		{
			size:   testHeapGroups*BlocksPerGroup*BytesPerBlock - allocSize,
			offset: allocSize,
		},
	})

	// Ensure that the object has been unmarked.
	hdr := (*objHeader)(unsafe.Add(ptr, -int(unsafe.Sizeof(objHeader{}))))
	if hdr.next != 0 {
		t.Error("object was not unmarked by sweep")
	}
}

// TODO: test more complex sweep cases

func (h Heap) sweepAndCheck(t *testing.T, free *FreeList, expect []expectFreeRange) {
	t.Helper()

	// Sweep the heap.
	expectBytes := uintptr(0)
	for _, v := range expect {
		expectBytes += v.size
	}
	gotBytes := h.Sweep(free)
	if expectBytes != gotBytes {
		t.Errorf("expected %d free bytes but got %d", expectBytes, gotBytes)
	}

	// Check if the free ranges are what we expect.
	var got []expectFreeRange
	for nextSize := free.head; nextSize != nil; nextSize = nextSize.nextSize {
		for nextWithSize := nextSize; nextWithSize != nil; nextWithSize = nextWithSize.nextWithSize {
			got = append(got, expectFreeRange{
				size:   nextWithSize.size,
				offset: uintptr(unsafe.Pointer(nextWithSize)) - uintptr(unsafe.Pointer(h.Base)),
			})
		}
	}
	if !slices.Equal(got, expect) {
		t.Errorf("expected free ranges %v but got %v", expect, got)
	}
}

type expectFreeRange struct {
	size   uintptr
	offset uintptr
}

func TestObjMem(t *testing.T) {
	t.Parallel()

	// Create a test heap.
	var th testHeap
	heap := th.toHeap()
	var free FreeList
	heap.Init(&free)

	// Allocate a large object at the start of the heap.
	const allocSize = (3*BlocksPerGroup + 5) * BytesPerBlock
	ptr := heap.TryAlloc(&free, allocSize, 0)
	if ptr == nil {
		t.Error("alloc failed")
	}

	// Use objMem to find the backing memory.
	obj := (*objHeader)(unsafe.Add(ptr, -int(unsafe.Sizeof(objHeader{}))))
	expected := unsafe.Slice(
		(*byte)(unsafe.Pointer(&th.blocks[0][0][unsafe.Sizeof(objHeader{})])),
		allocSize-unsafe.Sizeof(objHeader{}),
	)
	got := heap.objMem(obj)
	if !sliceRefEqual(got, expected) {
		t.Errorf("expected obj mem at %p:%d:%d but got %p:%d:%d",
			expected, len(expected), cap(expected),
			got, len(got), cap(got),
		)
	}
}
