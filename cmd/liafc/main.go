package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"liaf/pkg/checker"
	"liaf/pkg/codegen"
	"liaf/pkg/diagnostic"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "check":
		runCheck(args)
	case "build":
		runBuild(args)
	case "run":
		runRun(args)
	case "emit":
		runEmit(args)
	default:
		fmt.Fprintf(os.Stderr, "Comando desconhecido: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("LIAF Compiler (liafc)")
	fmt.Println("Uso:")
	fmt.Println("  liafc check <arquivo.liaf> [--json]")
	fmt.Println("  liafc build <arquivo.liaf> [-o saida]")
	fmt.Println("  liafc run <arquivo.liaf>")
	fmt.Println("  liafc emit <arquivo.liaf>")
}

func parseAndCheck(filePath string) (*parser.Parser, *checker.Checker, []diagnostic.Diagnostic, string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao ler arquivo %s: %v\n", filePath, err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l, filePath)
	prog := p.ParseProgram()

	allErrors := p.Errors()

	c := checker.New(filePath)
	checkErrors := c.Check(prog)
	allErrors = append(allErrors, checkErrors...)

	gen := codegen.New(prog)
	generatedCode := gen.Generate()

	return p, c, allErrors, generatedCode
}

func runCheck(args []string) {
	jsonOutput := false
	var filePath string

	for _, a := range args {
		if a == "--json" || a == "-json" {
			jsonOutput = true
		} else if !strings.HasPrefix(a, "-") && filePath == "" {
			filePath = a
		}
	}

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "Uso: liafc check <arquivo.liaf> [--json]")
		os.Exit(1)
	}

	_, _, allErrors, _ := parseAndCheck(filePath)

	report := diagnostic.NewReport(allErrors)

	if jsonOutput {
		jsonStr, _ := report.ToJSON()
		fmt.Println(jsonStr)
	} else {
		if len(allErrors) == 0 {
			fmt.Println("✓ Nenhum erro encontrado. Sintaxe e tipos válidos!")
		} else {
			fmt.Printf("✗ %d erro(s) encontrado(s):\n", len(allErrors))
			for _, err := range allErrors {
				fmt.Printf("  %s\n", err.String())
			}
		}
	}

	if len(allErrors) > 0 {
		os.Exit(1)
	}
}

func runEmit(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Uso: liafc emit <arquivo.liaf>")
		os.Exit(1)
	}

	filePath := args[0]
	_, _, allErrors, generatedCode := parseAndCheck(filePath)

	if len(allErrors) > 0 {
		for _, err := range allErrors {
			fmt.Fprintf(os.Stderr, "  %s\n", err.String())
		}
		os.Exit(1)
	}

	fmt.Println(generatedCode)
}

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

	_, _, allErrors, generatedCode := parseAndCheck(filePath)

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
			fmt.Fprintf(os.Stderr, "Aviso: não foi possível embutir pasta '%s': %v\n", targetDir, err)
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

func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func runRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Uso: liafc run <arquivo.liaf>")
		os.Exit(1)
	}

	filePath := args[0]
	_, _, allErrors, generatedCode := parseAndCheck(filePath)

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

func findGoMod(startDir string) string {
	curr := startDir
	for {
		if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
			return curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return ""
}
