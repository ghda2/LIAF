# Issue #048 — Motor de Conteúdo IA-First e Postagens Instantâneas (`std/content` + `liafc post`)

**Estado:** Concluída. **Criada em:** 17/10/2026. **Concluída em:** 17/10/2026. **Tipo:** stdlib/tooling/dx.  
**Componentes:** `std/content`, `pkg/content`, `pkg/builtins`, `pkg/runtime`, `cmd/liafc`.  

---

## 1. Contexto e Objetivo
Para substituir a dependência de geradores estáticos tradicionais (como Hugo/Astro) e pipelines de Git lentos, a LIAF conta agora com uma solução **IA-First de publicação instantânea**:
Um agente de IA operando localmente em uma IDE (ou desenvolvedor via SSH) cria um post em Markdown com frontmatter, valida a integridade com diagnósticos de auto-cura no terminal e publica imediatamente no servidor LIAF em produção com upload automático de imagens associadas.

---

## 2. Implementação Concluída

1. **Módulo Nativo `std/content`:**
   - Struct `Post` com campos tipados: `title`, `slug`, `date`, `author`, `summary`, `image`, `content_markdown`, `content_html`, `reading_time_min`, `word_count`.
   - Builtins de biblioteca padrão:
     - `content-parse(raw)`: parsing de frontmatter YAML/JSON + Markdown.
     - `content-slugify(title)`: conversão para slug URL-friendly canônico.
     - `markdown-to-html(md)`: renderização para HTML seguro e higienizado.
   - Funções de conveniência em `std/content.liaf`: `parse-post`, `slugify`, `render-markdown`.
2. **Ciclo de Auto-Cura no Terminal:**
   - Diagnósticos estruturados com código e `fix_hint` para a IA agir imediatamente:
     - `E_CONTENT_MISSING_TITLE`
     - `E_CONTENT_MISSING_SLUG`
     - `E_CONTENT_INVALID_SLUG`
     - `E_CONTENT_IMAGE_NOT_FOUND`
     - `E_CONTENT_EMBED_IMAGE_NOT_FOUND`
     - `E_POST_PARSE_ERROR`
3. **Subcomando CLI `liafc post`:**
   - `liafc post <post.md> [--check] [--url <url>] [--token <secret>] [--json]`
   - Suporte a verificação estática local (`--check`) e saída estruturada (`--json`).
   - Modo deploy com upload multipart de imagens para `std/storage` e publicação atômica via `/_liaf/publish`.
4. **Governança e Testes:**
   - Testes unitários `pkg/content/content_test.go`.
   - Testes de integração CLI `cmd/liafc/post_test.go`.
   - Teste de compilação LIAF com execução `pkg/codegen/codegen_content_test.go`.
   - 100% de conformidade com a skill de IA do compilador (`TestSkillCoverage`, `TestSkillExamples`).
