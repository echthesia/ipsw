//go:build windows && (amd64 || arm64)

package dyld

import "os"

func a2sMmap(f *os.File, size int) ([]byte, error) {
	return mmapFile(f, size)
}

func a2sMunmap(data []byte) error {
	return munmapFile(data)
}
