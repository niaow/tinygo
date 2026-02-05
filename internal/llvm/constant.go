package llvm

/*
#include "bindings.h"
*/
import "C"
import (
	"unsafe"
)

// Constant represents an LLVM constant value.
// This is a wrapper around Value to provide extra type checking.
type Constant struct {
	Value
}

// Zero returns the zero value of the type (null ptr, int 0, etc.).
func (t Type) Zero() Constant {
	// TODO: is this valid for all types?
	return Constant{Value{C.LLVMConstNull(t.ptr)}}
}

// IsConstantZero checks if this is the zero value constant of the type.
// This is equivalent to v == Value(v.Type().Zero()).
func (v Value) IsConstantZero() bool {
	// TODO: is this valid for all types?
	return C.LLVMIsNull(v.ptr) != C.false
}

// ConstBool creates a boolean constant.
func (c Context) ConstBool(value bool) Constant {
	var v uint64
	if value {
		v = 1
	}
	return c.ConstInt(1, v)
}

// ConstInt creates an integer constant of the specified bit width.
// The provided words are interpreted as little-endian (least-significant-first).
// The value is zero-extended or truncated to fit the type.
// It panics if the requested bit width is outside the range [1, 2^23].
// Future calls with equivalent constants will return the same value.
func (c Context) ConstInt(bits uint32, value ...uint64) Constant {
	return Constant{Value{C.LLVMGoConstInt(
		c.ptr,
		intWidth(bits),
		(*C.uint64_t)(unsafe.Pointer(unsafe.SliceData(value))),
		C.size_t(len(value)),
	)}}
}

// AsConstInt reads the value and bit-width of an integer constant.
// If the value is not an integer constant, this returns a nil ConstInt.
func (v Value) AsConstInt() ConstInt {
	return ConstInt{C.LLVMGoAsConstInt(v.ptr)}
}

// ConstInt is used to reference the contents of an LLVM integer constant.
// The backing memory is owned by the context, and is freed when it is destroyed.
type ConstInt struct {
	raw C.LLVMGoIntData
}

// NotZero checks if the value is not equal to zero.
// This is useful for reading booleans.
// It panics if c is nil.
func (c ConstInt) NotZero() bool {
	if c.IsNil() {
		panic("nil constant")
	}
	slice := c.Slice()
	for _, v := range slice {
		if v != 0 {
			return true
		}
	}
	return false
}

// Int64 converts a small integer (<= 64 bits) to an int64 by sign-extending it.
// It panics if c is nil or wider than 64 bits.
func (c ConstInt) Int64() int64 {
	slice := c.Slice()
	if len(slice) != 1 {
		panic("not a small integer constant")
	}
	shift := 64 - c.raw.bits
	return int64(slice[0]<<shift) >> shift
}

// Uint64 converts a small integer (<= 64 bits) to an uint64 by zero-extending it.
// It panics if c is nil or wider than 64 bits.
func (c ConstInt) Uint64() uint64 {
	slice := c.Slice()
	if len(slice) != 1 {
		panic("not a small integer constant")
	}
	return slice[0]
}

// MaybeUint64 converts a small integer (<= 64 bits) to a uint64.
// It returns nil if c is nil or wider than 64 bits.
// The returned pointer is owned by the context and must not be modified.
func (c ConstInt) MaybeUint64() *uint64 {
	slice := c.Slice()
	if len(slice) != 1 {
		return nil
	}
	return &slice[0]
}

// Slice returns a little-endian slice of words representing the integer.
// The width is rounded up to a multiple of 64 bits, filling extra bits with 0.
// It returns a nil slice if c is nil.
//
//go:nobounds
func (c ConstInt) Slice() []uint64 {
	// If c is nil, c.raw.value will be nil and c.raw.bits will be 1.
	// The bits value is <= 2^29, but the compiler does not know that.
	// Temporarily widen to 64 bits to prove that it will not overflow.
	return unsafe.Slice((*uint64)(c.raw.value), (uint64(c.raw.bits)+63)/64)
}

func (c ConstInt) IsNil() bool {
	return c.raw.bits == 0
}

// Bits returns the bit-width of the constant.
// It returns 0 if c is nil.
func (c ConstInt) Bits() uint32 {
	return uint32(c.raw.bits)
}

// NOTE: special floats have conversion issues, just bitcast from int for now

// Create a constant array from the provided elements.
func (elemTy Type) ConstArray(elements ...Constant) Constant {
	return Constant{Value{C.LLVMGoConstArray(
		elemTy.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(elements))),
		castArrayLen(uint64(len(elements))),
	)}}
}

// ConstIntArray creates an array constant from a slice of integers.
// The memory is copied to a flat buffer owned by LLVM instead of converting each element to a seperate constant.
// NOTE: This cannot be a method because type parameters are not allowed.
func ConstIntArray[T sizedInt](ctx Context, data []T) Constant {
	return Constant{Value{C.LLVMGoConstIntArray(
		ctx.ptr,
		(*C.char)(unsafe.Pointer(unsafe.SliceData(data))),
		C.size_t(unsafe.Sizeof(T(0))),
		C.size_t(len(data)),
	)}}
}

type sizedInt interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~int8 | ~int16 | ~int32 | ~int64
}

// Create a struct constant with an implicit type.
// The result type is a literal non-packed struct formed from the element types.
// Use Type.ConstStruct to create structs with a specific/named type.
func (c Context) ConstStruct(elements ...Constant) Constant {
	return Constant{Value{C.LLVMGoConstImplicitStruct(
		c.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(elements))),
		C.size_t(len(elements)),
	)}}
}

// Create a struct constant with an explicit type.
func (t Type) ConstStruct(elements ...Constant) Constant {
	return Constant{Value{C.LLVMGoConstExplicitStruct(
		t.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(elements))),
		C.size_t(len(elements)),
	)}}
}
