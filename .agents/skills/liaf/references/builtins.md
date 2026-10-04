# Catálogo Completo de Funções Nativas (Builtins) — LIAF v0.6

Todas as funções nativas estão disponíveis globalmente (não requerem `import`).
Funções com efeitos colaterais impuros exigem a declaração correspondente no bloco `effects(...)` da função ou rota chamadora.

---

## 1. Web, HTTP e Servidor

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `json-response` | `json-response(status int, corpo str) Response` | nenhum | Cria resposta HTTP com `Content-Type: application/json`. |
| `html-response` | `html-response(status int, corpo str) Response` | nenhum | Cria resposta HTTP com `Content-Type: text/html; charset=utf-8`. |
| `html-escape` | `html-escape(texto str) str` | nenhum | Escapa caracteres HTML (`&`, `<`, `>`, `"`, `'`) contra XSS. |
| `raw-response` | `raw-response(status int, tipo str, corpo str) Response` | nenhum | Cria resposta com `Content-Type` arbitrário (ex: `text/html`, binários). |
| `json-error` | `json-error(status int, mensagem str) Response` | nenhum | Cria resposta JSON de erro `{"error": "mensagem"}` com escape seguro de caracteres especiais. |
| `file-response` | `file-response(caminho str, tipo str) (result Response str)` | `fs` | Serve arquivo diretamente do disco. |
| `response-set-header` | `response-set-header(res Response, chave str, valor str) Response` | nenhum | Substitui ou define cabeçalho HTTP (cookies acumulam). |
| `response-add-header` | `response-add-header(res Response, chave str, valor str) Response` | nenhum | Adiciona cabeçalho HTTP (acumula valores). |
| `req-client-ip` | `req-client-ip(req Request) str` | nenhum | Extrai o IP real do cliente ciente de proxies (`X-Forwarded-For`, `X-Real-IP`). |
| `rate-limit` | `rate-limit(chave str, max_requisicoes int, janela_segundos int) bool` | nenhum | Rate limiting em memória com janela deslizante. Retorna `false` se o limite for ultrapassado. |
| `request-body` | `request-body(req Request) str` | nenhum | Obtém o corpo textual bruto da requisição HTTP. |
| `request-path` | `request-path(req Request) str` | nenhum | Retorna o caminho da URL requisitada. |
| `request-method` | `request-method(req Request) str` | nenhum | Retorna o método HTTP (`GET`, `POST`, `PUT`, `DELETE`, etc). |
| `request-header` | `request-header(req Request, chave str) (result str str)` | nenhum | Lê cabeçalho específico da requisição. |
| `request-query` | `request-query(req Request, param str) (result str str)` | nenhum | Lê parâmetro da query string (`?chave=valor`). |
| `request-form` | `request-form(req Request, campo str) (result str str)` | nenhum | Lê campo de formulário submetido via POST (`x-www-form-urlencoded` ou query fallback). |
| `request-cookie` | `request-cookie(req Request, nome str) (result str str)` | nenhum | Lê cookie HTTP específico da requisição. |
| `request-file-data` | `request-file-data(req Request, campo str) (result str str)` | nenhum | Obtém bytes de arquivo enviado via formulário multipart. |
| `request-file-name` | `request-file-name(req Request, campo str) (result str str)` | nenhum | Obtém o nome original do arquivo enviado. |
| `serve-hybrid` | `serve-hybrid(pasta_estatica str, porta str) (result bool str)` | `fs, net` | Inicia servidor com rotas compiladas + arquivos estáticos com fallback gracioso. Bloqueia o processo. |
| `serve-site` | `serve-site(pasta str, dominio str, porta str, auto_tls bool) void` | `fs, net` | Inicia servidor puramente estático. |

---

## 2. Cliente HTTP

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `http-fetch` | `http-fetch(metodo str, url str, headers (list str), corpo str) (result HttpReply str)` | `net` | Realiza requisição HTTP externa com timeout automático de 30s. |
| `reply-status` | `reply-status(res HttpReply) int` | nenhum | Obtém o código de status HTTP retornado pelo servidor remoto. |
| `reply-body` | `reply-body(res HttpReply) str` | nenhum | Obtém o corpo da resposta do servidor externo. |
| `reply-header` | `reply-header(res HttpReply, chave str) (result str str)` | nenhum | Lê cabeçalho retornado na resposta remota. |

---

