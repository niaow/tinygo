package llvm

/*
#include "bindings.h"
*/
import "C"
import "unsafe"

type Module struct {
	ptr C.LLVMModuleRef
}

func (ctx Context) CreateModule(
	name string,
	triple string,
	dataLayout DataLayout,
) Module {
	return Module{C.LLVMGoNewModule(
		ctx.ptr,
		stringRef(name),
		stringRef(triple),
		dataLayout.ptr,
	)}
}

func (mod Module) Destroy() {
	C.LLVMDisposeModule(mod.ptr)
}

func (mod Module) String() string {
	var dst string
	C.LLVMGoModuleString(unsafe.Pointer(&dst), mod.ptr)
	return dst
}

// Get a global value (function/variable) by name.
func (mod Module) Get(name string) (Global, bool) {
	ptr := C.LLVMGoGetNamedValue(mod.ptr, stringRef(name))
	return Global{Constant{Value{ptr}}}, ptr != nil
}

type Global struct {
	Constant
}

// ContainedType gets the type of the global.
// Functions/intrinsics will always use function types.
// External globals may use opaque structure types.
func (g Global) ContainedType() Type {
	return Type{C.LLVMGlobalGetValueType(g.ptr)}
}

func (mod Module) CreateFunction() {
	panic("TODO")
}

type Signature struct {
}

func (c Context) Signature(
	ret Type,
	args Type,
	varargs bool,
	returnsTwice bool,
	convention int, // TODO
	argAttrs []AttributeSet,
) Signature {
	panic("TODO")
}

type SignatureOptions struct {
	// Return is the resulting type of the call.
	Return Type

	// Arguments are the types of the call arguments, excluding varargs.
	Arguments []Type

	// VarArgs indicates that the function has C varargs in addition to the normal arguments.
	VarArgs bool

	// ReturnsTwice indicates that the function can return twice (e.g. setjmp).
	// NOTE: This is printed as an attribute in the IR, but it is not internally implemented as such.
	ReturnsTwice bool

	// TODO: calling convention

	// FunctionAttributes hold attributes directly attached to the function itself.
	FunctionAttributes AttributeSet

	// ReturnAttributes hold attributes attached to the function's return.
	ReturnAttributes AttributeSet

	// ArgumentAttributes hold attributes attached to the function's arguments.
	// This must be at most as long as the argument list (no limit if varargs).
	// If it is shorter than the argument list, attributes are only added to the first arguments.
	ArgumentAttributes []AttributeSet
}

type Function struct {
	Global
}

// TODO: addrspace is special, dont put it in link config
// TODO: TLS model does not apply to functions

type LinkConfig struct {
	// TODO: linkage type
	// TODO: unnamed addr

	// ExternallyInitialized indicates that this global is initialized by another module.
	ExternallyInitialized bool
}

// AppendBasicBlock appends a basic block to the end of the function.
func (fn Function) AppendBasicBlock(name string) BasicBlock {
	return BasicBlock{C.LLVMGoAppendBasicBlock(fn.ptr, stringRef(name))}
}

type BasicBlock struct {
	ptr C.LLVMBasicBlockRef
}

// AddAfter adds a new basic block after bb.
func (bb BasicBlock) AddAfter(name string) BasicBlock {
	return BasicBlock{C.LLVMGoAppendBasicBlockAfter(bb.ptr, stringRef(name))}
}
