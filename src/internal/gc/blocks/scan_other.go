//go:build !avr

package blocks

import (
	"unsafe"
)

// MarkRoots conservatively scans the provided memory.
// Every valid pointer location is marked.
// There are no special alignment requirements.
func (h Heap) MarkRoots(dst *ScanStack, mem []byte) {
	// Align the slice start.
	const ptrAlign = unsafe.Alignof(uintptr(0))
	alignSkip := -uintptr(unsafe.Pointer(unsafe.SliceData(mem))) % ptrAlign
	if alignSkip >= uintptr(len(mem)) {
		// Aligning this forwards would skip/reach the end.
		return
	}
	mem = mem[alignSkip:]

	// Scan conservatively.
	h.scanConservative(dst, mem)
}

// scanConservative marks all potential pointers in mem.
// The memory must be pointer-aligned.
// The length is rounded down to a multiple of pointer width.
func (h Heap) scanConservative(dst *ScanStack, mem []byte) {
	h.scanAllPointers(dst, bytesToUintptrs(mem))
}

// scanPrecise marks pointers in mem using the provided layout information.
// The memory must be pointer-aligned.
// The layout is encoded one of three ways:
//
// - layout == 0: all-pointer layout
//
// - layout&1 != 0: inline mask
//   - The lowest bit is the tag bit used to identify the encoding.
//   - Above that is the element length (in pointers) minus 1.
//   - The next element length bits are used as a reversed mask of pointer locations.
//   - All remaining bits must be 0.
//
// - layout&1 == 0: address of a global constant structure
//   - starts with a uintptr element length in pointers
//   - immediately followed by a bitmap
//   - divide element length by mask width (rounding up) to get length of mask array
//
// The memory is treated as an array of the layout element:
// - The memory is rounded down to a multiple of the element length.
// - The layout repeats after every element length.
//
// NOTE: This layout is slightly different for AVR (see scan_avr.go).
func (h Heap) scanPrecise(dst *ScanStack, mem []byte, layout uintptr) {
	// Convert the memory to a slice of pointer locations.
	memPtrs := bytesToUintptrs(mem)

	// Decode and use the layout to scan.
	switch {
	case layout == 0:
		// The layout is all pointers.
		h.scanAllPointers(dst, memPtrs)

	case layout&1 != 0:
		// The layout is encoded as an inline mask.
		layout >>= 1
		elemLen := (layout % maskBits) + 1
		// TODO: rewrite to a rotate right by elemLen?
		ptrs := mask((layout / maskBits) << (maskBits - elemLen))
		h.scanSimple(dst, memPtrs, ptrs, elemLen)

	default:
		// The layout is a pointer to a global constant.
		ptr := unsafe.Pointer(layout)
		elemLen := *(*uintptr)(ptr)
		if elemLen == 0 {
			// TODO: what should we do here?
		}
		ptr = unsafe.Add(ptr, unsafe.Sizeof(uintptr(0)))
		bitmap := unsafe.Slice((*mask)(ptr), (elemLen+maskBits-1)/maskBits)
		// TODO: use to scanSimple if len(bitmap) == 1?
		h.scanComplex(dst, memPtrs, bitmap, elemLen)
	}
}

// scanAllPointers marks each element of mem.
func (h Heap) scanAllPointers(dst *ScanStack, mem []uintptr) {
	for i, ptr := range mem {
		h.Mark(dst, ptr, uintptr(unsafe.Pointer(&mem[i])))
	}
}

// scanSimple marks pointers in an array, using a mask to specify pointer locations within the element type.
// Each element is scanned using scanWithMask.
// The length of the array is len(mem)/elemLen (rounding down).
func (h Heap) scanSimple(dst *ScanStack, mem []uintptr, ptrs mask, elemLen uintptr) {
	if ptrs == 0 {
		return
	}

	for ; uintptr(len(mem)) >= elemLen; mem = mem[elemLen:] {
		h.scanWithMask(dst, mem[:elemLen], ptrs)
	}
}

// scanComplex marks pointers in an array, using a large bitmap to specify pointer locations within the element type.
// Each element is scanned using scanWithBitmap.
// The length of the array is len(mem)/elemLen (rounding down).
func (h Heap) scanComplex(dst *ScanStack, mem []uintptr, layout []mask, elemLen uintptr) {
	for ; uintptr(len(mem)) >= elemLen; mem = mem[elemLen:] {
		h.scanWithBitmap(dst, mem[:elemLen], layout)
	}
}

// scanWithBitmap marks pointers at locations specified by a large bitmap.
// A bit at index j in mask i corresponds to the location mem[i*maskBits + (maskBits-1) - j].
func (h Heap) scanWithBitmap(dst *ScanStack, mem []uintptr, layout []mask) {
	for i, m := range layout {
		if m == 0 {
			// Skip empty masks.
			continue
		}

		// Find the backing memory.
		innerMem := sliceNoBounds(mem, int(maskBits)*i, len(mem))
		innerMem = innerMem[0:min(len(innerMem), int(maskBits))]

		// Scan the memory with the mask.
		h.scanWithMask(dst, innerMem, m)
	}
}

// scanWithMask marks pointers at locations specified by a mask.
// A bit at index i corresponds to the location mem[(maskBits-1) - i].
// The mask must not be zero.
// The mask is backwards because skipBackwards is simpler/faster on many architectures:
// - ARMv5+ (without CSSC), MIPS (r6): has CLZ but not CTZ
// - RISC-V (without B), MIPS (pre-r6): can test top bit with signed < 0, other bit tests require a seperate bitwise and instruction
func (h Heap) scanWithMask(dst *ScanStack, mem []uintptr, ptrs mask) {
	for {
		// Skip leading zeroes.
		skip := ptrs.skipBackwards()
		mem = sliceNoBounds(mem, skip-1, len(mem))

		// Mark the pointer.
		ptr := &sliceNoBounds(mem, 0, skip)[0]
		h.Mark(dst, *ptr, uintptr(unsafe.Pointer(ptr)))

		// Move to the next position.
		mem = sliceNoBounds(mem, 1, len(mem))
		if ptrs == 0 {
			break
		}
	}
}

// bytesToUintptrs casts the slice memory to []uintptr.
// The memory must have uintptr's alignment.
// The length is rounded down.
// The capacity is replaced with the length.
func bytesToUintptrs(mem []byte) []uintptr {
	return sliceFromPtr[uintptr](
		unsafe.Pointer(unsafe.SliceData(mem)),
		uintptr(len(mem))/unsafe.Sizeof(uintptr(0)),
	)
}
