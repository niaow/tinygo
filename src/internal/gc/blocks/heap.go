package blocks

import "unsafe"

const gcDebug = false

const ptrPerBlock = 4
const BytesPerBlock = unsafe.Sizeof([ptrPerBlock]uintptr{})
const BlocksPerGroup = maskBits
const MetaSizePerGroup = unsafe.Sizeof(mask(0)) + unsafe.Sizeof((*uintptr)(nil))

func HeapFromRange(start, end unsafe.Pointer) Heap {
	const fullGroupSize = BlocksPerGroup*BytesPerBlock +
		unsafe.Sizeof(mask(0)) +
		unsafe.Sizeof((*uintptr)(nil))
	return Heap{
		Base: start,
		Len:  (uintptr(end) - uintptr(start)) / fullGroupSize,
	}
}

type Heap struct {
	Base unsafe.Pointer
	Len  uintptr
}

func (h Heap) blocks() [][BlocksPerGroup][BytesPerBlock]byte {
	return sliceFromPtr[[BlocksPerGroup][BytesPerBlock]byte](h.Base, h.Len)
}

func (h Heap) startMasks() []mask {
	ptr := unsafe.Add(h.Base, h.Len*BlocksPerGroup*BytesPerBlock)
	return sliceFromPtr[mask](ptr, h.Len)
}

func (h Heap) skipBacks() []*uintptr {
	ptr := unsafe.Add(h.Base, h.Len*(BlocksPerGroup*BytesPerBlock+maskBytes))
	return sliceFromPtr[*uintptr](ptr, h.Len)
}

func (h Heap) Init(free *FreeList) {
	h.Grow(0, free)
}

// Grow the heap.
// This inserts the new free range but does not merge it.
// The GC may need to be re-run after growing in some extreme cases.
func (dst Heap) Grow(oldLen uintptr, free *FreeList) {
	if dst.Len == oldLen {
		return
	}

	// Create an old version of the heap.
	src := Heap{
		Base: dst.Base,
		Len:  oldLen,
	}

	// Clear the new skipBacks and copy the old ones.
	// The skipBacks are at the end, so they must be copied first.
	dstSkipBacks := dst.skipBacks()
	clear(dstSkipBacks[oldLen:])
	copy(dstSkipBacks, src.skipBacks())

	// Add new startMasks and copy the old ones.
	dstStartMasks := dst.startMasks()
	copy(dstStartMasks, src.startMasks())
	dstStartMasks[oldLen] = 1
	clear(dstStartMasks[oldLen:][1:])

	// Insert the new free range.
	free.push(
		(*freeRange)(unsafe.Pointer(&dst.blocks()[oldLen])),
		(dst.Len-src.Len)*BlocksPerGroup*BytesPerBlock,
	)
}

func AllocSize(size uintptr) (uintptr, bool) {
	const inc = unsafe.Sizeof(objHeader{}) + BytesPerBlock - 1
	if size > ^uintptr(0)-inc {
		return 0, true
	}
	return (size + inc) &^ (BytesPerBlock - 1), false
}

func InnerSize(size uintptr) uintptr {
	return size - unsafe.Sizeof(objHeader{})
}

func (h Heap) TryAlloc(free *FreeList, size uintptr, layout uintptr) unsafe.Pointer {
	_ = [1]byte{}[size%BytesPerBlock]

	// Pop a free range.
	ptr := free.pop(size)
	if ptr == nil {
		return nil
	}

	// Initialize the object header.
	obj := (*objHeader)(ptr)
	obj.next = 0
	obj.layout.set(layout)

	// Add the end block as a start (unless this allocation touches the heap end).
	rel := uintptr(ptr) - uintptr(h.Base)
	endBlock := (rel + size) / BytesPerBlock
	endMask := endBlock / BlocksPerGroup
	if endMask < h.Len {
		h.startMasks()[endMask] |= 1 << (endBlock % BlocksPerGroup)
	}

	// Set the skip-backs.
	startBlock := rel / BytesPerBlock
	h.setSkipBack(&obj.next, startBlock, endBlock)

	return unsafe.Add(ptr, unsafe.Sizeof(objHeader{}))
}

func (h Heap) Sweep(dst *FreeList) uintptr {
	if h.Len == 0 {
		return 0
	}

	// Clear the free list, turning free ranges into unmarked objects.
	dst.clear()

	blocks := h.blocks()
	startMasks := h.startMasks()
	skipBacks := h.skipBacks()

	var free unsafe.Pointer
	var totalFreeBytes uintptr
	for i, m := range startMasks {
		var nextBitIdx mask
		for m != 0 {
			// Find the next start.
			skip := m.skipForwards()
			nextBitIdx += mask(skip)
			bitIdx := nextBitIdx - 1

			// Examine the start.
			// TODO: prove bounds instead of using modulo
			bitIdx %= mask(BlocksPerGroup)
			ptr := unsafe.Pointer(&blocks[i][bitIdx])
			head := (*uintptr)(ptr)
			if *head == 0 {
				// This object is unmarked.
				if free == nil {
					// Start a new free range.
					free = ptr
				} else {
					// Merge with the existing free range.
					startMasks[i] &^= 1 << bitIdx
				}
			} else {
				// Unmark this object.
				*head = 0
				if free != nil {
					// End the previous free range.
					freeLen := uintptr(ptr) - uintptr(free)
					totalFreeBytes += freeLen
					dst.push((*freeRange)(free), freeLen)
					free = nil
				}
			}
		}
		if free != nil {
			// Clear the skipBack.
			skipBacks[i] = nil
		}
	}
	if free != nil {
		// End the trailing free range.
		freeLen := uintptr(unsafe.Pointer(&blocks[0])) + h.Len*BlocksPerGroup*BytesPerBlock - uintptr(free)
		totalFreeBytes += freeLen
		dst.push((*freeRange)(free), freeLen)
	}

	return totalFreeBytes
}

