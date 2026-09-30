# Catálogo Completo de Funções Nativas (Builtins) — LIAF v0.6

Todas as funções nativas estão disponíveis globalmente (não requerem `import`).
Funções que realizam efeitos colaterais impuros exigem a declaração correspondente no bloco `effects(...)` da função ou rota chamadora.

---

## 1. Web, HTTP e Servidor

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `json-response` | `json-response(status int, corpo str) Response` | nenhum | Cria resposta HTTP com `Content-Type: application/json`. |
| `raw-response` | `raw-response(status int, tipo str, corpo str) Response` | nenhum | Cria resposta com `Content-Type` arbitrário (ex: `text/html`, binários). |
| `file-response` | `file-response(caminho str, tipo str) (result Response str)` | `fs` | Serve arquivo diretamente do disco. |
| `response-set-header` | `response-set-header(res Response, chave str, valor str) Response` | nenhum | Substitui ou define header HTTP (cookies acumulam). |
| `response-add-header` | `response-add-header(res Response, chave str, valor str) Response` | nenhum | Adiciona header HTTP (acumula valores como `Vary`). |
| `request-body` | `request-body(req Request) str` | nenhum | Obtém corpo bruto da requisição HTTP. |
| `request-path` | `request-path(req Request) str` | nenhum | Retorna o caminho da URL requisitada. |
| `request-method` | `request-method(req Request) str` | nenhum | Retorna o método HTTP (`GET`, `POST`, etc). |
| `request-header` | `request-header(req Request, chave str) (result str str)` | nenhum | Lê cabeçalho da requisição. |
| `request-query` | `request-query(req Request, param str) (result str str)` | nenhum | Lê parâmetro da query string. |
| `request-cookie` | `request-cookie(req Request, nome str) (result str str)` | nenhum | Lê valor do cookie HTTP da requisição. |
| `request-file-data` | `request-file-data(req Request, campo str) (result str str)` | nenhum | Obtém bytes de arquivo enviado via upload multipart/form-data. |
| `request-file-name` | `request-file-name(req Request, campo str) (result str str)` | nenhum | Obtém nome do arquivo enviado em formulário. |
| `serve-hybrid` | `serve-hybrid(pasta_estatica str, porta str) (result bool str)` | `fs, net` | Inicia servidor com rotas compiladas + arquivos estáticos. Bloqueia. |
| `serve-site` | `serve-site(pasta str, dominio str, porta str, auto_tls bool) void` | `fs, net` | Inicia servidor puramente estático. |

---

## 2. Cliente HTTP

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `http-fetch` | `http-fetch(metodo str, url str, headers (list str), corpo str) (result HttpReply str)` | `net` | Realiza requisição HTTP externa com timeout de 30s. |
| `reply-status` | `reply-status(res HttpReply) int` | nenhum | Obtém código de status HTTP retornado. |
| `reply-body` | `reply-body(res HttpReply) str` | nenhum | Obtém corpo da resposta do servidor externo. |
| `reply-header` | `reply-header(res HttpReply, chave str) (result str str)` | nenhum | Lê cabeçalho retornado. |

---

## 3. Banco de Dados e Redis

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `db-connect` | `db-connect(driver str, dsn str) (result DBConnection str)` | `db` | Conecta a `"postgres"`, `"mysql"`, `"sqlite"` ou `"redis"`. |
| `db-close` | `db-close(conn DBConnection) (result void str)` | `db` | Fecha a conexão (opcional, gerenciado por pool). |
| `db-query` | `db-query(conn DBConnection, sql_literal str, TipoLinha, args...) (result (list TipoLinha) str)` | `db` | Executa SELECT mapeando cada linha para a struct `TipoLinha`. O SQL deve ser literal. |
| `db-exec` | `db-exec(conn DBConnection, sql_literal str, args...) (result int str)` | `db` | Executa INSERT, UPDATE ou DELETE e retorna linhas afetadas. |
| `redis-get` | `redis-get(conn DBConnection, chave str) (result str str)` | `db` | Obtém valor de chave no Redis. |
| `redis-set` | `redis-set(conn DBConnection, chave str, valor str, ttl int) (result void str)` | `db` | Grava chave no Redis com TTL em segundos. |

### Drivers e Formatos de DSN Suportados:
- **SQLite**: `db-connect("sqlite", "banco.db")` (ou `":memory:"`). Placeholders: `?`.
- **PostgreSQL**: `db-connect("postgres", "postgres://user:pass@localhost:5432/banco?sslmode=disable")`. Placeholders: `$1, $2...`.
- **MySQL**: `db-connect("mysql", "user:pass@tcp(localhost:3306)/banco")`. Placeholders: `?`.
- **Redis**: `db-connect("redis", "redis://:senha@localhost:6379/0")` (ou `"localhost:6379"`).

