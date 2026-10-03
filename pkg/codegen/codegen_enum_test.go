package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func TestCodegenEnumMatch(t *testing.T) {
	code := `
	enum Status
		Pendente
		Pago
		Cancelado
	end

	fn checar(s Status) str
		match s
			case Status.Pendente
				"Aguardando"
			case Status.Pago
				"Confirmado"
			case Status.Cancelado
				"Cancelado"
		end
	end

	fn main() void effects(io)
		let s = Status.Pago
		let msg = checar(s)
		println(msg)
	end
	`

	l := lexer.New(code)
	p := parser.New(l, "enum_test.liaf")
	mod := p.ParseModule()

	if len(p.Diagnostics) > 0 {
		t.Fatalf("erros no parser: %+v", p.Diagnostics)
	}

	g := New(mod)
	goCode := g.Generate()

	if !strings.Contains(goCode, "type Status string") {
		t.Fatalf("esperado type Status string, gerado:\n%s", goCode)
	}
	if !strings.Contains(goCode, "Status_Pendente Status = \"Pendente\"") {
		t.Fatalf("esperado constante Status_Pendente, gerado:\n%s", goCode)
	}
	if !strings.Contains(goCode, "switch") {
		t.Fatalf("esperado switch no match, gerado:\n%s", goCode)
	}
	if !strings.Contains(goCode, "case Status_Pago:") {
		t.Fatalf("esperado case Status_Pago, gerado:\n%s", goCode)
	}
}

func TestExecuteLinearEnumProgram(t *testing.T) {
	if testing.Short() {
		t.Skip("teste ignorado em modo curto")
	}

	code := `
	enum Status
		Pendente
		Pago
		Cancelado
	end

	fn checar(s Status) str
		match s
			case Status.Pendente
				"Aguardando"
			case Status.Pago
				"Confirmado"
			case Status.Cancelado
				"Cancelado"
		end
	end

	fn main() void effects(io)
		let s = Status.Pago
		let msg = checar(s)
		println(msg)
	end
	`

	l := lexer.New(code)
	p := parser.New(l, "main.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parser: %+v", p.Diagnostics)
	}

	g := New(mod)
	goCode := g.Generate()

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(goCode), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(dir, "enum_prog.exe")
	build := exec.Command("go", "build", "-o", bin, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	run := exec.Command(bin)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	if got != "Confirmado" {
		t.Fatalf("esperado output 'Confirmado', obteve: %q", got)
	}
}

func TestExecuteLinearEnumMatchExpr(t *testing.T) {
	if testing.Short() {
		t.Skip("teste ignorado em modo curto")
	}

	code := `
	enum Prioridade
		Baixa
		Media
		Alta
	end

	fn nivel(p Prioridade) int
		let n = match p
			case Prioridade.Baixa
				1
			case Prioridade.Media
				2
			case Prioridade.Alta
				3
		end
		n * 10
	end

	fn main() void effects(io)
		println(nivel(Prioridade.Media))
	end
	`

	l := lexer.New(code)
	p := parser.New(l, "main.liaf")
	mod := p.ParseModule()
	if len(p.Diagnostics) > 0 {
		t.Fatalf("parser: %+v", p.Diagnostics)
	}

	g := New(mod)
	goCode := g.Generate()

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(goCode), 0600); err != nil {
		t.Fatal(err)
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(dir, "enum_expr.exe")
	build := exec.Command("go", "build", "-o", bin, source)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	run := exec.Command(bin)
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	if got != "20" {
		t.Fatalf("esperado output '20', obteve: %q", got)
	}
}