func (h Heap) setSkipBack(ptr *uintptr, start uintptr, end uintptr) {
	skips := sliceNoBounds(
		h.skipBacks(),
		int((start/BlocksPerGroup)+1),
		int((end+(BlocksPerGroup-1))/BlocksPerGroup),
	)
	// TODO: faster fill?
	// NOTE: range breaks here because signed nonsense
	for i := uintptr(0); i < uintptr(len(skips)); i++ {
		skips[i] = ptr
	}
}

func (h Heap) Mark(dst *ScanStack, ptr, at uintptr) {
	// Find the head from the root address.
	head := h.findHead(ptr)
	if head == nil {
		// This is not a heap pointer.
		return
	}

	// Mark the object.
	if !casUintptr(head, 0, uintptr(unsafe.Pointer(dst.next))-1) {
		// This was either a free list entry or an already-marked object.
		return
	}
	if gcDebug {
		println("found unmarked pointer", ptr, "at address", at)
	}

	// Add the object to the scan stack.
	obj := (*objHeader)(unsafe.Pointer(head))
	if !obj.layout.hasPointers() {
		// This object does not contain pointers.
		// There is no need to scan it.
		// TODO: is this check worthwhile here (vs FinishScan)?
		return
	}
	dst.next = obj
}

func casUintptr(ptr *uintptr, old, new uintptr) bool {
	// NOTE: Switch to atomic.CompareAndSwapUintptr when we implement parallel marking.
	match := *ptr == old
	if match {
		*ptr = new
	}
	return match
}

func (h Heap) findHead(addr uintptr) *uintptr {
	// Find the root address within the heap.
	rel := addr - uintptr(h.Base)
	blockIdx := rel / BytesPerBlock
	maskIdx := blockIdx / BlocksPerGroup
	if maskIdx >= h.Len {
		// This is not a heap address.
		return nil
	}

	// Scan backwards to find the start.
	m := h.startMasks()[maskIdx]
	bitIdx := blockIdx % BlocksPerGroup
	m <<= (BlocksPerGroup - 1) - bitIdx
	if m == 0 {
		// Use skipBacks.
		return h.skipBacks()[maskIdx]
	}
	bitIdx -= uintptr(m.skipBackwards() - 1)
	// TODO: prove bitIdx bounds check instead of doing this?
	bitIdx %= maskBits
	return (*uintptr)(unsafe.Pointer(&h.blocks()[maskIdx][bitIdx]))
}

// FinishScan pops all objects off the stack and marks other reachable objects.
func (h Heap) FinishScan(stack *ScanStack) {
	for {
		// Pop the next object off the stack.
		obj := stack.pop()
		if obj == nil {
			break
		}

		// Scan the object.
		h.scanObject(stack, obj)
	}
}

func (h Heap) scanObject(dst *ScanStack, obj *objHeader) {
	h.scanWithLayout(dst, h.objMem(obj), obj.layout)
}

// objMem finds the object's backing memory (excluding the header).
func (h Heap) objMem(obj *objHeader) []byte {
	size := h.objMoreBlocks(obj) * BytesPerBlock
	size += BytesPerBlock - unsafe.Sizeof(objHeader{})
	ptr := unsafe.Add(unsafe.Pointer(obj), unsafe.Sizeof(objHeader{}))
	return sliceFromPtr[byte](ptr, size)
}

// objMoreBlocks counts the additional object blocks after the header.
func (h Heap) objMoreBlocks(obj *objHeader) uintptr {
	// Find the address within the heap, skipping the first block.
	rel := uintptr(unsafe.Pointer(obj)) - uintptr(h.Base)
	rel += BytesPerBlock
	blockIdx := rel / BytesPerBlock

	// Find the mask from the first block group.
	maskIdx := blockIdx / BlocksPerGroup
	if maskIdx >= h.Len {
		// The object occupies the last block of the heap.
		return 0
	}
	masks := h.startMasks()[maskIdx:]
	m := masks[0]

	// Shift off irrelevant blocks.
	bitIdx := blockIdx % BlocksPerGroup
	m >>= bitIdx

	var blocks uintptr
	if m == 0 {
		// Subtract the irrelevant blocks.
		blocks -= bitIdx

		// Scan forwards to find the next nonzero mask.
		for {
			blocks += BlocksPerGroup
			masks = masks[1:]
			if len(masks) == 0 {
				return blocks
			}
			m = masks[0]
			if m != 0 {
				break
			}
		}
	}

	// Count the blocks until the next start.
	// Add them to the size.
	blocks += uintptr(m.skipForwards() - 1)

	return blocks
}

func (h Heap) CountStarts() uintptr {
	var n uintptr
	for _, m := range h.startMasks() {
		n += uintptr(m.count())
	}
	return n
}

type ScanStack struct {
	next *objHeader
}

func (s *ScanStack) pop() *objHeader {
	obj := s.next
	if obj != nil {
		s.next = (*objHeader)(unsafe.Pointer(obj.next + 1))
	}
	return obj
}

// objHeader holds metadata for an object.
type objHeader struct {
	// next tracks the mark/scan status of the object.
	// It is set to 0 until the object is marked.
	// When marked, this holds the address of the next object to scan minus 1.
	// If there is no next object to scan, this is set to ^uintptr(0).
	next uintptr

	// layout holds the locations of pointers in the object.
	layout layout
}
