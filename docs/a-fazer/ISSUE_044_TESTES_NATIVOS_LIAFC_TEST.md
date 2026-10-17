# Issue #044 — Testes Nativos na Linguagem (`liafc test`)

**Estado:** Aberta. **Criada em:** 17/10/2026. **Tipo:** tooling/linguagem.  
**Componentes:** `pkg/parser`, `pkg/checker`, `pkg/codegen`, `cmd/liafc`.  

---

## 1. Contexto e Objetivo
Para atingir maturidade v1.0 e apoiar o ciclo de auto-cura mecânica por agentes de IA, programas escritos em LIAF precisam declarar testes diretamente no código-fonte, executados pelo comando `liafc test` com relatórios em texto ou JSON (`--json`).

---

## 2. Proposta Técnica

1. **Sintaxe Linear Declarativa de Teste:**
   ```liaf
   test "calculo de desconto"
     let res = calcular_total(100, 10)
     assert(res == 90, "total deveria ser 90")
   end
   ```
2. **Builtin de Asserção:**
   - `assert(cond bool, msg str) void`
   - `assert-eq(a any, b any, msg str) void`
3. **Subcomando CLI:**
   - `liafc test [caminho] [--json]`
   - Executa os blocos de teste em modo isolado, coletando falhas e tempo de execução.
4. **Casos de Aceitação:**
   - Teste automatizado ponta a ponta em `cmd/liafc/test_test.go` e caso `conformance/basics/044_test_block.liaf`.
