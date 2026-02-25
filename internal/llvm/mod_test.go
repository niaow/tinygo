package llvm_test

import (
	"testing"

	"github.com/tinygo-org/tinygo/internal/llvm"
)

// TODO

func TestGlobal(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create global definitions.
	for _, g := range []struct {
		name        string
		initializer llvm.Constant
		options     llvm.VariableOptions
		str         string
	}{
		{
			name:        "default",
			initializer: ctx.ConstInt(32, 0),
			options:     llvm.VariableOptions{},
			str:         "@default = global i32 0",
		},
		{
			name:        "string",
			initializer: ctx.ConstString("example"),
			options: llvm.VariableOptions{
				Constant: true,
				Link: llvm.LinkConfig{
					Linkage:     llvm.LinkageInternal,
					UnnamedAddr: llvm.UnnamedAddrGlobal,
				},
			},
			str: "@string = internal unnamed_addr constant [7 x i8] c\"example\"",
		},
		{
			name:        "threadLocal",
			initializer: ctx.Pointer(0).Zero(),
			options: llvm.VariableOptions{
				TLS: llvm.TLSLocalExec,
				Link: llvm.LinkConfig{
					Linkage:    llvm.LinkageOnceODR,
					Visibility: llvm.VisibilityHidden,
					DSOLocal:   true,
				},
			},
			str: "@threadLocal = linkonce_odr hidden thread_local(localexec) global ptr null",
		},
		{
			name:        "attributes",
			initializer: ctx.ConstInt(16, 5),
			options: llvm.VariableOptions{
				Attributes: ctx.AttributeSet(ctx.StringAttribute("key", "value")),
			},
			// The actual attribute set is printed seperartely.
			str: "@attributes = global i16 5 #0",
		},
		{
			name:        "DLLExport",
			initializer: ctx.ConstInt(64, 0),
			options: llvm.VariableOptions{
				Link: llvm.LinkConfig{
					Linkage:    llvm.LinkageWeakODR,
					DLLStorage: llvm.DLLStorageExport,
				},
			},
			str: "@DLLExport = weak_odr dllexport global i64 0",
		},
	} {
		t.Run(g.name, func(t *testing.T) {
			// Create the global.
			global := mod.CreateVariable(g.name, g.initializer, g.options)

			// Stringify the global.
			if str := global.LongString(); str != g.str {
				t.Errorf("unexpected string of global:\n\t%s\nexpected:\n\t%s", str, g.str)
			}

			// The type should be a pointer in the chosen address space.
			if gty := global.Type(); gty != ctx.Pointer(g.options.AddrSpace) {
				t.Errorf("ref type of global in address space %d is %s", g.options.AddrSpace, gty)
			}

			// The contained type should match the initializer.
			if gty, ity := global.ContainedType(), g.initializer.Type(); gty != ity {
				t.Errorf("contained type %s of global does not match initializer type %s", gty, ity)
			}

			// Read the initializer back.
			if init, ok := global.GetInitializer(); !ok {
				t.Error("missing initializer")
			} else if init != g.initializer {
				t.Errorf("initializer %s does not match provided initializer %s", init, g.initializer)
			}

			// Look up the global in the module.
			if got, ok := mod.Get(g.name); !ok {
				t.Errorf("global %q not found", g.name)
			} else if got != global.Global {
				t.Errorf("expected global %q but got %q", global, got)
			}
		})
	}

	// Create external global declarations.
	for _, g := range []struct {
		name    string
		ty      llvm.Type
		options llvm.VariableOptions
		str     string
	}{
		{
			// TODO: use an opaque struct type
			name:    "external",
			ty:      ctx.Pointer(0),
			options: llvm.VariableOptions{},
			str:     "@external = external global ptr",
		},
	} {
		t.Run(g.name, func(t *testing.T) {
			// Create the global.
			global := mod.CreateExternalVariable(g.name, g.ty, g.options)

			// Stringify the global.
			if str := global.LongString(); str != g.str {
				t.Errorf("unexpected string of global:\n\t%s\nexpected:\n\t%s", str, g.str)
			}

			// The type should be a pointer in the chosen address space.
			if gty := global.Type(); gty != ctx.Pointer(g.options.AddrSpace) {
				t.Errorf("ref type of global in address space %d is %s", g.options.AddrSpace, gty)
			}

			// The contained type should match the provided type.
			if gty := global.ContainedType(); gty != g.ty {
				t.Errorf("contained type %s of global does not match delcared type %s", gty, g.ty)
			}

			// Verify that no initializer is reported.
			if init, ok := global.GetInitializer(); ok {
				t.Errorf("unexpected initializer %s", init)
			}

			// Read the link config back.
			if link := global.LinkInfo(); link != g.options.Link {
				t.Error("unexpected link info:", link)
			}

			// Read the options back.
			if options := global.Options(); options != g.options {
				t.Error("unexpected options:", options)
			}
		})
	}

	// Test stringification of the module.
	if str := mod.String(); str != globalsModStr {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const globalsModStr = `
@default = global i32 0
@string = internal unnamed_addr constant [7 x i8] c"example"
@threadLocal = linkonce_odr hidden thread_local(localexec) global ptr null
@attributes = global i16 5 #0
@DLLExport = weak_odr dllexport global i64 0
@external = external global ptr

attributes #0 = { "key"="value" }
`

func TestFunc(t *testing.T) {
	t.Parallel()

	// Create a context to test with.
	ctx := llvm.CreateContext()
	defer ctx.Destroy()

	// Create a module.
	mod := ctx.CreateModule("", "", llvm.DataLayout{})
	defer mod.Destroy()

	// Create function definitions.
	for _, f := range []struct {
		name      string
		signature llvm.Signature
		link      llvm.LinkConfig
		str       string
	}{
		{
			name: "plain",
			signature: llvm.Signature{
				Type: ctx.Function(ctx.Void(), false),
			},
			str: "declare void @plain()",
		},
		{
			name: "mulWide",
			signature: llvm.Signature{
				Type:       ctx.Function(ctx.Int(64), false, ctx.Int(32), ctx.Int(32)),
				Convention: llvm.CallingConventionFast,
				Attributes: ctx.AttributeList(
					ctx.AttributeSet(
						ctx.EnumAttribute(llvm.AttributeInlineHint),
						ctx.EnumAttribute(llvm.AttributeNoUnwind),
					),
					ctx.AttributeSet(ctx.EnumAttribute(llvm.AttributeNoUndef)),
					ctx.AttributeSet(ctx.EnumAttribute(llvm.AttributeNoUndef)),
					ctx.AttributeSet(ctx.EnumAttribute(llvm.AttributeNoUndef)),
				),
			},
			link: llvm.LinkConfig{
				Linkage:     llvm.LinkageInternal,
				UnnamedAddr: llvm.UnnamedAddrGlobal,
				DSOLocal:    true,
			},
			str: "; Function Attrs: inlinehint nounwind\n" +
				"declare internal fastcc noundef i64 @mulWide(i32 noundef, i32 noundef) unnamed_addr #0",
		},
		{
			name: "ack",
			signature: llvm.Signature{
				Type:       ctx.Function(ctx.Int(64), false, ctx.Int(64), ctx.Int(64)),
				Convention: llvm.CallingConventionTail,
			},
			link: llvm.LinkConfig{
				Linkage:     llvm.LinkageInternal,
				UnnamedAddr: llvm.UnnamedAddrGlobal,
				DSOLocal:    true,
			},
			str: "declare internal tailcc i64 @ack(i64, i64) unnamed_addr",
		},
		{
			name: "printf",
			signature: llvm.Signature{
				Type: ctx.Function(ctx.Int(32), true, ctx.Pointer(0)),
			},
			link: llvm.LinkConfig{
				UnnamedAddr: llvm.UnnamedAddrLocal,
				DLLStorage:  llvm.DLLStorageImport,
			},
			str: "declare dllimport i32 @printf(ptr, ...) local_unnamed_addr",
		},
	} {
		t.Run(f.name, func(t *testing.T) {
			// Create the function declaration.
			fn := mod.CreateFunction(f.name, f.signature, f.link)

			// Stringify the function declaration.
			if str := fn.LongString(); str != f.str {
				t.Errorf("unexpected string of global:\n%s", str)
			}

			// The contained type should match the signature type.
			if cty := fn.ContainedType(); cty != f.signature.Type {
				t.Error("unexpected contained type:", cty)
			}

			// Read the signature back.
			if sig := fn.Signature(); sig != f.signature {
				t.Error("unexpected signature:", sig)
			}

			// Read the link config back.
			if link := fn.LinkInfo(); link != f.link {
				t.Error("unexpected link info:", link)
			}
		})
	}

	// Test stringification of the module.
	if str := mod.String(); str != fnModStr {
		t.Errorf("unexpected module string:\n%s", str)
	}
}

const fnModStr = `
declare void @plain()

; Function Attrs: inlinehint nounwind
declare internal fastcc noundef i64 @mulWide(i32 noundef, i32 noundef) unnamed_addr #0

declare internal tailcc i64 @ack(i64, i64) unnamed_addr

declare dllimport i32 @printf(ptr, ...) local_unnamed_addr

attributes #0 = { inlinehint nounwind }
`
