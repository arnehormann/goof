package memfis

import (
	"testing"
	"testing/fstest"
)

type tfile struct {
	all  string
	cidx int
}

var _ File = tfile{}

func (t tfile) GetName() string {
	return t.all[:t.cidx]
}

func (t tfile) GetContent() string {
	return t.all[t.cidx:]
}

func makeFiles(nameContentPairs ...string) []File {
	if len(nameContentPairs)%2 != 0 {
		panic("nameConentPairs must have an even number of arguments (it's *pairs*)")
	}
	f := make([]File, len(nameContentPairs)/2)
	for i, _ := range f {
		name, content := nameContentPairs[i*2], nameContentPairs[i*2+1]
		f[i] = tfile{
			all:  name + content,
			cidx: len(name),
		}
	}
	return f
}

func getNames(nameContentPairs ...string) []string {
	if len(nameContentPairs)%2 != 0 {
		panic("nameConentPairs must have an even number of arguments (it's *pairs*)")
	}
	n := make([]string, len(nameContentPairs)/2)
	for i, _ := range n {
		n[i] = nameContentPairs[i*2]
	}
	return n
}

func testFS(t *testing.T, nameContentPairs ...string) {
	names := getNames(nameContentPairs...)
	files := makeFiles(nameContentPairs...)
	fs, err := MakeMemFS(files...)
	if err != nil {
		t.Fatalf("file system creation failed: %v\n", err)
	}
	err = fstest.TestFS(fs, names...)
	if err != nil {
		t.Fatalf("file system test failed: %v\n", err)
	}
}

func TestMemFS(t *testing.T) {
	// TODO test same-named path elements occuring as dir and file or multiple times
	testFS(t,
		"a/a", "Hello",
		"a/b/c", "",
		"a/b/d", "123",
		"a/c/a", "Hi",
		"b", "",
		"c/a/b/c/d/e", "",
		"c/a/b/d/d/e", "",
		"c/a/b/d/f", "",
	)
}

func TestMemFSFilenameCollision(t *testing.T) {
	// file name a is not unique
	_, err := MakeMemFS(makeFiles(
		"a", "Hi",
		"a", "Ho",
	)...)
	if err == nil {
		t.Fatalf("MakeMemFS created two files with identical names. Names must be unique")
	}
}

func TestMemFSFileAndDirnameCollision(t *testing.T) {
	// file name a is also directory
	_, err := MakeMemFS(makeFiles(
		"a", "Hi",
		"a/a", "Ho",
	)...)
	if err == nil {
		t.Fatalf("MakeMemFS created a directory named like a file. Names must be unique")
	}
}
