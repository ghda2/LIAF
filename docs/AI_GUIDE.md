# Guia para agentes — LIAF v0.5

Use S-expressions. As tags `[fn ... /fn ...]` pertencem à v0.1 e não são aceitas pelo compilador atual.

```liaf
(module exemplo
  (fn main () void (effects io)
    (println "Olá")))
```

## O que muda na v0.5 (Sintaxe Compacta, Expressões e Redução de Tokens)

A v0.5 foi projetada primariamente para **modelos de linguagem (LLMs)**: reduz em até 35% o consumo de tokens BPE, elimina palavras-chave gramaticais redundantes e introduz combinadores funcionais para evitar mutabilidade imperativa com `set`.

**1. `struct` compacta sem a tag `fields`.**
```liaf
;; Forma canônica v0.5
(struct User (id int) (name str) (email str))

;; Forma v0.4 (ainda aceita para retrocompatibilidade)
(struct User (fields (id int) (name str) (email str)))
```

**2. `fn` e `route` compactas sem `params`, `returns` e `body`.**
A assinatura segue a ordem fixa canônica: `[nome] [params] [retorno] [efeitos] [expressões...]`.
```liaf
;; Função sem parâmetros
(fn get-db () (result DBConnection str) (effects db)
  (let db DBConnection (try (db-connect "sqlite" "chat.db")))
  (ok db))

;; Função com parâmetros tipados
(fn somar ((a int) (b int)) int
  (add a b))

;; Rota HTTP declarativa compacta
(route GET "/api/users/{id}" ((id int)) Response (effects db)
  (let db DBConnection (try (get-db)))
  (let user User (try (db-query db "SELECT id, name, email FROM users WHERE id = ?" User id)))
  (json-response 200 (try (json-encode user))))
```

**3. Retorno implícito.**
Em funções e rotas com retorno diferente de `void`, a **última expressão do corpo é o retorno implícito** (como em Rust e Clojure). Não use `(return ...)` no final de um bloco quando uma expressão direta for suficiente.

**4. `match` como expressão de valor (Elimina `set` e mutabilidade).**
`match` pode ser avaliado diretamente como valor atribuível a um `let`, sem precisar declarar variáveis mutáveis com `(let x str "")` e alterá-las com `set`. Além disso, o caractere `_` é aceito como identificador wildcard para ignorar erros ou valores não utilizados:
```liaf
(let token str
  (match (request-header req "X-Admin-Token")
    (ok t t)
    (err _
      (match (request-header req "Authorization")
        (ok auth auth)
        (err _ "")))))
```

**5. Combinador `unwrap-or`.**
Desempacota `(result T E)` ou `(option T)` diretamente com valor de fallback em 1 linha, eliminando blocos inteiros de `match` com mutação:
```liaf
;; Result com valor padrão
(let nome str (unwrap-or (request-query req "name") "Anônimo"))

;; Option com valor padrão
(let porta int (unwrap-or opt-port 8080))
```

