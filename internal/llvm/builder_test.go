package llvm_test

import (
	"testing"

	"github.com/tinygo-org/tinygo/internal/llvm"
)

// TODO

func TestBuildDoNothing(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function with no arguments that returns void.
	fn := mod.CreateFunction("doNothing", llvm.Signature{
		Type: ctx.Function(ctx.Void(), false),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	builder.Return()

	// Compare the module string.
	if str := mod.String(); str != doNothingMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const doNothingMod = `
define void @doNothing() {
  ret void
}
`

func TestBuilderIntCasts(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i8 := ctx.Int(8)
	i16 := ctx.Int(16)
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("intCasts", llvm.Signature{
		Type: ctx.Function(ctx.LiteralStruct(
			i8, i8, i8, i8,
			i32, i32,
		), false, i16),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	arg := fn.Param(0)
	builder.Return(
		builder.Truncate(arg, i8, "trunc", false, false),
		builder.Truncate(arg, i8, "trunc.u", true, false),
		builder.Truncate(arg, i8, "trunc.s", false, true),
		builder.Truncate(arg, i8, "trunc.us", true, true),
		// zext nneg must be a seprate test due to version restrictions
		builder.ZeroExtend(arg, i32, "zext", false),
		builder.SignExtend(arg, i32, "sext"),
	)

	// Compare the module string.
	if str := mod.String(); str != intCastsMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const intCastsMod = `
define { i8, i8, i8, i8, i32, i32 } @intCasts(i16 %0) {
  %trunc = trunc i16 %0 to i8
  %trunc.u = trunc nuw i16 %0 to i8
  %trunc.s = trunc nsw i16 %0 to i8
  %trunc.us = trunc nuw nsw i16 %0 to i8
  %zext = zext i16 %0 to i32
  %sext = sext i16 %0 to i32
  %mrv = insertvalue { i8, i8, i8, i8, i32, i32 } poison, i8 %trunc, 0
  %mrv1 = insertvalue { i8, i8, i8, i8, i32, i32 } %mrv, i8 %trunc.u, 1
  %mrv2 = insertvalue { i8, i8, i8, i8, i32, i32 } %mrv1, i8 %trunc.s, 2
  %mrv3 = insertvalue { i8, i8, i8, i8, i32, i32 } %mrv2, i8 %trunc.us, 3
  %mrv4 = insertvalue { i8, i8, i8, i8, i32, i32 } %mrv3, i32 %zext, 4
  %mrv5 = insertvalue { i8, i8, i8, i8, i32, i32 } %mrv4, i32 %sext, 5
  ret { i8, i8, i8, i8, i32, i32 } %mrv5
}
`

func TestBuilderZextNNeg(t *testing.T) {
	if llvm.VersionMajor < 18 {
		// The nneg flag was added in LLVM 18.
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function with no arguments that returns void.
	i8 := ctx.Int(8)
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("zextNNeg", llvm.Signature{
		Type: ctx.Function(i32, false, i8),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	arg := fn.Param(0)
	zextNNeg := builder.ZeroExtend(arg, i32, "zext.nneg", true)
	builder.Return(zextNNeg)

	// Compare the module string.
	if str := mod.String(); str != zextNNegMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const zextNNegMod = `
define i32 @zextNNeg(i8 %0) {
  %zext.nneg = zext nneg i8 %0 to i32
  ret i32 %zext.nneg
}
`

func TestBuilderIntMath(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("intMath", llvm.Signature{
		Type: ctx.Function(ctx.Array(18, i32), false, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.Add(x, y, "add", false, false),
		builder.Add(x, y, "add.u", true, false),
		builder.Add(x, y, "add.s", false, true),
		builder.Add(x, y, "add.us", true, true),
		builder.Subtract(x, y, "sub", false, false),
		builder.Subtract(x, y, "sub.u", true, false),
		builder.Subtract(x, y, "sub.s", false, true),
		builder.Subtract(x, y, "sub.us", true, true),
		builder.Multiply(x, y, "mul", false, false),
		builder.Multiply(x, y, "mul.u", true, false),
		builder.Multiply(x, y, "mul.s", false, true),
		builder.Multiply(x, y, "mul.us", true, true),
		builder.UnsignedDivide(x, y, "udiv", false),
		builder.UnsignedDivide(x, y, "udiv.exact", true),
		builder.SignedDivide(x, y, "sdiv", false),
		builder.SignedDivide(x, y, "sdiv.exact", true),
		builder.UnsignedRemainder(x, y, "urem"),
		builder.SignedRemainder(x, y, "srem"),
	)

	// Compare the module string.
	if str := mod.String(); str != intMathMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const intMathMod = `
define [18 x i32] @intMath(i32 %0, i32 %1) {
  %add = add i32 %0, %1
  %add.u = add nuw i32 %0, %1
  %add.s = add nsw i32 %0, %1
  %add.us = add nuw nsw i32 %0, %1
  %sub = sub i32 %0, %1
  %sub.u = sub nuw i32 %0, %1
  %sub.s = sub nsw i32 %0, %1
  %sub.us = sub nuw nsw i32 %0, %1
  %mul = mul i32 %0, %1
  %mul.u = mul nuw i32 %0, %1
  %mul.s = mul nsw i32 %0, %1
  %mul.us = mul nuw nsw i32 %0, %1
  %udiv = udiv i32 %0, %1
  %udiv.exact = udiv exact i32 %0, %1
  %sdiv = sdiv i32 %0, %1
  %sdiv.exact = sdiv exact i32 %0, %1
  %urem = urem i32 %0, %1
  %srem = srem i32 %0, %1
  %mrv = insertvalue [18 x i32] poison, i32 %add, 0
  %mrv1 = insertvalue [18 x i32] %mrv, i32 %add.u, 1
  %mrv2 = insertvalue [18 x i32] %mrv1, i32 %add.s, 2
  %mrv3 = insertvalue [18 x i32] %mrv2, i32 %add.us, 3
  %mrv4 = insertvalue [18 x i32] %mrv3, i32 %sub, 4
  %mrv5 = insertvalue [18 x i32] %mrv4, i32 %sub.u, 5
  %mrv6 = insertvalue [18 x i32] %mrv5, i32 %sub.s, 6
  %mrv7 = insertvalue [18 x i32] %mrv6, i32 %sub.us, 7
  %mrv8 = insertvalue [18 x i32] %mrv7, i32 %mul, 8
  %mrv9 = insertvalue [18 x i32] %mrv8, i32 %mul.u, 9
  %mrv10 = insertvalue [18 x i32] %mrv9, i32 %mul.s, 10
  %mrv11 = insertvalue [18 x i32] %mrv10, i32 %mul.us, 11
  %mrv12 = insertvalue [18 x i32] %mrv11, i32 %udiv, 12
  %mrv13 = insertvalue [18 x i32] %mrv12, i32 %udiv.exact, 13
  %mrv14 = insertvalue [18 x i32] %mrv13, i32 %sdiv, 14
  %mrv15 = insertvalue [18 x i32] %mrv14, i32 %sdiv.exact, 15
  %mrv16 = insertvalue [18 x i32] %mrv15, i32 %urem, 16
  %mrv17 = insertvalue [18 x i32] %mrv16, i32 %srem, 17
  ret [18 x i32] %mrv17
}
`

func TestBuilderIntCompare(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i1 := ctx.Int(1)
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("intCompare", llvm.Signature{
		Type: ctx.Function(ctx.Array(10, i1), false, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.Compare(llvm.IntEqual, x, y, "eq"),
		builder.Compare(llvm.IntNotEqual, x, y, "ne"),
		builder.Compare(llvm.IntUnsignedGreaterThan, x, y, "ugt"),
		builder.Compare(llvm.IntUnsignedGreaterOrEqual, x, y, "uge"),
		builder.Compare(llvm.IntUnsignedLessThan, x, y, "ult"),
		builder.Compare(llvm.IntUnsignedLessOrEqual, x, y, "ule"),
		builder.Compare(llvm.IntSignedGreaterThan, x, y, "sgt"),
		builder.Compare(llvm.IntSignedGreaterOrEqual, x, y, "sge"),
		builder.Compare(llvm.IntSignedLessThan, x, y, "slt"),
		builder.Compare(llvm.IntSignedLessOrEqual, x, y, "sle"),
	)

	// Compare the module string.
	if str := mod.String(); str != intCompareMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const intCompareMod = `
define [10 x i1] @intCompare(i32 %0, i32 %1) {
  %eq = icmp eq i32 %0, %1
  %ne = icmp ne i32 %0, %1
  %ugt = icmp ugt i32 %0, %1
  %uge = icmp uge i32 %0, %1
  %ult = icmp ult i32 %0, %1
  %ule = icmp ule i32 %0, %1
  %sgt = icmp sgt i32 %0, %1
  %sge = icmp sge i32 %0, %1
  %slt = icmp slt i32 %0, %1
  %sle = icmp sle i32 %0, %1
  %mrv = insertvalue [10 x i1] poison, i1 %eq, 0
  %mrv1 = insertvalue [10 x i1] %mrv, i1 %ne, 1
  %mrv2 = insertvalue [10 x i1] %mrv1, i1 %ugt, 2
  %mrv3 = insertvalue [10 x i1] %mrv2, i1 %uge, 3
  %mrv4 = insertvalue [10 x i1] %mrv3, i1 %ult, 4
  %mrv5 = insertvalue [10 x i1] %mrv4, i1 %ule, 5
  %mrv6 = insertvalue [10 x i1] %mrv5, i1 %sgt, 6
  %mrv7 = insertvalue [10 x i1] %mrv6, i1 %sge, 7
  %mrv8 = insertvalue [10 x i1] %mrv7, i1 %slt, 8
  %mrv9 = insertvalue [10 x i1] %mrv8, i1 %sle, 9
  ret [10 x i1] %mrv9
}
`

func TestBuildIntMinMax(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("minMax", llvm.Signature{
		Type: ctx.Function(ctx.Array(4, i32), false, i32, i32, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	a := fn.Param(0)
	b := fn.Param(1)
	c := fn.Param(2)
	d := fn.Param(3)
	builder.Return(
		builder.UnsignedMinimum("umin", a, b, c, d),
		builder.UnsignedMaximum("umax", a, b, c, d),
		builder.SignedMinimum("smin", a, b, c, d),
		builder.SignedMaximum("smax", a, b, c, d),
	)

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != intMinMaxFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const intMinMaxFunc = `define [4 x i32] @minMax(i32 %0, i32 %1, i32 %2, i32 %3) {
  %umin = call i32 @llvm.umin.i32(i32 %0, i32 %1)
  %umin1 = call i32 @llvm.umin.i32(i32 %umin, i32 %2)
  %umin2 = call i32 @llvm.umin.i32(i32 %umin1, i32 %3)
  %umax = call i32 @llvm.umax.i32(i32 %0, i32 %1)
  %umax3 = call i32 @llvm.umax.i32(i32 %umax, i32 %2)
  %umax4 = call i32 @llvm.umax.i32(i32 %umax3, i32 %3)
  %smin = call i32 @llvm.smin.i32(i32 %0, i32 %1)
  %smin5 = call i32 @llvm.smin.i32(i32 %smin, i32 %2)
  %smin6 = call i32 @llvm.smin.i32(i32 %smin5, i32 %3)
  %smax = call i32 @llvm.smax.i32(i32 %0, i32 %1)
  %smax7 = call i32 @llvm.smax.i32(i32 %smax, i32 %2)
  %smax8 = call i32 @llvm.smax.i32(i32 %smax7, i32 %3)
  %mrv = insertvalue [4 x i32] poison, i32 %umin2, 0
  %mrv9 = insertvalue [4 x i32] %mrv, i32 %umax4, 1
  %mrv10 = insertvalue [4 x i32] %mrv9, i32 %smin6, 2
  %mrv11 = insertvalue [4 x i32] %mrv10, i32 %smax8, 3
  ret [4 x i32] %mrv11
}`

func TestBuildShift(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("shift", llvm.Signature{
		Type: ctx.Function(ctx.Array(9, i32), false, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.BitShiftLeft(x, y, "shl", false, false),
		builder.BitShiftLeft(x, y, "shl.u", true, false),
		builder.BitShiftLeft(x, y, "shl.s", false, true),
		builder.BitShiftLeft(x, y, "shl.us", true, true),
		builder.LogicalShiftRight(x, y, "lshr", false),
		builder.LogicalShiftRight(x, y, "lshr.exact", true),
		builder.ArithmeticShiftRight(x, y, "ashr", false),
		builder.ArithmeticShiftRight(x, y, "ashr.exact", true),
		builder.FunnelShiftLeft(x, x, y, "rotl"),
	)

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != shiftFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const shiftFunc = `define [9 x i32] @shift(i32 %0, i32 %1) {
  %shl = shl i32 %0, %1
  %shl.u = shl nuw i32 %0, %1
  %shl.s = shl nsw i32 %0, %1
  %shl.us = shl nuw nsw i32 %0, %1
  %lshr = lshr i32 %0, %1
  %lshr.exact = lshr exact i32 %0, %1
  %ashr = ashr i32 %0, %1
  %ashr.exact = ashr exact i32 %0, %1
  %rotl = call i32 @llvm.fshl.i32(i32 %0, i32 %0, i32 %1)
  %mrv = insertvalue [9 x i32] poison, i32 %shl, 0
  %mrv1 = insertvalue [9 x i32] %mrv, i32 %shl.u, 1
  %mrv2 = insertvalue [9 x i32] %mrv1, i32 %shl.s, 2
  %mrv3 = insertvalue [9 x i32] %mrv2, i32 %shl.us, 3
  %mrv4 = insertvalue [9 x i32] %mrv3, i32 %lshr, 4
  %mrv5 = insertvalue [9 x i32] %mrv4, i32 %lshr.exact, 5
  %mrv6 = insertvalue [9 x i32] %mrv5, i32 %ashr, 6
  %mrv7 = insertvalue [9 x i32] %mrv6, i32 %ashr.exact, 7
  %mrv8 = insertvalue [9 x i32] %mrv7, i32 %rotl, 8
  ret [9 x i32] %mrv8
}`

func TestBitwise(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("bitwise", llvm.Signature{
		Type: ctx.Function(ctx.Array(3, i32), false, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.And(x, y, "and"),
		builder.Or(x, y, "or", false),
		builder.Xor(x, y, "xor"),
	)

	// Compare the module string.
	if str := mod.String(); str != bitwiseMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const bitwiseMod = `
define [3 x i32] @bitwise(i32 %0, i32 %1) {
  %and = and i32 %0, %1
  %or = or i32 %0, %1
  %xor = xor i32 %0, %1
  %mrv = insertvalue [3 x i32] poison, i32 %and, 0
  %mrv1 = insertvalue [3 x i32] %mrv, i32 %or, 1
  %mrv2 = insertvalue [3 x i32] %mrv1, i32 %xor, 2
  ret [3 x i32] %mrv2
}
`

func TestOrDisjoint(t *testing.T) {
	if llvm.VersionMajor < 21 {
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("orDisjoint", llvm.Signature{
		Type: ctx.Function(i32, false, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.Or(x, y, "or.disjoint", true),
	)

	// Compare the module string.
	if str := mod.String(); str != orDisjointMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const orDisjointMod = `
define i32 @orDisjoint(i32 %0, i32 %1) {
  %or.disjoint = or disjoint i32 %0, %1
  ret i32 %or.disjoint
}
`

func TestBitIntrinsics(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("bitIntrinsics", llvm.Signature{
		Type: ctx.Function(ctx.Array(7, i32), false, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	arg := fn.Param(0)
	builder.Return(
		builder.BitReverse(arg, "bitrev"),
		builder.ByteReverse(arg, "byterev"),
		builder.CountOnes(arg, "countones"),
		builder.CountLeadingZeroes(arg, "clz", false),
		builder.CountLeadingZeroes(arg, "clz.nz", true),
		builder.CountTrailingZeroes(arg, "ctz", false),
		builder.CountTrailingZeroes(arg, "ctz.nz", true),
	)

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != bitIntrinsicsFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const bitIntrinsicsFunc = `define [7 x i32] @bitIntrinsics(i32 %0) {
  %bitrev = call i32 @llvm.bitreverse.i32(i32 %0)
  %byterev = call i32 @llvm.bswap.i32(i32 %0)
  %countones = call i32 @llvm.ctpop.i32(i32 %0)
  %clz = call i32 @llvm.ctlz.i32(i32 %0, i1 false)
  %clz.nz = call i32 @llvm.ctlz.i32(i32 %0, i1 true)
  %ctz = call i32 @llvm.cttz.i32(i32 %0, i1 false)
  %ctz.nz = call i32 @llvm.cttz.i32(i32 %0, i1 true)
  %mrv = insertvalue [7 x i32] poison, i32 %bitrev, 0
  %mrv1 = insertvalue [7 x i32] %mrv, i32 %byterev, 1
  %mrv2 = insertvalue [7 x i32] %mrv1, i32 %countones, 2
  %mrv3 = insertvalue [7 x i32] %mrv2, i32 %clz, 3
  %mrv4 = insertvalue [7 x i32] %mrv3, i32 %clz.nz, 4
  %mrv5 = insertvalue [7 x i32] %mrv4, i32 %ctz, 5
  %mrv6 = insertvalue [7 x i32] %mrv5, i32 %ctz.nz, 6
  ret [7 x i32] %mrv6
}`

func TestFloatConvert(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	f32 := ctx.Float32()
	f64 := ctx.Float64()
	i8 := ctx.Int(8)
	fn := mod.CreateFunction("floatConvert", llvm.Signature{
		Type: ctx.Function(ctx.LiteralStruct(
			f32, f64,
			f32, f32, f32,
			i8, i8,
		), false, f32, f64, i8),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	f32Arg := fn.Param(0)
	f64Arg := fn.Param(1)
	i8Arg := fn.Param(2)
	builder.Return(
		builder.FloatTrunc(f64Arg, f32, "fptrunc"),
		builder.FloatExtend(f32Arg, f64, "fpext"),
		builder.UnsignedIntToFloat(i8Arg, f32, "u8tof32", false),
		builder.UnsignedIntToFloat(i8Arg, f32, "u8tof32.nneg", true),
		builder.SignedIntToFloat(i8Arg, f32, "s8tof32"),
		builder.FloatToUnsignedIntSaturating(f32Arg, i8, "f32tou8"),
		builder.FloatToSignedIntSaturating(f32Arg, i8, "f32tos8"),
	)

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != floatConvertFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const floatConvertFunc = `define { float, double, float, float, float, i8, i8 } @floatConvert(float %0, double %1, i8 %2) {
  %fptrunc = fptrunc double %1 to float
  %fpext = fpext float %0 to double
  %u8tof32 = uitofp i8 %2 to float
  %u8tof32.nneg = uitofp nneg i8 %2 to float
  %s8tof32 = sitofp i8 %2 to float
  %f32tou8 = call i8 @llvm.fptoui.sat.i8.f32(float %0)
  %f32tos8 = call i8 @llvm.fptosi.sat.i8.f32(float %0)
  %mrv = insertvalue { float, double, float, float, float, i8, i8 } poison, float %fptrunc, 0
  %mrv1 = insertvalue { float, double, float, float, float, i8, i8 } %mrv, double %fpext, 1
  %mrv2 = insertvalue { float, double, float, float, float, i8, i8 } %mrv1, float %u8tof32, 2
  %mrv3 = insertvalue { float, double, float, float, float, i8, i8 } %mrv2, float %u8tof32.nneg, 3
  %mrv4 = insertvalue { float, double, float, float, float, i8, i8 } %mrv3, float %s8tof32, 4
  %mrv5 = insertvalue { float, double, float, float, float, i8, i8 } %mrv4, i8 %f32tou8, 5
  %mrv6 = insertvalue { float, double, float, float, float, i8, i8 } %mrv5, i8 %f32tos8, 6
  ret { float, double, float, float, float, i8, i8 } %mrv6
}`

func TestBuilderFloatMath(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	f32 := ctx.Float32()
	fn := mod.CreateFunction("floatMath", llvm.Signature{
		Type: ctx.Function(ctx.Array(6, f32), false, f32, f32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.FloatNegate(x, "neg"),
		builder.FloatAdd(x, y, "add"),
		builder.FloatSubtract(x, y, "sub"),
		builder.FloatMultiply(x, y, "mul"),
		builder.FloatDivide(x, y, "div"),
		builder.FloatRemainder(x, y, "rem"),
	)

	// Compare the module string.
	if str := mod.String(); str != floatMathMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const floatMathMod = `
define [6 x float] @floatMath(float %0, float %1) {
  %neg = fneg float %0
  %add = fadd float %0, %1
  %sub = fsub float %0, %1
  %mul = fmul float %0, %1
  %div = fdiv float %0, %1
  %rem = frem float %0, %1
  %mrv = insertvalue [6 x float] poison, float %neg, 0
  %mrv1 = insertvalue [6 x float] %mrv, float %add, 1
  %mrv2 = insertvalue [6 x float] %mrv1, float %sub, 2
  %mrv3 = insertvalue [6 x float] %mrv2, float %mul, 3
  %mrv4 = insertvalue [6 x float] %mrv3, float %div, 4
  %mrv5 = insertvalue [6 x float] %mrv4, float %rem, 5
  ret [6 x float] %mrv5
}
`

func TestBuilderFloatCompare(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i1 := ctx.Int(1)
	f32 := ctx.Float32()
	fn := mod.CreateFunction("floatCompare", llvm.Signature{
		Type: ctx.Function(ctx.Array(14, i1), false, f32, f32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	x := fn.Param(0)
	y := fn.Param(1)
	builder.Return(
		builder.FloatCompare(llvm.FloatEqual, x, y, "oeq"),
		builder.FloatCompare(llvm.FloatGreater, x, y, "ogt"),
		builder.FloatCompare(llvm.FloatGreater|llvm.FloatEqual, x, y, "oge"),
		builder.FloatCompare(llvm.FloatLess, x, y, "olt"),
		builder.FloatCompare(llvm.FloatLess|llvm.FloatEqual, x, y, "ole"),
		builder.FloatCompare(llvm.FloatLess|llvm.FloatGreater, x, y, "one"),
		builder.FloatCompare(llvm.FloatNaN.Not(), x, y, "ord"),
		builder.FloatCompare(llvm.FloatNaN, x, y, "uno"),
		builder.FloatCompare(llvm.FloatEqual|llvm.FloatNaN, x, y, "ueq"),
		builder.FloatCompare(llvm.FloatGreater|llvm.FloatNaN, x, y, "ugt"),
		builder.FloatCompare(llvm.FloatLess.Not(), x, y, "uge"),
		builder.FloatCompare(llvm.FloatLess|llvm.FloatNaN, x, y, "ult"),
		builder.FloatCompare(llvm.FloatGreater.Not(), x, y, "ule"),
		builder.FloatCompare(llvm.FloatEqual.Not(), x, y, "une"),
	)

	// Compare the module string.
	if str := mod.String(); str != floatCompareMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const floatCompareMod = `
define [14 x i1] @floatCompare(float %0, float %1) {
  %oeq = fcmp oeq float %0, %1
  %ogt = fcmp ogt float %0, %1
  %oge = fcmp oge float %0, %1
  %olt = fcmp olt float %0, %1
  %ole = fcmp ole float %0, %1
  %one = fcmp one float %0, %1
  %ord = fcmp ord float %0, %1
  %uno = fcmp uno float %0, %1
  %ueq = fcmp ueq float %0, %1
  %ugt = fcmp ugt float %0, %1
  %uge = fcmp uge float %0, %1
  %ult = fcmp ult float %0, %1
  %ule = fcmp ule float %0, %1
  %une = fcmp une float %0, %1
  %mrv = insertvalue [14 x i1] poison, i1 %oeq, 0
  %mrv1 = insertvalue [14 x i1] %mrv, i1 %ogt, 1
  %mrv2 = insertvalue [14 x i1] %mrv1, i1 %oge, 2
  %mrv3 = insertvalue [14 x i1] %mrv2, i1 %olt, 3
  %mrv4 = insertvalue [14 x i1] %mrv3, i1 %ole, 4
  %mrv5 = insertvalue [14 x i1] %mrv4, i1 %one, 5
  %mrv6 = insertvalue [14 x i1] %mrv5, i1 %ord, 6
  %mrv7 = insertvalue [14 x i1] %mrv6, i1 %uno, 7
  %mrv8 = insertvalue [14 x i1] %mrv7, i1 %ueq, 8
  %mrv9 = insertvalue [14 x i1] %mrv8, i1 %ugt, 9
  %mrv10 = insertvalue [14 x i1] %mrv9, i1 %uge, 10
  %mrv11 = insertvalue [14 x i1] %mrv10, i1 %ult, 11
  %mrv12 = insertvalue [14 x i1] %mrv11, i1 %ule, 12
  %mrv13 = insertvalue [14 x i1] %mrv12, i1 %une, 13
  ret [14 x i1] %mrv13
}
`

func TestBuildFloatMinMax(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	f32 := ctx.Float32()
	fn := mod.CreateFunction("floatMinMax", llvm.Signature{
		Type: ctx.Function(ctx.Array(2, f32), false, f32, f32, f32, f32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	a := fn.Param(0)
	b := fn.Param(1)
	c := fn.Param(2)
	d := fn.Param(3)
	builder.Return(
		builder.FloatMinimum("min", a, b, c, d),
		builder.FloatMaximum("max", a, b, c, d),
	)

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != floatMinMaxFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const floatMinMaxFunc = `define [2 x float] @floatMinMax(float %0, float %1, float %2, float %3) {
  %min = call float @llvm.minimum.f32(float %0, float %1)
  %min1 = call float @llvm.minimum.f32(float %min, float %2)
  %min2 = call float @llvm.minimum.f32(float %min1, float %3)
  %max = call float @llvm.maximum.f32(float %0, float %1)
  %max3 = call float @llvm.maximum.f32(float %max, float %2)
  %max4 = call float @llvm.maximum.f32(float %max3, float %3)
  %mrv = insertvalue [2 x float] poison, float %min2, 0
  %mrv5 = insertvalue [2 x float] %mrv, float %max4, 1
  ret [2 x float] %mrv5
}`

func TestBuildOtherCasts(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	f32 := ctx.Float32()
	fn := mod.CreateFunction("casts", llvm.Signature{
		Type: ctx.Function(ctx.LiteralStruct(i32, ptr, f32), false, ptr, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	ptrArg := fn.Param(0)
	i32Arg := fn.Param(1)
	builder.Return(
		builder.PtrToInt(ptrArg, i32, "ptrtoint"),
		builder.IntToPtr(i32Arg, ptr, "inttoptr"),
		builder.BitCast(i32Arg, f32, "bitcast"),
	)

	// Compare the module string.
	if str := mod.String(); str != otherCastsMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const otherCastsMod = `
define { i32, ptr, float } @casts(ptr %0, i32 %1) {
  %ptrtoint = ptrtoint ptr %0 to i32
  %inttoptr = inttoptr i32 %1 to ptr
  %bitcast = bitcast i32 %1 to float
  %mrv = insertvalue { i32, ptr, float } poison, i32 %ptrtoint, 0
  %mrv1 = insertvalue { i32, ptr, float } %mrv, ptr %inttoptr, 1
  %mrv2 = insertvalue { i32, ptr, float } %mrv1, float %bitcast, 2
  ret { i32, ptr, float } %mrv2
}
`

// TODO: memory

func TestMem(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	i8 := ctx.Int(8)
	arr := ctx.Array(4, i8)
	fn := mod.CreateFunction("memCast", llvm.Signature{
		Type: ctx.Function(arr, false, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	alloca := builder.StaticAlloca(i32, 0, llvm.ImplicitAlignment, "tmp")
	builder.Store(fn.Param(0), alloca.Value, llvm.MemOptions{})
	castLoad := builder.Load(arr, alloca.Value, llvm.MemOptions{}, "load")
	builder.Return(castLoad)

	// Compare the module string.
	if str := mod.String(); str != memCastMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const memCastMod = `
define [4 x i8] @memCast(i32 %0) {
  %tmp = alloca i32, align 4
  store i32 %0, ptr %tmp, align 4
  %load = load [4 x i8], ptr %tmp, align 1
  ret [4 x i8] %load
}
`

func TestAtomicLoad(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	ptr := ctx.Pointer(0)
	fn := mod.CreateFunction("atomicLoad", llvm.Signature{
		Type: ctx.Function(ctx.Array(7, i32), false, ptr),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	arg := fn.Param(0)
	// Try all valid memory orders to verify that they are usable.
	builder.Return(
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 4,
			Order:     llvm.MemOrderUnordered,
		}, "load.unordered"),
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 4,
			Order:     llvm.MemOrderMonotonic,
		}, "load.monotonic"),
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 4,
			Order:     llvm.MemOrderAcquire,
		}, "load.acquire"),
		// MemOrderRelease/MemOrderAcquireRelease are invalid for loads
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 4,
			Order:     llvm.MemOrderSequentiallyConsistent,
		}, "load.seq_cst"),
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 4,
			Volatile:  true,
		}, "load.volatile"),
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 4,
			Order:     llvm.MemOrderSequentiallyConsistent,
			Volatile:  true,
		}, "load.seq_cst.volatile"),
		builder.Load(i32, arg, llvm.MemOptions{
			Alignment: 1,
			Order:     llvm.MemOrderSequentiallyConsistent,
		}, "load.seq_cst.unaligned"),
	)

	// Compare the module string.
	if str := mod.String(); str != atomicLoadMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const atomicLoadMod = `
define [7 x i32] @atomicLoad(ptr %0) {
  %load.unordered = load atomic i32, ptr %0 unordered, align 4
  %load.monotonic = load atomic i32, ptr %0 monotonic, align 4
  %load.acquire = load atomic i32, ptr %0 acquire, align 4
  %load.seq_cst = load atomic i32, ptr %0 seq_cst, align 4
  %load.volatile = load volatile i32, ptr %0, align 4
  %load.seq_cst.volatile = load atomic volatile i32, ptr %0 seq_cst, align 4
  %load.seq_cst.unaligned = load atomic i32, ptr %0 seq_cst, align 1
  %mrv = insertvalue [7 x i32] poison, i32 %load.unordered, 0
  %mrv1 = insertvalue [7 x i32] %mrv, i32 %load.monotonic, 1
  %mrv2 = insertvalue [7 x i32] %mrv1, i32 %load.acquire, 2
  %mrv3 = insertvalue [7 x i32] %mrv2, i32 %load.seq_cst, 3
  %mrv4 = insertvalue [7 x i32] %mrv3, i32 %load.volatile, 4
  %mrv5 = insertvalue [7 x i32] %mrv4, i32 %load.seq_cst.volatile, 5
  %mrv6 = insertvalue [7 x i32] %mrv5, i32 %load.seq_cst.unaligned, 6
  ret [7 x i32] %mrv6
}
`

func TestCASLoop(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	ptr := ctx.Pointer(0)
	fn := mod.CreateFunction("atomicMul", llvm.Signature{
		Type: ctx.Function(i32, false, ptr, i32),
		Attributes: ctx.AttributeList(
			llvm.AttributeSet{},
			llvm.AttributeSet{},
			ctx.AttributeSet(
				ctx.IntAttribute(llvm.AttributeAlign, 4),
				ctx.IntAttribute(llvm.AttributeDereferenceable, 4),
			),
		),
	}, 0, llvm.LinkConfig{})

	// Create the entry block.
	builder := ctx.Builder()
	defer builder.Destroy()
	entry := fn.AppendBasicBlock("entry")
	builder.AtEnd(entry)
	dst := fn.Param(0)
	initialLoad := builder.Load(i32, dst, llvm.MemOptions{
		Alignment: 4,
		Order:     llvm.MemOrderSequentiallyConsistent,
	}, "initial")
	loop, exit := builder.If(
		builder.Compare(llvm.IntNotEqual, initialLoad, ctx.ConstInt(32, 0).Value, "initial.nez"),
		"loop",
		"ret",
	)

	// Populate the loop body block.
	prev := builder.Phi(i32, "prev", llvm.PhiIncoming{initialLoad, entry})
	cas := builder.CompareAndExchange(
		dst,
		prev.Value,
		builder.Multiply(prev.Value, fn.Param(1), "mul", false, false),
		llvm.MemOrderSequentiallyConsistent,
		llvm.MemOrderMonotonic,
		llvm.CompareAndExchangeOptions{
			Alignment: 4,
			Weak:      true,
		},
		"cas",
	)
	old := builder.ExtractValue(cas, "cas.old", 0)
	builder.Branch(builder.ExtractValue(cas, "cas.ok", 1), loop, exit)
	prev.Add(llvm.PhiIncoming{old, loop})

	// Populate the exit block.
	builder.AtEnd(exit)
	builder.Return(builder.Phi(i32, "old",
		llvm.PhiIncoming{ctx.ConstInt(32, 0).Value, entry},
		llvm.PhiIncoming{prev.Value, loop},
	).Value)

	// Compare the module string.
	if str := mod.String(); str != atomicMulMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const atomicMulMod = `
define i32 @atomicMul(ptr align 4 dereferenceable(4) %0, i32 %1) {
entry:
  %initial = load atomic i32, ptr %0 seq_cst, align 4
  %initial.nez = icmp ne i32 %initial, 0
  br i1 %initial.nez, label %loop, label %ret

loop:                                             ; preds = %loop, %entry
  %prev = phi i32 [ %initial, %entry ], [ %cas.old, %loop ]
  %mul = mul i32 %prev, %1
  %cas = cmpxchg weak ptr %0, i32 %prev, i32 %mul seq_cst monotonic, align 4
  %cas.old = extractvalue { i32, i1 } %cas, 0
  %cas.ok = extractvalue { i32, i1 } %cas, 1
  br i1 %cas.ok, label %loop, label %ret

ret:                                              ; preds = %loop, %entry
  %old = phi i32 [ 0, %entry ], [ %prev, %loop ]
  ret i32 %old
}
`

func TestAtomicRMW(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	f32 := ctx.Float32()
	intResults := ctx.Array(20, i32)
	floatResults := ctx.Array(4, f32)
	fn := mod.CreateFunction("atomicrmw", llvm.Signature{
		Type: ctx.Function(
			ctx.LiteralStruct(intResults, floatResults),
			false,
			ctx.Pointer(0), i32, f32,
		),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	ptr := fn.Param(0)
	intArg := fn.Param(1)
	floatArg := fn.Param(2)
	builder.Return(
		// Test int operators.
		builder.BuildAggregate(intResults, "ret.ints",
			// Test all universally-supported operators.
			// Newer operators must be tested seperately behind version checks.
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{}, "swap"),
			builder.AtomicRMW(llvm.AtomicAdd, ptr, intArg, llvm.MemOptions{}, "add"),
			builder.AtomicRMW(llvm.AtomicSub, ptr, intArg, llvm.MemOptions{}, "sub"),
			builder.AtomicRMW(llvm.AtomicAnd, ptr, intArg, llvm.MemOptions{}, "and"),
			builder.AtomicRMW(llvm.AtomicNotAnd, ptr, intArg, llvm.MemOptions{}, "nand"),
			builder.AtomicRMW(llvm.AtomicOr, ptr, intArg, llvm.MemOptions{}, "or"),
			builder.AtomicRMW(llvm.AtomicXOr, ptr, intArg, llvm.MemOptions{}, "xor"),
			builder.AtomicRMW(llvm.AtomicSignedMax, ptr, intArg, llvm.MemOptions{}, "smax"),
			builder.AtomicRMW(llvm.AtomicSignedMin, ptr, intArg, llvm.MemOptions{}, "smin"),
			builder.AtomicRMW(llvm.AtomicUnsignedMax, ptr, intArg, llvm.MemOptions{}, "umax"),
			builder.AtomicRMW(llvm.AtomicUnsignedMin, ptr, intArg, llvm.MemOptions{}, "umin"),
			// Test all memory orders
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order: llvm.MemOrderUnordered,
			}, "swap.unordered"),
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order: llvm.MemOrderMonotonic,
			}, "swap.monotonic"), builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order: llvm.MemOrderAcquire,
			}, "swap.acquire"),
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order: llvm.MemOrderRelease,
			}, "swap.release"),
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order: llvm.MemOrderAcquireRelease,
			}, "swap.acq_rel"),
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order: llvm.MemOrderSequentiallyConsistent,
			}, "swap.seq_cst"),
			// Test volatile access.
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Volatile: true,
			}, "swap.volatile"),
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Order:    llvm.MemOrderSequentiallyConsistent,
				Volatile: true,
			}, "swap.seq_cst.volatile"),
			// Test unaligned access.
			builder.AtomicRMW(llvm.AtomicSwap, ptr, intArg, llvm.MemOptions{
				Alignment: 1,
			}, "swap.unaligned"),
		),
		// Test float operators.
		builder.BuildAggregate(floatResults, "ret.floats",
			// Test all universally-supported operators.
			// Newer operators must be tested seperately behind version checks.
			builder.AtomicRMW(llvm.AtomicFloatAdd, ptr, floatArg, llvm.MemOptions{}, "fadd"),
			builder.AtomicRMW(llvm.AtomicFloatSub, ptr, floatArg, llvm.MemOptions{}, "fsub"),
			builder.AtomicRMW(llvm.AtomicFloatMaxNum, ptr, floatArg, llvm.MemOptions{}, "fmaxnum"),
			builder.AtomicRMW(llvm.AtomicFloatMinNum, ptr, floatArg, llvm.MemOptions{}, "fminnum"),
		),
	)

	// Compare the module string.
	if str := mod.String(); str != atomicRMWMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const atomicRMWMod = `
define { [20 x i32], [4 x float] } @atomicrmw(ptr %0, i32 %1, float %2) {
  %swap = atomicrmw xchg ptr %0, i32 %1, align 4
  %add = atomicrmw add ptr %0, i32 %1, align 4
  %sub = atomicrmw sub ptr %0, i32 %1, align 4
  %and = atomicrmw and ptr %0, i32 %1, align 4
  %nand = atomicrmw nand ptr %0, i32 %1, align 4
  %or = atomicrmw or ptr %0, i32 %1, align 4
  %xor = atomicrmw xor ptr %0, i32 %1, align 4
  %smax = atomicrmw max ptr %0, i32 %1, align 4
  %smin = atomicrmw min ptr %0, i32 %1, align 4
  %umax = atomicrmw umax ptr %0, i32 %1, align 4
  %umin = atomicrmw umin ptr %0, i32 %1, align 4
  %swap.unordered = atomicrmw xchg ptr %0, i32 %1 unordered, align 4
  %swap.monotonic = atomicrmw xchg ptr %0, i32 %1 monotonic, align 4
  %swap.acquire = atomicrmw xchg ptr %0, i32 %1 acquire, align 4
  %swap.release = atomicrmw xchg ptr %0, i32 %1 release, align 4
  %swap.acq_rel = atomicrmw xchg ptr %0, i32 %1 acq_rel, align 4
  %swap.seq_cst = atomicrmw xchg ptr %0, i32 %1 seq_cst, align 4
  %swap.volatile = atomicrmw volatile xchg ptr %0, i32 %1, align 4
  %swap.seq_cst.volatile = atomicrmw volatile xchg ptr %0, i32 %1 seq_cst, align 4
  %swap.unaligned = atomicrmw xchg ptr %0, i32 %1, align 1
  %ret.ints = insertvalue [20 x i32] poison, i32 %swap, 0
  %ret.ints1 = insertvalue [20 x i32] %ret.ints, i32 %add, 1
  %ret.ints2 = insertvalue [20 x i32] %ret.ints1, i32 %sub, 2
  %ret.ints3 = insertvalue [20 x i32] %ret.ints2, i32 %and, 3
  %ret.ints4 = insertvalue [20 x i32] %ret.ints3, i32 %nand, 4
  %ret.ints5 = insertvalue [20 x i32] %ret.ints4, i32 %or, 5
  %ret.ints6 = insertvalue [20 x i32] %ret.ints5, i32 %xor, 6
  %ret.ints7 = insertvalue [20 x i32] %ret.ints6, i32 %smax, 7
  %ret.ints8 = insertvalue [20 x i32] %ret.ints7, i32 %smin, 8
  %ret.ints9 = insertvalue [20 x i32] %ret.ints8, i32 %umax, 9
  %ret.ints10 = insertvalue [20 x i32] %ret.ints9, i32 %umin, 10
  %ret.ints11 = insertvalue [20 x i32] %ret.ints10, i32 %swap.unordered, 11
  %ret.ints12 = insertvalue [20 x i32] %ret.ints11, i32 %swap.monotonic, 12
  %ret.ints13 = insertvalue [20 x i32] %ret.ints12, i32 %swap.acquire, 13
  %ret.ints14 = insertvalue [20 x i32] %ret.ints13, i32 %swap.release, 14
  %ret.ints15 = insertvalue [20 x i32] %ret.ints14, i32 %swap.acq_rel, 15
  %ret.ints16 = insertvalue [20 x i32] %ret.ints15, i32 %swap.seq_cst, 16
  %ret.ints17 = insertvalue [20 x i32] %ret.ints16, i32 %swap.volatile, 17
  %ret.ints18 = insertvalue [20 x i32] %ret.ints17, i32 %swap.seq_cst.volatile, 18
  %ret.ints19 = insertvalue [20 x i32] %ret.ints18, i32 %swap.unaligned, 19
  %fadd = atomicrmw fadd ptr %0, float %2, align 4
  %fsub = atomicrmw fsub ptr %0, float %2, align 4
  %fmaxnum = atomicrmw fmax ptr %0, float %2, align 4
  %fminnum = atomicrmw fmin ptr %0, float %2, align 4
  %ret.floats = insertvalue [4 x float] poison, float %fadd, 0
  %ret.floats20 = insertvalue [4 x float] %ret.floats, float %fsub, 1
  %ret.floats21 = insertvalue [4 x float] %ret.floats20, float %fmaxnum, 2
  %ret.floats22 = insertvalue [4 x float] %ret.floats21, float %fminnum, 3
  %mrv = insertvalue { [20 x i32], [4 x float] } poison, [20 x i32] %ret.ints19, 0
  %mrv23 = insertvalue { [20 x i32], [4 x float] } %mrv, [4 x float] %ret.floats22, 1
  ret { [20 x i32], [4 x float] } %mrv23
}
`

func TestAtomicRMW_LLVM16(t *testing.T) {
	if llvm.VersionMajor < 16 {
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("atomicrmw", llvm.Signature{
		Type: ctx.Function(ctx.Array(2, i32), false, ctx.Pointer(0), i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	ptr := fn.Param(0)
	arg := fn.Param(1)
	builder.Return(
		builder.AtomicRMW(llvm.AtomicUnsignedIncrementAndWrap, ptr, arg, llvm.MemOptions{}, "uinc_wrap"),
		builder.AtomicRMW(llvm.AtomicUnsignedDecrementAndWrap, ptr, arg, llvm.MemOptions{}, "udec_wrap"),
	)

	// Compare the module string.
	if str := mod.String(); str != atomicRMWLLVM16Mod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const atomicRMWLLVM16Mod = `
define [2 x i32] @atomicrmw(ptr %0, i32 %1) {
  %uinc_wrap = atomicrmw uinc_wrap ptr %0, i32 %1, align 4
  %udec_wrap = atomicrmw udec_wrap ptr %0, i32 %1, align 4
  %mrv = insertvalue [2 x i32] poison, i32 %uinc_wrap, 0
  %mrv1 = insertvalue [2 x i32] %mrv, i32 %udec_wrap, 1
  ret [2 x i32] %mrv1
}
`

func TestAtomicRMW_LLVM20(t *testing.T) {
	if llvm.VersionMajor < 20 {
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("atomicrmw", llvm.Signature{
		Type: ctx.Function(ctx.Array(2, i32), false, ctx.Pointer(0), i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	ptr := fn.Param(0)
	arg := fn.Param(1)
	builder.Return(
		builder.AtomicRMW(llvm.AtomicUnsignedSubtractIfNoUnderflow, ptr, arg, llvm.MemOptions{}, "usub_cond"),
		builder.AtomicRMW(llvm.AtomicUnsignedSubtractSaturating, ptr, arg, llvm.MemOptions{}, "usub_sat"),
	)

	// Compare the module string.
	if str := mod.String(); str != atomicRMWLLVM20Mod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const atomicRMWLLVM20Mod = `
define [2 x i32] @atomicrmw(ptr %0, i32 %1) {
  %usub_cond = atomicrmw usub_cond ptr %0, i32 %1, align 4
  %usub_sat = atomicrmw usub_sat ptr %0, i32 %1, align 4
  %mrv = insertvalue [2 x i32] poison, i32 %usub_cond, 0
  %mrv1 = insertvalue [2 x i32] %mrv, i32 %usub_sat, 1
  ret [2 x i32] %mrv1
}
`

func TestAtomicRMW_LLVM21(t *testing.T) {
	if llvm.VersionMajor < 21 {
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	f32 := ctx.Float32()
	fn := mod.CreateFunction("atomicrmw", llvm.Signature{
		Type: ctx.Function(ctx.Array(2, f32), false, ctx.Pointer(0), f32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	ptr := fn.Param(0)
	arg := fn.Param(1)
	builder.Return(
		builder.AtomicRMW(llvm.AtomicFloatMaximum, ptr, arg, llvm.MemOptions{}, "fmaximum"),
		builder.AtomicRMW(llvm.AtomicFloatMinimum, ptr, arg, llvm.MemOptions{}, "fminimum"),
	)

	// Compare the module string.
	if str := mod.String(); str != atomicRMWLLVM21Mod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const atomicRMWLLVM21Mod = `
define [2 x float] @atomicrmw(ptr %0, float %1) {
  %fmaximum = atomicrmw fmaximum ptr %0, float %1, align 4
  %fminimum = atomicrmw fminimum ptr %0, float %1, align 4
  %mrv = insertvalue [2 x float] poison, float %fmaximum, 0
  %mrv1 = insertvalue [2 x float] %mrv, float %fminimum, 1
  ret [2 x float] %mrv1
}
`

func TestBuilder_MemSet(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	fn := mod.CreateFunction("testMemSet", llvm.Signature{
		Type: ctx.Function(ctx.Void(), false, ctx.Pointer(0)),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	dst := fn.Param(0)
	builder.MemSet(dst, ctx.ConstInt(8, 0).Value, ctx.ConstInt(32, 8).Value, llvm.ImplicitAlignment, false)
	builder.MemSet(dst, ctx.ConstInt(8, 12).Value, ctx.ConstInt(32, 128).Value, 16, true)
	builder.Return()

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != testMemSetFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const testMemSetFunc = `define void @testMemSet(ptr %0) {
  call void @llvm.memset.p0.i32(ptr %0, i8 0, i32 8, i1 false)
  call void @llvm.memset.p0.i32(ptr align 16 %0, i8 12, i32 128, i1 true)
  ret void
}`

func TestBuilder_MemCopy(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	ptr := ctx.Pointer(0)
	fn := mod.CreateFunction("testMemCopy", llvm.Signature{
		Type: ctx.Function(ctx.Void(), false, ptr, ptr, ctx.Int(32)),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	dst := fn.Param(0)
	src := fn.Param(1)
	len := fn.Param(2)
	builder.MemCopy(
		dst, llvm.ImplicitAlignment,
		src, llvm.ImplicitAlignment,
		len, false,
	)
	builder.MemCopy(
		dst, 2,
		src, 8,
		len, true,
	)
	builder.Return()

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != testMemCopyFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const testMemCopyFunc = `define void @testMemCopy(ptr %0, ptr %1, i32 %2) {
  call void @llvm.memcpy.p0.p0.i32(ptr %0, ptr %1, i32 %2, i1 false)
  call void @llvm.memcpy.p0.p0.i32(ptr align 2 %0, ptr align 8 %1, i32 %2, i1 true)
  ret void
}`

func TestBuilder_MemMove(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function with no arguments that returns void.
	ptr := ctx.Pointer(0)
	fn := mod.CreateFunction("testMemMove", llvm.Signature{
		Type: ctx.Function(ctx.Void(), false, ptr, ptr, ctx.Int(32)),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	dst := fn.Param(0)
	src := fn.Param(1)
	len := fn.Param(2)
	builder.MemMove(
		dst, llvm.ImplicitAlignment,
		src, llvm.ImplicitAlignment,
		len, false,
	)
	builder.MemMove(
		dst, 2,
		src, 8,
		len, true,
	)
	builder.Return()

	// Compare the function string.
	// The attribute list is not stable across versions.
	if str := fn.LongString(); str != testMemMoveFunc {
		t.Errorf("unexpected func string:\n%s", str)
	}
}

const testMemMoveFunc = `define void @testMemMove(ptr %0, ptr %1, i32 %2) {
  call void @llvm.memmove.p0.p0.i32(ptr %0, ptr %1, i32 %2, i1 false)
  call void @llvm.memmove.p0.p0.i32(ptr align 2 %0, ptr align 8 %1, i32 %2, i1 true)
  ret void
}`

func TestBuilder_BuildAggregate(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	slice := ctx.NamedStruct("runtime._slice", ptr, i32, i32)
	fn := mod.CreateFunction("buildAggregate", llvm.Signature{
		Type: ctx.Function(slice, false, ptr, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	builder.Return(builder.BuildAggregate(slice, "slice",
		fn.Param(0), fn.Param(1), fn.Param(2),
	))

	// Compare the module string.
	if str := mod.String(); str != buildAggregateMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const buildAggregateMod = `
%runtime._slice = type { ptr, i32, i32 }

define %runtime._slice @buildAggregate(ptr %0, i32 %1, i32 %2) {
  %slice = insertvalue %runtime._slice poison, ptr %0, 0
  %slice1 = insertvalue %runtime._slice %slice, i32 %1, 1
  %slice2 = insertvalue %runtime._slice %slice1, i32 %2, 2
  ret %runtime._slice %slice2
}
`

func TestBuilder_InsertValue(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	slice := ctx.NamedStruct("runtime._slice", ptr, i32, i32)
	fn := mod.CreateFunction("insertvalue", llvm.Signature{
		Type: ctx.Function(slice, false, ptr, i32, i32),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	s := slice.Poison().Value
	s = builder.InsertValue(s, fn.Param(0), "slice.0", 0)
	s = builder.InsertValue(s, fn.Param(1), "slice.1", 1)
	s = builder.InsertValue(s, fn.Param(2), "slice.2", 2)
	builder.Return(s)

	// Compare the module string.
	if str := mod.String(); str != insertValueMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const insertValueMod = `
%runtime._slice = type { ptr, i32, i32 }

define %runtime._slice @insertvalue(ptr %0, i32 %1, i32 %2) {
  %slice.0 = insertvalue %runtime._slice poison, ptr %0, 0
  %slice.1 = insertvalue %runtime._slice %slice.0, i32 %1, 1
  %slice.2 = insertvalue %runtime._slice %slice.1, i32 %2, 2
  ret %runtime._slice %slice.2
}
`

func TestBuilder_ExtractValue(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	slice := ctx.NamedStruct("runtime._slice", ptr, i32, i32)
	fn := mod.CreateFunction("getLen", llvm.Signature{
		Type: ctx.Function(i32, false, slice),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	builder.Return(builder.ExtractValue(fn.Param(0), "slice.len", 1))

	// Compare the module string.
	if str := mod.String(); str != extractValueMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const extractValueMod = `
%runtime._slice = type { ptr, i32, i32 }

define i32 @getLen(%runtime._slice %0) {
  %slice.len = extractvalue %runtime._slice %0, 1
  ret i32 %slice.len
}
`

func TestBuilder_If(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("testIf", llvm.Signature{
		Type: ctx.Function(i32, false, ctx.Int(1)),
	}, 0, llvm.LinkConfig{})

	// Create the entry block.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock("entry"))
	ifTrue, ifFalse := builder.If(fn.Param(0), "ifTrue", "ifFalse")

	// Populate the if-true block.
	ret := fn.AppendBasicBlock("ret")
	builder.Jump(ret)

	// Populate the if-false block.
	builder.AtEnd(ifFalse)
	builder.Jump(ret)

	// Populate the return block.
	builder.AtEnd(ret)
	builder.Return(builder.Phi(i32, "result",
		llvm.PhiIncoming{ctx.ConstInt(32, 5).Value, ifTrue},
		llvm.PhiIncoming{ctx.ConstInt(32, 7).Value, ifFalse},
	).Value)

	// Compare the module string.
	if str := mod.String(); str != ifMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const ifMod = `
define i32 @testIf(i1 %0) {
entry:
  br i1 %0, label %ifTrue, label %ifFalse

ifTrue:                                           ; preds = %entry
  br label %ret

ifFalse:                                          ; preds = %entry
  br label %ret

ret:                                              ; preds = %ifFalse, %ifTrue
  %result = phi i32 [ 5, %ifTrue ], [ 7, %ifFalse ]
  ret i32 %result
}
`

func TestBuilder_Switch(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("testSwitch", llvm.Signature{
		Type: ctx.Function(i32, false, ctx.Int(8)),
	}, 0, llvm.LinkConfig{})

	// Create the entry block.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock("entry"))
	blockA := fn.AppendBasicBlock("A")
	blockB := fn.AppendBasicBlock("B")
	blockUnreachable := fn.AppendBasicBlock("unreachable")
	builder.Switch(fn.Param(0), blockUnreachable,
		llvm.SwitchCase{ctx.ConstInt(8, 'A'), blockA},
		llvm.SwitchCase{ctx.ConstInt(8, 'B'), blockB},
	)

	// Populate the A block.
	builder.AtEnd(blockA)
	ret := fn.AppendBasicBlock("ret")
	builder.Jump(ret)

	// Populate the B block.
	builder.AtEnd(blockB)
	builder.Jump(ret)

	// Populate the unreachable block.
	builder.AtEnd(blockUnreachable)
	builder.Unreachable()

	// Populate the return block.
	builder.AtEnd(ret)
	builder.Return(builder.Phi(i32, "result",
		llvm.PhiIncoming{ctx.ConstInt(32, 1).Value, blockA},
		llvm.PhiIncoming{ctx.ConstInt(32, 2).Value, blockB},
	).Value)

	// Compare the module string.
	if str := mod.String(); str != switchMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const switchMod = `
define i32 @testSwitch(i8 %0) {
entry:
  switch i8 %0, label %unreachable [
    i8 65, label %A
    i8 66, label %B
  ]

A:                                                ; preds = %entry
  br label %ret

B:                                                ; preds = %entry
  br label %ret

unreachable:                                      ; preds = %entry
  unreachable

ret:                                              ; preds = %B, %A
  %result = phi i32 [ 1, %A ], [ 2, %B ]
  ret i32 %result
}
`

func TestBuilder_Select(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	fn := mod.CreateFunction("select", llvm.Signature{
		Type: ctx.Function(i32, false, ctx.Int(1)),
	}, 0, llvm.LinkConfig{})

	// Populate the function body.
	builder := ctx.Builder()
	defer builder.Destroy()
	builder.AtEnd(fn.AppendBasicBlock(""))
	builder.Return(
		builder.Select(
			fn.Param(0),
			ctx.ConstInt(32, 5).Value,
			ctx.ConstInt(32, 7).Value,
			"select",
		),
	)

	// Compare the module string.
	if str := mod.String(); str != selectMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const selectMod = `
define i32 @select(i1 %0) {
  %select = select i1 %0, i32 5, i32 7
  ret i32 %select
}
`

func TestBuilder_Ackermann(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create a function.
	i32 := ctx.Int(32)
	noUndefSet := ctx.AttributeSet(ctx.EnumAttribute(llvm.AttributeNoUndef))
	sig := llvm.Signature{
		Type:       ctx.Function(i32, false, i32, i32),
		Convention: llvm.CallingConventionFast,
		Attributes: ctx.AttributeList(
			ctx.AttributeSet(ctx.EnumAttribute(llvm.AttributeNoUnwind)),
			noUndefSet,
			noUndefSet,
			noUndefSet,
		),
	}
	fn := mod.CreateFunction("ack", sig, 0, llvm.LinkConfig{})

	// Populate the function body.
	/*
		func ack(m, n uint32) uint32 {
			for m != 0 {
				if n != 0 {
					n = ack(m, n-1)
				} else {
					n = 1
				}
				m--
			}
			return n + 1
		}
	*/
	builder := ctx.Builder()
	defer builder.Destroy()
	entry := fn.AppendBasicBlock("entry")
	builder.AtEnd(entry)
	loopHeader := fn.AppendBasicBlock("loop.header")
	// for m != 0 {
	builder.Jump(loopHeader)
	builder.AtEnd(loopHeader)
	m := builder.Phi(i32, "m", llvm.PhiIncoming{fn.Param(0), entry})
	n := builder.Phi(i32, "n", llvm.PhiIncoming{fn.Param(1), entry})
	body, exit := builder.If(
		builder.Compare(llvm.IntNotEqual, m.Value, i32.Zero().Value, "loop.cond"),
		"loop.body", "loop.exit",
	)
	// if n != 0 {
	recurse, cont := builder.If(
		builder.Compare(llvm.IntNotEqual, n.Value, i32.Zero().Value, "recurse.cond"),
		"recurse", "loop.continue",
	)
	// n = ack(m, n-1)
	ndec := builder.Subtract(n.Value, ctx.ConstInt(32, 1).Value, "n.dec", true, false)
	nrec := builder.Call(sig, fn.Value, "n.recurse", m.Value, ndec)
	// m--
	builder.Jump(cont)
	builder.AtEnd(cont)
	nNext := builder.Phi(i32, "n.next",
		llvm.PhiIncoming{ctx.ConstInt(32, 1).Value, body},
		llvm.PhiIncoming{nrec, recurse},
	)
	mdec := builder.Subtract(m.Value, ctx.ConstInt(32, 1).Value, "m.dec", true, false)
	builder.Jump(loopHeader)
	m.Add(llvm.PhiIncoming{mdec, cont})
	n.Add(llvm.PhiIncoming{nNext.Value, cont})
	// return n + 1
	builder.AtEnd(exit)
	builder.Return(builder.Add(n.Value, ctx.ConstInt(32, 1).Value, "result", true, false))

	// Compare the module string.
	if str := mod.String(); str != ackermannMod {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const ackermannMod = `
; Function Attrs: nounwind
define fastcc noundef i32 @ack(i32 noundef %0, i32 noundef %1) #0 {
entry:
  br label %loop.header

loop.header:                                      ; preds = %loop.continue, %entry
  %m = phi i32 [ %0, %entry ], [ %m.dec, %loop.continue ]
  %n = phi i32 [ %1, %entry ], [ %n.next, %loop.continue ]
  %loop.cond = icmp ne i32 %m, 0
  br i1 %loop.cond, label %loop.body, label %loop.exit

loop.body:                                        ; preds = %loop.header
  %recurse.cond = icmp ne i32 %n, 0
  br i1 %recurse.cond, label %recurse, label %loop.continue

recurse:                                          ; preds = %loop.body
  %n.dec = sub nuw i32 %n, 1
  %n.recurse = call fastcc noundef i32 @ack(i32 noundef %m, i32 noundef %n.dec) #0
  br label %loop.continue

loop.continue:                                    ; preds = %recurse, %loop.body
  %n.next = phi i32 [ 1, %loop.body ], [ %n.recurse, %recurse ]
  %m.dec = sub nuw i32 %m, 1
  br label %loop.header

loop.exit:                                        ; preds = %loop.header
  %result = add nuw i32 %n, 1
  ret i32 %result
}

attributes #0 = { nounwind }
`
