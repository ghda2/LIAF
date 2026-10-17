package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"liaf/pkg/content"
)

type PostJSONResponse struct {
	Valid   bool                 `json:"valid"`
	Post    *content.Post        `json:"post,omitempty"`
	Images  []string             `json:"images,omitempty"`
	Errors  []content.Diagnostic `json:"errors,omitempty"`
	Message string               `json:"message,omitempty"`
	URL     string               `json:"url,omitempty"`
}

func runPost(args []string) {
	checkOnly := false
	asJSON := false
	url := ""
	token := os.Getenv("LIAF_DEPLOY_TOKEN")
	var targetFile string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--check":
			checkOnly = true
		case "--json":
			asJSON = true
		case "--url":
			if i+1 < len(args) {
				url = args[i+1]
				i++
			}
		case "--token":
			if i+1 < len(args) {
				token = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(arg, "-") && targetFile == "" {
				targetFile = arg
			}
		}
	}

	if targetFile == "" {
		if asJSON {
			fmt.Println(`{"valid":false,"message":"nenhum arquivo de post (.md) especificado"}`)
		} else {
			fmt.Fprintln(os.Stderr, "Uso: liafc post <arquivo.md> [--check] [--url <url_servidor>] [--token <secret>] [--json]")
		}
		os.Exit(1)
	}

	rawBytes, err := os.ReadFile(targetFile)
	if err != nil {
		if asJSON {
			fmt.Printf(`{"valid":false,"message":"erro ao ler arquivo: %s"}`+"\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "Erro ao ler '%s': %v\n", targetFile, err)
		}
		os.Exit(1)
	}

	post, parseErr := content.Parse(string(rawBytes))
	if parseErr != nil {
		if asJSON {
			resp := PostJSONResponse{
				Valid: false,
				Errors: []content.Diagnostic{
					{
						Code:    "E_POST_PARSE_ERROR",
						Message: parseErr.Error(),
						FixHint: "verifique se o cabeçalho '---' está formatado corretamente",
					},
				},
			}
			enc, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(enc))
		} else {
			fmt.Printf("❌ Erro de sintaxe no post '%s':\n  %v\n", targetFile, parseErr)
		}
		os.Exit(1)
	}

	baseDir := filepath.Dir(targetFile)
	diags := content.Validate(post, baseDir)
	localImages := content.ExtractLocalImages(post)

	if len(diags) > 0 {
		if asJSON {
			resp := PostJSONResponse{
				Valid:  false,
				Post:   post,
				Images: localImages,
				Errors: diags,
			}
			enc, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(enc))
		} else {
			fmt.Printf("❌ Falha de validação em '%s' (%d problema(s)):\n\n", filepath.Base(targetFile), len(diags))
			for _, d := range diags {
				fmt.Printf("  • [%s] %s\n", d.Code, d.Message)
				fmt.Printf("    💡 Dica de auto-cura: %s\n\n", d.FixHint)
			}
		}
		os.Exit(1)
	}

	// Se for apenas verificação local
	if checkOnly || url == "" {
		if asJSON {
			resp := PostJSONResponse{
				Valid:  true,
				Post:   post,
				Images: localImages,
			}
			enc, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(enc))
		} else {
			fmt.Printf("✓ Post válido: %q (slug: '%s')\n", post.Title, post.Slug)
			if post.Date != "" {
				fmt.Printf("  Data: %s\n", post.Date)
			}
			if len(post.Tags) > 0 {
				fmt.Printf("  Tags: %s\n", strings.Join(post.Tags, ", "))
			}
			fmt.Printf("  Palavras: %d (~%d min de leitura)\n", post.WordCount, post.ReadingTimeMin)
			if len(localImages) > 0 {
				fmt.Printf("  Imagens locais validadas: %s\n", strings.Join(localImages, ", "))
			}
			if url == "" {
				fmt.Println("\nPara publicar, forneça a flag --url <servidor>")
			}
		}
		return
	}

	// Modo publicação instantânea para o servidor
	client := &http.Client{Timeout: 30 * time.Second}

	// 1. Upload de imagens associadas (se houver)
	for _, imgName := range localImages {
		imgPath := filepath.Join(baseDir, imgName)
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			continue
		}

		uploadURL := strings.TrimRight(url, "/") + "/_liaf/storage/upload"
		var bodyBuf bytes.Buffer
		mw := multipart.NewWriter(&bodyBuf)
		part, err := mw.CreateFormFile("file", filepath.Base(imgName))
		if err == nil {
			_, _ = part.Write(imgData)
			_ = mw.Close()

			req, _ := http.NewRequest("POST", uploadURL, &bodyBuf)
			req.Header.Set("Content-Type", mw.FormDataContentType())
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
	}

	// 2. Publicação do conteúdo do Post via Publish API
	publishEndpoint := strings.TrimRight(url, "/") + "/_liaf/publish"
	destPath := fmt.Sprintf("posts/%s.md", post.Slug)

	pubReq, _ := http.NewRequest("POST", publishEndpoint, bytes.NewReader(rawBytes))
	pubReq.Header.Set("Content-Type", "text/markdown")
	pubReq.Header.Set("X-LIAF-Path", destPath)
	if token != "" {
		pubReq.Header.Set("Authorization", "Bearer "+token)
	}

	pubResp, err := client.Do(pubReq)
	if err != nil {
		if asJSON {
			fmt.Printf(`{"valid":false,"message":"falha na conexão com o servidor: %s"}`+"\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "Erro ao publicar no servidor '%s': %v\n", url, err)
		}
		os.Exit(1)
	}
	defer pubResp.Body.Close()

	if pubResp.StatusCode != 200 {
		respBody, _ := io.ReadAll(pubResp.Body)
		if asJSON {
			fmt.Printf(`{"valid":false,"status":%d,"message":%q}`+"\n", pubResp.StatusCode, string(respBody))
		} else {
			fmt.Fprintf(os.Stderr, "Servidor retornou erro HTTP %d: %s\n", pubResp.StatusCode, string(respBody))
		}
		os.Exit(1)
	}

	finalURL := fmt.Sprintf("%s/blog/%s", strings.TrimRight(url, "/"), post.Slug)
	if asJSON {
		resp := PostJSONResponse{
			Valid:   true,
			Post:    post,
			Images:  localImages,
			URL:     finalURL,
			Message: "post publicado com sucesso",
		}
		enc, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(enc))
	} else {
		fmt.Printf("🚀 Post publicado com sucesso em milissegundos!\n")
		fmt.Printf("   URL: %s\n", finalURL)
		fmt.Printf("   Destino: %s\n", destPath)
	}
}