**6. Interpolação de Strings (`fmt`).**
Substitui pirâmides aninhadas de `(concat ...)` por interpolação limpa com `{}`. Use `{{` e `}}` para emitir chaves literais:
```liaf
(println (fmt "Usuário {} ({}) conectado na porta {}" nome estado porta))
(println (fmt "{{\"online\":{}}}" count))
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
- **Campos com nomes de palavras reservadas:** Nomes de campos de struct podem coincidir com operadores ou palavras reservadas (ex: `sub`, `return`, `add`), sendo acessados normalmente com `(field s sub)` (essencial para decodificar claims padrão de JWT como `sub`).

## Ciclo de trabalho

1. `liafc check arquivo.liaf --json` após cada alteração.
2. Leia `status` e `errors`. Corrija a causa e revalide. O diagnóstico traz código, arquivo, linha, coluna e mensagem.
3. `liafc fmt arquivo.liaf` mostra a forma canônica. `-w` grava, `-l` lista o que está fora do formato. O formatador ainda não preserva comentários, então `-w` se recusa a gravar em arquivo comentado a menos que você passe `--drop-comments`.
4. Rode testes de comportamento. Passar no checker não prova que o algoritmo está correto.
5. `liafc build arquivo.liaf -o programa`. Use `--embed` para assets imutáveis dentro do executável.

Programas completos em `pkg/codegen/testdata/`: `task_api_v03.liaf` (API completa), `chat_ws.liaf`, `db_postgres.liaf` e `db_redis.liaf` (banco de dados), `loops.liaf`, `collections.liaf`, `fs_json.liaf` e `result.liaf`.

## Aritmética e Funções Numéricas (#020, #022)

- `add`, `mul`, `and` e `or` são variádicos (mínimo 2 operandos): `(add 1 2 3)`, `(mul 2 3 4)`, `(and true true false)`, `(or false false true)`. `sub` e `div` permanecem estritamente binários.
- Funções matemáticas básicas: `(mod a b)`, `(neg x)`, `(abs x)`, `(min a b)`, `(max a b)`, `(pow a b)`, `(sqrt x)`, `(floor x)`, `(ceil x)`, `(round x)`.
- Literais: notação científica (`1e3`, `2.5e-3`) é avaliada como `float`; hexadecimal (`0xff`) como `int`.
- Conversões de tipo (#021): `int-from-float`, `str-from-float`, `float-from-str`, `str-from-bool`, `bool-from-str`, `str-from-int`, `float-from-int`, `int-from-str`. Conversões falíveis retornam `(result T str)`.
- Semântica numérica segura (#022):
  - `div` e `mod` com operandos `int` retornam `(result int str)`. Se o divisor for zero, produzem `(err "div: divisao por zero")` ou `(err "mod: divisao por zero")`. Trate com `match` ou `try`.
  - `div` de `float` segue o padrão IEEE 754 e retorna `float` (`+Inf`, `-Inf`, `NaN`).
  - Operações aritméticas inteiras (`add`, `sub`, `mul`, `neg`, `abs`) abortam imediatamente com código de saída `1` caso ocorra overflow/underflow, impedindo corrupção silenciosa de dados.

## Strings e UTF-8 (#023)

- `str-len(s)` retorna a quantidade de caracteres Unicode (runes UTF-8). Ex: `(str-len "ação")` retorna `4`.
- `str-byte-len(s)` retorna o tamanho em bytes para protocolos e buffers binários.
- `str-slice(s, start, end)` fatia a string em índices de runes e retorna `(result str str)`.
- `str-get(s, i)` retorna o caractere na posição `i` (índice de rune) como `(result str str)`.
- Busca e verificação: `(str-contains s sub)`, `(str-starts-with s prefix)`, `(str-ends-with s suffix)` retornam `bool`. `(str-index s sub)` retorna o índice da rune ou `-1`.
- Manipulação: `(str-trim s)`, `(str-upper s)`, `(str-lower s)` (com suporte a acentos/UTF-8), `(str-replace s old new)`.
- Listas: `(str-split s sep)` retorna `(list str)`; `(str-join l sep)` junta uma `(list str)` em `str`.
- Comparações: `lt`, `gt`, `lte`, `gte` aceitam strings e realizam comparação lexicográfica.

## Headers e query string

Declare `((req Request))` na rota para ler a requisição:

- `(request-header req "Authorization")` e `(request-query req "status")` retornam `(result str str)`: use `(unwrap-or ... padrao)` para extrair diretamente com valor padrão em 1 linha, ou trate com `try`/`match`.
- `(response-set-header res "Nome" "valor")` retorna uma nova `Response`. Não existe `set-header` que altere a resposta no lugar.
- `Set-Cookie` repetido acumula; os demais headers substituem.
- Numa `ws-route`, declare `((req Request) (conn WSConn))` para ler o handshake. O navegador não manda `Authorization` em WebSocket: leia o token com `(unwrap-or (request-query req "token") "")` em `on-open` e recuse com `(ws-close conn 4401 "motivo")`.

Exemplo completo: `pkg/codegen/testdata/pedidos_api.liaf` (Bearer, filtro por query, CORS, cookie, painel WebSocket por restaurante).

## Criptografia

- Senha: grave `(password-hash senha)` (efeito `rand`) e confira com `(password-verify senha hash)`. Nunca grave a senha nem um `sha256` dela.
- Token de sessão: `(random-token)` (efeito `rand`).
- Assinatura: `(hmac-sha256 chave texto "hex")` ou `"base64url"`; a codificação é literal. Compare com `(secure-eq recebida calculada)`, não com `str-eq`.
- `(base64url-decode s)` retorna `(result str str)`.

## Módulos e biblioteca padrão

- `(import "std/auth")` usa a biblioteca padrão embutida; `(import "./arquivo.liaf")` usa um arquivo seu.
- Não declare funções com o prefixo de um módulo da std que você importou (`auth-...`): é `E_DUPLICATE_DECL`.
- `std/auth`: `(auth-bearer-token req)` retorna o token Bearer como `(result str str)`.
- `std/jwt`: `(jwt-sign segredo payload-json)` e `(jwt-verify segredo token)` (efeito `clock`). O payload precisa de `exp` em segundos; o verify retorna o payload JSON para `json-decode`.
- `std/cors`: `(cors-origin req (list "https://site"))`, `(cors-headers res origem)`, `(cors-preflight origem "GET, POST" "Authorization")`. Uma `(route OPTIONS ...)` por caminho.

## Cliente HTTP

- `(http-fetch "POST" url (list "Authorization: Bearer x") corpo)` (efeito `net`) retorna `(result HttpReply str)`.
- `err` só quando não houve resposta. Confira `(reply-status r)`: um 422 chega como `ok`. Leia `(reply-body r)` e `(reply-header r "Nome")`.
- Headers são `(list "Nome: valor")`; `(list)` para nenhum. Corpo sem `Content-Type` vai como JSON.
- `http-get`/`http-post` NÃO são cliente: registram rotas (forma antiga).

## Web e deploy

Para sites, `serve-site` recebe pasta, domínio, porta e auto-TLS. `serve-hybrid` oferece rotas dinâmicas junto aos arquivos estáticos e retorna um resultado tratável.

Por padrão o servidor escuta em todas as interfaces. `LIAF_HOST=127.0.0.1` restringe ao loopback: use atrás de um proxy reverso (Caddy/Nginx) na mesma máquina e em testes locais. No Windows, escutar só no loopback evita o aviso do firewall. As portas 80/443 do auto-TLS não são afetadas.

Publicação: `liafc publish pagina.md --url https://HOST/_liaf/publish --path blog/pagina.md`. Defina `LIAF_DEPLOY_TOKEN` no cliente e no servidor. Reload também exige POST autenticado. Não coloque credenciais em código ou documentação.

Deploy: inspecione a unidade com `liafc service install --name site --bin /opt/site/server --dry-run`. Para Caddy: `liafc deploy caddy-bind --domain HOST --upstream HOST:PORT --server srv0`.
