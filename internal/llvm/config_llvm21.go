//go:build !byollvm && !llvm15 && !llvm16 && !llvm17 && !llvm18 && !llvm19 && !llvm20

package llvm

/*
#cgo linux        CFLAGS:  -I/usr/include/llvm-21 -I/usr/include/llvm-c-21 -I/usr/lib/llvm-21/include
#cgo darwin,amd64 CFLAGS:  -I/usr/local/opt/llvm@21/include
#cgo darwin,arm64 CFLAGS:  -I/opt/homebrew/opt/llvm@21/include
#cgo freebsd      CFLAGS:  -I/usr/local/llvm21/include
#cgo linux        LDFLAGS: -L/usr/lib/llvm-21/lib -lLLVM-21
#cgo darwin,amd64 LDFLAGS: -L/usr/local/opt/llvm@21/lib -lLLVM-21
#cgo darwin,arm64 LDFLAGS: -L/opt/homebrew/opt/llvm@21/lib -lLLVM-21
#cgo freebsd      LDFLAGS: -L/usr/local/llvm21/lib -lLLVM-21
*/
import "C"
