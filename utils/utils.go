package utils

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/coldstar-507/down4backend/server"
)

func Fatal(err error, errMsg string) {
	if err != nil {
		log.Fatalf(errMsg+": %v\n", err)
	}
}

func NonFatal(err error, errMsg string) {
	if err != nil {
		log.Printf(errMsg+": %v\n", err)
	}
}

type ComposedId struct {
	Region string
	Shard  int
	Unik   string
}

func Tailed(s string) (string, error) {
	if len(s) == 0 {
		return "", errors.New("empty string parameter in Tailed")
	}
	return s[:len(s)-1], nil
}

func ParseComposedId(s string) (*ComposedId, error) {
	if unik, reg, shrd, err := Decompose(s); err != nil {
		return nil, err
	} else {
		return &ComposedId{Unik: unik, Region: reg, Shard: shrd}, nil
	}

}

func ParseMediaId(s string) (*ComposedId, error) {
	if len(s) == 0 {
		return nil, errors.New("empty string parameter for ParseMediaId")
	}
	vals := strings.Split(s, "@")
	if len(vals) == 3 {
		if cp, err := ParseComposedId(vals[0]); err != nil {
			return nil, err
		} else {
			return cp, nil
		}
	} else {
		s_, err := Tailed(s)
		if err != nil {
			return nil, err
		}
		if cp, err := ParseComposedId(s_); err != nil {
			return nil, err
		} else {
			return cp, nil
		}
	}
}

func ParseSingleRoot(r string) (*ComposedId, error) {
	cps, err := ParseRoot(r)
	if err != nil {
		return nil, err
	}

	if len(cps) != 1 {
		return nil, errors.New("more than one root parsed in ParseSingleRoot")
	}

	return &cps[0], nil
}

func ParseRoot(r string) ([]ComposedId, error) {
	sp := strings.Split(r, "^")
	roots := make([]ComposedId, 0, 2)
	for _, x := range sp {
		t, err := Tailed(x)
		if err != nil {
			continue
		}
		u, r, s, err := Decompose(t)
		if err != nil {
			continue
		}
		roots = append(roots, ComposedId{Unik: u, Region: r, Shard: s})
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("invalid parameter=%v for ParseRoot", r)
	}
	return roots, nil
}

func RootOfComposedIds(cpIds []ComposedId) string {
	return strings.Join(Map(cpIds, func(cp ComposedId) string {
		return cp.ToString() + "r"
	}), "^")
}

func UnikRoot(ids []ComposedId) string {
	return strings.Join(Map(ids, func(rt ComposedId) string { return rt.Unik }), "^")
}

func ParseMessageId(s string) (string, string, string, []ComposedId, error) {
	t, err := Tailed(s)
	if err != nil {
		return "", "", "", nil, err
	}

	vals := strings.Split(t, "@")
	if len(vals) != 2 {
		err := fmt.Errorf("string parameter=%s for ParseMessageId invalid", s)
		return "", "", "", nil, err
	}

	unik, rootStr := vals[0], vals[1]
	composedIds, err := ParseRoot(rootStr)
	if err != nil {
		return "", "", "", nil, err
	}
	
	return unik, rootStr, UnikRoot(composedIds), composedIds, nil
}

func (c *ComposedId) ServerShard() server.ServerShard {
	return server.Client.Shards[c.Region][c.Shard]
}

func UnixMilli() int64 {
	return time.Now().UnixMilli()
}

func MakePushKey() string {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf, uint64(UnixMilli()))
	rand.Read(buf[8:])
	return base58.Encode(buf)
}

func ForEach[T any](l []T, f func(t T)) {
	for _, x := range l {
		f(x)
	}
}

func Map[T any, E any](l []T, f func(e T) E) []E {
	r := make([]E, len(l))
	for i, x := range l {
		r[i] = f(x)
	}
	return r
}

func CopyMap[K comparable, J any](src, dst map[K]J) {
	for k, v := range src {
		dst[k] = v
	}
}

func CopyMap_[K comparable, J any](src map[K]J) map[K]J {
	dst := make(map[K]J, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
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

func (c *ComposedId) ToString() string {
	istr := strconv.Itoa(c.Shard)
	return c.Unik + "-" + c.Region + "-" + istr
}

// Returns unique, region, iShard
func Decompose(id string) (string, string, int, error) {
	if len(id) == 0 {
		return "", "", 0, errors.New("emptry string parameter in Decompose")
	}
	vals := strings.Split(id, "-")
	if len(vals) != 3 {
		return "", "", 0, fmt.Errorf("id isn't a composedID: %v", id)
	}

	var uni, reg, shrd string = vals[0], vals[1], vals[2]
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
