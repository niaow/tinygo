//go:build !runtime_asserts

package blocks

import "unsafe"

//go:nobounds
func sliceFromPtr[T any](base unsafe.Pointer, len uintptr) []T {
	return unsafe.Slice(castWithAlign[T](base), len)
}

//go:nobounds
func sliceNoBounds[T any](slice []T, start int, end int) []T {
	return slice[start:end]
}

//go:nobounds
func castWithAlign[T any](ptr unsafe.Pointer) *T {
	return (*T)(ptr)
}

//go:nobounds
func lenToCap[T any](slice []T) []T {
	return slice[0:len(slice):len(slice)]
}
