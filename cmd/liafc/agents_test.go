package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAgents(t *testing.T) {
	dir := t.TempDir()
	skill := filepath.Join(dir, ".agents", "skills", "liaf", "SKILL.md")

	res, err := writeAgents(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.created == 0 {
		t.Fatal("esperado que as skills fossem criadas")
	}
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("esperado %s: %v", skill, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents", "skills", "liaf-pdf", "SKILL.md")); err != nil {
		t.Fatalf("esperada a skill liaf-pdf: %v", err)
	}

	// Rodar de novo não muda nada.
	res, err = writeAgents(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.created != 0 || res.updated != 0 || len(res.skipped) != 0 {
		t.Fatalf("segunda execução deveria ser idempotente: %+v", res)
	}

	// Alteração local é preservada sem --force e sobrescrita com --force.
	if err := os.WriteFile(skill, []byte("editado"), 0644); err != nil {
		t.Fatal(err)
	}
	res, _ = writeAgents(dir, false)
	if len(res.skipped) != 1 {
		t.Fatalf("esperado 1 arquivo mantido, veio %+v", res)
	}
	if data, _ := os.ReadFile(skill); string(data) != "editado" {
		t.Fatal("alteração local não deveria ser sobrescrita sem --force")
	}
	res, _ = writeAgents(dir, true)
	if res.updated != 1 {
		t.Fatalf("esperado 1 arquivo atualizado com --force, veio %+v", res)
	}
	if data, _ := os.ReadFile(skill); string(data) == "editado" {
		t.Fatal("--force deveria sobrescrever")
	}
}
