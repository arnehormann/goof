package memfis

import (
	"io/fs"
	"strings"
	"time"
)

const (
	// directory with read/write/enter for users and read/enter for group members
	modeDir fs.FileMode = fs.ModeDir | 0o750
	typeDir             = modeDir & fs.ModeType
)

type memDir struct {
	rootpath string
	// start index of relative path in rootpath
	pidx int
}

func makeRootDir(rootpath string) memDir {
	if rootpath == "" {
		return memDir{}
	}
	return memDir{
		rootpath: rootpath,
		pidx:     strings.LastIndexByte(rootpath[:len(rootpath)-1], pathSeparator) + 1,
	}
}

var (
	_ File        = memDir{}
	_ fs.DirEntry = memDir{}
	_ fs.FileInfo = memDir{}
)

func (d memDir) Valid() bool {
	return d.pidx >= 0 && d.pidx < len(d.rootpath) && validPath(d.rootpath)
}

func (d memDir) GetName() string {
	return d.rootpath
}

func (d memDir) GetContent() string {
	return ""
}

func (d memDir) Name() string {
	ns := nextSegment(d.rootpath[d.pidx:])
	if ns == "" {
		return "."
	}
	return ns[:len(ns)-1]
}

func (d memDir) IsDir() bool {
	return true
}

func (d memDir) Type() fs.FileMode {
	return typeDir
}

func (d memDir) Info() (fs.FileInfo, error) {
	return d, nil
}

func (d memDir) Size() int64 {
	return 0
}

func (d memDir) Mode() fs.FileMode {
	return modeDir
}

func (d memDir) ModTime() time.Time {
	return time.Time{}
}

func (d memDir) Sys() any {
	return nil
}
