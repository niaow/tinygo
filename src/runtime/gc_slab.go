//go:build gc.slab

package runtime

import (
	"math/bits"
	"sync"
	"unsafe"
)

const gcDebug = true
const needsStaticHeap = true

const (
	ptrBytes = unsafe.Sizeof(uintptr(0))

	// NOTE: this does not work on AVR.
	ptrScaleBytes = 2 + (^uintptr(0) >> 63)

	ptrScaleBits = ptrScaleBytes + 3

	ptrBits = 1 << ptrScaleBits

	ptrTopBit = uintptr(1) << (ptrBits - 1)

	softPageScale = 12

	softPageSize = 1 << softPageScale

	sizeClasses = softPageScale - ptrScaleBytes

	trieBranchBits = softPageScale - ptrScaleBytes

	trieRootBits = (ptrBits - softPageScale) % trieBranchBits
)

var metadataStart uintptr
var heapPages uintptr

func initHeap() {
	// Align the heap.
	heapStart = (heapStart + (softPageSize - 1)) &^ (softPageSize - 1)

	// Set the heap size.
	resizeHeapInner(0, 0, heapEnd)
}

// setHeapEnd is called to expand the heap. The heap can only grow, not shrink.
// Also, the heap should grow substantially each time otherwise growing the heap
// will be expensive.
func setHeapEnd(newHeapEnd uintptr) {
	// Set the heap size.
	resizeHeapInner(metadataStart, heapPages, newHeapEnd)

	// Update heapEnd.
	heapEnd = newHeapEnd
}

func resizeHeapInner(oldMetadataStart uintptr, oldPages uintptr, newHeapEnd uintptr) {
	// Calculate the new page count.
	newHeapSize := newHeapEnd - heapStart
	if newHeapEnd <= heapStart {
		runtimePanic("gc: negative or empty heap")
	}
	newHeapPages := newHeapSize / (softPageSize + ptrBytes)
	if gcAsserts && newHeapPages <= oldPages {
		runtimePanic("gc: resizeHeapInner shrunk the heap")
	}
	heapPages = newHeapPages

	// Calculate the new metadata start address.
	metadataStart = heapStart + (newHeapPages << softPageScale)

	// Move and update the metadata.
	memmove(unsafe.Pointer(metadataStart), unsafe.Pointer(oldMetadataStart), oldPages<<ptrScaleBytes)
	clear(metaSlice()[oldPages:])

	if gcDebug {
		println("resizeHeapInner from", uint(oldPages), "to", uint(newHeapPages), "at", heapStart)
	}
}

func metaSlice() []*memHeader {
	return unsafe.Slice((**memHeader)(unsafe.Pointer(metadataStart)), heapPages)
}

// Header pools

var slabHeaderSmallPool gcHeaderPool[slabHeaderSmall]
var slabHeaderLargePool gcHeaderPool[slabHeaderLarge]
var memHeaderHugePool gcHeaderPool[memHeaderHuge]

type gcHeaderPool[T any] struct {
	inner gcHeaderPoolInner
}

func (pool *gcHeaderPool[T]) get() *T {
	return (*T)(pool.inner.get(unsafe.Sizeof([1]T{})))
}

func (pool *gcHeaderPool[T]) put(ptr *T) {
	pool.inner.put(unsafe.Pointer(ptr))
}

type gcHeaderPoolInner struct {
	head uintptr
}

func (pool *gcHeaderPoolInner) get(size uintptr) unsafe.Pointer {
	// Try to remove an existing object from the pool.
	got := unsafe.Pointer(pool.head)
	if got != nil {
		// Move the next object to the head.
		pool.head = *(*uintptr)(got)
		return got
	}

	// Allocate a page and insert the space into the pool.
	got = allocSoftPages(1, &hdrNoGC)
	if got == nil {
		return nil
	}
	var head uintptr
	for offset := size; offset+size <= softPageSize; offset += size {
		ptr := unsafe.Add(got, offset)
		*(*uintptr)(ptr) = head
		head = uintptr(ptr)
	}
	pool.head = head
	return got
}

func (pool *gcHeaderPoolInner) put(ptr unsafe.Pointer) {
	*(*uintptr)(ptr) = pool.head
	pool.head = uintptr(ptr)
}

// Page allocator

var hdrNoGC = memHeader{
	base: sizeClasses + 1,
}

