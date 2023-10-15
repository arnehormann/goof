package memfis

import (
	"cmp"
	"errors"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
)

type MemFS interface {
	// TODO
	// fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
	fs.StatFS
	fs.SubFS
}

type memFS struct {
	// files is authoritative and contains file entries sorted ascending by name (using cmp.Compare).
	// on creation, each file has to be checked with validPath.
	// If directories are ever supported, they are filenames with a terminal "/" are directories (content is ignored)
	files []File
	// rootpath is an optional subdirectory, it must end with "/" to be usable in length-based prefix cutting for e.g. Sub.
	rootpath string
}

var _ MemFS = (*memFS)(nil)

func MakeMemFS(files ...File) (MemFS, error) {
	fs := make([]File, len(files))
	copy(fs, files)
	for _, f := range fs {
		n := f.GetName()
		if isDir(n) && len(f.GetContent()) != 0 {
			// support empty directories with size 0 and name "" or ending in "/"
			return nil, errors.New("file ending with / is directory but has content: " + n)
		}
		if !validPath(n) {
			return nil, errors.New("unsupported file name " + n)
		}
	}
	if len(fs) <= 1 {
		// same return, but skips logic that's not needed in the no or one file case
		return &memFS{
			files: fs,
		}, nil
	}
	slices.SortStableFunc(fs, func(a, b File) int {
		return cmp.Compare(a.GetName(), b.GetName())
	})
	pn, dupe := "", false
	walk("", fs, func(rootpath string) {
		if dupe {
			// could alternatively also add return value to walk, but only needed here
			// and only for error cases, so all entries are processed
			return
		}
		if pn == rootpath {
			// duplicate filename
			dupe = true
			return
		}
		if isDir(rootpath) {
			// for a name collision with sorted names, pn must match rootpath up to the terminal slash
			if rootpath[:len(rootpath)-1] == pn {
				dupe = true
				return
			}
		}
		pn = rootpath
	})
	if dupe {
		return nil, errors.New("file names must be unique")
	}
	return &memFS{
		files: fs,
	}, nil
}

func (m *memFS) root(path string) string {
	if path == "." {
		return m.rootpath
	}
	return m.rootpath + path
}

func (m *memFS) rootdir(path string) string {
	return toDir(m.root(path))
}

// find reports if a file with GetName() == rootpath can be found in the sorted m.files.
// It will retrieve the index it would be found at (never negative) and whether it existed.
// idx will be in the inclusive interval [0, len(m.files)]
func (m *memFS) find(rootpath string) (idx int, found bool) {
	return slices.BinarySearchFunc(m.files, rootpath, func(f File, seek string) int {
		return cmp.Compare(f.GetName(), seek)
	})
}

// open returns the *memFile or *memReadableDir at rootpath
func (m *memFS) open(rootpath string) (*memFile, *memFS, error) {
	if rootpath == "" {
		// open current directory
		return nil, m, nil
	}
	low, lok := m.find(rootpath)
	if lok {
		// single file found
		file := makeFile(m.files[low])
		return file, nil, nil
	}
	numFiles := len(m.files)
	if numFiles <= low || !strings.HasPrefix(m.files[low].GetName(), rootpath) {
		// searched path not found
		return nil, nil, fs.ErrNotExist
	}
	high := numFiles
	// find high index by searching for ... path++
	if inc, ok := increment(rootpath); ok {
		// ok -> could increment, rootprefix was not already maximum
		high, _ = m.find(inc)
	}
	// must be directory
	fs := &memFS{
		files:    m.files[low:high],
		rootpath: toDir(rootpath),
	}
	return nil, fs, nil
}

func (m *memFS) Sub(dir string) (fs.FS, error) {
	rootdir := m.rootdir(dir)
	if rootdir != "" && !validPath(rootdir) {
		return nil, fsPathError("sub", dir, fs.ErrInvalid)
	}
	_, d, err := m.open(rootdir)
	if d == nil || err != nil {
		return nil, fsPathError("sub", dir, fs.ErrNotExist)
	}
	return d, nil
}

