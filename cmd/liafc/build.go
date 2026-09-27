package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runBuild(args []string) {
	var outFile string
	var filePath string
	embedAssets := false
	embedDir := "public"

	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outFile = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "-o=") {
			outFile = strings.TrimPrefix(args[i], "-o=")
		} else if args[i] == "--embed" || args[i] == "-embed" {
			embedAssets = true
		} else if strings.HasPrefix(args[i], "--embed=") {
			embedAssets = true
			embedDir = strings.TrimPrefix(args[i], "--embed=")
		} else if !strings.HasPrefix(args[i], "-") && filePath == "" {
			filePath = args[i]
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "Uso: liafc build <arquivo.liaf> [-o saida] [--embed]")
		os.Exit(1)
	}

	_, allErrors, generatedCode := parseAndCheck(filePath)

	if len(allErrors) > 0 {
		for _, err := range allErrors {
			fmt.Fprintf(os.Stderr, "  %s\n", err.String())
		}
		os.Exit(1)
	}

	if outFile == "" {
		base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		outFile = base
	}

	if !filepath.IsAbs(outFile) {
		cwd, _ := os.Getwd()
		outFile = filepath.Join(cwd, outFile)
	}

	tmpDir, err := os.MkdirTemp("", "liaf_build_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar pasta temporária: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	if embedAssets {
		// Localiza a pasta pública
		targetDir := embedDir
		if !filepath.IsAbs(targetDir) {
			candidate := filepath.Join(filepath.Dir(filePath), targetDir)
			if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
				targetDir = candidate
			}
		}

		destPublic := filepath.Join(tmpDir, "public")
		if err := copyDirectory(targetDir, destPublic); err != nil {
			fatal(fmt.Errorf("could not embed %s: %w", targetDir, err))
		} else {
			embedImports := "\t\"embed\"\n\t\"io/fs\"\n"
			generatedCode = strings.Replace(generatedCode, "import (\n", "import (\n"+embedImports, 1)

			embedInit := "\n//go:embed public\nvar _liaf_embedded_assets embed.FS\n\nfunc init() {\n\tif sub, err := fs.Sub(_liaf_embedded_assets, \"public\"); err == nil {\n\t\tweb.SetEmbeddedFS(sub)\n\t} else {\n\t\tweb.SetEmbeddedFS(_liaf_embedded_assets)\n\t}\n}\n"
			generatedCode = strings.Replace(generatedCode, ")\n\n", ")\n"+embedInit+"\n", 1)
			fmt.Printf("📦 [LIAF Pack] Pasta '%s' embutida com sucesso no binário nativo\n", targetDir)
		}
	}

	tmpGo := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpGo, []byte(generatedCode), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao gravar código gerado: %v\n", err)
		os.Exit(1)
	}

	// Identifica a raiz do projeto liaf para o go.mod
	execDir, _ := os.Getwd()
	goModDir := findGoMod(execDir)
	if goModDir == "" {
		// Fallback para diretório de trabalho
		goModDir = execDir
	}

	cmd := exec.Command("go", "build", "-o", outFile, tmpGo)
	cmd.Dir = goModDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro compilando binário: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Binário estático gerado com sucesso: %s\n", outFile)
}
