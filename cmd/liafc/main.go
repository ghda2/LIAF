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
	default:
		fmt.Fprintf(os.Stderr, "Comando desconhecido: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("LIAF Compiler & Toolchain (liafc)")
	fmt.Println("Uso:")
	fmt.Println("  liafc check <arquivo.liaf> [--json]")
	fmt.Println("  liafc build <arquivo.liaf> [-o saida] [--embed]")
	fmt.Println("  liafc run <arquivo.liaf>")
	fmt.Println("  liafc emit <arquivo.liaf>")
	fmt.Println("  liafc fmt <arquivo.liaf> [-w grava no arquivo] [-l lista fora de formato]")
	fmt.Println("  liafc publish <arquivo_local> --url <endpoint_url> [--path <rota>] [--token <token>]")
	fmt.Println("  liafc service install --name <nome> --bin <executavel> [--workdir <dir>]")
}
