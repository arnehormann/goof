package upto

// UpTo returns a slice taking up no memory for use with
// for i := range UpTo(n) { ... }
func UpTo(i int) []struct{} {
	upto := [i]struct{}{}
	return upto[:]
}
