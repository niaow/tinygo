package llvm

/*
#include "bindings.h"
*/
import "C"

// TODO: doc
type Value struct {
	ptr C.LLVMValueRef
}

// String formats the value as it would be printed in IR.
func (v Value) String() string {
	cstr := C.LLVMPrintValueToString(v.ptr)
	defer C.LLVMDisposeMessage(cstr)
	return C.GoString(cstr)
}

func (v Value) Type() Type {
	return Type{C.LLVMTypeOf(v.ptr)}
}

// TODO: pointer to opaque

// TODO: generalized decoding of constants?
