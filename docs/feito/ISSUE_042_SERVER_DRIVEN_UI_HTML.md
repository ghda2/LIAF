# Issue #042: Server-Driven UI e Módulo `std/html`

**Status:** Concluída  
**Componente:** `std/html.liaf`, `pkg/builtins/table_web.go`, `pkg/web/router.go`, `.agents/skills/`  
**Data:** 04 de outubro de 2026  

---

## 1. Contexto e Motivação

Aplicações web modernas sofrem com o peso excessivo de frameworks JavaScript no cliente (React/Angular/Vue), que aumentam o consumo de tokens na geração por IA, duplicam regras de negócio e exigem gigabytes de dependências `node_modules`.

A arquitetura **Server-Driven UI (HTML Over The Wire / HTMX)** permite que o backend LIAF renderize e devolva fragmentos HTML diretamente para o navegador, reduzindo em até 95% o peso do frontend, zerando o JavaScript complexo no cliente e permitindo que agentes de IA construam interfaces completas em metade dos tokens.

---

## 2. Entregas

1. **Módulo `std/html.liaf`:**
   - `html-render(status, conteudo)`: Criação de resposta HTTP com `Content-Type: text/html; charset=utf-8`.
   - `html-safe(texto)`: Sanitização e escape automático contra injeção de scripts (XSS).
   - `html-elem(tag, attrs, conteudo)`: Construtor declarativo de tags HTML.
   - `html-fragment(id, conteudo)`: Construtor de blocos e cartões com ID para troca no DOM (`swap`).

2. **Builtins no Compilador:**
   - `html-response` e `html-escape` integrados em `pkg/builtins/table_web.go` e `pkg/web/router.go`.

3. **Validação & Exemplos:**
   - Teste de integração ponta a ponta `TestCodegenServerDrivenUI` validando Content-Type e proteção XSS.
   - Exemplo funcional em [examples/server_driven_ui.liaf](../../.agents/skills/liaf/examples/server_driven_ui.liaf).
   - Documentação completa em [references/stdlib.md](../../.agents/skills/liaf/references/stdlib.md).
