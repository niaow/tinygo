package llvm

/*
#include "bindings.h"
*/
import "C"
import "unsafe"

// Context holds constants and types?
type Context struct {
	ptr C.LLVMContextRef
}

// CreateContext creates a new LLVM context.
// A context is not thread safe.
// Call Destroy to free memory used by the context.
func CreateContext() Context {
	return Context{C.LLVMContextCreate()}
}

// Destroy frees memory used by the context.
func (c Context) Destroy() {
	C.LLVMContextDispose(c.ptr)
}

// TODO: put this somewhere better

func stringRef(str string) C.LLVMGoStringRef {
	return C.LLVMGoStringRef{
		ptr: (*C.char)(unsafe.Pointer(unsafe.StringData(str))),
		len: C.size_t(len(str)),
	}
}

//go:nobounds
func refAsString(ref C.LLVMGoStringRef) string {
	return unsafe.String(
		(*byte)(unsafe.Pointer(ref.ptr)),
		ref.len,
	)
}

// goCloneString is used by C to copy strings to the Go heap.
//
//export goCloneString
func goCloneString(dst unsafe.Pointer, src C.LLVMGoStringRef) {
	*(*string)(dst) = string(refAsBytes(src))
}

//go:nobounds
func refAsBytes(ref C.LLVMGoStringRef) []byte {
	return unsafe.Slice(
		(*byte)(unsafe.Pointer(ref.ptr)),
		ref.len,
	)
}
