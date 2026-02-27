package llvm

/*
#include "bindings.h"
*/
import "C"
import (
	"unsafe"
)

// TODO
// TODO: describe names
// TOOD: describe folding
type Builder struct {
	ptr C.LLVMBuilderRef
}

// Builder creates a builder without a defined starting position.
func (ctx Context) Builder() Builder {
	return Builder{C.LLVMCreateBuilderInContext(ctx.ptr)}
}

func (b Builder) Destroy() {
	C.LLVMDisposeBuilder(b.ptr)
}

// AtEnd moves the insertion point to the end of the basic block.
func (b Builder) AtEnd(bb BasicBlock) {
	C.LLVMPositionBuilderAtEnd(b.ptr, bb.ptr)
}

// Integer conversion operations

// Truncate converts an integer to a smaller type by discarding the leading bits.
// This operation is performed elementwise on vectors.
//
// If unsignedExact ("nuw" in IR): the result is poisoned if any nonzero bits are discarded.
// If signedExact ("nsw" in IR): the result is poisoned if any discarded bits disagree with the sign bit of the result.
// When combined ("nuw nsw" in IR), the result is implied to be positive.
//
// https://llvm.org/docs/LangRef.html#trunc-to-instruction
func (b Builder) Truncate(v Value, to Type, name string, unsignedExact, signedExact bool) Value {
	return Value{C.LLVMGoCreateTrunc(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
		C.bool(unsignedExact),
		C.bool(signedExact),
	)}
}

// ZeroExtend converts an integer to a larger type by adding leading zero bits.
// This operation is performed elementwise on vectors.
//
// If positive ("nneg" in IR): the result is poisoned if the operand's top bit is set.
// This implies that the conversion is equivalent to a SignExtend.
// It is ignored until LLVM 18 (https://github.com/llvm/llvm-project/pull/67982).
//
// https://llvm.org/docs/LangRef.html#zext-to-instruction
func (b Builder) ZeroExtend(v Value, to Type, name string, positive bool) Value {
	return Value{C.LLVMGoCreateZExt(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
		C.bool(positive),
	)}
}

