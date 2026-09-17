# Guia para agentes — LIAF v0.4

Use S-expressions. As tags `[fn ... /fn ...]` pertencem à v0.1 e não são aceitas pelo compilador atual.

```liaf
(module exemplo
  (fn main (params) (returns void) (effects io)
    (body (do (println "Olá")))))
```

## O que muda na v0.3

A v0.3 existe para reduzir a carga cognitiva de escrever LIAF corretamente na primeira tentativa. São três mudanças, e todas as formas da v0.2 continuam aceitas.

**1. Chamadas diretas.** Escreva `(json-encode x)` em vez de `(call json-encode x)`. `call` continua válido, mas não é a forma canônica.

**2. `try` e `on-err` no lugar da pirâmide de `match`.** `(try expr)` desempacota um `(result T E)`: devolve `T` no sucesso e, no erro, sai da função na hora. O destino desse desvio é o bloco `(on-err var ...)`, que vem antes de `(body ...)`:

```liaf
(fn salvar (params (estado Estado)) (returns Response) (effects fs io)
  (on-err message (return (json-response 500 message)))
  (body
    (let texto str (try (json-encode estado)))
    (try (fs-write-atomic "estado.json" texto))
    (return (json-response 200 "{\"ok\":true}"))))
```

`try` só é permitido onde o erro tem para onde ir: uma função com `(on-err ...)` ou que retorne `(result ...)`. Sem isso o checker emite `E_UNHANDLED_RESULT`. Quando você precisa tratar os dois lados de forma diferente, `match` continua sendo a ferramenta certa.

**3. Rotas declarativas.** `(route MÉTODO "/caminho" ...)` é uma declaração de topo, irmã de `fn`:

```liaf
(route PUT "/tasks/{id}" (params (id int) (input UpdateInput)) (returns Response) (effects fs io)
  (on-err message (return (json-response 500 message)))
  (body
    (let estado Estado (try (carregar-estado)))
    (return (atualizar estado id input))))
```

Cada `{nome}` no caminho precisa de um parâmetro de mesmo nome, do tipo `int` ou `str` — caso contrário o checker emite `E_ROUTE_PARAM`. Esse parâmetro chega convertido, extraído da posição correta do caminho. Um parâmetro de struct recebe o corpo JSON já desserializado, e corpo malformado vira `400` sem nenhuma linha sua. Rotas se registram sozinhas: não chame `http-get` para elas.

## O que a v0.4 acrescenta

Banco de dados externo e WebSocket. Nenhuma forma da v0.3 mudou.

**1. Banco de dados, com o SQL sempre literal.** Drivers `"postgres"`, `"mysql"` e `"redis"`, e um efeito próprio, `db`:

```liaf
(fn buscar (params (db DBConnection) (id int)) (returns (result bool str)) (effects db io)
  (body
    (let achados (list User)
      (try (db-query db "SELECT id, name FROM users WHERE id = $1" User id)))
    (for-each u achados (println (field u name)))
    (return (ok true))))
```

**O texto da consulta tem de ser um literal escrito no fonte.** Montá-lo com `(concat ...)`, ou passar uma variável, é `E_SQL_INTERPOLATION` — não compila. Todo valor vai como argumento depois do tipo de linha (`db-query`) ou do SQL (`db-exec`), e o driver o envia fora do texto do comando. Essa é a forma segura *e* a única que existe.

O checker também confere quantos marcadores (`$1`, `$2`, ou `?` no MySQL) o SQL tem contra quantos argumentos você passou: divergir é `E_SQL_PARAM_COUNT`.

`db-query` exige uma struct como tipo de linha, e casa cada coluna com o campo de mesmo nome — `snake_case` no banco alimenta `kebab-case` na LIAF. Selecione as colunas que existem na struct.

**2. Transações que se desfazem sozinhas.**

```liaf
(db-transaction db
  (try (db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" 1 "alice"))
  (try (db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" 2 "bruno")))
```

Dentro do bloco, `db` passa a ser a transação — as consultas não mudam de forma. Confirma ao chegar ao fim; se o segundo `try` falhar e sair da função, o rollback já está garantido. Exige `(on-err ...)` ou uma função que retorne `(result ...)`.

