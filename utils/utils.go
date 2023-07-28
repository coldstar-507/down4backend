package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// Returns unique, region, iShard
func ParseID(id string) (string, string, int, error) {
	m := strings.Split(id, "~")
	if len(m) != 3 {
		return "", "", 0, fmt.Errorf("id isn't a composedID: %v", id)
	}

	var uni, reg, shrd string = m[0], m[1], m[2]
	iShrd, err := strconv.Atoi(shrd)
	if err != nil {
		return "", "", 0, err
	}

	return uni, reg, iShrd, nil
}

func Flatten[T any](m [][]T) []T {
	f := make([]T, 0)
	for _, l := range m {
		for _, x := range l {
			f = append(f, x)
		}
	}
	return f
}

// flattens matrix to array of uniques
func Flattenu[T comparable](m [][]T) []T {
	u := make([]T, 0)
	for _, l := range m {
		for _, x := range l {
			if !Contains(x, u) {
				u = append(u, x)
			}
		}
	}
	return u
}

func Contains[T comparable](e T, a []T) bool {
	for _, x := range a {
		if x == e {
			return true
		}
	}
	return false
}

func Unique[T comparable](l []T) []T {
	s := make([]T, 0, len(l))
	for _, x := range l {
		if !Contains(x, s) {
			s = append(s, x)
		}
	}
	return s
}

func Filter[T any](l []T, t func(e T) bool) []T {
	f := make([]T, 0, len(l))
	for _, x := range l {
		if t(x) {
			f = append(f, x)
		}
	}
	return f
}
