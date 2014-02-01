package goof

var upto = [(1 << 31) - 1]struct{}{}

func UpTo(i int) []struct{} {
	return upto[:i]
}
