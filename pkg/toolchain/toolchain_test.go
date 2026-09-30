package toolchain

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"liaf"
)

// O codigo gerado importa liaf/pkg/web e liaf/pkg/runtime. Todo pacote liaf/...
// de que eles dependem tem de estar embutido, senao liafc build quebra fora do
// repositorio com "package liaf/pkg/x is not in std".
func TestRuntimeFSCoversRuntimeDeps(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "liaf/pkg/web", "liaf/pkg/runtime", "liaf/pkg/docrt").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for pkg := range strings.FieldsSeq(string(out)) {
		if !strings.HasPrefix(pkg, "liaf/") {
			continue
		}
		dir := strings.TrimPrefix(pkg, "liaf/")
		entries, err := fs.ReadDir(liaf.RuntimeFS, dir)
		if err != nil || len(entries) == 0 {
			t.Errorf("%s nao esta embutido: acrescente %s/*.go ao //go:embed de runtime_embed.go", pkg, dir)
		}
	}
}

// Module materializa o runtime uma vez; a segunda chamada reaproveita a pasta,
// e nenhum _test.go vai para o cache.
func TestModuleIsCachedAndSkipsTests(t *testing.T) {
	t.Setenv(CacheEnv, t.TempDir())
	first, err := Module()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(first, "pkg", "runtime", "marcador")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := Module()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("modulo recriado: %s e %s", first, second)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("segunda chamada reescreveu o cache: %v", err)
	}
	err = filepath.WalkDir(first, func(p string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(p, "_test.go") {
			t.Errorf("teste copiado para o cache: %s", p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}