func allocSoftPages(pages uintptr, hdr *memHeader) unsafe.Pointer {
	if pages == 0 {
		runtimePanic("gc: zero-length page allocation")
	}

	// Pop a free range.
	ptr := popFreeRange(pages)
	if ptr == nil {
		return nil
	}

	// Update the metadata.
	pageIdx := (uintptr(ptr) - heapStart) >> softPageScale
	metaDst := metaSlice()[pageIdx:][:pages]
	for i := range metaDst {
		metaDst[i] = hdr
	}

	if gcDebug {
		println("allocSoftPages", uint(pages), "at", uint(pageIdx), "@", uintptr(ptr))
		printFreeRanges()
	}

	return ptr
}

var freeRanges *freeRange

type freeRange struct {
	// len is the length of the range in pages.
	len uintptr

	// nextLen is the next free range with a greater length.
	nextLen *freeRange

	// nextWithLen is the next free range with the same length.
	nextWithLen *freeRange
}

func printFreeRanges() {
	for nextLen := freeRanges; nextLen != nil; nextLen = nextLen.nextLen {
		println("-", uint(nextLen.len))
		endOff := (nextLen.len << softPageScale) - 1
		for nextWithLen := nextLen; nextWithLen != nil; nextWithLen = nextWithLen.nextWithLen {
			addr := uintptr(unsafe.Pointer(nextWithLen))
			println("  -", addr, "-", addr+endOff)
		}
	}
}

func insertFreeRange(len uintptr, addr uintptr) {
	insDst := &freeRanges
	var next *freeRange
	var nextLen, nextWithLen *freeRange
	for {
		next = *insDst
		if next == nil {
			// Insert at the end of the list.
			break
		}

		if len > next.len {
			// This must be inserted later in the list.
			insDst = &next.nextLen
			continue
		}

		if len < next.len {
			// This is the first free range of this length.
			nextLen = next
		} else {
			// Insert the new range at the head of this length list.
			nextLen = next.nextLen
			nextWithLen = next
		}
		break
	}
	newRange := (*freeRange)(unsafe.Pointer(addr))
	*newRange = freeRange{
		len:         len,
		nextLen:     nextLen,
		nextWithLen: nextWithLen,
	}
	*insDst = newRange
}

func popFreeRange(len uintptr) unsafe.Pointer {
	remDst := &freeRanges
	for {
		next := *remDst
		if next == nil {
			// We reached the end of the list before a matching range was found.
			return nil
		}

		nextLen := next.len
		if nextLen < len {
			// Search for a longer range.
			remDst = &next.nextLen
			continue
		}

		// Remove this range from the list.
		replace := next.nextLen
		if nextWithLen := next.nextWithLen; nextWithLen != nil {
			nextWithLen.nextLen = replace
			replace = nextWithLen
		}
		*remDst = replace

		if nextLen > len {
			// Re-insert the leftover pages.
			insertFreeRange(nextLen-len, uintptr(unsafe.Pointer(next))+(len<<softPageScale))
		}

		return unsafe.Pointer(next)
	}
}

const (
	firstMediumClass = softPageScale - ptrScaleBytes - ptrScaleBits
	firstLargeClass  = ptrScaleBits
)

// 32-bit size classes
//
// Small:
// 0:    4b : 1024 objs, 32 masks, layout 32 x 32 x  1
// 1:    8b :  512 objs, 16 masks, layout 32 x 16 x  2
// 2:   16b :  256 objs,  8 masks, layout 32 x  8 x  4
// 3:   32b :  128 objs,  4 masks, layout 32 x  4 x  8
// 4:   64b :   64 objs,  2 masks, layout 32 x  2 x 16
// Large:
// 5:  128b :   32 objs,  1 mask , layout 32 x ptr
// 6:  256b :   16 objs,  1 mask , layout 16 x ptr
// 7:  512b :    8 objs,  1 mask , layout  8 x ptr
// 8: 1024b :    4 objs,  1 mask , layout  4 x ptr
// 9: 2048b :    2 objs,  1 mask , layout  2 x ptr

// 64-bit size classes
//
// Small:
// 0:    8b :  512 objs,  8 masks, layout  8 x 64 x  1
// 1:   16b :  256 objs,  4 masks, layout  8 x 32 x  2
// 2:   32b :  128 objs,  2 masks, layout  8 x 16 x  4
// Medium:
// 3:   64b :   64 objs,  1 mask , layout  8 x  8 x  8
// 4:  128b :   32 objs,  1 mask , layout  8 x  4 x 16
// 5:  256b :   16 objs,  1 mask , layout  8 x  2 x 32
// Large:
// 6:  512b :    8 objs,  1 mask , layout  8 x ptr
// 7: 1024b :    4 objs,  1 mask , layout  4 x ptr
// 8: 2048b :    2 objs,  1 mask , layout  2 x ptr

