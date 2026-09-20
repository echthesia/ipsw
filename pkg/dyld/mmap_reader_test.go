//go:build unix || (windows && (amd64 || arm64))

package dyld

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMmapReaderAt(t *testing.T) {
	want := make([]byte, 3<<20+123) // spans several pages, odd tail
	for i := range want {
		want[i] = byte(i * 7)
	}
	path := filepath.Join(t.TempDir(), "cache")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	// Windows won't shrink a file while a view of it is live, even after the
	// handle is closed, so shrinking checks that a mapping was released.
	// (Deleting proves nothing: Windows 10+ allows it for mapped data files.)
	shrink := func() error { return os.Truncate(path, int64(len(want)-1)) }
	mapped := func() {
		t.Helper()
		if runtime.GOOS == "windows" && shrink() == nil {
			t.Fatal("shrank a mapped file, so the unmap checks prove nothing")
		}
	}
	released := func() {
		t.Helper()
		if err := shrink(); err != nil {
			t.Fatalf("file still mapped: %v", err)
		}
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	r, c, size, err := openCacheFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(want)) {
		t.Fatalf("size = %d, want %d", size, len(want))
	}
	data, ok := mappedBytes(r)
	if !ok || !bytes.Equal(data, want) {
		t.Fatalf("mappedBytes: ok=%v equal=%v", ok, bytes.Equal(data, want))
	}

	buf := make([]byte, 64)
	if n, err := r.ReadAt(buf, 1<<20); n != len(buf) || err != nil || !bytes.Equal(buf, want[1<<20:1<<20+64]) {
		t.Fatalf("ReadAt middle: n=%d err=%v", n, err)
	}
	tail := int64(len(want) - 10)
	if n, err := r.ReadAt(buf, tail); n != 10 || !errors.Is(err, io.EOF) || !bytes.Equal(buf[:n], want[tail:]) {
		t.Fatalf("ReadAt tail: n=%d err=%v", n, err)
	}
	if n, err := r.ReadAt(buf, size); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("ReadAt past end: n=%d err=%v", n, err)
	}
	mapped()
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	released()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a2s, err := a2sMmap(f, len(want))
	f.Close()
	if err != nil || !bytes.Equal(a2s, want) {
		t.Fatalf("a2sMmap: err=%v equal=%v", err, bytes.Equal(a2s, want))
	}
	mapped()
	if err := a2sMunmap(a2s); err != nil {
		t.Fatalf("a2sMunmap: %v", err)
	}
	released()
}
