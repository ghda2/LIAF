# Issue #006: Tratamento Estruturado de Erros (Result & Checked Errors)

**Status:** Implementada; revisao semantica em andamento
**Componente:** `pkg/parser`, `pkg/ast`, `pkg/codegen`, `pkg/diagnostic`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
Para cumprir a premissa de "AI First" e auto-cura sem panics silenciosos ou exceções cegas em runtime, o compilador da LIAF deve exigir que operações falíveis tratem seus erros explicitamente através do padrão `Result[T, E]` ou checagem determinística.

---

## 2. Especificações de Sintaxe

### 2.1. Tipo Result
Operações que podem falhar retornam um tipo encapsulado:
```liaf
(fn parse-config
  (params (path str))
  (returns (result str str))
  (effects io)
  (body
    (if (call fs-exists path)
      (then (return (ok (call fs-read-file path))))
      (else (return (err "Arquivo nao encontrado"))))))
```

### 2.2. Extração Segura com Match/Try
```liaf
(match (call parse-config "./app.conf")
  (ok data (do (call println data)))
  (err msg (do (call println (call concat "Erro: " msg)))))
```

---

## 3. Tarefas de Implementação
- [ ] Implementar tipos `(result T E)`, `ok` e `err` no sistema de tipos do compilador.
- [ ] Criar estrutura de controle de fluxo `(match ...)` para desestruturação de `Result`.
- [ ] Validador semântico que impede ignorar valores do tipo `Result` sem tratamento.
- [ ] Geração de código idiomático com structs de status em Go.


Contrato atual e ajustes de sintaxe: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
