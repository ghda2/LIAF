# Issue #012: Ergonomia Sintática e Eficiência de Tokens para IA

**Status:** Parcialmente implementada em 16/09/2026 — ver "Estado da implementação" no fim  
**Componente:** `pkg/parser`, `pkg/checker`, `pkg/codegen`, `specs/`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Motivação
A LIAF foi desenhada com S-expressions para eliminar ambiguidades gramaticais para agentes de IA. No entanto, em códigos do mundo real (como `task_api.liaf`), essa rigidez cobra um preço alto:
1. **Gasto Excessivo de Tokens:** Construções como `(call field state tasks)` ocupam o quádruplo de tokens que `state.tasks`.
2. **Pirâmide de Indentação com `match`:** Tratar `Result` aninhando blocos `(match expr (ok val ...) (err msg ...))` gera 5+ níveis de indentação, gastando tokens de contexto e aumentando a chance de erro de fechamento de parênteses por LLMs.
3. **Equilíbrio AI-First:** A IA precisa de código determinístico, mas também compacto e com early-return intuitivo.

---

## 2. Proposta Técnica (O "Meio-Termo")

### 2.1. Operador de Propagação de Erro / Early Return (`?` ou `try`)
Permitir desembalar um `(result T E)` diretamente dentro de funções que retornem `(result R E)` ou permitam retorno antecipado:
```lisp
;; Atual (pirâmide verbosa):
(match (call fs-read-file "data.json")
  (ok text
    (match (call json-decode text State)
      (ok state (return (call ok state)))
      (err e (return (call err e)))))
  (err e (return (call err e))))

;; Proposto (canônico e compacto):
(let text str (try (call fs-read-file "data.json")))
(let state State (try (call json-decode text State)))
(return (call ok state))
```

### 2.2. Acesso Canônico a Campos
Introduzir açúcar canônico sem ambiguidade:
- `(. state tasks)` ou `state.tasks` expandido diretamente pelo parser para o acesso de campo, eliminando o prolixo `(call field state tasks)`.

### 2.3. Formas Canônicas Estritas no Checker
- Proibir `(eq x false)` em favor de `(not x)`.
- Garantir que só exista **uma única forma válida** de escrever cada construção após a desaçucaração.

---

## 3. Tarefas de Implementação
- [ ] Implementar forma especial `(try expr)` ou operador `?` no parser e type-checker para tipos `(result T E)`.
- [ ] Implementar suporte sintático compacto a campos `(. obj field)` ou `obj.field`.
- [ ] Atualizar o formatador (`liafc fmt`) para normalizar código para a forma canônica.
- [ ] Atualizar `task_api.liaf` e exemplos com a nova sintaxe e medir redução percentual de tokens de contexto.

---

Contrato atual e especificações: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidências consolidadas: [STATUS.md](../STATUS.md).

---

## Estado da implementação (16/09/2026)

**Feito e coberto por teste:**

- `(try expr)` e bloco `(on-err var ...)` em funções e rotas. Parser, checker, codegen Go
  e formatador. `E_UNHANDLED_RESULT` quando o erro não tem para onde ir.
- Chamadas diretas sem `call`. `(call ...)` continua aceito, então nenhum programa v0.2 quebrou.
- Efeito colateral positivo: `examples/task_api_v03.liaf` deixou de ter pirâmide de `match`.
  A antiga `parse-id`, que fatiava o path com `(str-slice path 7 ...)`, sumiu junto com a
  chegada de rotas declarativas.

Testes: `pkg/parser/parser_v03_test.go`, `pkg/checker/checker_v03_test.go`,
`pkg/codegen/codegen_v03_test.go`.

**Não feito:**

- Acesso canônico a campos (`state.tasks` ou `(. state tasks)`). Continua `(field state tasks)`.
- Proibição de formas redundantes no checker, por exemplo exigir `(not x)` em vez de `(eq x false)`.

Esses dois itens seguem abertos e são o que resta da issue.
