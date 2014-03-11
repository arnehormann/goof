package goof

// UpTo returns a slice taking up no memory for use with
// for i := range UpTo(n) { ... }
func UpTo(i int) []struct{} {
	upto := [^uint(0) >> 1]struct{}{}
	return upto[:i]
}
