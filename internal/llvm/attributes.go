package llvm

/*
#include "bindings.h"
*/
import "C"
import (
	"unsafe"
)

// Attribute represents a single LLVM IR attribute.
type Attribute struct {
	ptr C.LLVMAttributeRef
}

// String formats the attribute as it would be printed in IR outside of a group.
func (attr Attribute) String() string {
	var dst string
	C.LLVMGoAttributeString(unsafe.Pointer(&dst), attr.ptr)
	return dst
}

// StringAttribute creates an arbitrary string attribute from a key-value pair.
func (ctx Context) StringAttribute(key, value string) Attribute {
	return Attribute{C.LLVMGoCreateStringAttribute(
		ctx.ptr,
		stringRef(key),
		stringRef(value),
	)}
}

// StringAttribute is the key of a string pair attribute.
type StringAttribute = string

const (
	// AttributeAllocFamily is used to identify corresponding sets of allocation functions.
	AttributeAllocFamily StringAttribute = "alloc-family"

	// AttributeTargetCPU is used to specifiy the CPU type to assume when compiling a function.
	AttributeTargetCPU StringAttribute = "target-cpu"

	// AttributeTargetFeatures holds a comma-seperated list of instruction set extensions to assume when compiling a function.
	AttributeTargetFeatures StringAttribute = "target-features"
)

// EnumAttribute creates an attribute with no value.
// This panics if kind is not a valid enum attribute name.
func (ctx Context) EnumAttribute(kind EnumAttribute) Attribute {
	ptr := C.LLVMGoCreateEnumAttribute(
		ctx.ptr,
		stringRef(string(kind)),
	)
	if ptr == nil {
		panic("invalid attribute kind")
	}
	return Attribute{ptr}
}

// EnumAttribute is a name of an attribute kind with no value.
//
// This is just a string, but uses a named type to avoid accidental calls to the wrong constructor.
type EnumAttribute string

const (
	// AttributeZeroExtend indicates that an integer argument or return value must be zero-extended by the callee/caller.
	//
	// This attribute is part of the function/call signature.
	// It must be applied to both the function and the call site.
	//
	// This EnumAttribute is only valid for arguments or returns.
	AttributeZeroExtend EnumAttribute = "zeroext"
	// AttributeSignExtend indicates that an integer argument or return value must be zero-extended by the callee/caller.
	//
	// This attribute is part of the function/call signature.
	// It must be applied to both the function and the call site.
	//
	// This EnumAttribute is only valid for arguments or returns.
	AttributeSignExtend EnumAttribute = "signext"

	// AttributeNoAlias is used to constrain memory aliasing for arguments or return values.
	// It has different semantics depending on whether it is used on an argument or return value.
	//
	// When used on an argument, it indicates that during the call:
	//  - Writes to memory locations based on the argument are not visible via accesses not based on it
	//  - Accesses based on the argument may not observe writes not based on it
	//
	// When used on a return, it indicates that locations based on the result are not otherwise accessible.
	// This is mainly used for memory allocation functions.
	//
	// When used in conjunction with AttributeReadOnly, this indicates that the memory is immutable.
	//
	// This is valid for all types.
	// "based on" means any pointer derived from the value, e.g.:
	//  - ptrtoint of an integer value
	//  - array/struct/vector elements
	AttributeNoAlias EnumAttribute = "noalias"

	// AttributeNonNull poisons a pointer argument or return if it is null.
	AttributeNonNull EnumAttribute = "nonnull"

	// AttributeNoUndef produces undefined behavior if an argument or return contains undef/poison.
	//
	// This attribute may be part of the function/call signature.
	// It must be applied to both the function and the call site.
	AttributeNoUndef EnumAttribute = "noundef"

	// AttributeNoAccess indicates that the function will not access memory based on an argument.
	// The corresponding memory may still be accessed via other methods (e.g. other arguments).
	AttributeNoAccess EnumAttribute = "readnone"
	// AttributeReadOnly indicates that the function will not write memory based on an argument.
	//
	// When used in conjunction with AttributeNoAlias, this indicates that the memory is immutable.
	AttributeReadOnly EnumAttribute = "readonly"
	// AttributeWriteOnly indicates that the function will not read memory based on an argument.
	AttributeWriteOnly EnumAttribute = "writeonly"

	// AttributeAllocAlignment indicates that this argument holds the requested alignment for a memory allocation.
	AttributeAllocAlignment EnumAttribute = "allocalign"
	// AttributeAllocPointer indicates that this argument will be manipulated by the memory allocator.
	// This only has meaning when used in conjunction with AllocKindRealloc or AllocKindFree.
	AttributeAllocPointer EnumAttribute = "allocptr"

	// AttributeReturnsTwice indicates that this function may return twice.
	// This must be applied to setjmp-like functions.
	AttributeReturnsTwice EnumAttribute = "returns_twice"

	// AttributeInlineAlways indicates that the function or call should ignore inlining heuristics.
	// This does not guarantee that the function will be inlined (e.g. if inlining is disabled or impossible).
	AttributeInlineAlways EnumAttribute = "alwaysinline"
	// AttributeInlineHint is a hint that the function or call should be inlined.
	// This is weaker than AttributeInlineAlways, and the inliner may ignore it.
	// This matches the C "inline" keyword.
	AttributeInlineHint EnumAttribute = "inlinehint"
	// AttributeInlineNever prevents the function or call from being inlined.
	AttributeInlineNever EnumAttribute = "noinline"

	// AttributeCold indicates that the function/call is rarely reached.
	AttributeCold EnumAttribute = "cold"

	// AttributeOptNone disables most optimizations for a function.
	// This cannot be combined with other optimization attributes.
	AttributeOptNone EnumAttribute = "optnone"
	// AttributeOptSize prioritizes size optimizations for a function.
	AttributeOptSize EnumAttribute = "optsize"
	// AttributeMinSize requests that the size of a function be minimized, even if it will dramatically impact performance.
	AttributeOptMinSize EnumAttribute = "minsize"
	// AttributeOptDebug requests that optimizations for a function preserve debug information as much as possible.
	// This cannot be combined with other optimization attributes.
	//
	// This requires LLVM 18 or newer.
	AttributeOptDebug EnumAttribute = "optdebug"

	// AttributeNoReturn indicates that the function/call will not return normally.
	// Any code following the call is assumed dead.
	AttributeNoReturn EnumAttribute = "noreturn"
	// AttributeNoUnwind indicates that the function/call will not throw a synchronous exception.
	AttributeNoUnwind EnumAttribute = "nounwind"
)

