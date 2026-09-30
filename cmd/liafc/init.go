package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const appTemplate = `// Aplicação LIAF
module app

struct Mensagem
  texto str
  autor str
end

route GET "/" () Response
  json-response(200, "{\"status\": \"ok\", \"mensagem\": \"Olá, Mundo! Bem-vindo ao LIAF.\"}")
end

route GET "/info" () Response
  on-err err_msg
    json-response(500, err_msg)
  end
  let msg = new Mensagem("LIAF v0.6", "liafc")
  json-response(200, try json-encode(msg))
end

fn main() void effects(fs, io, net)
  let porta = "8080"
  println(fmt("Servidor LIAF rodando em http://localhost:{}", porta))
  unwrap-or(serve-hybrid("./public", porta), false)
end
`

func runInit(args []string) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	createProject(dir, false)
}

func runNew(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Uso: liafc new <nome-do-projeto>")
		os.Exit(1)
	}
	dir := args[0]
	createProject(dir, true)
}

func createProject(targetDir string, isNew bool) {
	if isNew {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao criar diretório '%s': %v\n", targetDir, err)
			os.Exit(1)
		}
	} else {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao acessar diretório '%s': %v\n", targetDir, err)
			os.Exit(1)
		}
	}

	appFile := filepath.Join(targetDir, "app.liaf")
	if _, err := os.Stat(appFile); err == nil {
		fmt.Fprintf(os.Stderr, "O arquivo '%s' já existe. Abortando para evitar sobrescrita.\n", appFile)
		os.Exit(1)
	}

	if err := os.WriteFile(appFile, []byte(appTemplate), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao criar '%s': %v\n", appFile, err)
		os.Exit(1)
	}

	publicDir := filepath.Join(targetDir, "public")
	_ = os.MkdirAll(publicDir, 0755)

	fmt.Printf("✓ Projeto LIAF inicializado com sucesso em '%s'!\n", targetDir)
	fmt.Println("\nPara começar:")
	if isNew && targetDir != "." {
		fmt.Printf("  cd %s\n", targetDir)
		fmt.Println("  liafc run app.liaf")
	} else {
		fmt.Println("  liafc run app.liaf")
	}
}

func runVersion() {
	fmt.Println("LIAF Compiler & Toolchain (liafc) v0.6.0")
}
