package utils

func IsPrefixOf[T comparable](arr []T, test []T) bool {
	if len(test) > len(arr) {
		return false
	}
	for i, t := range test {
		if arr[i] != t {
			return false
		}
	}
	return true
}
