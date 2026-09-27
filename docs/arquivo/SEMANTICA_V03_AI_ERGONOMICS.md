# Especificação Semântica LIAF v0.3: Ergonomia e Baixa Carga Cognitiva para IA

**Status:** Implementada integralmente em 16/09/2026, incluindo as seções 3.1, 3.2 e 3.3, que ficaram para uma segunda passada. Mantida como registro do raciocínio de design.  
**Especificação normativa:** [docs/SPEC.md](../linguagem/SPEC.md), seção 18.  
**Data:** 16 de setembro de 2026  
**Foco:** Otimização para Small Language Models (1B a 8B) — redução drástica de aninhamento e contagem de tokens  

---

## 1. Princípios de Design

1. **Baixa Profundidade de Aninhamento:** Modelos pequenos perdem o rastreamento sintático ao ultrapassar 3-4 níveis de parênteses aninhados. O fluxo deve ser mantido prioritariamente plano (profundidade 1 a 2).
2. **Eliminação de Tokens Redundantes:** Em S-expressions, a cabeça da lista já denota a operação. A palavra-chave `call` é removida.
3. **Tratamento de Erros Declarativo e Plano:** A construção `(try expr)` substitui cadeias aninhadas de `match (ok ...) (err ...)`.
4. **Semântica Local Autossuficiente:** Assinaturas contendo tipos e efeitos bastam para o modelo consumir qualquer função sem inspecionar sua implementação.

---

## 2. Gramática e Sintaxe Central

### 2.1. Chamada Direta de Funções (Fim do `call`)
Qualquer símbolo no início de uma lista não-reservada é interpretado como chamada de função ou builtin.

```lisp
;; v0.2 (Legado verboso)
(call println (call concat "Port: " port))

;; v0.3 (Canônico)
(println (concat "Port: " port))
```

### 2.2. Acesso a Campos de Structs
O operador `field` é a forma canônica de leitura de propriedades:
```lisp
(field user name)
(field state tasks)
```

### 2.3. Propagação Plana de Erros com `(try ...)` e `(on-err ...)`

#### Regra do `(try <expr>)`:
- `<expr>` deve avaliar para um tipo `(result T E)`.
- Se o resultado for `(ok val)`, a expressão avalia diretamente para `val` do tipo `T`.
- Se o resultado for `(err msg)`:
  1. Se a função contiver um bloco `(on-err <var> <handler-body>)`, a execução salta imediatamente para o tratador, com `<var>` vinculado a `msg`.
  2. Se a função NÃO contiver `on-err`, mas a assinatura da função retornar `(result R E)`, o compilador realiza automaticamente o early-return de `(err msg)`.
  3. Se a função não retornar `(result ...)` e não tiver `on-err`, o checker rejeita o código com erro semântico de incompatibilidade de tipo.

#### Exemplo com retorno de `result`:
```lisp
(fn load-state (params) (returns (result State str)) (effects fs io)
  (body
    (if (not (fs-exists "task-api-data.json"))
      (then (return (ok (new State 1 (make-list Task))))))

    (let text str (try (fs-read-file "task-api-data.json")))
    (let state State (try (json-decode text State)))
    (return (ok state))))
```

#### Exemplo com tratador de fallback `(on-err ...)`:
```lisp
(fn save-response (params (state State) (status int) (payload str)) (returns Response) (effects fs io)
  (on-err message (return (error-response 500 message)))
  (body
    (let text str (try (json-encode state)))
    (try (fs-write-atomic "task-api-data.json" text))
    (return (json-response status payload))))
```

---

## 3. Especificação das Novas Funções da Biblioteca Padrão (Stdlib)

### 3.1. I/O Atômico e Sistema de Arquivos (`std/fs`)

| Função | Parâmetros | Retorno | Efeitos | Descrição |
| :--- | :--- | :--- | :--- | :--- |
| `fs-exists` | `(path str)` | `bool` | `fs` | Verifica existência de caminho. |
| `fs-read-file` | `(path str)` | `(result str str)` | `fs` | Lê o arquivo completo como texto. |
| `fs-write-file` | `(path str) (content str)` | `(result void str)` | `fs` | Escreve arquivo direto (não atômico). Retorna `void` no sucesso (sem duplo caminho bool). |
| `fs-rename` | `(old-path str) (new-path str)` | `(result void str)` | `fs` | Renomeia/move atomicamente no SO. |
| `fs-write-atomic` | `(path str) (content str)` | `(result void str)` | `fs` | Grava em `<path>.tmp.<uuid>` e renomeia atomicamente sobre `<path>`. |
| `fs-remove` | `(path str)` | `(result void str)` | `fs` | Remove arquivo ou diretório vazio. |

### 3.2. Serialização JSON Limpa (`std/json`)

- `(json-encode val)`: Serializa tipos primitivos, structs E coleções `(list T)` diretamente.
  - Para structs: `{"field": value, ...}`.
  - Para listas `(list T)`: array JSON nativo puro `[elem1, elem2, ...]`, sem envelopes Go/C como `{"Items": [...]}`.
- `(json-decode text Type)`: Desserializa JSON diretamente para structs ou `(list T)`.

### 3.3. Sistema Estrito de Efeitos (Checker)
- Funções puras **não podem** declarar efeitos.
- Funções que declaram `(effects io)` **são obrigadas** a invocar ao menos uma operação que consuma `io`. Efeitos mortos provocam erro de compilação.
- `json-encode` e `json-decode` são **funções puras** (sem efeitos de `io`).

---

## 4. Comparativo de Complexidade (Exemplo Prático)

### Antes (LIAF v0.2): 6 níveis de aninhamento, 4 matches, 39 linhas
```lisp
(fn save-response (params (state State) (status int) (payload str)) (returns Response) (effects fs io)
  (body
    (match (call json-encode state)
      (ok text
        (match (call fs-write-file "task-api-data.json" text)
          (ok written
            (if written
              (then (return (call json-response status payload)))
              (else (return (call error-response 500 "Failed to save tasks")))))
          (err message (return (call error-response 500 message)))))
      (err message (return (call error-response 500 message))))))
```

### Agora (LIAF v0.3): 1 nível de aninhamento, 0 matches, 6 linhas
```lisp
(fn save-response (params (state State) (status int) (payload str)) (returns Response) (effects fs io)
  (on-err message (return (error-response 500 message)))
  (body
    (let text str (try (json-encode state)))
    (try (fs-write-atomic "task-api-data.json" text))
    (return (json-response status payload))))
```
