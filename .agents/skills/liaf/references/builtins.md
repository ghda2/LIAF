# Builtins da LIAF — todas as funções nativas

As 185 funções embutidas no compilador, por categoria, na sintaxe linear. Não precisam de
`import`. Para as funções da biblioteca padrão (`std/auth`, `std/jwt`...), veja
[stdlib.md](stdlib.md).

Cada seção tem uma tabela com a assinatura e um **programa completo** que usa todas as funções
dela. Os programas não são ilustrativos: o teste `TestSkillExamples`, do repositório da LIAF,
passa cada um pelo `liafc check` e roda os blocos `test` com `liafc test`. Os valores nos
`assert-eq` são, portanto, o que a função devolve de fato.

Convenções das tabelas:

- **`T`** é qualquer tipo; **`(result T str)`** é sucesso `T` ou erro em texto (veja "Erros" em
  [syntax.md](syntax.md)).
- **Efeito** é o que a função exige em `effects(...)` de quem a chama (`—` = pura).
- Argumentos marcados como **tipo** recebem um nome de tipo, não um valor: `make-list(int)`,
  `json-decode(texto, Usuario)`.

## Sumário

1. [Texto e expressões regulares](#1-texto-e-expressões-regulares)
2. [Números, conversões e lógica](#2-números-conversões-e-lógica)
3. [Listas, mapas, option, result e structs](#3-listas-mapas-option-result-e-structs)
4. [JSON, CSV e criptografia](#4-json-csv-e-criptografia)
5. [Sistema, tempo e terminal](#5-sistema-tempo-e-terminal)
6. [Arquivos, documentos, imagens e storage](#6-arquivos-documentos-imagens-e-storage)
7. [Banco de dados](#7-banco-de-dados)
8. [Web: rotas, requisição, resposta e cliente HTTP](#8-web-rotas-requisição-resposta-e-cliente-http)
9. [WebSocket](#9-websocket)
10. [Concorrência](#10-concorrência)
11. [Testes](#11-testes)
12. [Conteúdo e Markdown](#12-conteúdo-e-markdown)

---

## 1. Texto e expressões regulares

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `concat(a, b, ...)` | `str` | — | Junta qualquer quantidade de valores de qualquer tipo. `+` não junta texto |
| `fmt(molde, args...)` | `str` | — | Cada `{}` recebe o próximo argumento; `{{` e `}}` são chaves literais. Não monte JSON com ele (`W_MANUAL_JSON_FMT`) |
| `str-len(s)` | `int` | — | Caracteres Unicode: `str-len("ação")` = 4 |
| `str-byte-len(s)` | `int` | — | Bytes: use para dados binários (PDF, imagem, upload) |
| `str-slice(s, inicio, fim)` | `(result str str)` | — | Por caractere, `fim` exclusivo; índice inválido é `err` |
| `str-get(s, i)` | `(result str str)` | — | O caractere na posição `i` |
| `str-index(s, sub)` | `int` | — | Posição (em caracteres) da primeira ocorrência, ou `-1` |
| `str-contains(s, sub)` | `bool` | — | |
| `str-starts-with(s, prefixo)` | `bool` | — | |
| `str-ends-with(s, sufixo)` | `bool` | — | |
| `str-eq(a, b)` | `bool` | — | O mesmo que `a == b`. Para comparar segredos, use `secure-eq` |
| `str-trim(s)` | `str` | — | Tira espaços das pontas |
| `str-upper(s)` / `str-lower(s)` | `str` | — | Entendem acentos |
| `str-replace(s, velho, novo)` | `str` | — | Troca **todas** as ocorrências |
| `str-split(s, sep)` | `(list str)` | — | |
| `str-join(lista, sep)` | `str` | — | `lista` é `(list str)` |
| `regex-match(padrao, s)` | `(result bool str)` | — | `err` só se o padrão for inválido. Sintaxe RE2 (sem lookahead/backreference) |
| `regex-find(padrao, s)` | `(result (option str) str)` | — | Primeira ocorrência |
| `regex-find-all(padrao, s)` | `(result (list str) str)` | — | Todas as ocorrências |
| `regex-replace(padrao, s, troca)` | `(result str str)` | — | Troca todas; `$1` referencia grupo |

Para validar sem tratar `result`, `std/regex` tem `regex-is-match`, `is-email`, `is-cpf`... (veja
[stdlib.md](stdlib.md)).

```liaf
module texto

test "montar texto"
  assert-eq(concat("a", "b", 1), "ab1", "concat aceita qualquer valor")
  assert-eq(fmt("{} tem {} anos", "Ana", 30), "Ana tem 30 anos", "fmt interpola em ordem")
  assert-eq(fmt("{{x}} = {}", 1), "{x} = 1", "chaves duplas viram literais")
  assert-eq(str-join(list("a", "b", "c"), ", "), "a, b, c", "str-join")
  assert-eq(list-len(str-split("a,b,c", ",")), 3, "str-split")
end

test "medir e fatiar"
  assert-eq(str-len("ação"), 4, "str-len conta caracteres")
  assert-eq(str-byte-len("ação"), 6, "str-byte-len conta bytes")
  assert-eq(unwrap-or(str-slice("liaf-lang", 0, 4), ""), "liaf", "str-slice e por caractere")
  assert-eq(unwrap-or(str-get("liaf", 1), ""), "i", "str-get")
  assert-eq(str-index("banana", "na"), 2, "str-index")
  assert-eq(str-index("banana", "x"), -1, "str-index ausente")
end

test "buscar e transformar"
  assert-true(str-contains("banana", "nan"))
  assert-true(str-starts-with("liaf.exe", "liaf"))
  assert-true(str-ends-with("liaf.exe", ".exe"))
  assert-true(str-eq("a", "a"))
  assert-eq(str-trim("  oi  "), "oi", "str-trim")
  assert-eq(str-upper("ação"), "AÇÃO", "str-upper")
  assert-eq(str-lower("AÇÃO"), "ação", "str-lower")
  assert-eq(str-replace("a-b-c", "-", "+"), "a+b+c", "str-replace troca todas")
end

test "expressoes regulares"
  assert-true(unwrap-or(regex-match("^[0-9]+$", "2026"), false))
  match unwrap-or(regex-find("[0-9]+", "pedido 42 e 7"), none(str))
    case some(n)
      assert-eq(n, "42", "regex-find devolve o primeiro")
    case none
      assert(false, "deveria achar")
  end
  assert-eq(list-len(unwrap-or(regex-find-all("[0-9]+", "1 2 3"), make-list(str))), 3, "regex-find-all")
  assert-eq(unwrap-or(regex-replace("[0-9]", "a1b2", "#"), ""), "a#b#", "regex-replace")
end

fn main() void
end
```

## 2. Números, conversões e lógica

| Função / operador | Retorno | Efeito | Notas |
|---|---|---|---|
| `a + b`, `a - b`, `a * b` | `int` ou `float` | — | Os dois lados do mesmo tipo. Overflow de `int` aborta o programa |
| `a / b` | `(result int str)` com `int`; `float` com `float` | — | Divisão inteira trunca; divisão por zero é `err`. Use `try (a / b)` |
| `mod(a, b)` | `(result int str)` | — | Resto inteiro. Não existe `%` |
| `neg(x)`, `abs(x)` | o tipo de `x` | — | |
| `min(a, b)`, `max(a, b)` | o tipo dos argumentos | — | |
| `pow(base, exp)` | o tipo dos argumentos | — | `int` ou `float` |
| `sqrt(x)`, `floor(x)`, `ceil(x)`, `round(x)` | `float` | — | Recebem `float` |
| `str-from-int(n)`, `str-from-float(x)`, `str-from-bool(b)` | `str` | — | |
| `int-from-str(s)`, `float-from-str(s)`, `bool-from-str(s)` | `(result T str)` | — | Texto inválido é `err` |
| `float-from-int(n)` | `float` | — | |
| `int-from-float(x)` | `int` | — | Trunca em direção a zero |
| `a == b`, `!=`, `<`, `<=`, `>`, `>=` | `bool` | — | Também comparam `str` (ordem lexicográfica) |
| `a && b`, `a \|\| b`, `!a`, `not(a)` | `bool` | — | `&&` e `\|\|` são curto-circuito |

Literais: `1e3` é `float`, `0xff` é `int`. `add`, `sub`, `mul`, `div`, `eq`... são palavras
reservadas da forma antiga: na sintaxe linear use os operadores.

```liaf
module numeros

test "aritmetica inteira"
  assert-eq(7 + 3, 10, "+")
  assert-eq(7 - 3, 4, "-")
  assert-eq(7 * 3, 21, "*")
  assert-eq(unwrap-or(7 / 2, 0), 3, "/ entre int trunca e devolve result")
  assert-eq(unwrap-or(mod(7, 3), 0), 1, "mod devolve result")
  assert-eq(unwrap-or(mod(7, 0), -1), -1, "divisao por zero vira err")
  assert-eq(neg(5), -5, "neg")
  assert-eq(abs(-5), 5, "abs")
  assert-eq(min(3, 8), 3, "min")
  assert-eq(max(3, 8), 8, "max")
  assert-eq(pow(2, 10), 1024, "pow com int")
end

test "aritmetica float"
  assert-eq(7.0 / 2.0, 3.5, "/ entre float devolve float")
  assert-eq(sqrt(16.0), 4.0, "sqrt")
  assert-eq(floor(2.7), 2.0, "floor")
  assert-eq(ceil(2.1), 3.0, "ceil")
  assert-eq(round(2.5), 3.0, "round")
  assert-eq(pow(2.0, 0.5) > 1.41, true, "pow com float")
  assert-eq(1e3, 1000.0, "notacao cientifica e float")
  assert-eq(0xff, 255, "hexadecimal e int")
end

test "conversoes"
  assert-eq(str-from-int(42), "42", "str-from-int")
  assert-eq(str-from-float(2.5), "2.5", "str-from-float")
  assert-eq(str-from-bool(true), "true", "str-from-bool")
  assert-eq(unwrap-or(int-from-str("42"), 0), 42, "int-from-str")
  assert-eq(unwrap-or(int-from-str("abc"), -1), -1, "int-from-str invalido vira err")
  assert-eq(unwrap-or(float-from-str("2.5"), 0.0), 2.5, "float-from-str")
  assert-eq(unwrap-or(bool-from-str("false"), true), false, "bool-from-str")
  assert-eq(float-from-int(3), 3.0, "float-from-int")
  assert-eq(int-from-float(3.9), 3, "int-from-float trunca")
end

test "logica"
  assert-true(true && !false)
  assert-true(false || true)
  assert-false(not(true))
  assert-true(3 <= 3 && 3 >= 3 && 2 < 3 && 3 > 2 && 1 != 2)
end

fn main() void
end
```

## 3. Listas, mapas, option, result e structs

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `list(a, b, ...)` | `(list T)` | — | Lista com os valores dados. Para lista vazia, `make-list` |
| `make-list(T)` | `(list T)` | — | **tipo**: `make-list(Usuario)`. Só aceita nome simples, não `(list str)` |
| `list-len(l)` | `int` | — | |
| `list-push(l, v)` | `void` | — | Acrescenta no fim, **no lugar** |
| `list-get(l, i)` | `(result T str)` | — | Índice fora é `err` |
| `list-set(l, i, v)` | `(result bool str)` | — | Substitui no lugar |
| `list-pop(l)` | `(result T str)` | — | Tira e devolve o último |
| `list-remove(l, i)` | `(result bool str)` | — | Tira pelo índice |
| `list-contains(l, v)` | `bool` | — | |
| `list-sort(l)` | `void` | — | Ordena no lugar; só `int`, `float` ou `str` |
| `make-map(K, V)` | `(map K V)` | — | **tipos**; a chave tem de ser escalar (`E_INVALID_MAP_KEY`) |
| `map-set(m, k, v)` | `void` | — | |
| `map-get(m, k)` | `(result V str)` | — | Chave ausente é `err` |
| `map-has(m, k)` | `bool` | — | |
| `map-delete(m, k)` | `void` | — | |
| `map-len(m)` | `int` | — | |
| `map-keys(m)` | `(list K)` | — | |
| `some(v)` / `none(T)` | `(option T)` | — | `none` recebe o **tipo** |
| `ok(v)` / `err(e)` | `(result T E)` | — | O tipo vem do retorno da função (`E_RESULT_CONTEXT` fora dele) |
| `unwrap-or(r, padrao)` | `T` | — | Desempacota `result` ou `option` |
| `new Tipo(c1, c2, ...)` | `Tipo` | — | Campos na ordem da declaração. Também cria variantes: `Status.Pago` |
| `p.campo` / `field(p, campo)` | o tipo do campo | — | Os dois são o mesmo |

```liaf
module colecoes

struct Ponto
  x int
  y int
end

fn primeiro_par(xs (list int)) (option int)
  for-each x xs
    if unwrap-or(mod(x, 2), 1) == 0
      return some(x)
    end
  end
  none(int)
end

fn validar(idade int) (result int str)
  if idade < 0
    return err("idade negativa")
  end
  ok(idade)
end

test "listas"
  let xs = make-list(int)
  list-push(xs, 3)
  list-push(xs, 1)
  list-push(xs, 2)
  assert-eq(list-len(xs), 3, "list-len")
  assert-eq(unwrap-or(list-get(xs, 0), 0), 3, "list-get devolve result")
  assert-eq(unwrap-or(list-get(xs, 9), -1), -1, "indice fora vira err")
  let _ = list-set(xs, 0, 30)
  assert-true(list-contains(xs, 30))
  list-sort(xs)
  assert-eq(unwrap-or(list-get(xs, 0), 0), 1, "list-sort ordena no lugar")
  assert-eq(unwrap-or(list-pop(xs), 0), 30, "list-pop tira o ultimo")
  assert-true(unwrap-or(list-remove(xs, 0), false))
  assert-eq(list-len(xs), 1, "sobrou um")
  let nomes = list("ana", "bia")
  assert-eq(list-len(nomes), 2, "list literal")
end

test "mapas"
  let idades = make-map(str, int)
  map-set(idades, "ana", 30)
  map-set(idades, "bia", 25)
  assert-eq(map-len(idades), 2, "map-len")
  assert-true(map-has(idades, "ana"))
  assert-eq(unwrap-or(map-get(idades, "ana"), 0), 30, "map-get devolve result")
  map-delete(idades, "ana")
  assert-false(map-has(idades, "ana"))
  assert-eq(list-len(map-keys(idades)), 1, "map-keys")
end

test "option, result e structs"
  assert-eq(unwrap-or(primeiro_par(list(1, 4, 6)), 0), 4, "some")
  assert-eq(unwrap-or(primeiro_par(list(1, 3)), -1), -1, "none")
  assert-eq(unwrap-or(validar(20), 0), 20, "ok")
  assert-eq(unwrap-or(validar(-1), 0), 0, "err")
  let p = new Ponto(1, 2)
  assert-eq(p.x + field(p, y), 3, "p.campo e field(p, campo) sao o mesmo")
end

fn main() void
end
```

## 4. JSON, CSV e criptografia

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `json-encode(v)` | `(result str str)` | — | Struct vira objeto (chaves = nomes dos campos); `(list T)` vira array |
| `json-decode(texto, T)` | `(result T str)` | — | **tipo** no 2º argumento. Só nome simples: para decodificar uma lista, embrulhe-a numa struct |
| `csv-parse(texto)` | `(result (list (list str)) str)` | — | RFC 4180: aspas, vírgulas e quebras dentro de campo |
| `csv-encode(linhas)` | `str` | — | `linhas` é `(list (list str))` |
| `sha256(texto, cod)` | `str` | — | `cod` é o literal `"hex"` ou `"base64url"` (`E_UNKNOWN_ENCODING`) |
| `hmac-sha256(chave, texto, cod)` | `str` | — | Assinatura; confira com `secure-eq` |
| `secure-eq(a, b)` | `bool` | — | Comparação em tempo constante, para tokens e assinaturas |
| `base64-encode(s)` / `base64-decode(s)` | `str` / `(result str str)` | — | Padrão, com `=` |
| `base64url-encode(s)` / `base64url-decode(s)` | `str` / `(result str str)` | — | URL-safe, sem padding |
| `password-hash(senha)` | `str` | `rand` | Argon2id. Grave o hash, nunca a senha |
| `password-verify(senha, hash)` | `bool` | — | |
| `random-token()` | `str` | `rand` | Token aleatório criptográfico (base64url), para sessão e CSRF |

```liaf
module dados

struct Usuario
  id int
  nome str
  tags (list str)
end

test "json"
  let u = new Usuario(1, "Ana", list("admin"))
  let texto = unwrap-or(json-encode(u), "")
  assert-eq(texto, "{\"id\":1,\"nome\":\"Ana\",\"tags\":[\"admin\"]}", "json-encode usa os nomes dos campos")
  match json-decode(texto, Usuario)
    case ok(de_volta)
      assert-eq(de_volta.nome, "Ana", "json-decode recebe o tipo como 2o argumento")
    case err(e)
      assert(false, e)
  end
  assert-false(unwrap-or(json-decode("{nao e json", Usuario), new Usuario(0, "", make-list(str))).id == 1)
end

test "csv"
  match csv-parse("nome,idade\nAna,30\n")
    case ok(linhas)
      assert-eq(list-len(linhas), 2, "csv-parse devolve linhas de colunas")
      assert-true(str-starts-with(csv-encode(linhas), "nome,idade"))
    case err(e)
      assert(false, e)
  end
end

test "criptografia"
  assert-eq(sha256("abc", "hex"), "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", "sha256 em hex")
  let assinatura = hmac-sha256("chave", "mensagem", "base64url")
  assert-true(secure-eq(assinatura, hmac-sha256("chave", "mensagem", "base64url")))
  assert-eq(base64-encode("oi"), "b2k=", "base64-encode")
  assert-eq(unwrap-or(base64-decode("b2k="), ""), "oi", "base64-decode")
  assert-eq(base64url-encode("oi?"), "b2k_", "base64url-encode sem padding")
  assert-eq(unwrap-or(base64url-decode("b2k_"), ""), "oi?", "base64url-decode")
  let hash = password-hash("senha-forte-123")
  assert-true(password-verify("senha-forte-123", hash))
  assert-false(password-verify("errada", hash))
  assert-eq(str-len(random-token()) > 20, true, "random-token")
end

fn main() void
end
```

## 5. Sistema, tempo e terminal

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `println(a, ...)` / `print(a, ...)` | `void` | `io` | Separa por espaço; `print` não quebra a linha |
| `log-info(a, ...)` / `log-error(a, ...)` | `void` | — | Prefixos `[INFO]` e `[ERROR]` na saída padrão |
| `read-line()` | `(result str str)` | `io` | Uma linha da entrada padrão; fim da entrada é `err` |
| `args()` | `(list str)` | `io` | Argumentos do programa, sem o nome do executável |
| `cli-args()` | `(list str)` | — | O mesmo, sem efeito. Para flags, `std/cli` |
| `env-get(nome)` | `(result str str)` | `env` | Variável ausente é `err` |
| `exit(codigo)` / `exit-process(codigo)` | `void` | `io` | Encerra na hora (os dois são sinônimos) |
| `os-exec(cmd, args)` | `(result str str)` | `io` | Roda um programa (sem shell) e devolve a saída; `args` é `(list str)` |
| `os-exec-status(cmd, args)` | `int` | `io` | Só o código de saída |
| `rand-int(n)` | `int` | `rand` | De `0` a `n - 1`. Não use para segredos: veja `random-token` |
| `sleep-ms(ms)` | `void` | `clock` | |
| `now-ms()` / `time-now-ms()` | `int` | `clock` | Milissegundos desde 1970 |
| `time-now-unix()` | `int` | `clock` | Segundos desde 1970 (UTC) |
| `time-now()` | `str` | `clock` | Agora em ISO 8601 |
| `time-parse-iso(s)` | `(result int str)` | — | `"2026-10-17T14:30:05Z"` → segundos |
| `time-format-iso(ts)` | `str` | — | Segundos → ISO 8601 (UTC) |
| `time-format(ts, layout)` | `str` | — | Layout do Go: `"02/01/2006 15:04"` (dia/mês/ano hora:min) |
| `time-add-seconds(ts, s)` | `int` | — | |
| `time-diff-seconds(a, b)` | `int` | — | `a - b` em segundos |
| `time-year`, `time-month`, `time-day`, `time-hour`, `time-minute`, `time-second(ts)` | `int` | — | Partes da data, em UTC |
| `time-weekday(ts)` | `int` | — | `0` = domingo ... `6` = sábado |

```liaf
module sistema

test "tempo"
  let ts = unwrap-or(time-parse-iso("2026-10-17T14:30:05Z"), 0)
  assert-eq(time-year(ts), 2026, "time-year")
  assert-eq(time-month(ts), 10, "time-month")
  assert-eq(time-day(ts), 17, "time-day")
  assert-eq(time-hour(ts), 14, "time-hour")
  assert-eq(time-minute(ts), 30, "time-minute")
  assert-eq(time-second(ts), 5, "time-second")
  assert-eq(time-weekday(ts), 6, "time-weekday: 0 = domingo, 6 = sabado")
  assert-eq(time-format-iso(ts), "2026-10-17T14:30:05Z", "time-format-iso")
  assert-eq(time-format(ts, "02/01/2006 15:04"), "17/10/2026 14:30", "time-format usa o layout do Go")
  let depois = time-add-seconds(ts, 3600)
  assert-eq(time-diff-seconds(depois, ts), 3600, "time-diff-seconds")
  assert-true(time-now-unix() > ts - 999999999)
  assert-true(time-now-ms() > 0)
  assert-true(now-ms() > 0)
  assert-true(str-len(time-now()) > 0)
  sleep-ms(1)
end

test "processo e ambiente"
  assert-eq(list-len(args()) >= 0, true, "args")
  assert-eq(list-len(cli-args()) >= 0, true, "cli-args")
  assert-true(rand-int(10) < 10)
  let _ = env-get("PATH")
  assert-eq(os-exec-status("go", list("version")), 0, "os-exec-status devolve o codigo de saida")
  assert-true(str-starts-with(unwrap-or(os-exec("go", list("version")), ""), "go version"))
end

fn main() void effects(io)
  print("sem quebra de linha ")
  println("com quebra de linha")
  log-info("evento", "registrado")
  log-error("falha", "registrada")
  match read-line()
    case ok(linha)
      println(fmt("leu: {}", linha))
    case err(_)
      println("sem entrada")
  end
  if list-len(args()) > 0
    exit-process(2) // sinonimo de exit
  end
  exit(0)
end
```

## 6. Arquivos, documentos, imagens e storage

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `fs-read-file(caminho)` | `(result str str)` | `fs` | Conteúdo inteiro (texto ou binário) |
| `fs-write-file(caminho, dados)` | `(result void str)` | `fs` | Cria ou sobrescreve |
| `fs-write-atomic(caminho, dados)` | `(result void str)` | `fs` | Grava num temporário e renomeia: nunca deixa arquivo pela metade |
| `fs-exists(caminho)` / `fs-is-dir(caminho)` | `bool` | `fs` | |
| `fs-file-size(caminho)` | `(result int str)` | `fs` | Em bytes |
| `fs-list-dir(caminho)` | `(result (list str) str)` | `fs` | Nomes das entradas |
| `fs-mkdir-all(caminho)` | `(result bool str)` | `fs` | Cria a árvore inteira |
| `fs-rename(de, para)` | `(result void str)` | `fs` | |
| `fs-remove(caminho)` | `(result void str)` | `fs` | Um arquivo ou pasta vazia |
| `fs-remove-all(caminho)` | `(result bool str)` | `fs` | Recursivo |
| `pdf-typst(layout, json)` | `(result str str)` | — | Bytes do PDF. O documento lê os dados em `json("/dados.json")`. Detalhes na skill `liaf-pdf` |
| `pdf-typst-files(layout, json, arquivos)` | `(result str str)` | — | `arquivos` é `(map str str)` nome → bytes, lidos no documento como `"/nome"` |
| `png-typst(layout, json)` / `png-typst-files(...)` | `(result str str)` | — | Prévia PNG da 1ª página |
| `pdf-response(pdf, nome)` | `Response` | — | `application/pdf` aberto no navegador (`inline`), com `nome` ao salvar |
| `image-dimensions(img)` | `(result str str)` | — | `"LARGURAxALTURA"` |
| `image-resize(img, max_l, max_a)` | `(result str str)` | — | Mantém a proporção |
| `image-to-webp(img)` | `(result str str)` | — | WebP sem perdas |
| `image-to-webp-quality(img, nivel)` | `(result str str)` | — | Esforço de compressão de 0 a 9 |
| `image-process(img, max_l, max_a, nivel)` | `(result str str)` | — | Redimensiona e converte |
| `storage-dir-init(dir)` | `(result bool str)` | `fs` | Cria a pasta de armazenamento |
| `storage-save-image(dir, nome, img)` | `(result str str)` | `fs` | Converte para WebP e devolve o nome final: o original com `.webp`. **Um nome repetido sobrescreve**: gere nomes únicos (ex.: `random-token()`) |
| `storage-save-image-opt(dir, nome, img, max_l, max_a, max_bytes, nivel)` | `(result str str)` | `fs` | Com limites; `0` = sem limite |
| `storage-file-exists(dir, nome)` | `bool` | `fs` | |
| `storage-file-get(dir, nome)` | `(result str str)` | `fs` | |
| `storage-file-delete(dir, nome)` | `(result bool str)` | `fs` | |

Os nomes de `storage-*` são saneados contra `../` (path traversal). Para upload direto de uma
requisição, `std/storage` tem `storage-upload-image`.

```liaf
module arquivos

test "arquivos e pastas"
  let _ = fs-remove-all("tmp_liaf")
  assert-true(unwrap-or(fs-mkdir-all("tmp_liaf/sub"), false))
  assert-true(fs-is-dir("tmp_liaf/sub"))
  let _ = fs-write-file("tmp_liaf/a.txt", "ola")
  let _ = fs-write-atomic("tmp_liaf/b.json", "{}")
  assert-true(fs-exists("tmp_liaf/a.txt"))
  assert-eq(unwrap-or(fs-read-file("tmp_liaf/a.txt"), ""), "ola", "fs-read-file")
  assert-eq(unwrap-or(fs-file-size("tmp_liaf/a.txt"), 0), 3, "fs-file-size em bytes")
  let _ = fs-rename("tmp_liaf/a.txt", "tmp_liaf/c.txt")
  assert-false(fs-exists("tmp_liaf/a.txt"))
  assert-eq(list-len(unwrap-or(fs-list-dir("tmp_liaf"), make-list(str))), 3, "fs-list-dir: b.json, c.txt e sub")
  let _ = fs-remove("tmp_liaf/c.txt")
  assert-false(fs-exists("tmp_liaf/c.txt"))
  assert-true(unwrap-or(fs-remove-all("tmp_liaf"), false))
end

test "documentos e imagens"
  let layout = "#set page(width: 4cm, height: 2cm)\n#let d = json(\"/dados.json\")\n= #d.titulo"
  let pdf = unwrap-or(pdf-typst(layout, "{\"titulo\":\"Oi\"}"), "")
  assert-true(str-starts-with(pdf, "%PDF"))
  let png = unwrap-or(png-typst(layout, "{\"titulo\":\"Oi\"}"), "")
  assert-eq(unwrap-or(image-dimensions(png), ""), "227x113", "image-dimensions devolve LARGURAxALTURA")
  let arquivos = make-map(str, str)
  map-set(arquivos, "logo.png", png)
  let com_imagem = "#set page(width: 4cm, height: 2cm)\n#image(\"/logo.png\")"
  assert-true(str-starts-with(unwrap-or(pdf-typst-files(com_imagem, "{}", arquivos), ""), "%PDF"))
  assert-true(str-byte-len(unwrap-or(png-typst-files(com_imagem, "{}", arquivos), "")) > 0)
  let webp = unwrap-or(image-to-webp(png), "")
  assert-true(str-starts-with(webp, "RIFF"))
  assert-true(str-byte-len(unwrap-or(image-to-webp-quality(png, 9), "")) > 0)
  assert-eq(unwrap-or(image-dimensions(unwrap-or(image-resize(png, 50, 50), "")), ""), "50x24", "image-resize mantem a proporcao")
  assert-true(str-byte-len(unwrap-or(image-process(png, 50, 50, 6), "")) > 0)
  let _ = pdf-response(pdf, "doc.pdf")
end

test "storage"
  let _ = storage-dir-init("tmp_storage")
  let layout = "#set page(width: 2cm, height: 2cm)\nx"
  let png = unwrap-or(png-typst(layout, "{}"), "")
  let nome = unwrap-or(storage-save-image("tmp_storage", "foto.png", png), "")
  assert-true(str-ends-with(nome, ".webp"))
  assert-true(storage-file-exists("tmp_storage", nome))
  assert-true(str-starts-with(unwrap-or(storage-file-get("tmp_storage", nome), ""), "RIFF"))
  let nome2 = unwrap-or(storage-save-image-opt("tmp_storage", "foto2.png", png, 20, 20, 1000000, 6), "")
  assert-true(storage-file-exists("tmp_storage", nome2))
  assert-true(unwrap-or(storage-file-delete("tmp_storage", nome), false))
  assert-false(storage-file-exists("tmp_storage", nome))
  let _ = fs-remove-all("tmp_storage")
end

fn main() void
end
```

## 7. Banco de dados

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `db-connect(driver, endereco)` | `(result DBConnection str)` | `db` | `driver` é o literal `"sqlite"`, `"postgres"`, `"mysql"` ou `"redis"` |
| `db-exec(db, sql, args...)` | `(result int str)` | `db` | `INSERT`/`UPDATE`/`DELETE`/DDL; devolve as linhas afetadas |
| `db-query(db, sql, Tipo, args...)` | `(result (list Tipo) str)` | `db` | **tipo** de linha é uma struct; cada coluna casa com o campo de mesmo nome |
| `db-close(db)` | `(result void str)` | `db` | |
| `redis-set(db, chave, valor, ttl_segundos)` | `(result void str)` | `db` | `ttl` `0` = sem expiração |
| `redis-get(db, chave)` | `(result str str)` | `db` | Chave ausente é `err` |
| `db-transaction db ... end` | (bloco) | `db` | Confirma no fim; desfaz se um `try` lá dentro falhar |

Regras do checker: o SQL é **sempre um literal** (`E_SQL_INTERPOLATION`), os valores vão como
argumentos e o número de marcadores bate com o de argumentos (`E_SQL_PARAM_COUNT`). Marcadores:
`?` no SQLite e no MySQL, `$1, $2...` no Postgres.

```liaf
module banco

struct Conta
  id int
  dono str
  saldo int
end

fn preparar(db DBConnection) (result bool str) effects(db)
  try db-exec(db, "DROP TABLE IF EXISTS contas")
  try db-exec(db, "CREATE TABLE contas (id INTEGER PRIMARY KEY, dono TEXT, saldo INTEGER)")
  try db-exec(db, "INSERT INTO contas (dono, saldo) VALUES (?, ?)", "ana", 100)
  try db-exec(db, "INSERT INTO contas (dono, saldo) VALUES (?, ?)", "bia", 50)
  ok(true)
end

fn transferir(db DBConnection, de str, para str, valor int) (result bool str) effects(db)
  db-transaction db
    try db-exec(db, "UPDATE contas SET saldo = saldo - ? WHERE dono = ?", valor, de)
    try db-exec(db, "UPDATE contas SET saldo = saldo + ? WHERE dono = ?", valor, para)
  end
  ok(true)
end

fn saldo(db DBConnection, dono str) (result int str) effects(db)
  let contas = try db-query(db, "SELECT id, dono, saldo FROM contas WHERE dono = ?", Conta, dono)
  let conta = try list-get(contas, 0)
  ok(conta.saldo)
end

// Redis usa o mesmo DBConnection; precisa de um servidor rodando.
fn cache(endereco str) (result str str) effects(db)
  let r = try db-connect("redis", endereco)
  try redis-set(r, "sessao:1", "ana", 3600)
  let valor = try redis-get(r, "sessao:1")
  try db-close(r)
  ok(valor)
end

// try nao vale dentro de test (nao ha on-err): o cenario fica numa fn que devolve result.
fn cenario() (result str str) effects(db)
  let db = try db-connect("sqlite", "banco_exemplo.db")
  try preparar(db)
  try transferir(db, "ana", "bia", 30)
  let a = try saldo(db, "ana")
  let b = try saldo(db, "bia")
  try db-close(db)
  ok(fmt("{} {}", a, b))
end

test "sqlite com transacao"
  assert-eq(unwrap-or(cenario(), "falhou"), "70 80", "a transferencia move o saldo")
  let _ = fs-remove("banco_exemplo.db")
end

fn main() void effects(db, io)
  match cache("localhost:6379")
    case ok(v)
      println(v)
    case err(e)
      println(fmt("redis indisponivel: {}", e))
  end
end
```

## 8. Web: rotas, requisição, resposta e cliente HTTP

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `serve-hybrid(pasta, porta)` | `(result bool str)` | `net`, `fs` | Sobe rotas + arquivos estáticos de `pasta`. Bloqueia até o servidor parar |
| `serve-site(pasta, dominio, porta, auto_tls)` | `void` | `net`, `fs` | Site estático; com `true`, certificado Let's Encrypt (portas 80/443) |
| `request-method(req)`, `request-path(req)` | `str` | — | |
| `request-body(req)` | `str` | — | Corpo cru |
| `request-header(req, nome)` | `(result str str)` | — | Ausente é `err` |
| `request-query(req, nome)` | `(result str str)` | — | Parâmetro da query string |
| `request-cookie(req, nome)` | `(result str str)` | — | |
| `request-form(req, campo)` | `(result str str)` | — | Formulário urlencoded ou multipart |
| `request-file-data(req, campo)` | `(result str str)` | — | Bytes do arquivo enviado |
| `request-file-name(req, campo)` | `(result str str)` | — | Nome original do arquivo |
| `request-set-header(req, nome, valor)` | `Request` | — | Cópia da requisição com o header trocado; é como um middleware passa contexto |
| `req-client-ip(req)` | `str` | — | Lê `X-Forwarded-For`/`X-Real-IP`. **Sem proxy na frente, devolve sempre `127.0.0.1`**, e o cliente pode forjar esses headers: só confie atrás de um proxy que os reescreve |
| `json-response(status, json)` | `Response` | — | `json` é `str`: serialize com `json-encode` |
| `json-error(status, mensagem)` | `Response` | — | `{"error": mensagem}` com o escape feito |
| `html-response(status, html)` | `Response` | — | `text/html` |
| `html-escape(s)` | `str` | — | Escapa `< > & " '`: use em todo dado do usuário que vai para HTML |
| `raw-response(status, tipo, corpo)` | `Response` | — | Qualquer `Content-Type`, corpo binário |
| `file-response(caminho, tipo)` | `(result Response str)` | `fs` | Serve um arquivo do disco |
| `response-set-header(res, nome, valor)` | `Response` | — | Devolve **nova** resposta; substitui |
| `response-add-header(res, nome, valor)` | `Response` | — | Acumula (use para `Set-Cookie`) |
| `rate-limit(chave, max, janela_s)` | `bool` | — | `true` se ainda está no limite; conta por chave em memória |
| `http-fetch(metodo, url, headers, corpo)` | `(result HttpReply str)` | `net` | `headers` é `(list str)` no formato `"Nome: valor"`. `err` só se não houve resposta |
| `reply-status(r)` | `int` | — | Um 404 ou 500 chega como `ok`: confira o status |
| `reply-body(r)` | `str` | — | |
| `reply-header(r, nome)` | `(result str str)` | — | |
| `http-get`, `http-post`, `http-put`, `http-delete(caminho, handler)` | `void` | `net` | **Forma antiga** de registrar rota, com `handler(req Request) Response`. Não é cliente HTTP. Prefira `route` |

As declarações `route`, `middleware` e `with(...)` estão em [syntax.md](syntax.md).

```liaf
module web

struct Contato
  nome str
  email str
end

// Respostas JSON sempre por struct + json-encode: montar JSON com fmt deixa o
// cliente injetar campos (o checker recusa com W_MANUAL_JSON_FMT).
struct Eco
  id int
  metodo str
  caminho str
  bytes int
  agente str
  pagina str
  sessao str
  ip str
end

struct Arquivo
  nome str
  bytes int
end

struct Usuario
  usuario str
end

// Leitura da requisicao: tudo que vem do cliente.
route POST "/eco/{id}" (id int, req Request) Response
  on-err motivo
    json-error(500, motivo)
  end
  let metodo = request-method(req)
  let caminho = request-path(req)
  let corpo = request-body(req)
  let agente = unwrap-or(request-header(req, "User-Agent"), "desconhecido")
  let pagina = unwrap-or(request-query(req, "pagina"), "1")
  let sessao = unwrap-or(request-cookie(req, "sid"), "")
  let ip = req-client-ip(req)
  let eco = new Eco(id, metodo, caminho, str-byte-len(corpo), agente, pagina, sessao, ip)
  json-response(200, try json-encode(eco))
end

// Middleware: barra sem token e entrega o usuario a rota num header.
middleware exigir_token(req Request) (result Request Response)
  let token = unwrap-or(request-header(req, "Authorization"), "")
  if token != "Bearer segredo"
    return err(json-error(401, "token obrigatorio"))
  end
  ok(request-set-header(req, "X-Usuario", "ana"))
end

route GET "/painel" with(exigir_token) (req Request) Response
  on-err motivo
    json-error(500, motivo)
  end
  let usuario = unwrap-or(request-header(req, "X-Usuario"), "")
  json-response(200, try json-encode(new Usuario(usuario)))
end

// Formulario HTML (application/x-www-form-urlencoded ou multipart).
route POST "/contato" (req Request) Response
  let nome = unwrap-or(request-form(req, "nome"), "")
  if nome == ""
    return json-error(400, "nome obrigatorio")
  end
  html-response(200, fmt("<p>Obrigado, {}!</p>", html-escape(nome)))
end

// Upload: bytes e nome do arquivo enviado no campo "arquivo".
route POST "/upload" (req Request) Response
  on-err motivo
    json-error(400, motivo)
  end
  let dados = try request-file-data(req, "arquivo")
  let nome = try request-file-name(req, "arquivo")
  json-response(201, try json-encode(new Arquivo(nome, str-byte-len(dados))))
end

// Corpo JSON tipado: struct como parametro.
route POST "/contatos" (c Contato) Response
  json-response(201, unwrap-or(json-encode(c), "{}"))
end

// Respostas com headers, binario e arquivo do disco.
route GET "/csv" () Response
  let res = raw-response(200, "text/csv; charset=utf-8", "nome,email\nana,ana@x.com\n")
  let res2 = response-set-header(res, "Content-Disposition", "attachment; filename=contatos.csv")
  response-add-header(res2, "Set-Cookie", "visto=1; Path=/")
end

route GET "/logo" () Response effects(fs)
  on-err motivo
    json-error(404, motivo)
  end
  try file-response("public/logo.png", "image/png")
end

// Limite de taxa por IP: 60 requisicoes por 60 segundos.
route GET "/api/busca" (req Request) Response
  if !rate-limit(req-client-ip(req), 60, 60)
    return json-error(429, "muitas requisicoes")
  end
  json-response(200, "[]")
end

// Cliente HTTP: chamar outra API.
fn cotacao() (result str str) effects(net)
  let r = try http-fetch("GET", "https://api.exemplo.com/cotacao", list("Accept: application/json"), "")
  if reply-status(r) != 200
    return err(fmt("status {}", reply-status(r)))
  end
  let tipo = unwrap-or(reply-header(r, "Content-Type"), "")
  ok(fmt("{} ({})", reply-body(r), tipo))
end

// Forma antiga (v0.2) de registrar rotas: prefira `route`.
fn ping(req Request) Response
  json-response(200, "{\"pong\":true}")
end

test "html-escape e rate-limit"
  assert-eq(html-escape("<b>&\"</b>"), "&lt;b&gt;&amp;&#34;&lt;/b&gt;", "html-escape")
  assert-true(rate-limit("teste", 2, 60))
  assert-true(rate-limit("teste", 2, 60))
  assert-false(rate-limit("teste", 2, 60))
end

fn main() void effects(fs, io, net)
  http-get("/ping", ping)
  http-post("/ping", ping)
  http-put("/ping", ping)
  http-delete("/ping", ping)
  println(unwrap-or(cotacao(), "sem cotacao"))
  if list-len(args()) > 5
    // Site estatico com auto-TLS (Let's Encrypt) nas portas 80/443.
    serve-site("./public", "exemplo.com", "443", true)
  end
  unwrap-or(serve-hybrid("./public", "8080"), false)
end
```

## 9. WebSocket

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `ws-join(conn, topico)` | `void` | `net` | Inscreve a conexão no tópico |
| `ws-leave(conn, topico)` | `void` | `net` | Fechar a conexão já desinscreve de tudo |
| `ws-send(conn, texto)` | `(result void str)` | `net` | Só para esta conexão |
| `ws-send-json(conn, valor)` | `(result void str)` | `net` | Serializa `valor` (struct, lista...) e envia |
| `ws-broadcast(topico, texto)` | `(result int str)` | `net` | Para todos do tópico; devolve quantos receberam |
| `ws-topic-size(topico)` | `int` | `net` | Conexões inscritas |
| `ws-close(conn, codigo, motivo)` | `(result void str)` | `net` | Códigos 4000–4999 são da aplicação (ex.: `4401` não autorizado) |
| `ws-enable-redis(endereco)` | `(result bool str)` | `net` | Broadcast entre várias instâncias via Redis pub/sub |
| `ws-disable-redis()` | `bool` | `net` | Volta ao broadcast local |

A declaração `ws-route` (blocos `on-open`, `on-message`, `on-close`) está em
[syntax.md](syntax.md). O navegador não envia `Authorization` no WebSocket: passe o token na
query e confira em `on-open`.

```liaf
module ws

struct Aviso
  sala str
  online int
end

ws-route "/ws/{sala}" (sala str, req Request, conn WSConn) effects(io, net)
  on-err motivo
    println(fmt("ws: {}", motivo))
  end
  on-open
    let token = unwrap-or(request-query(req, "token"), "")
    if token == ""
      try ws-close(conn, 4401, "token obrigatorio")
      return
    end
    ws-join(conn, sala)
    ws-join(conn, "todos")
    try ws-send(conn, "bem-vindo")
    try ws-send-json(conn, new Aviso(sala, ws-topic-size(sala)))
  end
  on-message texto
    if texto == "sair-de-todos"
      ws-leave(conn, "todos")
      return
    end
    let entregues = try ws-broadcast(sala, texto)
    println(fmt("{} conexao(oes) receberam", entregues))
  end
  on-close
    println(fmt("saiu de {}", sala))
  end
end

fn main() void effects(fs, io, net)
  // Varias instancias atras de um balanceador: o broadcast passa pelo Redis.
  match ws-enable-redis("localhost:6379")
    case ok(_)
      println("pub/sub via redis")
    case err(e)
      println(fmt("seguindo sem redis: {}", e))
      let _ = ws-disable-redis()
  end
  unwrap-or(serve-hybrid("./public", "8080"), false)
end
```

## 10. Concorrência

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `make-chan(T)` | `(chan T)` | — | Canal; **tipo** do elemento |

`spawn`, `send` e `recv` ainda **não existem na sintaxe linear**. A parte concorrente se declara
em S-expression, que pode conviver no mesmo arquivo linear, como abaixo. `spawn` exige o efeito
`spawn`.

```liaf
module canais

fn calcular(n int) int
  n * n
end

// spawn, send e recv ainda nao existem na sintaxe linear: declare a parte
// concorrente em S-expression, que pode conviver no mesmo arquivo.
(fn trabalhador ((ch (chan int)) (n int)) void (effects)
  (send ch (calcular n)))

(fn somar_em_paralelo () int (effects spawn)
  (let ch (chan int) (make-chan int))
  (spawn (trabalhador ch 3))
  (spawn (trabalhador ch 4))
  (add (recv ch) (recv ch)))

test "canais"
  assert-eq(somar_em_paralelo(), 25, "9 + 16")
end

fn main() void
end
```

## 11. Testes

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `assert(cond, mensagem)` | `void` | — | |
| `assert-true(cond)` / `assert-false(cond)` | `void` | — | |
| `assert-eq(obtido, esperado, mensagem)` | `void` | — | Compara estruturas inteiras; a falha mostra os dois valores |

Os blocos `test "nome" ... end` rodam com `liafc test arquivo.liaf` (ou `--json`). Dentro deles
os efeitos estão liberados, mas `try` não é aceito (não há `on-err`): ponha o cenário numa `fn`
que devolve `result` e confira com `unwrap-or`, como em [Banco de dados](#7-banco-de-dados).
Todas as seções acima usam asserções; veja qualquer uma delas como exemplo.

## 12. Conteúdo e Markdown

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `content-parse(raw)` | `(result str str)` | — | Faz o parse de frontmatter YAML e devolve JSON com campos estruturados |
| `content-slugify(titulo)` | `str` | — | Converte título para slug URL-friendly |
| `markdown-to-html(md)` | `str` | — | Converte markdown em HTML higienizado contra XSS |

```liaf
module exemplo_conteudo

test "builtins de conteudo"
  let slug = content-slugify("Meu Artigo Incrível")
  assert-eq(slug, "meu-artigo-incrivel", "slug limpo")

  let html = markdown-to-html("# Titulo\ntexto")
  assert-true(str-contains(html, "<h1>Titulo</h1>"))

  match content-parse("---\ntitle: Post\n---\nCorpo")
    case ok(j)
      assert-true(str-contains(j, "Post"))
    case err(e)
      assert(false, e)
  end
end

fn main() void
end
```
