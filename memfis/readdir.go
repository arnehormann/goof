package memfis

import (
	"io/fs"
	"strings"
	"syscall"
)

// memReadableDir is a support data structure to represent virtual directories based on a SubFS.
type memReadableDir struct {
	// sub-fs inside directory
	fs *memFS
	// index into fs.files for ReadDir
	dc dirCursor
}

var _ fs.ReadDirFile = (*memReadableDir)(nil)

func (d *memReadableDir) closed() bool {
	return d.dc.idx < 0
}

// cwd retrieves the current working directory
func (d *memReadableDir) cwd() string {
	n := d.fs.rootpath
	if n == "" {
		return "."
	}
	n = n[:len(n)-1]
	return n[strings.LastIndexByte(n, pathSeparator)+1:]
}

func (d *memReadableDir) Stat() (fs.FileInfo, error) {
	if d.closed() {
		return nil, memPathError("stat", d.cwd(), errStatClosed)
	}
	return makeRootDir(d.fs.rootpath), nil
}

func (d *memReadableDir) Read(r []byte) (int, error) {
	// SPEC unclear: expected error for directories?
	// determined by cmd/fstester: the PathError below is a valid value
	if d.closed() {
		return 0, memPathError("read", d.cwd(), errClosed)
	}
	return 0, memPathError("read", d.cwd(), syscall.EISDIR)
}

// ResetReadDir reopens the directoriy and resets its internal ReadDir state.
func (d *memReadableDir) ResetReadDir() {
	d.dc = dirCursor{}
}

// Seek will reset non-closed directories for ReadDir.
func (d *memReadableDir) Seek(offset int64, whence int) (int64, error) {
	if d.closed() {
		// TODO validate this is okay behavior
		return 0, memPathError("seek", d.cwd(), errClosed)
	}
	// observed behavior on os.File: Seek on directory resets ReadDir and returns 0, nil
	d.ResetReadDir()
	return 0, nil
}

func (d *memReadableDir) Close() error {
	// SPEC unclear: expected error for directories?
	// valid variant determined by cmd/fstester:
	// return nil on first call, then PathError
	if d.closed() {
		return memPathError("close", d.cwd(), errClosed)
	}
	// make closed
	d.dc.idx = -1
	return nil
}

func (d *memReadableDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.closed() {
		return nil, memPathError("readdir", d.cwd(), errClosed)
	}
	de, dc, err := d.fs.dirEntries(nil, d.dc, n)
	if err != nil {
		return nil, err
	}
	d.dc = dc
	return de, nil
}
