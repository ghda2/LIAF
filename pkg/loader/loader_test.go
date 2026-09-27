package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"liaf/pkg/checker"
	"liaf/pkg/loader"
)

func TestLoaderBasicImport(t *testing.T) {
	dir := t.TempDir()

	mathFile := filepath.Join(dir, "math.liaf")
	mathContent := `(module math
  (fn dobrar (params (n int)) (returns int) (effects)
    (body (return (add n n)))))`
	if err := os.WriteFile(mathFile, []byte(mathContent), 0600); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(dir, "main.liaf")
	mainContent := `(module main
  (import "./math.liaf")
  (fn main (params) (returns void) (effects io)
    (body
      (println (dobrar 21)))))`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0600); err != nil {
		t.Fatal(err)
	}

	mod, diags := loader.Load(mainFile)
	if len(diags) > 0 {
		t.Fatalf("unexpected loader diags: %v", diags)
	}

	if mod == nil {
		t.Fatal("expected non-nil module")
	}

	// Verify checker passes on the resolved module
	_, checkDiags := checker.Check(mod, mainFile)
	if len(checkDiags) > 0 {
		t.Fatalf("unexpected checker diags: %v", checkDiags)
	}
}

func TestLoaderCircularImport(t *testing.T) {
	dir := t.TempDir()

	fileA := filepath.Join(dir, "a.liaf")
	fileB := filepath.Join(dir, "b.liaf")

	contentA := `(module a
  (import "./b.liaf")
  (fn fa (params) (returns void) (effects) (body (return))))`
	contentB := `(module b
  (import "./a.liaf")
  (fn fb (params) (returns void) (effects) (body (return))))`

	_ = os.WriteFile(fileA, []byte(contentA), 0600)
	_ = os.WriteFile(fileB, []byte(contentB), 0600)

	_, diags := loader.Load(fileA)
	found := false
	for _, d := range diags {
		if d.Code == "E_CIRCULAR_IMPORT" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected E_CIRCULAR_IMPORT, got: %v", diags)
	}
}

func TestLoaderImportNotFound(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.liaf")
	mainContent := `(module main
  (import "./missing.liaf")
  (fn main (params) (returns void) (effects) (body (return))))`
	_ = os.WriteFile(mainFile, []byte(mainContent), 0600)

	_, diags := loader.Load(mainFile)
	found := false
	for _, d := range diags {
		if d.Code == "E_IMPORT_NOT_FOUND" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected E_IMPORT_NOT_FOUND, got: %v", diags)
	}
}

func TestLoaderDirectory(t *testing.T) {
	dir := t.TempDir()

	typesFile := filepath.Join(dir, "types.liaf")
	typesContent := `(module types
  (struct User (fields (name str) (age int))))`
	_ = os.WriteFile(typesFile, []byte(typesContent), 0600)

	mainFile := filepath.Join(dir, "main.liaf")
	mainContent := `(module main
  (fn main (params) (returns void) (effects io)
    (body
      (let u User (new User "Alice" 30))
      (println (field u name)))))`
	_ = os.WriteFile(mainFile, []byte(mainContent), 0600)

	mod, diags := loader.Load(dir)
	if len(diags) > 0 {
		t.Fatalf("unexpected loader diags for directory: %v", diags)
	}

	_, checkDiags := checker.Check(mod, dir)
	if len(checkDiags) > 0 {
		t.Fatalf("unexpected checker diags: %v", checkDiags)
	}
}
