package utils

import "github.com/samber/lo"

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

func IsArrayChanged[T comparable](old []T, new []T) bool {
	changed := false
	for _, resource := range old {
		if !lo.Contains(new, resource) {
			changed = true
			break
		}
	}
	if len(old) == len(new) && !changed {
		return false
	}
	return true
}
