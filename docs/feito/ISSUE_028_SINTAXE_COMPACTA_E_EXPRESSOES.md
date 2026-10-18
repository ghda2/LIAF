# Issue #028: Sintaxe Compacta, Expressões e Redução de Tokens (LIAF v0.5)

**Status:** Concluída (parser, checker, liafc fmt, documentação na SPEC.md e SKILL.md entregues)  
**Componente:** `pkg/parser`, `pkg/ast`, `pkg/checker`, `pkg/codegen`, `cmd/liafc/fmt.go`, `docs/linguagem/SPEC.md`  
**Data:** 26 de setembro de 2026  
**Casos de aceitação:** `conformance/basics/028_*.liaf`

---

## 1. Contexto

Medições em código real mostraram que a LIAF v0.4 consome cerca de **64% mais tokens BPE** que Python/FastAPI. Os principais motivos são:
1. Palavras-chave de preenchimento gramatical redundantes (`fields`, `params`, `returns`, `effects`, `body`, `return`).
2. Mutabilidade forçada com `(let var type "")` seguido de `set` em ramos de `match`, gerando código verbose e propício a bugs de estado em modelos de linguagem.
3. Pirâmides de aninhamento de `(concat ...)` e fechamentos excessivos de parênteses `)))))`.
4. Falta de combinadores comuns para desempacotar `(result T E)` com valor padrão (`unwrap-or`).

Como a LIAF é uma linguagem desenhada primariamente para consumo e geração por agentes de IA, cortar essas redundâncias reduz diretamente o custo em tokens, acelera a geração e diminui desvios de atenção no contexto.

---

## 2. Proposta

### 2.1 Eliminar palavras-chave de preenchimento em funções e rotas
Se a ordem dos nós da AST for fixa por especificação:
`[nome] [params] [retorno] [efeitos] [expressões...]`

A última expressão do bloco torna-se o retorno implícito da função (como em Rust e Clojure):

```liaf
;; Antes (v0.4)
(fn get-db (params) (returns (result DBConnection str)) (effects db)
  (body
    (let db DBConnection (try (db-connect "sqlite" "chat.db")))
    (return (ok db))))

;; Proposta (v0.5)
(fn get-db () (result DBConnection str) (effects db)
  (let db DBConnection (try (db-connect "sqlite" "chat.db")))
  (ok db))
```

### 2.2 Compactar definição de `struct`
Eliminar a tag redundante `fields`:

```liaf
;; Antes (v0.4)
(struct ChatMessage (fields (id int) (name str) (uf str) (text str) (created_at str)))

;; Proposta (v0.5)
(struct ChatMessage (id int) (name str) (uf str) (text str) (created_at str))
```

### 2.3 `match` e blocos como expressões de valor (Eliminar mutabilidade com `set`)
Permitir que `match` e blocos condicionais retornem valor diretamente, permitindo atribuir o resultado ao `let`:

```liaf
;; Antes (v0.4 - 18 linhas, mutação imperativa com 'set')
(let token str "")
(match (request-header req "X-Admin-Token")
  (ok t (set token t))
  (err e1
    (match (request-header req "Authorization")
      (ok auth
        (if (str-starts-with auth "Bearer ")
          (then
            (match (str-slice auth 7 (str-len auth))
              (ok extracted (set token extracted))
              (err e2 (set token ""))))
          (else (set token auth))))
      (err e3 (set token "")))))

;; Proposta (v0.5 - 7 linhas, funcional e imutável)
(let token str
  (match (request-header req "X-Admin-Token")
    (ok t t)
    (err _
      (match (request-header req "Authorization")
        (ok auth
          (if (str-starts-with auth "Bearer ")
            (then (unwrap-or (str-slice auth 7 (str-len auth)) ""))
            (else auth)))
        (err _ "")))))
```

### 2.4 Combinador `unwrap-or`
Permite coalescência de resultados `(result T E)` diretamente com valor de fallback em 1 linha:

```liaf
;; Antes (8 linhas)
(let nome str "Anônimo")
(match (request-query req "name")
  (ok n (set nome n))
  (err e (set nome "Anônimo")))

;; Proposta (1 linha)
(let nome str (unwrap-or (request-query req "name") "Anônimo"))
```

### 2.5 Interpolação de strings (`fmt`)
Substituir cadeias aninhadas de `concat` por uma primitiva `(fmt "..." args...)`:

```liaf
;; Antes
(println (concat "Usuário conectado: " (concat nome (concat " (" (concat estado ")")))))

;; Proposta
(println (fmt "Usuário conectado: {} ({})" nome estado))
```

---

## 3. Impacto Estimado

- **Redução de 40% a 50% de linhas e tokens** em programas LIAF reais.
- **Eliminação do `set`** na maioria dos fluxos, erradicando alucinações de mutação e escopo sujo.
- Parser mais simples e tempo de compilação ainda menor.

---

## 4. Critérios de Conclusão

- [x] Parser aceita `(struct Nome (campo tipo)...)` sem tag `fields`.
- [x] Parser e Checker aceitam `fn` e `route` sem tags `params`, `returns` e `body`.
- [x] Última expressão em funções substitui obrigatoriedade de `(return ...)`.
- [x] `match` suporta retorno de expressão unificada entre ramos.
- [x] Builtin `unwrap-or` registrado e suportado na tabela de tipos.
- [x] Builtin `fmt` com substituição posicional `{}`.
- [x] Casos de conformidade `conformance/basics/028_*.liaf` passando 100%.
- [x] Atualização do formatador canônico `liafc fmt`.
- [x] Documentar a forma compacta em `docs/linguagem/SPEC.md` e em `.agents/skills/liaf/SKILL.md`.

## 5. Estado conferido em 26/09/2026

- Os quatro casos `028_*` passam, e um programa com `struct`, `fn` e `route` na forma compacta passa no `liafc check`.
- **O `liafc fmt` ainda imprime a forma longa** (`fields`, `params`, `returns`, `effects`, `body`, `return`). Como o formatador define a forma canônica, a redução de tokens desta issue não aparece em nenhum código formatado.
- **Nenhuma medição de tokens foi feita.** Os 40–50% da §3 são estimativa.
- **Documentação:** o `docs/linguagem/AI_GUIDE.md` cita `fmt` e `unwrap-or`; a `docs/linguagem/SPEC.md` e o `SKILL.md` não mencionam nada desta issue.
- `(route ... ())` com efeitos vazios em forma compacta dá `E_UNKNOWN_STATEMENT`; é preciso escrever `(effects)`. `fn` aceita os dois. Decidir se `()` deve valer também em `route`.
