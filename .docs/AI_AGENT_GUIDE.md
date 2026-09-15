# Guia de Engenharia para Agentes de IA — LIAF

Este documento define as regras operacionais para IAs (LLMs, agentes autônomos) gerarem código, executarem auto-cura e realizarem o deploy de sistemas LIAF com máxima confiabilidade.

---

## 1. Gramática Rígida para LLMs

LIAF foi desenhada para eliminar ambiguidades sintáticas comuns em linguagens como C, Rust e Python:

| Característica | Regra LIAF | Motivo para a IA |
|---|---|---|
| **Fechamento de Blocos** | `[fn x ...] /fn x]` | Impede alucinações de profundidade de chaves `}}}` |
| **Precedência de Operadores** | `(add a b)` prefixado | Elimina erros de ordem matemática sem parênteses extras |
| **Tipagem Estática** | `[let x: int 10]` | Previne inferências incorretas de tipo em tempo de execução |
| **Concorrência Segura** | `[spawn (fn)]`, `[send c v]` | Green threads e canais CSP sem data races |

---

## 2. Fluxo de Auto-Cura (Self-Healing Protocol)

Quando uma IA escreve ou altera código LIAF:

1. Executa o validador:
   ```bash
   liafc check <arquivo.liaf> --json
   ```
2. Se houver erro, a saída trará a correção recomendada no campo `suggested_patch`:
   ```json
   {
     "code": "TAG_NAME_MISMATCH",
     "file": "main.liaf",
     "line": 12,
     "col": 5,
     "node": "FuncDecl",
     "message": "Nome no fechamento '/fn processar' não corresponde a '/fn process'",
     "suggested_patch": "/fn process]"
   }
   ```
3. A IA aplica a correção diretamente no arquivo e revalida até `status == "success"`.

---

## 3. Web Engine Integrado e Deploy Rápido

Para subir sites estáticos ou SPAs com footprint menor que 4 MB:

1. **Definição no código (`web_engine.liaf`):**
   ```liaf
   [fn main () -> (void)
     (serve_site "./public" "" "7070" false)
   /fn main]
   ```
2. **Compilação Estática Cruzada (Exemplo Linux x86_64):**
   ```bash
   GOOS=linux GOARCH=amd64 liafc build web_engine.liaf -o liaf_server
   ```
3. **Componentização e Layouts em RAM:**
   - **`base.html`:** Arquivo mestre com tags `<!-- include "menu.html" -->` e `<!-- content -->`.
   - **Páginas:** Iniciam com `<!-- layout "base.html" -->`.
   - **Resolução em Boot:** O compilador/runtime junta e minifica tudo na RAM uma única vez.
   - **Clean URLs:** Acessível via `/sobre` ou `/servicos` automaticamente.

4. **Recursos Avançados de SSG em RAM:**
   - **Markdown com Frontmatter:** Artigos `.md` com frontmatter YAML (`title`, `date`, `layout`) são parseados no boot em RAM e encaixados no layout.
   - **Coleções de Dados:** Arquivos `data/*.json` são indexados automaticamente como coleções.
   - **Loops de Template:** Suporte nativo a `<!-- for item in colecao --> ... <!-- endfor -->` com interpolação `{{ item.campo }}` ou `{{ post.title }}`.
   - **Coleção Automática `posts`:** Todos os arquivos Markdown alimentam a coleção `posts` ordenados por data.

5. **Deploy Single-Binary (`--embed`):**
   - Para embutir os assets e templates diretamente dentro do binário (sem necessidade de transferir a pasta `public/`):
     ```bash
     GOOS=linux GOARCH=amd64 liafc build web_engine.liaf -o liaf_server --embed
     ```
   - O executável torna-se 100% autossuficiente e roda de forma instantânea no servidor de produção.

