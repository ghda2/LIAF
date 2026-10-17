package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"liaf/pkg/builtins"
	rt "liaf/pkg/runtime"
)

// As skills de .agents/skills sao o que um agente le antes de escrever LIAF (o
// `liafc agents` as grava em qualquer projeto). Estes testes garantem que elas
// descrevem o compilador como ele e: todo exemplo compila, os blocos `test`
// passam, e nenhuma funcao ou codigo de erro novo fica sem documentacao.

const skillsDir = "../../.agents/skills"

// TestSkillExamples: todo bloco ```liaf das skills e um programa completo que passa
// no checker; os que tem blocos `test` sao compilados e executados. Os .liaf de
// examples/ tambem passam no checker.
func TestSkillExamples(t *testing.T) {
	docs := skillFiles(t, ".md")
	if len(docs) == 0 {
		t.Fatal("nenhum .md encontrado em " + skillsDir)
	}
	for _, doc := range docs {
		data, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(skillsDir, doc)
		for _, b := range liafBlocks(string(data)) {
			b := b
			name := fmt.Sprintf("%s:%d", filepath.ToSlash(rel), b.line)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				if !strings.HasPrefix(strings.TrimSpace(b.code), "module ") {
					t.Fatalf("%s: bloco não é um programa completo (falta `module`)", name)
				}
				dir := t.TempDir()
				file := filepath.Join(dir, "exemplo.liaf")
				if err := os.WriteFile(file, []byte(b.code), 0600); err != nil {
					t.Fatal(err)
				}
				_, diags, code := parseAndCheck(file)
				for _, d := range diags {
					t.Errorf("%s: %s", name, d.String())
				}
				if len(diags) > 0 || !strings.Contains(b.code, "\ntest \"") {
					return
				}
				if testing.Short() {
					t.Skip("execução dos blocos test ignorada em modo curto")
				}
				runLiafTests(t, name, file, code, dir)
			})
		}
	}

	for _, src := range skillFiles(t, ".liaf") {
		rel, _ := filepath.Rel(skillsDir, src)
		_, diags, _ := parseAndCheck(src)
		for _, d := range diags {
			t.Errorf("%s: %s", filepath.ToSlash(rel), d.String())
		}
	}
}

// runLiafTests compila o programa em modo de teste e exige que todo bloco `test` passe.
// O diretório de trabalho é temporário: os exemplos gravam arquivos.
func runLiafTests(t *testing.T, name, file, code, dir string) {
	t.Helper()
	bin := filepath.Join(dir, "teste.exe")
	if err := compileGo(code, "", bin, file); err != nil {
		t.Fatalf("%s: compilação falhou: %v", name, err)
	}
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LIAF_TEST_MODE=1", "LIAF_TEST_JSON=1")
	out, _ := cmd.CombinedOutput()
	var rep rt.TestReport
	if err := json.Unmarshal(out, &rep); err != nil {
		t.Fatalf("%s: saída do teste não é JSON: %v\n%s", name, err, out)
	}
	for _, it := range rep.Tests {
		if it.Status != "pass" {
			t.Errorf("%s: test %q falhou: %s", name, it.Name, it.Error)
		}
	}
	if rep.Passed == 0 {
		t.Errorf("%s: nenhum bloco test rodou", name)
	}
}

// TestSkillCoverage: cada builtin aparece num exemplo de builtins.md, cada função
// pública da std num exemplo de stdlib.md, e cada código de erro em auto_healing.md.
func TestSkillCoverage(t *testing.T) {
	ref := filepath.Join(skillsDir, "liaf", "references")

	examples := blocksOf(t, filepath.Join(ref, "builtins.md"))
	for _, b := range builtins.All() {
		if !mentionsCall(examples, b.Name) {
			t.Errorf("builtin %s sem exemplo em references/builtins.md", b.Name)
		}
	}

	stdExamples := blocksOf(t, filepath.Join(ref, "stdlib.md"))
	pubFn := regexp.MustCompile(`(?m)\(pub fn ([a-z0-9-]+)|^pub fn ([a-z0-9_-]+)\(`)
	stdFiles, err := filepath.Glob("../../std/*.liaf")
	if err != nil || len(stdFiles) == 0 {
		t.Fatalf("std/*.liaf não encontrado: %v", err)
	}
	for _, f := range stdFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range pubFn.FindAllStringSubmatch(string(data), -1) {
			name := m[1] + m[2]
			if !mentionsCall(stdExamples, name) {
				t.Errorf("%s: %s sem exemplo em references/stdlib.md", filepath.Base(f), name)
			}
		}
	}

	catalog, err := os.ReadFile(filepath.Join(ref, "auto_healing.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range diagnosticCodes(t) {
		if !strings.Contains(string(catalog), "`"+code+"`") {
			t.Errorf("código %s sem entrada em references/auto_healing.md", code)
		}
	}
}

func mentionsCall(text, name string) bool {
	if name == "new" {
		return regexp.MustCompile(`\bnew [A-Z]`).MatchString(text)
	}
	// linear `nome(` ou S-expression `(nome ` (usada nos exemplos de concorrência)
	q := regexp.QuoteMeta(name)
	return regexp.MustCompile(`(^|[^a-z0-9-])` + q + `\(|\(` + q + ` `).MatchString(text)
}

func blocksOf(t *testing.T, doc string) string {
	t.Helper()
	data, err := os.ReadFile(doc)
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, b := range liafBlocks(string(data)) {
		sb.WriteString(b.code)
	}
	return sb.String()
}

// diagnosticCodes coleta os códigos E_/W_ emitidos pelo código de produção.
func diagnosticCodes(t *testing.T) []string {
	t.Helper()
	re := regexp.MustCompile(`"([EW]_[A-Z0-9_]+)"`)
	seen := map[string]bool{}
	for _, root := range []string{"../../pkg", "."} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, m := range re.FindAllStringSubmatch(string(data), -1) {
				seen[m[1]] = true
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	codes := make([]string, 0, len(seen))
	for c := range seen {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

func skillFiles(t *testing.T, ext string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ext) {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
