//go:build avr

package blocks

import (
	"unsafe"
)

// MarkRoots conservatively scans the provided memory.
func (h Heap) MarkRoots(dst *ScanStack, mem []byte) {
	h.scanConservative(dst, mem)
}

// scanConservative marks all potential pointers in mem.
// AVR lacks alignment, so any pair of adjacent bytes is a valid pointer.
func (h Heap) scanConservative(dst *ScanStack, mem []byte) {
	if len(mem) < 2 {
		// This is smaller than a pointer.
		return
	}

	// Load the first byte.
	low := mem[0]
	mem = mem[1:]
	for len(mem) > 0 {
		// TODO: verify that LLVM merges the registers properly
		high := mem[0]
		h.Mark(dst, uintptr(low)|(uintptr(high)<<8), uintptr(unsafe.Pointer(&mem[0]))-1)
		low = high
		mem = mem[1:]
	}
}

// scanPrecise marks pointers in mem using the provided layout information.
// The memory must be pointer-aligned.
// The layout is encoded one of three ways:
//
// - layout == 0: unknown layout, scan conservatively (see scanConservative)
//
// - layout&1 != 0: inline mask
//   - The lowest bit is the tag bit used to identify the encoding.
//   - The rest of the lower byte is the element length (in bytes) minus 2.
//   - The upper byte is a mask of pointer locations.
//     A bit at index i corresponds to the location mem[i:][0:2].
//
// - layout&1 == 0: address of a global constant structure
//   - aligned to 2 bytes
//   - starts with a uintptr element length in bytes
//   - immediately followed by a bitmap
//   - divide element length by mask width (rounding up) to get length of mask array
//   - TODO: move this to program memory somehow?
//
// The memory is treated as an array of the layout element:
// - The memory is rounded down to a multiple of the element length.
// - The layout repeats after every element length.
//
// NOTE: This layout is slightly different for all other platforms (see scan_other.go).
func (h Heap) scanPrecise(dst *ScanStack, mem []byte, layout uintptr) {
	switch {
	case layout == 0:
		// The layout is unknown.
		// Scan this conservatively.
		h.scanConservative(dst, mem)

	case layout&1 != 0:
		// The layout is stored inline.
		elemLen := (uint8(layout) >> 1) + 2
		ptrs := mask(layout >> 8)
		h.scanSimple(dst, mem, ptrs, elemLen)

	default:
		// The layout is a pointer to a constant structure.
		// It consists of a length followed by masks.
		ptr := unsafe.Pointer(layout)
		elemLen := *(*uintptr)(ptr)
		if elemLen == 0 {
			// TODO: what should we do here?
		}
		ptr = unsafe.Add(ptr, unsafe.Sizeof(uintptr(0)))
		bitmap := unsafe.Slice((*mask)(ptr), (elemLen+maskBits-1)/maskBits)
		// NOTE: we could use scanSimple if len(bitmap) == 1, but that should never happen
		h.scanComplex(dst, mem, bitmap, elemLen)
	}
}

// scanSimple marks pointers in an array, using a mask to specify pointer locations within the element type.
// Each element is scanned using scanWithMask.
// The length of the array is len(mem)/elemLen (rounding down).
func (h Heap) scanSimple(dst *ScanStack, mem []byte, ptrs mask, elemLen uint8) {
	if ptrs == 0 {
		return
	}

	for ; len(mem) >= int(elemLen); mem = mem[elemLen:] {
		h.scanWithMask(dst, mem[:elemLen], ptrs)
	}
}

// scanComplex marks pointers in an array, using a large bitmap to specify pointer locations within the element type.
// Each element is scanned using scanWithBitmap.
// The length of the array is len(mem)/elemLen (rounding down).
func (h Heap) scanComplex(dst *ScanStack, mem []byte, layout []mask, elemLen uintptr) {
	for ; uintptr(len(mem)) >= elemLen; mem = mem[elemLen:] {
		h.scanWithBitmap(dst, mem[:elemLen], layout)
	}
}

// scanWithBitmap marks pointers at locations specified by a large bitmap.
// A bit at index j in mask i corresponds to the location mem[i*maskBits + j].
func (h Heap) scanWithBitmap(dst *ScanStack, mem []byte, layout []mask) {
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
// A bit at index i corresponds to the location mem[i].
// The mask must not be zero.
func (h Heap) scanWithMask(dst *ScanStack, mem []byte, ptrs mask) {
	for {
		// Skip trailing zeroes.
		skip := ptrs.skipForwards()
		mem = sliceNoBounds(mem, skip-1, len(mem))

		// Load and mark the pointer.
		at := unsafe.Pointer((*[2]byte)(sliceNoBounds(mem, 0, 2)))
		h.Mark(dst, *(*uintptr)(at), uintptr(at))

		// Move to the next position.
		mem = sliceNoBounds(mem, 2, len(mem))
		ptrs <<= 1
		if ptrs == 0 {
			break
		}
	}
}
