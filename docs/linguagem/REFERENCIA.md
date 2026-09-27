# LIAF — Referência completa

Tudo o que a linguagem aceita hoje, organizado por assunto. Para **como** escrever bem (fluxo de
trabalho, padrões, armadilhas), leia o [AI_GUIDE.md](AI_GUIDE.md). Para o **porquê** de cada decisão
de design, a [SPEC.md](SPEC.md).

Conferido contra o compilador em 26/09/2026. Onde o comportamento surpreende, a referência diz
explicitamente: essas notas vieram de programas de teste, não da intenção do design.

**Convenções desta página.** `T`, `K`, `V`, `E` são tipos quaisquer. "Falível" quer dizer que o
retorno é `(result T str)` e precisa ser tratado (§5). A coluna *Efeitos* diz o que a função que
chama precisa declarar (§4).

---

## Sumário

1. [Estrutura de um programa](#1-estrutura-de-um-programa)
2. [Declarações](#2-declarações)
3. [Instruções e expressões](#3-instruções-e-expressões)
4. [Efeitos](#4-efeitos)
5. [Erros: `result`, `option`, `try`, `match`](#5-erros-result-option-try-match)
6. [Tipos](#6-tipos)
7. [Operadores](#7-operadores)
8. [Builtins](#8-builtins) — todos os 128, por categoria
9. [Biblioteca padrão (`std/`)](#9-biblioteca-padrão-std)
10. [Servidor web](#10-servidor-web)
11. [Diagnósticos](#11-diagnósticos) — todos os códigos
12. [CLI `liafc`](#12-cli-liafc)
13. [Variáveis de ambiente e limites](#13-variáveis-de-ambiente-e-limites)
14. [O que não existe](#14-o-que-não-existe)

---

## 1. Estrutura de um programa

Todo arquivo é **um** `(module ...)`. Sem ele: `E_EXPECTED_MODULE`. Um programa executável tem
`(fn main () void ...)`, sem parâmetros (`E_MAIN_SIGNATURE`, `E_MISSING_MAIN`).

```liaf
; comentário vai do ponto e vírgula até o fim da linha
(module ola
  (fn main () void (effects io)
    (println "Olá, LIAF")))
```

### 1.1 Léxico

| Elemento | Forma | Observação |
|---|---|---|
| Comentário | `; ...` ou `;; ...` | Até o fim da linha. O `liafc fmt` ainda **apaga** comentários |
| Inteiro | `42`, `-7`, `0xff` | `int` de 64 bits. Fora do intervalo: `E_INVALID_NUMBER` |
| Real | `3.14`, `1e3`, `2.5e-3` | `float` de 64 bits. Notação científica é sempre `float` |
| Texto | `"a\n\t\"b\"\\"` | UTF-8. Escapes: `\n`, `\t`, `\"`, `\\` |
| Booleano | `true`, `false` | |
| Identificador | `total`, `created-at`, `user_id`, `userId` | Letras, dígitos, `-` e `_`. A convenção é `kebab-case` |
| Curinga | `_` | Em `match` e `let`, descarta o valor |

Nomes de campo podem coincidir com palavras reservadas (`sub`, `return`, `add`...), exceto `true` e
`false`. `(field claims sub)` lê o campo; `(sub a b)` continua sendo subtração.

### 1.2 Nível superior

Dentro de `(module nome ...)` só podem aparecer: `import`, `struct`, `fn`, `route`, `ws-route`
(`E_INVALID_TOPLEVEL`). A ordem entre eles não importa: uma função pode chamar outra declarada
depois.

---

## 2. Declarações

### 2.1 `import`

```liaf
(module app
  (import "std/auth")        ; biblioteca padrão, embutida no liafc
  (import "./modelos.liaf")  ; arquivo, relativo a quem importa
  (import "./rotas")         ; pasta: carrega todos os .liaf dela
  (fn main () void (effects io) (println "ok")))
```

- Tudo o que um import declara entra num **espaço de nomes único**. Não há `modulo.funcao`.
- Nome declarado em dois arquivos: `E_DUPLICATE_DECL`. Se um deles é da std, o nome é dela.
- Rota repetida (mesmo método e caminho), mesmo em arquivos diferentes: `E_DUPLICATE_ROUTE`.
- Ciclo de imports: `E_CIRCULAR_IMPORT`. Arquivo ausente: `E_IMPORT_NOT_FOUND` (para `std/`, a
  mensagem lista os módulos que existem). Pasta sem `.liaf`: `E_NO_LIAF_FILES`.
- Um módulo da std só importa outros da std (`E_STD_RELATIVE_IMPORT`).

### 2.2 `struct`

```liaf
(module m
  (struct Pedido (id int) (cliente str) (itens (list str)) (pago bool))
  (fn main () void (effects io)
    (let p Pedido (new Pedido 1 "Ana" (list "pizza") false))
    (println (field p cliente))))
```

- `(new Tipo v1 v2 ...)`: um valor por campo, **na ordem da declaração**.
- `(field valor campo)` lê um campo. Não existe `valor.campo` nem alteração de campo: para "mudar",
  construa outra com `new`.
- A forma antiga `(struct P (fields (id int)))` ainda é aceita.

### 2.3 `fn`

Forma canônica (compacta):

```
(fn nome ((p1 T1) (p2 T2)) TipoRetorno [(effects e1 e2)] [(on-err var ...)]
  instrução...
  expressão-final)
```

```liaf
(module m
  (fn dobro ((n int)) int
    (mul n 2))

  (fn saudar ((nome str)) void (effects io)
    (println (fmt "Olá, {}" nome)))

  (fn main () void (effects io)
    (saudar "Ana")
    (println (dobro 21))))
```

- Sem parâmetros: `()`. Sem `(effects ...)`: a função é pura.
- **Retorno implícito:** a última forma do corpo é o valor devolvido. Precisa ser uma forma entre
  parênteses: um nome ou literal solto no fim (`x`, `1`, `"a"`) é erro de sintaxe. Nesses casos use
  `(return x)`.
- `(return valor)` sai antes; numa função `void`, `(return)`.
- Função não-`void` em que algum caminho não devolve nada: `E_MISSING_RETURN`.
- A forma longa `(fn f (params (n int)) (returns int) (effects) (body (return (mul n 2))))` continua
  aceita. O `liafc fmt` reescreve para a compacta.

### 2.4 `route` — rota HTTP

```
(route MÉTODO "/caminho/{param}" ((param T) (entrada Struct) (req Request)) Response [(effects ...)] [(on-err var ...)]
  instrução...)
```

```liaf
(module api
  (struct NovaTarefa (titulo str))

  (route GET "/tarefas/{id}" ((id int)) Response
    (json-response 200 (fmt "{{\"id\":{}}}" id)))

  (route POST "/tarefas" ((entrada NovaTarefa)) Response
    (json-response 201 (fmt "{{\"titulo\":\"{}\"}}" (field entrada titulo))))

  (fn main () void (effects fs io net)
    (match (serve-hybrid "./public" "8080")
      (ok parado (println "fim"))
      (err e (println e)))))
```

- Métodos: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `OPTIONS`, `HEAD`.
- Cada `{nome}` do caminho exige um parâmetro homônimo `int` ou `str`, que chega convertido
  (`E_ROUTE_PARAM`). Um `{id}` que não é número responde `400` sozinho.
- Um parâmetro **struct** recebe o corpo JSON desserializado; corpo inválido responde `400`.
- Um parâmetro **`Request`** dá acesso a headers, query e corpo cru (§10.2).
- A rota se registra sozinha. Ela só atende depois que `main` chama `serve-hybrid`.

### 2.5 `ws-route` — WebSocket

```
(ws-route "/caminho/{param}" (params (param T) [(req Request)] (conn WSConn)) (effects net ...) [(on-err var ...)]
  [(on-open instrução...)]
  (on-message var instrução...)
  [(on-close instrução...)])
```

```liaf
(module chat
  (ws-route "/ws/{sala}" (params (sala str) (conn WSConn)) (effects io net)
    (on-err erro (println erro))
    (on-open (ws-join conn sala))
    (on-message texto (try (ws-broadcast sala texto)))
    (on-close (println "saiu")))

  (fn main () void (effects fs io net)
    (match (serve-hybrid "./public" "8080")
      (ok parado (println "fim"))
      (err e (println e)))))
```

- Exatamente um `WSConn`; no máximo um `Request`; os demais vêm de `{nome}` e são `int` ou `str`
  (`E_WS_PARAM`). `(effects ...)` inclui `net`.
- Ordem fixa dos blocos. `on-message` é obrigatório. Cada bloco tem escopo próprio e devolve `void`,
  então `try` dentro deles exige `(on-err ...)`.
- `on-close` roda no fechamento limpo e na queda. Fechar desinscreve de todos os tópicos.
- Autenticação: o navegador não envia `Authorization` em WebSocket. Leia o token da query com
  `(request-query req "token")` em `on-open` e recuse com `(ws-close conn 4401 "motivo")`.

---

## 3. Instruções e expressões

Todo corpo é uma sequência de formas entre parênteses.

| Forma | O que faz |
|---|---|
| `(let nome Tipo valor)` | Declara. O tipo pode ser omitido: `(let n (str-len s))`. Veja a restrição abaixo |
| `(set nome valor)` | Reatribui uma variável já declarada, com o mesmo tipo |
| `(return valor)` / `(return)` | Sai da função |
| `(if cond (then ...) (else ...))` | Condicional. `else` é opcional como instrução |
| `(if cond (then valor) (else valor))` | Como expressão: `else` obrigatório (`E_IF_EXPR_MISSING_ELSE`) e os dois ramos do mesmo tipo (`E_IF_EXPR_BRANCH_TYPE`) |
| `(while cond instrução...)` | Laço |
| `(for-range i início fim instrução...)` | `i` de `início` até `fim - 1` (fim exclusivo). Limites avaliados uma vez |
| `(for-each x lista instrução...)` | Percorre uma `(list T)`. Para mapas: `(for-each k (map-keys m) ...)` |
| `(break)` / `(continue)` | Só dentro de laço (`E_LOOP_CONTROL`) |
| `(match r (ok v ...) (err e ...))` | Trata um `result` (§5) |
| `(match o (some v ...) (none ...))` | Trata um `option` (§5) |
| `(try expr)` | Desempacota um `result` ou sai da função com o erro (§5) |
| `(db-transaction conn instrução...)` | Transação (§8.13) |
| `(spawn (funcao args...))` | Executa em paralelo (efeito `spawn`). A função chamada devolve `void` |
| `(send canal valor)` / `(recv canal)` | Envia / recebe num canal (§6) |
| `(do expr)` | Avalia uma expressão como instrução. Raramente necessário |
| `(call f args...)` | Forma antiga de `(f args...)`. Aceita, mas não canônica |

**Escopo.** Cada bloco (`then`, `else`, corpo de laço, ramo de `match`) tem escopo próprio. Repetir um
nome no mesmo escopo é `E_DUPLICATE_SYMBOL`; num escopo interno, o nome novo esconde o de fora.

**Restrição do `let` sem tipo.** Quando o valor começa com `list`, `map`, `result`, `option` ou
`chan`, o parser lê essa lista como **tipo**. Então:

```liaf
(module m
  (fn main () void (effects io)
    (let n (str-len "abc"))              ; ok: inferido como int
    (let nomes (list str) (list "a" "b")) ; com list, o tipo é obrigatório
    (println (fmt "{} {}" n (list-len nomes)))))
```

**Valores em lista e mapa são referências.** `(list-push l x)` dentro de uma função altera a lista de
quem chamou. Structs e textos são imutáveis.

---

## 4. Efeitos

Toda função declara o que faz fora do cálculo puro. O checker confere nos dois sentidos:

- Usar um efeito sem declarar: `E_UNDECLARED_EFFECT`.
- Declarar sem usar: `E_UNUSED_EFFECT`.
- Nome desconhecido: `E_UNKNOWN_EFFECT`; repetido: `E_DUPLICATE_EFFECT`.

Efeitos são **transitivos**: chamar uma função que declara `fs` exige `fs` em quem chama.

| Efeito | Para quê | Exemplos de builtins |
|---|---|---|
| `io` | Terminal e argumentos | `println`, `print`, `read-line`, `args`, `exit` |
| `fs` | Disco | `fs-*`, `serve-hybrid`, `serve-site`, `file-response`, `storage-*` |
| `net` | Rede | `http-fetch`, `ws-*`, `serve-*`, `http-get` |
| `db` | Banco de dados | `db-*`, `redis-*` |
| `clock` | Relógio e espera | `now-ms`, `sleep-ms`, `jwt-verify` |
| `rand` | Aleatoriedade | `rand-int`, `random-token`, `password-hash` |
| `env` | Variáveis de ambiente | `env-get` |
| `spawn` | Concorrência | a instrução `spawn` |

Puras (sem efeito): aritmética, textos, coleções, `json-encode`/`json-decode`, crypto (exceto as de
`rand`), construir `Response`, ler `Request`.

---

## 5. Erros: `result`, `option`, `try`, `match`

Operações que podem falhar devolvem `(result T E)`, quase sempre `(result T str)`. **Ignorar um
`result` é erro de compilação** (`E_UNHANDLED_RESULT`). Há quatro formas de tratar:

```liaf
(module m
  (fn ler-porta () (result int str) (effects env)
    (let texto str (try (env-get "PORTA")))   ; 1. try: propaga o erro
    (int-from-str texto))

  (fn main () void (effects io env)
    (match (ler-porta)                        ; 2. match: trata os dois lados
      (ok p (println (fmt "porta {}" p)))
      (err e (println (fmt "sem porta: {}" e))))
    (let porta int (unwrap-or (ler-porta) 8080)) ; 3. unwrap-or: valor padrão
    (println porta)))
```

A quarta é **devolver** o `result` para quem chamou, como `ler-porta` faz com `(int-from-str texto)`.

### 5.1 `try` e `on-err`

`(try expr)` devolve o valor de sucesso ou sai da função com o erro. Só é permitido onde o erro tem
destino (`E_UNHANDLED_RESULT` caso contrário):

- a função devolve `(result ...)`: o erro vira o retorno; ou
- a função tem um bloco `(on-err var ...)`: o erro cai nele, em `var`.

```liaf
(module m
  (struct Estado (total int))
  (fn salvar ((estado Estado)) Response (effects fs)
    (on-err mensagem (json-response 500 mensagem))
    (let texto str (try (json-encode estado)))
    (try (fs-write-atomic "estado.json" texto))
    (json-response 200 texto))
  (fn main () void (effects fs io)
    (println (str-from-int 1))
    (let r Response (salvar (new Estado 1)))
    (println "salvo")))
```

**Aplique `try` direto na chamada.** `(let r (result int str) (f))` seguido de `(try r)` não conta
como tratamento de `r`. Com um `result` guardado em variável, use `match` ou `unwrap-or`.

### 5.2 `match`

- `result`: ramos `(ok v ...)` e `(err e ...)`, **nessa ordem** e ambos obrigatórios.
- `option`: ramos `(some v ...)` e `(none ...)`, nessa ordem.
- Faltou um ramo: `E_NONEXHAUSTIVE_MATCH`.
- Como **expressão**, cada ramo tem uma única expressão e os dois são do mesmo tipo:

```liaf
(module m
  (fn main () void (effects io)
    (let r (result int str) (int-from-str "42"))
    (let texto str (match r
      (ok n (fmt "número {}" n))
      (err _ "inválido")))
    (println texto)))
```

### 5.3 Construir `result` e `option`

- `(ok v)` e `(err e)` só onde o tipo esperado é conhecido: retorno declarado, `let` com tipo ou
  argumento tipado (`E_RESULT_CONTEXT`).
- `(some v)` e `(none T)`: o `none` leva o tipo, como em `(none int)`.

---

## 6. Tipos

| Tipo | Valores | Notas |
|---|---|---|
| `int` | inteiro de 64 bits | Overflow **aborta** o programa (saída 1) |
| `float` | real de 64 bits | IEEE 754. `int` e `float` não se misturam: converta com `float-from-int` |
| `str` | texto UTF-8 imutável | Tamanho e índices contam caracteres, não bytes |
| `bool` | `true`, `false` | |
| `void` | — | Só como retorno |
| `(list T)` | lista homogênea | Referência. Vira array JSON |
| `(map K V)` | mapa | `K` escalar: `int`, `float`, `str` ou `bool` (`E_INVALID_MAP_KEY`) |
| `(option T)` | `(some v)` / `(none T)` | |
| `(result T E)` | `(ok v)` / `(err e)` | |
| `(chan T)` | canal entre tarefas | `(make-chan T)`, `send`, `recv` |
| Struct | `(new Tipo ...)` | Declarada com `struct` |
| `Request`, `Response` | HTTP | Opacos: sem campos (`E_UNKNOWN_FIELD`) |
| `DBConnection`, `WSConn`, `HttpReply` | Banco, WebSocket, cliente HTTP | Opacos |

- Tipo desconhecido: `E_UNKNOWN_TYPE`. Construtor de tipo desconhecido: `E_INVALID_TYPE_CONSTRUCTOR`.
- `eq`/`neq` só comparam escalares (`int`, `float`, `str`, `bool`).
- `println`, `print` e `concat` só aceitam escalares. Para listas e structs, use `json-encode` ou `fmt`.

**Concorrência:**

```liaf
(module m
  (fn trabalhar ((saida (chan int))) void
    (send saida 42))
  (fn main () void (effects io spawn)
    (let c (chan int) (make-chan int))
    (spawn (trabalhar c))
    (println (recv c))))
```

---

## 7. Operadores

Prefixados, como toda chamada.

| Forma | Tipos | Retorno | Notas |
|---|---|---|---|
| `(add a b ...)` | `int` ou `float` | o mesmo | Variádico (2 ou mais) |
| `(mul a b ...)` | `int` ou `float` | o mesmo | Variádico |
| `(sub a b)` | `int` ou `float` | o mesmo | **Só 2** operandos |
| `(div a b)` | `int` | `(result int str)` | Falível: divisão por zero é `err` |
| `(div a b)` | `float` | `float` | Divisão por zero dá `+Inf`/`NaN` |
| `(eq a b)`, `(neq a b)` | escalares | `bool` | |
| `(lt a b)`, `(gt a b)`, `(lte a b)`, `(gte a b)` | `int`, `float` ou `str` | `bool` | Texto compara em ordem lexicográfica |
| `(and a b ...)`, `(or a b ...)` | `bool` | `bool` | Variádicos |
| `(not a)` | `bool` | `bool` | É builtin, não operador |

Os dois lados sempre do mesmo tipo: `(add 1 2.0)` é `E_TYPE_MISMATCH`.

---

## 8. Builtins

<!-- builtins:inicio -->

### 8.1 Terminal e sistema

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(println v ...)` | `void` | `io` | Escalares separados por nada, com quebra de linha no fim |
| `(print v ...)` | `void` | `io` | Sem quebra de linha |
| `(read-line)` | `(result str str)` | `io` | Lê uma linha da entrada, sem o `\n`. Fim da entrada é `err` |
| `(args)` | `(list str)` | `io` | Argumentos da linha de comando, sem o nome do programa |
| `(exit código)` | `void` | `io` | Encerra o processo |
| `(env-get "NOME")` | `(result str str)` | `env` | Ausente é `err` |
| `(now-ms)` | `int` | `clock` | Milissegundos desde 1970 |
| `(sleep-ms n)` | `void` | `clock` | |
| `(rand-int n)` | `int` | `rand` | De `0` a `n - 1`. `n <= 0` aborta |

### 8.2 Aritmética

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(mod a b)` | `(result int str)` | — | Só `int`. Divisor zero é `err` |
| `(neg x)` | tipo de `x` | — | `int` ou `float` |
| `(abs x)` | tipo de `x` | — | `int` ou `float` |
| `(min a b)` | tipo de `a` | — | `a` e `b` do mesmo tipo numérico |
| `(max a b)` | tipo de `a` | — | Idem |
| `(pow a b)` | tipo de `a` | — | Idem |
| `(sqrt x)` | `float` | — | |
| `(floor x)` | `float` | — | |
| `(ceil x)` | `float` | — | |
| `(round x)` | `float` | — | Meio arredonda para longe do zero: `2.5` → `3` |

### 8.3 Conversões

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(str-from-int n)` | `str` | — | |
| `(str-from-float x)` | `str` | — | |
| `(str-from-bool b)` | `str` | — | `"true"` / `"false"` |
| `(int-from-str s)` | `(result int str)` | — | |
| `(float-from-str s)` | `(result float str)` | — | |
| `(bool-from-str s)` | `(result bool str)` | — | |
| `(float-from-int n)` | `float` | — | |
| `(int-from-float x)` | `int` | — | Trunca: `2.9` → `2` |

### 8.4 Texto

Índices e tamanhos contam **caracteres** (runes), não bytes: `(str-len "ação")` é `4`.

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(concat v ...)` | `str` | — | Junta escalares |
| `(fmt "modelo {}" v ...)` | `str` | — | Cada `{}` recebe o próximo argumento. `{{` e `}}` produzem chaves literais |
| `(str-len s)` | `int` | — | Em caracteres |
| `(str-byte-len s)` | `int` | — | Em bytes UTF-8 |
| `(str-slice s início fim)` | `(result str str)` | — | Fim exclusivo. Fora do intervalo é `err` |
| `(str-get s i)` | `(result str str)` | — | O caractere na posição `i` |
| `(str-index s sub)` | `int` | — | Posição da primeira ocorrência, ou `-1` |
| `(str-eq a b)` | `bool` | — | Igual a `(eq a b)` para textos |
| `(str-contains s sub)` | `bool` | — | |
| `(str-starts-with s prefixo)` | `bool` | — | |
| `(str-ends-with s sufixo)` | `bool` | — | |
| `(str-split s sep)` | `(list str)` | — | Mantém partes vazias: `"a,,b"` dá 3 partes |
| `(str-join lista sep)` | `str` | — | |
| `(str-trim s)` | `str` | — | Espaços e quebras nas pontas |
| `(str-upper s)` | `str` | — | Com acentos |
| `(str-lower s)` | `str` | — | Com acentos |
| `(str-replace s antigo novo)` | `str` | — | Todas as ocorrências |

### 8.5 Coleções

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(list a b ...)` | `(list T)` | — | Literal. Vazia, `(list)`, só com tipo conhecido: `(let l (list int) (list))` |
| `(make-list T)` | `(list T)` | — | Lista vazia |
| `(list-len l)` | `int` | — | |
| `(list-push l x)` | `void` | — | Acrescenta no fim. Altera `l` |
| `(list-get l i)` | `(result T str)` | — | Índice fora é `err` |
| `(list-set l i x)` | `(result bool str)` | — | |
| `(list-remove l i)` | `(result bool str)` | — | Remove a posição `i`, deslocando as seguintes |
| `(list-pop l)` | `(result T str)` | — | Remove e devolve o último. Vazia é `err` |
| `(list-sort l)` | `void` | — | Só `int`, `float` ou `str`. Ordena no lugar |
| `(list-contains l x)` | `bool` | — | |
| `(make-map K V)` | `(map K V)` | — | Mapa vazio |
| `(map-len m)` | `int` | — | |
| `(map-get m k)` | `(result V str)` | — | Chave ausente é `err` |
| `(map-set m k v)` | `void` | — | |
| `(map-has m k)` | `bool` | — | |
| `(map-delete m k)` | `void` | — | |
| `(map-keys m)` | `(list K)` | — | Sempre em ordem crescente |

### 8.6 `result`, `option` e structs

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(ok v)` | `(result T E)` | — | Exige tipo esperado conhecido (`E_RESULT_CONTEXT`) |
| `(err e)` | `(result T E)` | — | Idem |
| `(some v)` | `(option T)` | — | |
| `(none T)` | `(option T)` | — | Recebe o tipo: `(none int)` |
| `(unwrap-or r padrão)` | `T` | — | Aceita `result` ou `option` |
| `(new Tipo v ...)` | `Tipo` | — | Um valor por campo, na ordem |
| `(field valor campo)` | tipo do campo | — | Campo inexistente: `E_UNKNOWN_FIELD` |

### 8.7 JSON

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(json-encode v)` | `(result str str)` | — | Qualquer valor. `(list T)` vira array `[...]` |
| `(json-decode texto Tipo)` | `(result Tipo str)` | — | `Tipo` pode ser composto: `(json-decode s (list Pedido))` |

- A **chave JSON é o nome do campo exatamente como declarado**. O campo `created-at` lê e escreve
  `"created-at"`, e não `"created_at"`. Para conversar com APIs em `snake_case` ou `camelCase`,
  declare o campo com esse nome: `(struct U (created_at str) (userId int))`.
- Campo ausente no JSON vira o valor zero do tipo (`0`, `""`, `false`), sem erro. Tipo errado é `err`.

### 8.8 Arquivos

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(fs-read-file caminho)` | `(result str str)` | `fs` | |
| `(fs-write-file caminho texto)` | `(result void str)` | `fs` | Sobrescreve |
| `(fs-write-atomic caminho texto)` | `(result void str)` | `fs` | Grava num temporário e renomeia: quem lê nunca vê o arquivo pela metade |
| `(fs-rename de para)` | `(result void str)` | `fs` | |
| `(fs-remove caminho)` | `(result void str)` | `fs` | |
| `(fs-exists caminho)` | `bool` | `fs` | |

`(result void str)`: o `ok` não carrega valor. Trate com `(try ...)` ou `match`, sem testar conteúdo.

### 8.9 Criptografia

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(sha256 texto "hex")` | `str` | — | Codificação literal: `"hex"` ou `"base64url"` (`E_UNKNOWN_ENCODING`) |
| `(hmac-sha256 chave texto "base64url")` | `str` | — | Idem |
| `(secure-eq a b)` | `bool` | — | Comparação em tempo constante. Use para assinaturas e tokens |
| `(base64url-encode s)` | `str` | — | Sem padding |
| `(base64url-decode s)` | `(result str str)` | — | |
| `(random-token)` | `str` | `rand` | 32 bytes aleatórios em base64url (43 caracteres) |
| `(password-hash senha)` | `str` | `rand` | argon2id (19 MiB, t=2), formato PHC com os parâmetros dentro |
| `(password-verify senha hash)` | `bool` | — | Hash malformado também dá `false` |

Nenhum parâmetro de segurança é configurável: o caminho fácil é o seguro.

### 8.10 Requisição e resposta HTTP

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(json-response status corpo)` | `Response` | — | `Content-Type: application/json` |
| `(raw-response status tipo corpo)` | `Response` | — | Qualquer `Content-Type`, corpo pode ser binário |
| `(file-response caminho tipo)` | `(result Response str)` | `fs` | Serve um arquivo do disco |
| `(response-set-header res "Nome" "valor")` | `Response` | — | Devolve **nova** resposta. `Set-Cookie` acumula; os outros substituem |
| `(request-body req)` | `str` | — | Corpo cru |
| `(request-path req)` | `str` | — | |
| `(request-method req)` | `str` | — | |
| `(request-header req "Nome")` | `(result str str)` | — | Ausente é `err`; presente e vazio é `(ok "")` |
| `(request-query req "chave")` | `(result str str)` | — | |
| `(request-file-data req "campo")` | `(result str str)` | — | Bytes do arquivo enviado (multipart, ou o corpo inteiro num upload direto) |
| `(request-file-name req "campo")` | `(result str str)` | — | Nome do arquivo (multipart, header `X-Filename` ou query) |

### 8.11 Servidor

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(serve-hybrid pasta porta)` | `(result bool str)` | `net fs` | Rotas declaradas + arquivos estáticos da pasta. Bloqueia até parar |
| `(serve-site pasta domínio porta auto-tls)` | `void` | `net fs` | Só site estático. `domínio` `""` desliga TLS |
| `(http-get "/caminho" handler)` | `void` | `net` | **Forma antiga** de registrar rota. `handler` é `(fn h ((req Request)) Response ...)` |
| `(http-post "/caminho" handler)` | `void` | `net` | Idem |
| `(http-put "/caminho" handler)` | `void` | `net` | Idem |
| `(http-delete "/caminho" handler)` | `void` | `net` | Idem |

`http-get`/`http-post` **não são cliente HTTP**. Para chamar outra API, use `http-fetch`.

### 8.12 Cliente HTTP

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(http-fetch "POST" url (list "Nome: valor") corpo)` | `(result HttpReply str)` | `net` | `err` só quando não houve resposta |
| `(reply-status r)` | `int` | — | Um `422` chega como `ok`: confira o status |
| `(reply-body r)` | `str` | — | |
| `(reply-header r "Nome")` | `(result str str)` | — | |

Método literal é conferido na compilação (`E_UNKNOWN_METHOD`). Headers: `(list)` para nenhum. Corpo
sem `Content-Type` vai como JSON. Timeout de 30 s, resposta de até 10 MiB e obrigatoriamente texto.
Mensagens de erro citam método e host, nunca a URL.

### 8.13 Banco de dados

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(db-connect "postgres" dsn)` | `(result DBConnection str)` | `db` | Drivers: `"postgres"`, `"mysql"`, `"sqlite"`, `"redis"` (literal) |
| `(db-close conn)` | `(result void str)` | `db` | Opcional num servidor |
| `(db-query conn "SQL" Linha arg ...)` | `(result (list Linha) str)` | `db` | `Linha` é uma struct |
| `(db-exec conn "SQL" arg ...)` | `(result int str)` | `db` | Linhas afetadas |
| `(redis-get conn chave)` | `(result str str)` | `db` | Chave ausente é `err` |
| `(redis-set conn chave valor ttl)` | `(result void str)` | `db` | `ttl` em segundos (`int`) |

- **O SQL é sempre um literal escrito no fonte.** Variável ou `concat` é `E_SQL_INTERPOLATION`. Todo
  valor entra como argumento (`int`, `float`, `str` ou `bool`), enviado fora do texto do comando.
- Marcadores: `$1, $2...` no PostgreSQL; `?` no MySQL. O número de marcadores tem de bater com o de
  argumentos (`E_SQL_PARAM_COUNT`).
- Cada coluna preenche o campo de mesmo nome; `snake_case` na coluna também preenche o campo
  `kebab-case`. `NULL` vira o valor zero.
- DSN em URL (`postgres://u:s@host:5432/banco?sslmode=require`); no SQLite, o caminho do arquivo.
  `pool_max=N` limita o pool (padrão 16). Chamar `db-connect` em cada rota é barato: o mesmo DSN
  reaproveita o pool.

```liaf
(module banco
  (struct Usuario (id int) (nome str))

  (fn listar () (result (list Usuario) str) (effects db)
    (let conn DBConnection (try (db-connect "sqlite" "app.db")))
    (db-query conn "SELECT id, nome FROM usuarios WHERE id > ?" Usuario 0))

  (fn transferir ((de int) (para int)) (result bool str) (effects db)
    (let conn DBConnection (try (db-connect "sqlite" "app.db")))
    (db-transaction conn
      (try (db-exec conn "UPDATE contas SET saldo = saldo - 10 WHERE id = ?" de))
      (try (db-exec conn "UPDATE contas SET saldo = saldo + 10 WHERE id = ?" para)))
    (ok true))

  (fn main () void (effects db io)
    (match (listar)
      (ok us (for-each u us (println (field u nome))))
      (err e (println e)))
    (match (transferir 1 2)
      (ok _ (println "ok"))
      (err e (println e)))))
```

`db-transaction`: dentro do bloco, `conn` designa a transação. Confirma no fim; qualquer saída
antecipada (inclusive um `try` que falhou) desfaz. Exige `(on-err ...)` ou retorno `result`. Não
aninha.

### 8.14 WebSocket

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(ws-send conn texto)` | `(result void str)` | `net` | |
| `(ws-send-json conn valor)` | `(result void str)` | `net` | Serializa qualquer valor |
| `(ws-close conn código motivo)` | `(result void str)` | `net` | Ex.: `4401` para não autorizado |
| `(ws-join conn tópico)` | `void` | `net` | Sem `ws-join`, a conexão não recebe broadcast |
| `(ws-leave conn tópico)` | `void` | `net` | |
| `(ws-broadcast tópico texto)` | `(result int str)` | `net` | Quantos receberam. Local ao processo |
| `(ws-topic-size tópico)` | `int` | `net` | |

Mensagem de até 1 MiB; ping a cada 30 s; conexão ociosa cai em 90 s.

### 8.15 Imagens e armazenamento

Os bytes de arquivos binários trafegam como `str`.

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(image-to-webp dados)` | `(result str str)` | — | PNG, JPEG, GIF ou BMP → WebP sem perdas |
| `(image-to-webp-quality dados nível)` | `(result str str)` | — | `nível` de esforço, 0 a 9 |
| `(image-dimensions dados)` | `(result str str)` | — | `"LARGURAxALTURA"` |
| `(storage-dir-init pasta)` | `(result bool str)` | `fs` | Cria a pasta se não existir |
| `(storage-save-image pasta nome dados)` | `(result str str)` | `fs` | Converte para WebP, salva e devolve o nome final |
| `(storage-file-get pasta nome)` | `(result str str)` | `fs` | Bytes do arquivo |
| `(storage-file-delete pasta nome)` | `(result bool str)` | `fs` | |
| `(storage-file-exists pasta nome)` | `bool` | `fs` | |

Os nomes são saneados contra `../`. Para o fluxo completo de upload, prefira `std/storage` (§9.4).

### 8.16 Lógica e canais

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(not b)` | `bool` | — | Negação. Os demais operadores lógicos estão em §7 |
| `(make-chan T)` | `(chan T)` | — | Use com `send`, `recv` e `spawn` |

<!-- builtins:fim -->

---

## 9. Biblioteca padrão (`std/`)

Escrita em LIAF e embutida no `liafc`: `(import "std/nome")` nunca lê o disco. Os nomes públicos
levam o prefixo do módulo (`auth-`, `jwt-`...). Não declare funções com esses prefixos.

### 9.1 `std/auth`

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(auth-bearer-token req)` | `(result str str)` | — | Token de `Authorization: Bearer ...`. Esquema em qualquer caixa. `err` para header ausente, outro esquema ou token vazio |

### 9.2 `std/jwt`

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(jwt-sign segredo payload-json)` | `(result str str)` | — | HS256. Segredo com menos de 32 bytes é `err` |
| `(jwt-verify segredo token)` | `(result str str)` | `clock` | Devolve o payload JSON. Recusa assinatura inválida, `alg` diferente de HS256 (inclusive `none`), token vencido e token sem `exp` |

O payload precisa de `exp` em segundos desde 1970. Decodifique o resultado no seu próprio struct:

```liaf
(module sessao
  (import "std/jwt")
  (struct Sessao (sub str) (exp int))

  (fn emitir ((segredo str) (usuario str)) (result str str) (effects clock)
    (let expira int (add (unwrap-or (div (now-ms) 1000) 0) 3600))
    (jwt-sign segredo (try (json-encode (new Sessao usuario expira)))))

  (fn validar ((segredo str) (token str)) (result Sessao str) (effects clock)
    (json-decode (try (jwt-verify segredo token)) Sessao))

  (fn main () void (effects clock io)
    (let segredo str "um-segredo-com-pelo-menos-32-bytes!!")
    (match (emitir segredo "ana")
      (ok t (match (validar segredo t)
        (ok s (println (field s sub)))
        (err e (println e))))
      (err e (println e)))))
```

### 9.3 `std/cors`

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(cors-origin req (list "https://site"))` | `(result str str)` | — | A origem, se estiver na lista. Nunca `*` |
| `(cors-headers res origem)` | `Response` | — | `Access-Control-Allow-Origin` + `Vary: Origin` |
| `(cors-preflight origem "GET, POST" "Authorization")` | `Response` | — | `204` para o `OPTIONS` |
| `(cors-headers-credentials res origem)` | `Response` | — | Acrescenta `Allow-Credentials`. Só para API que usa cookie |
| `(cors-preflight-credentials origem métodos headers)` | `Response` | — | Idem |

Não há middleware: o preflight exige uma `(route OPTIONS "/caminho" ...)` por caminho. `cors-headers`
**substitui** um `Vary` já existente.

### 9.4 `std/storage`

| Forma | Retorna | Efeitos | Notas |
|---|---|---|---|
| `(storage-init pasta)` | `(result bool str)` | `fs` | Cria a pasta de armazenamento caso não exista |
| `(storage-upload-image req "campo" pasta)` | `(result str str)` | `fs` | Lê o upload, converte para WebP lossless (nível 4), salva e devolve o nome |
| `(storage-upload-image-opt req "campo" pasta max-w max-h max-bytes level)` | `(result str str)` | `fs` | Upload com redimensionamento proporcional, limite de bytes e nível de compressão WebP (0-9) |
| `(storage-serve-image pasta nome)` | `Response` | `fs` | `image/webp` com cache longo; `404` se não existe |
| `(storage-delete-image pasta nome)` | `(result bool str)` | `fs` | |
| `(storage-image-exists pasta nome)` | `bool` | `fs` | |

---

## 10. Servidor web

### 10.1 Como sobe

`main` chama `(serve-hybrid "./public" porta)`. Ele atende, nesta ordem, as rotas `route` e
`ws-route` declaradas em qualquer arquivo do programa, e depois os arquivos da pasta. A pasta é
relativa ao **diretório atual** de quem executa o binário. Método não permitido responde `405`.

Com `liafc build --embed`, a pasta vai dentro do executável; sem ele, é lida do disco na partida.

### 10.2 Dentro de uma rota

```liaf
(module api
  (import "std/auth")

  (route GET "/perfil" ((req Request)) Response
    (on-err motivo (response-set-header (json-response 401 "{}") "WWW-Authenticate" "Bearer"))
    (let token str (try (auth-bearer-token req)))
    (let ordem str (unwrap-or (request-query req "ordem") "recentes"))
    (response-set-header
      (json-response 200 (fmt "{{\"token\":\"{}\",\"ordem\":\"{}\"}}" token ordem))
      "Cache-Control" "no-store"))

  (fn main () void (effects fs io net)
    (match (serve-hybrid "./public" "8080")
      (ok parado (println "fim"))
      (err e (println e)))))
```

Header com nome inválido ou quebra de linha no valor responde `500`, sem vazar a resposta: isso
bloqueia injeção de header.

### 10.3 Arquivos estáticos e SSG

Tudo é carregado na RAM na partida (mídias grandes ficam no disco e são servidas por streaming, com
suporte a `Range`):

- **Minificação e gzip** de HTML, CSS e JS, com `ETag` e `304`. Desligue a minificação com
  `LIAF_MINIFY=false`.
- **URLs limpas:** `/sobre` serve `sobre.html`.
- **Versão automática de assets:** `href`/`src` para CSS, JS e imagens ganham `?v=<hash>`.
- **Layout:** na página, `<!-- layout "base.html" -->`; no layout, `<!-- content -->` (ou
  `<!-- slot -->`) marca onde a página entra.
- **Inclusão:** `<!-- include "menu.html" -->`, até 10 níveis.
- **Markdown:** `pasta/post.md` vira `/pasta/post`, com o layout do frontmatter (`layout: x.html`) ou
  `base.html`. O frontmatter vira variáveis `{{ title }}` / `{{ page.title }}`.
- **Coleções:** cada `data/nome.json` (um array de objetos) e todos os posts Markdown (`posts`,
  ordenados por `date` decrescente) podem ser listados:

```html
<!-- for post in posts -->
  <a href="{{ post.url }}">{{ post.title }}</a>
<!-- endfor -->
```

- **Recarga e publicação:** `POST /_liaf/reload` relê a pasta; `POST /_liaf/publish` grava um
  arquivo. Os dois exigem `Authorization: Bearer $LIAF_DEPLOY_TOKEN` e ficam desligados sem o token.

---

## 11. Diagnósticos

`liafc check arquivo.liaf --json` devolve:

```json
{
  "status": "error",
  "errors": [
    { "code": "E_TYPE_MISMATCH", "file": "app.liaf", "line": 12, "col": 5, "message": "Expected int, received float" }
  ]
}
```

Com sucesso, `"status": "success"`. Não há campo de correção sugerida: a correção vem do código e da
mensagem, usando a tabela abaixo.

### 11.1 Sintaxe (parser)

| Código | Causa | Correção |
|---|---|---|
| `E_EXPECTED_MODULE` | Arquivo não começa com `(module nome` | Envolva tudo num `(module nome ...)` |
| `E_EXPECTED_MODULE_NAME` | `(module)` sem nome | `(module app ...)` |
| `E_EXPECTED_LPAREN` | Algo solto no nível superior | Toda declaração começa com `(` |
| `E_INVALID_TOPLEVEL` | Forma não permitida no topo (ex.: `let` fora de função) | Só `import`, `struct`, `fn`, `route`, `ws-route` |
| `E_UNEXPECTED_TOKEN` | Parêntese sobrando ou faltando; nome/literal solto como última expressão; `sub`/`div` com 3 operandos | Conte os parênteses da forma indicada; troque `x` final por `(return x)` |
| `E_UNKNOWN_STATEMENT` | Instrução que não começa com uma forma conhecida | Confira a lista de §3 |
| `E_INVALID_EXPR` | Expressão composta mal formada | |
| `E_EXPECTED_FN_NAME`, `E_EXPECTED_STRUCT_NAME`, `E_EXPECTED_FIELD_NAME`, `E_EXPECTED_PARAM_NAME` | Falta o nome | |
| `E_EXPECTED_LET_IDENT`, `E_EXPECTED_SET_IDENT` | `let`/`set` sem nome de variável | `(let nome Tipo valor)` |
| `E_EXPECTED_ON_ERR_VAR` | `(on-err ...)` sem a variável | `(on-err mensagem ...)` |
| `E_EXPECTED_ON_MESSAGE`, `E_EXPECTED_ON_MESSAGE_VAR` | `ws-route` sem `(on-message var ...)` | |
| `E_EXPECTED_ROUTE_METHOD`, `E_EXPECTED_ROUTE_PATH` | `route` sem método ou caminho literal | `(route GET "/x" ...)` |
| `E_EXPECTED_IMPORT_PATH` | `import` sem texto | `(import "std/auth")` |
| `E_EXPECTED_CALL_IDENT` | `(call)` sem função | |
| `E_EXPECTED_CALL_IN_SPAWN` | `spawn` sem chamada | `(spawn (f x))` |
| `E_EXPECTED_TYPE_CONSTRUCTOR`, `E_INVALID_TYPE_CONSTRUCTOR` | Tipo composto desconhecido | `list`, `map`, `option`, `result`, `chan` |
| `E_INVALID_TYPE` | Algo no lugar do tipo; `(let l (list 1 2))` sem tipo; `new` sem struct; `db-query` sem struct | Anote o tipo: `(let l (list int) (list 1 2))` |
| `E_INVALID_NUMBER` | Inteiro fora de 64 bits | |
| `E_NONEXHAUSTIVE_MATCH` | Falta ramo, ou ordem trocada | `(ok ..) (err ..)` ou `(some ..) (none ..)` |

### 11.2 Tipos, nomes e fluxo (checker)

| Código | Causa | Correção |
|---|---|---|
| `E_TYPE_MISMATCH` | Tipo diferente do esperado; `int` com `float`; composto em `println`/`concat`/`eq` | Converta (`float-from-int`, `str-from-int`) ou serialize (`json-encode`, `fmt`) |
| `E_UNDEFINED_SYMBOL` | Variável ou função inexistente | Declare antes com `let`; confira o nome e o import |
| `E_DUPLICATE_SYMBOL` | Nome repetido no mesmo escopo | Renomeie ou use `set` |
| `E_UNKNOWN_TYPE` | Tipo não declarado | Declare a `struct` ou importe o arquivo |
| `E_UNKNOWN_FIELD` | Campo inexistente, ou campo em tipo opaco | Confira a `struct` |
| `E_WRONG_ARITY`, `E_ARITY_MISMATCH` | Número de argumentos errado | Veja a assinatura em §8 |
| `E_TYPE_INFERENCE_FAILED` | `let` sem tipo com valor `void` ou indeterminado | Escreva o tipo |
| `E_INVALID_MAP_KEY` | Chave de mapa não escalar | Use `int`, `float`, `str` ou `bool` |
| `E_RESULT_CONTEXT` | `ok`/`err` sem tipo esperado | Use em `let` com tipo ou como retorno declarado |
| `E_UNHANDLED_RESULT` | `result` ignorado; `try` sem destino; `try` sobre variável | `try` direto na chamada, com `on-err` ou retorno `result`; ou `match`/`unwrap-or` |
| `E_MISSING_RETURN` | Função não-`void` sem retorno em algum caminho | Termine cada caminho com uma expressão ou `return` |
| `E_IF_EXPR_MISSING_ELSE` | `if` como valor sem `else` | Acrescente `(else ...)` |
| `E_IF_EXPR_BRANCH_TYPE` | Ramos de tipos diferentes | Iguale os tipos |
| `E_LOOP_CONTROL` | `break`/`continue` fora de laço | |
| `E_CHANNEL_TYPE_MISMATCH` | `send`/`recv` sem canal | |
| `E_MAIN_SIGNATURE`, `E_MISSING_MAIN` | `main` ausente, com parâmetros ou sem `void` | `(fn main () void ...)` |
| `E_HANDLER_SIGNATURE` | Handler de `http-get` não é `(Request) -> Response` | Ou use `route` |

### 11.3 Efeitos

| Código | Causa | Correção |
|---|---|---|
| `E_UNDECLARED_EFFECT` | Usa efeito não declarado (inclusive por uma função chamada) | Acrescente em `(effects ...)` |
| `E_UNUSED_EFFECT` | Declara efeito que não usa | Remova |
| `E_UNKNOWN_EFFECT` | Efeito inexistente | Só os de §4 |
| `E_DUPLICATE_EFFECT` | Efeito repetido | |

### 11.4 Web, banco, crypto

| Código | Causa | Correção |
|---|---|---|
| `E_ROUTE_PARAM` | `{nome}` sem parâmetro homônimo, ou parâmetro de caminho que não é `int`/`str` | |
| `E_WS_PARAM` | `ws-route` sem exatamente um `WSConn`, com dois `Request`, ou parâmetro fora do caminho | |
| `E_SQL_INTERPOLATION` | SQL não literal | Escreva o SQL inteiro no fonte e passe valores como argumentos |
| `E_SQL_PARAM_COUNT` | Marcadores ≠ argumentos, ou `$1` e `?` misturados | |
| `E_UNKNOWN_DRIVER` | Driver fora da lista, ou não literal | `"postgres"`, `"mysql"`, `"sqlite"`, `"redis"` |
| `E_UNKNOWN_ENCODING` | Codificação de `sha256`/`hmac-sha256` não literal ou inválida | `"hex"` ou `"base64url"` |
| `E_UNKNOWN_METHOD` | Método de `http-fetch` inválido | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`... |

### 11.5 Módulos e arquivos (loader)

| Código | Causa |
|---|---|
| `E_IMPORT_NOT_FOUND` | Arquivo ou módulo da std inexistente (a mensagem lista os da std) |
| `E_CIRCULAR_IMPORT` | Ciclo de imports |
| `E_DUPLICATE_DECL` | Mesmo nome declarado em dois arquivos |
| `E_DUPLICATE_ROUTE` | Mesma rota em dois lugares |
| `E_STD_RELATIVE_IMPORT` | Módulo da std importando arquivo local |
| `E_NO_LIAF_FILES` | Import de pasta sem `.liaf` |
| `E_FILE_NOT_FOUND`, `E_FILE_READ`, `E_DIR_READ`, `E_INVALID_PATH` | Problema no caminho passado ao `liafc` |
| `E_UNRESOLVED_IMPORT` | Erro interno: o checker recebeu imports não resolvidos |

---

## 12. CLI `liafc`

| Comando | Faz |
|---|---|
| `liafc check arq.liaf [--json]` | Verifica sintaxe, tipos e efeitos. Não gera nada |
| `liafc run arq.liaf` | Compila e executa |
| `liafc build arq.liaf [-o saida] [--embed]` | Gera o executável. `--embed` põe a pasta pública dentro dele |
| `liafc emit arq.liaf` | Mostra o Go gerado |
| `liafc fmt arq.liaf [-w] [-l]` | Forma canônica. `-w` grava, `-l` lista quem está fora. Recusa gravar arquivo com comentários sem `--drop-comments` |
| `liafc publish arq --url URL [--path rota] [--token T]` | Publica um arquivo num servidor em execução |
| `liafc service install --name N --bin B [--workdir D] [--dry-run]` | Gera e instala uma unidade systemd |
| `liafc deploy caddy-bind --domain D --upstream H:P` | Registra proxy reverso no Caddy |

Cross-compilação: `GOOS=linux GOARCH=amd64 liafc build app.liaf -o app`.

**Limitação atual:** `run` e `build` precisam ser executados de dentro do repositório da LIAF (o
código gerado importa o runtime dele). `check` e `fmt` funcionam em qualquer lugar.

---

## 13. Variáveis de ambiente e limites

| Variável | Efeito |
|---|---|
| `LIAF_HOST` | Interface de escuta. `127.0.0.1` restringe ao loopback (atrás de proxy, em testes, e evita o aviso do firewall do Windows) |
| `LIAF_DEPLOY_TOKEN` | Liga `/_liaf/publish` e `/_liaf/reload`. Sem ela, ficam desligados |
| `LIAF_MINIFY=false` / `LIAF_NO_MINIFY=1` | Desliga a minificação de HTML, CSS e JS |

| Limite | Valor |
|---|---|
| Corpo de requisição HTTP | 32 MiB |
| Mensagem WebSocket | 1 MiB |
| `http-fetch` | 30 s; resposta até 10 MiB |
| Operação de banco | 30 s |
| Pool de banco | 16 conexões (`pool_max`) |

---

## 14. O que não existe

Para não perder tentativas procurando:

- Variáveis globais e constantes de módulo. Passe valores por parâmetro, ou chame uma função que
  devolva o valor (`(fn origens () (list str) (list "https://a.com"))`).
- Funções como valor, closures, lambdas, `map`/`filter` sobre listas. Use `for-each`.
- Genéricos declarados pelo usuário, enums e tipos soma (além de `result` e `option`).
- Acesso por ponto (`p.nome`), alteração de campo, métodos.
- Middleware: cada rota chama a autenticação e o CORS explicitamente.
- Funções privadas: tudo o que um arquivo declara é visível a quem o importa.
- Testes escritos em LIAF.
- Tipo de bytes: dados binários trafegam como `str`.
- Data e hora formatadas (só `now-ms`), expressões regulares, tarefas agendadas.
- Ler cookies (não há `request-cookie`; leia o header `Cookie` com `request-header`).
