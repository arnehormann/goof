package memfis

import (
	"io"
	"io/fs"
	"strings"
	"time"
)

// File is a minimal representation of a file that is read only and provides only its name and contents.
// It is a bad fit for large files as its only way to access file contents is retrieving all of it as a string.
type File interface {
	// GetName returns the root-relative file path.
	// The path must use "/" as directory separator.
	// It must not use escaping or "." and ".." segements.
	// The characters "/" and "\" are forbidden in path segments.
	GetName() string
	// GetContent returns the data contained in the file.
	GetContent() string

	// function names were chosen to directly support
	// google.golang.org/protobuf/types/pluginpb/CodeGeneratorResponse_File
}

// FileSizer is a file that supports direct retrieval of the file size.
type FileSizer interface {
	File
	// Size retrieves the file size in bytes; it must match len(GetContent())
	Size() int64
}

// fileSize retrieves the size of a file using Size() for FileSizer.
func fileSize(f File) int64 {
	if fs, ok := f.(FileSizer); ok {
		return fs.Size()
	}
	return int64(len(f.GetContent()))
}

const (
	// regular file with read/write for users and read for group members
	modeFile fs.FileMode = 0o640
	typeFile             = modeFile & fs.ModeType
)

// memFile is a File with optionally supplied FileInfo.
// Reading, seeking and Size will use File.GetContent a lot,
// file should ideally provide a cheap and fast implementation.
type memFile struct {
	file File
	name string
	// offset into file.GetContent(), negative on close
	ridx int
}

// for convenience reasons, required interfaces are all implemented by the same read-only
// data structure. In contrast to memDir, memFile stores state to support reading the content slice.
var (
	_ File          = (*memFile)(nil)
	_ fs.DirEntry   = (*memFile)(nil)
	_ fs.File       = (*memFile)(nil)
	_ fs.FileInfo   = (*memFile)(nil)
	_ io.ReadSeeker = (*memFile)(nil)
	_ io.ReaderAt   = (*memFile)(nil)
	_ io.WriterTo   = (*memFile)(nil)
)

func makeFile(file File) *memFile {
	n := file.GetName()
	return &memFile{
		file: file,
		name: n[strings.LastIndexByte(n, pathSeparator)+1:],
	}
}

func (f *memFile) GetName() string {
	return f.file.GetName()
}

func (f *memFile) GetContent() string {
	return f.file.GetContent()
}

func (f *memFile) Name() string {
	return f.name
}

func (f *memFile) Size() int64 {
	return fileSize(f.file)
}

func (f *memFile) Mode() fs.FileMode {
	return modeFile
}

func (f *memFile) ModTime() time.Time {
	return time.Time{}
}

func (f *memFile) IsDir() bool {
	return false
}

func (f *memFile) Sys() any {
	return nil
}

func (m *memFile) Type() fs.FileMode {
	return typeFile
}

func (m *memFile) Info() (fs.FileInfo, error) {
	return m, nil
}

func (f *memFile) isClosed() bool {
	return f.ridx < 0
}

func (f *memFile) Close() error {
	// ridx < 0 as close marker; alternative >= len(f.GetContent()) requires more calls
	f.ridx = -1
	return nil
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	if f.isClosed() {
		return nil, fsPathError("stat", f.Name(), fs.ErrClosed)
	}
	return f, nil
}

func (f *memFile) Read(r []byte) (int, error) {
	if f.isClosed() {
		return 0, fsPathError("read", f.Name(), fs.ErrClosed)
	}
	data := f.file.GetContent()
	if f.ridx >= len(data) {
		return 0, io.EOF
	}
	n := copy(r, data[f.ridx:])
	f.ridx += n
	return n, nil
}

func (f *memFile) ReadAt(r []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, fsPathError("readat", f.Name(), errNegativeOffset)
	}
	// path errors with "read" instead of "readat" is aligned with os.File
	if f.isClosed() {
		return 0, fsPathError("read", f.Name(), fs.ErrClosed)
	}
	data := f.GetContent()
	o := int(off)
	if o > len(data) {
		return 0, fsPathError("read", f.Name(), io.ErrUnexpectedEOF)
	}
	n = copy(r, data[o:])
	if n < len(r) {
		return n, io.EOF
	}
	return n, nil
}

func (f *memFile) WriteTo(w io.Writer) (n int64, err error) {
	if f.isClosed() {
		return 0, fsPathError("read", f.Name(), fs.ErrClosed)
	}
	i, err := io.WriteString(w, f.GetContent())
	f.ridx += i
	if err != nil {
		return int64(i), fsPathError("read", f.Name(), err)
	}
	return int64(i), nil
}

func (f *memFile) Seek(offset int64, whence int) (int64, error) {
	if f.isClosed() {
		return 0, fsPathError("seek", f.Name(), fs.ErrClosed)
	}
	data := f.GetContent()
	var ridx int64
	switch whence {
	case io.SeekStart:
		ridx = offset
	case io.SeekCurrent:
		ridx = int64(f.ridx) + offset
	case io.SeekEnd:
		ridx = int64(len(data)) + offset
	default:
		return 0, fsPathError("seek", f.Name(), fs.ErrInvalid)
	}
	if ridx < 0 || ridx > int64(len(data)) {
		return 0, fsPathError("seek", f.Name(), fs.ErrInvalid)
	}
	f.ridx = int(ridx)
	// int64 vs int may overflow on 32 bit systems but this keeps it consistent
	// and the api does not support anything sensible with len vs Write vs Seek
	return int64(f.ridx), nil
}
