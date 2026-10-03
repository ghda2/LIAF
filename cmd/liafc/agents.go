package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"liaf"
)

// runAgents grava as skills embutidas (.agents/skills) no diretório indicado.
// Arquivos iguais são ignorados; arquivos que diferem só são sobrescritos com --force,
// para não apagar ajustes locais do usuário.
func runAgents(args []string) {
	dir := "."
	force := false
	for _, a := range args {
		switch a {
		case "--force", "-f":
			force = true
		default:
			dir = a
		}
	}

	res, err := writeAgents(dir, force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao gravar .agents: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Skills LIAF gravadas em '%s'\n", filepath.Join(dir, ".agents", "skills"))
	fmt.Printf("  %d criado(s), %d atualizado(s), %d sem mudança\n", res.created, res.updated, res.same)
	if len(res.skipped) > 0 {
		fmt.Printf("  %d arquivo(s) com alterações locais mantido(s) (use --force para sobrescrever):\n", len(res.skipped))
		for _, p := range res.skipped {
			fmt.Printf("    %s\n", p)
		}
	}
}

type agentsResult struct {
	created, updated, same int
	skipped                []string
}

func writeAgents(dir string, force bool) (agentsResult, error) {
	var res agentsResult
	err := fs.WalkDir(liaf.AgentsFS, ".agents", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := fs.ReadFile(liaf.AgentsFS, p)
		if err != nil {
			return err
		}
		existing, err := os.ReadFile(target)
		switch {
		case err == nil && bytes.Equal(existing, data):
			res.same++
			return nil
		case err == nil && !force:
			res.skipped = append(res.skipped, target)
			return nil
		case err == nil:
			res.updated++
		case os.IsNotExist(err):
			res.created++
		default:
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	return res, err
}