// IntAttribute creates an attribute with an integer value.
// This panics if kind is not a valid integer attribute name.
func (ctx Context) IntAttribute(kind IntAttribute, value uint64) Attribute {
	ptr := C.LLVMGoCreateIntAttribute(
		ctx.ptr,
		stringRef(string(kind)),
		C.uint64_t(value),
	)
	if ptr == nil {
		panic("invalid attribute kind")
	}
	return Attribute{ptr}
}

// IntAttribute is a name of an attribute kind with an integer value.
//
// This is just a string, but uses a named type to avoid accidental calls to the wrong constructor.
type IntAttribute string

const (
	// AttributeAlign poisons a pointer argument or return if it lacks the specified alignment.
	// This is applied elementwise to vectors of pointers.
	AttributeAlign IntAttribute = "align"

	// AttributeDereferenceable specifies that a region at the pointer address is valid and dereferenceable.
	// The value is the size of the region in bytes.
	// This implies AttributeNoUndef and AttributeNonNull (unless the function has null_pointer_is_valid).
	//
	// This attribute is only valid for pointer arguments and returns.
	AttributeDereferenceable IntAttribute = "dereferenceable"

	// AttributeDereferenceableOrNull specifies that a pointer is null or satisfies AttributeDereferenceable.
	//
	// This is used by TinyGo to avoid escapes via null checks.
	//
	// This attribute is only valid for pointer arguments and returns.
	AttributeDereferenceableOrNull IntAttribute = "dereferenceable_or_null"

	// AttributeAllocKind holds information about how a function allocates memory.
	AttributeAllocKind IntAttribute = "allockind"

	// AttributeAllocSize describes how the size of an allocation will be calculated.
	//
	// Use EncodeAllocSize/DecodeAllocSize to create or read the value.
	AttributeAllocSize IntAttribute = "allocsize"

	// AttributeUnwindTable is used when an ABI (ELF x86_64) always requires unwind tables.
	// The value specifies what type of unwind table to generate.
	// Pass UnwindTableNone/UnwindTableSynchronous/UnwindTableAsynchronous as the value.
	AttributeUnwindTable IntAttribute = "uwtable"
)

