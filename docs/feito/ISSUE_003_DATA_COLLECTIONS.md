# Issue #003: Coleções de Dados Nativas (Listas e Mapas)

**Status:** Concluida e validada localmente
**Componente:** `pkg/token`, `pkg/lexer`, `pkg/parser`, `pkg/ast`, `pkg/codegen`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
Atualmente a LIAF suporta apenas tipos primitivos (`int`, `float`, `str`, `bool`), `chan` e `struct`. Para manipular conjuntos de dados, respostas de APIs e coleções em memória, precisamos de tipos nativos para listas indexadas e dicionários/mapas chave-valor.

---

## 2. Especificações de Sintaxe

### 2.1. Listas Tipadas (`list[T]`)
Criação, acesso e manipulação de arrays/slices dinâmicos:
```liaf
(let items (list int) (call make-list int))
(do (call list-push items 42))
(let first int (call list-get items 0))
(let size int (call list-len items))
```

### 2.2. Mapas Chave-Valor (`map[K, V]`)
Dicionários associativos indexados por chave:
```liaf
(let scores (map str int) (call make-map str int))
(do (call map-set scores "alice" 100))
(let val int (call map-get scores "alice"))
(let has bool (call map-has scores "bob"))
```

### 2.3. Iteração em Coleções
```liaf
(for-each item items
  (do (call println item)))
```

---

## 3. Tarefas de Implementação
- [x] Suporte a tipos parametrizados `(list T)` e `(map K V)` no parser de tipos.
- [x] Built-ins de runtime: `make-list`, `list-push`, `list-get`, `list-len`, `make-map`, `map-set`, `map-get`, `map-has`.
- [x] Geração de código Go (`[]T` e `map[K]V`) em `pkg/codegen`.
- [x] Testes de validação em `examples/collections.liaf`.


Contrato atual e ajustes de sintaxe: [docs/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
