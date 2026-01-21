//go:build !byollvm && llvm18

package llvm

/*
#cgo linux        CFLAGS:  -I/usr/include/llvm-18 -I/usr/include/llvm-c-18 -I/usr/lib/llvm-18/include
#cgo darwin,amd64 CFLAGS:  -I/usr/local/opt/llvm@18/include
#cgo darwin,arm64 CFLAGS:  -I/opt/homebrew/opt/llvm@18/include
#cgo freebsd      CFLAGS:  -I/usr/local/llvm18/include
#cgo linux        LDFLAGS: -L/usr/lib/llvm-18/lib -lLLVM-18
#cgo darwin,amd64 LDFLAGS: -L/usr/local/opt/llvm@18/lib -lLLVM-18
#cgo darwin,arm64 LDFLAGS: -L/opt/homebrew/opt/llvm@18/lib -lLLVM-18
#cgo freebsd      LDFLAGS: -L/usr/local/llvm18/lib -lLLVM-18
*/
import "C"
