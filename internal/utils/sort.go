package utils

import (
	"sort"
	"strings"
)

// TypeRank returns a priority rank: lower is higher priority.
func TypeRank(t string) int {
	switch strings.ToLower(t) {
	case "core":
		return 0
	case "dep":
		return 1
	case "optional":
		return 2
	default:
		return 99
	}
}

// LessByTypeThenKey applies (type rank) then (alpha by key, case-insensitive).
func LessByTypeThenKey(typeA, typeB, keyA, keyB string) bool {
	rA, rB := TypeRank(typeA), TypeRank(typeB)
	if rA != rB {
		return rA < rB
	}
	return strings.ToLower(keyA) < strings.ToLower(keyB)
}

// SortByTypeAndKey sorts any slice using extractor funcs.
func SortByTypeAndKey[T any](xs []T, typeOf func(T) string, keyOf func(T) string) {
	sort.SliceStable(xs, func(i, j int) bool {
		return LessByTypeThenKey(typeOf(xs[i]), typeOf(xs[j]), keyOf(xs[i]), keyOf(xs[j]))
	})
}
