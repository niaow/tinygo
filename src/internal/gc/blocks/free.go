package blocks

import "unsafe"

// FreeList holds a list of free memory ranges.
// TODO: more explanation
type FreeList struct {
	head *freeRange
}

// TODO: document
func (list *FreeList) push(node *freeRange, size uintptr) {
	// Set the new node's size.
	node.size = size

	// Find the corresponding size node.
	insertAt := &list.head
	var sizeNode *freeRange
	for {
		sizeNode = *insertAt
		if sizeNode == nil || sizeNode.size >= size {
			break
		}
		insertAt = &sizeNode.nextSize
	}

	if sizeNode == nil || sizeNode.size > size {
		// Insert as a new size node.
		*insertAt = node
		node.nextWithSize = nil
		node.nextSize = sizeNode
	} else {
		// Insert into this size bucket.
		next := sizeNode.nextWithSize
		sizeNode.nextWithSize = node
		node.nextWithSize = next
	}
}

// TODO: document
func (list *FreeList) pop(size uintptr) unsafe.Pointer {
	// Find a long-enough size node.
	removeFrom := &list.head
	var sizeNode *freeRange
	var newSize uintptr
	for {
		sizeNode = *removeFrom
		if sizeNode == nil {
			// There are no more size nodes.
			return nil
		}
		newSize = sizeNode.size
		if newSize >= size {
			break
		}
		removeFrom = &sizeNode.nextSize
	}

	// Remove a node of the selected size.
	popped := sizeNode.nextWithSize
	if popped != nil {
		sizeNode.nextWithSize = popped.nextWithSize
	} else {
		*removeFrom = sizeNode.nextSize
		popped = sizeNode
	}
	ptr := unsafe.Pointer(popped)

	if newSize > size {
		// Re-insert the excess space.
		list.push((*freeRange)(unsafe.Add(ptr, size)), newSize-size)
	}

	return ptr
}

func (list *FreeList) clear() {
	nextSize := list.head
	list.head = nil
	for ; nextSize != nil; nextSize = nextSize.nextSize {
		for nextWithSize := nextSize; nextWithSize != nil; nextWithSize = nextWithSize.nextWithSize {
			nextWithSize.size = 0
		}
	}
}

//go:noinline
func (list FreeList) Count() (entries, bytes uintptr) {
	for nextSize := list.head; nextSize != nil; nextSize = nextSize.nextSize {
		// NOTE: A count-with-size + multiply would be faster (1 loop counter instead of 2).
		// Unfortunately, LLVM rewrites that back into this.
		nextWithSize := nextSize
		size := nextSize.size
		for {
			entries++
			bytes += size
			nextWithSize = nextWithSize.nextWithSize
			if nextWithSize == nil {
				break
			}
		}
	}
	return
}

func (list FreeList) Dump() {
	for nextSize := list.head; nextSize != nil; nextSize = nextSize.nextSize {
		println("- size:", nextSize.size)
		for nextWithSize := nextSize; nextWithSize != nil; nextWithSize = nextWithSize.nextWithSize {
			println("  -", nextWithSize)
		}
	}
}

// freeRange represents a range of free memory.
// It is stored at the start of the free range.
type freeRange struct {
	freeRangeInner
	_ [BytesPerBlock - unsafe.Sizeof(freeRangeInner{})]byte
}

type freeRangeInner struct {
	// size of the range in bytes.
	size uintptr

	// nextWithSize is the next free range with this size.
	// This functions as an inner linked list.
	nextWithSize *freeRange

	// nextSize is the next larger free range.
	// This is not set for ranges on the nextWithSize list.
	nextSize *freeRange
}
