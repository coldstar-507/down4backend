package utils

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"
)



func CopyMap[K comparable, J any](src, dst map[K]J) {
	for k, v := range src {
		dst[k] = v
	}
}

func MaxKey[T comparable](m map[T]int) string {
	mr := func(a [2]interface{}, k T, v int) [2]interface{} {
		if v >= a[1].(int) {
			return [2]interface{}{k, v}
		} else {
			return a
		}
	}

	maxReg := MapReduce(m, [2]interface{}{"", 0}, mr)[0].(string)
	return maxReg
}

func RandomBytes(n int) []byte {
	buf := make([]byte, 0, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return buf
}

func MakeID(unik, region string, ishard int) string {
	istr := strconv.Itoa(ishard)
	return unik + "~" + region + "~" + istr
}

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

func Reduce[T any, E any](l []T, acc E, combine func(a E, b T) E) E {
	for _, x := range l {
		acc = combine(acc, x)
	}
	return acc
}

func MapReduce[K comparable, V any, R any](m map[K]V, acc R, combine func(a R, k K, v V) R) R {
	for k, v := range m {
		acc = combine(acc, k, v)
	}
	return acc
}

func Map[T any, E any](l []T, f func(e T) E) []E {
	r := make([]E, len(l))
	for i, x := range l {
		r[i] = f(x)
	}
	return r
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

func Every[T any](l []T, f func(T) bool) bool {
	for _, x := range l {
		if !f(x) {
			return false
		}
	}
	return true
}

func Any[T any](l []T, f func(T) bool) bool {
	for _, x := range l {
		if f(x) {
			return true
		}
	}
	return false
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
