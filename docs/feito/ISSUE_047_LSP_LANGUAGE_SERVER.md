# Issue #047 — Language Server Protocol Nativo (`liafc lsp`)

**Estado:** Concluída. **Criada em:** 17/10/2026. **Concluída em:** 17/10/2026. **Tipo:** tooling/dx.  
**Componentes:** `pkg/lsp`, `cmd/liafc`.  

---

## 1. Contexto e Objetivo
Para consolidar a experiência de desenvolvimento e integração com editores modernos (VS Code, Cursor, Neovim, Zed), a LIAF conta agora com um Language Server nativo executável via `liafc lsp`. O servidor comunica-se via JSON-RPC 2.0 padrão (stdin/stdout) e fornece diagnósticos instantâneos, autocompleção, hover de tipos/documentação e formatação automática.

---

## 2. Implementação Concluída

1. **Protocolo Base (JSON-RPC 2.0 / LSP):**
   - Framing padrão HTTP-like com `Content-Length: ...\r\n\r\n`.
   - Handshake `initialize`, `initialized`, `shutdown`, `exit`.
2. **Gerenciamento de Documentos:**
   - `textDocument/didOpen`, `textDocument/didChange` (full sync) e `textDocument/didClose`.
3. **Diagnósticos em Tempo Real (`textDocument/publishDiagnostics`):**
   - Pipeline de validação instantânea integrando `lexer`, `parser` e `checker`.
   - Tradução precisa de coordenadas 1-indexed do compilador para 0-indexed do LSP.
4. **Autocompleção (`textDocument/completion`):**
   - Palavras-chave da linguagem.
   - Builtins da biblioteca padrão com tipo de retorno, parâmetros e efeitos exigidos.
   - Símbolos locais (funções, structs, enums, middlewares).
5. **Hover (`textDocument/hover`):**
   - Documentação rica em Markdown ao passar o mouse sobre builtins, palavras-chave e símbolos declarados.
6. **Formatação (`textDocument/formatting`):**
   - Formatação canônica de código para S-expressions e indentação de blocos para sintaxe linear (v0.6).
7. **Subcomando CLI:**
   - `liafc lsp` executa o servidor em stdin/stdout.
8. **Testes:**
   - Suíte `pkg/lsp/server_test.go` validando o ciclo de vida completo do LSP.

---

## 3. Como usar no VS Code / Cursor

Para usar com extensões genéricas de LSP (como `generic-lsp` ou extensão própria):

```json
{
  "languageServerExample.trace.server": "verbose",
  "liaf.serverPath": "liafc",
  "liaf.serverArgs": ["lsp"]
}
```
ou em Neovim:
```lua
vim.lsp.start({
  name = 'liaf-lsp',
  cmd = {'liafc', 'lsp'},
  root_dir = vim.fs.dirname(vim.fs.find({'app.liaf', '.git'}, { upward = true })[1]),
})
```
