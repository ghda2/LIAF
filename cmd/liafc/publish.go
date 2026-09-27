package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runPublish(args []string) {
	var localFile, endpointURL, targetPath, token string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--url" && i+1 < len(args) {
			endpointURL = args[i+1]
			i++
		} else if a == "--path" && i+1 < len(args) {
			targetPath = args[i+1]
			i++
		} else if a == "--token" && i+1 < len(args) {
			token = args[i+1]
			i++
		} else if !strings.HasPrefix(a, "-") && localFile == "" {
			localFile = a
		}
	}

	if localFile == "" || endpointURL == "" {
		fmt.Fprintln(os.Stderr, "Uso: liafc publish <arquivo_local> --url <endpoint_url> [--path <rota>] [--token <token>]")
		os.Exit(1)
	}

	data, err := os.ReadFile(localFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro lendo arquivo local: %v\n", err)
		os.Exit(1)
	}

	if targetPath == "" {
		targetPath = filepath.Base(localFile)
	}

	req, err := http.NewRequest(http.MethodPost, endpointURL, strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro criando requisicao: %v\n", err)
		os.Exit(1)
	}

	if token == "" {
		token = os.Getenv("LIAF_DEPLOY_TOKEN")
	}
	req.Header.Set("X-LIAF-Path", targetPath)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro enviando publicacao: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Falha na publicacao (%d): %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	fmt.Printf("✓ Publicado com sucesso! Resposta: %s\n", string(body))
}
