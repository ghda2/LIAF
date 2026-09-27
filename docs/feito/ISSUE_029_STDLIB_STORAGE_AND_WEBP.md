# Issue #029 — Storage desacoplado e compressão WebP sem perdas

**Estado:** Concluída. **Criada em:** 26/09/2026.  
**Componentes:** `pkg/runtime`, `pkg/builtins`, `pkg/web`, `std/storage.liaf`, `examples/image_storage.liaf`.  
**Testes de aceitação:** `pkg/runtime/image_test.go`, `pkg/runtime/storage_test.go`, `TestStdEveryModuleChecks/storage` (`pkg/loader`), compilação e verificação com `liafc check` e `liafc build`.

---

## 1. Contexto e Objetivo
Implementar na linguagem LIAF suporte nativo e desacoplado a upload de arquivos e imagens estilo MinIO, realizando conversão automática para WebP e compressão estritamente sem perdas de qualidade (lossless VP8L) em Go puro (sem dependências de CGO).

---

## 2. O que foi entregue

| Frente | Detalhes da Entrega |
|---|---|
| **Processamento de Imagem** | `ImageToWebP`, `ImageToWebPQuality`, `ImageDimensions` em Go puro via `github.com/HugoSmits86/nativewebp` e decoders nativos de PNG, JPEG, GIF, BMP e WebP. |
| **Storage / MinIO-like** | `StorageInit`, `StorageSaveImage`, `StorageGet`, `StorageDelete`, `StorageExists` e `FileResponse` com sanitização estrita de caminho contra directory traversal. |
| **Upload HTTP** | Suporte a corpos de requisição até 32MB em `pkg/web/router.go` (`MaxRequestBodySize`); helpers `RequestFileData` e `RequestFileName` com suporte a `multipart/form-data` e streaming binário direto. |
| **Respostas Arbitrárias** | `raw-response` e `file-response` para servir imagens e binários com headers de cache HTTP (`Cache-Control: public, max-age=31536000, immutable`). |
| **Módulo Std** | `std/storage.liaf` embutido no compilador (`std.FS`), provendo `storage-init`, `storage-upload-image`, `storage-serve-image`, `storage-delete-image` e `storage-image-exists`. |
| **Documentação** | Documentado em `docs/AI_GUIDE.md` com exemplos e tipos. |
