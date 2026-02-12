package llvm_test

import (
	"slices"
	"testing"

	"github.com/tinygo-org/tinygo/internal/llvm"
)

func TestStringAttribute(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a simple string attribute.
	attr := ctx.StringAttribute("key", "value")

	// Test stringification.
	if str := attr.String(); str != "\"key\"=\"value\"" {
		t.Errorf("unexpected string of string attribute: %q", str)
	}

	// Try reading the attribute back.
	key, isString := attr.Kind()
	if !isString {
		t.Error("not a string attribute")
	}
	if key != "key" {
		t.Errorf("unexpected key: %q", key)
	}
	if isString {
		value := attr.StringValue()
		if value != "value" {
			t.Errorf("unexpected value: %q", value)
		}
	}

	// A duplicate attribute should be equal.
	if ctx.StringAttribute("key", "value") != attr {
		t.Error("duplicate attribute is not equal")
	}
}

func TestEnumAttributes(t *testing.T) {
	t.Parallel()

	for _, kind := range []llvm.EnumAttribute{
		llvm.AttributeZeroExtend,
		llvm.AttributeSignExtend,
		llvm.AttributeNoAlias,
		llvm.AttributeNonNull,
		llvm.AttributeNoUndef,
		llvm.AttributeReadNone,
		llvm.AttributeReadOnly,
		llvm.AttributeWriteOnly,
	} {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			// Create a context to test with.
			ctx := llvm.CreateContext()
			defer ctx.Destroy()

			// Create the attribute.
			attr := ctx.EnumAttribute(kind)

			// Test stringification.
			if str := attr.String(); str != string(kind) {
				t.Errorf("unexpected string of attribute: %q", str)
			}

			// Try reading the attribute back.
			key, isString := attr.Kind()
			if isString {
				t.Error("enum attribute is string")
			}
			if key != string(kind) {
				t.Errorf("unexpected attribute key: %q", key)
			}

			// A duplicate attribute should be equal.
			if ctx.EnumAttribute(kind) != attr {
				t.Error("duplicate attribute is not equal")
			}
		})
	}
}

func TestIntAttribute(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		kind  llvm.IntAttribute
		value uint64
		str   string
	}{
		{
			kind:  llvm.AttributeAlign,
			value: 4,
			str:   "align 4",
		},
		{
			kind:  llvm.AttributeAlign,
			value: 16,
			str:   "align 16",
		},
		{
			kind:  llvm.AttributeDereferenceable,
			value: 7,
			str:   "dereferenceable(7)",
		},
		{
			kind:  llvm.AttributeDereferenceableOrNull,
			value: 19,
			str:   "dereferenceable_or_null(19)",
		},
	} {
		c := c
		t.Run(c.str, func(t *testing.T) {
			t.Parallel()

			// Create a context to test with.
			ctx := llvm.CreateContext()
			defer ctx.Destroy()

			// Create the attribute.
			attr := ctx.IntAttribute(c.kind, c.value)

			// Test stringification.
			if str := attr.String(); str != c.str {
				t.Errorf("unexpected string of attribute: %q", str)
			}

			// Try reading the attribute back.
			key, isString := attr.Kind()
			if isString {
				t.Error("int attribute is string")
			}
			if key != string(c.kind) {
				t.Errorf("unexpected attribute key: %q", key)
			}
			if value := attr.IntValue(); value != c.value {
				t.Errorf("unexpected attribute value: %q", value)
			}

			// A duplicate attribute should be equal.
			if ctx.IntAttribute(c.kind, c.value) != attr {
				t.Error("duplicate attribute is not equal")
			}
		})
	}
}

