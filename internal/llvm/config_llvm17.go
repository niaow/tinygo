//go:build !byollvm && llvm17

package llvm

/*
#cgo linux        CFLAGS:  -I/usr/include/llvm-17 -I/usr/include/llvm-c-17 -I/usr/lib/llvm-17/include
#cgo darwin,amd64 CFLAGS:  -I/usr/local/opt/llvm@17/include
#cgo darwin,arm64 CFLAGS:  -I/opt/homebrew/opt/llvm@17/include
#cgo freebsd      CFLAGS:  -I/usr/local/llvm17/include
#cgo linux        LDFLAGS: -L/usr/lib/llvm-17/lib -lLLVM-17
#cgo darwin,amd64 LDFLAGS: -L/usr/local/opt/llvm@17/lib -lLLVM-17
#cgo darwin,arm64 LDFLAGS: -L/opt/homebrew/opt/llvm@17/lib -lLLVM-17
#cgo freebsd      LDFLAGS: -L/usr/local/llvm17/lib -lLLVM-17
*/
import "C"
