package typst

import (
	"bytes"
	"context"
	"crypto/sha256"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testOnce sync.Once
	testEng  *Engine
	testErr  error
)

// engine compartilha um Engine entre os testes: compilar o motor é a parte cara.
func engine(t *testing.T) *Engine {
	t.Helper()
	testOnce.Do(func() {
		start := time.Now()
		testEng, testErr = New(context.Background(), Options{})
		t.Logf("motor pronto em %v", time.Since(start))
	})
	if testErr != nil {
		t.Fatal(testErr)
	}
	return testEng
}

func src(s string) map[string][]byte { return map[string][]byte{"/main.typ": []byte(s)} }

const hello = `#set page(paper: "a4")
#set text(font: "Inter", lang: "pt")
= Olá, LIAF
Acentuação: ação, coração, pão, avó.`

func TestRenderPDF(t *testing.T) {
	eng := engine(t)
	start := time.Now()
	res, err := eng.Render(context.Background(), Job{Files: src(hello)})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("primeiro PDF em %v", time.Since(start))
	if err := res.Err(); err != nil {
		t.Fatal(err)
	}
	if len(res.Outputs) != 1 || !bytes.HasPrefix(res.Outputs[0], []byte("%PDF-")) {
		t.Fatalf("esperava um PDF, veio %d saídas", len(res.Outputs))
	}
	if len(res.Pages) != 1 || res.Pages[0].WidthPt < 595 || res.Pages[0].WidthPt > 596 {
		t.Fatalf("página A4 esperada, veio %+v", res.Pages)
	}
	for _, w := range res.Warnings() {
		t.Errorf("aviso inesperado: %s", w)
	}

	start = time.Now()
	if _, err := eng.Render(context.Background(), Job{Files: src(hello)}); err != nil {
		t.Fatal(err)
	}
	t.Logf("mesmo PDF de novo em %v", time.Since(start))
}

func TestRenderDeterministic(t *testing.T) {
	eng := engine(t)
	job := Job{Files: src(hello), Ident: "teste"}
	a, err := eng.Render(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	b, err := eng.Render(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(a.Outputs[0]) != sha256.Sum256(b.Outputs[0]) {
		t.Fatal("a mesma entrada gerou PDFs diferentes")
	}
}

func TestRenderDiagnostics(t *testing.T) {
	eng := engine(t)
	res, err := eng.Render(context.Background(), Job{Files: src("linha 1\n#let x = (1, 2\n")})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatal("sintaxe inválida foi aceita")
	}
	d := res.Diagnostics[0]
	if d.File != "/main.typ" || d.Line != 2 {
		t.Fatalf("posição errada: %+v", d)
	}
}

func TestRenderDataFile(t *testing.T) {
	eng := engine(t)
	files := map[string][]byte{
		"/main.typ":   []byte(`#let d = json("/dados.json")` + "\n" + `Nome: #d.nome`),
		"/dados.json": []byte(`{"nome": "Ana"}`),
	}
	res, err := eng.Render(context.Background(), Job{Files: files, Format: FormatCheck})
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Err(); err != nil {
		t.Fatal(err)
	}
	if len(res.Outputs) != 0 || len(res.Pages) != 1 {
		t.Fatalf("check não deveria gerar saída: %d saídas, %d páginas", len(res.Outputs), len(res.Pages))
	}
}

func TestRenderPNG(t *testing.T) {
	eng := engine(t)
	res, err := eng.Render(context.Background(), Job{Files: src(hello), Format: FormatPNG, PPI: 72})
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Err(); err != nil {
		t.Fatal(err)
	}
	if len(res.Outputs) != 1 || !bytes.HasPrefix(res.Outputs[0], []byte("\x89PNG")) {
		t.Fatal("esperava um PNG")
	}
}

func TestRenderPackageUnavailable(t *testing.T) {
	eng := engine(t)
	res, err := eng.Render(context.Background(), Job{Files: src(`#import "@preview/foo:0.1.0": *`)})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || !strings.Contains(res.Err().Error(), "pacote") {
		t.Fatalf("esperava erro de pacote indisponível, veio %v", res.Err())
	}
}

func TestRenderConcurrent(t *testing.T) {
	eng := engine(t)
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := eng.Render(context.Background(), Job{Files: src(hello)})
			if err == nil {
				err = res.Err()
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
