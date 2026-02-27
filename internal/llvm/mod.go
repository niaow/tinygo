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

type GlobalVariable struct {
	Global
}

// CreateVariable appends a new global variable definition to the module.
func (mod Module) CreateVariable(
	name string,
	initializer Constant,
	options VariableOptions,
) GlobalVariable {
	var global GlobalVariable
	global.ptr = C.LLVMGoCreateGlobal(
		mod.ptr,
		stringRef(name),
		initializer.ptr,
		options.toC(),
	)
	return global
}

// CreateExternalVariable appends a new external global variable declaration to the module.
func (mod Module) CreateExternalVariable(
	name string,
	ty Type,
	options VariableOptions,
) GlobalVariable {
	var global GlobalVariable
	global.ptr = C.LLVMGoCreateExternalGlobal(
		mod.ptr,
		stringRef(name),
		ty.ptr,
		options.toC(),
	)
	return global
}

// VariableOptions holds optional configuration for a GlobalVariable.
type VariableOptions struct {
	// Constant indicates that the variable is immutable.
	Constant bool
	// AddrSpace is the address space to place the variable in.
	// Use DataLayout.AddressSpaces to find the standard address space for globals.
	AddrSpace uint32
	// TLS determines whether the symbol is thread-local, and if so what model to use.
	TLS TLSMode
	// Link specifies how the variable should be linked.
	Link LinkConfig
	// Attributes are applied to the variable.
	Attributes AttributeSet
	// TODO: metadata
	// ExternallyInitialized indicates that the global may be modified before it is initialized.
	ExternallyInitialized bool
}

func (options VariableOptions) toC() C.LLVMGoVariableOptions {
	return C.LLVMGoVariableOptions{
		isConstant:            C.bool(options.Constant),
		addrSpace:             C.unsigned(options.AddrSpace),
		tls:                   C.LLVMThreadLocalMode(options.TLS),
		link:                  options.Link.toC(),
		attrs:                 options.Attributes.ptr,
		externallyInitialized: C.bool(options.ExternallyInitialized),
	}
}

func (v GlobalVariable) Options() VariableOptions {
	return variableOptionsFromC(C.LLVMGoGetVariableOptions(v.ptr))
}

func variableOptionsFromC(src C.LLVMGoVariableOptions) VariableOptions {
	return VariableOptions{
		Constant:              bool(src.isConstant),
		AddrSpace:             uint32(src.addrSpace),
		TLS:                   TLSMode(src.tls),
		Link:                  linkConfigFromC(src.link),
		Attributes:            AttributeSet{src.attrs},
		ExternallyInitialized: bool(src.externallyInitialized),
	}
}

// TLSMode specifies the TLS model of a variable.
// The zero value is TLSNone which specifies that the variable is not thread-local.
type TLSMode C.LLVMThreadLocalMode

const (
	// TLSNone indicates that this variable is not thread-local.
	TLSNone TLSMode = C.LLVMNotThreadLocal
	// TLSGeneralDynamic is the default TLS mode.
	// It does not assume that any parameters are known at link time.
	// Use this for thread-local variables which are externally visible.
	TLSGeneralDynamic TLSMode = C.LLVMGeneralDynamicTLSModel
	// TLSLocalDynamic is used for TLS variables defined within the current shared object.
	// The offsets within the module are known at link time, but not the module ID.
	TLSLocalDynamic TLSMode = C.LLVMLocalDynamicTLSModel
	// TLSInitialExec is used for TLS variables defined within the current static object.
	// The constant offset of the variable is computed by the dynamic linker.
	TLSInitialExec TLSMode = C.LLVMInitialExecTLSModel
	// TLSLocalExec is used for TLS variables defined within the current executable (not a library).
	// The constant offset of the variable is computed at link time.
	TLSLocalExec TLSMode = C.LLVMLocalExecTLSModel
)

// String formats the TLS mode as it would be printed in IR.
func (mode TLSMode) String() string {
	return tlsModeNames[mode]
}

