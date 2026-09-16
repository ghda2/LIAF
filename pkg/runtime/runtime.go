// Package runtime contains the typed standard library used by generated programs.
package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Result[T, E any] struct {
	OK    bool
	Value T
	Error E
}

func Ok[T, E any](v T) Result[T, E]  { return Result[T, E]{OK: true, Value: v} }
func Err[T, E any](e E) Result[T, E] { return Result[T, E]{Error: e} }
func fromError[T any](v T, e error) Result[T, string] {
	if e != nil {
		return Err[T, string](e.Error())
	}
	return Ok[T, string](v)
}

// Lists have reference semantics; passing a list to a function preserves appends.
type List[T any] struct{ Items []T }

func (l List[T]) MarshalJSON() ([]byte, error) {
	if l.Items == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(l.Items)
}

func (l *List[T]) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &l.Items)
}

func MakeList[T any]() *List[T]       { return &List[T]{Items: []T{}} }
func ListPush[T any](l *List[T], v T) { l.Items = append(l.Items, v) }
func ListGet[T any](l *List[T], i int64) Result[T, string] {
	if i < 0 || i >= int64(len(l.Items)) {
		return Err[T, string]("list index out of bounds")
	}
	return Ok[T, string](l.Items[i])
}
func ListSet[T any](l *List[T], i int64, v T) Result[bool, string] {
	if i < 0 || i >= int64(len(l.Items)) {
		return Err[bool, string]("list index out of bounds")
	}
	l.Items[i] = v
	return Ok[bool, string](true)
}
func MapGet[K comparable, V any](m map[K]V, k K) Result[V, string] {
	v, ok := m[k]
	if !ok {
		return Err[V, string]("map key not found")
	}
	return Ok[V, string](v)
}
func MapHas[K comparable, V any](m map[K]V, k K) bool { _, ok := m[k]; return ok }
func MapSet[K comparable, V any](m map[K]V, k K, v V) { m[k] = v }

func ReadFile(p string) Result[string, string] {
	b, e := os.ReadFile(p)
	return fromError(string(b), e)
}
func WriteFile(p, s string) Result[bool, string] {
	e := os.WriteFile(p, []byte(s), 0644)
	return fromError(e == nil, e)
}
func Rename(oldPath, newPath string) Result[bool, string] {
	e := os.Rename(oldPath, newPath)
	return fromError(e == nil, e)
}
func WriteAtomic(p, s string) Result[bool, string] {
	tmp := fmt.Sprintf("%s.tmp.%d", p, os.Getpid())
	if e := os.WriteFile(tmp, []byte(s), 0644); e != nil {
		return fromError(false, e)
	}
	e := os.Rename(tmp, p)
	if e != nil {
		_ = os.Remove(tmp)
	}
	return fromError(e == nil, e)
}
func Remove(p string) Result[bool, string] { e := os.Remove(p); return fromError(e == nil, e) }
func Exists(p string) bool                 { _, e := os.Stat(p); return e == nil }
func JSONEncode[T any](v T) Result[string, string] {
	b, e := json.Marshal(v)
	return fromError(string(b), e)
}
func JSONDecode[T any](s string) Result[T, string] {
	var v T
	e := json.Unmarshal([]byte(s), &v)
	return fromError(v, e)
}
func IntFromStr(s string) Result[int64, string] {
	v, e := strconv.ParseInt(s, 10, 64)
	return fromError(v, e)
}
func StrSlice(s string, start, end int64) Result[string, string] {
	if start < 0 || end < start || end > int64(len(s)) {
		return Err[string, string]("string byte range out of bounds")
	}
	return Ok[string, string](s[start:end])
}
func Args() *List[string] { return &List[string]{Items: append([]string{}, os.Args[1:]...)} }
func Failure(e error) Result[bool, string] {
	if e != nil {
		return Err[bool, string](fmt.Sprint(e))
	}
	return Ok[bool, string](true)
}