func TestTypeAttribute(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	for _, c := range []struct {
		kind  llvm.TypeAttribute
		value llvm.Type
		str   string
	}{
		{
			kind:  llvm.AttributeByVal,
			value: ctx.Array(8, ctx.Int(8)),
			str:   "byval([8 x i8])",
		},
		{
			kind: llvm.AttributeByRef,
			value: ctx.LiteralStruct(
				ctx.Pointer(0),
				ctx.Int(32),
				ctx.Int(32),
			),
			str: "byref({ ptr, i32, i32 })",
		},
		{
			kind:  llvm.AttributeStackReturn,
			value: ctx.Array(5, ctx.Float32()),
			str:   "sret([5 x float])",
		},
	} {
		t.Run(c.str, func(t *testing.T) {
			// Create the attribute.
			attr := ctx.TypeAttribute(c.kind, c.value)

			// Test stringification.
			if str := attr.String(); str != c.str {
				t.Errorf("unexpected string of attribute: %q", str)
			}

			// Try reading the attribute back.
			key, isString := attr.Kind()
			if isString {
				t.Error("type attribute is string")
			}
			if key != string(c.kind) {
				t.Errorf("unexpected attribute key: %q", key)
			}
			if value := attr.TypeValue(); value != c.value {
				t.Errorf("unexpected attribute value: %q", value)
			}

			// A duplicate attribute should be equal.
			if ctx.TypeAttribute(c.kind, c.value) != attr {
				t.Error("duplicate attribute is not equal")
			}
		})
	}
}

