# Issue #018: Diagnósticos Precisos e Sugestões de Auto-Cura para LLMs

**Status:** Concluída (critérios marcados abaixo). Conferido em 26/09/2026.  
**Componente:** `pkg/parser`, `pkg/checker`, `cmd/liafc`  
**Data:** 24 de setembro de 2026  

---

## 1. Contexto e Motivação
Em testes reais de inferência com modelos locais compactos (ex: `qwen2.5-coder:7b` via Ollama na pasta `teste_ia/`), o modelo acertou ~90% da estrutura da linguagem (structs, rotas, efeitos, json e coleções), mas travou no ciclo de auto-cura por 5 tentativas no mesmo ponto:

```lisp
(let itens (field input itens))
```

O compilador retornou:
```json
{
  "code": "E_UNEXPECTED_TOKEN",
  "line": 52,
  "col": 37,
  "message": "Token inesperado ao iniciar expressão: \")\""
}
```

Como o erro foi emitido como uma falha gramatical genérica (`E_UNEXPECTED_TOKEN`), o modelo não conseguiu deduzir que a sintaxe exigida era `(let nome tipo valor)` (3 argumentos) e repetiu o erro até estourar o limite de tentativas.

---

## 2. Proposta Técnica

### 2.1. Diagnósticos Especializados no Parser
Substituir erros sintáticos genéricos por mensagens semânticas quando a estrutura do formulário for reconhecida:
- **`E_LET_ARITY` / `E_LET_MISSING_TYPE`:** Quando `let` for fechado com apenas 2 elementos:
  - *Mensagem:* `"(let nome tipo valor) exige nome, tipo e valor. Faltou declarar o tipo da variável."`
  - *Patch sugerido:* Popular o campo `suggested_patch` do JSON.
- **Formas comuns:** Fazer o mesmo para `set`, `fn`, `struct` e `if`.

### 2.2. Inferência Opcional de Tipo no `let` (Avaliação)
Avaliar permitir a omissão de tipo quando a expressão à direita tiver tipo unívoco inferível pelo checker:
```lisp
;; Opcional compacto:
(let itens (field input itens))
;; Expandido pelo checker para:
(let itens (list ItemPedido) (field input itens))
```
Isso economizaria tokens e eliminaria um dos pontos mais frequentes de atrito para modelos de IA.

---

## 3. Critérios de Conclusão
- [x] Parser distingue construtores de tipo (`chan`, `list`, `map`, `option`, `result`) de expressões comuns.
- [x] Inferência opcional de tipo implementada no `let` quando o tipo for omitido (`(let x valor)`), propagando o tipo deduzido no checker para o codegen.
- [x] Suíte de testes `go test ./...` 100% aprovada.
- [x] Teste de desafio real com banco de dados usando `qwen2.5-coder:7b` atingiu **100% de sucesso na Tentativa 1 (12.5s)**.
- [x] Binário autônomo gerado (`solucao.exe`) e executado com sucesso.