var tlsModeNames = [...]string{
	TLSNone:           "",
	TLSGeneralDynamic: "thread_local",
	TLSLocalDynamic:   "thread_local(localdynamic)",
	TLSInitialExec:    "thread_local(initialexec)",
	TLSLocalExec:      "thread_local(localexec)",
}

func (gv GlobalVariable) GetInitializer() (Constant, bool) {
	ptr := C.LLVMGetInitializer(gv.ptr)
	return Constant{Value{ptr}}, ptr != nil
}

func (gv GlobalVariable) SetInitializer(init Constant) {
	C.LLVMSetInitializer(gv.ptr, init.ptr)
}

type Function struct {
	Global
}

func (mod Module) CreateFunction(
	name string,
	signature Signature,
	link LinkConfig,
) Function {
	var fn Function
	fn.ptr = C.LLVMGoCreateFunction(
		mod.ptr,
		stringRef(name),
		signature.toC(),
		link.toC(),
	)
	return fn
}

type Signature struct {
	// Type holds the type of the function.
	Type Type

	// Convention is the calling convention for the function.
	Convention CallingConvention

	// AddrSpace specifies the address space of the function.
	AddrSpace uint32

	// Attributes is a list of attributes for the function/call.
	// NOTE: ABI attributes must match between the function and the call site.
	Attributes AttributeList
}

func (sig Signature) toC() C.LLVMGoSignature {
	return C.LLVMGoSignature{
		ty:        sig.Type.ptr,
		conv:      C.unsigned(sig.Convention),
		addrSpace: C.unsigned(sig.AddrSpace),
		attrs:     sig.Attributes.ptr,
	}
}

func (fn Function) Signature() Signature {
	return signatureFromC(C.LLVMGoFunctionSignature(fn.ptr))
}

func signatureFromC(src C.LLVMGoSignature) Signature {
	return Signature{
		Type:       Type{src.ty},
		Convention: CallingConvention(src.conv),
		AddrSpace:  uint32(src.addrSpace),
		Attributes: AttributeList{src.attrs},
	}
}

// CallingConvention is used to hold a calling convention ID.
type CallingConvention C.unsigned

// Standard calling conventions
const (
	// CallingConventionC ("ccc" or omitted in IR) is the C calling convention.
	// This is the default calling convention.
	// This calling convention supports varargs.
	// This calling convention is available on all targets.
	CallingConventionC CallingConvention = C.LLVMCCallConv
	// CallingConventionFast ("fastcc" in IR) is an internal calling convention that may be more efficient than CallingConventionC.
	// It does not conform to any ABI and may change between LLVM versions.
	// This calling convention is available on all targets.
	CallingConventionFast CallingConvention = C.LLVMFastCallConv
	// CallingConventionCold ("coldcc" in IR) is a calling convention that minimizes the impact on the caller.
	// Functions with CallingConventionCold cannot be inlined.
	// This calling convention is available on all targets.
	CallingConventionCold CallingConvention = C.LLVMColdCallConv
	// CallingConventionTail ("tailcc" in IR) is a calling convention that always supports tail calls.
	// This calling convention is available on all targets.
	// NOTE: This is not available through the C bindings???
	CallingConventionTail CallingConvention = 18
)

// LinkConfig contains general options for symbol linking.
// The zero value corresponds to a normal C exported symbol.
type LinkConfig struct {
	// Linkage specifies how references are resolved within or between modules.
	Linkage Linkage

	// UnnamedAddr controls merging of identical functions/constants.
	UnnamedAddr UnnamedAddr

	// Visibility controls how a symbol is treated after linking into the final executable/library.
	Visibility Visibility

	// DLLStorage controls imports/exports for PE/XCOFF libraries.
	DLLStorage DLLStorage

	// DSOLocal indicates that the compiler can assume a symbol will not be overriden at runtime.
	DSOLocal bool
}