type memHeader struct {
	// base holds the starting address of the backing memory and the size class.
	// The backing memory is aligned to 4 KiB and the size class is stored in the alignment bits.
	base uintptr

	// next is used to create linked lists of memHeaders.
	next *memHeader
}

func (hdr *memHeader) splitBase() (base unsafe.Pointer, class uintptr) {
	rawBase := hdr.base
	return unsafe.Pointer(rawBase &^ (softPageSize - 1)), rawBase & (softPageSize - 1)
}

func (hdr *memHeader) small() *slabHeaderSmall {
	return (*slabHeaderSmall)(unsafe.Add(unsafe.Pointer(hdr), -unsafe.Offsetof(slabHeaderSmall{}.memHeader)))
}

func (hdr *memHeader) medium() *slabHeaderMedium {
	return (*slabHeaderMedium)(unsafe.Add(unsafe.Pointer(hdr), -unsafe.Offsetof(slabHeaderMedium{}.memHeader)))
}

func (hdr *memHeader) large() *slabHeaderLarge {
	return (*slabHeaderLarge)(unsafe.Add(unsafe.Pointer(hdr), -unsafe.Offsetof(slabHeaderLarge{}.memHeader)))
}

func (hdr *memHeader) huge() *memHeaderHuge {
	return (*memHeaderHuge)(unsafe.Add(unsafe.Pointer(hdr), -unsafe.Offsetof(memHeaderHuge{}.memHeader)))
}

const smallMasks = 1 << (softPageScale - ptrScaleBytes - ptrScaleBits)

type slabHeaderSmall struct {
	memHeader

	// liveNest tracks masks within live that are full.
	liveNest uint32
	// pendingMaskNest tracks masks within pendingMark that are non-empty.
	pendingMarkNest uint32

	// live is a bitmap of objects that have been allocated.
	// Each mask correponds to a bit in liveNest.
	live [smallMasks]uintptr
	// marked is a bitmap of objects that have been marked by the current GC cycle.
	marked [smallMasks]uintptr
	// pendingMark is a bitmap of objects that have been marked but have not been scanned.
	// Each mask corresponds to a bit in pendingMarkNest.
	pendingMark [smallMasks]uintptr

	// layout is a bitmap of pointer locations within the span.
	layout [smallMasks]uintptr
}

type slabHeaderMedium struct {
	memHeader

	// live is a mask of objects that have been allocated.
	live uintptr
	// marked is a mask of objects that have been marked by the current GC cycle.
	marked uintptr
	// pendingMark is a bitmap of objects that have been marked but not scanned.
	pendingMark uintptr

	// layout is a bitmap of pointer locations within the span.
	layout [smallMasks]uintptr
}

type slabHeaderLarge struct {
	memHeader

	// live is a mask of objects that have been allocated.
	live uintptr
	// marked is a mask of objects that have been marked by the current GC cycle.
	marked uintptr
	// pendingMark is a bitmap of objects that have been marked but not scanned.
	pendingMark uintptr

	// layout holds the layout of each object.
	// It must be decoded with parseLayout.
	layout [smallMasks]uintptr
}

type memHeaderHuge struct {
	memHeader

	// length of the referenced range in pages.
	// The top bit is used to track whether this has been marked by the current GC cycle.
	length uintptr

	// layout of the object.
	// It must be decoded with parseLayout.
	layout uintptr
}

// GC code

func GC() {
	gcLock.Lock()
	gcInner()
	gcLock.Unlock()
}

