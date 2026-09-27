package runtime_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"liaf/pkg/runtime"
	"liaf/pkg/web"
)

func TestStorageLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	initRes := runtime.StorageInit(tempDir)
	if !initRes.OK {
		t.Fatalf("StorageInit falhou: %s", initRes.Error)
	}

	pngData := createTestPNG(32, 32)

	// Salva imagem
	saveRes := runtime.StorageSaveImage(tempDir, "minha_foto.png", string(pngData))
	if !saveRes.OK {
		t.Fatalf("StorageSaveImage falhou: %s", saveRes.Error)
	}

	savedFile := saveRes.Value
	if filepath.Ext(savedFile) != ".webp" {
		t.Errorf("esperava extensão .webp, obteve %q", savedFile)
	}

	// Verifica existência
	if !runtime.StorageExists(tempDir, savedFile) {
		t.Fatalf("arquivo %s deveria existir no storage", savedFile)
	}

	// Lê arquivo
	getRes := runtime.StorageGet(tempDir, savedFile)
	if !getRes.OK {
		t.Fatalf("StorageGet falhou: %s", getRes.Error)
	}
	if len(getRes.Value) == 0 {
		t.Fatal("arquivo lido está vazio")
	}

	// FileResponse
	fileRespRes := runtime.FileResponse(filepath.Join(tempDir, savedFile), "image/webp")
	if !fileRespRes.OK {
		t.Fatalf("FileResponse falhou: %s", fileRespRes.Error)
	}
	if fileRespRes.Value.Status != 200 || fileRespRes.Value.ContentType != "image/webp" {
		t.Errorf("FileResponse retorno inesperado: status=%d, contentType=%s",
			fileRespRes.Value.Status, fileRespRes.Value.ContentType)
	}

	// Remove arquivo
	delRes := runtime.StorageDelete(tempDir, savedFile)
	if !delRes.OK {
		t.Fatalf("StorageDelete falhou: %s", delRes.Error)
	}

	if runtime.StorageExists(tempDir, savedFile) {
		t.Fatal("arquivo não deveria existir após StorageDelete")
	}
}

func TestRequestFileDataMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("foto", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	sampleBytes := []byte("fake-image-bytes")
	_, _ = part.Write(sampleBytes)
	writer.Close()

	header := make(http.Header)
	header.Set("Content-Type", writer.FormDataContentType())

	req := web.Request{
		Method: "POST",
		Path:   "/upload",
		Body:   body.String(),
		Header: header,
		Query:  url.Values{},
	}

	// Extrai dados
	dataRes := runtime.RequestFileData(req, "foto")
	if !dataRes.OK {
		t.Fatalf("RequestFileData falhou: %s", dataRes.Error)
	}
	if dataRes.Value != "fake-image-bytes" {
		t.Errorf("esperava 'fake-image-bytes', obteve %q", dataRes.Value)
	}

	// Extrai nome
	nameRes := runtime.RequestFileName(req, "foto")
	if !nameRes.OK {
		t.Fatalf("RequestFileName falhou: %s", nameRes.Error)
	}
	if nameRes.Value != "avatar.png" {
		t.Errorf("esperava 'avatar.png', obteve %q", nameRes.Value)
	}
}

func TestRequestFileDataRawBody(t *testing.T) {
	header := make(http.Header)
	header.Set("Content-Type", "image/png")
	header.Set("X-Filename", "documento.png")

	req := web.Request{
		Method: "POST",
		Path:   "/upload",
		Body:   "raw-image-bytes",
		Header: header,
		Query:  url.Values{},
	}

	dataRes := runtime.RequestFileData(req, "")
	if !dataRes.OK {
		t.Fatalf("RequestFileData falhou: %s", dataRes.Error)
	}
	if dataRes.Value != "raw-image-bytes" {
		t.Errorf("esperava 'raw-image-bytes', obteve %q", dataRes.Value)
	}

	nameRes := runtime.RequestFileName(req, "")
	if !nameRes.OK {
		t.Fatalf("RequestFileName falhou: %s", nameRes.Error)
	}
	if nameRes.Value != "documento.png" {
		t.Errorf("esperava 'documento.png', obteve %q", nameRes.Value)
	}
}
