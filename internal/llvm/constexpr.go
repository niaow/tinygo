package llvm

/*
#include "bindings.h"
*/
import "C"

// Truncate converts a constant to a smaller integer type by discarding upper bits.
// NOTE: Unlike runtime trunc and other constants, there is no constant nsw/nuw here.
//
// https://llvm.org/docs/LangRef.html#trunc-to-instruction
func (c Constant) Truncate(to Type) Constant {
	return Constant{Value{C.LLVMConstTrunc(c.ptr, to.ptr)}}
}

// PtrToInt converts a pointer to an integer, truncating or zero-extending to match the destination type.
// The result can be cast back to a pointer with IntToPtr.
//
// https://llvm.org/docs/LangRef.html#ptrtoint-to-instruction
func (c Constant) PtrToInt(to Type) Constant {
	return Constant{Value{C.LLVMConstPtrToInt(c.ptr, to.ptr)}}
}

// TODO: add ptrtoaddr when we suppport LLVM 22

// IntToPtr converts an integer to a pointer, zero-extending or truncating to match the index type.
//
// https://llvm.org/docs/LangRef.html#inttoptr-to-instruction
func (c Constant) IntToPtr(to Type) Constant {
	return Constant{Value{C.LLVMConstIntToPtr(c.ptr, to.ptr)}}
}

// BitCast reinterprets a constant as another type by reusing the raw bit layout.
// NOTE: Big-endian vectors w/ byte-multiple elements are reversed when bitcasting.
//
// https://llvm.org/docs/LangRef.html#bitcast-to-instruction
func (c Constant) BitCast(to Type) Constant {
	return Constant{Value{C.LLVMConstBitCast(c.ptr, to.ptr)}}
}

// TODO: add addrspacecast if we need it

// NOTE: getelementptr is handled in gep.go

// TODO: add extractelement if we need it

// TODO: add insertelement if we need it

// TODO: add shufflevector if we need it

// Create a sum of two integer constants.
// https://llvm.org/docs/LangRef.html#add-instruction
func (x Constant) Add(y Constant, noUnsignedWrap, noSignedWrap bool) Constant {
	return Constant{Value{C.LLVMGoConstAdd(
		x.ptr,
		y.ptr,
		C.bool(noUnsignedWrap),
		C.bool(noSignedWrap),
	)}}
}

// Create a subtraction of integer constants.
// https://llvm.org/docs/LangRef.html#sub-instruction
func (minuend Constant) Subtract(subtrahend Constant, noUnsignedWrap, noSignedWrap bool) Constant {
	return Constant{Value{C.LLVMGoConstSub(
		minuend.ptr,
		subtrahend.ptr,
		C.bool(noUnsignedWrap),
		C.bool(noSignedWrap),
	)}}
}

// Create a bitwise exclusive or of integer constants.
// https://llvm.org/docs/LangRef.html#xor-instruction
func (x Constant) Xor(y Constant) Constant {
	return Constant{Value{C.LLVMConstXor(x.ptr, y.ptr)}}
}
