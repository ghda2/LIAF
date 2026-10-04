package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func runTest(args []string) {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	var testFiles []string
	info, err := os.Stat(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao acessar '%s': %v\n", target, err)
		os.Exit(1)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao ler diretório '%s': %v\n", target, err)
			os.Exit(1)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".liaf") {
				if strings.HasSuffix(e.Name(), "_test.liaf") || strings.HasPrefix(e.Name(), "test_") {
					testFiles = append(testFiles, filepath.Join(target, e.Name()))
				}
			}
		}
		if len(testFiles) == 0 {
			// Se não houver _test.liaf específico, roda os .liaf da pasta
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".liaf") {
					testFiles = append(testFiles, filepath.Join(target, e.Name()))
				}
			}
		}
	} else {
		testFiles = append(testFiles, target)
	}

	if len(testFiles) == 0 {
		fmt.Println("Nenhum arquivo de teste LIAF (.liaf) encontrado.")
		return
	}

	fmt.Printf("🧪 Executando suíte de testes LIAF (%d arquivo(s))...\n\n", len(testFiles))

	passed := 0
	failed := 0

	for _, file := range testFiles {
		start := time.Now()
		_, allErrors, generatedCode := parseAndCheck(file)
		if len(allErrors) > 0 {
			fmt.Printf("❌ FAIL %s (erros de tipo/sintaxe):\n", filepath.Base(file))
			for _, err := range allErrors {
				fmt.Printf("  %s\n", err.String())
			}
			failed++
			continue
		}

		tmpDir, err := os.MkdirTemp("", "liaf_test_*")
		if err != nil {
			fmt.Printf("❌ FAIL %s (falha ao criar pasta temporária): %v\n", filepath.Base(file), err)
			failed++
			continue
		}

		binPath := filepath.Join(tmpDir, "test_bin.exe")
		if err := compileGo(generatedCode, "", binPath, file); err != nil {
			os.RemoveAll(tmpDir)
			fmt.Printf("❌ FAIL %s (compilação falhou): %v\n", filepath.Base(file), err)
			failed++
			continue
		}

		cmd := exec.Command(binPath)
		cmd.Env = append(os.Environ(), "LIAF_TEST_MODE=1")
		out, runErr := cmd.CombinedOutput()
		dur := time.Since(start)
		os.RemoveAll(tmpDir)

		if runErr != nil {
			fmt.Printf("❌ FAIL %s (%.2fs)\n", filepath.Base(file), dur.Seconds())
			if len(out) > 0 {
				fmt.Println(string(out))
			}
			failed++
		} else {
			fmt.Printf("✓ PASS %s (%.2fs)\n", filepath.Base(file), dur.Seconds())
			passed++
		}
	}

	fmt.Printf("\nResultado: %d passaram, %d falharam (Total: %d)\n", passed, failed, passed+failed)
	if failed > 0 {
		os.Exit(1)
	}
}
