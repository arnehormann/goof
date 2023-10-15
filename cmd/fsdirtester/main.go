package main

import (
	"fmt"
	"io/fs"
	"os"
)

func main() {
	var (
		err  error
		f    *os.File
		n    int
		n64  int64
		info fs.FileInfo
		de   []fs.DirEntry
		d    = make([]byte, 1<<10)
	)
	f, err = os.Open(".")
	fmt.Printf("Open directory: err=%#v, f=%v\n", err, f)
	if err != nil {
		return
	}
	info, err = f.Stat()
	fmt.Printf("Stat(): err=%#v / err=%[1]q, info=%[2]v\n", err, info)
	n, err = f.Read(d)
	fmt.Printf("Read(...): err=%#v / err=%[1]q, n=%[2]v\n", err, n)
	de, err = f.ReadDir(-1)
	fmt.Printf("ReadDir(-1) #1.1: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(1)
	fmt.Printf("ReadDir(1) #1.2: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(1)
	fmt.Printf("ReadDir(1) #1.3: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(2)
	fmt.Printf("ReadDir(2) #1.4: err=%#v, entries=%v\n", err, de)
	err = f.Close()
	fmt.Printf("Close() #1: err=%#v / err=%[1]q\n", err)
	n, err = f.Read(d)
	fmt.Printf("Read(...): err=%#v / err=%[1]q, n=%[2]v\n", err, n)
	info, err = f.Stat()
	fmt.Printf("Stat(): err=%#v / err=%[1]q, info=%[2]v\n", err, info)
	err = f.Close()
	fmt.Printf("Close() #2: err=%#v / err=%[1]q\n", err)
	// 2nd attempt to reset ReadDir state
	f, _ = os.Open(".")
	de, err = f.ReadDir(1)
	fmt.Printf("ReadDir(1) #2.1: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(1)
	fmt.Printf("ReadDir(1) #2.2: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(-1)
	fmt.Printf("ReadDir(-1) #2.3: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(1)
	fmt.Printf("ReadDir(1) #2.4: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(-1)
	fmt.Printf("ReadDir(-1) #2.5: err=%#v, entries=%v\n", err, de)
	de, err = f.ReadDir(1)
	fmt.Printf("ReadDir(1) #2.6: err=%#v, entries=%v\n", err, de)
	f.Close()
	// does Seek work on directories? It apparently does
	f, _ = os.Open(".")
	n64, err = f.Seek(0, 1)
	fmt.Printf("Seek(0,1): err=%#v / err=%[1]q, n=%[2]v\n", err, n64)
	de, err = f.ReadDir(2)
	fmt.Printf("ReadDir(2) #3.1: err=%#v, entries=%v\n", err, de)
	n64, err = f.Seek(0, 0)
	fmt.Printf("Seek(0,0): err=%#v / err=%[1]q, n=%[2]v\n", err, n64)
	de, err = f.ReadDir(0)
	fmt.Printf("ReadDir(0) #4.1: err=%#v, entries=%v\n", err, de)
	f.Close()
}
