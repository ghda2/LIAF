package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "version", "-v", "--version":
		runVersion()
	case "init":
		runInit(args)
	case "new":
		runNew(args)
	case "check":
		runCheck(args)
	case "build":
		runBuild(args)
	case "run":
		runRun(args)
	case "emit":
		runEmit(args)
	case "fmt":
		runFmt(args)
	case "service":
		installService(args)
	case "deploy":
		runDeploy(args)
	case "publish":
		runPublish(args)
	case "doc":
		runDoc(args)
	default:
		fmt.Fprintf(os.Stderr, "Comando desconhecido: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("LIAF Compiler & Toolchain (liafc)")
	fmt.Println("Uso:")
	fmt.Println("  liafc init [caminho]                               Inicializa um projeto LIAF no diretório")
	fmt.Println("  liafc new <nome>                                   Cria um novo projeto LIAF em uma nova pasta")
	fmt.Println("  liafc check <arquivo.liaf> [--json]                Valida sintaxe e tipos")
	fmt.Println("  liafc build <arquivo.liaf> [-o saida] [--embed]    Compila para executável binário autônomo")
	fmt.Println("  liafc run <arquivo.liaf>                           Compila e executa diretamente")
	fmt.Println("  liafc emit <arquivo.liaf>                          Emite o código fonte Go intermediário")
	fmt.Println("  liafc doc check|preview <doc.typ> [--dados d.json] Confere ou visualiza um documento (PDF)")
	fmt.Println("  liafc fmt <arquivo.liaf> [-w grava no arquivo]     Formata o código no padrão canônico")
	fmt.Println("  liafc deploy caddy-bind [flags]                    Configura proxy reverso Caddy")
	fmt.Println("  liafc publish <arquivo_local> --url <endpoint_url> Publica arquivo em servidor LIAF")
	fmt.Println("  liafc service install --name <n> --bin <b>         Registra serviço no systemd (Linux)")
	fmt.Println("  liafc version, -v, --version                       Exibe a versão da toolchain")
}
