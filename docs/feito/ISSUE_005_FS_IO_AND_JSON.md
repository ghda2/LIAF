# Issue #005: I/O de Sistema de Arquivos e Serialização JSON

**Status:** Concluida e validada localmente
**Componente:** `pkg/codegen`, `pkg/ast`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
Para carregar configurações, persistir dados locais e interagir com formatos universais na web, a LIAF precisa de suporte nativo a leitura/escrita em disco e encoders/decoders de JSON integrados ao sistema de tipos e structs.

---

## 2. Especificações de Sintaxe

### 2.1. Manipulação de Arquivos (FS)
```liaf
(let content str (call fs-read-file "./config.json"))
(do (call fs-write-file "./output.txt" "dados gerados"))
(let exists bool (call fs-exists "./output.txt"))
```

### 2.2. Manipulação de JSON
Integração com tipos e structs LIAF:
```liaf
(let payload str (call json-encode user-struct))
(let user User (call json-decode payload User))
```

---

## 3. Tarefas de Implementação
- [x] Built-ins de I/O em `pkg/codegen`: `fs-read-file`, `fs-write-file`, `fs-exists`, `fs-remove`.
- [x] Helpers de serialização baseados em `encoding/json`: `json-encode` e `json-decode`.
- [x] Testes de escrita, leitura e parse de JSON em `examples/fs_json.liaf`.


Contrato atual e ajustes de sintaxe: [docs/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
