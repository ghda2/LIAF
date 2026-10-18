package main

import (
	"fmt"
	"os"

	"liaf/pkg/lsp"
)

func runLSP(args []string) {
	server := lsp.NewServer(os.Stdin, os.Stdout)
	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "LIAF LSP encerrou com erro: %v\n", err)
		os.Exit(1)
	}
}