func gcInner() {
	if gcDebug {
		println("start gc")
	}

	// Mark stacks and globals.
	gcMarkReachable()

	// Scan marked heap objects.
	for {
		// Remove the next memHeader to scan.
		next := markList
		if next == nil {
			// All reachable memory has been marked.
			break
		}
		markList = next.next

		// Scan the memHeader.
		next.scan()
	}

	// If we're using threads, resume all other threads before starting the
	// sweep.
	gcResumeWorld()

	if gcDebug {
		println("sweeping")
	}

	// Sweep empty slabs.
	clear(freeLists[:])
	meta := metaSlice()
	for i := 0; i < len(meta); {
		// Find the next memHeader.
		hdr := meta[i]
		if hdr == nil {
			i++
			continue
		}

		// Identify the memory type.
		_, class := hdr.splitBase()
		var keep bool
		if class < sizeClasses {
			// This is a slab.
			if class < firstMediumClass {
				// This is a slab of small objects.
				small := hdr.small()
				masks := smallMasks >> class
				for i := 0; i < masks; i++ {
					if small.marked[i] != 0 {
						keep = true
						break
					}
				}
				if keep {
					// Copy while updating liveNest.
					var newLiveNest uint32
					for i := 0; i < masks; i++ {
						mask := small.marked[i]
						small.live[i] = mask
						if mask == ^uintptr(1) {
							newLiveNest |= uint32(1) << i
						}
					}
					clear(small.marked[:])
				} else {
					// Return the header to the pool.
					slabHeaderSmallPool.put(small)
				}
			} else {
				// This is a slab of medium/large objects.
				// Medium and large behave the same here.
				// Do not seperate the cases.
				large := hdr.large()
				keep = large.marked != 0
				if keep {
					// Move the marked mask to the live mask.
					large.live = large.marked
					large.marked = 0
				} else {
					// Return the header to the pool.
					slabHeaderLargePool.put((*slabHeaderLarge)(large))
				}
			}
			if keep {
				// Insert this into the free list.
				hdr.next = freeLists[class]
				freeLists[class] = hdr
			} else {
				// Mark this page as free.
				meta[i] = nil
			}
			i++
		} else if class == sizeClasses {
			// This is a huge object.
			huge := hdr.huge()
			keep = huge.length&ptrTopBit != 0
			if keep {
				// Clear the mark bit.
				huge.length &^= ptrTopBit

				// Advance past the range.
				for i < len(meta) && meta[i] == hdr {
					i++
				}
			} else {
				// Return the header to the pool.
				memHeaderHugePool.put(huge)

				// Mark the pages as free.
				for i < len(meta) && meta[i] == hdr {
					meta[i] = nil
					i++
				}
			}
		} else {
			// This is non-GC memory.
			i++
		}
	}

	// Re-build the free ranges.
	rebuildFreeRanges()

	if gcDebug {
		println("gc finished")
	}

	// TODO: sort free lists by load?
}

func rebuildFreeRanges() {
	metaSlice := metaSlice()
	freeRanges = nil
	for len(metaSlice) > 0 {
		// Skip in-use pages.
		if metaSlice[len(metaSlice)-1] != nil {
			metaSlice = metaSlice[:len(metaSlice)-1]
			continue
		}

		// Isolate the range of free pages.
		rangeEnd := len(metaSlice)
		for len(metaSlice) > 0 && metaSlice[len(metaSlice)-1] == nil {
			metaSlice = metaSlice[:len(metaSlice)-1]
		}

		// Insert the new free range.
		insertFreeRange(uintptr(rangeEnd-len(metaSlice)), (uintptr(len(metaSlice))<<softPageScale)+heapStart)
	}

	if gcDebug {
		// Print the new free ranges.
		println("rebuilt free ranges:")
		printFreeRanges()
	}
}

// Scanning code

func (hdr *memHeader) scan() {
	base, class := hdr.splitBase()
	switch {
	case class < firstMediumClass:
		hdr.small().scan(base, class)
	case class < firstLargeClass:
		hdr.medium().scan(base, class)
	case class < sizeClasses:
		hdr.large().scan(base, class)
	default:
		hdr.huge().scan(base)
	}
}

func (hdr *slabHeaderSmall) scan(base unsafe.Pointer, class uintptr) {
	// Remove the pendingMarkNest bitmap.
	pendingMarkNest := hdr.pendingMarkNest
	hdr.pendingMarkNest = 0

	for {
		// Find the next inner bitmap to access.
		outerIdx := bits.TrailingZeros32(pendingMarkNest)
		if outerIdx >= smallMasks>>class {
			return
		}
		pendingMarkNest &= pendingMarkNest - 1

		// Remove the inner mask.
		innerPending := hdr.pendingMark[outerIdx]
		hdr.pendingMark[outerIdx] = 0
		for ; innerPending != 0; innerPending &= innerPending - 1 {
			// Scan the next object.
			innerIdx := bits.TrailingZeros(uint(innerPending))
			objIdx := (outerIdx << ptrScaleBits) + innerIdx
			scanBitmapLayout(base, class, uintptr(objIdx), &hdr.layout)
		}
	}
}

func (hdr *slabHeaderMedium) scan(base unsafe.Pointer, class uintptr) {
	// Remove the pending mark bitmap.
	pendingMark := hdr.pendingMark
	hdr.pendingMark = 0

	// Scan pending objects.
	for ; pendingMark != 0; pendingMark &= pendingMark - 1 {
		objIdx := bits.TrailingZeros(uint(pendingMark))
		scanBitmapLayout(base, class, uintptr(objIdx), &hdr.layout)
	}
}

