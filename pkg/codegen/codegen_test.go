package codegen

import (
	"strings"
	"testing"

	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func TestCodegenMathExample(t *testing.T) {
	input := `
(module example
  (fn sum
    (params
      (a int)
      (b int))
    (returns int)
    (effects)
    (body
      (return (add a b))))
  (fn main
    (params)
    (returns void)
    (effects io)
    (body
      (do (call println (call sum 20 22))))))
`

	l := lexer.New(input)
	p := parser.New(l, "example.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnostics: %+v", p.Diagnostics)
	}

	gen := New(mod)
	code := gen.Generate()

	if !strings.Contains(code, "func sum(a int64, b int64) int64 {") {
		t.Errorf("assinatura de sum incorreta:\n%s", code)
	}
	if !strings.Contains(code, "return (a + b)") {
		t.Errorf("corpo de sum incorreto:\n%s", code)
	}
	if !strings.Contains(code, "fmt.Println(sum(20, 22))") {
		t.Errorf("chamada println(sum) incorreta:\n%s", code)
	}
}

func TestCodegenConcurrencyExample(t *testing.T) {
	input := `
(module concurrency-example
  (fn worker
    (params
      (channel (chan str))
      (id int))
    (returns void)
    (effects clock)
    (body
      (do (call sleep-ms 50))
      (send channel (call concat "worker-" (call str-from-int id)))))
  (fn main
    (params)
    (returns void)
    (effects io spawn)
    (body
      (let channel (chan str) (call make-chan str))
      (spawn (call worker channel 1))
      (let first str (recv channel))
      (do (call println first)))))
`

	l := lexer.New(input)
	p := parser.New(l, "concurrency.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("diagnostics: %+v", p.Diagnostics)
	}

	gen := New(mod)
	code := gen.Generate()

	if !strings.Contains(code, "func worker(channel chan string, id int64)") {
		t.Errorf("worker incorreto:\n%s", code)
	}
	if !strings.Contains(code, "channel <- _liaf_concat(\"worker-\", _liaf_str_from_int(int64(id)))") {
		t.Errorf("send incorreto:\n%s", code)
	}
	if !strings.Contains(code, "var channel chan string = make(chan string)") {
		t.Errorf("make chan incorreto:\n%s", code)
	}
	if !strings.Contains(code, "go worker(channel, 1)") {
		t.Errorf("spawn incorreto:\n%s", code)
	}
	if !strings.Contains(code, "var first string = <-channel") {
		t.Errorf("recv incorreto:\n%s", code)
	}
}
