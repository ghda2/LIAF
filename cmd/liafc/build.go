package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"liaf/pkg/toolchain"
)

func runBuild(args []string) {
	var outFile string
	var filePath string
	embedAssets := false
	embedDir := "public"
	assetsDir := ""
	docFonts := ""

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
		} else if strings.HasPrefix(args[i], "--doc-fonts=") {
			docFonts = strings.TrimPrefix(args[i], "--doc-fonts=")
		} else if !strings.HasPrefix(args[i], "-") && filePath == "" {
			filePath = args[i]
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "Uso: liafc build <arquivo.liaf> [-o saida] [--embed] [--doc-fonts=inter,serif,math,mono]")
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

	// Como o go build: sem extensao, o Windows nao executa o binario. Vale o
	// sistema de destino, entao GOOS=linux no Windows nao ganha .exe.
	targetOS := os.Getenv("GOOS")
	if targetOS == "" {
		targetOS = runtime.GOOS
	}
	if targetOS == "windows" && filepath.Ext(outFile) == "" {
		outFile += ".exe"
	}

	if !filepath.IsAbs(outFile) {
		cwd, _ := os.Getwd()
		outFile = filepath.Join(cwd, outFile)
	}

	if embedAssets {
		// Localiza a pasta pública
		targetDir := embedDir
		if !filepath.IsAbs(targetDir) {
			candidate := filepath.Join(filepath.Dir(filePath), targetDir)
			if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
				targetDir = candidate
			}
		}

		if stat, err := os.Stat(targetDir); err != nil || !stat.IsDir() {
			fatal(fmt.Errorf("could not embed %s: not a directory", targetDir))
		}
		assetsDir = targetDir

		embedImports := "\t\"embed\"\n\t\"io/fs\"\n"
		generatedCode = strings.Replace(generatedCode, "import (\n", "import (\n"+embedImports, 1)

		embedInit := "\n//go:embed public\nvar _liaf_embedded_assets embed.FS\n\nfunc init() {\n\tif sub, err := fs.Sub(_liaf_embedded_assets, \"public\"); err == nil {\n\t\tweb.SetEmbeddedFS(sub)\n\t} else {\n\t\tweb.SetEmbeddedFS(_liaf_embedded_assets)\n\t}\n}\n"
		generatedCode = strings.Replace(generatedCode, ")\n\n", ")\n"+embedInit+"\n", 1)
		fmt.Printf("📦 [LIAF Pack] Pasta '%s' embutida com sucesso no binário nativo\n", targetDir)
	}

	fontTags, err := docFontTags(docFonts)
	if err != nil {
		fatal(err)
	}
	if err := compileGo(generatedCode, assetsDir, outFile, filePath, fontTags...); err != nil {
		fmt.Fprintf(os.Stderr, "Erro compilando binário: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Binário estático gerado com sucesso: %s\n", outFile)
}

// compileGo compila o Go gerado em outFile usando o runtime embutido no
// liafc (issue #032): funciona em qualquer pasta, sem go.mod do usuario.
// assetsDir, se nao vazio, e copiado como public/ ao lado do main.go para o
// //go:embed do --embed. srcPath e o .liaf de origem: os pacotes Typst citados
// sao resolvidos a partir dele (arquivo de trava ao lado).
func compileGo(generatedCode, assetsDir, outFile, srcPath string, tags ...string) error {
	app, err := toolchain.NewApp(generatedCode)
	if err != nil {
		return err
	}
	defer app.Remove()
	app.Tags = tags

	if assetsDir != "" {
		if err := copyDirectory(assetsDir, filepath.Join(app.Dir, "public")); err != nil {
			return fmt.Errorf("could not embed %s: %w", assetsDir, err)
		}
	}

	if err := embedTypstPackages(app, generatedCode, srcPath); err != nil {
		return fmt.Errorf("pacotes Typst: %w", err)
	}

	cmd := app.Build(outFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