func (m *memFS) Open(name string) (fs.File, error) {
	rootpath := m.root(name)
	f, d, err := m.open(rootpath)
	if err != nil {
		return nil, fsPathError("open", name, err)
	}
	if d != nil {
		rd := &memReadableDir{
			fs: d,
		}
		return rd, nil
	}
	return f, nil
}

func (m *memFS) Stat(name string) (fs.FileInfo, error) {
	f, d, err := m.open(m.root(name))
	if err != nil {
		return nil, fsPathError("stat", name, err)
	}
	if d != nil {
		return makeRootDir(d.rootpath), nil
	}
	return f, nil
}

func (m *memFS) ReadFile(name string) ([]byte, error) {
	f, _, _ := m.open(m.root(name))
	if f == nil {
		return nil, fsPathError("readfile", name, fs.ErrNotExist)
	}
	return []byte(f.GetContent()), nil
}

func (m *memFS) ReadDir(name string) ([]fs.DirEntry, error) {
	_, d, _ := m.open(m.root(name))
	if d == nil {
		return nil, fsPathError("readdir", name, fs.ErrNotExist)
	}
	entries, _, err := d.dirEntries(nil, dirCursor{}, 0)
	return entries, err
}

func (m *memFS) Glob(pattern string) (matches []string, err error) {
	if _, err = path.Match(pattern, ""); err != nil {
		// path.Match documents that the only possible error is path.ErrBadPattern;
		// check pattern early to safely ignore err later
		return nil, fsPathError("glob", ".", err)
	}
	rpl := len(m.rootpath)
	walk(m.rootpath, m.files, func(rp string) {
		n := fsPath(rp[rpl:])
		if ok, _ := path.Match(pattern, n); ok {
			matches = append(matches, n)
		}
	})
	return matches, nil
}

// index into files; iterator state of dirEntries
type dirCursor struct {
	// last added directory, "" for none
	prev string
	// offset in memFS.files; either index of next valid entry or len(memFS.files) when done
	idx int
}

// dirEntries appends DirEntrys to entries starting at dc.idx.
// It will handle n just like fs.ReadDirFile.ReadDir does.
func (m *memFS) dirEntries(entries []fs.DirEntry, dc dirCursor, n int) ([]fs.DirEntry, dirCursor, error) {
	if dc.idx < 0 || dc.idx > len(m.files) {
		// return same dc, error state has to be handled by caller
		return nil, dc, fs.ErrInvalid
	}
	// SPEC unclear: expected error cases?
	// determined one valid behavior by cmd/fstester
	// - preserve already read directory state
	// - for n <= 0: return all remaining entries (can be none) but nil error
	// - for n > 0: return up to n entries starting at current state
	// - on first n > 0 with all files already read, return io.EOF error
	if dc.idx == len(m.files) {
		if n <= 0 {
			return []fs.DirEntry{}, dirCursor{idx: dc.idx}, nil
		}
		return []fs.DirEntry{}, dirCursor{idx: dc.idx}, io.EOF
	}
	ne := len(entries)
	rp := m.rootpath
	// TODO think of this in terms of walk
	for ; dc.idx < len(m.files); dc.idx++ {
		if n > 0 && len(entries) == ne+n {
			break
		}
		f := m.files[dc.idx]
		fn := f.GetName()
		// optional: skip directories after first occurance by binary search
		rest, ok := strings.CutPrefix(fn, rp)
		if !ok {
			// no longer in same rootpath, should not happen
			return nil, dc, errChangedRoot
		}
		next := nextSegment(rest)
		if dc.prev == next {
			continue
		}
		dc.prev = next
		if isDir(next) {
			entries = append(
				entries,
				memDir{
					rootpath: fn[:len(rp)+len(next)],
					pidx:     len(rp),
				},
			)
			continue
		}
		entries = append(entries, makeFile(f))
	}
	return entries, dc, nil
}