func scanBitmapLayout(base unsafe.Pointer, class uintptr, objIdx uintptr, bitmap *[smallMasks]uintptr) {
	// Find the corresponding layout bitmap.
	ptrOff := objIdx << class
	layout := bitmap[ptrOff>>ptrScaleBits]
	layout >>= ptrOff & (ptrBits - 1)
	layout &= (uintptr(1) << class) - 1

	// Mark pointers within the object.
	objBase := unsafe.Add(base, ptrOff<<ptrScaleBytes)
	maskedScan(objBase, layout)
}

func (hdr *slabHeaderLarge) scan(base unsafe.Pointer, class uintptr) {
	// Remove the pending mark bitmap.
	pendingMark := hdr.pendingMark
	hdr.pendingMark = 0

	// Scan pending objects.
	for {
		// Find the next pendng object.
		objIdx := bits.TrailingZeros(uint(pendingMark))
		if objIdx > smallMasks>>(class-firstLargeClass) {
			return
		}
		pendingMark &= pendingMark - 1

		// Scan the object.
		objBase := unsafe.Add(base, (objIdx<<ptrScaleBytes)<<class)
		scanBig(objBase, 1<<class, hdr.layout[objIdx])
	}
}

func (hdr *memHeaderHuge) scan(base unsafe.Pointer) {
	// NOTE: the mark bit in hdr.length will be shifted off
	scanBig(base, hdr.length<<(softPageScale-ptrScaleBytes), hdr.layout)
}

func scanBig(base unsafe.Pointer, len uintptr, layout uintptr) {
	size, mask := parseLayout(layout)
	if size <= ptrBits {
		scanBigSimple(base, len, size, mask)
	} else {
		scanBigComplex(base, len, size, mask)
	}
}

func scanBigSimple(base unsafe.Pointer, len uintptr, size, mask uintptr) {
	if mask == 0 {
		// There is nothing to do.
		return
	}
	if mask == (uintptr(1)<<size)-1 {
		// Everything here is a pointer.
		markRoots(uintptr(base), uintptr(base)+(len<<ptrBytes))
		return
	}

	// TODO: pre-list bits if no ctz inst?

	// Scan full loops of the mask.
	for ; len > size; len -= size {
		maskedScan(base, mask)
		base = unsafe.Add(base, size<<ptrScaleBytes)
	}

	// Scan the remaining pointers.
	maskedScan(base, mask&((uintptr(1)<<size)-1))
}

func scanBigComplex(base unsafe.Pointer, len uintptr, size, mask uintptr) {
	// Scan full loops of the bitmap.
	for ; len > size; len -= size {
		scanBigBitmap(base, size, mask)
		base = unsafe.Add(base, len)
	}

	// Scan the remaining portion.
	scanBigBitmap(base, len, mask)
}

func scanBigBitmap(base unsafe.Pointer, len uintptr, mask uintptr) {
	// TODO: use full-width mask when possible?
	for ; len > 8; len -= 8 {
		// TODO: use smaller maskedScan if no ctz is available
		maskedScan(base, uintptr(*(*uint8)(unsafe.Pointer(mask))))
		mask += 1
		base = unsafe.Add(base, 8*ptrBytes)
	}
	maskedScan(base, uintptr(*(*uint8)(unsafe.Pointer(mask)))&((uintptr(1)<<len)-1))
}

func maskedScan(base unsafe.Pointer, mask uintptr) {
	// TODO: optimize when there is no ctz inst
	for ; mask != 0; mask &= mask - 1 {
		ptrIdx := bits.TrailingZeros(uint(mask))
		addr := unsafe.Add(base, ptrIdx<<ptrScaleBytes)
		markRoot(uintptr(addr), *(*uintptr)(addr))
	}
}

// Marking code