// SignExtend converts an integer to a larger type by repeating the sign bit.
// This operation is performed elementwise on vectors.
//
// https://llvm.org/docs/LangRef.html#sext-to-instruction
func (b Builder) SignExtend(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateSExt(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// Integer arithmetic

// Add creates a (wrapping) integer addition instruction.
// The result type matches the operand types.
// If noUnsignedWrap ("nuw" in IR): results with unsigned overflow are poisoned.
// If noSignedWrap ("nsw" in IR): results with signed overflow are poisoned.
//
// https://llvm.org/docs/LangRef.html#add-instruction
func (b Builder) Add(x, y Value, name string, noUnsignedWrap, noSignedWrap bool) Value {
	return Value{C.LLVMGoCreateAdd(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
		C.bool(noUnsignedWrap),
		C.bool(noSignedWrap),
	)}
}

// Subtract creates a (wrapping) integer subtraction instruction.
// The result type matches the operand types.
// If noUnsignedWrap ("nuw" in IR): results with unsigned overflow are poisoned.
// If noSignedWrap ("nsw" in IR): results with signed overflow are poisoned.
//
// https://llvm.org/docs/LangRef.html#sub-instruction
func (b Builder) Subtract(minuend, subtrahend Value, name string, noUnsignedWrap, noSignedWrap bool) Value {
	return Value{C.LLVMGoCreateSub(
		b.ptr,
		minuend.ptr,
		subtrahend.ptr,
		stringRef(name),
		C.bool(noUnsignedWrap),
		C.bool(noSignedWrap),
	)}
}

// Multiply creates a (wrapping) integer multiplication instruction.
// The result type matches the operand types.
// If noUnsignedWrap ("nuw" in IR): results with unsigned overflow are poisoned.
// If noSignedWrap ("nsw" in IR): results with signed overflow are poisoned.
//
// https://llvm.org/docs/LangRef.html#mul-instruction
func (b Builder) Multiply(x, y Value, name string, noUnsignedWrap, noSignedWrap bool) Value {
	return Value{C.LLVMGoCreateMul(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
		C.bool(noUnsignedWrap),
		C.bool(noSignedWrap),
	)}
}

// UnsignedDivide creates an unsigned integer divison instruction.
// The result type matches the operand types.
// The divisor must be nonzero.
// If exact ("exact" in IR): results with nonzero remainder are poisoned.
//
// https://llvm.org/docs/LangRef.html#udiv-instruction
func (b Builder) UnsignedDivide(dividend, divisor Value, name string, exact bool) Value {
	return Value{C.LLVMGoCreateUDiv(
		b.ptr,
		dividend.ptr,
		divisor.ptr,
		stringRef(name),
		C.bool(exact),
	)}
}

// SignedDivide creates a signed integer divison instruction.
// The result type matches the operand types.
// The divisor must be nonzero, and the result must not overflow.
// If exact ("exact" in IR): results with nonzero remainder are poisoned.
//
// https://llvm.org/docs/LangRef.html#sdiv-instruction
func (b Builder) SignedDivide(dividend, divisor Value, name string, exact bool) Value {
	return Value{C.LLVMGoCreateSDiv(
		b.ptr,
		dividend.ptr,
		divisor.ptr,
		stringRef(name),
		C.bool(exact),
	)}
}

// UnsignedRemainder creates an unsigned integer remainder instruction.
// The result type matches the operand types.
// The divisor must be nonzero.
//
// https://llvm.org/docs/LangRef.html#urem-instruction
func (b Builder) UnsignedRemainder(dividend, divisor Value, name string) Value {
	return Value{C.LLVMGoCreateURem(
		b.ptr,
		dividend.ptr,
		divisor.ptr,
		stringRef(name),
	)}
}

// SignedRemainder creates a signed integer remainder instruction.
// The result type matches the operand types.
// The divisor must be nonzero, and the corresponding divide must not overflow.
//
// https://llvm.org/docs/LangRef.html#srem-instruction
func (b Builder) SignedRemainder(dividend, divisor Value, name string) Value {
	return Value{C.LLVMGoCreateSRem(
		b.ptr,
		dividend.ptr,
		divisor.ptr,
		stringRef(name),
	)}
}

// Compare creates an integer comparion instruction.
// This is also valid on pointers, which are compared by address.
// This operation is performed elementwise on vectors.
//
// TODO: add the samesign flag.
// LLVM 20 introduced a flag called samesign which is used to turn unsigned comparisons into signed comparions.
// This is potentially useful to us, but it is not exposed through the builder API.
// It is difficult to set after insertion because it could be folded (and the folder is private).
//
// https://llvm.org/docs/LangRef.html#icmp-instruction
func (b Builder) Compare(cmp IntComparison, lhs, rhs Value, name string) Value {
	return Value{C.LLVMGoCreateICmp(
		b.ptr,
		C.LLVMGoIntComparison(cmp),
		lhs.ptr,
		rhs.ptr,
		stringRef(name),
	)}
}

// IntComparison is an integer comparion operator.
// The signedness is considered part of the comparison.
type IntComparison C.LLVMGoIntComparison

const (
	// IntEqual is the == operator.
	// Signedness is irrelevant when comparing for equality.
	IntEqual IntComparison = C.LLVMGoIntEQ

	// IntNotEqual is the != operator.
	// Signedness is irrelevant when comparing for equality.
	IntNotEqual IntComparison = C.LLVMGoIntNE

	// IntUnsignedGreaterThan is the > operator for unsigned integers or pointers.
	IntUnsignedGreaterThan IntComparison = C.LLVMGoIntUGT

	// IntUnsignedGreaterOrEqual is the >= operator for unsigned integers or pointers.
	IntUnsignedGreaterOrEqual IntComparison = C.LLVMGoIntUGE

	// IntUnsignedLessThan is the < operator for unsigned integers or pointers.
	IntUnsignedLessThan IntComparison = C.LLVMGoIntULT

	// IntUnsignedLessOrEqual is the <= operator for unsigned integers or pointers.
	IntUnsignedLessOrEqual IntComparison = C.LLVMGoIntULE

	// IntSignedGreaterThan is the > operator for signed integers.
	IntSignedGreaterThan IntComparison = C.LLVMGoIntSGT

	// IntSignedGreaterOrEqual is the >= operator for signed integers.
	IntSignedGreaterOrEqual IntComparison = C.LLVMGoIntSGE

	// IntSignedLessThan is the < operator for signed integers.
	IntSignedLessThan IntComparison = C.LLVMGoIntSLT

	// IntSignedLessOrEqual is the <= operator for signed integers.
	IntSignedLessOrEqual IntComparison = C.LLVMGoIntSLE
)

// String formats the comparison operator as it would be printed in IR.
func (icmp IntComparison) String() string {
	return intComparisonNames[icmp]
}

var intComparisonNames = [...]string{
	IntEqual:                  "eq",
	IntNotEqual:               "ne",
	IntUnsignedGreaterThan:    "ugt",
	IntUnsignedGreaterOrEqual: "uge",
	IntUnsignedLessThan:       "ult",
	IntUnsignedLessOrEqual:    "ule",
	IntSignedGreaterThan:      "sgt",
	IntSignedGreaterOrEqual:   "sge",
	IntSignedLessThan:         "slt",
	IntSignedLessOrEqual:      "sle",
}

// UnsignedMinimum finds the minimum of the provided unsigned integers.
// The result type matches the operand types.
//
// https://llvm.org/docs/LangRef.html#llvm-umin-intrinsic
func (b Builder) UnsignedMinimum(name string, first Value, more ...Value) Value {
	return Value{C.LLVMGoCreateUMin(
		b.ptr,
		stringRef(name),
		first.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)}
}

// SignedMinimum finds the minimum of the provided signed integers.
// The result type matches the operand types.
//
// https://llvm.org/docs/LangRef.html#llvm-smin-intrinsic
func (b Builder) SignedMinimum(name string, first Value, more ...Value) Value {
	return Value{C.LLVMGoCreateSMin(
		b.ptr,
		stringRef(name),
		first.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)}
}

// UnsignedMaxiumum finds the maximum of the provided unsigned integers.
// The result type matches the operand types.
//
// https://llvm.org/docs/LangRef.html#llvm-umax-intrinsic
func (b Builder) UnsignedMaximum(name string, first Value, more ...Value) Value {
	return Value{C.LLVMGoCreateUMax(
		b.ptr,
		stringRef(name),
		first.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)}
}

// SignedMaxiumum finds the maximum of the provided signed integers.
// The result type matches the operand types.
//
// https://llvm.org/docs/LangRef.html#llvm-smax-intrinsic
func (b Builder) SignedMaximum(name string, first Value, more ...Value) Value {
	return Value{C.LLVMGoCreateSMax(
		b.ptr,
		stringRef(name),
		first.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)}
}

// Bitwise operations

// BitShiftLeft multiplies integer src by 2^by.
// The result type matches the operand types.
// The result is poisoned if by is greater than the source bit width.
// If noUnsignedWrap ("nuw" in IR): results with unsigned overflow are poisoned.
// If noSignedWrap ("nsw" in IR): results with signed overflow are poisoned.
//
// https://llvm.org/docs/LangRef.html#shl-instruction
func (b Builder) BitShiftLeft(src, by Value, name string, noUnsignedWrap, noSignedWrap bool) Value {
	return Value{C.LLVMGoCreateShl(
		b.ptr,
		src.ptr,
		by.ptr,
		stringRef(name),
		C.bool(noUnsignedWrap),
		C.bool(noSignedWrap),
	)}
}

// LogicalShiftRight divides unsigned integer src by 2^by.
// The result type matches the operand types.
// The result is poisoned if by is greater than the source bit width.
// If exact ("exact" in IR): results with nonzero remainder are poisoned.
//
// https://llvm.org/docs/LangRef.html#lshr-instruction
func (b Builder) LogicalShiftRight(src, by Value, name string, exact bool) Value {
	return Value{C.LLVMGoCreateLShr(
		b.ptr,
		src.ptr,
		by.ptr,
		stringRef(name),
		C.bool(exact),
	)}
}

// ArithmeticShiftRight divides signed integer src by 2^by.
// The result type matches the operand types.
// The result is poisoned if by is greater than the source bit width.
// If exact ("exact" in IR): results with nonzero remainder are poisoned.
//
// https://llvm.org/docs/LangRef.html#ashr-instruction
func (b Builder) ArithmeticShiftRight(src, by Value, name string, exact bool) Value {
	return Value{C.LLVMGoCreateAShr(
		b.ptr,
		src.ptr,
		by.ptr,
		stringRef(name),
		C.bool(exact),
	)}
}

// And creates a bitwise and instruction.
// The result type matches the operand types.
// This is equivalent to a boolean and when the type is i1.
//
// https://llvm.org/docs/LangRef.html#and-instruction
func (b Builder) And(x, y Value, name string) Value {
	return Value{C.LLVMGoCreateAnd(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
	)}
}

// Or creates a bitwise or instruction.
// The result type matches the operand types.
// This is equivalent to a boolean or when the type is i1.
// If disjoint ("disjoint" in IR): the result is poisoned if any bits are present in both operands.
// This is used by some platforms (e.g. x86) to turn shift-or operations into shift-add instructions.
// BUG: The disjoint flag is discarded until LLVM 21 due to IRBuilder limitations.
//
// https://llvm.org/docs/LangRef.html#or-instruction
func (b Builder) Or(x, y Value, name string, disjoint bool) Value {
	return Value{C.LLVMGoCreateOr(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
		C.bool(disjoint),
	)}
}

// Xor creates a bitwise exclusive or instruction.
// The result type matches the operand types.
// This is equivalent to a boolean != when the type is i1.
//
// https://llvm.org/docs/LangRef.html#xor-instruction
func (b Builder) Xor(x, y Value, name string) Value {
	return Value{C.LLVMGoCreateXOr(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
	)}
}

// Special bitwise operations

// BitReverse reverses the order of bits within integer src.
// This is equivalent to math/bits.Reverse*.
//
// The result type matches the source type.
// This operation is performed elementwise on vectors.
//
// https://llvm.org/docs/LangRef.html#llvm-bitreverse-intrinsics
func (b Builder) BitReverse(src Value, name string) Value {
	return Value{C.LLVMGoCreateBitReverse(
		b.ptr,
		src.ptr,
		stringRef(name),
	)}
}

// ByteReverse reverses the order of bytes within integer src.
// This is equivalent to math/bits.ReverseBytes*.
//
// The result type matches the source type.
// The bit-width of the integer type must be a multiple of 16 bits.
// This operation is performed elementwise on vectors.
//
// https://llvm.org/docs/LangRef.html#llvm-bswap-intrinsics
func (b Builder) ByteReverse(src Value, name string) Value {
	return Value{C.LLVMGoCreateBSwap(
		b.ptr,
		src.ptr,
		stringRef(name),
	)}
}

// CountOnes counts the set bits within integer src.
// This is equivalent to math/bits.OnesCount*.
//
// The result type matches the source type.
// This operation is performed elementwise on vectors.
//
// https://llvm.org/docs/LangRef.html#llvm-ctpop-intrinsic
func (b Builder) CountOnes(src Value, name string) Value {
	return Value{C.LLVMGoCreateCtPop(
		b.ptr,
		src.ptr,
		stringRef(name),
	)}
}

// CountLeadingZeroes counts the leading zero bits within integer src.
// This is equivalent to math/bits.LeadingZeros*.
//
// The result type matches the source type.
// This operation is performed elementwise on vectors.
//
// If nonZero (second operand in IR): the result is poisoned if src is 0.
//
// https://llvm.org/docs/LangRef.html#llvm-ctlz-intrinsic
func (b Builder) CountLeadingZeroes(src Value, name string, nonZero bool) Value {
	return Value{C.LLVMGoCreateCtLZ(
		b.ptr,
		src.ptr,
		stringRef(name),
		C.bool(nonZero),
	)}
}

// CountTrailingZeroes counts the trailing zero bits within integer src.
// This is equivalent to math/bits.TrailingZeros*.
//
// The result type matches the source type.
// This operation is performed elementwise on vectors.
//
// If nonZero (second operand in IR): the result is poisoned if src is 0.
//
// https://llvm.org/docs/LangRef.html#llvm-cttz-intrinsic
func (b Builder) CountTrailingZeroes(src Value, name string, nonZero bool) Value {
	return Value{C.LLVMGoCreateCtTZ(
		b.ptr,
		src.ptr,
		stringRef(name),
		C.bool(nonZero),
	)}
}

// FunnelShiftLeft shifts high left, filling with bits from low.
// This is equivalent to a rotate when high and low are equal.
//
// The result type matches the operand types.
//
// The shift length "by" is used modulo the type width.
// This matches the behavior used by math/bits.RotateLeft*.
//
// https://llvm.org/docs/LangRef.html#llvm-fshl-intrinsic
func (b Builder) FunnelShiftLeft(high, low, by Value, name string) Value {
	return Value{C.LLVMGoCreateFShl(
		b.ptr,
		high.ptr,
		low.ptr,
		by.ptr,
		stringRef(name),
	)}
}

// TODO: FunnelShiftRight if we need it

// Floating-point casts

// TODO: add float flags when we need them
// (cont): The Go spec permits the "contract" flag, but not across conversions (including no-op conversions).
// (cont): This will require more research (and potentially frontend/backend work) to actually implement.

// TODO: are LLVM's floating-point environment assumptions actually correct/appropriate for us?

// FloatTrunc converts a floating-point value to a smaller type.
// This operation is performed elementwise on vectors.
//
// The name has no relation to the rounding mode colloquially known as truncation.
// This instruction uses the rounding mode from the builder's floating-point config.
// This is round-to-nearest, ties-to-even by default.
//
// LLVM does not provide a way to convert between floating-point types of equal bit-width.
// It is possible to work around this for half and bfloat16 by temporarily widening to float.
// No such workaround exists for conversions between fp128 and ppc_fp128.
//
// https://llvm.org/docs/LangRef.html#fptrunc-to-instruction
func (b Builder) FloatTrunc(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateFPTrunc(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// FloatExtend widens a floating-point value to a larger type.
// This operation is performed elementwise on vectors.
//
// https://llvm.org/docs/LangRef.html#fpext-to-instruction
func (b Builder) FloatExtend(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateFPExt(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// NOTE: DO NOT WRAP IRBuilder<>::CreateFPCast().
// (cont): It is poorly defined, not all conversions exist, and it breaks when the types are identical.

// UnsignedIntToFloat converts an unsigned integer value to a floating-point type.
// If the integer is larger than the maximum finite value, it is cast to infinity.
//
// Integers are rounded if they cannot be exactly cast.
// This inherits the builder's current floating-point configuration.
//
// If positive ("nneg" in IR): the result is poisoned if the operand's top bit is set.
// This allows the uitofp instruction to be rewritten as a sitofp instruction.
//
// https://llvm.org/docs/LangRef.html#uitofp-to-instruction
// https://reviews.llvm.org/D47807
func (b Builder) UnsignedIntToFloat(v Value, to Type, name string, positive bool) Value {
	return Value{C.LLVMGoCreateUIToFP(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
		C.bool(positive),
	)}
}

// SignedIntToFloat converts an unsigned integer value to a floating-point type.
// If the integer is too large to fit in the float type, it is cast to the corresponding infinity.
//
// Integers are rounded if they cannot be exactly cast.
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#sitofp-to-instruction
// https://reviews.llvm.org/D47807
func (b Builder) SignedIntToFloat(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateSIToFP(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// TinyGo uses saturating float-to-int casts.
// Do not bother implementing non-saturating float-to-int casts right now.

// FloatToUnsignedIntSaturating converts a floating-point value to an unsigned integer.
// Negative or NaN floats are converted to 0.
// Floats greater than the maximum result are clamped.
// The result is otherwise rounded towards zero.
//
// TODO: this cannot be constrained (https://reviews.llvm.org/D54749#2219014)
//
// https://llvm.org/docs/LangRef.html#llvm-fptoui-sat-intrinsic
func (b Builder) FloatToUnsignedIntSaturating(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateFPToUISat(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// FloatToSignedIntSaturating converts a floating-point value to an signed integer.
// Floats less than the minimum result or greater than the maximum result are clamped.
// NaN is converted to 0.
// The result is otherwise rounded towards zero.
//
// TODO: this cannot be constrained (https://reviews.llvm.org/D54749#2219014)
//
// https://llvm.org/docs/LangRef.html#llvm-fptosi-sat-intrinsic
func (b Builder) FloatToSignedIntSaturating(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateFPToSISat(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// Floating-point arithmetic

// FloatNegate creates a floating-point negation instruction.
// There is no constrained negation intrinsic.
// This flips the sign bit of the float, even if it is zero.
// This preserves the signaling bit and payload of NaN operands.
//
// The result type matches the source type.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#fneg-instruction
func (b Builder) FloatNegate(src Value, name string) Value {
	return Value{C.LLVMGoCreateFNeg(
		b.ptr,
		src.ptr,
		stringRef(name),
	)}
}

// FloatAdd creates a floating-point addition instruction or intrinsic.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#fadd-instruction
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-fadd-intrinsic
func (b Builder) FloatAdd(x, y Value, name string) Value {
	return Value{C.LLVMGoCreateFAdd(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
	)}
}

// FloatSubtract creates a floating-point subtraction instruction or intrinsic.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#fsub-instruction
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-fsub-intrinsic
func (b Builder) FloatSubtract(minuend, subtrahend Value, name string) Value {
	return Value{C.LLVMGoCreateFSub(
		b.ptr,
		minuend.ptr,
		subtrahend.ptr,
		stringRef(name),
	)}
}

// FloatMultiply creates a floating-point multiplication instruction or intrinsic.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#fmul-instruction
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-fmul-intrinsic
func (b Builder) FloatMultiply(x, y Value, name string) Value {
	return Value{C.LLVMGoCreateFMul(
		b.ptr,
		x.ptr,
		y.ptr,
		stringRef(name),
	)}
}

// FloatDivide creates a floating-point division instruction or intrinsic.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#fdiv-instruction
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-fdiv-intrinsic
func (b Builder) FloatDivide(dividend, divisor Value, name string) Value {
	return Value{C.LLVMGoCreateFDiv(
		b.ptr,
		dividend.ptr,
		divisor.ptr,
		stringRef(name),
	)}
}

// FloatRemainder creates a floating-point remainder instruction or intrinsic.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#frem-instruction
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-frem-intrinsic
func (b Builder) FloatRemainder(dividend, divisor Value, name string) Value {
	return Value{C.LLVMGoCreateFRem(
		b.ptr,
		dividend.ptr,
		divisor.ptr,
		stringRef(name),
	)}
}

// FloatCompare creates a floating-point comparison instruction.
// This operation is performed elementwise on vectors.
//
// https://llvm.org/docs/LangRef.html#fcmp-instruction
func (b Builder) FloatCompare(cmp FloatComparison, lhs, rhs Value, name string) Value {
	return Value{C.LLVMGoCreateFCmp(
		b.ptr,
		C.uint8_t(cmp),
		lhs.ptr,
		rhs.ptr,
		stringRef(name),
	)}
}

// FloatComparison represents a floating-point comparison operator.
// It is structured as a bitmask.
// The FloatEqual/FloatGreater/FloatLess/FloatNaN can be or'd together to produce any comparison.
type FloatComparison uint8

const (
	// FloatEqual is equivalent to Go's == operator.
	// Positive zero is equal to negative zero.
	// The comparison fails if either operand is NaN.
	FloatEqual FloatComparison = C.LLVMGoFloatEqual

	// FloatGreater is equivalent to Go's > operator.
	// Positive zero is not greater than negative zero.
	// The comparison fails if either operand is NaN.
	FloatGreater FloatComparison = C.LLVMGoFloatGreater

	// FloatGreater is equivalent to Go's < operator.
	// Negative zero is not less than positive zero.
	// The comparison fails if either operand is NaN.
	FloatLess FloatComparison = C.LLVMGoFloatLess

	// FloatNaN matches if either operand is NaN.
	FloatNaN FloatComparison = C.LLVMGoFloatNaN
)

// Not inverts the comparison condition.
func (fcmp FloatComparison) Not() FloatComparison {
	return (FloatEqual | FloatGreater | FloatLess | FloatNaN) ^ fcmp
}

// String formats the comparison operator as it would be printed in IR.
func (fcmp FloatComparison) String() string {
	return floatComparisonNames[fcmp]
}

var floatComparisonNames = [...]string{
	// 0bNLGE
	0b0000: "false",
	0b0001: "oeq",
	0b0010: "ogt",
	0b0011: "oge",
	0b0100: "olt",
	0b0101: "ole",
	0b0110: "one",
	0b0111: "ord",
	0b1000: "uno",
	0b1001: "ueq",
	0b1010: "ugt",
	0b1011: "uge",
	0b1100: "ult",
	0b1101: "ule",
	0b1110: "une",
	0b1111: "true",
}

// FloatMinimum finds the minimum of the provided floating-point values.
// The result is NaN if any operand is NaN.
// Negative zero is considered less than positive zero.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#llvm-minimum-intrinsic
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-minimum-intrinsic
func (b Builder) FloatMinimum(name string, first Value, more ...Value) Value {
	return Value{C.LLVMGoCreateMinimum(
		b.ptr,
		stringRef(name),
		first.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)}
}

// FloatMaximum finds the minimum of the provided floating-point values.
// The result is NaN if any operand is NaN.
// Negative zero is considered less than positive zero.
//
// The result type matches the operand types.
// This operation is performed elementwise on vectors.
//
// This inherits the builder's current floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#llvm-maximum-intrinsic
// https://llvm.org/docs/LangRef.html#llvm-experimental-constrained-maximum-intrinsic
func (b Builder) FloatMaximum(name string, first Value, more ...Value) Value {
	return Value{C.LLVMGoCreateMaximum(
		b.ptr,
		stringRef(name),
		first.ptr,
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(more))),
		C.size_t(len(more)),
	)}
}

// Pointer conversions

// PtrToInt converts a pointer to an integer, truncating or zero-extending to match the destination type.
// The result can be cast back to a pointer with IntToPtr.
//
// https://llvm.org/docs/LangRef.html#ptrtoint-to-instruction
func (b Builder) PtrToInt(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreatePtrToInt(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// TODO: add ptrtoaddr when we suppport LLVM 22

// IntToPtr converts an integer to a pointer, zero-extending or truncating to match the index type.
//
// https://llvm.org/docs/LangRef.html#inttoptr-to-instruction
func (b Builder) IntToPtr(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateIntToPtr(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// TODO: addrspacecast if we need it

// NOTE: bitcast doesn't really fit here, but it doesn't really fit anywhere else either

// BitCast reinterprets a constant as another type by reusing the raw bit layout.
// NOTE: Big-endian vectors w/ byte-multiple elements are reversed when bitcasting.
//
// https://llvm.org/docs/LangRef.html#bitcast-to-instruction
func (b Builder) BitCast(v Value, to Type, name string) Value {
	return Value{C.LLVMGoCreateBitCast(
		b.ptr,
		v.ptr,
		to.ptr,
		stringRef(name),
	)}
}

// Memory operations

// StaticAlloca creates a fixed-width stack variable in the entry block of the current function.
func (b Builder) StaticAlloca(ty Type, addrSpace uint32, align Alignment, name string) Alloca {
	return Alloca{Value{C.LLVMGoCreateStaticAlloca(
		b.ptr,
		ty.ptr,
		C.unsigned(addrSpace),
		align.toC(),
		stringRef(name),
	)}}
}

type Alloca struct {
	Value
}

// StartLifetime inserts an intrinsic that indicates that the provided alloca is not live until this point.
// This must be paired with EndLifetime.
func (b Builder) StartLifetime(alloca Alloca) {
	C.LLVMGoCreateLifetimeStart(b.ptr, alloca.ptr)
}

// EndLifetime inserts an intrinsic that indicates that the provided alloca is not live after this point.
// This must be paired with StartLifetime.
func (b Builder) EndLifetime(alloca Alloca) {
	C.LLVMGoCreateLifetimeEnd(b.ptr, alloca.ptr)
}

// GEP is handled in gep.go

// Load creates an instruction to load from a pointer with a specific type.
//
// When loading a type with a width that is not a multiple of 8 bits (e.g. i1), the source store must use exactly the same type.
// The type otherwise only describes how the value will be stored, and does not imply aliasing constraints.
//
// Atomic (opts.Order != MemOrderNormal) loads are only valid for:
// - integers
// - floats
// - pointers
// - vectors of the other 3 types
// The total bit-width of the atomically loaded type must be a power of 2 of at least 8.
// MemOrderRelease and MemOrderAcquireRelease are not valid on load instructions.
//
// https://llvm.org/docs/LangRef.html#load-instruction
func (b Builder) Load(as Type, from Value, opts MemOptions, name string) Value {
	switch opts.Order {
	case MemOrderRelease, MemOrderAcquireRelease:
		panic("cannot release with a load")
	}
	return Value{C.LLVMGoCreateLoad(
		b.ptr,
		as.ptr,
		from.ptr,
		opts.toC(),
		stringRef(name),
	)}
}

// Store creates an instruction to store a value into memory.
//
// Atomic (opts.Order != MemOrderNormal) stores are only valid for:
// - integers
// - floats
// - pointers
// - vectors of the other 3 types
// The total bit-width of the atomically stored type must be a power of 2 greater than 8.
//
// https://llvm.org/docs/LangRef.html#store-instruction
func (b Builder) Store(value, to Value, opts MemOptions) {
	C.LLVMGoCreateStore(
		b.ptr,
		value.ptr,
		to.ptr,
		opts.toC(),
	)
}

// CompareAndExchange atomically loads a value from ptr and replaces it with to if it matches from.
// The access type matches the from/to operands.
// The result is a struct containing the loaded value and a boolean indicating whether the operation succeeded.
//
// The from/to type must be an integer/pointer.
// The type's bit-width must be a power of 2 of at least 8.
//
// The cmpxchg instruction has seperate memory orderings for success and failure.
// Simple CAS loops may use a weaker failure order because consistency only matters when exiting the loop.
// MemOrderRelease and MemOrderAcquireRelease are not valid failure orderings.
// MemOrderNormal and MemOrderUnordered are not valid for either case.
//
// https://llvm.org/docs/LangRef.html#cmpxchg-instruction
func (b Builder) CompareAndExchange(ptr, from, to Value, success, failure MemOrder, opts CompareAndExchangeOptions, name string) Value {
	switch success {
	case MemOrderNormal, MemOrderUnordered:
		panic("cmpxchg must be atomic")
	}
	switch failure {
	case MemOrderNormal, MemOrderUnordered:
		panic("cmpxchg must be atomic")
	case MemOrderRelease, MemOrderAcquireRelease:
		panic("cannot release on failure")
	}
	return Value{C.LLVMGoCreateCmpXchg(
		b.ptr,
		ptr.ptr,
		from.ptr,
		to.ptr,
		C.LLVMGoMemOrder(success),
		C.LLVMGoMemOrder(failure),
		opts.toC(),
		stringRef(name),
	)}
}

// CompareAndExchangeOptions holds optional information for a cmpxchg instruction.
type CompareAndExchangeOptions struct {
	// Alignment is the required alignment for the accessed address.
	Alignment Alignment

	// Omit sync scope.
	// See MemOptions for more info.

	// Volatile determines whether this is a volatile access.
	Volatile bool

	// Weak allows the operation to fail even if the comparison succeeds.
	// This matches the behavior of exclusive-monitor atomics on ARM/MIPS.
	// Non-weak cmpxchg instructions will be retried internally if this happens.
	Weak bool
}

func (opts CompareAndExchangeOptions) toC() C.LLVMGoCmpXchgOptions {
	return C.LLVMGoCmpXchgOptions{
		align:      opts.Alignment.toC(),
		isVolatile: C.bool(opts.Volatile),
		isWeak:     C.bool(opts.Weak),
	}
}

// AtomicRMW creates an atomic read-modify-write operation.
// The loaded/stored type matches the value type.
// The result is the original value before modification.
//
// The bit-width of the type must be a power of 2 of at least 8.
// The AtomicSwap op accepts integer/float/pointer scalars.
// Integer ops (e.g. AtomicAdd) only accept integer scalars.
// Floating-point ops (e.g. AtomicFAdd) accept floating-point scalars or fixed-width vectors.
//
// https://llvm.org/docs/LangRef.html#atomicrmw-instruction
func (b Builder) AtomicRMW(op AtomicOp, ptr, value Value, opts MemOptions, name string) Value {
	if !op.Supported() {
		panic("unsupported atomic op")
	}
	return Value{C.LLVMGoCreateAtomicRMW(
		b.ptr,
		C.LLVMGoAtomicRMWOp(op),
		ptr.ptr,
		value.ptr,
		opts.toC(),
		stringRef(name),
	)}
}

// AtomicOp is an operator for AtomicRMW.
// Not all operators are supported by all LLVM versions.
// Use AtomicOp.Suppported to check if an operation is supported by the current LLVM version.
//
// https://llvm.org/docs/LangRef.html#atomicrmw-instruction (semantics)
type AtomicOp C.LLVMGoAtomicRMWOp

const (
	// AtomicSwap ("xchg" in IR) replaces the old value with the provided value.
	// This matches the behavior of sync/atomic.Swap* when using MemOrderSequentiallyConsistent.
	AtomicSwap AtomicOp = C.LLVMGoAtomicRMWOpXchg

	// AtomicAdd ("add" in IR) adds the provided integer to the old integer, wrapping on overflow.
	// This matches the behavior of sync/atomic.Add* when using MemOrderSequentiallyConsistent.
	AtomicAdd AtomicOp = C.LLVMGoAtomicRMWOpAdd

	// AtomicSub ("sub" in IR) subtracts the provided integer from the old integer, wrapping on overflow.
	// This is equivalent to AtomicAdd with a negated operand.
	AtomicSub AtomicOp = C.LLVMGoAtomicRMWOpSub

	// AtomicAnd ("and" in IR) performs an bitwise and with the provided integer.
	// This matches the behavior of sync/atomic.And* when using MemOrderSequentiallyConsistent.
	AtomicAnd AtomicOp = C.LLVMGoAtomicRMWOpAnd

	// AtomicNotAnd ("nand" in IR) performs an bitwise not-and with the provided integer.
	AtomicNotAnd AtomicOp = C.LLVMGoAtomicRMWOpNAnd

	// AtomicOr ("or" in IR) performs an bitwise or with the provided integer.
	// This matches the behavior of sync/atomic.Or* when using MemOrderSequentiallyConsistent.
	AtomicOr AtomicOp = C.LLVMGoAtomicRMWOpOr

	// AtomicXOr ("xor" in IR) performs an bitwise exclusive or with the provided integer.
	AtomicXOr AtomicOp = C.LLVMGoAtomicRMWOpXOr

	// AtomicSignedMax ("max" in IR) replaces the old signed integer with the provided signed integer if the new integer is greater.
	AtomicSignedMax AtomicOp = C.LLVMGoAtomicRMWOpMax

	// AtomicSignedMin ("min" in IR) replaces the old signed integer with the provided signed integer if the new integer is less.
	AtomicSignedMin AtomicOp = C.LLVMGoAtomicRMWOpMin

	// AtomicUnsignedMax ("umax" in IR) replaces the old unsigned integer with the provided unsigned integer if the new integer is greater.
	AtomicUnsignedMax AtomicOp = C.LLVMGoAtomicRMWOpUMax

	// AtomicUnsignedMin ("umin" in IR) replaces the old unsigned integer with the provided unsigned integer if the new integer is less.
	AtomicUnsignedMin AtomicOp = C.LLVMGoAtomicRMWOpUMin

	// AtomicUnsignedIncrementAndWrap ("uinc_wrap" in IR) increments the old unsigned integer if it is less than the provided unsigned integer.
	// Otherwise, it sets it to 0.
	//
	// This requires LLVM 16 or newer.
	AtomicUnsignedIncrementAndWrap AtomicOp = C.LLVMGoAtomicRMWOpUIncWrap

	// AtomicUnsignedDecrementAndWrap ("udec_wrap" in IR) decrements the integer if it nonzero.
	// Otherwise, it sets it to the provided integer.
	//
	// This requires LLVM 16 or newer.
	AtomicUnsignedDecrementAndWrap AtomicOp = C.LLVMGoAtomicRMWOpUDecWrap

	// AtomicUnsignedSubtractIfNoUnderflow ("usub_cond" in IR) subtracts the provided integer from the old integer.
	// If the provided unsigned integer is greater than the unsigned old integer, the old value is left unchanged.
	//
	// This requires LLVM 20 or newer.
	AtomicUnsignedSubtractIfNoUnderflow AtomicOp = C.LLVMGoAtomicRMWOpUSubCond

	// AtomicUnsignedSubtractSaturating ("usub_sat" in IR) subtracts the provided integer from the old integer.
	// If the unsigned subtraction underflows, the value is set to 0.
	//
	// This requires LLVM 20 or newer.
	AtomicUnsignedSubtractSaturating AtomicOp = C.LLVMGoAtomicRMWOpUSubSat

	// AtomicFloatAdd adds the provided float to the old float.
	AtomicFloatAdd AtomicOp = C.LLVMGoAtomicRMWOpFAdd

	// AtomicFloatSub subtracts the provided float to the old float.
	AtomicFloatSub AtomicOp = C.LLVMGoAtomicRMWOpFSub

	// AtomicFloatMaxNum ("fmax" in IR) uses llvm.maxnum.* with the old and provided floats.
	// If one of the floats is NaN, it will use the other float.
	// Use AtomicFloatMaximum instead to get sane NaN handling.
	AtomicFloatMaxNum AtomicOp = C.LLVMGoAtomicRMWOpFMax

	// AtomicFloatMinNum ("fmin" in IR) uses llvm.minnum.* with the old and provided floats.
	// If one of the floats is NaN, it will use the other float.
	// Use AtomicFloatMinimum instead to get sane NaN handling.
	AtomicFloatMinNum AtomicOp = C.LLVMGoAtomicRMWOpFMin

	// AtomicFloatMaximum ("fmaximum" in IR) uses llvm.maximum.* with the old and provided floats.
	// If one of the floats is NaN, the value is set to NaN.
	//
	// This requires LLVM 21 or newer.
	AtomicFloatMaximum AtomicOp = C.LLVMGoAtomicRMWOpFMaximum

	// AtomicFloatMinimum ("fminimum" in IR) uses llvm.minimum.* with the old and provided floats.
	// If one of the floats is NaN, the value is set to NaN.
	//
	// This requires LLVM 21 or newer.
	AtomicFloatMinimum AtomicOp = C.LLVMGoAtomicRMWOpFMinimum
)

// String returns the name of the op as it would be printed in IR.
func (op AtomicOp) String() string {
	return atomicOpNames[op]
}

var atomicOpNames = [...]string{
	AtomicSwap:                          "xchg",
	AtomicAdd:                           "add",
	AtomicSub:                           "sub",
	AtomicAnd:                           "and",
	AtomicNotAnd:                        "nand",
	AtomicOr:                            "or",
	AtomicXOr:                           "xor",
	AtomicSignedMax:                     "max",
	AtomicSignedMin:                     "min",
	AtomicUnsignedMax:                   "umax",
	AtomicUnsignedMin:                   "umin",
	AtomicUnsignedIncrementAndWrap:      "uinc_wrap",
	AtomicUnsignedDecrementAndWrap:      "udec_wrap",
	AtomicUnsignedSubtractIfNoUnderflow: "usub_cond",
	AtomicUnsignedSubtractSaturating:    "usub_sat",
	AtomicFloatAdd:                      "fadd",
	AtomicFloatSub:                      "fsub",
	AtomicFloatMaxNum:                   "fmax",
	AtomicFloatMinNum:                   "fmin",
	AtomicFloatMaximum:                  "fmaximum",
	AtomicFloatMinimum:                  "fminimum",
}

// Supported checks if this operation is supported by the current LLVM version.
func (op AtomicOp) Supported() bool {
	// Negative ops are invalid but we may as well mark them as unsupported.
	// NOTE: Ignore the staticcheck warning about uint32 never being negative.
	// (cont): The underlying type of an enum is platform-dependent.
	return op >= 0 && op <= C.LLVMGoAtomicRMWOpSupported
}

type MemOptions struct {
	// Alignment is the required alignment for the accessed address.
	Alignment Alignment

	// Order defines the atomicity and synchronization of the access.
	Order MemOrder

	// Omit the sync scope for now.
	// It is incredibly annoying to implement because:
	// - The default (system) is not 0 (that is single-thread)
	// - Some values are assigned at runtime (so we cant just use a table like everything else)

	// Volatile determines whether this is a volatile access.
	Volatile bool
}

func (opts MemOptions) toC() C.LLVMGoMemOptions {
	return C.LLVMGoMemOptions{
		align:      opts.Alignment.toC(),
		order:      C.LLVMGoMemOrder(opts.Order),
		isVolatile: C.bool(opts.Volatile),
	}
}

// Alignment is a power of 2 that a memory address must be a multiple of.
// The zero value (ImplicitAlignment) is used to infer the alignment from the memory type.
type Alignment uint64

// ImplicitAlignment is a sentinel used to omit explicit alignment from memory operations.
// LLVM will infer the alignment from the operation's type.
const ImplicitAlignment Alignment = 0

func (align Alignment) toC() C.uint64_t {
	if align&(align-1) != 0 || align >= 1<<32 {
		panic("invalid alignment")
	}
	return C.uint64_t(align)
}

// MemOrder defines the atomicity and synchronization of memory accesses.
//
// https://llvm.org/docs/LangRef.html#atomic-memory-ordering-constraints
// https://llvm.org/docs/Atomics.html
// https://en.cppreference.com/w/cpp/atomic/memory_order.html
type MemOrder C.LLVMGoMemOrder

const (
	// MemOrderNormal (the default) has no special synchronization properties.
	// This is used for regular memory accesses.
	//
	// https://llvm.org/docs/Atomics.html#notatomic
	MemOrderNormal MemOrder = C.LLVMGoMemOrderNormal

	// MemOrderUnordered provides extremely weak guarantees:
	// - Concurrent access will not result in undefined behavior.
	// - A loaded value has been previously stored.
	// - No locking is performed.
	//
	// Accesses may be reordered and no total ordering is guaranteed.
	//
	// The backend will produce an error if the CPU does not have an exactly-matching instruction.
	//
	// https://llvm.org/docs/Atomics.html#unordered
	MemOrderUnordered MemOrder = C.LLVMGoMemOrderUnordered

	// MemOrderMonotonic performs memory access atomically, but does not synchronize.
	// This matches C++'s std::mem_order_relaxed.
	//
	// https://llvm.org/docs/Atomics.html#monotonic
	// https://en.cppreference.com/w/cpp/atomic/memory_order.html#Relaxed_ordering
	MemOrderMonotonic MemOrder = C.LLVMGoMemOrderMonotonic

	// MemOrderAcquire performs memory access atomically and synchronizes with MemOrderRelease operations.
	// This intended for use when acquiring locks and matches C++'s std::mem_order_release.
	//
	// Memory access after the acquire (under the lock) cannot be moved before it (outside the lock).
	// Memory access before the acquire (outside the lock) can be moved after it (under the lock).
	//
	// https://llvm.org/docs/Atomics.html#acquire
	// https://en.cppreference.com/w/cpp/atomic/memory_order.html#Release-Acquire_ordering
	MemOrderAcquire MemOrder = C.LLVMGoMemOrderAcquire

	// MemOrderRelease performs memory access atomically and synchronizes with MemOrderAcquire operations.
	// This intended for use when releasing locks and matches C++'s std::mem_order_release.
	//
	// Memory access before the release (under the lock) cannot be moved after it (outside the lock).
	// Memory access after the release (outside the lock) can be moved before it (under the lock).
	//
	// https://llvm.org/docs/Atomics.html#release
	// https://en.cppreference.com/w/cpp/atomic/memory_order.html#Release-Acquire_ordering
	MemOrderRelease MemOrder = C.LLVMGoMemOrderRelease

	// MemOrderAcquireRelease combines MemOrderAcquire and MemOrderRelease.
	// This matches C++'s std::mem_order_acq_rel.
	//
	// https://llvm.org/docs/Atomics.html#acquirerelease
	// https://en.cppreference.com/w/cpp/atomic/memory_order.html#Release-Acquire_ordering
	MemOrderAcquireRelease MemOrder = C.LLVMGoMemOrderAcquireRelease

	// MemOrderSequentiallyConsistent performs memory access atomically and guarantees a consistent global ordering.
	// This matches Go's atomics and C++'s std::mem_order_seq_cst.
	//
	// https://llvm.org/docs/Atomics.html#sequentiallyconsistent
	// https://en.cppreference.com/w/cpp/atomic/memory_order.html#Sequentially-consistent_ordering
	MemOrderSequentiallyConsistent MemOrder = C.LLVMGoMemOrderSequentiallyConsistent
)

// MemSet fills an array with repeats of a value.
// This works with all memory address spaces.
//
// https://llvm.org/docs/LangRef.html#llvm-memset-intrinsics
func (b Builder) MemSet(dst, value, len Value, align Alignment, volatile bool) {
	C.LLVMGoCreateMemSet(
		b.ptr,
		dst.ptr,
		value.ptr,
		len.ptr,
		align.toC(),
		C.bool(volatile),
	)
}

// MemCpy copies memory between non-overlapping buffers.
// This works between any pair of address spaces.
//
// https://llvm.org/docs/LangRef.html#llvm-memcpy-intrinsic
func (b Builder) MemCopy(
	dst Value, dstAlign Alignment,
	src Value, srcAlign Alignment,
	len Value, volatile bool,
) {
	C.LLVMGoCreateMemCpy(
		b.ptr,
		dst.ptr, dstAlign.toC(),
		src.ptr, srcAlign.toC(),
		len.ptr, C.bool(volatile),
	)
}

// MemMove copies memory between buffers which may overlap.
// This works between any pair of address spaces.
//
// https://llvm.org/docs/LangRef.html#llvm-memmove-intrinsic
func (b Builder) MemMove(
	dst Value, dstAlign Alignment,
	src Value, srcAlign Alignment,
	len Value, volatile bool,
) {
	C.LLVMGoCreateMemMove(
		b.ptr,
		dst.ptr, dstAlign.toC(),
		src.ptr, srcAlign.toC(),
		len.ptr, C.bool(volatile),
	)
}

// Aggregate manipulation

// BuildAggregate constructs an aggregate (struct/array) from the provided values.
// The number of provided values must exactly match the number of elements in the type.
//
// This is equivalent to a sequence of InsertValue calls.
//
// https://llvm.org/docs/LangRef.html#insertvalue-instruction
func (b Builder) BuildAggregate(ty Type, name string, values ...Value) Value {
	return Value{C.LLVMGoCreateAggregate(
		b.ptr,
		ty.ptr,
		stringRef(name),
		(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(values))),
		castAggLen(len(values)),
	)}
}

func castAggLen(src int) C.unsigned {
	cast := C.unsigned(src)
	if int(cast) != src {
		panic("aggregate len overflow")
	}
	return cast
}

// InsertValue replaces an element of an aggregate (struct/array).
// The provided indices represent a path to the element to replace.
//
// https://llvm.org/docs/LangRef.html#insertvalue-instruction
func (b Builder) InsertValue(into, value Value, name string, idxs ...uint32) Value {
	return Value{C.LLVMGoCreateInsertValue(
		b.ptr,
		into.ptr,
		value.ptr,
		(*C.unsigned)(unsafe.Pointer(unsafe.SliceData(idxs))),
		C.size_t(len(idxs)),
		stringRef(name),
	)}
}

// ExtractValue gets an element of an aggregate (struct/array).
// The provided indices represent a path to the element to extract.
//
// https://llvm.org/docs/LangRef.html#extractvalue-instruction
func (b Builder) ExtractValue(from Value, name string, idxs ...uint32) Value {
	return Value{C.LLVMGoCreateExtractValue(
		b.ptr,
		from.ptr,
		(*C.unsigned)(unsafe.Pointer(unsafe.SliceData(idxs))),
		C.size_t(len(idxs)),
		stringRef(name),
	)}
}

// Control-flow

// Return creates an instruction to exit the function, returning the provided values.
//
// If no values are passed, a void return is created.
// If a single value is passed, it is returned.
// If multiple values are passed, they are combined into an aggregate and returned.
//
// NOTE: It is also legal to create a void return by passing a nil or void value.
//
// This must be called at the end of a block.
//
// https://llvm.org/docs/LangRef.html#ret-instruction
func (b Builder) Return(values ...Value) {
	// This internally just dispatches to the correct constructor, but this API is more readable.
	switch len(values) {
	case 0:
		C.LLVMBuildRetVoid(b.ptr)
	case 1:
		C.LLVMBuildRet(b.ptr, values[0].ptr)
	default:
		C.LLVMBuildAggregateRet(
			b.ptr,
			(*C.LLVMValueRef)(unsafe.Pointer(unsafe.SliceData(values))),
			castAggLen(len(values)),
		)
	}
}

// Jump creates an unconditional branch instruction.
//
// This must be called at the end of a block.
//
// https://llvm.org/docs/LangRef.html#br-instruction
func (b Builder) Jump(to BasicBlock) {
	C.LLVMBuildBr(b.ptr, to.ptr)
}

// Branch creates a conditional branch instruction.
//
// This must be called at the end of a block.
//
// https://llvm.org/docs/LangRef.html#br-instruction
func (b Builder) Branch(condition Value, ifTrue, ifFalse BasicBlock) {
	C.LLVMBuildCondBr(b.ptr, condition.ptr, ifTrue.ptr, ifFalse.ptr)
}

// If creates a conditional branch to two new blocks.
// The insertion point is moved to the ifBlock.
//
// This must be called at the end of a block.
//
// https://llvm.org/docs/LangRef.html#br-instruction
func (b Builder) If(condition Value, ifName, elseName string) (ifBlock, elseBlock BasicBlock) {
	blocks := C.LLVMGoCreateIf(
		b.ptr,
		condition.ptr,
		stringRef(ifName),
		stringRef(elseName),
	)
	return BasicBlock{blocks.ifBlock}, BasicBlock{blocks.elseBlock}
}

// Switch creates a branch based on an integer index.
//
// This must be called at the end of a block.
//
// https://llvm.org/docs/LangRef.html#switch-instruction
func (b Builder) Switch(on Value, defaultBlock BasicBlock, cases ...SwitchCase) {
	C.LLVMGoCreateSwitch(
		b.ptr,
		on.ptr,
		defaultBlock.ptr,
		(*C.LLVMGoSwitchCase)(unsafe.Pointer(unsafe.SliceData(cases))),
		C.size_t(len(cases)),
	)
}

type SwitchCase struct {
	// NOTE: This must be kept in sync with C.LLVMGoSwitchCase.

	// Index is a constant integer value to match.
	// It must be a raw integer constant - constant expressions cannot be used.
	Index Constant

	// Block is the block to branch to if the switch value matches the index.
	Block BasicBlock
}

// Unreachable creates an unreachable terminator instruction.
//
// This must be called at the end of a block.
//
// https://llvm.org/docs/LangRef.html#unreachable-instruction
func (b Builder) Unreachable() {
	C.LLVMBuildUnreachable(b.ptr)
}

// Phi creates a phi node.
// A phi node is used to combine values after a branch.
//
// The provided incoming values are added to the phi node.
// Additional incoming values can be added with Phi.Add.
//
// This must be called at the start of a non-entry block.
//
// https://llvm.org/docs/LangRef.html#phi-instruction
func (b Builder) Phi(ty Type, name string, incoming ...PhiIncoming) Phi {
	return Phi{Value{C.LLVMGoCreatePhi(
		b.ptr,
		ty.ptr,
		stringRef(name),
		(*C.LLVMGoPhiIncoming)(unsafe.Pointer(unsafe.SliceData(incoming))),
		C.size_t(len(incoming)),
	)}}
}

// Phi is an instruction which can be used to combine values after a branch.
//
// https://llvm.org/docs/LangRef.html#phi-instruction
type Phi struct {
	Value
}

// Add incoming values to the phi node.
func (p Phi) Add(incoming ...PhiIncoming) {
	C.LLVMGoAddPhiIncoming(
		p.ptr,
		(*C.LLVMGoPhiIncoming)(unsafe.Pointer(unsafe.SliceData(incoming))),
		C.size_t(len(incoming)),
	)
}

type PhiIncoming struct {
	// NOTE: This must be kept in sync with C.LLVMGoPhiIncoming.

	Value Value
	From  BasicBlock
}

// Select creates an instruction to pick one of two values based on a condition.
// This is used to represent a "cond ? ifTrue : ifFalse" expression without branching.
// The result is not poisoned unless either the condition or the selected value is poisoned.
//
// The result type matches the provided values.
// If the condition is a vector, this operation is performed elementwise.
//
// If the result is a floating-point scalar or vector, it will inherit the builder's floating-point configuration.
//
// https://llvm.org/docs/LangRef.html#select-instruction
func (b Builder) Select(condition, ifTrue, ifFalse Value, name string) Value {
	return Value{C.LLVMGoCreateSelect(
		b.ptr,
		condition.ptr,
		ifTrue.ptr,
		ifFalse.ptr,
		stringRef(name),
	)}
}

// CallDirect calls a function using its defined signature.
func (b Builder) CallDirect(callee Function, args ...Value) Value {
	panic("TODO")
}

// CallIndirect calls a function with an explicit signature.
func (b Builder) CallIndirect(signature Signature, callee Value, args ...Value) Value {
	panic("TODO")
}