// The encoding of allockind is somewhat fragile, but there is not much we can do about that.
// Just redefine the bit set here.
const (
	// AllocKindAlloc indicates that the returned pointer (if not null) refers to newly allocated memory.
	// This cannot be combined with AllocKindRealloc/AllocKindFree.
	AllocKindAlloc uint64 = 1 << 0
	// AllocKindRealloc indicates that the function resizes (and possibly moves) the arugment with AttributeAllocPointer.
	// If the return is null, the original argument remains valid and is not resized.
	// This cannot be combined with AllocKindAlloc/AllocKindFree.
	AllocKindRealloc uint64 = 1 << 1
	// AllocKindFree indicates that the argument with AttributeAllocPointer is freed.
	// This cannot be combined with AllocKindAlloc/AllocKindRealloc.
	AllocKindFree uint64 = 1 << 2

	// AllocKindUninitialized indicates that the new portion of the memory is uninitialized.
	AllocKindUninitialized uint64 = 1 << 3
	// AllocKindZeroed indicates that the new portion of the memory is initialized with zero bytes.
	AllocKindZeroed uint64 = 1 << 4

	// AllocKindAligned indicates that the returned pointer is aligned according to the argument with AttributeAllocAlignment.
	AllocKindAligned uint64 = 1 << 5
)

// EncodeAllocSize encodes an allocation size attribute.
//
// The elementSize is the size of an allocated element in bytes.
// The elementCountArgument is the index of the argument holding the number of allocated elements.
// These values are multiplied to compute the allocated size.
//
// The elementCountArgument may be set to AllocSizeUnknownElementCount if the element count is not an argument.
func EncodeAllocSize(elementSize, elementCountArgument uint32) uint64 {
	return (uint64(elementSize) << 32) | uint64(elementCountArgument)
}

// DecodeAllocSize decodes an allocation size attribute.
//
// The elementSize is the size of an allocated element in bytes.
// The elementCountArgument is the index of the argument holding the number of allocated elements.
// These values are multiplied to compute the allocated size.
//
// The elementCountArgument may be set to AllocSizeUnknownElementCount if the element count is not an argument.
func DecodeAllocSize(raw uint64) (elementSize, elementCountArgument uint32) {
	return uint32(raw >> 32), uint32(raw)
}

// AllocSizeOneElement indicates that the number of allocated elements is 1.
// This is used as a sentinel for the elementCount argument of AttributeAllocSize.
const AllocSizeOneElement = ^uint32(0)

const (
	// UnwindTableSynchronous creates a synchronous unwind table.
	UnwindTableSynchronous uint64 = 1

	// UnwindTableAsynchronous creates an asynchronous unwind table.
	// This permits unwinding from any instruction in the function.
	UnwindTableAsynchronous uint64 = 2
)

// TypeAttribute creates an attribute with a type value.
// This panics if kind is not a valid type attribute name.
func (ctx Context) TypeAttribute(kind TypeAttribute, value Type) Attribute {
	ptr := C.LLVMGoCreateTypeAttribute(
		ctx.ptr,
		stringRef(string(kind)),
		value.ptr,
	)
	if ptr == nil {
		panic("invalid attribute kind")
	}
	return Attribute{ptr}
}

// TypeAttribute is a name of an attribute kind with an type value.
//
// This is just a string, but uses a named type to avoid accidental calls to the wrong constructor.
type TypeAttribute string

const (
	// AttributeByVal is used to pass large arguments through stack memory.
	// The backend will copy the contents to a new stack variable at the call site.
	// The callee will recieve the pointer to the copied variable - not the pointer passed at the call site.
	// As such, the callee can write to the pointer without the caller observing any changes.
	//
	// This is used in some C ABIs to allow the caller to allocate and initialize argument variables.
	// This can be more efficient than copying inside the callee.
	//
	// The provided type is used for the new stack variable and the copy used to initialize it.
	// The allocation and copy alignment are specified by an AttributeAlign on the argument.
	//
	// This attribute is part of the function/call signature.
	// It must be applied to both the function and the call site.
	AttributeByVal TypeAttribute = "byval"

	// AttributeByRef is used to pass large arguments without a copy.
	// This implies AttributeDereferencable with the type's size.
	//
	// This attribute is part of the function/call signature.
	// It must be applied to both the function and the call site.
	//
	// TODO: this is poorly documented
	AttributeByRef TypeAttribute = "byref"

	// AttributeStackReturn is used to return large values through stack memory.
	// This implies AttributeDereferencable with the type's size.
	//
	// This attribute is part of the function/call signature.
	// It must be applied to both the function and the call site.
	//
	// This attribute is only valid for pointer arguments.
	// The associated call/function must return void.
	AttributeStackReturn TypeAttribute = "sret"
)

