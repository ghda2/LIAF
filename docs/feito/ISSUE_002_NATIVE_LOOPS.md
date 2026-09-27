# Issue #002: Estruturas Nativas de Repetição e Loops

**Status:** Concluida e validada localmente
**Componente:** `pkg/token`, `pkg/lexer`, `pkg/parser`, `pkg/ast`, `pkg/codegen`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
Atualmente, repetições em LIAF dependem exclusivamente de recursão ou despacho de concorrência com canais. Para algoritmos iterativos, processamento sequencial e performance de IA, a linguagem necessita de primitivas de repetição explícitas e determinísticas.

---

## 2. Especificações de Sintaxe

### 2.1. While / Loop Condicional
Repetição baseada em predicado booleano:
```liaf
(while (lt i 10)
  (do (call println i))
  (set i (add i 1)))
```

### 2.2. For com Contador / Intervalo (Range)
Iteração com limites definidos:
```liaf
(for-range i 0 10
  (do (call println i)))
```

### 2.3. Break e Continue
Controle explícito de saída de laços:
```liaf
(break)
(continue)
```

---

## 3. Tarefas de Implementação
- [x] Adicionar tokens `while`, `for-range`, `break`, `continue` em `pkg/token`.
- [x] Criar nós de AST correspondentes em `pkg/ast/ast.go`.
- [x] Implementar regras de parsing em `pkg/parser/parser.go`.
- [x] Implementar geração de código Go (`for condition { ... }`) em `pkg/codegen/codegen.go`.
- [x] Criar testes unitários e exemplo em `examples/loops.liaf`.


Contrato atual e ajustes de sintaxe: [docs/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
