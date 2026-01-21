//go:build !byollvm && llvm15

package llvm

/*
#cgo linux        CFLAGS:  -I/usr/lib/llvm-15/include
#cgo darwin,amd64 CFLAGS:  -I/usr/local/opt/llvm@15/include
#cgo darwin,arm64 CFLAGS:  -I/opt/homebrew/opt/llvm@15/include
#cgo freebsd      CFLAGS:  -I/usr/local/llvm15/include
#cgo linux        LDFLAGS: -L/usr/lib/llvm-15/lib -lLLVM-15
#cgo darwin,amd64 LDFLAGS: -L/usr/local/opt/llvm@15/lib -lLLVM-15
#cgo darwin,arm64 LDFLAGS: -L/opt/homebrew/opt/llvm@15/lib -lLLVM-15
#cgo freebsd      LDFLAGS: -L/usr/local/llvm15/lib -lLLVM-15
*/
import "C"
