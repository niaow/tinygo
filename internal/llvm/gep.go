package llvm

/*
#include "bindings.h"
*/
import "C"

// GEPMode is used to specify the attributes to attach to a GEP.
type GEPMode C.LLVMGoGEPMode

const (
	// GEPInboundsForwards ("inbounds nuw") combines GEPInbounds ("inbounds") with GEPUnsigned ("nuw").
	// This contains the constraints of GEPForwards ("nsw nuw") because GEPInbounds ("inbounds") contains GEPSigned ("nusw").
	// This should be used for operations which always produce a safe interior pointer.
	// This is replaced with GEPInbounds for compatibility when using LLVM 18 and below.
	GEPInboundsForwards GEPMode = C.LLVMGoGEPInboundsForwards

	// GEPInbounds ("inbounds") requires the address to remain within the source object after adding each offset.
	// The end of an object (base + size) is also considered inbounds.
	//
	// It includes the constraints of GEPSigned ("nusw") but not GEPUnsigned ("nuw").
	// This is true even for LLVM 18 and below where GEPSigned is not otherwise available.
	//
	// The overflow constraints are stronger than GEPSigned.
	// This is because the (unsigned) address cannot overflow when each (signed) offset is added individually.
	// GEPSigned only requres the combined offset to not overflow when added to the base pointer.
	// The following is valid for GEPSigned but not GEPInbounds: base + -MAX + MAX
	//
	// Unlike GEPInboundsForwards, this can be used to index backwards within an object.
	GEPInbounds GEPMode = C.LLVMGoGEPInbounds

	// GEPForwards ("nusw nuw") combines GEPSigned and GEPUnsigned.
	// The combined constraints imply that all indices must be positive.
	// This is replaced with GEPWrapping for compatibility when using LLVM 18 and below.
	GEPForwards GEPMode = C.LLVMGoGEPForwards

	// GEPSigned ("nusw") provides a mix of signed and unsigned overflow constraints:
	// - index truncation preserves the signed value
	// - signed multiplication of a converted index by the element size will not overflow
	// - when adding the offsets from left to right, no additions experience signed overflow
	// - addition of the signed combined offset to the unsigned base pointer will not overflow
	// This is replaced with GEPWrapping for compatibility when using LLVM 18 and below.
	GEPSigned GEPMode = C.LLVMGoGEPSigned

	// GEPUnsigned ("nuw") indicates that none of the GEP's internal operations will experience unsigned overflow:
	// - index truncation must not discard any set bits
	// - unsigned multiplication of a converted index by the element size will not overflow
	// - unsigned addition of offsets to the base pointer will not overflow
	// These constraints ensure that the resulting address is >= the base address.
	// This is replaced with GEPWrapping for compatibility when using LLVM 18 and below.
	GEPUnsigned GEPMode = C.LLVMGoGEPUnsigned

	// GEPWrapping allows any internal operation to wrap.
	GEPWrapping GEPMode = C.LLVMGoGEPWrapping
)

// IndexPointer creates the constant &base[index], treating base as a pointer to an array of elemType.
//
// Negative indices and overflows are valid unless the GEPMode restricts them.
// Use the GEPInboundsForwards for safe interior pointers with positive indices.
//
// Use Builder.IndexPointer to create runtime expressions.
//
// https://llvm.org/docs/LangRef.html#getelementptr-instruction
// https://llvm.org/docs/GetElementPtr.html
func (elemType Type) IndexPointer(base, index Constant, mode GEPMode) Constant {
	return Constant{Value{C.LLVMGoConstIndexPointer(
		elemType.ptr,
		base.ptr,
		index.ptr,
		C.LLVMGoGEPMode(mode),
	)}}
}

// FieldPointer creates a constant pointer to an element of an aggregate type.
//
// Negative indices (for arrays) and overflows are valid unless the GEPMode restricts them.
// Use the GEPInboundsForwards for safe interior pointers with positive indices.
//
// Use Builder.FieldPointer to create runtime expressions.
//
// https://llvm.org/docs/LangRef.html#getelementptr-instruction
// https://llvm.org/docs/GetElementPtr.html
func (aggType Type) FieldPointer(base Constant, index int32, mode GEPMode) Constant {
	return Constant{Value{C.LLVMGoConstFieldPointer(
		aggType.ptr,
		base.ptr,
		C.uint32_t(index),
		C.LLVMGoGEPMode(mode),
	)}}
}

// IndexPointer creates the pointer &base[index], treating base as a pointer to an array of elemType.
//
// Negative indices and overflows are valid unless the GEPMode restricts them.
// Use the GEPInboundsForwards for safe interior pointers with positive indices.
//
// This operation is performed elementwise on vectors.
//
// Use Type.IndexPointer to create constant expressions.
//
// https://llvm.org/docs/LangRef.html#getelementptr-instruction
// https://llvm.org/docs/GetElementPtr.html
func (b Builder) IndexPointer(elemType Type, base, index Value, mode GEPMode, name string) Value {
	return Value{C.LLVMGoCreateIndexPointer(
		b.ptr,
		elemType.ptr,
		base.ptr,
		index.ptr,
		C.LLVMGoGEPMode(mode),
		stringRef(name),
	)}
}

// FieldPointer creates a pointer to an element of an aggregate type.
// Negative indices (for arrays) and overflows are valid unless the GEPMode restricts them.
// Use the GEPInboundsForwards for safe interior pointers with positive indices.
// This operation is performed elementwise on vectors.
// Use Type.FieldPointer to create constant expressions.
//
// https://llvm.org/docs/LangRef.html#getelementptr-instruction
// https://llvm.org/docs/GetElementPtr.html
func (b Builder) FieldPointer(aggType Type, base Value, index int32, mode GEPMode, name string) Value {
	return Value{C.LLVMGoCreateFieldPointer(
		b.ptr,
		aggType.ptr,
		base.ptr,
		C.uint32_t(index),
		C.LLVMGoGEPMode(mode),
		stringRef(name),
	)}
}
