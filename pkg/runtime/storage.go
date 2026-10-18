package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"liaf/pkg/web"
)

// StorageInit inicializa o diretório de armazenamento criando-o caso não exista.
func StorageInit(dir string) Result[bool, string] {
	if dir == "" {
		dir = "storage"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Err[bool, string]("failed to init storage dir: " + err.Error())
	}
	return Ok[bool, string](true)
}

// StorageSaveImageOpt valida o tamanho máximo (maxBytes), redimensiona proporcionalmente
// se maxWidth ou maxHeight forem maiores que 0, comprime em WebP com o nível de esforço
// (level de 0 a 9) e salva no diretório de armazenamento seguro.
func StorageSaveImageOpt(dir, filename, data string, maxWidth, maxHeight, maxBytes, level int64) Result[string, string] {
	if maxBytes > 0 && int64(len(data)) > maxBytes {
		return Err[string, string](fmt.Sprintf("file size (%d bytes) exceeds maximum allowed limit of %d bytes", len(data), maxBytes))
	}

	if dir == "" {
		dir = "storage"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Err[string, string]("failed to create storage dir: " + err.Error())
	}

	// Processa a imagem com redimensionamento proporcional e compressão WebP
	webpRes := ImageProcess(data, maxWidth, maxHeight, level)
	if !webpRes.OK {
		return Err[string, string](webpRes.Error)
	}
	webpData := webpRes.Value

	// Sanitiza o nome do arquivo prevenindo directory traversal
	cleanName := filepath.Base(filename)
	ext := filepath.Ext(cleanName)
	base := strings.TrimSuffix(cleanName, ext)
	if base == "" || base == "." || base == "auto" {
		hash := sha256.Sum256([]byte(webpData))
		base = fmt.Sprintf("img_%d_%s", time.Now().Unix(), hex.EncodeToString(hash[:8]))
	}

	finalFilename := base + ".webp"
	destPath := filepath.Join(dir, finalFilename)

	if err := os.WriteFile(destPath, []byte(webpData), 0644); err != nil {
		return Err[string, string]("failed to write image file: " + err.Error())
	}

	return Ok[string, string](finalFilename)
}

// StorageSaveImage processa a imagem (converte para WebP sem perdas de qualidade),
// salva no diretório de armazenamento com extensão .webp e retorna o nome final do arquivo salvo.
func StorageSaveImage(dir, filename, data string) Result[string, string] {
	return StorageSaveImageOpt(dir, filename, data, 0, 0, 0, 4)
}

// StorageGet lê o conteúdo de um arquivo do diretório de armazenamento.
func StorageGet(dir, filename string) Result[string, string] {
	cleanName := filepath.Base(filename)
	target := filepath.Join(dir, cleanName)
	b, err := os.ReadFile(target)
	if err != nil {
		return Err[string, string]("file not found: " + err.Error())
	}
	return Ok[string, string](string(b))
}

// StorageDelete remove um arquivo do armazenamento.
func StorageDelete(dir, filename string) Result[bool, string] {
	cleanName := filepath.Base(filename)
	target := filepath.Join(dir, cleanName)
	err := os.Remove(target)
	return fromError(err == nil, err)
}

// StorageExists verifica se o arquivo existe no armazenamento.
func StorageExists(dir, filename string) bool {
	cleanName := filepath.Base(filename)
	target := filepath.Join(dir, cleanName)
	_, err := os.Stat(target)
	return err == nil
}

// FileResponse serve um arquivo diretamente do disco com cabeçalhos otimizados para assets estáticos.
func FileResponse(filePath, contentType string) Result[web.Response, string] {
	cleanPath := filepath.Clean(filePath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return Err[web.Response, string]("file not found: " + err.Error())
	}
	res := web.Response{
		Status:      200,
		ContentType: contentType,
		Body:        string(data),
		Headers: []web.ResponseHeader{
			{Name: "Cache-Control", Value: "public, max-age=31536000, immutable"},
			{Name: "X-Content-Type-Options", Value: "nosniff"},
		},
	}
	return Ok[web.Response, string](res)
}