func TestRangeAttribute(t *testing.T) {
	if !llvm.SupportsAttributeRange {
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create the attribute.
	attr := ctx.RangeAttribute(llvm.AttributeRange, 32, nil, []uint64{12})

	// Test stringification.
	if str := attr.String(); str != "range(i32 0, 12)" {
		t.Errorf("unexpected string of attribute: %q", str)
	}

	// Try reading the attribute back.
	key, isString := attr.Kind()
	if isString {
		t.Error("enum attribute is string")
	}
	if key != string(llvm.AttributeRange) {
		t.Errorf("unexpected attribute key: %q", key)
	}
	bits, lower, upper := attr.RangeValue()
	if bits != 32 {
		t.Errorf("unexpected bit width: %d", bits)
	}
	if !slices.Equal(lower, []uint64{0}) {
		t.Errorf("unexpected lower bound: %v", lower)
	}
	if !slices.Equal(upper, []uint64{12}) {
		t.Errorf("unexpected upper bound: %v", upper)
	}

	// A duplicate attribute should be equal.
	if ctx.RangeAttribute(llvm.AttributeRange, 32, nil, []uint64{12}) != attr {
		t.Error("duplicate attribute is not equal")
	}
}

func TestStringAttributeSet(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create simple string attributes.
	xAttr := ctx.StringAttribute("x", "1")
	yAttr := ctx.StringAttribute("y", "xyzzy")

	// Combine the attributes into a set.
	set := ctx.AttributeSet(xAttr, yAttr)

	// Test stringification.
	if str := set.String(); str != "\"x\"=\"1\" \"y\"=\"xyzzy\"" {
		t.Errorf("unexpected string of attribute set: %q", str)
	}

	// A duplicate set should be equal.
	if ctx.AttributeSet(xAttr, yAttr) != set {
		t.Error("duplicate set is not equal")
	}

	// Reversing the argument order should still produce the same set.
	if ctx.AttributeSet(yAttr, xAttr) != set {
		t.Error("element reversal produced a different set")
	}

	// Fetch the attributes from the set.
	testGetStringAttr(t, set, "x", xAttr, "1")
	testGetStringAttr(t, set, "y", yAttr, "xyzzy")

	// Ensure an unrelated attribute is reported absent.
	if attr, ok := set.GetStringAttr("apple"); ok {
		t.Errorf("unexpected attribute %q (from Attribute.GetStringAttr)", attr)
	}
	if value, ok := set.GetString("apple"); ok {
		t.Errorf("unexpected attribute \"apple\"=%q (from Attribute.GetString)", value)
	}
}

func testGetStringAttr(t *testing.T, set llvm.AttributeSet, kind string, attr llvm.Attribute, value string) {
	t.Helper()
	if got, ok := set.GetStringAttr(kind); !ok {
		t.Errorf("attribute %q reported missing by Attribute.GetStringAttr", kind)
	} else if got != attr {
		t.Errorf("attribute %q is %q (expected %q)", kind, got, attr)
	}
	if got, ok := set.GetString(kind); !ok {
		t.Errorf("attribute %q reported missing by Attribute.GetString", kind)
	} else if got != value {
		t.Errorf("attribute %q is %q (expected %q)", kind, got, value)
	}
}

func TestAttributeSetPtrRead(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a set for an aligned array of read-only bytes passed by value through a pointer.
	readOnly := ctx.EnumAttribute(llvm.AttributeReadOnly)
	align8 := ctx.IntAttribute(llvm.AttributeAlign, 8)
	byteArr := ctx.Array(8, ctx.Int(8))
	byValByteArr := ctx.TypeAttribute(llvm.AttributeByVal, byteArr)
	set := ctx.AttributeSet(readOnly, align8, byValByteArr)

	// Fetch the raw attributes from the set.
	testGetAttr(t, set, string(llvm.AttributeReadOnly), readOnly)
	testGetAttr(t, set, string(llvm.AttributeAlign), align8)
	testGetAttr(t, set, string(llvm.AttributeByVal), byValByteArr)

	// Read the attribute values back.
	if !set.HasEnum(llvm.AttributeReadOnly) {
		t.Error("readonly attribute reported missing")
	}
	if a, ok := set.GetInt(llvm.AttributeAlign); !ok {
		t.Error("align attribute reported missing")
	} else if a != 8 {
		t.Errorf("unexpected alignment: %d", a)
	}
	if ty, ok := set.GetType(llvm.AttributeByVal); !ok {
		t.Error("byval attribute reported missing")
	} else if ty != byteArr {
		t.Errorf("expected byval type %s but got %s", byteArr, ty)
	}

	// Ensure that other attributes are missing.
	if set.HasEnum(llvm.AttributeWriteOnly) {
		t.Error("set contains writeonly")
	}
	if size, ok := set.GetInt(llvm.AttributeDereferenceableOrNull); ok {
		t.Errorf("set contains dereferenceable_or_null(%d)", size)
	}
	if ty, ok := set.GetType(llvm.AttributeByRef); ok {
		t.Errorf("set contains byref(%s)", ty)
	}
}

func TestAttributeSetRangeInt(t *testing.T) {
	if !llvm.SupportsAttributeRange {
		t.Skip()
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a set for a string length on a 32-bit system.
	zext := ctx.EnumAttribute(llvm.AttributeZeroExtend)
	lenRange := ctx.RangeAttribute(llvm.AttributeRange, 32, []uint64{0}, []uint64{1 << 31})
	set := ctx.AttributeSet(zext, lenRange)

	// Fetch the raw attributes from the set.
	testGetAttr(t, set, string(llvm.AttributeZeroExtend), zext)
	testGetAttr(t, set, string(llvm.AttributeRange), lenRange)

	// Read the attribute values back.
	if !set.HasEnum(llvm.AttributeZeroExtend) {
		t.Error("zeroext attribute reported missing")
	}
	if bits, lower, upper, ok := set.GetRange(llvm.AttributeRange); !ok {
		t.Error("range attribute reported missing")
	} else {
		if bits != 32 {
			t.Errorf("expected 32-bit range, but got %d bits", bits)
		}
		if !slices.Equal(lower, []uint64{0}) {
			t.Errorf("unexpected lower bound: %v", lower)
		}
		if !slices.Equal(upper, []uint64{1 << 31}) {
			t.Errorf("unexpected upper bound: %v", upper)
		}
	}
}

func testGetAttr(t *testing.T, set llvm.AttributeSet, kind string, value llvm.Attribute) {
	t.Helper()
	attr, ok := set.Get(kind)
	if !ok {
		t.Errorf("missing attribute %q", kind)
	} else if attr != value {
		t.Errorf("attribute %q is %q (expected %q)", kind, attr, value)
	}
}

func TestMergeStringAttributeSets(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create simple string attributes.
	xAttr := ctx.StringAttribute("x", "1")
	yAttr := ctx.StringAttribute("y", "xyzzy")

	// Place the attributes into sets and merge them.
	set := ctx.MergeAttributeSets(
		ctx.AttributeSet(xAttr),
		ctx.AttributeSet(yAttr),
	)

	// Test stringification.
	if str := set.String(); str != "\"x\"=\"1\" \"y\"=\"xyzzy\"" {
		t.Errorf("unexpectred string of attribute set: %q", str)
	}
}

// TODO: test intersect

func TestCaptureAttributes(t *testing.T) {
	if llvm.VersionMajor < 20 {
		t.Skip("captures(...) requires LLVM 20 or newer")
	}

	t.Parallel()

	for _, c := range []struct {
		info       llvm.CaptureInfo
		str        string
		infoString string
	}{
		{
			info:       llvm.CaptureInfo{},
			str:        "",
			infoString: "address, provenance",
		},
		{
			info: llvm.CaptureInfo{
				Other: llvm.CaptureComponents{
					Address:    llvm.AddressCaptureNone,
					Provenance: llvm.ProvenanceCaptureNone,
				},
			},
			str:        "captures(ret: address, provenance)",
			infoString: "ret: address, provenance",
		},
		{
			info: llvm.CaptureInfo{
				Other: llvm.CaptureComponents{
					Address:    llvm.AddressCaptureIsNull,
					Provenance: llvm.ProvenanceCaptureNone,
				},
				Return: llvm.CaptureComponents{
					Address: llvm.AddressCaptureNone,
				},
			},
			str:        "captures(address_is_null, ret: provenance)",
			infoString: "address_is_null, ret: provenance",
		},
	} {
		c := c
		t.Run(c.infoString, func(t *testing.T) {
			t.Parallel()

			// Create a context to test with.
			ctx := llvm.CreateContext()
			defer ctx.Destroy()

			// Create the attribute set.
			set := ctx.CaptureAttributes(c.info)

			// Test stringification.
			if str := set.String(); str != c.str {
				t.Errorf("unexpected string of set: %q", str)
			}

			// Try reading the info back.
			if info := set.CaptureInfo(); info != c.info {
				t.Errorf("unexpected capture info: %s", info)
			}

			// Test stringification of the raw info.
			if str := c.info.String(); str != c.infoString {
				t.Errorf("unexpected string of info: %q", str)
			}
		})
	}
}

func TestLegacyNoCapture(t *testing.T) {
	// TODO: remove this test when we drop LLVM 19 support.
	if llvm.VersionMajor >= 20 {
		t.Skip("nocapture removed by LLVM 20 in favor of captures(...)")
	}

	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a simple set with the nocapture attribute.
	noCaptureSet := ctx.AttributeSet(ctx.EnumAttribute("nocapture"))

	// Ensure that CaptureAttributes creates the same set.
	noCaptureComponents := llvm.CaptureComponents{
		Address:    llvm.AddressCaptureNone,
		Provenance: llvm.ProvenanceCaptureNone,
	}
	noCaptureInfo := llvm.CaptureInfo{
		Other:  noCaptureComponents,
		Return: noCaptureComponents,
	}
	if set := ctx.CaptureAttributes(noCaptureInfo); set != noCaptureSet {
		t.Errorf("expected attr set %q but got %q", noCaptureSet, set)
	}

	// Ensure that CaptureAttributes creates an empty set if something is captured.
	if set := ctx.CaptureAttributes(llvm.CaptureInfo{}); set != (llvm.AttributeSet{}) {
		t.Errorf("expected empty set from partial capture but got %q", set)
	}

	// Test reading back the info from the noCaptureSet.
	if info := noCaptureSet.CaptureInfo(); info != noCaptureInfo {
		t.Errorf("unexpected info from nocapture: %q", info)
	}

	// Test info from an empty set.
	if info := (llvm.AttributeSet{}).CaptureInfo(); info != noCaptureInfo {
		t.Errorf("unexpected info from empty set: %q", info)
	}
}
