//go:build gc.precise

// This implements the block-based GC as a partially precise GC. This means that
// for most heap allocations it is known which words contain a pointer and which
// don't. This should in theory make the GC faster (because it can skip
// non-pointer object) and have fewer false positives in a GC cycle. It does
// however use a bit more RAM to store the layout of each object.
//
// The pointer/non-pointer information for objects is stored in the first word
// of the object. It is described below but in essence it contains a bitstring
// of a particular size. This size does not indicate the size of the object:
// instead the allocated object is a multiple of the bitstring size. This is so
// that arrays and slices can store the size of the object efficiently. The
// bitstring indicates where the pointers are in the object (the bit is set when
// the value may be a pointer, and cleared when it certainly isn't a pointer).
// Some examples (assuming a 32-bit system for the moment):
//
// | object type | size | bitstring | note
// |-------------|------|-----------|------
// | int         | 1    |   0       | no pointers in this object
// | string      | 2    |  01       | {pointer, len} pair so there is one pointer
// | []int       | 3    | 001       | {pointer, len, cap}
// | [4]*int     | 1    |   1       | even though it contains 4 pointers, an array repeats so it can be stored with size=1
// | [30]byte    | 1    |   0       | there are no pointers so the layout is very simple
//
// The garbage collector scans objects by starting at the first word value in
// the object. If the least significant bit of the bitstring is clear, it is
// skipped (it's not a pointer). If the bit is set, it is treated as if it could
// be a pointer. The garbage collector continues by scanning further words in
// the object and checking them against the corresponding bit in the bitstring.
// Once it reaches the end of the bitstring, it wraps around (for arrays,
// slices, strings, etc).
//
// The layout as passed to the runtime.alloc function and stored in the object
// is a pointer-sized value. If the least significant bit of the value is set,
// the bitstring is contained directly inside the value, of the form
// pppp_pppp_ppps_sss1.
//   * The 'p' bits indicate which parts of the object are a pointer.
//   * The 's' bits indicate the size of the object. In this case, there are 11
//     pointer bits so four bits are enough for the size (0-15).
//   * The lowest bit is always set to distinguish this value from a pointer.
// This example is for a 16-bit architecture. For example, 32-bit architectures
// use a layout format of pppppppp_pppppppp_pppppppp_ppsssss1 (26 bits for
// pointer/non-pointer information, 5 size bits, and one bit that's always set).
//
// For larger objects that don't fit in an uintptr, the layout value is a
// pointer to a global with a format as follows:
//     struct {
//         size uintptr
//         bits [...]uint8
//     }
// The 'size' field is the number of bits in the bitstring. The 'bits' field is
// a byte array that contains the bitstring itself, in little endian form. The
// length of the bits array is ceil(size/8).

package runtime

import "unsafe"

const preciseHeap = true

type gcLayout struct {
	layout uintptr
}

func (gcl *gcLayout) set(ptr unsafe.Pointer) {
	gcl.layout = uintptr(ptr)
}

// scan an object using this layout information.
// The starting address is inclusive and the ending address is exclusive.
func (gcl gcLayout) scan(start, end uintptr) {
	layout := gcl.layout
	switch {
	case layout == 0:
		// Unknown layout. Assume all words in the object could be pointers.
		markRoots(start, end)

	case layout&1 != 0:
		// Layout is stored directly in the integer value.
		// Determine format of bitfields in the integer.
		const layoutBits = uint64(unsafe.Sizeof(layout) * 8)
		var sizeFieldBits uint64
		switch layoutBits { // note: this switch should be resolved at compile time
		case 16:
			sizeFieldBits = 4
		case 32:
			sizeFieldBits = 5
		case 64:
			sizeFieldBits = 6
		default:
			runtimePanic("unknown pointer size")
		}

		// Extract values from the bitfields.
		// See comment at the top of this file for more information.
		size := (layout >> 1) & (1<<sizeFieldBits - 1)
		mask := layout >> (1 + sizeFieldBits)

		// Scan with this mask.
		scanSimple(start, end, size, mask)

	default:
		// Layout is stored separately in a global object.
		layoutAddr := unsafe.Pointer(layout)
		size := *(*uintptr)(layoutAddr)
		bitmapAddr := unsafe.Add(layoutAddr, unsafe.Sizeof(uintptr(0)))
		bitmap := unsafe.Slice((*uint8)(bitmapAddr), (size+7)/8)

		// Scan with this bitmap.
		scanComplex(start, end, size, bitmap)
	}
}

// scan pointers in an object using a mask of offsets in each element.
// The starting address is inclusive and the ending address is exclusive.
// The element size is size*unsafe.Alignof(uintptr(0)).
func scanSimple(start, end, size, mask uintptr) {
	if mask == 0 {
		// There are no pointers in this object.
		return
	}

	rem := end - start
	step := size * unsafe.Alignof(start)
	for rem >= step {
		scanWithMask(start, mask)
		rem -= step
		start += step
	}
}

// scan pointers in an object using a bitmap of offsets in each element.
func scanComplex(start, end, size uintptr, bitmap []uint8) {
	rem := end - start
	step := size * unsafe.Alignof(start)
	for rem >= step {
		scanWithBitmap(start, bitmap)
		rem -= step
		start += step
	}
}

// scan pointers in an object using a large bitmap of offsets.
func scanWithBitmap(addr uintptr, bitmap []uint8) {
	const maskStep = 8 * unsafe.Alignof(addr)
	for i, mask := range bitmap {
		scanWithMask(addr+uintptr(i)*maskStep, uintptr(mask))
	}
}

// scan pointers in an object using a mask of offsets.
func scanWithMask(addr uintptr, mask uintptr) {
	// TODO: use ctz if available?
	for mask != 0 {
		if mask&1 != 0 {
			markRoot(addr, *(*uintptr)(unsafe.Pointer(addr)))
		}
		mask >>= 1
		addr += unsafe.Alignof(addr)
	}
}
