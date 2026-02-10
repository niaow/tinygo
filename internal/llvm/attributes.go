package llvm

/*
#include "bindings.h"
*/
import "C"
import "unsafe"

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

	// AttributeReadNone indicates that the function will not access memory based on an argument.
	// Contrary to the name, this also disallows writes.
	// The corresponding memory may still be accessed via other methods (e.g. other arguments).
	AttributeReadNone EnumAttribute = "readnone"

	// AttributeReadOnly indicates that the function will not write memory based on an argument.
	//
	// When used in conjunction with AttributeNoAlias, this indicates that the memory is immutable.
	AttributeReadOnly EnumAttribute = "readonly"

	// AttributeWriteOnly indicates that the function will not read memory based on an argument.
	AttributeWriteOnly EnumAttribute = "writeonly"
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
const SupportsAttributeRange = C.LLVM_VERSION_MAJOR >= 19

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
	if set.ptr == nil {
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

type CaptureComponents struct {
	Address    AddressCapture
	Provenance ProvenanceCapture
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

type AddressCapture C.LLVMGoCaptureAddress

type ProvenanceCapture C.LLVMGoCaptureProvenance