## 3. Banco de Dados e Redis

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `db-connect` | `db-connect(driver str, dsn str) (result DBConnection str)` | `db` | Conecta a `"sqlite"`, `"postgres"`, `"mysql"` ou `"redis"`. |
| `db-close` | `db-close(conn DBConnection) (result void str)` | `db` | Fecha a conexão (gerenciado automaticamente por pool). |
| `db-query` | `db-query(conn DBConnection, sql_literal str, TipoLinha, args...) (result (list TipoLinha) str)` | `db` | Executa SELECT mapeando cada linha para a struct `TipoLinha`. Suporta `?1, ?2...` ou `?` no SQLite/MySQL e `$1, $2...` no Postgres. |
| `db-exec` | `db-exec(conn DBConnection, sql_literal str, args...) (result int str)` | `db` | Executa INSERT, UPDATE ou DELETE e retorna quantidade de linhas afetadas. |
| `redis-get` | `redis-get(conn DBConnection, chave str) (result str str)` | `db` | Obtém valor de chave no Redis. |
| `redis-set` | `redis-set(conn DBConnection, chave str, valor str, ttl int) (result void str)` | `db` | Grava chave no Redis com TTL em segundos. |

---

## 4. WebSockets em Tempo Real

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `ws-send` | `ws-send(conn WSConn, texto str) (result void str)` | `net` | Envia mensagem de texto diretamente para o cliente conectado. |
| `ws-send-json` | `ws-send-json(conn WSConn, valor T) (result void str)` | `net` | Serializa valor para JSON e envia para o cliente. |
| `ws-close` | `ws-close(conn WSConn, codigo int, motivo str) (result void str)` | `net` | Encerra a conexão WebSocket com código e mensagem. |
| `ws-join` | `ws-join(conn WSConn, topico str) void` | `net` | Inscreve a conexão em uma sala/tópico pub/sub. |
| `ws-leave` | `ws-leave(conn WSConn, topico str) void` | `net` | Remove a conexão da sala/tópico. |
| `ws-broadcast` | `ws-broadcast(topico str, texto str) (result int str)` | `net` | Transmite texto para todos os inscritos na sala e retorna o total notificado. |
| `ws-topic-size` | `ws-topic-size(topico str) int` | `net` | Retorna a quantidade de conexões ativas na sala/tópico. |

---

## 5. Sistema de Arquivos (FS) e Subprocessos (OS)

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `fs-read-file` | `fs-read-file(caminho str) (result str str)` | `fs` | Lê o conteúdo completo de um arquivo de texto. |
| `fs-write-file` | `fs-write-file(caminho str, conteudo str) (result void str)` | `fs` | Grava/sobrescreve arquivo no disco. |
| `fs-write-atomic`| `fs-write-atomic(caminho str, conteudo str) (result void str)` | `fs` | Gravação atômica via arquivo temporário (à prova de corrupção). |
| `fs-mkdir-all` | `fs-mkdir-all(caminho str) (result bool str)` | `fs` | Cria a árvore de diretórios especificada (`mkdir -p`). |
| `fs-remove-all`| `fs-remove-all(caminho str) (result bool str)` | `fs` | Remove arquivo ou diretório recursivamente (`rm -rf`). |
| `fs-list-dir` | `fs-list-dir(caminho str) (result (list str) str)` | `fs` | Lista todos os arquivos e pastas dentro de um diretório. |
| `fs-is-dir` | `fs-is-dir(caminho str) bool` | `fs` | Verifica se o caminho especificado é um diretório. |
| `fs-file-size` | `fs-file-size(caminho str) (result int str)` | `fs` | Obtém o tamanho em bytes do arquivo especificado. |
| `fs-rename` | `fs-rename(antigo str, novo str) (result void str)` | `fs` | Renomeia ou move arquivo/diretório no disco. |
| `fs-exists` | `fs-exists(caminho str) bool` | `fs` | Verifica se arquivo ou pasta existe. |
| `cli-args` | `cli-args() (list str)` | nenhum | Devolve a lista de argumentos de linha de comando (sem o binário). |
| `exit-process` | `exit-process(codigo int) void` | `io` | Encerra o processo imediatamente com o código fornecido. |
| `os-exec` | `os-exec(cmd str, args (list str)) (result str str)` | `io` | Executa subprocesso e captura sua saída padrão. |
| `os-exec-status`| `os-exec-status(cmd str, args (list str)) int` | `io` | Executa subprocesso e devolve seu código de saída inteiro. |

---