```liaf
// Exemplo de uso de banco
struct Usuario
  id int
  nome str
  email str
end

fn buscar_usuarios(conn DBConnection) (result (list Usuario) str) effects(db)
  db-query(conn, "SELECT id, nome, email FROM usuarios WHERE id > ?", Usuario, 0)
end
```

---

## 4. WebSockets em Tempo Real

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `ws-send` | `ws-send(conn WSConn, texto str) (result void str)` | `net` | Envia mensagem de texto para o cliente conectado. |
| `ws-send-json` | `ws-send-json(conn WSConn, valor T) (result void str)` | `net` | Serializa valor e envia como mensagem JSON. |
| `ws-close` | `ws-close(conn WSConn, codigo int, motivo str) (result void str)` | `net` | Encerra a conexão WebSocket com código de status. |
| `ws-join` | `ws-join(conn WSConn, topico str) void` | `net` | Inscreve a conexão em um canal/tópico pub/sub. |
| `ws-leave` | `ws-leave(conn WSConn, topico str) void` | `net` | Remove a conexão do canal/tópico. |
| `ws-broadcast` | `ws-broadcast(topico str, texto str) (result int str)` | `net` | Transmite texto a todos os inscritos no tópico e retorna total. |
| `ws-topic-size` | `ws-topic-size(topico str) int` | `net` | Retorna quantidade de clientes ativos no tópico. |

---

## 5. Sistema de Arquivos (FS)

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `fs-read-file` | `fs-read-file(caminho str) (result str str)` | `fs` | Lê todo o conteúdo de um arquivo de texto. |
| `fs-write-file` | `fs-write-file(caminho str, conteudo str) (result void str)` | `fs` | Grava/sobrescreve arquivo no disco. |
| `fs-write-atomic`| `fs-write-atomic(caminho str, conteudo str) (result void str)` | `fs` | Gravação atômica via arquivo temporário + rename (sem corrupção). |
| `fs-rename` | `fs-rename(antigo str, novo str) (result void str)` | `fs` | Renomeia ou move arquivo no disco. |
| `fs-remove` | `fs-remove(caminho str) (result void str)` | `fs` | Exclui arquivo do sistema. |
| `fs-exists` | `fs-exists(caminho str) bool` | `fs` | Verifica se caminho existe. |

---

## 6. Criptografia e Segurança

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `sha256` | `sha256(texto str, encoding str) str` | nenhum | Gera hash SHA-256 (`"hex"` ou `"base64url"`). |
| `hmac-sha256` | `hmac-sha256(chave str, texto str, encoding str) str` | nenhum | Gera HMAC SHA-256 (`"hex"` ou `"base64url"`). |
| `secure-eq` | `secure-eq(a str, b str) bool` | nenhum | Comparação em tempo constante contra timing-attacks. |
| `base64url-encode` | `base64url-encode(s str) str` | nenhum | Codifica texto para Base64 URL-safe (sem padding). |
| `base64url-decode` | `base64url-decode(s str) (result str str)` | nenhum | Decodifica Base64 URL-safe. |
| `random-token` | `random-token() str` | `rand` | Gera 32 bytes criptograficamente seguros em Base64URL. |
| `password-hash`| `password-hash(senha str) str` | `rand` | Hash de senha com Argon2id formato PHC. |
| `password-verify`| `password-verify(senha str, hash str) bool` | nenhum | Verifica senha contra hash Argon2id em tempo constante. |

---

## 7. JSON e Tipagem

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `json-encode` | `json-encode(valor T) (result str str)` | nenhum | Serializa structs, listas e primitivos para string JSON. |
| `json-decode` | `json-decode(json_str str, TipoAlvo) (result TipoAlvo str)` | nenhum | Deserializa JSON estritamente para o tipo especificado. |

---

## 8. Coleções: Listas e Mapas

