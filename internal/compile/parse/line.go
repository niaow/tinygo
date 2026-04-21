package parse

import (
	"bytes"
	"math/bits"
	"runtime"
)

// LineTable holds the number of bytes after each newline in the file.
type LineTable []uint32

const hasUnalignedLoad = runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"

var newlineArr = [1]byte{'\n'}

// Lines constructs a LineTable for a source file.
// The source file must be no more than math.MaxUint32 bytes long.
func Lines(src []byte) LineTable {
	// Count the newline bytes before allocating the buffer.
	// The count is cheaper than growing the slice as we go.
	count := bytes.Count(src, newlineArr[:])
	lines := make(LineTable, count)

	// Only use this fallback when the CPU cannot perform unaligned loads.
	// The inline scan is otherwise much faster due to the lack of function call overhead.
	// Perf diff: 2.6 GB/s inline, 1.3 GB/s bytes.IndexByte (Threadripper 2990WX, single thread)
	if !hasUnalignedLoad {
		// Use bytes.IndexByte as a fallback if the architecture does not support unaligned loads.
		lines = lines[:0]
		for {
			next := uint(bytes.IndexByte(src, '\n'))
			if next >= uint(len(src)) {
				break
			}
			src = src[next+1:]
			lines = append(lines, uint32(len(src)))
		}
		return lines
	}

	oldLen := len(src)

	// Scan backwards in batches of 64.
	// Perf diff: 2.6 GB/s with, 1.6 GB/s without (Threadripper 2990WX, single thread)
	// We could probbably go much faster with VPMOVMSKB, but I do not want to deal with assembly right now.
	for ; len(src) >= 64; src = src[:len(src)-64] {
		// Load the batch into 8 little-endian uint64s.
		b0 := uint64(src[len(src)-1])<<070 |
			uint64(src[len(src)-2])<<060 |
			uint64(src[len(src)-3])<<050 |
			uint64(src[len(src)-4])<<040 |
			uint64(src[len(src)-5])<<030 |
			uint64(src[len(src)-6])<<020 |
			uint64(src[len(src)-7])<<010 |
			uint64(src[len(src)-8])<<000
		b1 := uint64(src[len(src)-9])<<070 |
			uint64(src[len(src)-10])<<060 |
			uint64(src[len(src)-11])<<050 |
			uint64(src[len(src)-12])<<040 |
			uint64(src[len(src)-13])<<030 |
			uint64(src[len(src)-14])<<020 |
			uint64(src[len(src)-15])<<010 |
			uint64(src[len(src)-16])<<000
		b2 := uint64(src[len(src)-17])<<070 |
			uint64(src[len(src)-18])<<060 |
			uint64(src[len(src)-19])<<050 |
			uint64(src[len(src)-20])<<040 |
			uint64(src[len(src)-21])<<030 |
			uint64(src[len(src)-22])<<020 |
			uint64(src[len(src)-23])<<010 |
			uint64(src[len(src)-24])<<000
		b3 := uint64(src[len(src)-25])<<070 |
			uint64(src[len(src)-26])<<060 |
			uint64(src[len(src)-27])<<050 |
			uint64(src[len(src)-28])<<040 |
			uint64(src[len(src)-29])<<030 |
			uint64(src[len(src)-30])<<020 |
			uint64(src[len(src)-31])<<010 |
			uint64(src[len(src)-32])<<000
		b4 := uint64(src[len(src)-33])<<070 |
			uint64(src[len(src)-34])<<060 |
			uint64(src[len(src)-35])<<050 |
			uint64(src[len(src)-36])<<040 |
			uint64(src[len(src)-37])<<030 |
			uint64(src[len(src)-38])<<020 |
			uint64(src[len(src)-39])<<010 |
			uint64(src[len(src)-40])<<000
		b5 := uint64(src[len(src)-41])<<070 |
			uint64(src[len(src)-42])<<060 |
			uint64(src[len(src)-43])<<050 |
			uint64(src[len(src)-44])<<040 |
			uint64(src[len(src)-45])<<030 |
			uint64(src[len(src)-46])<<020 |
			uint64(src[len(src)-47])<<010 |
			uint64(src[len(src)-48])<<000
		b6 := uint64(src[len(src)-49])<<070 |
			uint64(src[len(src)-50])<<060 |
			uint64(src[len(src)-51])<<050 |
			uint64(src[len(src)-52])<<040 |
			uint64(src[len(src)-53])<<030 |
			uint64(src[len(src)-54])<<020 |
			uint64(src[len(src)-55])<<010 |
			uint64(src[len(src)-56])<<000
		b7 := uint64(src[len(src)-57])<<070 |
			uint64(src[len(src)-58])<<060 |
			uint64(src[len(src)-59])<<050 |
			uint64(src[len(src)-60])<<040 |
			uint64(src[len(src)-61])<<030 |
			uint64(src[len(src)-62])<<020 |
			uint64(src[len(src)-63])<<010 |
			uint64(src[len(src)-64])<<000
		// XOr each byte with '\n'.
		b0 ^= '\n' * byteMask64
		b1 ^= '\n' * byteMask64
		b2 ^= '\n' * byteMask64
		b3 ^= '\n' * byteMask64
		b4 ^= '\n' * byteMask64
		b5 ^= '\n' * byteMask64
		b6 ^= '\n' * byteMask64
		b7 ^= '\n' * byteMask64
		// Set the top bit in each byte if it is zero.
		// https://graphics.stanford.edu/~seander/bithacks.html#ZeroInWord
		b0 = (b0 - byteMask64) &^ b0
		b1 = (b1 - byteMask64) &^ b1
		b2 = (b2 - byteMask64) &^ b2
		b3 = (b3 - byteMask64) &^ b3
		b4 = (b4 - byteMask64) &^ b4
		b5 = (b5 - byteMask64) &^ b5
		b6 = (b6 - byteMask64) &^ b6
		b7 = (b7 - byteMask64) &^ b7
		// Isolate the top bit of each byte.
		b0 &= byteMask64 << 7
		b1 &= byteMask64 << 7
		b2 &= byteMask64 << 7
		b3 &= byteMask64 << 7
		b4 &= byteMask64 << 7
		b5 &= byteMask64 << 7
		b6 &= byteMask64 << 7
		b7 &= byteMask64 << 7
		// Group the bits together backwards.
		const groupMul = 1 | 1<<007 | 1<<016 | 1<<025 | 1<<034 | 1<<043 | 1<<052 | 1<<061
		b0 *= groupMul
		b1 *= groupMul
		b2 *= groupMul
		b3 *= groupMul
		b4 *= groupMul
		b5 *= groupMul
		b6 *= groupMul
		b7 *= groupMul
		b0 &= 0xFF << 070
		b1 &= 0xFF << 070
		b2 &= 0xFF << 070
		b3 &= 0xFF << 070
		b4 &= 0xFF << 070
		b5 &= 0xFF << 070
		b6 &= 0xFF << 070
		b1 >>= 010
		b2 >>= 020
		b3 >>= 030
		b4 >>= 040
		b5 >>= 050
		b6 >>= 060
		b7 >>= 070
		mask := b0 | b1 | b2 | b3 | b4 | b5 | b6 | b7

		trail := uint(oldLen - len(src))
		for ; mask != 0; mask &= mask - 1 {
			// Assert that there is space in the lines list.
			// This is simpler than the lines[len(lines)-1] check.
			if len(lines) == 0 {
				panic("too many lines")
			}

			// Find the next newline in the mask.
			idx := uint(bits.TrailingZeros64(mask))

			// Add it to the lines list.
			lines[len(lines)-1] = uint32(trail + idx)
			lines = lines[:len(lines)-1]
		}
	}

	// Scan backwards in batches of 8.
	// Perf diff: 1.6 GB/s with, 677 MB/s without (Threadripper 2990WX, single thread)
	for ; len(src) >= 8; src = src[:len(src)-8] {
		// Load as a little-endian uint64.
		mask := uint64(src[len(src)-8])<<000 |
			uint64(src[len(src)-7])<<010 |
			uint64(src[len(src)-6])<<020 |
			uint64(src[len(src)-5])<<030 |
			uint64(src[len(src)-4])<<040 |
			uint64(src[len(src)-3])<<050 |
			uint64(src[len(src)-2])<<060 |
			uint64(src[len(src)-1])<<070

		// XOr each byte with '\n'.
		// This will remap the newline bytes to 0.
		mask ^= '\n' * byteMask64
		// Create a mask of 0 bytes.
		mask = ((mask - byteMask64) &^ mask) & (byteMask64 << 7)
		if mask == 0 {
			// There are no newlines in this batch.
			continue
		}

		// Convert to big-endian to reverse iteration order.
		mask = bits.ReverseBytes64(mask)
		trail := uint(oldLen - len(src))
		for {
			// Assert that there is space in the lines list.
			// This is simpler than the lines[len(lines)-1] check.
			if len(lines) == 0 {
				panic("too many lines")
			}

			// Find the next newline in the mask.
			idx := uint(bits.TrailingZeros64(mask)) / 8

			// Add it to the lines list.
			lines[len(lines)-1] = uint32(trail + idx)
			lines = lines[:len(lines)-1]

			// Remove it from the mask.
			mask &= mask - 1
			if mask == 0 {
				break
			}
		}
	}

	// Scan the remaining bytes for newlines.
	for ; len(src) > 0; src = src[:len(src)-1] {
		// Skip until the next newline.
		if src[len(src)-1] != '\n' {
			continue
		}

		// Assert that there is space in the lines list.
		// This is simpler than the lines[len(lines)-1] check.
		if len(lines) == 0 {
			panic("too many lines")
		}

		// Add it to the lines list.
		lines[len(lines)-1] = uint32(oldLen - len(src))
		lines = lines[:len(lines)-1]
	}

	// There should be no empty space left in the lines buffer.
	if len(lines) != 0 {
		panic("did not finish lines")
	}

	return lines[:count]
}

// OffsetToPosition converts the (reverse) offset to a line and column number.
// The line number starts at 1 and increments after every newline.
// The column number is the number of bytes since the start of the line plus 1.
func (lines LineTable) OffsetToPosition(offset uint32) (line, column uint32) {
	oldLines := lines
	var i uint
	for {
		mid := (i + uint(len(lines))) >> 1
		if mid >= uint(len(lines)) {
			// The compiler cannot prove that mid is inbounds if we use
			// i < len(lines) as loop condition. If i == len(lines), then mid
			// will be len(lines). We can combine the bounds check with the
			// loop exit condition by comparing mid instead of i.
			break
		}
		if lines[mid] >= offset {
			i = mid + 1
		} else {
			lines = lines[:mid]
		}
	}
	if i-1 < uint(len(oldLines)) {
		// We can combine the bounds check with the i > 0 check by comparing
		// after subtraction. If i=0, the subtraction will underflow and the
		// index will be out of bounds. Otherwise, the check behaves like
		// i <= len(oldLines).
		column = oldLines[i-1] - offset
	}
	return uint32(i) + 1, column + 1
}