func (config LinkConfig) toC() C.LLVMGoLinkConfig {
	return C.LLVMGoLinkConfig{
		linkage:     C.LLVMGoLinkage(config.Linkage),
		unnamedAddr: C.LLVMUnnamedAddr(config.UnnamedAddr),
		visibility:  C.LLVMVisibility(config.Visibility),
		dllStorage:  C.LLVMDLLStorageClass(config.DLLStorage),
		isDSOLocal:  C.bool(config.DSOLocal),
	}
}

func (g Global) LinkInfo() LinkConfig {
	return linkConfigFromC(C.LLVMGoLinkInfo(g.ptr))
}

func linkConfigFromC(src C.LLVMGoLinkConfig) LinkConfig {
	return LinkConfig{
		Linkage:     Linkage(src.linkage),
		UnnamedAddr: UnnamedAddr(src.unnamedAddr),
		Visibility:  Visibility(src.visibility),
		DLLStorage:  DLLStorage(src.dllStorage),
		DSOLocal:    bool(src.isDSOLocal),
	}
}

// Linkage specifies how references are resolved within or between modules.
type Linkage C.LLVMGoLinkage

const (
	// LinkageExternal ("external" in IR) allows the symbol to be linked from another module.
	// This linkage is also valid for function/variable declarations.
	// This is the default symbol linkage.
	LinkageExternal Linkage = C.LLVMGoLinkageExternal
	// LinkageAvailableExternally ("available_externally" in IR) is applied to copies of symbols defined elsewhere.
	// The copy can be inlined, but any remaining references will be linked externally.
	LinkageAvailableExternally Linkage = C.LLVMGoLinkageAvailableExternally
	// LinkageOnceAny ("linkonce" in IR) ensures that exactly one copy of the symbol is used consistently.
	// This linkage blocks inlining because the chosen copy is not known until link time.
	LinkageOnceAny Linkage = C.LLVMGoLinkageOnceAny
	// LinkageOnceODR ("linkonce_odr" in IR) specifies that all definitions of a symbol are identical.
	// Exactly one copy of the symbol is used consistently (like LinkageOnceAny).
	// Inlining is permitted because the local definition is known to match the chosen.
	// This is the default linkage for C++.
	LinkageOnceODR Linkage = C.LLVMGoLinkageOnceODR
	// LinkageWeakAny ("weak" in IR) matches LinkageOnceAny except unused symbols may not be discarded.
	LinkageWeakAny Linkage = C.LLVMGoLinkageWeakAny
	// LinkageWeakODR ("weak_odr" in IR) matches LinkageOnceODR except unused symbols may not be discarded.
	LinkageWeakODR Linkage = C.LLVMGoLinkageWeakODR
	// LinkageAppending ("appending" in IR) concatenates all definitions of a symbol.
	LinkageAppending Linkage = C.LLVMGoLinkageAppending
	// LinkageInternal ("internal" in IR) is used for symbols only usable within the current module.
	// It is still included in the symbol table (useful for debugging).
	// This matches C static linkage.
	LinkageInternal Linkage = C.LLVMGoLinkageInternal
	// LinkagePrivate ("private" in IR) is used for symbols only usable within the current module.
	// It is not included in the symbol table.
	LinkagePrivate Linkage = C.LLVMGoLinkagePrivate
	// LinkageExternalWeak ("extern_weak" in IR) behaves like LinkageWeakAny except undefined symbols are replaced with null.
	LinkageExternalWeak Linkage = C.LLVMGoLinkageExternalWeak
	// LinkageCommon ("common" in IR) is used for C tentative definitions.
	// It mostly behaves like LinkageWeakAny.
	LinkageCommon Linkage = C.LLVMGoLinkageCommon
)

// String formats the linkage as it would be printed in IR.
func (linkage Linkage) String() string {
	return linkageNames[linkage]
}

var linkageNames = [...]string{
	LinkageExternal:            "external",
	LinkageAvailableExternally: "available_externally",
	LinkageOnceAny:             "linkonce",
	LinkageOnceODR:             "linkonce_odr",
	LinkageWeakAny:             "weak",
	LinkageWeakODR:             "weak_odr",
	LinkageAppending:           "appending",
	LinkageInternal:            "internal",
	LinkagePrivate:             "private",
	LinkageExternalWeak:        "extern_weak",
	LinkageCommon:              "common",
}

