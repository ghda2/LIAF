---
name: liaf
description: Guia definitivo e workflow para agentes de IA escreverem, validarem com auto-cura mecânica, compilarem e realizarem deploy de aplicações na linguagem LIAF (Language for AI First).
---

# LIAF — AI Agent Skill

Este guia capacita agentes de IA a gerar código, auto-curar erros e compilar aplicações utilizando a linguagem **LIAF (Language for AI First)**.

---

## 1. Princípio Fundamental: Sintaxe Linear v0.6 (69% Menos Tokens)

A partir da versão v0.6, a LIAF adota **sintaxe linear baseada em linhas e delimitadores leves (`end`)**. Não use pirâmides de parênteses aninhados da v0.1/v0.3.

```liaf
// Definição de tipos
struct Task
  id int
  title str
  done bool
end

// Funções com operadores infixos naturais
fn somar(a int, b int) int
  a + b
end

// Rotas declarativas com try/on-err planos
route GET "/tasks" () Response effects(fs)
  on-err msg
    json-response(500, msg)
  end
  let state = try load-state()
  json-response(200, try json-encode(state.tasks))
end
```

### Regras de Ouro para Modelos de IA:
1. **Delimitadores explícitos:** Blocos como `struct`, `fn`, `route`, `ws-route`, `if`, `while`, `for-range` terminam sempre com `end`.
2. **Retorno implícito:** Em funções e rotas não-void, a **última expressão avaliada é o retorno**. Evite `return` redundante no final do bloco.
3. **Operadores infixos:** Use `+`, `-`, `*`, `/`, `==`, `!=`, `<`, `<=`, `>`, `>=`, `&&`, `||`, `!`.
4. **Tratamento de erros:** Use `try expr` para propagar resultados e `on-err errVar` para capturar falhas sem aninhar `match`.
5. **Inferência de tipos:** `let nome = expr` infere o tipo automaticamente. Especifique apenas se necessário (`let x int = 10`).
6. **Acesso a campos:** Use `objeto.campo` diretamente (ex: `state.next_id`, `req.user.name`).
7. **Efeitos obrigatórios:** Declare `effects(...)` para operações impuras (`io`, `fs`, `net`, `clock`, `db`, `rand`). Funções puras omitem ou usam `effects()`.

---

## 2. Loop de Auto-Cura Mecânica (Auto-Healing Loop)

Sempre que gerar ou editar um arquivo `.liaf`, execute a validação autônoma:

```bash
liafc check caminho/app.liaf --json
```

### Resposta JSON do Compilador:
- Se `"status": "success"`: Código pronto para compilar.
- Se `"status": "error"`: O compilador devolve as coordenadas e o patch exato:
  - `code`: Identificador estático do erro (ex: `E_UNDEFINED_SYMBOL`, `E_TYPE_MISMATCH`).
  - `line` e `col`: Coordenadas exatas.
  - `suggested_patch`: Correção mecânica recomendada pelo compilador.

Aplique o patch sugerido e reexecute `liafc check` até obter sucesso. Veja o catálogo detalhado em [references/auto_healing.md](references/auto_healing.md).

---

## 3. Comandos do Compilador (`liafc`)

O executável `liafc` é a ferramenta central:

| Ação | Comando | Descrição |
|---|---|---|
| **Checar** | `liafc check app.liaf --json` | Valida sintaxe e tipos emitindo JSON para auto-cura |
| **Formatar** | `liafc fmt app.liaf -w` | Normaliza a indentação e o layout canônico |
| **Executar** | `liafc run app.liaf` | Executa o programa diretamente |
| **Compilar** | `liafc build app.liaf -o server.exe` | Compila executável nativo autônomo |
| **Cross-compilar** | `$env:GOOS="linux"; liafc build app.liaf -o app_linux` | Gera binário para servidor Linux x86_64 |

---

## 4. Biblioteca de Referência

Consulte os guias especializados da skill conforme sua necessidade:
- [references/syntax.md](references/syntax.md): Especificação completa da gramática v0.6 e v0.5.
- [references/stdlib.md](references/stdlib.md): Guia de `std/cookie`, `std/auth`, `std/jwt`, `std/cors` e `std/storage`.
- [references/auto_healing.md](references/auto_healing.md): Tabela de códigos de erro e estratégias de correção.
- [examples/rest_api.liaf](examples/rest_api.liaf): Exemplo canônico de API REST com persistência atômica.
- [examples/websocket.liaf](examples/websocket.liaf): Exemplo canônico de WebSockets e pub/sub.
- [examples/auth_cookies.liaf](examples/auth_cookies.liaf): Exemplo canônico de autenticação por Cookies e rotas seguras.