// markRoots reads all pointers from start to end (exclusive) and if they look
// like a heap pointer and are unmarked, marks them and scans that object as
// well (recursively). The start and end parameters must be valid pointers and
// must be aligned.
// TODO: share across collectors instead of copying
func markRoots(start, end uintptr) {
	if gcDebug {
		//println("mark from", start, "to", end, int(end-start))
	}
	if gcAsserts {
		if start >= end {
			runtimePanic("gc: unexpected range to mark")
		}
		if start%unsafe.Alignof(start) != 0 {
			runtimePanic("gc: unaligned start pointer")
		}
		if end%unsafe.Alignof(end) != 0 {
			runtimePanic("gc: unaligned end pointer")
		}
	}

	// Reduce the end bound to avoid reading too far on platforms where pointer alignment is smaller than pointer size.
	// If the size of the range is 0, then end will be slightly below start after this.
	end -= unsafe.Sizeof(end) - unsafe.Alignof(end)

	for addr := start; addr < end; addr += unsafe.Alignof(addr) {
		root := *(*uintptr)(unsafe.Pointer(addr))
		markRoot(addr, root)
	}
}

// mark a GC root at the address addr.
func markRoot(addr, root uintptr) {
	// Calculate the corresponding page index.
	pageIdx := (root - heapStart) >> softPageScale

	// Find the page metadata.
	meta := metaSlice()

	// Load the corresponding page metadata.
	if pageIdx >= uintptr(len(meta)) {
		// This is not a valid page index.
		return
	}
	if hdr := meta[pageIdx]; hdr != nil {
		// Mark the address within this memory.
		hdr.markAddr(root)
	}
}

func (hdr *memHeader) markAddr(addr uintptr) {
	// Extract the size class.
	base := hdr.base
	sizeClass := base & (softPageSize - 1)
	var needScan bool
	if sizeClass < sizeClasses {
		// Convert the address to an index within this slab.
		addr &= softPageSize - 1
		addr >>= ptrScaleBytes
		addr >>= sizeClass

		// Mark the index in the slab.
		if sizeClass < firstLargeClass {
			if sizeClass < firstMediumClass {
				// This is a small size class.
				needScan = hdr.small().markIndex(addr)
			} else {
				// This is a medium size class.
				needScan = hdr.medium().markIndex(addr)
			}
		} else {
			// This is a large size class.
			needScan = hdr.large().markIndex(addr)
		}
	} else if sizeClass == sizeClasses {
		// This is a huge allocation.
		needScan = hdr.huge().mark()
	}

	if needScan {
		// Add this header to the mark list.
		hdr.next = markList
		markList = hdr
	}
}

func (hdr *slabHeaderSmall) markIndex(idx uintptr) bool {
	// Break idx into a mask index and a bit.
	maskIdx := idx >> ptrScaleBits
	maskBit := uintptr(1) << (idx & (ptrBits - 1))

	// Check if this object is live.
	if hdr.live[maskIdx]&maskBit == 0 {
		// This is not live.
		return false
	}

	// Mark the object.
	if hdr.marked[maskIdx]&maskBit != 0 {
		// This has already been marked.
		return false
	}
	hdr.marked[maskIdx] |= maskBit

	// Update the pending mark mask.
	oldPendingMask := hdr.pendingMark[maskIdx]
	hdr.pendingMark[maskIdx] |= maskBit
	if oldPendingMask != 0 {
		// This mask was already pending.
		return false
	}

	// Update the pending mark nest mask.
	oldPendingNest := hdr.pendingMarkNest
	hdr.pendingMarkNest |= uint32(1) << maskIdx
	if oldPendingNest != 0 {
		// This slab is already pending.
		return false
	}

	// Request a scan of this slab.
	return true
}

func (hdr *slabHeaderMedium) markIndex(idx uintptr) bool {
	// The only difference between medium and large is the meaning of layout.
	// That does not matter here.
	return (*slabHeaderLarge)(hdr).markIndex(idx)
}

func (hdr *slabHeaderLarge) markIndex(idx uintptr) bool {
	// Check if the object is live.
	maskBit := uintptr(1) << idx
	if hdr.marked&maskBit == 0 {
		// It is not live.
		return false
	}

	// Mark the object.
	if hdr.marked&maskBit != 0 {
		// This object has already been marked.
		return false
	}
	hdr.marked |= maskBit

	// Update the pending mark mask.
	oldPendingMark := hdr.pendingMark
	hdr.pendingMark |= maskBit
	if oldPendingMark != 0 {
		// This slab is already pending.
		return false
	}

	// Request a scan of this slab.
	return true
}

func (hdr *memHeaderHuge) mark() bool {
	if hdr.length&ptrTopBit != 0 {
		// This has already been marked.
		return false
	}
	hdr.length |= ptrTopBit
	return true
}

var markList *memHeader

// Alloc code

var gcLock sync.Mutex

