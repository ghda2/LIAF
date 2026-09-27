package runtime

import (
	"strings"
	"unicode/utf8"
)

func StrLen(s string) int64 {
	return int64(utf8.RuneCountInString(s))
}

func StrByteLen(s string) int64 {
	return int64(len(s))
}

func StrContains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func StrStartsWith(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func StrEndsWith(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}

func StrIndex(s, sub string) int64 {
	byteIdx := strings.Index(s, sub)
	if byteIdx < 0 {
		return -1
	}
	return int64(utf8.RuneCountInString(s[:byteIdx]))
}

func StrSplit(s, sep string) *List[string] {
	return &List[string]{Items: strings.Split(s, sep)}
}

func StrJoin(l *List[string], sep string) string {
	if l == nil || len(l.Items) == 0 {
		return ""
	}
	return strings.Join(l.Items, sep)
}

func StrTrim(s string) string {
	return strings.TrimSpace(s)
}

func StrUpper(s string) string {
	return strings.ToUpper(s)
}

func StrLower(s string) string {
	return strings.ToLower(s)
}

func StrReplace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func StrGet(s string, i int64) Result[string, string] {
	runes := []rune(s)
	if i < 0 || i >= int64(len(runes)) {
		return Err[string, string]("string index out of bounds")
	}
	return Ok[string, string](string(runes[i]))
}
