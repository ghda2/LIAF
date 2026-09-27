package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Uso: liafc run <arquivo.liaf>")
		os.Exit(1)
	}

	filePath := args[0]
	_, allErrors, generatedCode := parseAndCheck(filePath)

	if len(allErrors) > 0 {
		for _, err := range allErrors {
			fmt.Fprintf(os.Stderr, "  %s\n", err.String())
		}
		os.Exit(1)
	}

	tmpDir, err := os.MkdirTemp("", "liaf_run_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar pasta temporária: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	tmpGo := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpGo, []byte(generatedCode), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao gravar código gerado: %v\n", err)
		os.Exit(1)
	}

	execDir, _ := os.Getwd()
	goModDir := findGoMod(execDir)
	if goModDir == "" {
		goModDir = execDir
	}

	cmd := exec.Command("go", "run", tmpGo)
	cmd.Dir = goModDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro executando programa: %v\n", err)
		os.Exit(1)
	}
}
