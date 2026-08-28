package common

// Filter returns items for which keep returns true while preserving order. The
// input slice is never modified.
func Filter[T any](items []T, keep func(T) bool) []T {
	if len(items) == 0 {
		return []T{}
	}
	result := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

// Unique returns items with duplicate values removed while preserving the first
// occurrence order. The input slice is never modified.
func Unique[T comparable](items []T) []T {
	return UniqueBy(items, func(item T) T { return item })
}

// UniqueBy returns items with duplicate keys removed while preserving the first
// occurrence order. The input slice is never modified.
func UniqueBy[T any, K comparable](items []T, key func(T) K) []T {
	if len(items) == 0 {
		return []T{}
	}
	seen := make(map[K]struct{}, len(items))
	result := make([]T, 0, len(items))
	for _, item := range items {
		k := key(item)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		result = append(result, item)
	}
	return result
}
