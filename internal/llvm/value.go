package llvm

/*
#include "bindings.h"
*/
import "C"
import (
	"unsafe"
)

// TODO: doc
type Value struct {
	ptr C.LLVMValueRef
}

// String formats the value as an operand in IR.
func (v Value) String() string {
	var dst string
	C.LLVMGoValueShortString(unsafe.Pointer(&dst), v.ptr)
	return dst
}

// LongString formats the the value as it would be printed alone in IR.
// This formats the whole instruction/global/function.
func (v Value) LongString() string {
	var dst string
	C.LLVMGoValueLongString(unsafe.Pointer(&dst), v.ptr)
	return dst
}

// Type returns the type of the value.
func (v Value) Type() Type {
	return Type{C.LLVMTypeOf(v.ptr)}
}

// TODO: generalized decoding of constants?