// RangeAttribute creates an attribute with a numeric range value.
//
// The lower/upper values are specified the same way as by ConstantInt.
//
// The lower bound is inclusive and the upper bound is exclusive.
// The range may wrap if upper is less than lower.
// The range [0, 0) is used to represent an empty range.
// The lower and upper bounds may not be equal otherwise.
//
// This panics if kind is not a valid range attribute name.
func (ctx Context) RangeAttribute(kind RangeAttribute, bits uint32, lower []uint64, upper []uint64) Attribute {
	ptr := C.LLVMGoCreateRangeAttribute(
		ctx.ptr,
		stringRef(string(kind)),
		intWidth(bits),
		(*C.uint64_t)(unsafe.Pointer(unsafe.SliceData(lower))),
		C.size_t(len(lower)),
		(*C.uint64_t)(unsafe.Pointer(unsafe.SliceData(upper))),
		C.size_t(len(upper)),
	)
	if ptr == nil {
		panic("invalid attribute kind")
	}
	return Attribute{ptr}
}

// TypeAttribute is a name of an attribute kind with an integer range value.
//
// This is just a string, but uses a named type to avoid accidental calls to the wrong constructor.
type RangeAttribute string

const (
	// AttributeRange posions an argument or return if it is not within the specified numeric range.
	// This is applied elementwise to vectors.
	// The type of the range must match the value or element type.
	//
	// This requires LLVM 19 or newer.
	// Use SupportsAttributeRange to check if this feature is available.
	AttributeRange RangeAttribute = "range"
)

// SupportsAttributeRange indicates whether the current LLVM version supports range attributes.
// https://github.com/llvm/llvm-project/pull/83171
const SupportsAttributeRange = VersionMajor >= 19

// Kind checks whether the attribute is a string and reads the kind/key.
func (attr Attribute) Kind() (kind string, isString bool) {
	var rawKind C.LLVMGoStringRef
	isString = bool(C.LLVMGoAttributeKind(attr.ptr, &rawKind))
	return refAsString(rawKind), isString
}

// StringValue reads the value of a string attribute.
// This will panic if the attribute is not a string.
// The returned string memory is owned by the context.
func (attr Attribute) StringValue() string {
	var dst C.LLVMGoStringRef
	ok := C.LLVMGoAttributeStringValue(attr.ptr, &dst)
	if !ok {
		panic("attribute is not a string")
	}
	return refAsString(dst)
}

// IntValue reads the value of an integer attribute.
// This will panic if the attribute is not an int.
func (attr Attribute) IntValue() uint64 {
	var dst C.uint64_t
	ok := C.LLVMGoAttributeIntValue(attr.ptr, &dst)
	if !ok {
		panic("attribute is not an int")
	}
	return uint64(dst)
}

// TypeValue reads the value of a type attribute.
// This will panic if the attribute is not a type.
func (attr Attribute) TypeValue() Type {
	ptr := C.LLVMGoAttributeTypeValue(attr.ptr)
	if ptr == nil {
		panic("attribute is not a type")
	}
	return Type{ptr}
}

// RangeValue reads the value of a range attribute.
// This will panic if the attribute is not a range.
// The returned slices are owned by the context and must not be modified.
func (attr Attribute) RangeValue() (bits uint32, lower, upper []uint64) {
	val := C.LLVMGoAttributeRangeValue(attr.ptr)
	if val.bits == 0 {
		panic("attribute is not a range")
	}
	return unpackRange(val)
}

func unpackRange(val C.LLVMGoConstRange) (bits uint32, lower, upper []uint64) {
	// The bits value is <= 2^29, but the compiler does not know that.
	// Temporarily widen to 64 bits to prove that it will not overflow.
	words := (uint64(val.bits) + 63) / 64
	return uint32(val.bits), castU64Slice(val.lower, words), castU64Slice(val.upper, words)
}

//go:nobounds
func castU64Slice(ptr *C.uint64_t, len uint64) []uint64 {
	return unsafe.Slice((*uint64)(unsafe.Pointer(ptr)), len)
}

// AttributeSet holds a set of attributes.
// The zero value is the empty set.
// Sets from the same context can be compared for equality.
type AttributeSet struct {
	ptr C.LLVMGoAttributeSetRef
}

// String formats the set as it would be printed in IR outside of a group.
// An empty set becomes an empty string.
func (set AttributeSet) String() string {
	if set == (AttributeSet{}) {
		return ""
	}
	var dst string
	C.LLVMGoAttributeSetString(unsafe.Pointer(&dst), set.ptr)
	return dst
}

// AttributeSet combines the provided attributes into a set.
// If multiple attributes of the same kind are provided, only the last value will be used.
func (ctx Context) AttributeSet(attrs ...Attribute) AttributeSet {
	if len(attrs) == 0 {
		return AttributeSet{}
	}
	return AttributeSet{C.LLVMGoAttributeSetCreate(
		ctx.ptr,
		(*C.LLVMAttributeRef)(unsafe.Pointer(unsafe.SliceData(attrs))),
		C.size_t(len(attrs)),
	)}
}

