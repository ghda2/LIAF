# Guia para agentes — LIAF v0.7

Escreva LIAF na **sintaxe linear**: blocos fechados por `end`, chamadas `f(a, b)` e operadores
infixos. O compilador ainda aceita S-expressions (`(fn ...)`) por compatibilidade, mas não
escreva código novo nelas e não misture as duas formas no mesmo arquivo.

Todo bloco `liaf` deste guia é um programa completo, e um teste (`cmd/liafc/readme_test.go`) os
passa pelo `liafc check` a cada `go test`. Se algum exemplo daqui não compilar, o defeito está
no guia.

Para a lista completa de builtins, tipos e códigos de erro, veja a
[referência](REFERENCIA.md). Ela está escrita em S-expression, mas os nomes, argumentos e
retornos valem igual na sintaxe linear: `(json-response 200 corpo)` vira
`json-response(200, corpo)`.

## Ciclo de trabalho

1. `liafc check app.liaf --json` após cada alteração. Leia `status` e `errors`: cada erro traz
   `code`, `line`, `col` e `message`. Corrija a causa e rode de novo.
2. `liafc test app.liaf` (ou uma pasta) roda os blocos `test`. Passar no checker não prova que
   o algoritmo está certo.
3. `liafc run app.liaf` compila e executa. `liafc build app.liaf -o app` gera o binário
   autônomo; `--embed` embute os arquivos estáticos.
4. **Não rode `liafc fmt -w` num arquivo linear:** o formatador ainda imprime a forma
   canônica em S-expression e reescreveria o arquivo inteiro nela.

## O núcleo

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

fn main() void effects(io)
  let p = new Pedido(1, "Ana", 150.0, Status.Pago, list("cafe", "pao"))
  println(fmt("pedido {} de {}: {}", p.id, p.cliente, rotulo(p.status)))
  println(fmt("desconto: {}", desconto(p.total)))

  for-each item p.itens
    println(item)
  end

  let contador = 0
  while contador < 3
    set contador = contador + 1
  end

  for-range i 0 3
    println(str-from-int(i))
  end

  if contador == 3 && !false
    println("terminou")
  else
    println("nunca")
  end
end
```

- **Declarações:** `module` primeiro; depois, em qualquer ordem, `import`, `struct`, `enum`,
  `fn`, `middleware`, `route`, `ws-route` e `test`. Tudo é privado ao arquivo; `pub` na frente
  de `fn`, `struct`, `enum` ou `middleware` exporta para quem o importar.
- **Funções:** `fn nome(a int, b str) Tipo effects(...)`. Sem tipo de retorno, é `void`.
- **Retorno implícito:** num bloco não-`void`, a última expressão é o retorno. Use `return` só
  para sair antes do fim.
- **Variáveis:** `let x = expr` infere o tipo; `let x int = 10` o declara. Reatribuir exige
  `set x = expr`.
- **Structs:** `new Tipo(campo1, campo2, ...)`, com os campos na ordem da declaração; leia com
  `p.campo`.
- **Enums:** a variante se escreve `Status.Pago`. Um `match` num enum precisa cobrir todas as
  variantes (`E_NONEXHAUSTIVE_MATCH`).
- **Laços:** `for-range i 0 3` vai de 0 a 2. `break` e `continue` funcionam nos três laços.
- **Operadores:** `+ - * /`, `== != < <= > >=`, `&& ||` e `!` (ou `not(x)`).
- **Texto:** `fmt("{} e {}", a, b)` interpola. Para juntar, use `concat(a, b, ...)`: `+` só
  aceita números.

## Erros: `result`, `option`, `try`, `on-err`

```liaf
module erros

fn dividir(a int, b int) (result int str)
  on-err motivo
    err(fmt("dividir: {}", motivo))
  end
  let q = try (a / b)
  ok(q)
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

  let n = unwrap-or(int-from-str("42"), 0)
  let resto = unwrap-or(mod(n, 5), 0)
  println(str-from-int(resto))