## 6. Formatos de Dados: JSON e CSV

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `json-encode` | `json-encode(valor T) (result str str)` | nenhum | Serializa structs, listas e tipos primitivos para string JSON. |
| `json-decode` | `json-decode(json_str str, TipoAlvo) (result TipoAlvo str)` | nenhum | Deserializa string JSON estritamente para a struct ou tipo indicado. |
| `csv-parse` | `csv-parse(conteudo str) (result (list (list str)) str)` | nenhum | Analisa string CSV em uma matriz de linhas e colunas. |
| `csv-encode` | `csv-encode(linhas (list (list str))) str` | nenhum | Serializa matriz de linhas de volta para formato CSV. |

---

## 7. Criptografia, JWT e Segurança

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `jwt-sign` | `jwt-sign(segredo str, payload_json str) (result str str)` | nenhum | Assina payload JSON com HS256. Segredo exige mínimo de 32 bytes; payload deve conter `"exp"`. |
| `jwt-verify` | `jwt-verify(segredo str, token str) (result str str)` | `clock` | Valida assinatura e expiração do token JWT; devolve o payload JSON. |
| `sha256` | `sha256(texto str, encoding str) str` | nenhum | Gera hash SHA-256 (`"hex"` ou `"base64url"`). |
| `hmac-sha256` | `hmac-sha256(chave str, texto str, encoding str) str` | nenhum | Gera HMAC SHA-256 (`"hex"` ou `"base64url"`). |
| `secure-eq` | `secure-eq(a str, b str) bool` | nenhum | Comparação de strings em tempo constante contra ataques de temporização. |
| `base64url-encode` | `base64url-encode(s str) str` | nenhum | Codifica texto para Base64 URL-safe (sem padding `=`). |
| `base64url-decode` | `base64url-decode(s str) (result str str)` | nenhum | Decodifica Base64 URL-safe. |
| `random-token` | `random-token() str` | `rand` | Gera 32 bytes criptograficamente seguros codificados em Base64URL. |
| `password-hash`| `password-hash(senha str) str` | `rand` | Gera hash de senha com Argon2id formato PHC. |
| `password-verify`| `password-verify(senha str, hash str) bool` | nenhum | Valida senha contra hash Argon2id em tempo constante. |

---

## 8. Coleções: Listas e Mapas

### Listas (`(list T)`)
| Função | Assinatura | Descrição |
|---|---|---|
| `make-list(Tipo)` | `make-list(T) (list T)` | Cria nova lista homogênea vazia do tipo `T`. |
| `list-len(l)` | `list-len(l (list T)) int` | Retorna a quantidade de elementos na lista. |
| `list-push(l, x)` | `list-push(l (list T), item T) void` | Adiciona item ao final da lista (mutação). |
| `list-get(l, i)` | `list-get(l (list T), idx int) (result T str)` | Obtém elemento pelo índice (0-indexado). |
| `list-set(l, i, x)`| `list-set(l (list T), idx int, item T) (result bool str)` | Substitui elemento no índice especificado. |
| `list-remove(l, i)`| `list-remove(l (list T), idx int) (result bool str)` | Remove elemento deslocando os itens posteriores. |
| `list-pop(l)` | `list-pop(l (list T)) (result T str)` | Remove e devolve o último item da lista. |
| `list-sort(l)` | `list-sort(l (list T)) void` | Ordena lista numérica ou de texto no local. |
| `list-contains(l, x)`| `list-contains(l (list T), item T) bool` | Verifica se o item está presente na lista. |

### Mapas (`(map K V)`)
| Função | Assinatura | Descrição |
|---|---|---|
| `make-map(K, V)` | `make-map(K, V) (map K V)` | Cria mapa vazio com chave escalar `K` e valor `V`. |
| `map-len(m)` | `map-len(m (map K V)) int` | Retorna o total de chaves registradas no mapa. |
| `map-get(m, k)` | `map-get(m (map K V), chave K) (result V str)` | Busca valor associado à chave. |
| `map-set(m, k, v)`| `map-set(m (map K V), chave K, valor V) void` | Insere ou substitui valor da chave no mapa. |
| `map-has(m, k)` | `map-has(m (map K V), chave K) bool` | Verifica se a chave existe no mapa. |
| `map-delete(m, k)`| `map-delete(m (map K V), chave K) void` | Remove a chave do mapa. |
| `map-keys(m)` | `map-keys(m (map K V)) (list K)` | Devolve todas as chaves em ordem crescente. |

---

## 9. Manipulação de Texto (Strings UTF-8)