// GetStringAttr finds a string attribute in the set.
// Use GetString instead to get the value.
func (set AttributeSet) GetStringAttr(key string) (Attribute, bool) {
	if set == (AttributeSet{}) {
		return Attribute{}, false
	}
	ptr := C.LLVMGoAttributeSetGetString(set.ptr, stringRef(key))
	return Attribute{ptr}, ptr != nil
}

// GetString gets the value of a string attribute if it is present in the set.
func (set AttributeSet) GetString(key string) (string, bool) {
	if set == (AttributeSet{}) {
		return "", false
	}
	var dst C.LLVMGoStringRef
	ok := C.LLVMGoAttributeSetGetStringValue(set.ptr, stringRef(key), &dst)
	return refAsString(dst), bool(ok)
}

// Get a non-string attribute if it is in the set.
//
// Use HasEnum/GetInt/GetType/GetRange instead to get the value.
// Only use this if you actually want the attribute reference.
//
// This will return false if attr is not a valid attribute name.
func (set AttributeSet) Get(key string) (Attribute, bool) {
	if set == (AttributeSet{}) {
		return Attribute{}, false
	}
	ptr := C.LLVMGoAttributeSetGet(set.ptr, stringRef(key))
	return Attribute{ptr}, ptr != nil
}

// HasEnum checks if an enum attribute is present in the set.
//
// This will return false if attr is not a valid enum attribute name.
func (set AttributeSet) HasEnum(attr EnumAttribute) bool {
	if set == (AttributeSet{}) {
		return false
	}
	return bool(C.LLVMGoAttributeSetHasEnum(set.ptr, stringRef(string(attr))))
}

// GetInt gets the value of an integer attribute if it is present in the set.
//
// This will return false if attr is not a valid integer attribute name.
func (set AttributeSet) GetInt(attr IntAttribute) (uint64, bool) {
	if set == (AttributeSet{}) {
		return 0, false
	}
	var dst C.uint64_t
	ok := C.LLVMGoAttributeSetGetInt(set.ptr, stringRef(string(attr)), &dst)
	return uint64(dst), bool(ok)
}

// GetType gets the value of a type attribute if it is present in the set.
//
// This will return false if attr is not a valid type attribute name.
func (set AttributeSet) GetType(attr TypeAttribute) (Type, bool) {
	if set == (AttributeSet{}) {
		return Type{}, false
	}
	ptr := C.LLVMGoAttributeSetGetType(set.ptr, stringRef(string(attr)))
	return Type{ptr}, ptr != nil
}

// GetRange gets the value of a range attribute if it is present in the set.
//
// The returned slices are owned by the context and must not be modified.
//
// This will return false if attr is not a valid range attribute name.
func (set AttributeSet) GetRange(attr RangeAttribute) (bits uint32, lower, upper []uint64, ok bool) {
	if set == (AttributeSet{}) {
		return 0, nil, nil, false
	}
	val := C.LLVMGoAttributeSetGetRange(set.ptr, stringRef(string(attr)))
	bits, lower, upper = unpackRange(val)
	return bits, lower, upper, bits != 0
}

// MergeAttributeSets combines the attributes from each set into a single set.
// If no sets are provided, this returns the empty set.
// If multiple sets contain attributes of the same kind, only the last value will be used.
func (ctx Context) MergeAttributeSets(sets ...AttributeSet) AttributeSet {
	switch len(sets) {
	case 0:
		return AttributeSet{}
	case 1:
		return sets[0]
	}
	return AttributeSet{C.LLVMGoAttributeSetMerge(
		ctx.ptr,
		(*C.LLVMGoAttributeSetRef)(unsafe.Pointer(unsafe.SliceData(sets))),
		C.size_t(len(sets)),
	)}
}