### Listas (`(list T)`)
| Função | Assinatura | Descrição |
|---|---|---|
| `make-list(Tipo)` | `make-list(T) (list T)` | Cria nova lista homogênea vazia. |
| `list-len(l)` | `list-len(l (list T)) int` | Retorna o número de elementos na lista. |
| `list-push(l, x)` | `list-push(l (list T), item T) void` | Adiciona item ao final da lista (mutação). |
| `list-get(l, i)` | `list-get(l (list T), idx int) (result T str)` | Obtém elemento pelo índice (0-indexado). |
| `list-set(l, i, x)`| `list-set(l (list T), idx int, item T) (result bool str)` | Substitui elemento no índice. |
| `list-remove(l, i)`| `list-remove(l (list T), idx int) (result bool str)` | Remove elemento deslocando os seguintes. |
| `list-pop(l)` | `list-pop(l (list T)) (result T str)` | Remove e devolve o último item. |
| `list-sort(l)` | `list-sort(l (list T)) void` | Ordena lista numérica ou de texto no local. |
| `list-contains(l, x)`| `list-contains(l (list T), item T) bool` | Verifica existência do item na lista. |

### Mapas (`(map K V)`)
| Função | Assinatura | Descrição |
|---|---|---|
| `make-map(K, V)` | `make-map(K, V) (map K V)` | Cria mapa vazio com chave escalar `K` e valor `V`. |
| `map-len(m)` | `map-len(m (map K V)) int` | Retorna total de chaves no mapa. |
| `map-get(m, k)` | `map-get(m (map K V), chave K) (result V str)` | Busca valor pela chave. |
| `map-set(m, k, v)`| `map-set(m (map K V), chave K, valor V) void` | Insere ou atualiza chave no mapa. |
| `map-has(m, k)` | `map-has(m (map K V), chave K) bool` | Verifica se chave existe no mapa. |
| `map-delete(m, k)`| `map-delete(m (map K V), chave K) void` | Remove chave do mapa. |
| `map-keys(m)` | `map-keys(m (map K V)) (list K)` | Devolve todas as chaves em ordem crescente. |

---

## 9. Manipulação de Texto (Strings UTF-8)

| Função | Assinatura | Descrição |
|---|---|---|
| `fmt` | `fmt(template str, args...) str` | Interpolação de texto substituindo `{}` pelos argumentos. |
| `str-len` | `str-len(s str) int` | Tamanho em caracteres UTF-8 (runes). |
| `str-byte-len` | `str-byte-len(s str) int` | Tamanho em bytes brutos UTF-8. |
| `str-slice` | `str-slice(s str, inicio int, fim int) (result str str)` | Fatia de string por caracteres (fim exclusivo). |
| `str-get` | `str-get(s str, pos int) (result str str)` | Obtém caractere na posição (0-indexado). |
| `str-index` | `str-index(s str, termo str) int` | Posição da primeira ocorrência (`-1` se ausente). |
| `str-contains` | `str-contains(s str, termo str) bool` | Verifica se contém substring. |
| `str-starts-with`| `str-starts-with(s str, prefixo str) bool` | Verifica início da string. |
| `str-ends-with` | `str-ends-with(s str, sufixo str) bool` | Verifica fim da string. |
| `str-split` | `str-split(s str, separador str) (list str)` | Divide texto por separador em lista. |
| `str-join` | `str-join(lista (list str), separador str) str` | Junta elementos da lista usando separador. |
| `str-trim` | `str-trim(s str) str` | Remove espaços em branco das pontas. |
| `str-upper` | `str-upper(s str) str` | Converte texto para maiúsculas (com acentuação). |
| `str-lower` | `str-lower(s str) str` | Converte texto para minúsculas (com acentuação). |
| `str-replace` | `str-replace(s str, antigo str, novo str) str` | Substitui todas as ocorrências de um termo. |

---

## 10. Conversões de Tipos

| Função | Retorno | Descrição |
|---|---|---|
| `str-from-int(n)` | `str` | Converte inteiro para texto decimal. |
| `str-from-float(x)` | `str` | Converte float para texto. |
| `str-from-bool(b)` | `str` | Retorna `"true"` ou `"false"`. |
| `int-from-str(s)` | `(result int str)` | Analisa texto para inteiro com validação de formato. |
| `float-from-str(s)` | `(result float str)` | Analisa texto para número real. |
| `bool-from-str(s)` | `(result bool str)` | Analisa texto para booleano. |
| `float-from-int(n)` | `float` | Converte inteiro para float explicitamente. |
| `int-from-float(x)` | `int` | Converte float para int truncando casas decimais. |

---

## 11. Matemática e Aritmética

