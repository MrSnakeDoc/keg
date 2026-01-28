package utils

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func Filter[T any](in []T, keep func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func Map[T any, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func Some[T any](in []T, pred func(T) bool) bool {
	for _, v := range in {
		if pred(v) {
			return true
		}
	}
	return false
}

func Includes[T comparable](in []T, target T) bool {
	for _, v := range in {
		if v == target {
			return true
		}
	}
	return false
}

func Flat[T any](slices [][]T) []T {
	total := 0
	for _, s := range slices {
		total += len(s)
	}
	out := make([]T, 0, total)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}
