package parse

import (
	"encoding/binary"
	"math/bits"
)

type identTable struct {
	// roots is a power-of-2 size hash table.
	// It holds the ending index of an entry in data.
	// Empty slots are set to 0.
	roots []uint32

	// data holds the strings and their attached metadata.
	// Each entry is len(str)+9 bytes:
	//  - the actual string data
	//  - little-endian uint32 token ID
	//  - little-endian uint32 ending offset of the next entry in the chain
	//  - a 0 byte
	// The length is not stored. It can be found by scanning backwards until the 0 byte from the previous entry.
	// The entries are stored in ascending token ID order.
	data []byte

	// last is greatest token ID in the table.
	last uint32
}

func (itab *identTable) Get(str []byte) uint32 {
	// Grow the roots table if the load factor is >= 1.
	// This ensures len(itab.roots) > 0, which removes the bounds check later.
	for uint(len(itab.roots)) <= uint(itab.last) {
		itab.reindex()
	}

	// Hash the string.
	var hash uint64
	hashRem := str
	for len(hashRem) >= 8 {
		// Load the batch into a uint64.
		batch := uint64(hashRem[len(hashRem)-8]) |
			uint64(hashRem[len(hashRem)-7])<<010 |
			uint64(hashRem[len(hashRem)-6])<<020 |
			uint64(hashRem[len(hashRem)-5])<<030 |
			uint64(hashRem[len(hashRem)-4])<<040 |
			uint64(hashRem[len(hashRem)-3])<<050 |
			uint64(hashRem[len(hashRem)-2])<<060 |
			uint64(hashRem[len(hashRem)-1])<<070

		// Hash the batch.
		hash = hash64(hash, batch)

		// Slice off the consumed bytes.
		hashRem = hashRem[:len(hashRem)-8]
	}
	var strLead uint64
	leadMask := uint64(1)<<(len(hashRem)<<3) - 1
	if cap(hashRem) >= 8 {
		// Load the first 8 bytes and mask off the extra.
		strLead = binary.LittleEndian.Uint64(hashRem[:8])
	} else {
		// Load individual bytes.
		for ; len(hashRem) > 0; hashRem = hashRem[:len(hashRem)-1] {
			strLead = strLead<<8 | uint64(hashRem[len(hashRem)-1])
		}
	}
	// Hash the final batch.
	hash = hash64(hash, strLead&leadMask)

	// Load the corresponding entry in the roots table.
	roots := itab.roots
	rootIdx := hash64Finish(hash) & uint(len(roots)-1)
	oldRoot := uint(roots[rootIdx])

	// Search for the string in the selected list.
	data := itab.data
	next := oldRoot
	for next <= uint(len(data)) {
		// Compare the string.
		entry := data[:next]
		if len(entry) < 9 {
			break
		}
		body := entry[:len(entry)-9]
		base := len(body) - len(str)
		if uint(base) < uint(len(body)) {
			other := body[base:]
			// Using an inline comparsion provides a ~17% speedup in overall parse speed.
			// TODO: The hash table has changed dramatically since this was measured, I should re-evaluate this.
		cmp:
			switch {
			case len(other) == len(str):
				// This is equivalent to len(body) - (len(body) - len(str)) == len(str).
				// The compiler should be able to prove this comparison, but it cannot.
				// This is necessay to prove the bounds checks in the loop.

				// Compare backwards in batches of 8.
				for i := len(other); i >= 8; i -= 8 {
					sb := uint64(str[i-8])<<000 |
						uint64(str[i-7])<<010 |
						uint64(str[i-6])<<020 |
						uint64(str[i-5])<<030 |
						uint64(str[i-4])<<040 |
						uint64(str[i-3])<<050 |
						uint64(str[i-2])<<060 |
						uint64(str[i-1])<<070
					ob := uint64(other[i-8])<<000 |
						uint64(other[i-7])<<010 |
						uint64(other[i-6])<<020 |
						uint64(other[i-5])<<030 |
						uint64(other[i-4])<<040 |
						uint64(other[i-3])<<050 |
						uint64(other[i-2])<<060 |
						uint64(other[i-1])<<070
					if sb != ob {
						break cmp
					}
				}

				// Compare the leading bytes.
				// NOTE: cap(other) is always > 8 (the 9-byte header is stored afer len), but the compiler cannot prove this.
				if (binary.LittleEndian.Uint64(other[:8])^strLead)&leadMask != 0 {
					break cmp
				}

				if base > 0 && body[base-1] != 0 {
					// The compared string is longer than the matched string.
					break cmp
				}

				// The strings are equal.
				// Return the ID stored in the header.
				return uint32(entry[next-9]) |
					uint32(entry[next-8])<<010 |
					uint32(entry[next-7])<<020 |
					uint32(entry[next-6])<<030
			}
		}

		// Fetch the index of the next entry.
		next = uint(entry[next-5]) |
			uint(entry[next-4])<<010 |
			uint(entry[next-3])<<020 |
			uint(entry[next-2])<<030
	}

	// Append an entry.
	data = append(data, str...)
	id := itab.last + 1
	itab.last = id
	// Keep these appends separate.
	// I attempted to merge them through a few methods:
	//  - Write out the full expression for each byte:
	//    This breaks the load combining logic, and the compiler emits separate
	//    stores for each byte.
	//  - Write the full header to a buffer and append the contents:
	//    The compiler stores id and oldRoot together and does a uint64 copy.
	//    (Most) CPUs cannot forward partial overlap, so they wait until the
	//    conflicting stores retire. This stall is more expensive than the
	//    extra capacity checks.
	data = binary.LittleEndian.AppendUint32(data, id)
	data = binary.LittleEndian.AppendUint32(data, uint32(oldRoot))
	data = append(data, 0)
	itab.data = data

	// TODO: handle len(data) > math.MaxUint32

	// Update the root.
	roots[rootIdx] = uint32(len(data))

	return id
}

