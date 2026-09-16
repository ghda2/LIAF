package checker_test

import (
	"liaf/pkg/checker"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
	"testing"
)

func TestSemanticFailures(t *testing.T) {
	cases := []struct{ body, effects, code string }{
		{`(do (call println missing))`, "io", "E_UNDEFINED_SYMBOL"},
		{`(let n int "text")`, "", "E_TYPE_MISMATCH"},
		{`(do (call println 1))`, "", "E_UNDECLARED_EFFECT"},
		{`(do (call fs-read-file "missing"))`, "fs", "E_UNHANDLED_RESULT"},
		{`(let r (result str str) (call fs-read-file "missing"))`, "fs", "E_UNHANDLED_RESULT"},
		{`(break)`, "", "E_LOOP_CONTROL"},
		{`(do (call str-from-int 1 2))`, "", "E_WRONG_ARITY"},
		{`(let items (list int) (call make-list int)) (do (call list-push items "bad"))`, "", "E_TYPE_MISMATCH"},
		{`(let c (chan int) (call make-chan int)) (send c "bad")`, "", "E_TYPE_MISMATCH"},
		{`(let r (result int str) (call ok "bad"))`, "", "E_TYPE_MISMATCH"},
		{`(let r (result int str) (call ok 1)) (if true (then (match r (ok n) (err e))))`, "", "E_UNHANDLED_RESULT"},
		{`(let r (result int str) (call ok 1)) (while false (match r (ok n) (err e)))`, "", "E_UNHANDLED_RESULT"},
		{`(let r (result int str) (call ok 1)) (if true (then (return))) (match r (ok n) (err e))`, "", "E_UNHANDLED_RESULT"},
	}
	for _, tc := range cases {
		t.Run(tc.code+tc.body, func(t *testing.T) {
			src := `(module test (fn main (params) (returns void) (effects ` + tc.effects + `) (body ` + tc.body + `)))`
			p := parser.New(lexer.New(src), "test.liaf")
			m := p.ParseModule()
			if len(p.Diagnostics) > 0 {
				t.Fatal(p.Diagnostics)
			}
			_, errors := checker.Check(m, "test.liaf")
			for _, e := range errors {
				if e.Code == tc.code {
					return
				}
			}
			t.Fatalf("expected %s, got %+v", tc.code, errors)
		})
	}
}

func TestMalformedInputTerminates(t *testing.T) {
	for _, src := range []string{
		`(module m (struct S (fields garbage)))`,
		`(module m (fn main (params) (returns void) (effects) (body garbage)))`,
		`(module m (fn main (params) (returns void) (effects) (body (do (call print @)))))`,
		`(module m (fn main (params) (returns void) (effects) (body (let x (list @) 1))))`,
		`(module m (fn main (params) (returns void) (effects) (body (do 9999999999999999999999999))))`,
	} {
		p := parser.New(lexer.New(src), "bad.liaf")
		if p.ParseModule() != nil || len(p.Diagnostics) == 0 {
			t.Fatalf("accepted %s", src)
		}
	}
}
