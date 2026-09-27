# Issue #033: LIAF v0.6 — Sintaxe Linear Baseada em Linhas e Delimitadores Leves

**Status:** Proposta / Aberta  
**Componente:** `pkg/lexer`, `pkg/parser`, `docs/linguagem/SPEC.md`, `docs/linguagem/AI_GUIDE.md`  
**Data:** 26 de setembro de 2026  

---

## 1. Contexto

A LIAF foi criada como uma linguagem desenhada primariamente para consumo e geração por **modelos de linguagem (LLMs)**.
Medições empíricas revelaram:
1. **Consumo excessivo de tokens BPE:** Parênteses aninhados (`( ... ( ... )))))`) consomem muitos tokens sem agregar semântica.
2. **Alucinação de contagem de escopo:** Modelos de linguagem frequentemente erram ao fechar 4, 5 ou 6 parênteses seguidos no final de funções ou blocos aninhados (`E_UNEXPECTED_TOKEN` ou `E_TRAILING_TOKENS`).
3. **Fragilidade de indentação invisível:** Linguagens com indentação pura (Python/YAML) sofrem com erros invisíveis de tabs/espaços.

---

## 2. Proposta (LIAF v0.6)

Evoluir da sintaxe Lisp/S-expressions para uma **sintaxe linear com delimitadores leves explícitos**:

### 2.1 Visão da Sintaxe

```liaf
// Definição de tipos
struct Item
  id int
  nome str
  preco float
end

// Funções
fn somar(a int, b int) int
  a + b
end

// Rotas declarativas
route GET "/api/cardapio" () Response effects(db)
  let db = try db-connect("sqlite", "cardapio.db")
  let itens = try db-query(db, "SELECT * FROM itens", Item)
  json-response(200, try json-encode(itens))
end

// WebSocket
ws-route "/ws/chat" (req Request, conn WSConn) effects(io, net)
  on-open
    let nome = unwrap-or(request-query(req, "name"), "Anônimo")
    ws-join(conn, "chat_geral")
    println(fmt("Conectado: {}", nome))
  end

  on-message text
    try ws-broadcast("chat_geral", text)
  end
end
```

### 2.2 Princípios de Design
1. **Fim das pirâmides de `)))))`:** Fim definitivo de erros de contagem de parênteses por IAs.
2. **Terminador explícito leve (`end`):** Elimina a ambiguidade de espaços em branco invisíveis.
3. **Preservação de 100% da AST, Type Checker e Codegen:** O pipeline intermediário e os emissores Go/C permanecem idênticos.

---

## 3. Critérios de Aceitação

1. O `pkg/lexer` e `pkg/parser` reconhecem a nova gramática sem parênteses envolventes.
2. AST resultante permanece compatível com `pkg/checker` e `pkg/codegen`.
3. Redução comprovada de pelo menos 40% nos tokens gerados para o mesmo caso de uso.
4. Suíte de conformidade adaptada e validada com 100% de sucesso.
