# Sintaxe da LIAF (linear, v1.0)

A forma canônica é a **sintaxe linear**: blocos fechados por `end`, chamadas `f(a, b)` e
operadores infixos. O compilador ainda aceita S-expressions (`(fn ...)`) por compatibilidade:
não escreva código novo nelas, salvo o caso de [concorrência](builtins.md#10-concorrência).

Todo bloco `liaf` desta página é um programa completo, conferido pelo `liafc check` no teste
`TestSkillExamples` do repositório (e os blocos `test` são executados).

## Sumário

1. [Estrutura do arquivo](#1-estrutura-do-arquivo)
2. [Tipos](#2-tipos)
3. [Declarações, variáveis e controle](#3-declarações-variáveis-e-controle)
4. [Erros: result, option, try e on-err](#4-erros-result-option-try-e-on-err)
5. [Efeitos](#5-efeitos)
6. [Rotas HTTP e middlewares](#6-rotas-http-e-middlewares)
7. [WebSocket](#7-websocket)
8. [Banco de dados](#8-banco-de-dados)
9. [Testes](#9-testes)
10. [Armadilhas](#10-armadilhas)

## 1. Estrutura do arquivo

- **Início:** a primeira linha é `module nome`. Depois vêm, em qualquer ordem, `import`,
  `struct`, `enum`, `fn`, `middleware`, `route`, `ws-route` e `test`. Uma função pode chamar outra
  declarada mais abaixo.
- **Não há código solto nem constantes de topo:** um valor fixo vira uma `fn` sem parâmetros.
- **Comentários:** `//` ou `;;`, até o fim da linha. `#` não é comentário.
- **Ponto de entrada:** `fn main() void`, sem parâmetros (`E_MAIN_SIGNATURE`). Um arquivo sem
  `main` só é aceito se tiver blocos `test`.
- **Visibilidade:** tudo é privado ao arquivo. `pub` na frente de `fn`, `struct`, `enum` ou
  `middleware` exporta para quem o importar; usar algo privado de outro arquivo é
  `E_PRIVATE_SYMBOL`.
- **Imports:** `import "std/auth"` traz um módulo da biblioteca padrão
  ([stdlib.md](stdlib.md)); `import "./formas.liaf"` traz um arquivo seu, com o caminho relativo
  ao arquivo que importa.

Dois arquivos, lado a lado:

```text
// formas.liaf
module formas

pub fn area_retangulo(l float, a float) float
  l * a
end

// app.liaf
module app

import "./formas.liaf"

fn main() void effects(io)
  println(str-from-float(area_retangulo(2.0, 3.0)))
end
```

## 2. Tipos

| Tipo | Exemplo | Notas |
|---|---|---|
| `int` | `42`, `-7`, `0xff` | 64 bits; overflow aborta o programa |
| `float` | `3.14`, `1e3` | `1e3` é `float` |
| `str` | `"oi\n"` | UTF-8. Também guarda bytes (PDF, imagem) |
| `bool` | `true`, `false` | |
| `void` | | Só como retorno |
| `(list T)` | `list(1, 2)`, `make-list(int)` | |
| `(map K V)` | `make-map(str, int)` | Chave escalar |
| `(option T)` | `some(x)`, `none(int)` | |
| `(result T E)` | `ok(x)`, `err("motivo")` | `E` quase sempre é `str` |
| `(chan T)` | `make-chan(int)` | Veja [concorrência](builtins.md#10-concorrência) |
| `Request`, `Response`, `DBConnection`, `WSConn`, `HttpReply` | | Opacos: só os builtins os criam e leem |
| sua `struct` / seu `enum` | `new Pedido(...)`, `Status.Pago` | |

Na assinatura e nos campos, os tipos compostos vão entre parênteses: `itens (list str)`,
`fn f() (result int str)`.

## 3. Declarações, variáveis e controle

```liaf
module loja

enum Status
  Pendente
  Pago
end

struct Pedido
  id int
  cliente str
  total float
  status Status
  itens (list str)
end

fn rotulo(s Status) str
  match s
    case Status.Pendente
      "aguardando pagamento"
    case Status.Pago
      "pago"
  end
end

fn desconto(total float) float
  if total > 100.0
    return total * 0.1
  end
  0.0
end

fn classificar(n int) str
  if n < 0
    return "negativo"
  else
    if n == 0
      return "zero"
    end
  end
  "positivo"
end

fn main() void effects(io)
  let p = new Pedido(1, "Ana", 150.0, Status.Pago, list("cafe", "pao"))
  println(fmt("pedido {} de {}: {}", p.id, p.cliente, rotulo(p.status)))
  println(fmt("desconto: {}", desconto(p.total)))
  println(classificar(-3))

  for-each item p.itens
    println(item)
  end

  let contador = 0
  while contador < 10
    set contador = contador + 1
    if contador == 2
      continue
    end
    if contador == 4
      break
    end
  end

  for-range i 0 3
    println(str-from-int(i))
  end

  if contador == 4 && !false
    println("terminou")
  else
    println("nunca")
  end
end
```

- **Funções:** `fn nome(a int, b str) Tipo effects(...)`. Sem tipo de retorno, é `void`.
- **Retorno implícito:** num bloco não-`void`, a última expressão é o valor devolvido. `return`
  serve para sair antes do fim.
- **Variáveis:** `let x = expr` infere o tipo; `let x int = 10` declara. Para reatribuir,
  `set x = expr`. `let _ = expr` descarta um valor.
- **Structs:** `new Tipo(c1, c2, ...)` com os campos na ordem da declaração; leia com `p.campo`.
  Listas e mapas mudam no lugar: `list-push(p.itens, x)` altera a lista do pedido.
- **Enums:** variantes sem dados, escritas `Status.Pago`. `match` num enum cobre todas as
  variantes (`E_NONEXHAUSTIVE_MATCH`). **Variantes com dados (`Circulo(raio float)`) passam no
  parser, mas ainda não dá para construí-las:** modele com struct + enum de rótulo.
- **Condição:** `if cond ... else ... end`. **Não há `else if`:** aninhe um `if` dentro do
  `else`, como em `classificar`.
- **Laços:** `while cond ... end`; `for-range i 0 3` vai de 0 a 2; `for-each x lista`.
  `break` e `continue` valem nos três.

## 4. Erros: result, option, try e on-err

```liaf
module erros

fn dividir(a int, b int) (result int str)
  on-err motivo
    err(fmt("dividir: {}", motivo))
  end
  let q = try (a / b)
  ok(q)
end

fn dobro(texto str) (result int str)
  ok(2 * try int-from-str(texto))
end

fn achar(xs (list int), alvo int) (option int)
  for-each x xs
    if x == alvo
      return some(x)
    end
  end
  none(int)
end

fn main() void effects(io)
  match dividir(10, 0)
    case ok(v)
      println(str-from-int(v))
    case err(msg)
      println(msg)
  end

  match achar(list(1, 2, 3), 2)
    case some(v)
      println(fmt("achou {}", v))
    case none
      println("nada")
  end

  println(str-from-int(unwrap-or(dobro("21"), 0)))
  let resto = unwrap-or(mod(17, 5), 0)
  println(str-from-int(resto))
end
```

- **Falível devolve `result`:** toda operação que pode falhar devolve `(result T E)`, e o
  resultado não pode ser ignorado. Trate com `try`, `match` ou `unwrap-or`.
- **`try expr`:** devolve o valor de sucesso. No erro, desvia para o `on-err` ou devolve o erro,
  se a função retornar `result` com o mesmo tipo de erro. Sem nenhum dos dois:
  `E_UNHANDLED_RESULT`. Vale em qualquer posição: `ok(2 * try f())`,
  `json-response(200, try json-encode(x))`.
- **O `try` pega só o termo seguinte:** `try a / b` aplica o `try` em `a`. Escreva
  `try (a / b)`.
- **`on-err var ... end`:** vem logo depois da assinatura. A última expressão dele é o que a
  função devolve quando um `try` falha; `var` é a mensagem de erro (`str`).
- **Casos do `match`, entre parênteses:** `case ok(v)`, `case err(msg)`, `case some(v)` e
  `case none`. Para ignorar o valor, `case err(_)`.
- **`unwrap-or(expr, padrao)`:** desempacota `result` ou `option` com um valor padrão.

## 5. Efeitos

A assinatura declara exatamente o que a função toca: `effects(io, fs)`.

| Efeito | Para |
|---|---|
| `io` | `println`, `print`, `read-line`, `args`, `exit`, `os-exec` |
| `fs` | `fs-*`, `file-response`, `storage-*`, `serve-hybrid` |
| `net` | `serve-hybrid`, `http-fetch`, `ws-*` |
| `clock` | `now-ms`, `sleep-ms`, `time-now*`, `jwt-verify` |
| `db` | `db-*`, `redis-*` |
| `rand` | `password-hash`, `random-token`, `rand-int` |
| `env` | `env-get` |
| `spawn` | `spawn` (S-expression) |

- **Os efeitos se herdam:** chamar uma função que declara `fs` exige `fs` em quem chama.
- **Exatidão:** efeito faltando é `E_UNDECLARED_EFFECT`; sobrando, `E_UNUSED_EFFECT`.
- **Funções puras:** `json-encode`, `json-decode`, `pdf-typst`, `log-info` e todas as de texto,
  números e coleções não pedem efeito.
- **Rotas e middlewares:** cada um declara só os efeitos do próprio corpo.

## 6. Rotas HTTP e middlewares

```liaf
module tarefas

struct Tarefa
  id int
  titulo str
end

struct NovaTarefa
  titulo str
end

middleware exigir_token(req Request) (result Request Response)
  let token = unwrap-or(request-header(req, "Authorization"), "")
  if token != "Bearer segredo"
    return err(json-error(401, "token obrigatorio"))
  end
  ok(request-set-header(req, "X-Usuario", "ana"))
end

route GET "/tarefas/{id}" (id int) Response
  on-err motivo
    json-error(500, motivo)
  end
  json-response(200, try json-encode(new Tarefa(id, "Aprender LIAF")))
end

route POST "/tarefas" with(exigir_token) (entrada NovaTarefa, req Request) Response effects(io)
  on-err motivo
    json-error(500, motivo)
  end
  let autor = unwrap-or(request-header(req, "X-Usuario"), "")
  println(fmt("{} criou: {}", autor, entrada.titulo))
  json-response(201, try json-encode(new Tarefa(1, entrada.titulo)))
end

fn main() void effects(fs, io, net)
  let porta = unwrap-or(list-get(args(), 0), "8080")
  println(fmt("ouvindo na porta {}", porta))
  unwrap-or(serve-hybrid("./public", porta), false)
end
```

- **Forma:** `route MÉTODO "/caminho" [with(m1, m2)] (params) Response [effects(...)] ... end`.
  Os métodos são `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` e `HEAD`.
- **Registro:** a rota se registra sozinha e só atende depois que o `main` chama
  `serve-hybrid(pasta, porta)`, que também serve os arquivos estáticos da pasta.
- **Parâmetro de caminho:** cada `{nome}` tem um parâmetro de mesmo nome, `int` ou `str`
  (`E_ROUTE_PARAM`). O `{}` ocupa um segmento inteiro: `/x/{id}/pdf` funciona e `/x/{id}.pdf`
  não. Um `{id}` que não é número responde 400 sozinho.
- **Corpo JSON:** um parâmetro do tipo struct recebe o corpo já desserializado; corpo malformado
  vira 400.
- **Parâmetro `Request`:** dá acesso a `request-header`, `request-query`, `request-cookie`,
  `request-form`, `request-body`, `request-file-data`...
- **Resposta:** `json-response(status, texto)` recebe texto. Serialize com `json-encode` de uma
  struct, nunca com `fmt` (`W_MANUAL_JSON_FMT`); para erro, `json-error(status, msg)`.
- **Middleware:** assinatura fixa `(req Request) (result Request Response)`. `ok(req)` segue em
  frente e `err(resposta)` encerra a cadeia com essa resposta. O `with(...)` roda na ordem
  escrita, antes da leitura dos parâmetros. Para passar contexto à rota,
  `request-set-header(req, ...)`, que também substitui um header forjado pelo cliente.
- **`try` em middleware:** o erro dele é `Response`, não `str`; para um builtin que falha com
  `str`, use `unwrap-or` ou `match`.
- **Exportar:** `pub middleware` exporta como `pub fn`. `ws-route` não aceita `with`.

## 7. WebSocket

```liaf
module chat

ws-route "/ws/{sala}" (sala str, req Request, conn WSConn) effects(io, net)
  on-err motivo
    println(motivo)
  end
  on-open
    let nome = unwrap-or(request-query(req, "user"), "visitante")
    ws-join(conn, sala)
    try ws-broadcast(sala, fmt("{} entrou", nome))
  end
  on-message texto
    try ws-broadcast(sala, texto)
  end
  on-close
    println("saiu")
  end
end

fn main() void effects(fs, net)
  unwrap-or(serve-hybrid("./public", "8080"), false)
end
```

- **Parâmetros:** exatamente um `WSConn`, opcionalmente um `Request` (o do handshake), e os
  demais vindos dos `{nome}` do caminho (`E_WS_PARAM`). `effects` inclui `net`.
- **Blocos:** `on-message nome` é obrigatório; `on-open` e `on-close`, opcionais. Os três são
  `void`, então `try` lá dentro precisa do `on-err`.
- **Tópicos:** a inscrição é explícita, com `ws-join`. Fechar desinscreve de tudo.
- **Autenticação:** o navegador não envia `Authorization` no WebSocket. Leia o token da query em
  `on-open` e recuse com `ws-close(conn, 4401, "motivo")`.

## 8. Banco de dados

```liaf
module banco

struct Usuario
  id int
  nome str
end

fn listar() (result (list Usuario) str) effects(db)
  let db = try db-connect("sqlite", "app.db")
  try db-exec(db, "CREATE TABLE IF NOT EXISTS usuarios (id INTEGER PRIMARY KEY, nome TEXT)")
  db-transaction db
    try db-exec(db, "INSERT INTO usuarios (nome) VALUES (?)", "ana")
  end
  db-query(db, "SELECT id, nome FROM usuarios WHERE nome = ?", Usuario, "ana")
end

fn main() void effects(db, io)
  match listar()
    case ok(us)
      println(fmt("{} usuario(s)", list-len(us)))
    case err(msg)
      println(msg)
  end
end
```

- **SQL literal:** o SQL é **sempre** escrito inteiro no fonte (`E_SQL_INTERPOLATION`), e os
  valores vão como argumentos depois dele.
- **Marcadores:** `?` no SQLite e no MySQL, `$1, $2...` no Postgres, em número igual ao de
  argumentos (`E_SQL_PARAM_COUNT`).
- **`db-query`:** recebe uma struct como tipo de linha e casa cada coluna com o campo de mesmo
  nome.
- **Transação:** `db-transaction db ... end` confirma no fim e desfaz tudo se um `try` lá dentro
  sair da função.

## 9. Testes

```liaf
module calculo

fn total_com_desconto(total int, pct int) (result int str)
  let desconto = try (total * pct / 100)
  ok(total - desconto)
end

test "desconto de 10 por cento"
  assert-eq(unwrap-or(total_com_desconto(200, 10), 0), 180, "200 com 10% deveria dar 180")
end

test "sem desconto"
  assert(unwrap-or(total_com_desconto(50, 0), 0) == 50, "sem desconto o total nao muda")
end

fn main() void
end
```

- **Rodar:** `liafc test arquivo.liaf`, ou uma pasta (`*_test.liaf` e `test_*.liaf`, ou todos os
  `.liaf` se não houver). `--json` devolve o relatório estruturado.
- **Efeitos e `try`:** dentro de `test` todos os efeitos estão liberados, mas `try` não é aceito.
  Ponha o cenário numa `fn` que devolve `result`.
- **Asserções:** `assert`, `assert-true`, `assert-false` e `assert-eq`, descritas em
  [builtins.md](builtins.md#11-testes).

## 10. Armadilhas

O compilador atual impõe estas regras. Todas foram conferidas nele.

| Você escreveria | Escreva | Por quê |
|---|---|---|
| `else if cond` | `else` + `if cond ... end` aninhado | Não há `else if` |
| `case ok v` | `case ok(v)` | Sem parênteses, `v` não é ligado (`E_UNDEFINED_SYMBOL`) |
| `try a / b` | `try (a / b)` | O `try` pega só o termo seguinte |
| `n % 5` | `unwrap-or(mod(n, 5), 0)` | Não há `%`; `mod` devolve `result` |
| `let x = if c then a else b` | `if` com `return`, ou `match` | `if` não é expressão na sintaxe linear |
| `"a" + "b"` | `concat("a", "b")` ou `fmt` | `+` só aceita números |
| `add(a, b)`, `div(a, b)` | `a + b`, `a / b` | São palavras reservadas, não funções |
| `fmt("{{\"id\":{}}}", id)` | struct + `json-encode` | JSON com `fmt` é `W_MANUAL_JSON_FMT` (injeção) |
| `json-response(200, struct)` | `json-response(200, try json-encode(struct))` | O corpo é `str` |
| `json-decode(texto, (list Item))` | struct com um campo `(list Item)` | Argumento de tipo só aceita nome simples |
| `make-list((list str))` | uma struct que tenha a lista | Idem |
| `try` dentro de `test` | `unwrap-or`, `match`, ou uma `fn` que devolve `result` | `test` não tem `on-err` |
| `route GET "/x/{id}.pdf"` | `route GET "/x/{id}/pdf"` | `{nome}` ocupa um segmento inteiro |
| `Forma.Circulo(1.0)` (enum com dados) | struct + enum de rótulo | Construtor de variante ainda não existe |
| `spawn`/`send`/`recv` em linear | a função em S-expression | Ainda não existem na sintaxe linear |
| `const X = 1` no topo | `fn x() int` | Não há constantes de topo |
| `# comentário` | `// comentário` | `#` não é comentário |
| `liafc fmt -w app.liaf` | `liafc check` | O `fmt` reescreve o arquivo inteiro em S-expression |
