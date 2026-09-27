# Issue #019: Tabela Única de Builtins

**Status:** Concluída e validada
**Componente:** `pkg/builtins` (novo), `pkg/checker`, `pkg/codegen`, `pkg/codegen/c`, `pkg/runtime`
**Data:** 24 de setembro de 2026
**Pré-requisito de:** #020 a #026

---

## 1. Contexto
Recorte da [#014](../arquivo/ISSUE_014_MODULARIZACAO_CODEBASE.md). A superfície de builtins está repetida em
cinco lugares (`checker.go`, `library.go`, `codegen.go`, `c/codegen.go`, `runtime.go`), com contagens
divergentes (32/31/34/33). As issues #020–#026 acrescentam ~45 builtins; fazer isso no esquema atual
multiplicaria a duplicação e a dessincronia.

## 2. Proposta
Criar `pkg/builtins` com uma tabela declarativa. Cada entrada define:

- nome e aridade (fixa, mínima ou variádica);
- tipos de parâmetro e de retorno, incluindo polimorfismo simples (`int|float`, `(list T)`);
- efeitos exigidos;
- emissão Go (template ou função);
- emissão C, ou marcação explícita "não suportado no backend C".

O checker e os dois backends passam a consultar a tabela. Builtins com regras de tipo especiais
(`json-decode`, `new`, `field`, `make-list`, `db-*`) podem manter um gancho próprio, registrado na
mesma tabela.

## 3. Critérios de conclusão
- [x] Um builtin simples novo é adicionado editando **um** arquivo.
- [x] Auditoria: listar quais entradas faltavam em cada uma das cinco tabelas antigas.
- [x] `go test ./...` verde, sem mudar o diagnóstico nem o código gerado dos exemplos
      (comparar a saída de `liafc check --json` e do codegen de todos os `examples/` antes e depois).
- [x] `TestConformanceBasics` continua com os mesmos casos `done`.
- [x] Refatoração pura: nenhuma mudança de semântica.
