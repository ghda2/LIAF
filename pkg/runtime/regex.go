package runtime

import (
	"regexp"
	"sync"
)

var (
	regexCache   sync.Map // pattern -> *regexp.Regexp
	regexCompile = regexp.Compile
)

func getCompiledRegex(pattern string) (*regexp.Regexp, error) {
	if val, ok := regexCache.Load(pattern); ok {
		return val.(*regexp.Regexp), nil
	}
	re, err := regexCompile(pattern)
	if err != nil {
		return nil, err
	}
	regexCache.Store(pattern, re)
	return re, nil
}

// RegexMatch verifica se o texto satisfaz o padrao regular (RE2 seguro).
func RegexMatch(pattern, text string) Result[bool, string] {
	re, err := getCompiledRegex(pattern)
	if err != nil {
		return Err[bool, string](err.Error())
	}
	return Ok[bool, string](re.MatchString(text))
}

// RegexFind localiza a primeira ocorrencia do padrao no texto.
func RegexFind(pattern, text string) Result[Option[string], string] {
	re, err := getCompiledRegex(pattern)
	if err != nil {
		return Err[Option[string], string](err.Error())
	}
	loc := re.FindString(text)
	if loc == "" && !re.MatchString(text) {
		return Ok[Option[string], string](None[string]())
	}
	return Ok[Option[string], string](Some[string](loc))
}

// RegexFindAll localiza todas as ocorrencias do padrao no texto.
func RegexFindAll(pattern, text string) Result[*List[string], string] {
	re, err := getCompiledRegex(pattern)
	if err != nil {
		return Err[*List[string], string](err.Error())
	}
	matches := re.FindAllString(text, -1)
	if matches == nil {
		matches = []string{}
	}
	return Ok[*List[string], string](&List[string]{Items: matches})
}

// RegexReplace substitui todas as ocorrencias do padrao por repl.
func RegexReplace(pattern, text, repl string) Result[string, string] {
	re, err := getCompiledRegex(pattern)
	if err != nil {
		return Err[string, string](err.Error())
	}
	return Ok[string, string](re.ReplaceAllString(text, repl))
}