| Função / Operador | Descrição |
|---|---|
| `+`, `-`, `*` | Operadores aritméticos naturais infixos (`a + b`, `x * y`). |
| `/` | Divisão: para `int` retorna `(result int str)` (divisão por zero é `err`). Para `float` retorna `float`. |
| `mod(a, b)` | Resto da divisão inteira `(result int str)`. |
| `neg(x)` | Inverte o sinal (`-x`). |
| `abs(x)` | Valor absoluto de inteiro ou float. |
| `min(a, b)` / `max(a, b)` | Menor ou maior valor entre dois números de mesmo tipo. |
| `pow(a, b)` | Potenciação de números de mesmo tipo. |
| `sqrt(x)` | Raiz quadrada (`float`). |
| `floor(x)` / `ceil(x)` / `round(x)` | Arredondamentos para baixo, cima ou mais próximo (`float`). |

---

## 12. Terminal, Ambiente e Tempo

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `println` | `println(args...) void` | `io` | Imprime argumentos no terminal seguidos de quebra de linha. |
| `print` | `print(args...) void` | `io` | Imprime argumentos sem quebra de linha. |
| `read-line` | `read-line() (result str str)` | `io` | Lê uma linha da entrada padrão (stdin). |
| `args` | `args() (list str)` | `io` | Lista dos argumentos passados via CLI (excluindo binário). |
| `exit` | `exit(codigo int) void` | `io` | Encerra execução imediatamente com código de saída. |
| `env-get` | `env-get(chave str) (result str str)` | `env` | Lê variável de ambiente do sistema. |
| `now-ms` | `now-ms() int` | `clock` | Retorna timestamp atual em milissegundos (Unix Epoch). |
| `sleep-ms` | `sleep-ms(ms int) void` | `clock` | Pausa a execução pela quantidade de milissegundos. |
| `rand-int` | `rand-int(max int) int` | `rand` | Gera número aleatório inteiro entre `0` e `max - 1`. |

---

## 13. Concorrência com Canais e Tarefas

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `make-chan(Tipo)` | `make-chan(T) (chan T)` | nenhum | Cria canal de comunicação tipado. |
| `send(c, valor)` | `send(c (chan T), item T) void` | nenhum | Envia valor através do canal. |
| `recv(c)` | `recv(c (chan T)) T` | nenhum | Recebe valor do canal (bloqueia até disponível). |
| `spawn` | `spawn f(...) void` | `spawn` | Executa função em paralelo / background goroutine. |

---

## 14. Documentos e PDF (#034)

Sem modelos prontos: o documento é código Typst (arquivo `.typ` ao lado do `.liaf` ou string). O motor Typst vai embutido no executável: sem Rust, sem binário extra e sem rede. Os bytes trafegam como `str`: para medir, use `str-byte-len`, não `str-len`.

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `pdf-typst` | `pdf-typst(fonte str, dados_json str) (result str str)` | nenhum | Renderiza o documento Typst em PDF. Os dados ficam em `/dados.json`. |
| `pdf-typst-files` | `pdf-typst-files(fonte str, dados_json str, arquivos (map str str)) (result str str)` | nenhum | Como `pdf-typst`, com arquivos extras (imagens, SVG, CSV): `{"img/logo.png": bytes}` fica em `/img/logo.png`. |
| `png-typst` | `png-typst(fonte str, dados_json str) (result str str)` | nenhum | Prévia PNG da primeira página (144 ppi). |
| `png-typst-files` | `png-typst-files(fonte str, dados_json str, arquivos (map str str)) (result str str)` | nenhum | Prévia com arquivos extras. |
| `pdf-response` | `pdf-response(pdf str, nome str) Response` | nenhum | Resposta HTTP `application/pdf` com `Content-Disposition: inline`. |

```liaf
let layout = try fs-read-file("curriculo.typ")
let pdf = try pdf-typst(layout, try json-encode(cv))
try fs-write-file("curriculo.pdf", pdf)
```

Fontes embutidas: Inter, Libertinus Serif, New Computer Modern Math (fórmulas `$ ... $`) e DejaVu Sans Mono (código). `liafc build --doc-fonts=inter,math` exclui as demais do binário. Pacotes do Typst Universe (`#import "@preview/nome:1.0.0"`) são baixados e embutidos pelo `liafc build`/`run`, com SHA-256 em `liaf-typst.lock` (versione esse arquivo).

Laço de correção: `liafc doc check doc.typ --dados d.json --json` devolve erros com posição (`E_TYPST`) e avisos de layout com `fix` (`W_LAYOUT_OUT_OF_PAGE`, `W_LAYOUT_BLANK_PAGE`, `W_LAYOUT_EMPTY_TAIL`, `W_LAYOUT_LOW_CONTRAST`); `liafc doc preview doc.typ -o p.png` gera a página em PNG; `liafc doc watch` reconfere a cada mudança. Exemplos: `curriculo.liaf` + `curriculo.typ` na raiz e `examples/cobranca_pix_pdf/`.