**3. Rotas WebSocket.** `ws-route` é declaração de topo, como `route`, e não tem `(returns ...)` nem `(body ...)`:

```liaf
(ws-route "/ws/chat/{room}" (params (room str) (conn WSConn)) (effects io net)
  (on-err message (println message))
  (on-open
    (ws-join conn room))
  (on-message text
    (try (ws-broadcast room text)))
  (on-close
    (println "saiu")))
```

Exatamente um parâmetro `WSConn`; os demais vêm de `{nome}` no caminho, como nas rotas HTTP (`E_WS_PARAM` se fugir disso). `(effects ...)` tem de incluir `net`.

A inscrição no tópico é explícita: sem `ws-join`, a conexão não recebe broadcast. Fechar desinscreve de tudo, então `on-close` não precisa de `ws-leave`. `on-close` roda tanto no fechamento limpo quanto na queda da conexão.

Os três blocos devolvem `void`, então `try` dentro deles precisa de `(on-err ...)`. O handshake é `GET`, então a rota convive com suas rotas HTTP no mesmo `serve-hybrid`.

`ws-send`, `ws-send-json`, `ws-close` e `ws-broadcast` devolvem `result`; `ws-join`, `ws-leave` e `ws-topic-size` não, porque não podem falhar.

## Regras que o checker impõe

- **Declare exatamente os efeitos que usa.** `io`, `fs`, `net`, `clock`, `spawn`, `db`, incluindo os efeitos transitivos das funções que você chama. Faltando, é `E_UNDECLARED_EFFECT`; sobrando, é `E_UNUSED_EFFECT`. Uma assinatura que promete `fs` sem tocar o disco engana quem a lê.
- **Resultado não pode ser ignorado.** Toda operação falível devolve `(result T E)` e precisa de `try`, `match` ou propagação.
- **`fs-write-file`, `fs-rename`, `fs-write-atomic` e `fs-remove` retornam `(result void str)`.** O sucesso não carrega valor: `ok` já significa que a operação terminou. Não teste o conteúdo.
- **`json-encode` e `json-decode` são puras.** Não declare `io` por causa delas — serializam em memória. `json-decode` aceita tipos compostos: `(json-decode texto (list Task))`.
- **`(list T)` serializa como array JSON nativo** `[...]`. Devolva a lista direto; não monte colchetes com `concat`.

## Ciclo de trabalho

1. `liafc check arquivo.liaf --json` após cada alteração.
2. Leia `status` e `errors`. Corrija a causa e revalide. O diagnóstico traz código, arquivo, linha, coluna e mensagem.
3. `liafc fmt arquivo.liaf` mostra a forma canônica. `-w` grava, `-l` lista o que está fora do formato. O formatador ainda não preserva comentários, então `-w` se recusa a gravar em arquivo comentado a menos que você passe `--drop-comments`.
4. Rode testes de comportamento. Passar no checker não prova que o algoritmo está correto.
5. `liafc build arquivo.liaf -o programa`. Use `--embed` para assets imutáveis dentro do executável.

Exemplos executáveis: `examples/task_api_v03.liaf` (API completa em v0.3), `chat_ws.liaf` (WebSocket com salas e frontend em `public/chat.html`), `db_postgres.liaf` e `db_redis.liaf` (banco de dados), `loops.liaf`, `collections.liaf`, `fs_json.liaf`, `result.liaf` e `api_server.liaf`.

## Web e deploy

Para sites, `serve-site` recebe pasta, domínio, porta e auto-TLS. `serve-hybrid` oferece rotas dinâmicas junto aos arquivos estáticos e retorna um resultado tratável.

Publicação: `liafc publish pagina.md --url https://HOST/_liaf/publish --path blog/pagina.md`. Defina `LIAF_DEPLOY_TOKEN` no cliente e no servidor. Reload também exige POST autenticado. Não coloque credenciais em código ou documentação.

Deploy: inspecione a unidade com `liafc service install --name site --bin /opt/site/server --dry-run`. Para Caddy: `liafc deploy caddy-bind --domain HOST --upstream HOST:PORT --server srv0`.
