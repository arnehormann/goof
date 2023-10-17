package memfis

import (
	"errors"
	"io/fs"
	"strings"
)

const pathSeparator = '/'

var (
	errClosed         = errors.New("file already closed")
	errStatClosed     = errors.New("use of closed file")
	errChangedRoot    = errors.New("subfs changed root directory")
	errNegativeOffset = errors.New("negative offset")
)

// nextSegment returns the next part of path up to and including a "/".
func nextSegment(path string) string {
	i := strings.IndexByte(path, pathSeparator)
	if i < 0 {
		return path
	}
	return path[:i+1]
}

// isDir reports if path is a directory
func isDir(path string) bool {
	return path == "" || path[len(path)-1] == pathSeparator
}

// toDir converts a name to a directory path in memfs representation
func toDir(path string) string {
	if path == "" {
		return path
	}
	if path[len(path)-1] == pathSeparator {
		return path
	}
	return path + string(pathSeparator)
}

// fsPath translates the internal path representation of memfs to the one required by io/fs.
// memfs uses "" instead of "." to represent the current directory and all directory names end in "/".
// io/fs forbids this.
func fsPath(path string) string {
	if path == "" {
		// change local directory representation of "" to "."
		return "."
	}
	if last := len(path) - 1; path[last] == pathSeparator {
		// truncate terminal "/" used by directories in memfs but disallowed in io/fs.
		return path[:last]
	}
	return path
}

// validPath reports if a path in the memfs internal representation is valid according to io/fs.
func validPath(path string) bool {
	if path == "." {
		// "." is not a valid name in the memfs internal representation
		return false
	}
	return fs.ValidPath(fsPath(path))
}

// lenCommon retrieves the index of the first byte diffrent in a and b.
func lenCommon(a, b string) int {
	var i int
	for i = 0; i < min(len(a), len(b)); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return i
}

// commonPath retrieves the common directory prefix including the trailing '/'.
func commonPath(a, b string) string {
	return a[:strings.LastIndexByte(a[:lenCommon(a, b)], pathSeparator)+1]
}

// increment a string s by 1 based on its binary representation.
// Increment can be used to search for upper boundaries.
func increment(s string) (string, bool) {
	b := []byte(s)
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == 255 {
			if i == 0 {
				// input string already had every bit set
				return s, false
			}
			b[i] = 0
			continue
		}
		b[i]++
		break
	}
	return string(b), true
}

// walk all directories and files in m and call fn with their rootpath.
func walk(rootpath string, fs []File, fn func(rootpath string)) {
	prevdir := rootpath
	for _, f := range fs {
		n := f.GetName()
		prevdir = commonPath(prevdir, n)
		o := len(prevdir)
		for {
			i := strings.IndexByte(n[o:], pathSeparator)
			if i < 0 {
				break
			}
			o += i + 1
			prevdir = n[:o]
			fn(prevdir)
		}
		fn(n)
	}
}

// fsPathError creates a fs.PathError using the io/fs conformant path
func fsPathError(op, fspath string, err error) *fs.PathError {
	return &fs.PathError{
		Op:   op,
		Path: fspath,
		Err:  err,
	}
}

// memPathError makes the mempath io/fs conformant and creates a fs.PathError
func memPathError(op, mempath string, err error) *fs.PathError {
	return &fs.PathError{
		Op:   op,
		Path: fsPath(mempath),
		Err:  err,
	}
}
