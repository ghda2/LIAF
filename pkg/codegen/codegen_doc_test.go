package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"liaf/pkg/checker"
	"liaf/pkg/loader"
)

func generateSource(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.liaf")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	mod, diags := loader.Load(path)
	if len(diags) > 0 {
		t.Fatalf("load: %+v", diags)
	}
	if _, diags := checker.Check(mod, "doc_test"); len(diags) > 0 {
		t.Fatalf("check: %+v", diags)
	}
	return New(mod).Generate()
}

// O motor de documentos (#034) só entra no binário de quem usa pdf-*.
func TestDocImportOnlyWhenUsed(t *testing.T) {
	const docImport = `liafdoc "liaf/pkg/docrt"`

	sem := generateSource(t, `module sem_pdf
fn main() void effects(io)
  println("oi")
end
`)
	if strings.Contains(sem, docImport) {
		t.Error("programa sem pdf-* importou o motor de documentos")
	}

	com := generateSource(t, `module com_pdf
fn main() void effects(fs, io)
  on-err e
    exit(1)
  end
  let pdf = try pdf-template("cv-moderno", "{\"nome\": \"Ana\"}")
  try fs-write-file("cv.pdf", pdf)
end
`)
	if !strings.Contains(com, docImport) {
		t.Error("programa com pdf-template não importou o motor")
	}
	if !strings.Contains(com, "liafdoc.Template(") {
		t.Error("pdf-template não virou liafdoc.Template")
	}
}
