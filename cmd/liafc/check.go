package main

import (
	"fmt"
	"os"
	"strings"

	"liaf/pkg/checker"
	"liaf/pkg/codegen"
	"liaf/pkg/diagnostic"
	"liaf/pkg/loader"
	"liaf/pkg/parser"
)

func parseAndCheck(filePath string) (*parser.Parser, []diagnostic.Diagnostic, string) {
	mod, loadErrors := loader.Load(filePath)
	if len(loadErrors) > 0 || mod == nil {
		return nil, loadErrors, ""
	}

	_, semanticErrors := checker.Check(mod, filePath)
	if len(semanticErrors) > 0 {
		return nil, semanticErrors, ""
	}

	gen := codegen.New(mod)
	generatedCode := gen.Generate()

	return nil, nil, generatedCode
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

	_, allErrors, _ := parseAndCheck(filePath)

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
	_, allErrors, generatedCode := parseAndCheck(filePath)

	if len(allErrors) > 0 {
		for _, err := range allErrors {
			fmt.Fprintf(os.Stderr, "  %s\n", err.String())
		}
		os.Exit(1)
	}

	fmt.Println(generatedCode)
}
