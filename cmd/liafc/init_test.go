package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitAndNew(t *testing.T) {
	tempDir := t.TempDir()

	// Test new
	newDir := filepath.Join(tempDir, "meu_projeto")
	runNew([]string{newDir})

	appFile := filepath.Join(newDir, "app.liaf")
	if _, err := os.Stat(appFile); os.IsNotExist(err) {
		t.Fatalf("esperado que %s existisse após liafc new", appFile)
	}

	// Test init
	initDir := filepath.Join(tempDir, "projeto_existente")
	if err := os.MkdirAll(initDir, 0755); err != nil {
		t.Fatal(err)
	}
	runInit([]string{initDir})

	appFile2 := filepath.Join(initDir, "app.liaf")
	if _, err := os.Stat(appFile2); os.IsNotExist(err) {
		t.Fatalf("esperado que %s existisse após liafc init", appFile2)
	}
}
