package llvm

/*
#include "bindings.h"
*/
import "C"
import "unsafe"

type Metadata struct {
	ptr C.LLVMMetadataRef
}

// There isn't really a sane way to print metadata by itself.
// Only NamedMDNode can be printed sanely.
// If you use it otherwise, it will just print the address of the metadata pointer.

// MetadataString creates metadata with a string value.
// Calls with the same string will produce the same Metadata value.
func (ctx Context) MetadataString(str string) Metadata {
	return Metadata{C.LLVMMDStringInContext2(
		ctx.ptr,
		(*C.char)(unsafe.Pointer(unsafe.StringData(str))),
		C.size_t(len(str)),
	)}
}

// MetadataNode creates indistinct Metadata from a list of operands.
// Calls with the same operand list will produce the same Metadata value.
func (ctx Context) MetadataNode(operands ...Metadata) Metadata {
	return Metadata{C.LLVMMDNodeInContext2(
		ctx.ptr,
		(*C.LLVMMetadataRef)(unsafe.Pointer(unsafe.SliceData(operands))),
		C.size_t(len(operands)),
	)}
}

// TODO: do we need to be able to create distinct metadata?

// MetadataValue creates a value containing the provided metadata.
func (ctx Context) MetadataValue(md Metadata) Value {
	return Value{C.LLVMMetadataAsValue(ctx.ptr, md.ptr)}
}

// Metadata converts the value to metadata.
// If the value is metadata (from Context.MetadataValue), this will return the contained metadata.
// Otherwise, it will create a metadata containing the value.
func (v Value) Metadata() Metadata {
	return Metadata{C.LLVMValueAsMetadata(v.ptr)}
}

// StringValue gets the contents of a string Metadata (from Context.MetadataString).
// This will panic if the metadata is not a string.
// The returned string memory is owned by the context.
func (md Metadata) StringValue() string {
	str, ok := md.MaybeStringValue()
	if !ok {
		panic("metadata is not a string")
	}
	return str
}

// MaybeStringValue gets the contents of a string Metadata (from Context.MetadataString).
// This will return the empty string and false if the metadata is not a string.
// The returned string memory is owned by the context.
func (md Metadata) MaybeStringValue() (string, bool) {
	// NOTE: The LLVM C bindings contain a similar function, but:
	// - It accepts a LLVMValueRef instead of an LLVMMetadataRef.
	// - The length is cast to C.unsigned (may overflow on 64-bit systems).
	ref := C.LLVMGoMetadataString(md.ptr)
	return refAsString(ref), ref.ptr != nil
}

// NodeOperands gets the contents of a node Metadata (from Context.MetadataNode).
// This will panic if the metadata is not a node.
func (md Metadata) NodeOperands() []Metadata {
	operands := md.MaybeNodeOperands()
	if operands == nil {
		panic("metadata is not a node")
	}
	return operands
}

// MaybeNodeOperands gets the contents of a node Metadata (from Context.MetadataNode).
// This will return nil if the metadata is not a node.
func (md Metadata) MaybeNodeOperands() []Metadata {
	n := C.LLVMGoMetadataOperandsCount(md.ptr)
	if n == C.LLVMGoMetadataNotANode {
		return nil
	}
	buf := make([]Metadata, n)
	C.LLVMGoMetadataOperands(md.ptr, (*C.LLVMMetadataRef)(unsafe.Pointer(unsafe.SliceData(buf))))
	return buf
}

// UnwrapValue gets the contents of a value Metadata (from Value.Metadata).
// This will panic if the Metadata is not a wrapped value.
func (md Metadata) UnwrapValue() Value {
	value, ok := md.MaybeUnwrapValue()
	if !ok {
		panic("metadata is not a value")
	}
	return value
}

// MaybeUnwrapValue gets the contents of a value Metadata (from Value.Metadata).
// This will return false if the Metadata is not a wrapped value.
func (md Metadata) MaybeUnwrapValue() (Value, bool) {
	ptr := C.LLVMGoUnwrapMetadataValue(md.ptr)
	return Value{ptr}, ptr != nil
}
