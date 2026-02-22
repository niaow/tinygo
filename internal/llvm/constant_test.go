package llvm_test

import (
	"slices"
	"testing"

	"github.com/tinygo-org/tinygo/internal/llvm"
)

func TestConstIntZero(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create an i32 0 the obvious way.
	i32_0 := ctx.ConstInt(32, 0)

	// Test stringification.
	if str := i32_0.String(); str != "i32 0" {
		t.Errorf("unexpected string of i32 0: %q", str)
	}

	// The type should be i32.
	if ty := i32_0.Type(); ty != ctx.Int(32) {
		t.Errorf("unexpected type: %q", ty)
	}

	// Create an equivalent value with a zero-length source slice.
	if dup := ctx.ConstInt(32); dup != i32_0 {
		t.Errorf("duplicate 0 with zero-length slice: %q", dup)
	}

	// Create an equivalent value through truncation.
	if dup := ctx.ConstInt(32, 1<<32, ^uint64(0)); dup != i32_0 {
		t.Errorf("duplicate 0 through truncation: %q", dup)
	}

	// Create an equivalent value through Type.Zero().
	if dup := ctx.Int(32).Zero(); dup != i32_0 {
		t.Errorf("duplicate 0 through Type.Zero(): %q", dup)
	}

	// Check that this is considered a zero value.
	if !i32_0.IsConstantZero() {
		t.Error("i32 0 is not considered a zero value")
	}

	// Inspect the value with AsConstInt.
	ci := i32_0.AsConstInt()
	if bits := ci.Bits(); bits != 32 {
		t.Errorf("unexpected bit width: %d", bits)
	}
	if uval := ci.Uint64(); uval != 0 {
		t.Errorf("unexpected uint64 value of 0: %d", uval)
	}
	if sval := ci.Int64(); sval != 0 {
		t.Errorf("unexpected int64 value of 0: %d", sval)
	}
}

func TestConstIntBig(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a 256-bit constant.
	bigConst := ctx.ConstInt(256, 1, 2, 3, 4)

	// Test stringification.
	if str := bigConst.String(); str != "i256 25108406941546723056364004793593481054836439088298861789185" {
		t.Errorf("unexpected string of big int constant: %q", str)
	}

	// The type should be i256.
	if ty := bigConst.Type(); ty != ctx.Int(256) {
		t.Errorf("unexpected type: %q", ty)
	}

	// Inspect the value with AsConstInt.
	ci := bigConst.AsConstInt()
	if bits := ci.Bits(); bits != 256 {
		t.Errorf("unexpected bit width: %d", bits)
	}
	if slice := ci.Slice(); !slices.Equal(slice, []uint64{1, 2, 3, 4}) {
		t.Errorf("unexpected data for big int constant: %v", slice)
	}
}

// TODO: test other int edge cases

func TestConstBool(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create constant booleans.
	constTrue := ctx.ConstBool(true)
	constFalse := ctx.ConstBool(false)

	// Test stringification.
	if str := constTrue.String(); str != "i1 true" {
		t.Errorf("unexpected true: %q", str)
	}
	if str := constFalse.String(); str != "i1 false" {
		t.Errorf("unexpected false: %q", str)
	}

	// Both constants should use be of the type i1.
	i1 := ctx.Int(1)
	if ty := constTrue.Type(); ty != i1 {
		t.Errorf("unexpected type of true: %q", ty)
	}
	if ty := constFalse.Type(); ty != i1 {
		t.Errorf("unexpected type of false: %q", ty)
	}

	// Verify that ConstInt produces the same constant.
	if dupTrue := ctx.ConstInt(1, 1); dupTrue != constTrue {
		t.Errorf("duplicate true: %q", dupTrue)
	}
	if dupFalse := ctx.ConstInt(1, 0); dupFalse != constFalse {
		t.Errorf("duplicate false: %q", dupFalse)
	}

	// Inspect true with AsConstInt.
	ciTrue := constTrue.AsConstInt()
	if bits := ciTrue.Bits(); bits != 1 {
		t.Errorf("unexpected bit width of true: %d", bits)
	}
	if uval := ciTrue.Uint64(); uval != 1 {
		t.Errorf("unexpected uint64 value of true: %d", uval)
	}
	if sval := ciTrue.Int64(); sval != -1 {
		t.Errorf("unexpected int64 value of true: %d", sval)
	}

	// Inspect false with AsConstInt.
	ciFalse := constFalse.AsConstInt()
	if bits := ciFalse.Bits(); bits != 1 {
		t.Errorf("unexpected bit width of false: %d", bits)
	}
	if uval := ciFalse.Uint64(); uval != 0 {
		t.Errorf("unexpected uint64 value of false: %d", uval)
	}
	if sval := ciFalse.Int64(); sval != 0 {
		t.Errorf("unexpected int64 value of false: %d", sval)
	}

	// Test compare-to-zero on boolean constants.
	if constTrue.IsConstantZero() {
		t.Error("constant true is considered zero by IsConstantZero")
	}
	if !constFalse.IsConstantZero() {
		t.Error("constant false is not considered zero by IsConstantZero")
	}
	if !ciTrue.NotZero() {
		t.Errorf("constant true is considered zero by ConstInt.NotZero")
	}
	if ciFalse.NotZero() {
		t.Errorf("constant false is not considered zero by ConstInt.NotZero")
	}
}

