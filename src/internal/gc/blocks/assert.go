//go:build runtime_asserts

package blocks

import "unsafe"

func sliceFromPtr[T any](base unsafe.Pointer, len uintptr) []T {
	return unsafe.Slice(castWithAlign[T](base), len)
}

func sliceNoBounds[T any](slice []T, start int, end int) []T {
	return slice[start:end]
}

func castWithAlign[T any](ptr unsafe.Pointer) *T {
	// TODO: make runtimePanic wrapper that works with pure go
	if uintptr(ptr)%unsafe.Alignof(T{}) != 0 {
		panic("ptr not aligned")
	}
	return (*T)(ptr)
}

func lenToCap[T any](slice []T) []T {
	return slice[0:len(slice):len(slice)]
}