func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	if size == 0 {
		return unsafe.Pointer(&zeroSizedAlloc)
	}

	// Calculate the size of the allocation in pointers.
	sizePtr := size >> ptrScaleBytes
	if size&(ptrBytes-1) != 0 {
		sizePtr += 1
	}

	// Determine the corresponding size class.
	sizeClass := uintptr(bits.Len(uint(sizePtr - 1)))

	gcLock.Lock()
	var ranGC bool
	var ptr unsafe.Pointer
	for {
		if sizeClass < sizeClasses {
			ptr = trySlabAlloc(sizeClass, layout)
		} else {
			ptr = tryHugeAlloc((sizePtr+((softPageSize>>ptrScaleBytes)-1))>>(softPageScale-ptrScaleBytes), layout)
		}
		if ptr != nil {
			break
		}

		if ranGC {
			if growHeap() {
				// Try the allocation again.
				continue
			}
			runtimePanic("out of memory")
		}
		gcInner()
		ranGC = true
	}
	gcLock.Unlock()
	if gcDebug {
		println("alloc", uint(size), "class", uint(sizeClass), "@", uintptr(ptr), "-", uintptr(ptr)+size-1)
	}
	return ptr
}

func tryHugeAlloc(pages uintptr, layout unsafe.Pointer) unsafe.Pointer {
	// Allocate a header.
	header := memHeaderHugePool.get()
	if header == nil {
		return nil
	}

	// Allocate a backing page range.
	ptr := allocSoftPages(pages, &header.memHeader)
	if ptr == nil {
		memHeaderHugePool.put(header)
		return nil
	}

	// Clear the memory.
	clear(unsafe.Slice((*[softPageSize]byte)(ptr), pages))

	// Initialize the header.
	*header = memHeaderHuge{
		memHeader: memHeader{
			base: uintptr(ptr) | sizeClasses,
		},
		length: pages,
		layout: uintptr(layout),
	}

	if gcDebug {
		println("new huge alloc", pages, "at", ptr)
	}

	return ptr
}

var freeLists [sizeClasses]*memHeader

func trySlabAlloc(class uintptr, layout unsafe.Pointer) unsafe.Pointer {
	// Try to get a slab from the free list.
	slab := freeLists[class]
	for {
		if slab != nil {
			// Allocate a pointer from this slab.
			if ptr := slab.alloc(class, layout); ptr != nil {
				clear(unsafe.Slice((*uintptr)(ptr), 1<<class))
				return ptr
			}

			// Remove the empty slab from this free list.
			slab = slab.next
		} else {
			// Acquire a new slab.
			slab = tryAcquireSlab(class)
			if slab == nil {
				return nil
			}
		}
		freeLists[class] = slab
	}
}

func tryAcquireSlab(class uintptr) *memHeader {
	// Allocate a header.
	var header *memHeader
	if class < firstMediumClass {
		small := slabHeaderSmallPool.get()
		if small == nil {
			return nil
		}
		*small = slabHeaderSmall{}
		header = &small.memHeader
	} else {
		large := slabHeaderLargePool.get()
		if large == nil {
			return nil
		}
		*large = slabHeaderLarge{}
		header = &large.memHeader
	}

	// Allocate a backing page.
	page := allocSoftPages(1, header)
	if page == nil {
		// The page allocation failed.
		if class < firstMediumClass {
			slabHeaderSmallPool.put(header.small())
		} else {
			slabHeaderLargePool.put(header.large())
		}
		return nil
	}

	// Populate the header.
	header.base = uintptr(page) | class

	return header
}

// zeroSizedAlloc is just a sentinel that gets returned when allocating 0 bytes.
var zeroSizedAlloc uint8

func (hdr *memHeader) alloc(class uintptr, layout unsafe.Pointer) unsafe.Pointer {
	// Parse the base and class.
	base := hdr.base
	if base&(softPageSize-1) != class || class > sizeClasses {
		runtimePanic("unexpected size class")
	}
	basePtr := unsafe.Pointer(base &^ (softPageSize - 1))
	if basePtr == nil {
		runtimePanic("nil base pointer")
	}

	// Cast to the correct type and allocate.
	switch {
	case class < firstMediumClass:
		return hdr.small().alloc(basePtr, class, layout)
	case class < firstLargeClass:
		return hdr.medium().alloc(basePtr, class, layout)
	default:
		return hdr.large().alloc(basePtr, class, layout)
	}
}

