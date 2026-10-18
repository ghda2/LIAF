# Issue #008: Armazenamento Híbrido e Streaming de Mídias (RAM + Disco)

**Status:** Concluida e validada localmente
**Componente:** `pkg/web`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
Para sites com dezenas de milhares de matérias ou arquivos de mídia pesados (vídeos MP4, PDFs, áudios, imagens de alta resolução), manter tudo em memória RAM pode se tornar inviável.

O objetivo é criar uma arquitetura híbrida inteligente de duas camadas (*Tiered Storage*):
1. **Tier 1 (RAM / VFS):** Código, templates, CSS, JS, Markdown, JSON e SVGs mantidos 100% em memória para latência zero (< 1 ms).
2. **Tier 2 (Disco / Streaming):** Arquivos binários grandes servidos diretamente do disco com suporte a `HTTP 206 Partial Content` (Range Requests para vídeos), zero cópia desnecessária em RAM e cache LRU dos itens mais acessados.

---

## 2. Especificações Técnicas

### 2.1. Limiar de Tamanho Automático (Threshold)
- Arquivos abaixo do limiar (ex: `< 512 KB` ou extensões textuais) são comprimidos e carregados na RAM.
- Arquivos acima do limiar ou com extensões de mídia pesada (`.mp4`, `.webm`, `.pdf`, `.zip`, `.mov`) são registrados no VFS apenas como referências de caminho em disco.

### 2.2. Streaming Direto com HTTP 206 (Range Requests)
- Uso de `http.ServeContent` para mídias em disco, permitindo seek de vídeo, retomada de downloads e buffering otimizado.

### 2.3. Cache LRU Opcional para Mídias Quentes
- As mídias mais requisitadas podem manter blocos em cache LRU configurável (ex: limite de 50 MB de RAM para mídias).

---

## 3. Tarefas de Implementação
- [x] Adicionar suporte a `AssetTypeDisk` no `pkg/web/cache.go`.
- [x] Implementar regra de limiar (`MaxRAMAssetSize`) em `ServerOptions`.
- [x] Implementar manipulador de Range Requests (`http.ServeContent`) em `pkg/web/server.go`.
- [x] Adicionar testes de streaming e validação de consumo de RAM em `pkg/web/server_test.go`.


Contrato atual e ajustes de sintaxe: [docs/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
