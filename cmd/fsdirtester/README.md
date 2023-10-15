# fsdirtester

Calling `fsdirtester` opens, reads from and closes a directory.
It's intended to supply valid expected behavior for memfs in situations where the Go ducmentation for `io/fs` is unclear.

## Expected

Calling `cd cmd/fsdirtester; go build . && ./fsdirtester`

### macos / M1

```
Open directory: err=<nil>, f=&{0x140000a6120}
Stat(): err=<nil> / err=%!q(<nil>), info=&{. 160 2147484141 {889396975 63831925382 0x1023b1a20} {16777231 16877 5 14343427 501 20 0 [0 0 0 0] {1696328582 929989847} {1696328582 889396975} {1696328582 889396975} {1696267228 17709420} 160 0 4096 0 0 0 [0 0]}}
Read(...): err=&fs.PathError{Op:"read", Path:".", Err:0x15} / err="read .: is a directory", n=0
ReadDir(-1) #1.1: err=<nil>, entries=[- fsdirtester - README.md - main.go]
ReadDir(1) #1.2: err=&errors.errorString{s:"EOF"}, entries=[]
ReadDir(1) #1.3: err=&errors.errorString{s:"EOF"}, entries=[]
ReadDir(2) #1.4: err=&errors.errorString{s:"EOF"}, entries=[]
Close() #1: err=<nil> / err=%!q(<nil>)
Read(...): err=&fs.PathError{Op:"read", Path:".", Err:(*errors.errorString)(0x1023ad9a0)} / err="read .: file already closed", n=0
Stat(): err=&fs.PathError{Op:"stat", Path:".", Err:(*errors.errorString)(0x1023adb80)} / err="stat .: use of closed file", info=<nil>
Close() #2: err=&fs.PathError{Op:"close", Path:".", Err:(*errors.errorString)(0x1023ad9a0)} / err="close .: file already closed"
ReadDir(1) #2.1: err=<nil>, entries=[- fsdirtester]
ReadDir(1) #2.2: err=<nil>, entries=[- README.md]
ReadDir(-1) #2.3: err=<nil>, entries=[- main.go]
ReadDir(1) #2.4: err=&errors.errorString{s:"EOF"}, entries=[]
ReadDir(-1) #2.5: err=<nil>, entries=[]
ReadDir(1) #2.6: err=&errors.errorString{s:"EOF"}, entries=[]
Seek(0,1): err=<nil> / err=%!q(<nil>), n=0
ReadDir(2) #3.1: err=<nil>, entries=[- fsdirtester - README.md]
Seek(0,0): err=<nil> / err=%!q(<nil>), n=0
ReadDir(0) #4.1: err=<nil>, entries=[- fsdirtester - README.md - main.go]
```

### Linux / amd64

```
Open directory: err=<nil>, f=&{0xc00007c180}
Stat(): err=<nil> / err=%!q(<nil>), info=&{. 5 2147484141 {361821934 63832008675 0x53eec0} {47 230042 2 16877 1000 1000 0 0 5 131072 1 {1696408869 119344247} {1696411875 361821934} {1696411875 361821934} [0 0 0]}}
Read(...): err=&fs.PathError{Op:"read", Path:".", Err:0x15} / err="read .: is a directory", n=0
ReadDir(-1) #1.1: err=<nil>, entries=[- README.md - fsdirtester - main.go]
ReadDir(1) #1.2: err=&errors.errorString{s:"EOF"}, entries=[]
ReadDir(1) #1.3: err=&errors.errorString{s:"EOF"}, entries=[]
ReadDir(2) #1.4: err=&errors.errorString{s:"EOF"}, entries=[]
Close() #1: err=<nil> / err=%!q(<nil>)
Read(...): err=&fs.PathError{Op:"read", Path:".", Err:(*errors.errorString)(0x53a860)} / err="read .: file already closed", n=0
Stat(): err=&fs.PathError{Op:"stat", Path:".", Err:(*errors.errorString)(0x53aa40)} / err="stat .: use of closed file", info=<nil>
Close() #2: err=&fs.PathError{Op:"close", Path:".", Err:(*errors.errorString)(0x53a860)} / err="close .: file already closed"
ReadDir(1) #2.1: err=<nil>, entries=[- README.md]
ReadDir(1) #2.2: err=<nil>, entries=[- fsdirtester]
ReadDir(-1) #2.3: err=<nil>, entries=[- main.go]
ReadDir(1) #2.4: err=&errors.errorString{s:"EOF"}, entries=[]
ReadDir(-1) #2.5: err=<nil>, entries=[]
ReadDir(1) #2.6: err=&errors.errorString{s:"EOF"}, entries=[]
Seek(0,1): err=<nil> / err=%!q(<nil>), n=0
ReadDir(2) #3.1: err=<nil>, entries=[- README.md - fsdirtester]
Seek(0,0): err=<nil> / err=%!q(<nil>), n=0
ReadDir(0) #4.1: err=<nil>, entries=[- README.md - fsdirtester - main.go]
```
