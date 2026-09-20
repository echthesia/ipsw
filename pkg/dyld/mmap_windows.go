//go:build windows && (amd64 || arm64)

package dyld

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// mmapFile maps the first size bytes of f read-only. The view holds its own
// references to the mapping and the file, so both handles can be closed once
// it exists. 32-bit Windows reads the cache files instead (mmap_other.go): a
// cache's subcaches don't fit in its address space together.
func mmapFile(f *os.File, size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("mmap: invalid size %d", size)
	}
	h, err := syscall.CreateFileMapping(syscall.Handle(f.Fd()), nil, syscall.PAGE_READONLY, uint32(uint64(size)>>32), uint32(size), nil)
	if err != nil {
		return nil, os.NewSyscallError("CreateFileMapping", err)
	}
	defer syscall.CloseHandle(h)
	addr, err := syscall.MapViewOfFile(h, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		return nil, os.NewSyscallError("MapViewOfFile", err)
	}
	// addr is OS-mapped memory, not Go heap, so converting it is sound; vet's
	// unsafeptr check can't tell (x/exp/mmap and syscall.Mmap do the same).
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

func munmapFile(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return os.NewSyscallError("UnmapViewOfFile", syscall.UnmapViewOfFile(uintptr(unsafe.Pointer(&data[0]))))
}
