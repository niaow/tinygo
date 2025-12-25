package blocks

import "unsafe"

const maskBytes = unsafe.Sizeof(mask(0))
const maskBits = 8 * maskBytes
