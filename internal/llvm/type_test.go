package llvm_test

import (
	"slices"
	"testing"

	"github.com/tinygo-org/tinygo/internal/llvm"
)

func TestTypeVoid(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a void type.
	void := ctx.Void()

	// Test stringification.
	if str := void.String(); str != "void" {
		t.Errorf("unexpected string of void: %q", str)
	}

	// Future calls to ctx.Void should return the same type.
	if dup := ctx.Void(); dup != void {
		t.Errorf("duplicate voids: %q, %q", void, dup)
	}

	// Try using .Info to identify the type.
	if kind := void.Info().Kind(); kind != llvm.TypeKindVoid {
		t.Errorf("unexpected void kind: %s", kind)
	}
}

func TestTypeInt(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a 64-bit integer type.
	i64 := ctx.Int(64)

	// Test stringification.
	if str := i64.String(); str != "i64" {
		t.Errorf("unexpected string of void: %q", str)
	}

	// Future calls to ctx.Int(64) should return the same type.
	if dup := ctx.Int(64); dup != i64 {
		t.Errorf("duplicate i64: %q, %q", i64, dup)
	}

	// Try using .Info to identify the type.
	info := i64.Info()
	if kind := info.Kind(); kind != llvm.TypeKindInteger {
		t.Errorf("unexpected i64 kind: %s", kind)
	}
	if width := info.IntegerWidth(); width != 64 {
		t.Errorf("unexpected bit-width of i64: %d", width)
	}
}

func TestTypeFloat32(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a float32 type.
	f32 := ctx.Float32()

	// Test stringification.
	if str := f32.String(); str != "float" {
		t.Errorf("unexpected string of float32: %q", str)
	}

	// Future calls to ctx.Float32 should return the same type.
	if dup := ctx.Float32(); dup != f32 {
		t.Errorf("duplicate float32s: %q, %q", f32, dup)
	}

	// Try using .Info to identify the type.
	if kind := f32.Info().Kind(); kind != llvm.TypeKindFloat32 {
		t.Errorf("unexpected float64 kind: %s", kind)
	}
}

func TestTypeFloat64(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a float64 type.
	f64 := ctx.Float64()

	// Test stringification.
	if str := f64.String(); str != "double" {
		t.Errorf("unexpected string of float64: %q", str)
	}

	// Future calls to ctx.Float64 should return the same type.
	if dup := ctx.Float64(); dup != f64 {
		t.Errorf("duplicate float64s: %q, %q", f64, dup)
	}

	// Try using .Info to identify the type.
	if kind := f64.Info().Kind(); kind != llvm.TypeKindFloat64 {
		t.Errorf("unexpected float64 kind: %s", kind)
	}
}

func TestTypePointer0(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a pointer type in the default address space.
	ptr := ctx.Pointer(0)

	// Test stringification.
	if str := ptr.String(); str != "ptr" {
		t.Errorf("unexpected string of pointer: %q", str)
	}

	// Future calls to ctx.Pointer(0) should return the same type.
	if dup := ctx.Pointer(0); dup != ptr {
		t.Errorf("duplicate ptrs: %q, %q", ptr, dup)
	}

	// Try using .Info to identify the type.
	info := ptr.Info()
	if kind := info.Kind(); kind != llvm.TypeKindPointer {
		t.Errorf("unexpected ptr kind: %s", kind)
	}
	if addrSpace := info.AddressSpace(); addrSpace != 0 {
		t.Errorf("unexpected address space of ptr: %d", addrSpace)
	}
}

func TestTypePointer1(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a pointer type in the address space 1.
	ptr := ctx.Pointer(1)

	// Test stringification.
	if str := ptr.String(); str != "ptr addrspace(1)" {
		t.Errorf("unexpected string of pointer: %q", str)
	}

	// Future calls to ctx.Pointer(1) should return the same type.
	if dup := ctx.Pointer(1); dup != ptr {
		t.Errorf("duplicate ptrs: %q, %q", ptr, dup)
	}

	// Try using .Info to identify the type.
	info := ptr.Info()
	if kind := info.Kind(); kind != llvm.TypeKindPointer {
		t.Errorf("unexpected ptr kind: %s", kind)
	}
	if addrSpace := info.AddressSpace(); addrSpace != 1 {
		t.Errorf("unexpected address space of ptr: %d", addrSpace)
	}
}

func TestTypeLiteralStruct(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a literal struct shaped like a slice header.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	shdr := ctx.LiteralStruct(ptr, i32, i32)

	// Test stringification.
	if str := shdr.String(); str != "{ ptr, i32, i32 }" {
		t.Errorf("unexpected string of struct: %q", str)
	}

	// An identical call to ctx.LiteralStruct should return the same type.
	if dup := ctx.LiteralStruct(ptr, i32, i32); dup != shdr {
		t.Errorf("duplicate structs: %q, %q", shdr, dup)
	}

	// Try using .Info to identify the type.
	info := shdr.Info()
	if kind := info.Kind(); kind != llvm.TypeKindStruct {
		t.Errorf("unexpected struct kind: %s", kind)
	}
	if elements := info.StructElements(); !slices.Equal(elements, []llvm.Type{ptr, i32, i32}) {
		t.Errorf("unexpected struct elements: %v", elements)
	}
}

func TestTypeNamedStruct(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a named struct shaped like a string header.
	ptr := ctx.Pointer(0)
	i32 := ctx.Int(32)
	shdr := ctx.NamedStruct("runtime._string", ptr, i32)

	// Test stringification.
	if str := shdr.String(); str != "%runtime._string = type { ptr, i32 }" {
		t.Errorf("unexpected string of struct: %q", str)
	}

	// Create an identical struct.
	// The new type should be different.
	if shdr2 := ctx.NamedStruct("runtime._string", ptr, i32); shdr2 == shdr {
		t.Error("duplicate struct is equal")
	}

	// Try using .Info to identify the type.
	info := shdr.Info()
	if kind := info.Kind(); kind != llvm.TypeKindStruct {
		t.Errorf("unexpected struct kind: %s", kind)
	}
	if elements := info.StructElements(); !slices.Equal(elements, []llvm.Type{ptr, i32}) {
		t.Errorf("unexpected struct elements: %v", elements)
	}
}

func TestTypeArray(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create an array of 3 bytes.
	i8 := ctx.Int(8)
	arr := ctx.Array(3, i8)

	// Test stringification.
	if str := arr.String(); str != "[3 x i8]" {
		t.Errorf("unexpected string of array: %q", str)
	}

	// An identical call to ctx.Array should return the same type.
	if dup := ctx.Array(3, i8); dup != arr {
		t.Errorf("duplicate arrays: %q, %q", arr, dup)
	}

	// Try using .Info to identify the type.
	info := arr.Info()
	if kind := info.Kind(); kind != llvm.TypeKindArray {
		t.Errorf("unexpected array kind: %s", kind)
	}
	if len, elem := info.ArrayShape(); len != 3 || elem != i8 {
		t.Errorf("unexpected array shape: %d x %s", len, elem)
	}
}