// UnnamedAddr indicates whether the address of a symbol is required to be unique.
// This is used to determine whether identical symbols can be merged by the linker.
type UnnamedAddr C.LLVMUnnamedAddr

const (
	// UnnamedAddrUnique indicates that the current module depends on uniqueness of the symbol address.
	// This is the default value.
	UnnamedAddrUnique UnnamedAddr = C.LLVMNoUnnamedAddr
	// UnnamedAddrLocal indicates that the current module does not depend on uniqueness of the symbol address.
	// The symbol may be merged if this is the case for all modules that reference it.
	UnnamedAddrLocal UnnamedAddr = C.LLVMLocalUnnamedAddr
	// UnnamedAddrGlobal indicates that no module depends on uniqueness of the symbol address.
	// The symbol may be merged.
	UnnamedAddrGlobal UnnamedAddr = C.LLVMGlobalUnnamedAddr
)

// String formats the UnnamedAddr as it would be printed in IR.
func (unnamedAddr UnnamedAddr) String() string {
	return unnamedAddrNames[unnamedAddr]
}

var unnamedAddrNames = [...]string{
	UnnamedAddrUnique: "",
	UnnamedAddrLocal:  "local_unnamed_addr",
	UnnamedAddrGlobal: "unnamed_addr",
}

// Visibility controls how a symbol is treated after linking into the final executable/library.
type Visibility C.LLVMVisibility

const (
	// VisibilityDefault generally permits other executables/libraries to access the symbol.
	// Exact semantics depend on the executable format.
	// This is required by LinkageInternal/LinkagePrivate.
	VisibilityDefault Visibility = C.LLVMDefaultVisibility
	// VisibilityHidden does not permit other executables/libraries to access or override the symbol.
	VisibilityHidden Visibility = C.LLVMHiddenVisibility
	// VisibilityProtected permits other executables/libraries to access the symbol but not override it.
	VisibilityProtected Visibility = C.LLVMProtectedVisibility
)

// String formats the visibility as it would be printed in IR (excluding the quotes).
func (visibility Visibility) String() string {
	return visibilityNames[visibility]
}

var visibilityNames = [...]string{
	VisibilityDefault:   "default",
	VisibilityHidden:    "hidden",
	VisibilityProtected: "protected",
}

// DLLStorage controls imports/exports for PE/XCOFF libraries.
type DLLStorage C.LLVMDLLStorageClass

const (
	// DLLStorageDefault does not import or export the symbol.
	// This is required by LinkageInternal/LinkagePrivate.
	DLLStorageDefault DLLStorage = C.LLVMDefaultStorageClass
	// DLLStorageImport imports the symbol from a DLL.
	DLLStorageImport DLLStorage = C.LLVMDLLImportStorageClass
	// DLLStorageExports the symbol from the current DLL.
	DLLStorageExport DLLStorage = C.LLVMDLLExportStorageClass
)

// String formats the DLLStorage as it would be printed in IR.
func (storage DLLStorage) String() string {
	return dllStorageNames[storage]
}

var dllStorageNames = [...]string{
	DLLStorageDefault: "",
	DLLStorageImport:  "dllimport",
	DLLStorageExport:  "dllexport",
}

type BasicBlock struct {
	ptr C.LLVMBasicBlockRef
}

// AppendBasicBlock appends a basic block to the end of the function.
func (fn Function) AppendBasicBlock(name string) BasicBlock {
	return BasicBlock{C.LLVMGoAppendBasicBlock(fn.ptr, stringRef(name))}
}

// AddAfter adds a new basic block after bb.
func (bb BasicBlock) AddAfter(name string) BasicBlock {
	return BasicBlock{C.LLVMGoAppendBasicBlockAfter(bb.ptr, stringRef(name))}
}

func (fn Function) Param(idx uint32) Value {
	return Value{C.LLVMGetParam(fn.ptr, C.unsigned(idx))}
}
