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
