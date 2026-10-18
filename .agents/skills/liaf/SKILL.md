---
name: liaf
description: Escrever, validar, testar, compilar e publicar programas na linguagem LIAF (Language for AI First) — APIs HTTP, middlewares, WebSocket, banco de dados, CLIs, PDFs e testes nativos. Use sempre que o usuário pedir código em LIAF, mencionar arquivos .liaf, o compilador liafc, ou erros como E_UNDECLARED_EFFECT e E_UNHANDLED_RESULT.
---

# LIAF — skill para agentes

A LIAF compila para um binário Go autônomo. Ela foi feita para um agente escrever código certo
com poucas tentativas: tipos e efeitos são explícitos, todo erro vem em `result`, e o
`liafc check --json` aponta cada problema com código, linha e coluna.

Esta skill descreve **o compilador como ele é hoje**. Todo bloco `liaf` dela é um programa
completo, conferido pelo teste `TestSkillExamples` do repositório da LIAF, e os blocos `test`
são executados. Se o que está escrito aqui divergir do `liafc`, o `liafc` está certo; avise o
usuário.

## Como trabalhar

1. Escreva o arquivo `.liaf` na **sintaxe linear** ([references/syntax.md](references/syntax.md)).
2. Rode `liafc check app.liaf --json`. Leia o **primeiro** erro, corrija a causa e rode de novo
   até `"status": "success"`. Cada código está em
   [references/auto_healing.md](references/auto_healing.md).
3. Escreva blocos `test` para a lógica e rode `liafc test app.liaf`. Passar no checker não
   prova que o algoritmo está certo.
4. Rode com `liafc run app.liaf`, ou gere o binário com `liafc build app.liaf -o app`.
5. **Não rode `liafc fmt -w`** num arquivo linear: o formatador ainda reescreve tudo em
   S-expression.

## Programa mínimo

```liaf
module ola

struct Saudacao
  nome str
  vezes int
end

fn montar(nome str) (result Saudacao str)
  if nome == ""
    return err("nome vazio")
  end
  ok(new Saudacao(nome, str-len(nome)))
end

test "monta a saudacao"
  assert-eq(unwrap-or(montar("Ana"), new Saudacao("", 0)).vezes, 3, "Ana tem 3 letras")
end

fn main() void effects(io)
  match montar("Ana")
    case ok(s)
      println(fmt("Olá, {}!", s.nome))
    case err(e)
      println(e)
  end
end
```

## As regras que mais derrubam o `check`

1. **Efeitos exatos:** declare em `effects(...)` cada efeito usado (`io`, `fs`, `net`, `clock`,
   `db`, `rand`, `env`, `spawn`), inclusive os das funções chamadas. Nem a mais nem a menos.
2. **`result` não se ignora:** trate com `try` (precisa de `on-err` ou de a função devolver
   `result` com o mesmo tipo de erro), `match` ou `unwrap-or`.
3. **Casos do `match` entre parênteses:** `case ok(v)`, `case err(e)`, `case some(v)` e
   `case none`.
4. **Sem `else if`, `%`, `if` como valor nem `+` em texto:** aninhe o `if`, use
   `mod(a, b)`, `return`/`match` e `concat`/`fmt`.
5. **`try` só pega o termo seguinte:** `try (a / b)`. `/` e `mod` entre inteiros devolvem
   `result`.
6. **JSON só por struct:** `json-response(200, try json-encode(x))`. JSON montado com `fmt` é
   recusado (`W_MANUAL_JSON_FMT`); para erro, `json-error(status, msg)`.
7. **SQL sempre literal**, com os valores como argumentos.

A lista completa de armadilhas está no fim de [references/syntax.md](references/syntax.md).

## Referências

| Arquivo | Conteúdo |
|---|---|
| [references/syntax.md](references/syntax.md) | Toda a sintaxe: módulos, tipos, controle, erros, efeitos, rotas, middlewares, WebSocket, banco, testes e armadilhas |
| [references/builtins.md](references/builtins.md) | As 185 funções nativas, com assinatura, efeito e um programa testado por categoria |
| [references/stdlib.md](references/stdlib.md) | As 84 funções de `std/auth`, `jwt`, `cookie`, `cors`, `html`, `seo`, `storage`, `cli`, `csv`, `fs`, `regex`, `time` |
| [references/auto_healing.md](references/auto_healing.md) | Todos os códigos de erro (`E_...`, `W_...`) e a correção de cada um |
| Skill `liaf-pdf` | PDFs e imagens com Typst: layout, dados, `liafc doc check` |

## Exemplos completos

| Arquivo | O que mostra |
|---|---|
| [examples/rest_api.liaf](examples/rest_api.liaf) | API REST com persistência atômica em JSON |
| [examples/auth_jwt.liaf](examples/auth_jwt.liaf) | Login com Argon2id, emissão de JWT e rota protegida |
| [examples/auth_cookies.liaf](examples/auth_cookies.liaf) | Sessão por cookie `HttpOnly`/`Secure` |
| [examples/websocket.liaf](examples/websocket.liaf) | Chat com `ws-route` e broadcast |
| [examples/cli_tool.liaf](examples/cli_tool.liaf) | Utilitário de terminal com flags e CSV |
| [examples/server_driven_ui.liaf](examples/server_driven_ui.liaf) | HTML gerado no servidor com `std/html` |
| [examples/demo_app](examples/demo_app) | App com HTMX e arquivos estáticos em `public/` |
| [examples/demo_pro](examples/demo_pro) | App completo: SQLite, WebSocket multiusuário, SVG no backend, rate limit |

## Comandos do `liafc`

| Comando | Para |
|---|---|
| `liafc check app.liaf --json` | Validar sintaxe, tipos e efeitos |
| `liafc test [arquivo ou pasta] [--json]` | Rodar os blocos `test` |
| `liafc run app.liaf` | Compilar em memória e executar |
| `liafc build app.liaf -o app [--embed]` | Gerar o binário (`--embed` embute os estáticos) |
| `liafc new nome` / `liafc init [pasta]` | Criar um projeto |
| `liafc doc check doc.typ --dados d.json --json` | Conferir um documento Typst (skill `liaf-pdf`) |
| `liafc emit app.liaf` | Ver o Go gerado (para depurar o compilador) |
| `liafc agents [pasta]` | Gravar estas skills em `.agents/skills` |

Para gerar binário para Linux a partir do Windows: `$env:GOOS="linux"; liafc build app.liaf -o app`.
O servidor escuta em todas as interfaces; `LIAF_HOST=127.0.0.1` restringe ao loopback (atrás de
um proxy reverso ou em testes).
