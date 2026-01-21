package llvm

/*
#include "bindings.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

type TargetMachine struct {
	ptr C.LLVMTargetMachineRef
}

func CreateTargetMachine(
	triple string,
	cpu string,
	features string,
	relocationModel RelocationModel,
	codeModel CodeModel,
	codeGenLevel int32,
) (TargetMachine, error) {
	var errMsg string
	result := C.LLVMGoCreateTargetMachine(
		C.LLVMGoTargetMachineConfig{
			triple:          stringRef(triple),
			cpu:             stringRef(cpu),
			features:        stringRef(features),
			relocationModel: relocationModel,
			codeModel:       codeModel,
			codeGenLevel:    C.int(min(max(codeGenLevel, 0), 3)),
		},
		unsafe.Pointer(&errMsg),
	)
	if result == nil {
		return TargetMachine{}, errors.New("failed to create target machine: " + errMsg)
	}
	return TargetMachine{result}, nil
}

// RelocationModel defines how references to symbols should be encoded.
type RelocationModel = C.LLVMGoRelocationModel

const (
	// RelocationModelDefault selects the default model for the target triple.
	RelocationModelDefault RelocationModel = C.LLVMGoRelocModelDefault

	// RelocationModelStatic resolves all symbol addresses at static link time.
	// This should be used for microcontroller firmware.
	RelocationModelStatic RelocationModel = C.LLVMGoRelocModelStatic

	// RelocationModelPositionIndependent generates position-independent code.
	// This should be used for shared libraries.
	RelocationModelPositionIndependent RelocationModel = C.LLVMGoRelocModelPIC

	// RelocationModelDynamicPositionDependent is non-PIC code which can link against PIC code.
	// This is obscure and should generally be avoided.
	RelocationModelDynamicPositionDependent RelocationModel = C.LLVMGoRelocModelDynamicNoPIC

	// RelocationModelROPI accesses code and constants through PC-relative offsets resolved at static link time.
	// This is ARM-specific.
	RelocationModelROPI RelocationModel = C.LLVMGoRelocModelROPI

	// RelocationModelRWPI accesses mutable globals through relative offsets resolved at static link time.
	// This is ARM-specific.
	// Register r9 is reserved for use as a base address.
	RelocationModelRWPI RelocationModel = C.LLVMGoRelocModelRWPI

	// RelocationModelROPIRWPI combines RelocationModelROPI and RelocationModelRWPI.
	RelocationModelROPIRWPI RelocationModel = C.LLVMGoRelocModelROPIRWPI
)

// CodeModel constrains the possible addresses of code.
// TODO: explain this without just saying that it matches the AMD64 System V ABI (does it?).
type CodeModel = C.LLVMGoCodeModel

const (
	CodeModelDefault CodeModel = C.LLVMGoCodeModelDefault
	CodeModelTiny    CodeModel = C.LLVMGoCodeModelTiny
	CodeModelSmall   CodeModel = C.LLVMGoCodeModelSmall
	CodeModelKernel  CodeModel = C.LLVMGoCodeModelKernel
	CodeModelMedium  CodeModel = C.LLVMGoCodeModelMedium
	CodeModelLarge   CodeModel = C.LLVMGoCodeModelLarge
)

func (tm TargetMachine) Destroy() {
	C.LLVMDisposeTargetMachine(tm.ptr)
}

type DataLayout struct {
	// This is a pointer to a llvm::DataLayout object.
	// The C bindings still use the old name for the class.
	ptr C.LLVMTargetDataRef
}

func (tm TargetMachine) DataLayout() DataLayout {
	return DataLayout{C.LLVMCreateTargetDataLayout(tm.ptr)}
}

func (td DataLayout) Destroy() {
	C.LLVMDisposeTargetData(td.ptr)
}

// AddressSpaces identifies the memory address spaces used for certain operations:
// - program: the address space used to store functions
// - global: the address space typically used to store global varaibles
// - alloca: the address space used by stack allocations
// These are equal to the default address space 0 on most platforms.
// AVR uses address space 1 for program memory (flash).
func (td DataLayout) AddressSpaces() (program, global, alloca uint32) {
	result := C.LLVMGoGetAddressSpaces(td.ptr)
	return uint32(result.program), uint32(result.global), uint32(result.alloca)
}

// PointerSize returns the size (in bytes) of a pointer in the specified address space.
func (td DataLayout) PointerSize(addrSpace uint32) uint32 {
	return uint32(C.LLVMPointerSizeForAS(td.ptr, C.unsigned(addrSpace)))
}

// ABITypeFixed computes the ABI size and alignment of a fixed-width type.
// This will panic if the type is scalable.
func (td DataLayout) ABITypeFixed(ty Type) (size, align uint64) {
	size, align, scalable := td.ABIType(ty)
	if scalable {
		panic("cannot compute size of scalable type")
	}
	return size, align
}

// ABIType computes the ABI size and alignment of a type.
// If the type is scalable, the size must be multiplied by vscale at runtime.
func (td DataLayout) ABIType(ty Type) (size, align uint64, scalable bool) {
	result := C.LLVMGoGetABIType(td.ptr, ty.ptr)
	return uint64(result.size), uint64(result.align), bool(result.scalable)
}