| Função | Assinatura | Descrição |
|---|---|---|
| `fmt` | `fmt(template str, args...) str` | Interpolação substituindo `{}` pelos argumentos em ordem. |
| `str-len` | `str-len(s str) int` | Quantidade de caracteres/runas UTF-8. |
| `str-byte-len` | `str-byte-len(s str) int` | Quantidade de bytes brutos UTF-8. |
| `str-slice` | `str-slice(s str, inicio int, fim int) (result str str)` | Substring por caracteres (índice final exclusivo). |
| `str-get` | `str-get(s str, pos int) (result str str)` | Obtém caractere na posição (0-indexado). |
| `str-index` | `str-index(s str, termo str) int` | Posição da primeira ocorrência (`-1` se ausente). |
| `str-contains` | `str-contains(s str, termo str) bool` | Verifica se contém a substring especificada. |
| `str-starts-with`| `str-starts-with(s str, prefixo str) bool` | Verifica se a string inicia com o prefixo. |
| `str-ends-with` | `str-ends-with(s str, sufixo str) bool` | Verifica se a string termina com o sufixo. |
| `str-split` | `str-split(s str, separador str) (list str)` | Divide texto pelo delimitador retornando lista de partes. |
| `str-join` | `str-join(lista (list str), separador str) str` | Junta elementos da lista usando separador intermediário. |
| `str-trim` | `str-trim(s str) str` | Remove espaços em branco das extremidades. |
| `str-upper` | `str-upper(s str) str` | Converte texto para maiúsculas (compatível com acentuação). |
| `str-lower` | `str-lower(s str) str` | Converte texto para minúsculas (compatível com acentuação). |
| `str-replace` | `str-replace(s str, antigo str, novo str) str` | Substitui todas as ocorrências do termo antigo pelo novo. |

---

## 10. Conversões de Tipos

| Função | Retorno | Descrição |
|---|---|---|
| `str-from-int(n)` | `str` | Converte inteiro para texto decimal. |
| `str-from-float(x)` | `str` | Converte número float para texto. |
| `str-from-bool(b)` | `str` | Retorna `"true"` ou `"false"`. |
| `int-from-str(s)` | `(result int str)` | Analisa texto para inteiro com validação de formato. |
| `float-from-str(s)` | `(result float str)` | Analisa texto para número real. |
| `bool-from-str(s)` | `(result bool str)` | Analisa texto para booleano. |
| `float-from-int(n)` | `float` | Converte inteiro para float explicitamente. |
| `int-from-float(x)` | `int` | Converte float para int truncando casas decimais. |

---

## 11. Terminal, Logs e Ambiente

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `println` | `println(args...) void` | `io` | Imprime argumentos no terminal seguidos de quebra de linha. |
| `print` | `print(args...) void` | `io` | Imprime argumentos sem quebra de linha. |
| `log-info` | `log-info(args...) void` | nenhum | Emite log formatado com prefixo `[INFO]` (isento de efeito `io`). |
| `log-error` | `log-error(args...) void` | nenhum | Emite log formatado com prefixo `[ERROR]` (isento de efeito `io`). |
| `read-line` | `read-line() (result str str)` | `io` | Lê uma linha da entrada padrão (stdin). |
| `env-get` | `env-get(chave str) (result str str)` | `env` | Lê variável de ambiente do sistema operacional. |
| `now-ms` | `now-ms() int` | `clock` | Retorna timestamp atual em milissegundos (Unix Epoch). |
| `sleep-ms` | `sleep-ms(ms int) void` | `clock` | Pausa a execução pela quantidade de milissegundos. |
| `rand-int` | `rand-int(max int) int` | `rand` | Gera número aleatório inteiro entre `0` e `max - 1`. |

---

## 12. Documentos e PDFs Nativos (Typst Embutido)

| Função | Assinatura | Efeito | Descrição |
|---|---|---|---|
| `pdf-typst` | `pdf-typst(fonte str, dados_json str) (result str str)` | nenhum | Renderiza layout Typst em documento PDF embutido (dados em `/dados.json`). |
| `pdf-typst-files` | `pdf-typst-files(fonte str, dados_json str, arquivos (map str str)) (result str str)` | nenhum | Renderiza documento Typst incluindo assets adicionais (logos, imagens, CSVs). |
| `png-typst` | `png-typst(fonte str, dados_json str) (result str str)` | nenhum | Gera imagem PNG da primeira página para pré-visualização. |
| `pdf-response` | `pdf-response(pdf_bytes str, nome_arquivo str) Response` | nenhum | Cria resposta HTTP `application/pdf` pronta para download/visualização no navegador. |
