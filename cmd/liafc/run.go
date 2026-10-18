package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func runRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Uso: liafc run <arquivo.liaf> [argumentos do programa...]")
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

	// Compila e executa em vez de "go run": o programa roda na pasta atual do
	// usuario (caminhos relativos de fs-* resolvem ali, nao no cache do
	// runtime) e o codigo de saida dele chega intacto a quem chamou.
	bin := filepath.Join(tmpDir, "programa")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if err := compileGo(generatedCode, "", bin, filePath); err != nil {
		os.RemoveAll(tmpDir)
		fmt.Fprintf(os.Stderr, "Erro compilando programa: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command(bin, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	os.RemoveAll(tmpDir)

	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		os.Exit(exitErr.ExitCode())
	case err != nil:
		fmt.Fprintf(os.Stderr, "Erro executando programa: %v\n", err)
		os.Exit(1)
	}
}
