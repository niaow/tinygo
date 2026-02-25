package llvm_test

import (
	"slices"
	"testing"

	"github.com/tinygo-org/tinygo/internal/llvm"
)

func TestMetadataStringEmpty(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a string metadata.
	md := ctx.MetadataString("")

	// Get the value from the metadata.
	if str := md.StringValue(); str != "" {
		t.Errorf("unexpected value of metadata: %q", str)
	}

	// A duplicate call should produce the same metadata.
	if dup := ctx.MetadataString(""); dup != md {
		t.Error("duplicate metadata is not equal")
	}
}

func TestMetadataString(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a string metadata.
	md := ctx.MetadataString("test")

	// Get the value from the metadata.
	if str := md.StringValue(); str != "test" {
		t.Errorf("unexpected value of metadata: %q", str)
	}

	// A duplicate call should produce the same metadata.
	if dup := ctx.MetadataString("test"); dup != md {
		t.Error("duplicate metadata is not equal")
	}
}

func TestMetadataNodeEmpty(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create an empty node.
	md := ctx.MetadataNode()

	// Read the operands back.
	if operands := md.NodeOperands(); len(operands) != 0 {
		t.Error("unexpected operands of supposedly-empty node:", operands)
	}

	// A duplicate call should produce the same metadata.
	if dup := ctx.MetadataNode(); dup != md {
		t.Error("duplicate metadata is not equal")
	}
}

func TestMetadataNodeStrings(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create some metadata strings.
	strA := ctx.MetadataString("a")
	strB := ctx.MetadataString("b")

	// Create a metadata node.
	operands := []llvm.Metadata{strA, strB}
	node := ctx.MetadataNode(operands...)

	// Read the operands back.
	if got := node.NodeOperands(); !slices.Equal(got, operands) {
		t.Errorf("expected operands %v but got %v", got, operands)
	}
}

func TestMetadataAsValue(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a metadata string.
	md := ctx.MetadataString("mdstr")

	// Convert the metadata to a value.
	val := ctx.MetadataValue(md)

	// Ensure that the value casts back to the original metadata.
	if cast := val.Metadata(); cast != md {
		t.Error("metadata -> value -> metadata did not round-trip")
	}

	// Test stringification of the value.
	if str := val.String(); str != "metadata !\"mdstr\"" {
		t.Errorf("unexpected string of metadata value: %q", str)
	}
}

func TestValueAsMetadata(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a constant int32 5.
	int5 := ctx.ConstInt(32, 5)

	// Convert the value to a metadata.
	md := int5.Metadata()

	// Ensure that the metadata casts back to the original value.
	if val := md.UnwrapValue(); val != int5.Value {
		t.Errorf("cast %s to metadata and back to %s", int5, val)
	}
}