func TestConstArrayEmpty(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create an empty array of booleans.
	arr := ctx.Int(1).ConstArray()

	// Test stringification.
	if str := arr.String(); str != "[0 x i1] zeroinitializer" {
		t.Errorf("unexpected empty bool array: %q", str)
	}

	// The type should be a zero-length array of i1.
	if ty := arr.Type(); ty != ctx.Array(0, ctx.Int(1)) {
		t.Errorf("unexpected type: %q", ty)
	}

	// The empty array should be a zero value.
	if !arr.IsConstantZero() {
		t.Error("empty array is not a zero value")
	}
}

func TestConstArray(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create an empty array of booleans.
	arr := ctx.Int(1).ConstArray(
		ctx.ConstBool(false),
		ctx.ConstBool(true),
		ctx.ConstBool(true),
		ctx.ConstBool(false),
		ctx.ConstBool(false),
	)

	// Test stringification.
	if str := arr.String(); str != "[5 x i1] [i1 false, i1 true, i1 true, i1 false, i1 false]" {
		t.Errorf("unexpected bool array: %q", str)
	}

	// The type should be an array of 5 i1s.
	if ty := arr.Type(); ty != ctx.Array(5, ctx.Int(1)) {
		t.Errorf("unexpected type: %q", ty)
	}

	// The array should not be a zero value.
	if arr.IsConstantZero() {
		t.Error("array is a zero value")
	}
}

func TestConstString(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Convert a string to a byte array constant.
	arr := ctx.ConstString("abc")

	// Test stringification.
	if str := arr.String(); str != "[3 x i8] c\"abc\"" {
		t.Errorf("unexpected string byte array: %q", str)
	}

	// The type should be an array of 3 bytes.
	if ty := arr.Type(); ty != ctx.Array(3, ctx.Int(8)) {
		t.Errorf("unexpected type: %q", ty)
	}
}

func TestConstIntArrayInt64(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Convert a slice of int64 to an integer array constant.
	arr := llvm.ConstIntArray(ctx, []int64{
		-1, 1, -1 << 63, (1 << 63) - 1,
	})

	// Test stringification.
	if str := arr.String(); str != "[4 x i64] [i64 -1, i64 1, i64 -9223372036854775808, i64 9223372036854775807]" {
		t.Errorf("unexpected int64 array: %q", str)
	}

	// The type should be an array of 4 i64s.
	if ty := arr.Type(); ty != ctx.Array(4, ctx.Int(64)) {
		t.Errorf("unexpected type: %q", ty)
	}
}

func TestConstStructImplicit(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a struct with integer types.
	structConst := ctx.ConstStruct(
		ctx.ConstBool(true),
		ctx.ConstInt(2, ^uint64(0)),
		ctx.ConstInt(64, 5),
	)

	// Test stringification.
	if str := structConst.String(); str != "{ i1, i2, i64 } { i1 true, i2 -1, i64 5 }" {
		t.Errorf("unexpected struct const: %q", str)
	}

	// The struct type should match a literal formed from the element types.
	if ty := structConst.Type(); ty != ctx.LiteralStruct(ctx.Int(1), ctx.Int(2), ctx.Int(64)) {
		t.Errorf("unexpected struct type: %q", ty)
	}
}

func TestConstStructNamed(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create the string header type.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	shdr := ctx.NamedStruct("runtime._string", ptr, i32)

	// Create an empty string header constant.
	strEmpty := shdr.ConstStruct(
		ptr.Zero(),
		ctx.ConstInt(32, 0),
	)

	// Test stringification.
	if str := strEmpty.String(); str != "%runtime._string zeroinitializer" {
		t.Errorf("unexpected empty string header constant: %q", str)
	}

	// The constant should use the provided type.
	if ty := strEmpty.Type(); ty != shdr {
		t.Errorf("unexpected type of string header: %q", ty)
	}
}
