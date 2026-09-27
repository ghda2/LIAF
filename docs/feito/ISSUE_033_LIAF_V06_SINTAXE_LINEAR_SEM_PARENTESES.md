# Issue #033: LIAF v0.6 — Sintaxe Linear Baseada em Linhas e Delimitadores Leves

**Status:** Concluída  
**Componente:** `pkg/token`, `pkg/lexer`, `pkg/parser`, `pkg/codegen`, `docs/linguagem/SPEC.md`, `.agents/skills/liaf/SKILL.md`  
**Data:** 27 de setembro de 2026  

---

## 1. Contexto e Motivação

A LIAF foi criada como uma linguagem desenhada primariamente para consumo e geração por **modelos de linguagem (LLMs)**.
Medições empíricas revelaram:
1. **Consumo excessivo de tokens BPE:** Parênteses aninhados (`( ... ( ... )))))`) consumiam muitos tokens sem agregar semântica.
2. **Alucinação de contagem de escopo:** Modelos de linguagem frequentemente erravam ao fechar 4, 5 ou 6 parênteses seguidos no final de funções ou blocos aninhados (`E_UNEXPECTED_TOKEN` ou `E_TRAILING_TOKENS`).
3. **Fragilidade de indentação invisível:** Linguagens com indentação pura (Python/YAML) sofrem com erros invisíveis de tabs/espaços.

---

## 2. Implementação

1. **`pkg/token` & `pkg/lexer`:**
   - Adicionados tokens de delimitadores: `END ("end")`, `COMMA (",")`, `COLON (":")`, `ASSIGN ("=")`.
   - Adicionados operadores infixos e lógicos: `+`, `-`, `*`, `/`, `==`, `!=`, `<`, `<=`, `>`, `>=`, `&&`, `||`, `!`.
   - Suporte nativo a comentários de linha `//` além do legado `;`.
2. **`pkg/parser`:**
   - Implementado `parseLinearModule` em `pkg/parser/linear.go` com suporte a `struct ... end`, `fn ... end`, `route ... end`, `ws-route ... end`.
   - Expressões lineares com algoritmo Pratt / Precedence Climbing para operadores binários.
   - Retorno implícito e blocos `on-err`.
   - Preservação de 100% de compatibilidade retroativa com arquivos v0.2–v0.5 em S-expressions.
3. **Integração e Execução:**
   - O pipeline intermediário (`ast`), checagem estática (`pkg/checker`) e gerador de código (`pkg/codegen`) permaneceram 100% compatíveis.
   - Validação ponta a ponta com subida de servidor HTTP real em `TestExecuteV06TaskAPI`.

---

## 3. Critérios de Aceitação e Evidências

1. **Reconhecimento da nova gramática:**
   - Testes unitários do parser em `pkg/parser/linear_test.go` cobrindo tipos, funções, rotas, websockets, precedência de operadores e controle de fluxo.
2. **Compatibilidade com Checker e Codegen:**
   - `TestGenerateV06LinearRoutes` e `TestExecuteV06TaskAPI` aprovados com `go test ./pkg/codegen/...`.
3. **Redução de tokens BPE comprovada:**
   - Medição com `tiktoken` (codificadores `cl100k_base` e `o200k_base`) entre `task_api_v03.liaf` e `task_api_v06.liaf`:
     - **v0.3 (S-expr):** 1.860 tokens (7.035 caracteres)
     - **v0.6 (Linear):** 571 tokens (2.072 caracteres)
     - **Redução real obtida:** **69,3% de economia de tokens** (superando a meta de 40%).
4. **Suíte completa:**
   - `go test ./...` executou com sucesso (código 0) em todos os pacotes.