func (hdr *slabHeaderSmall) alloc(base unsafe.Pointer, class uintptr, layout unsafe.Pointer) unsafe.Pointer {
	// Find the first mask with an available object.
findAvail:
	liveNest := hdr.liveNest
	maskIdx := uintptr(bits.TrailingZeros32(^liveNest))
	if maskIdx >= smallMasks>>class {
		// There are no available objects.
		return nil
	}

	// Find the first available object within the mask.
	bitIdx := uintptr(bits.TrailingZeros(^uint(hdr.live[maskIdx])))
	if bitIdx >= ptrBits {
		// The mask is now full.
		// Update liveNest and try again.
		hdr.liveNest |= uint32(1) << maskIdx
		goto findAvail
	}
	liveBit := uintptr(1) << bitIdx

	// Mark the object as live.
	hdr.live[maskIdx] |= liveBit

	// Compute the object index.
	objIndex := (maskIdx << ptrScaleBits) | bitIdx

	// Populate the layout bitmap.
	insertBitmapLayout(&hdr.layout, class, objIndex, layout)

	// Calculate the result address.
	return unsafe.Add(base, (objIndex<<class)<<ptrScaleBytes)
}

func (hdr *slabHeaderMedium) alloc(base unsafe.Pointer, class uintptr, layout unsafe.Pointer) unsafe.Pointer {
	// Find the first availavble object.
	objIndex := uintptr(bits.TrailingZeros(^uint(hdr.live)))
	if objIndex >= 1<<((softPageScale-ptrScaleBytes)-class) {
		// There are no free objects.
		return nil
	}

	// Mark the object as live.
	hdr.live |= uintptr(1) << objIndex

	// Populate the layout bitmap.
	insertBitmapLayout(&hdr.layout, class, objIndex, layout)

	// Calculate the result address.
	return unsafe.Add(base, (objIndex<<class)<<ptrScaleBytes)
}

func insertBitmapLayout(
	dst *[smallMasks]uintptr,
	class uintptr,
	index uintptr,
	layout unsafe.Pointer,
) {
	// Parse the layout.
	size, mask := parseLayout(uintptr(layout))
	if size > ptrBits {
		runtimePanic("gc: large bitmap for small object")
	}

	// Repeat the mask.
	// TODO: repeat in the compiler?
	classPtrs := uintptr(1) << class
	for size < min(ptrBits, classPtrs) {
		mask |= mask << size
		size <<= 1
	}

	// Insert the mask into the bitmap, clearing the old value.
	shidx := index << class
	dstWord := &dst[shidx>>ptrScaleBits]
	oldMask := *dstWord
	shidx &= ptrBits - 1
	replaceMask := ((uintptr(1) << classPtrs) - 1) << shidx
	*dstWord = (oldMask &^ replaceMask) | ((mask << shidx) & replaceMask)
}

func parseLayout(layout uintptr) (size, mask uintptr) {
	if layout == 0 {
		// This is an unknown layout.
		// Treat all possible pointers as valid.
		return ptrBits, ^uintptr(0)
	} else if layout&1 != 0 {
		// Layout is stored directly in the integer value.
		// Extract values from the bitfields.
		size = (layout >> 1) & (ptrBits - 1)
		mask = layout >> (ptrScaleBits + 1)
	} else {
		// Layout is stored separately in a global object.
		// Load the size.
		size = *(*uintptr)(unsafe.Pointer(layout))
		layout += ptrBytes
		if size > ptrBits {
			// This is a pointer to a large bitmap.
			return size, layout
		}

		// Load the bitmap.
		rem := (size + 7) / 8
		for rem > 0 {
			rem -= 1
			mask <<= 8
			mask |= uintptr(*(*uint8)(unsafe.Pointer(layout + rem)))
		}
	}
	if size == 0 {
		runtimePanic("gc: zero-length layout")
	}

	return
}

func (hdr *slabHeaderLarge) alloc(base unsafe.Pointer, class uintptr, layout unsafe.Pointer) unsafe.Pointer {
	// Find the first availavble object.
	objIndex := uintptr(bits.TrailingZeros(^uint(hdr.live)))
	if objIndex >= 1<<((softPageScale-ptrScaleBytes)-class) {
		// There are no free objects.
		return nil
	}

	// Mark the object as live.
	hdr.live |= uintptr(1) << objIndex

	// Save the object layout.
	hdr.layout[objIndex] = uintptr(layout)

	// Calculate the result address.
	return unsafe.Add(base, (objIndex<<class)<<ptrScaleBytes)
}

func SetFinalizer(obj interface{}, finalizer interface{}) {
	// Unimplemented.
}

func ReadMemStats(m *MemStats) {
	// Unimplemented.
}

func free(ptr unsafe.Pointer) {
	// TODO: free objects on request, when the compiler knows they're unused.
}
