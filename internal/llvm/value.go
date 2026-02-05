package llvm

/*
#include "bindings.h"
*/
import "C"
import "unsafe"

// TODO: doc
type Value struct {
	ptr C.LLVMValueRef
}

// String formats the value as it would be printed in IR.
func (v Value) String() string {
	var dst string
	C.LLVMGoValueString(unsafe.Pointer(&dst), v.ptr)
	return dst
}

func (v Value) Type() Type {
	return Type{C.LLVMTypeOf(v.ptr)}
}

// TODO: pointer to opaque

// TODO: generalized decoding of constants?
