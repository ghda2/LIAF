# Issue #025: `if` como Expressão

**Status:** Concluída
**Componente:** `pkg/parser`, `pkg/ast`, `pkg/checker`, `pkg/codegen`, `cmd/liafc/fmt.go`
**Data:** 24 de setembro de 2026 (concluída em 26 de setembro de 2026)
**Casos de aceitação:** `conformance/basics/025_*.liaf`

---

## 1. Contexto
`(let x int (if c (then 1) (else 2)))` é rejeitado com `E_INVALID_EXPR`. Hoje é preciso declarar a
variável com um valor provisório e usar `set` nos dois ramos: mais tokens e um valor inicial sem
sentido.

## 2. Proposta
`if` em posição de expressão exige os dois ramos, cada um com exatamente uma expressão, ambas do
mesmo tipo. Em posição de instrução, nada muda.

- `E_IF_EXPR_MISSING_ELSE`: `if` usado como valor sem `else`.
- `E_IF_EXPR_BRANCH_TYPE`: ramos com tipos diferentes; a mensagem mostra os dois tipos.

Codegen Go: função anônima invocada na hora, ou variável temporária antes da instrução.

## 3. Critérios de conclusão
- [x] Caso `025_if_valor` com `status: done`.
- [x] Testes de checker para os dois diagnósticos novos.
- [x] `liafc fmt` imprime a forma canônica.