end
```

- **Toda operação falível devolve `(result T E)`,** e o resultado não pode ser ignorado: trate
  com `try`, `match` ou `unwrap-or`.
- **`try expr`** devolve o valor em caso de sucesso. Em caso de erro, desvia para o bloco
  `on-err` ou devolve o erro, se a função retornar `result` com o mesmo tipo de erro. Sem
  nenhum dos dois, o checker acusa `E_UNHANDLED_RESULT`.
- **O `try` pega só o termo seguinte:** `try a / b` aplica o `try` em `a`. Use parênteses:
  `try (a / b)`.
- **O bloco `on-err var ... end`** vem logo depois da assinatura. A última expressão dele é o
  que a função devolve quando um `try` falha.
- **Os casos do `match` vão entre parênteses:** `case ok(v)`, `case err(msg)`, `case some(v)` e
  `case none`. Para descartar o valor, use `_`: `case err(_)`.
- **`unwrap-or(expr, padrao)`** desempacota `result` ou `option` com valor padrão.
- **Conta inteira que pode falhar:** `/` entre inteiros e `mod(a, b)` devolvem
  `(result int str)` (divisão por zero vira `err`). `+`, `-` e `*` abortam o programa em caso
  de overflow. `/` entre `float` segue o IEEE 754 e devolve `float`.
- **Ignorar um resultado:** quando o erro realmente não importa, `let _ = expr`.

## Efeitos

A assinatura declara exatamente o que a função toca: `effects(io, fs)`. Os efeitos são `io`,
`fs`, `net`, `clock`, `spawn`, `db`, `rand` e `env`, incluindo os herdados das funções que ela
chama.

- **Efeito faltando** é `E_UNDECLARED_EFFECT`; **efeito sobrando**, `E_UNUSED_EFFECT`.
- **`json-encode` e `json-decode` são puras:** não declare `io` por causa delas.
- **`serve-hybrid` exige `fs` e `net`;** `println`, `io`.

## Rotas HTTP e middlewares

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

- **A rota se registra sozinha** e só atende depois que o `main` chama `serve-hybrid`. Não
  chame `http-get` para ela: essa é a forma antiga.
- **Parâmetro de caminho:** cada `{nome}` precisa de um parâmetro de mesmo nome, `int` ou `str`
  (`E_ROUTE_PARAM`). Um `{id}` que não é número responde 400 sozinho.
- **Corpo JSON:** um parâmetro do tipo struct recebe o corpo já desserializado. Corpo malformado
  vira 400, sem nenhuma linha sua.
- **Parâmetro `Request`:** dá acesso a `request-header`, `request-query`, `request-cookie` e
  `request-body`. Os três primeiros devolvem `(result str str)`.
- **`json-response(status, texto)`** recebe texto, não struct: serialize com `json-encode`.
  `(list T)` vira um array JSON nativo.
- **Middleware:** a assinatura é fixa, `(req Request) (result Request Response)`. `ok(req)`
  segue para a rota; `err(resposta)` encerra a cadeia com essa resposta. `with(a, b)` vem logo
  depois do caminho e roda na ordem escrita. Para entregar contexto à rota, devolva
  `request-set-header(req, ...)`, que também substitui um header de mesmo nome forjado pelo
  cliente.
- **`try` dentro de middleware:** o erro dele é `Response`, não `str`. Um `try` sobre um
  builtin que falha com `str` não compila ali: use `unwrap-or` ou `match`.
- **Os efeitos da rota** são só os do próprio corpo; o middleware declara os dele.

## WebSocket

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

- **Parâmetros:** exatamente um `WSConn`; opcionalmente um `Request` (o handshake); os demais
  vêm dos `{nome}` do caminho (`E_WS_PARAM`). `effects` precisa incluir `net`.
- **Tópicos:** a inscrição é explícita, com `ws-join`; fechar a conexão desinscreve de tudo.
- **Retornos:** `ws-send`, `ws-broadcast` e `ws-close` devolvem `result`; `ws-join` não.
- **Autenticação:** o navegador não envia `Authorization` no WebSocket. Leia o token da query
  em `on-open` e recuse com `ws-close(conn, 4401, "motivo")`.

## Banco de dados

```liaf
module banco

struct Usuario
  id int
  nome str
end

fn listar() (result (list Usuario) str) effects(db)
  let db = try db-connect("sqlite", "app.db")
  try db-exec(db, "CREATE TABLE IF NOT EXISTS usuarios (id INTEGER PRIMARY KEY, nome TEXT)")
  try db-exec(db, "INSERT INTO usuarios (nome) VALUES (?)", "ana")
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

- **Drivers:** `"sqlite"`, `"postgres"`, `"mysql"` e `"redis"`, com o efeito próprio `db`.
- **O SQL tem de ser um literal escrito no fonte.** Montá-lo com `concat`/`fmt` ou passar uma
  variável é `E_SQL_INTERPOLATION`. Os valores vão como argumentos depois do SQL (`db-exec`) ou
  do tipo de linha (`db-query`).
- **Marcadores:** a quantidade de `?` (ou `$1`, `$2` no Postgres) precisa bater com a de
  argumentos (`E_SQL_PARAM_COUNT`).
- **`db-query`** recebe uma struct como tipo de linha e casa cada coluna com o campo de mesmo
  nome.
- **Transação:** `db-transaction db ... end` confirma ao chegar ao fim e desfaz tudo se um
  `try` lá dentro sair da função.

## Testes nativos

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

Asserções: `assert(cond, msg)`, `assert-true(cond)`, `assert-false(cond)` e
`assert-eq(a, b, msg)`. `liafc test --json` devolve o relatório estruturado.

## Armadilhas que o compilador atual impõe

| Você escreveria | Escreva | Por quê |
|---|---|---|
| `else if cond` | `else` + `if cond ... end` aninhado | Não há encadeamento `else if` |
| `case ok v` | `case ok(v)` | Sem parênteses, `v` não é ligado (`E_UNDEFINED_SYMBOL`) |
| `try a / b` | `try (a / b)` | O `try` pega só o termo seguinte |
| `n % 5` | `unwrap-or(mod(n, 5), 0)` | Não existe `%`; `mod` devolve `result` |
| `let x = if c then a else b` | um `if` com `return`, ou `match` | `if` não é expressão na sintaxe linear |
| `"a" + "b"` | `concat("a", "b")` ou `fmt` | `+` só aceita números |
| `add(a, b)`, `div(a, b)` | `a + b`, `a / b` | São palavras reservadas, não funções |
| `json-response(200, struct)` | `json-response(200, try json-encode(struct))` | O corpo é `str` |
| `liafc fmt -w app.liaf` | `liafc check` | O `fmt` reescreve em S-expression |

## Onde aprofundar

- [REFERENCIA.md](REFERENCIA.md): todos os builtins, tipos, a biblioteca padrão (`std/auth`,
  `std/jwt`, `std/cors`, `std/storage`, `std/time`, `std/regex`...) e todos os códigos de erro.
- `pkg/codegen/testdata/task_api_v06.liaf` e `middleware_api.liaf`: APIs completas e
  executadas em teste.
- `examples/`: geração de PDF com Typst (`cobranca_pix_pdf`, `relatorio_lambari`).
