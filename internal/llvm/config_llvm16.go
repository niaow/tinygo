//go:build !byollvm && llvm16

package llvm

/*
#cgo linux        CFLAGS:  -I/usr/lib/llvm-16/include
#cgo darwin,amd64 CFLAGS:  -I/usr/local/opt/llvm@16/include
#cgo darwin,arm64 CFLAGS:  -I/opt/homebrew/opt/llvm@16/include
#cgo freebsd      CFLAGS:  -I/usr/local/llvm16/include
#cgo linux        LDFLAGS: -L/usr/lib/llvm-16/lib -lLLVM-16
#cgo darwin,amd64 LDFLAGS: -L/usr/local/opt/llvm@16/lib -lLLVM-16
#cgo darwin,arm64 LDFLAGS: -L/opt/homebrew/opt/llvm@16/lib -lLLVM-16
#cgo freebsd      LDFLAGS: -L/usr/local/llvm16/lib -lLLVM-16
*/
import "C"
