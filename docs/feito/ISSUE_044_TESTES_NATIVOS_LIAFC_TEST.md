# Issue #044 — Testes Nativos na Linguagem (`liafc test`)

**Estado:** Concluída. **Criada em:** 17/10/2026. **Concluída em:** 17/10/2026. **Tipo:** tooling/linguagem.  
**Componentes:** `pkg/token`, `pkg/ast`, `pkg/parser`, `pkg/checker`, `pkg/codegen`, `pkg/runtime`, `cmd/liafc`.  

---

## 1. Contexto e Objetivo
Para atingir maturidade v1.0 e apoiar o ciclo de auto-cura mecânica por agentes de IA, programas escritos em LIAF agora declaram testes diretamente no código-fonte, executados pelo comando `liafc test` com relatórios em texto ou JSON (`--json`).

---

## 2. Implementação Concluída

1. **Sintaxe Linear e S-Expression Declarativa de Teste:**
   ```liaf
   test "calculo de desconto"
     let res = calcular_total(100, 10)
     assert(res == 90, "total deveria ser 90")
   end
   ```
   e `(test "nome" ...)` em S-expressions.
2. **Builtins de Asserção:**
   - `assert(cond bool, msg str) void`
   - `assert-true(cond bool) void`
   - `assert-false(cond bool) void`
   - `assert-eq(a any, b any, msg str) void` com equivalência estrutural profunda e tolerância numérica de larguras inteiras/ponto flutuante.
3. **Subcomando CLI e Flag `--json`:**
   - `liafc test [caminho] [--json]`
   - Executa arquivos individuais ou pastas, compila os binários em modo de teste e emite relatório em texto amigável ou JSON unificado.
4. **Casos de Aceitação & Conformidade:**
   - Caso `conformance/basics/044_test_block.liaf` validado.
   - Teste de integração de compilação e execução `pkg/codegen/codegen_tests_test.go`.
