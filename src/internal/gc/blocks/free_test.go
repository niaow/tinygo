package blocks

import (
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
	"unsafe"
)

// TestFreeListEmpty verifies that nothing can be popped from an empty free list.
func TestFreeListEmpty(t *testing.T) {
	t.Parallel()

	// An empty list should not pop anything.
	var list FreeList
	if list.pop(BytesPerBlock) != nil {
		t.Fatal("popped something from an empty free list")
	}
}

// TestFreeListSingle verifies that pushing and popping a single node works.
func TestFreeListSingle(t *testing.T) {
	t.Parallel()

	// Push and pop a single node.
	var list FreeList
	var a freeRange
	list.push(&a, BytesPerBlock)
	if got := list.pop(BytesPerBlock); got != unsafe.Pointer(&a) {
		t.Fatal("popped a never-inserted node")
	}
	if list.pop(BytesPerBlock) != nil {
		t.Fatal("free list should be empty")
	}
}

// TestFreeListTooLong verifies that a request longer than any inserted range fails.
func TestFreeListTooLong(t *testing.T) {
	t.Parallel()

	// Push a 2-block span, then try to pop a 3-block span.
	var list FreeList
	var two [2]freeRange
	list.push(&two[0], 2*BytesPerBlock)
	if list.pop(3*BytesPerBlock) != nil {
		t.Fatal("too-long pop succeeded")
	}
}

// TestFreeListFILO verifies that ranges are popped in reverse insertion order.
func TestFreeListFILO(t *testing.T) {
	t.Parallel()

	// Push 4 free nodes.
	var list FreeList
	var nodes [4]freeRange
	for i := range nodes {
		list.push(&nodes[i], BytesPerBlock)
	}

	// The nodes should be popped backwards.
	for i := len(nodes) - 1; i >= 0; i-- {
		expected := unsafe.Pointer(&nodes[i])
		got := list.pop(BytesPerBlock)
		if got != expected {
			t.Errorf("expected %p (index %d), but popped %p", expected, i, got)
		}
	}
}

// TestFreeListSplit verifies that free ranges can be split.
func TestFreeListSplit(t *testing.T) {
	t.Parallel()

	// Push a 4-block span.
	var list FreeList
	var nodes [4]freeRange
	list.push(&nodes[0], 4*BytesPerBlock)

	// Pop a single block.
	// This should use the first block in the inserted span.
	if got := list.pop(BytesPerBlock); got != unsafe.Pointer(&nodes[0]) {
		t.Fatal("incorrect popped block")
	}

	// Pop two blocks.
	// This should use the second and third blocks from the inserted span.
	if got := list.pop(2 * BytesPerBlock); got != unsafe.Pointer(&nodes[1]) {
		t.Fatal("incorrect popped block")
	}

	// Pop the remaining block.
	// This should use the last block in the inserted span.
	if got := list.pop(BytesPerBlock); got != unsafe.Pointer(&nodes[3]) {
		t.Fatal("incorrect popped block")
	}
}

func TestFreeListPriority(t *testing.T) {
	t.Parallel()

	// Push several ranges of varying lengths.
	var list FreeList
	var one freeRange
	list.push(&one, BytesPerBlock)
	var two [2]freeRange
	list.push(&two[0], 2*BytesPerBlock)
	var five [5]freeRange
	list.push(&five[0], 5*BytesPerBlock)

	// Pop the two-block range.
	if got := list.pop(2 * BytesPerBlock); got != unsafe.Pointer(&two[0]) {
		t.Fatal("incorrect popped block")
	}

	// Pop another two-block range.
	// This should split the five-block range.
	if got := list.pop(2 * BytesPerBlock); got != unsafe.Pointer(&five[0]) {
		t.Fatal("incorrect popped block")
	}

	// Pop the single block.
	if got := list.pop(BytesPerBlock); got != unsafe.Pointer(&one) {
		t.Fatal("incorrect popped block")
	}

	// Pop another single block.
	// This should split the remaining portion of the five-block range.
	if got := list.pop(BytesPerBlock); got != unsafe.Pointer(&five[2]) {
		t.Fatal("incorrect popped block")
	}

	// Pop another two-block range.
	// This should remove the remaining portion of the five-block range.
	if got := list.pop(2 * BytesPerBlock); got != unsafe.Pointer(&five[3]) {
		t.Fatal("incorrect popped block")
	}
}

func TestFreeListCount(t *testing.T) {
	cases := []struct {
		name    string
		sizes   []uintptr
		entries uintptr
		bytes   uintptr
	}{
		{
			name:    "Empty",
			sizes:   []uintptr{},
			entries: 0,
			bytes:   0,
		},
		{
			name: "Single",
			sizes: []uintptr{
				12,
			},
			entries: 1,
			bytes:   12,
		},
		{
			name: "Repeat",
			sizes: []uintptr{
				12,
				12,
				12,
			},
			entries: 3,
			bytes:   36,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var list FreeList
			for _, v := range c.sizes {
				list.push(&freeRange{}, v)
			}
			entries, bytes := list.Count()
			if entries != c.entries {
				t.Errorf("expected %d entries but got %d", c.entries, entries)
			}
			if bytes != c.bytes {
				t.Errorf("expected %d bytes but got %d", c.bytes, bytes)
			}
		})
	}
}

func BenchmarkFreeListBuildSingles(b *testing.B) {
	for i := 1; i <= 1<<24; i <<= 1 {
		width := i
		b.Run(strconv.FormatInt(int64(width), 10), func(b *testing.B) {
			src := make([]freeRange, i)
			b.SetBytes(int64(width) * int64(BytesPerBlock))
			for n := b.N; n > 0; n-- {
				var list FreeList
				for i := range src {
					list.push(&src[i], BytesPerBlock)
				}
			}
		})
	}
}

func BenchmarkFreeListBuildRandom(b *testing.B) {
	// Create a random bitmap.
	maxWidth := 1 << 20
	pcg := rand.PCG{}
	bitmap := make([]uint64, maxWidth)
	for i := range bitmap {
		bitmap[i] = pcg.Uint64()
	}

	for i := 1; i <= maxWidth; i <<= 1 {
		width := i
		type memListEntry struct {
			ptr  *freeRange
			size uintptr
		}
		var memList []memListEntry
		var buildOnce sync.Once
		b.Run(strconv.FormatInt(64*int64(width), 10), func(b *testing.B) {
			// Create a range list from the bitmap.
			buildOnce.Do(func() {
				bitmap := bitmap[:width]
				heap := make([]freeRange, 64*width)
				i := 0
				var curList []memListEntry
				for i < len(heap) {
					start := i
					for i < len(heap) && bitmap[i/64]&(uint64(1)<<(i%64)) != 0 {
						i++
					}
					end := i
					i++
					if start != end {
						curList = append(curList, memListEntry{
							ptr:  &heap[start],
							size: uintptr(end-start) * BytesPerBlock,
						})
					}
				}
				memList = curList
			})
			b.SetBytes(64 * int64(BytesPerBlock) * int64(width))
			b.ResetTimer()

			memList := memList
			for n := b.N; n > 0; n-- {
				var list FreeList
				for _, v := range memList {
					list.push(v.ptr, v.size)
				}
			}
		})
	}
}

// TODO: benchmark pop
