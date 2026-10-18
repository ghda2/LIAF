package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"liaf/pkg/typst"
)

func TestDocJobMapsFilesLikeTheRuntime(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	doc := write("doc.typ", `#let d = json("/dados.json")`+"\n#d.nome #read(\"/img/nota.txt\")")
	dados := write("d.json", `{"nome": "Ana"}`)
	nota := write("nota.txt", "ok")

	opts, err := parseDocArgs([]string{doc, "--dados", dados, "--arquivo", "img/nota.txt=" + nota, "--json"})
	if err != nil {
		t.Fatal(err)
	}
	job, err := opts.job()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/main.typ", "/dados.json", "/img/nota.txt"} {
		if _, ok := job.Files[want]; !ok {
			t.Errorf("arquivo virtual ausente: %s", want)
		}
	}

	eng, err := typst.Default()
	if err != nil {
		t.Fatal(err)
	}
	if code := docCheck(context.Background(), eng, job, opts); code != 0 {
		t.Fatalf("documento válido saiu com código %d", code)
	}
	job.Files["/main.typ"] = []byte("#let x = (")
	if code := docCheck(context.Background(), eng, job, opts); code != 1 {
		t.Fatalf("documento inválido saiu com código %d", code)
	}
}

func TestDocArgsErrors(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"a.typ", "--arquivo", "sem-igual"},
		{"a.typ", "--pagina", "zero"},
		{"a.typ", "--desconhecida"},
		{"a.typ", "b.typ"},
	} {
		if _, err := parseDocArgs(args); err == nil {
			t.Errorf("aceitou %v", args)
		}
	}
}

func TestDocWatchDetectsChanges(t *testing.T) {
	f := filepath.Join(t.TempDir(), "a.typ")
	if err := os.WriteFile(f, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := mtimes([]string{f})
	if !changed(nil, first) {
		t.Fatal("a primeira passada precisa conferir")
	}
	if changed(first, mtimes([]string{f})) {
		t.Fatal("sem mudança, não deveria reconferir")
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(f, later, later); err != nil {
		t.Fatal(err)
	}
	if !changed(first, mtimes([]string{f})) {
		t.Fatal("mudança no arquivo não detectada")
	}
}
