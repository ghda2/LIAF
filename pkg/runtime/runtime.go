// Package runtime contains the typed standard library used by generated programs.
package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
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

type Option[T any] struct {
	Some  bool
	Value T
}

func Some[T any](v T) Option[T] { return Option[T]{Some: true, Value: v} }
func None[T any]() Option[T]    { return Option[T]{Some: false} }

func UnwrapOrResult[T, E any](r Result[T, E], fallback T) T {
	if r.OK {
		return r.Value
	}
	return fallback
}

func UnwrapOrOption[T any](opt Option[T], fallback T) T {
	if opt.Some {
		return opt.Value
	}
	return fallback
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
func MapDelete[K comparable, V any](m map[K]V, k K)   { delete(m, k) }

func ListRemove[T any](l *List[T], i int64) Result[bool, string] {
	if i < 0 || i >= int64(len(l.Items)) {
		return Err[bool, string]("list index out of bounds")
	}
	l.Items = append(l.Items[:i], l.Items[i+1:]...)
	return Ok[bool, string](true)
}

func ListPop[T any](l *List[T]) Result[T, string] {
	n := len(l.Items)
	if n == 0 {
		return Err[T, string]("list is empty")
	}
	val := l.Items[n-1]
	l.Items = l.Items[:n-1]
	return Ok[T, string](val)
}

func ListSort[T any](l *List[T]) {
	if l == nil || len(l.Items) <= 1 {
		return
	}
	switch any(l.Items).(type) {
	case []int64:
		slices.Sort(any(l.Items).([]int64))
	case []float64:
		slices.Sort(any(l.Items).([]float64))
	case []string:
		slices.Sort(any(l.Items).([]string))
	}
}

func ListContains[T comparable](l *List[T], x T) bool {
	if l == nil {
		return false
	}
	for _, item := range l.Items {
		if item == x {
			return true
		}
	}
	return false
}

func MapKeys[K comparable, V any](m map[K]V) *List[K] {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if len(keys) > 1 {
		switch any(keys).(type) {
		case []int64:
			slices.Sort(any(keys).([]int64))
		case []float64:
			slices.Sort(any(keys).([]float64))
		case []string:
			slices.Sort(any(keys).([]string))
		case []bool:
			slices.SortFunc(any(keys).([]bool), func(a, b bool) int {
				if !a && b {
					return -1
				}
				if a && !b {
					return 1
				}
				return 0
			})
		}
	}
	return &List[K]{Items: keys}
}

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
	v, e := strconv.ParseInt(s, 0, 64)
	if e != nil {
		return Err[int64, string](fmt.Sprintf("int-from-str: %q nao e um inteiro valido", s))
	}
	return Ok[int64, string](v)
}
func FloatFromStr(s string) Result[float64, string] {
	v, e := strconv.ParseFloat(s, 64)
	if e != nil {
		return Err[float64, string](fmt.Sprintf("float-from-str: %q nao e um float valido", s))
	}
	return Ok[float64, string](v)
}
func BoolFromStr(s string) Result[bool, string] {
	if s == "true" {
		return Ok[bool, string](true)
	}
	if s == "false" {
		return Ok[bool, string](false)
	}
	return Err[bool, string](fmt.Sprintf("bool-from-str: %q nao e um booleano valido", s))
}
func StrFromFloat(x float64) string {
	return strconv.FormatFloat(x, 'g', -1, 64)
}
func StrFromBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func StrSlice(s string, start, end int64) Result[string, string] {
	runes := []rune(s)
	n := int64(len(runes))
	if start < 0 || end < start || end > n {
		return Err[string, string]("string range out of bounds")
	}
	return Ok[string, string](string(runes[start:end]))
}
func Args() *List[string] { return &List[string]{Items: append([]string{}, os.Args[1:]...)} }
func Failure(e error) Result[bool, string] {
	if e != nil {
		return Err[bool, string](fmt.Sprint(e))
	}
	return Ok[bool, string](true)
}

func NowMs() int64 {
	return time.Now().UnixMilli()
}

func RandInt(n int64) int64 {
	if n <= 0 {
		fmt.Fprintln(os.Stderr, "liaf: n <= 0 em rand-int")
		os.Exit(1)
	}
	return rand.Int64N(n)
}

func EnvGet(name string) Result[string, string] {
	v, ok := os.LookupEnv(name)
	if !ok {
		return Err[string, string](fmt.Sprintf("env-get: %q nao definida", name))
	}
	return Ok[string, string](v)
}

var (
	stdinReader = bufio.NewReader(os.Stdin)
	stdinMutex  sync.Mutex
)

func ReadLine() Result[string, string] {
	stdinMutex.Lock()
	defer stdinMutex.Unlock()
	line, err := stdinReader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return Err[string, string]("read-line: eof")
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return Ok[string, string](line)
}

func Exit(code int64) {
	os.Exit(int(code))
}
