package utils

func RemoveDuplicates[T comparable](in []T) []T {
	seen := make(map[T]bool)
	out := []T{}
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
