package llvm

/*
#include "bindings.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// TODO: doc
type Type struct {
	ptr C.LLVMTypeRef
}

// String formats the type as it would be printed in IR.
func (t Type) String() string {
	var dst string
	C.LLVMGoTypeString(unsafe.Pointer(&dst), t.ptr)
	return dst
}

// Void gets the void type (for returns) in this context.
func (c Context) Void() Type {
	return Type{C.LLVMVoidTypeInContext(c.ptr)}
}

// Int finds or creates an integer type with the specified bit width.
// Future calls with the same bit-width will return the same type.
// It panics if the requested bit width is outside the range [1, 2^23].
func (c Context) Int(bits uint32) Type {
	return Type{C.LLVMIntTypeInContext(c.ptr, intWidth(bits))}
}

func intWidth(bits uint32) C.unsigned {
	if bits == 0 || bits > 1<<23 {
		panic(fmt.Errorf("invalid integer bit width: %d", bits))
	}
	return C.unsigned(bits)
}

// Float32 gets the single-precision floating point type in this context.
func (c Context) Float32() Type {
	return Type{C.LLVMFloatTypeInContext(c.ptr)}
}

// Float64 gets the double-precision floating point type in this context.
func (c Context) Float64() Type {
	return Type{C.LLVMDoubleTypeInContext(c.ptr)}
}

// Pointer finds or creates an opaque pointer type in the specified address space.
// Future calls with the same address space will return the same type.
func (c Context) Pointer(addrSpace uint32) Type {
	// TODO: check addrSpace index validity
	return Type{C.LLVMPointerTypeInContext(c.ptr, C.unsigned(addrSpace))}
}

// Array creates a new array type.
func (c Context) Array(len uint64, element Type) Type {
	return Type{C.LLVMGoArrayType(element.ptr, castArrayLen(len))}
}

// The type of an array length was C.unsigned up until LLVM 17.
// We must check that the length fits if this is the case.
// TODO: remove this cast when we drop LLVM 16 support
func castArrayLen(len uint64) C.LLVMGoArrayLen {
	trunc := C.LLVMGoArrayLen(len)
	if uint64(trunc) != len {
		panic("array len too big")
	}
	return trunc
}

// LiteralStruct finds or creates a literal struct with the provided element types.
// Future calls with the same element list will return the same type.
func (c Context) LiteralStruct(elements ...Type) Type {
	return Type{C.LLVMStructTypeInContext(
		c.ptr,
		(*C.LLVMTypeRef)(unsafe.Pointer(unsafe.SliceData(elements))),
		C.unsigned(len(elements)),
		C.false,
	)}
}

// NamedStruct creates a new named struct type.
// Each call returns a unique type, even if the names and elements are the same.
func (c Context) NamedStruct(name string, elements ...Type) Type {
	return Type{C.LLVMGoCreateNamedStruct(
		c.ptr,
		(*C.LLVMTypeRef)(unsafe.Pointer(unsafe.SliceData(elements))),
		C.size_t(len(elements)),
		stringRef(name),
	)}
}

// Info queries information about a type.
func (t Type) Info() TypeInfo {
	return TypeInfo{C.LLVMGoGetTypeInfo(t.ptr)}
}

// TypeInfo holds basic information about a type.
type TypeInfo struct {
	raw C.LLVMGoTypeInfo
}

// IntegerWidth gets the width of an integer type in bits.
// It panics on non-integer types.
func (info TypeInfo) IntegerWidth() uint32 {
	info.assertKind(TypeKindInteger)
	return uint32(info.raw.len)
}

// AddressSpace returns the address space of a pointer.
// It panics on non-pointer types.
// NOTE: This does not support TypeKindTypedPointer.
func (info TypeInfo) AddressSpace() uint32 {
	info.assertKind(TypeKindPointer)
	return uint32(info.raw.len)
}

// StructElements returns the element types of the struct.
// It panics on non-struct types.
// The caller must not modify the returned slice.
// The slice is owned by the type's context and will be freed when it is destroyed.
func (info TypeInfo) StructElements() []Type {
	info.assertKind(TypeKindStruct)
	return unsafe.Slice((*Type)(info.raw.ptr), info.raw.len)
}

// ArrayShape returns the length and element type of the array.
// It panics on non-array types.
func (info TypeInfo) ArrayShape() (uint64, Type) {
	info.assertKind(TypeKindArray)
	return uint64(info.raw.len), Type{C.LLVMTypeRef(info.raw.ptr)}
}

// assertKind is used to detect and panic on type kind mismatches.
func (info TypeInfo) assertKind(kind TypeKind) {
	if info.Kind() != kind {
		panic("type kind mismatch")
	}
}

// Kind retuns the kind of the type.
func (info TypeInfo) Kind() TypeKind {
	return TypeKind(info.raw.kind)
}

type TypeKind C.LLVMGoTypeKind

const (
	// TypeKindVoid is a non-type used for functions that do not return anything.
	TypeKindVoid = C.LLVMGoVoidTypeKind
	// TypeKindInteger is an integer type of arbitrary width.
	TypeKindInteger = C.LLVMGoIntegerTypeKind
	// TypeKindPointer is an opaque pointer type.
	// A context may have multiple pointer types - one for each address space.
	TypeKindPointer = C.LLVMGoPointerTypeKind

	// TypeKindFloat16 is the IEEE 754 binary16 floating-point format.
	TypeKindFloat16 = C.LLVMGoFloat16Kind
	// TypeKindFloat32 is the IEEE 754 binary32 floating-point format.
	// It is equivalent to Go's float32 type.
	TypeKindFloat32 = C.LLVMGoFloat32Kind
	// TypeKindFloat64 is the IEEE 754 binary64 floating-point format.
	// It is equivalent to Go's float64 type.
	TypeKindFloat64 = C.LLVMGoFloat64Kind
	// TypeKindFloat128 is the IEEE 754 binary128 floating-point format.
	TypeKindFloat128 = C.LLVMGoFloat128Kind

	// TypeKindBFloat16 is a 16-bit floating-point type with an 8-bit exponent.
	TypeKindBFloat16 = C.LLVMGoBFloat16Kind

	// TypeKindX87Float80 is the 80-bit floating-point format used by the x87 FPU.
	TypeKindX87Float80 = C.LLVMGoX87Float80Kind
	// TypeKindPPCFloat128Kind is the IBM Extended Double format sometimes used on PowerPC.
	// It is stored as a pair of IEEE 754 binary64 floats.
	TypeKindPPCFloat128Kind = C.LLVMGoPPCFloat128Kind

	// TypeKindStruct is usually a tuple of other types.
	// A special-case opaque struct type is sometimes used for external globals.
	TypeKindStruct = C.LLVMGoStructTypeKind
	// TypeKindArray repeats a type to form a list.
	// TODO: better words
	TypeKindArray = C.LLVMGoArrayTypeKind
	// TypeKindFixedVector is formed by concatenating a repeated type.
	// This is like an array except:
	// - Arithmetic and comparisons can be performed elementwise.
	// - If the element type is a multiple of 8 bits, the order is reversed in memory.
	// - If the element type is not a multiple of 8 bits, the elements are conatenated into an integer in memory.
	TypeKindFixedVector = C.LLVMGoFixedVectorTypeKind
	// TypeKindScalableVector is like TypeKindFixedVector, except the length is determined at runtime.
	// This is used by the ARM SVE and RISC-V V instruction sets.
	TypeKindScalableVector = C.LLVMGoScalableVectorTypeKind

	// TypeKindFunction is used to define the signature of a function call.
	// Functions themselves do not use this type - they are passed as opaque pointers.
	TypeKindFunction = C.LLVMGoFunctionTypeKind
	// TypeKindLabel is the type of a function's basic block.
	// Blocks are values, but can only be used directly as constant destinations to branch instructions.
	// It is possible to convert a label to a pointer with blockaddress.
	TypeKindLabel = C.LLVMGoLabelTypeKind
	// TypeKindToken is a symbolic type used for exception handling and some intrinsics.
	TypeKindToken = C.LLVMGoTokenTypeKind
	// TypeKindMetadata is used for passing metadata as a value?
	// TODO: when is this used?
	TypeKindMetadata = C.LLVMGoMetadataTypeKind

	// TypeKindX86MMX is used by x86 MMX intrinsics prior to LLVM 20.
	// TODO: remove when we drop LLVM 19 support
	TypeKindX86MMX = C.LLVMGoX86MMXTypeKind
	// TypeKindX86AMX is used by x86 AMX intrinsics.
	TypeKindX86AMX = C.LLVMGoX86AMXTypeKind
	// TypeKindTypedPointer is a special type used for some GPU platforms (SPIR-V/DXIL).
	// This is not relevant to TinyGo.
	TypeKindTypedPointer = C.LLVMGoTypedPointerTypeKind
	// TypeKindTargetExtension is used for bespoke platform-specific types.
	// Most uses are irrelevant to TinyGo, except maybe these could indirectly end up in the IR:
	// - ARM SVE's predicate-as-counter type
	// - RISC-V V's vector tuples
	TypeKindTargetExtension = C.LLVMGoTargetExtensionTypeKind
)

func (kind TypeKind) String() string {
	return kindNames[kind]
}

var kindNames = [...]string{
	TypeKindVoid:            "void",
	TypeKindInteger:         "integer",
	TypeKindPointer:         "pointer",
	TypeKindFloat16:         "float16",
	TypeKindFloat32:         "float32",
	TypeKindFloat64:         "float64",
	TypeKindFloat128:        "float128",
	TypeKindBFloat16:        "bfloat16",
	TypeKindX87Float80:      "x87_float80",
	TypeKindPPCFloat128Kind: "ppc_float128",
	TypeKindStruct:          "struct",
	TypeKindArray:           "array",
	TypeKindFixedVector:     "fixed_vector",
	TypeKindScalableVector:  "scalable_vector",
	TypeKindFunction:        "function",
	TypeKindLabel:           "label",
	TypeKindToken:           "token",
	TypeKindMetadata:        "metadata",
	TypeKindX86MMX:          "x86_mmx",
	TypeKindX86AMX:          "x86_amx",
	TypeKindTypedPointer:    "typed_pointer",
	TypeKindTargetExtension: "target_extension",
}