// IntersectAttributeSets isolates common attributes from the provided sets.
// If the sets contain conflicting attributes, this will return the empty set and false.
func (ctx Context) IntersectAttributeSets(first AttributeSet, more ...AttributeSet) (set AttributeSet, ok bool) {
	if len(more) == 0 {
		return first, true
	}
	res := C.LLVMGoAttributeSetIntersect(
		ctx.ptr,
		first.ptr,
		(*C.LLVMGoAttributeSetRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)
	return AttributeSet{res.result}, bool(res.ok)
}

// CaptureAttributes creates an attribute set with the provided constraints.
func (ctx Context) CaptureAttributes(info CaptureInfo) AttributeSet {
	if info == (CaptureInfo{}) {
		// This is the default capture info.
		return AttributeSet{}
	}
	return AttributeSet{C.LLVMGoCreateCaptureAttributes(ctx.ptr, info.toC())}
}

// CaptureInfo inspects capture-related attributes from a set.
func (set AttributeSet) CaptureInfo() CaptureInfo {
	return captureInfoFromC(C.LLVMGoGetCaptureInfo(set.ptr))
}

// CaptureInfo holds information on how a function argument may capture pointers.
type CaptureInfo struct {
	// Other indicates how the argument may be captured through means other than the return.
	Other CaptureComponents

	// Return indicates how the argument may be captured through the function's return.
	Return CaptureComponents
}

// String formats the capture info as it would be printed in IR.
func (info CaptureInfo) String() string {
	switch {
	case info.Return == info.Other:
		return info.Other.String()
	case info.Other.Address == AddressCaptureNone && info.Other.Provenance == ProvenanceCaptureNone:
		return "ret: " + info.Return.String()
	default:
		return info.Other.String() + ", ret: " + info.Return.String()
	}
}

func captureInfoFromC(info C.LLVMGoCaptureInfo) CaptureInfo {
	return CaptureInfo{
		Other:  captureComponentsFromC(info.other),
		Return: captureComponentsFromC(info.returned),
	}
}

func (info CaptureInfo) toC() C.LLVMGoCaptureInfo {
	return C.LLVMGoCaptureInfo{
		other:    info.Other.toC(),
		returned: info.Return.toC(),
	}
}

// CaptureComponents holds capture information for a single scope.
type CaptureComponents struct {
	// Address specifies how the address identity will be captured.
	Address AddressCapture

	// Provenance specifies how the pointer will be captured for future access.
	Provenance ProvenanceCapture
}

// String formats the capture components as they would be printed in IR.
func (components CaptureComponents) String() string {
	switch {
	case components.Address == AddressCaptureNone:
		return components.Provenance.String()
	case components.Provenance == ProvenanceCaptureNone:
		return components.Address.String()
	default:
		return components.Address.String() + ", " + components.Provenance.String()
	}
}

func captureComponentsFromC(info C.LLVMGoCaptureComponents) CaptureComponents {
	return CaptureComponents{
		Address:    AddressCapture(info.address),
		Provenance: ProvenanceCapture(info.provenance),
	}
}

func (components CaptureComponents) toC() C.LLVMGoCaptureComponents {
	return C.LLVMGoCaptureComponents{
		address:    C.LLVMGoCaptureAddress(components.Address),
		provenance: C.LLVMGoCaptureProvenance(components.Provenance),
	}
}

// AddressCapture specifies how a function will capture the identity of an argument.
type AddressCapture C.LLVMGoCaptureAddress

const (
	// AddressCaptureFull captures the identity of the pointer.
	// This is the default AddressCapture.
	AddressCaptureFull AddressCapture = C.LLVMGoCaptureAddressFull

	// AddressCaptureIsNull captures whether the pointer is null.
	AddressCaptureIsNull AddressCapture = C.LLVMGoCaptureAddressIsNull

	// AddressCaptureNone does not capture the identity of the pointer.
	AddressCaptureNone AddressCapture = C.LLVMGoCaptureAddressNone
)

// String formats the AddressCapture as it would be printed in IR.
func (ac AddressCapture) String() string {
	return addressCaptureNames[ac]
}

var addressCaptureNames = [...]string{
	AddressCaptureFull:   "address",
	AddressCaptureIsNull: "address_is_null",
	AddressCaptureNone:   "none",
}

// ProvenanceCapture specifies how memory may be accessed from an argument after the callee returns.
type ProvenanceCapture C.LLVMGoCaptureProvenance

const (
	// ProvenanceCaptureFull captures the pointer such that it may be read from or written to after the callee returns.
	// This is the default ProvenanceCapture.
	ProvenanceCaptureFull ProvenanceCapture = C.LLVMGoCaptureProvenanceFull

	// ProvenanceCaptureFull captures the pointer such that it may be read from after the callee returns.
	ProvenanceCaptureReadOnly ProvenanceCapture = C.LLVMGoCaptureProvenanceReadOnly

	// ProvenanceCapture indicates that the pointer will not be accessed after the callee returns.
	ProvenanceCaptureNone ProvenanceCapture = C.LLVMGoCaptureProvenanceNone
)

// String formats the ProvenanceCapture as it would be printed in IR.
func (pc ProvenanceCapture) String() string {
	return provenanceCaptureNames[pc]
}

var provenanceCaptureNames = [...]string{
	ProvenanceCaptureFull:     "provenance",
	ProvenanceCaptureReadOnly: "read_provenance",
	ProvenanceCaptureNone:     "none",
}

// MemoryEffects creates a function attribute set from the memory effects.
func (ctx Context) MemoryEffects(effects MemoryEffects) AttributeSet {
	if effects == (MemoryEffects{}) {
		return AttributeSet{}
	}
	return AttributeSet{C.LLVMGoCreateMemoryEffectsAttributes(ctx.ptr, effects.flags)}
}

// MemoryEffects inspects memory effects attributes from a function attribute set.
func (set AttributeSet) MemoryEffects() MemoryEffects {
	if set == (AttributeSet{}) {
		return MemoryEffects{}
	}
	return MemoryEffects{C.LLVMGoGetMemoryEffects(set.ptr)}
}

// MemoryEffects tracks the means by which a function may access memory.
// This behaves like a map[MemoryLocation]MemoryAccess.
type MemoryEffects struct {
	flags C.LLVMGoMemoryEffectsMask
}

// String formats the MemoryEffects as they would be printed in IR by LLVM 21.
func (effects MemoryEffects) String() string {
	// Check if all locations are the same.
	other := effects.Get(MemoryLocationOther)
	otherAll := other.All()
	if effects == otherAll {
		return other.String()
	}

	// Format the differing locations.
	var buf = make([]byte, 0, 64)
	var needsComma bool
	if other != MemoryAccessNone {
		buf = append(buf, other.String()...)
		needsComma = true
	}
	for location := MemoryLocationOther + 1; location <= MemoryLocationLast; location++ {
		flags := effects.Get(location)
		if flags == other {
			continue
		}
		if needsComma {
			buf = append(buf, ", "...)
		}
		buf = append(buf, location.String()...)
		buf = append(buf, ": "...)
		buf = append(buf, flags.String()...)
		needsComma = true
	}
	return string(buf)
}

// Get the memory access flags for a location.
func (effects MemoryEffects) Get(location MemoryLocation) MemoryAccess {
	return MemoryAccess(effects.flags>>location.shift()) & MemoryAccessNone
}

// Intersect combines two MemoryEffects constraints for the same operation.
func (effects MemoryEffects) Intersect(other MemoryEffects) MemoryEffects {
	return MemoryEffects{effects.flags | other.flags}
}

// Union combines two MemoryEffects constraints from separate operations.
func (effects MemoryEffects) Union(other MemoryEffects) MemoryEffects {
	return MemoryEffects{effects.flags & other.flags}
}

// Satisfies checks if the provided effects satisfy the provided constraints.
// For example effects.Satisfies(MemoryAccessNoRead.All()) checks if no memory will be read.
func (effects MemoryEffects) Satisfies(constraints MemoryEffects) bool {
	return constraints.flags&^effects.flags != 0
}

// MemoryAccess is a set of flags used to specify whether reads or writes to memory are possible.
// The zero value implies that both are possible.
type MemoryAccess C.LLVMGoMemoryAccessFlags

const (
	// MemoryAccessAny permits reads and writes.
	MemoryAccessAny MemoryAccess = C.LLVMGoMemoryAccessAny

	// MemoryAccessNoRead indicates that no memory reads are possible.
	MemoryAccessNoRead MemoryAccess = C.LLVMGoMemoryAccessNoRead

	// MemoryAccessNoWrite indicates that no memory writes are possible.
	MemoryAccessNoWrite MemoryAccess = C.LLVMGoMemoryAccessNoWrite

	// MemoryAccessNone indicates that no memory will be read or written.
	MemoryAccessNone MemoryAccess = MemoryAccessNoRead | MemoryAccessNoWrite
)

func (access MemoryAccess) String() string {
	return memoryAccessNames[access]
}

var memoryAccessNames = [...]string{
	MemoryAccessAny:     "readwrite",
	MemoryAccessNoRead:  "write",
	MemoryAccessNoWrite: "read",
	MemoryAccessNone:    "none",
}

// All applies the MemoryAccess constraints to all locations.
func (access MemoryAccess) All() MemoryEffects {
	return MemoryEffects{C.LLVMGoMemoryEffectsMask(access) * C.LLVMGoMemoryEffectsBroadcast}
}

// Only creates MemoryEffects for an access to a specific location only.
func (access MemoryAccess) Only(location MemoryLocation) MemoryEffects {
	return access.At(location).Intersect(location.Only())
}

// At creates MemoryEffects which may only access a location in a specific manner.
// All other locations are unconstrained.
func (access MemoryAccess) At(location MemoryLocation) MemoryEffects {
	return MemoryEffects{C.LLVMGoMemoryEffectsMask(access) << location.shift()}
}

// MemoryLocation is used to specify what memory to apply a MemoryAccess constraint to.
type MemoryLocation C.LLVMGoMemoryLocation

const (
	// MemoryLocationOther is used for memory locations which lack a specialization.
	MemoryLocationOther MemoryLocation = C.LLVMGoMemoryLocationOther
	// MemoryLocationArguments is used to constrain how the function will access memory through its arguments.
	MemoryLocationArguments MemoryLocation = C.LLVMGoMemoryLocationArguments
	// MemoryLocationInaccessible is used to constrain how the function will access memory outside the current module.
	MemoryLocationInaccessible MemoryLocation = C.LLVMGoMemoryLocationInaccessible
	// MemoryLocationErrno is used to constrain how the function will access the C errno value.
	// This location was added in LLVM 21 and is merged to/from MemoryLocationOther for prior versions.
	MemoryLocationErrno MemoryLocation = C.LLVMGoMemoryLocationErrno

	// MemoryLocationLast is the highest-index memory location.
	// This is useful for looping over locations.
	MemoryLocationLast = MemoryLocationErrno
)

func (location MemoryLocation) String() string {
	return memoryLocationNames[location]
}

var memoryLocationNames = [...]string{
	MemoryLocationOther:        "other",
	MemoryLocationArguments:    "argmem",
	MemoryLocationInaccessible: "inaccessiblemem",
	MemoryLocationErrno:        "errnomem",
}

// Only creates MemoryEffects that may only access this location.
func (location MemoryLocation) Only() MemoryEffects {
	return MemoryEffects{MemoryAccessNone.All().flags &^ MemoryAccessNone.At(location).flags}
}

func (location MemoryLocation) shift() uint {
	return uint(location) * C.LLVMGoMemoryAccessFlagsBits
}

// AttributeList holds attribute sets for a function or call.
// The zero value represents an empty list.
type AttributeList struct {
	ptr C.LLVMGoAttributeListRef
}

// AttributeList creates an attribute list from the provided sets.
func (ctx Context) AttributeList(
	functionAttributes AttributeSet,
	returnAttributes AttributeSet,
	argumentAttributes ...AttributeSet,
) AttributeList {
	if len(argumentAttributes) > (1<<32)-2 {
		// TODO: verify that this is the correct limit
		panic("too many arguments")
	}
	return AttributeList{C.LLVMGoAttributeListCreate(
		ctx.ptr,
		functionAttributes.ptr,
		returnAttributes.ptr,
		(*C.LLVMGoAttributeSetRef)(unsafe.Pointer(unsafe.SliceData(argumentAttributes))),
		C.unsigned(len(argumentAttributes)),
	)}
}

// String formats the AttributeList.
func (list AttributeList) String() string {
	// LLVM's builtin formatting for an attribute does not mix with other formatting (newlines, not very readable).
	// Format it ourselves, vaguely matching the layout of a function declaration.
	if list == (AttributeList{}) {
		return "@()"
	}
	var dst = make([]byte, 0, 64)
	if ra := list.Return(); ra != (AttributeSet{}) {
		dst = append(append(dst, ra.String()...), ' ')
	}
	dst = append(dst, "@("...)
	args := list.Arguments()
	if args > 0 {
		dst = append(dst, list.Argument(0).String()...)
		for i := uint32(1); i < args; i++ {
			dst = append(append(dst, ", "...), list.Argument(1).String()...)
		}
	}
	dst = append(dst, ')')
	if fa := list.Function(); fa != (AttributeSet{}) {
		dst = append(append(dst, ' '), fa.String()...)
	}
	return string(dst)
}

// Return gets the attribute set attached to the function return.
func (list AttributeList) Return() AttributeSet {
	if list == (AttributeList{}) {
		return AttributeSet{}
	}
	return AttributeSet{C.LLVMGoAttibuteListGetReturn(list.ptr)}
}

// Function gets the attribute set attached to the function itself.
func (list AttributeList) Function() AttributeSet {
	if list == (AttributeList{}) {
		return AttributeSet{}
	}
	return AttributeSet{C.LLVMGoAttibuteListGetFunction(list.ptr)}
}

// Argument gets the attribute set attached to an argument.
func (list AttributeList) Argument(argument uint32) AttributeSet {
	if list == (AttributeList{}) {
		return AttributeSet{}
	}
	return AttributeSet{C.LLVMGoAttibuteListGetArgument(list.ptr, C.unsigned(argument))}
}

func (list AttributeList) Arguments() uint32 {
	return uint32(C.LLVMGoAttributeListArguments(list.ptr))
}
