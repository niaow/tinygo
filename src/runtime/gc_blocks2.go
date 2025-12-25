//go:build gc.conservative2 || gc.precise2

package runtime

import (
	"internal/gc/blocks"
	"internal/task"
	"runtime/interrupt"
	"unsafe"
)

const needsStaticHeap = true

const gcDebug = false

var gcLock task.PMutex
var freeList blocks.FreeList
var scanStack blocks.ScanStack
var heapLen uintptr

var gcMallocs uint64
var gcTotalAlloc uint64

// initHeap is called when the heap is first initialized at program start.
func initHeap() {
	heap := blocks.HeapFromRange(unsafe.Pointer(heapStart), unsafe.Pointer(heapEnd))
	heapLen = heap.Len
	heap.Init(&freeList)
}

// zeroSizedAlloc is just a sentinel that gets returned when allocating 0 bytes.
var zeroSizedAlloc uint8

// alloc is called to allocate memory.
//
//go:noinline
func alloc(size uintptr, layout unsafe.Pointer) unsafe.Pointer {
	if size == 0 {
		return unsafe.Pointer(&zeroSizedAlloc)
	}

	// Acquire a lock on the GC.
	gcLock.Lock()

	// Update the GC counters.
	gcMallocs++
	gcTotalAlloc += uint64(size)

	// Round up the allocation size.
	var overflow bool
	size, overflow = blocks.AllocSize(size)
	if overflow {
		// The size overflowed.
		runtimePanicAt(returnAddress(0), "out of memory")
	}

	// Load the heap bounds.
	var grewToFit bool
reload:
	heap := getHeap()

	// Try to allocate the requested size.
	// NOTE: ranGC is here because we need to re-run the GC after a grow.
	// TODO: Merge the trailing free range to avoid this?
	var ranGC bool
retry:
	ptr := heap.TryAlloc(&freeList, size, uintptr(layout))
	if ptr == nil {
		if !ranGC {
			// Run the collector.
			freeBytes := runGC()
			ranGC = true

			// Ensure there is at least 33% headroom.
			// This percentage was arbitrarily chosen, and may need to
			// be tuned in the future.
			heapSize := heapEnd - heapStart
			if freeBytes < heapSize/3 && growHeap() {
				// We grew the heap to fit the requested size.
				// Reload the heap and try again.
				goto reload
			}

			// Try allocating again.
			goto retry
		}

		// Grow the heap to fit the allocation.
		if gcDebug && !grewToFit {
			println("grow heap for request:", uint(size))
			freeList.Dump()
		}
		if growHeap() {
			// We grew the heap to fit this allocation.
			grewToFit = true
			// The heap must be reloaded after it is grown.
			goto reload
		}

		// Unfortunately the heap could not be increased. This
		// happens on baremetal systems for example (where all
		// available RAM has already been dedicated to the heap).
		runtimePanicAt(returnAddress(0), "out of memory")
	}

	// Release the GC lock now that we are done.
	gcLock.Unlock()

	// Exclude the metadata from the allocated size.
	size = blocks.InnerSize(size)

	// Clear the object.
	memzero(ptr, size)

	// Return the new pointer.
	return ptr
}

// free is called to explicitly free a previously allocated pointer.
func free(ptr unsafe.Pointer) {
	// This cannot be done.
}

//go:nobounds
func markRoots(start, end uintptr) {
	mem := unsafe.Slice((*byte)(unsafe.Pointer(start)), end-start)
	getHeap().MarkRoots(&scanStack, mem)
}

// mark a GC root at the address addr.
func markRoot(addr, root uintptr) {
	getHeap().Mark(&scanStack, root, addr)
}

// GC is called to explicitly run garbage collection.
func GC() {
	gcLock.Lock()
	runGC()
	gcLock.Unlock()
}

// runGC performs a garbage collection cycle. It is the internal implementation
// of the runtime.GC() function.
func runGC() uintptr {
	if gcDebug {
		println("running collection cycle...")
	}

	// Mark phase: mark all reachable objects, recursively.
	gcMarkReachable()

	heap := getHeap()
	if baremetal && hasScheduler {
		// Channel operations in interrupts may move task pointers around while we are marking.
		// Therefore we need to scan the runqueue separately.
		var markedTaskQueue task.Queue
	runqueueScan:
		runqueue := schedulerRunQueue()
		for !runqueue.Empty() {
			// Pop the next task off of the runqueue.
			t := runqueue.Pop()

			// Mark the task if it has not already been marked.
			heap.Mark(&scanStack, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(runqueue)))

			// Push the task onto our temporary queue.
			markedTaskQueue.Push(t)
		}

		heap.FinishScan(&scanStack)

		// Restore the runqueue.
		i := interrupt.Disable()
		if !runqueue.Empty() {
			// Something new came in while finishing the mark.
			interrupt.Restore(i)
			goto runqueueScan
		}
		*runqueue = markedTaskQueue
		interrupt.Restore(i)
	} else {
		heap.FinishScan(&scanStack)
	}

	// Sweep the heap, reclaiming unused space.
	return heap.Sweep(&freeList)
}

// SetFinalizer registers a finalizer.
func SetFinalizer(obj interface{}, finalizer interface{}) {
	// TODO
}

func markCurrentGoroutineStack(sp uintptr) {
	// This could be optimized by only marking the stack area that's currently
	// in use.
	markRoot(0, sp)
}

// ReadMemStats populates m with memory statistics.
func ReadMemStats(m *MemStats) {
	gcLock.Lock()

	// Calculate the raw size of the heap.
	heapEnd := heapEnd
	heapStart := heapStart
	heapSize := heapEnd - heapStart
	m.Sys = uint64(heapSize)
	heapLen := heapLen
	blockCount := heapLen * blocks.BlocksPerGroup
	dataBytes := blockCount * blocks.BytesPerBlock
	m.HeapSys = uint64(heapSize - dataBytes)
	// TODO: should GCSys include objHeaders?
	m.GCSys = uint64(heapLen * blocks.MetaSizePerGroup)
	m.HeapReleased = 0 // always 0, we don't currently release memory back to the OS.

	// Count free ranges.
	freeRanges, freeBytes := freeList.Count()
	m.HeapIdle = uint64(freeBytes)

	// Subtract the free size from the heap data size to get the live size.
	liveBytes := uint64(dataBytes - freeBytes)
	m.HeapInuse = liveBytes
	m.HeapAlloc = liveBytes
	m.Alloc = liveBytes

	// Count the current starts.
	starts := (blocks.Heap{
		Base: unsafe.Pointer(heapStart),
		Len:  heapLen,
	}).CountStarts()
	// Subtract the free ranges from the starts to count live objects.
	liveObjects := starts - freeRanges

	// Record the number of allocated objects.
	gcMallocs := gcMallocs
	m.Mallocs = gcMallocs

	// Subtract live objects from allocated objects to count freed objects.
	m.Frees = gcMallocs - uint64(liveObjects)

	// Record the total allocated bytes.
	m.TotalAlloc = gcTotalAlloc

	gcLock.Unlock()
}

func setHeapEnd(newHeapEnd uintptr) {
	oldHeap := getHeap()
	newHeap := blocks.HeapFromRange(unsafe.Pointer(heapStart), unsafe.Pointer(newHeapEnd))
	newHeap.Grow(oldHeap.Len, &freeList)
	heapLen = newHeap.Len
	heapEnd = newHeapEnd
}

func getHeap() blocks.Heap {
	return blocks.Heap{
		Base: unsafe.Pointer(heapStart),
		Len:  heapLen,
	}
}