func (itab *identTable) reindex() {
	newRoots := make([]uint32, uint64(1)<<uint(bits.Len32(itab.last)))
	rootMask := uint(len(newRoots) - 1)
	data := itab.data
	for len(data) >= 9 {
		// Slice off the string while hashing it.
		oldData := data
		data = data[:len(data)-9]
		var hash uint64
		var batch uint64
		for {
			// Load the batch into a uint64.
			if len(data) < 8 {
				// Load the first 8 bytes and mask off the extra.
				batch = binary.LittleEndian.Uint64(oldData[:8])
				batch &= 1<<(len(data)<<3) - 1
				break
			}
			batch = uint64(data[len(data)-8]) |
				uint64(data[len(data)-7])<<010 |
				uint64(data[len(data)-6])<<020 |
				uint64(data[len(data)-5])<<030 |
				uint64(data[len(data)-4])<<040 |
				uint64(data[len(data)-3])<<050 |
				uint64(data[len(data)-2])<<060 |
				uint64(data[len(data)-1])<<070

			// Check for zero bytes.
			zeroMask := ((batch - byteMask64) &^ batch) & (byteMask64 << 7)
			if zeroMask != 0 {
				// Shift off the preceding bytes before and including the zero.
				zeroLen := uint(bits.Len64(zeroMask))
				batch >>= zeroLen
				// Update the length.
				data = data[:uint(len(data))-(8-(zeroLen/8))]
				break
			}

			// Hash the batch.
			hash = hash64(hash, batch)

			// Slice off the processed bytes.
			data = data[:len(data)-8]
		}
		// Hash the final batch.
		hash = hash64(hash, batch)

		// Insert the string into the roots table.
		rootIdx := hash64Finish(hash) & rootMask
		prev := newRoots[rootIdx]
		newRoots[rootIdx] = uint32(len(oldData))
		oldData[len(oldData)-5] = byte(prev)
		oldData[len(oldData)-4] = byte(prev >> 010)
		oldData[len(oldData)-3] = byte(prev >> 020)
		oldData[len(oldData)-2] = byte(prev >> 030)
	}
	itab.roots = newRoots
}

// collisionDist collects metrics about hash collisions in the table.
// This is useful for evaluating changes to the hash function.
func (itab *identTable) collisionDist() ([]uint, float64, float64, float64) {
	var collisions []uint
	roots := itab.roots
	data := itab.data
	var comparsions uint64
	var entries uint
	for _, root := range roots {
		var n uint
		next := uint(root)
		for next <= uint(len(data)) {
			entry := data[:next]
			if len(entry) < 9 {
				break
			}
			n++
			comparsions += uint64(n)
			entries++
			next = uint(entry[next-5]) |
				uint(entry[next-4])<<010 |
				uint(entry[next-3])<<020 |
				uint(entry[next-2])<<030
		}
		for uint(len(collisions)) <= n {
			collisions = append(collisions, 0)
		}
		collisions[n]++
	}
	load := float64(entries) / float64(len(roots))
	avgCmp := float64(comparsions) / float64(entries)
	expAvgCmp := 1.0 + 0.5*load - (load / (2 * float64(entries)))
	return collisions, load, avgCmp, expAvgCmp
}

// hash64 updates the hash with a 64-bit batch.
// This hash function is cheap to inline.
func hash64(hash uint64, batch uint64) uint64 {
	// XOr the batch into the hash.
	hash ^= batch

	// XOr the upper 32 bits into the lower 32 bits of the hash.
	// This prevents their entropy from getting stuck at the top.
	hash ^= batch >> 32

	// Multiply by a fibonacci prime.
	hash *= 0x9E37_79B9_7F4A_7C15

	return hash
}

// hash64Finish converts the hash to an integer with entropy concentrated in the lower bits.
// The result can be masked to create an index into a hash table.
func hash64Finish(hash uint64) uint {
	// The upper bits in the hash have the highest entropy.
	// The root selection uses the lower bits of the result.
	// Reverse the bytes to move the entropy downwards.
	// Most CPUs have a byte reversal instruction, so this is cheap.
	// A bit reversal might work slightly better, but x86 does not have a bit-reversal instruction.
	return uint(bits.ReverseBytes64(hash))
}
