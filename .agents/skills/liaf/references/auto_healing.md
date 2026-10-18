# Catálogo de erros e como corrigir

`liafc check app.liaf --json` devolve `{"status": "error", "errors": [...]}`. Cada erro traz
`code`, `file`, `line`, `col` e `message`; alguns trazem também `suggested_patch` com a
correção pronta. O laço é: leia o **primeiro** erro, corrija a causa, rode de novo. Um erro de
sintaxe costuma gerar outros em cascata, que somem junto com ele.

Esta tabela cobre **todos** os códigos que o compilador emite (o teste `TestSkillExamples`
falha se aparecer um código novo sem entrada aqui). Os que começam com `W_` também reprovam o
`check`: não são avisos para ignorar.

## Sintaxe (parser)

| Código | Causa | Correção |
|---|---|---|
| `E_UNEXPECTED_TOKEN` | Token fora do lugar: faltou `end`, `)` ou `,`; usou `%`; `if` como expressão; `else if` | Confira o bloco anterior à linha apontada. Veja as armadilhas em [syntax.md](syntax.md) |
| `E_EXPECTED_MODULE` | O arquivo não começa com `module nome` | Primeira linha: `module nome` |
| `E_EXPECTED_MODULE_NAME` | `module` sem nome | `module loja` |
| `E_INVALID_TOPLEVEL` | Declaração de topo desconhecida, ou código solto fora de função | Só `import`, `struct`, `enum`, `fn`, `middleware`, `route`, `ws-route`, `test`. Não há constantes de topo: use uma `fn` |
| `E_INVALID_PUB_DECL` | `pub` antes de algo que não é `fn`, `struct`, `enum` ou `middleware` | Tire o `pub` |
| `E_EXPECTED_LPAREN` | Declaração em S-expression sem `(` | Na sintaxe linear, não abra parênteses na declaração |
| `E_EXPECTED_IMPORT_PATH` | `import` sem texto | `import "std/auth"` ou `import "./outro.liaf"` |
| `E_EXPECTED_STRUCT_NAME`, `E_EXPECTED_ENUM_NAME`, `E_EXPECTED_FN_NAME` | Declaração sem nome | `struct Pedido`, `enum Status`, `fn calcular(...)` |
| `E_EXPECTED_FIELD_NAME` | Campo de struct malformado, ou `.` sem nome depois | `nome str`; `p.nome` |
| `E_EXPECTED_VARIANT_NAME` | Variante de enum malformada | Uma variante por linha: `Pago` |
| `E_EXPECTED_PARAM_NAME` | Parâmetro sem nome | `fn f(a int, b str)` |
| `E_EXPECTED_TYPE_CONSTRUCTOR`, `E_INVALID_TYPE_CONSTRUCTOR` | Tipo composto desconhecido | `(list T)`, `(map K V)`, `(option T)`, `(result T E)`, `(chan T)` |
| `E_EXPECTED_LET_NAME`, `E_EXPECTED_LET_IDENT` | `let` sem nome | `let x = expr` |
| `E_EXPECTED_SET_NAME`, `E_EXPECTED_SET_IDENT` | `set` sem nome | `set x = expr` (a variável já existe) |
| `E_EXPECTED_LOOP_VAR` | `for-range`/`for-each` sem variável | `for-range i 0 10`, `for-each item lista` |
| `E_EXPECTED_CASE` | Bloco de `match` sem `case` | `case ok(v)`, `case Status.Pago` |
| `E_EXPECTED_MATCH_PATTERN`, `E_EXPECTED_IDENT` | `case` sem padrão ou sem variável | `case err(msg)`; use `_` para ignorar |
| `E_EXPECTED_ON_ERR_VAR` | `on-err` sem variável | `on-err motivo` |
| `E_EXPECTED_ROUTE_METHOD`, `E_EXPECTED_ROUTE_PATH` | `route` sem método ou caminho literal | `route GET "/x" () Response` |
| `E_EXPECTED_MIDDLEWARE_NAME` | `with()` vazio ou com algo que não é nome | `with(exigir_token)` |
| `E_EXPECTED_ON_MESSAGE`, `E_EXPECTED_ON_MESSAGE_VAR` | `ws-route` sem `on-message texto` | Toda `ws-route` precisa de `on-message nome ... end` |
| `E_EXPECTED_CONN_NAME` | `db-transaction` sem a conexão | `db-transaction db ... end` |
| `E_EXPECTED_TEST_NAME` | `test` sem nome em texto | `test "calcula desconto"` |
| `E_EXPECTED_CALL_IDENT`, `E_EXPECTED_CALL_IN_SPAWN` | S-expression: `call`/`spawn` sem uma chamada | `(spawn (trabalhador ch 1))` |
| `E_INVALID_EXPR` | Expressão composta inválida, ou palavra reservada usada como base de `.campo` | Renomeie a variável (`if.x` não vale) |
| `E_UNKNOWN_STATEMENT` | S-expression: comando desconhecido | Confira a grafia |
| `E_INVALID_NUMBER` | Inteiro fora do intervalo de `int` (64 bits) | Use `float` ou divida o valor |

## Nomes e tipos (checker)

| Código | Causa | Correção |
|---|---|---|
| `E_UNDEFINED_SYMBOL` | Variável, função ou conexão não declarada. Também: `case ok v` sem parênteses | Declare com `let`, importe o módulo, ou escreva `case ok(v)` |
| `E_TYPE_MISMATCH` | Tipo diferente do esperado: `+` com texto, `json-response(200, struct)`, `try` sobre algo que não é `result`, `/` inteiro sem `try` | Converta (`str-from-int`...), serialize (`try json-encode(x)`), ou trate o `result` |
| `E_TYPE_INFERENCE_FAILED` | O `let` não conseguiu deduzir o tipo (em geral um erro anterior na expressão) | Corrija o erro anterior ou declare: `let x int = ...` |
| `E_UNKNOWN_TYPE` | Tipo não declarado | Declare a `struct`/`enum` ou corrija a grafia |
| `E_INVALID_TYPE` | Argumento de tipo inválido: `new` de algo que não é struct, `make-list((list str))`, `db-query` sem struct de linha | Use um nome de struct simples |
| `E_UNKNOWN_FIELD` | Campo inexistente na struct, ou variante sem nome | Confira os campos da declaração |
| `E_INVALID_MAP_KEY` | Chave de mapa não escalar | Chave `str`, `int`, `float` ou `bool` |
| `E_WRONG_ARITY` | Número errado de argumentos num builtin | Veja a assinatura em [builtins.md](builtins.md) |
| `E_ARITY_MISMATCH` | Variante de enum com número errado de valores, ou `fmt` sem molde | Passe um valor por campo da variante |
| `E_DUPLICATE_SYMBOL` | Dois nomes iguais no mesmo arquivo (função, struct, campo, variante) | Renomeie um deles |
| `E_PRIVATE_SYMBOL` | Função de outro módulo sem `pub` | Declare `pub fn` no módulo de origem |
| `E_REDUNDANT_BOOL_COMPARE` | `x == true` / `x == false` | Use `x` ou `!x` (vem com `suggested_patch`) |
| `E_MAIN_SIGNATURE` | `main` com parâmetro ou retorno | `fn main() void`; leia argumentos com `args()` |
| `E_MISSING_MAIN` | Programa sem `main` e sem `test` | Acrescente `fn main() void ... end` |
| `E_MISSING_RETURN` | Função não-`void` que pode terminar sem valor | Termine com uma expressão, ou cubra todos os caminhos com `return` |
| `E_HANDLER_SIGNATURE` | `http-get(caminho, f)` com `f` que não é `(req Request) Response` | Ajuste `f`, ou use `route` |
| `E_IF_EXPR_MISSING_ELSE`, `E_IF_EXPR_BRANCH_TYPE` | S-expression: `if` como valor sem `else`, ou com ramos de tipos diferentes | Na sintaxe linear `if` não é expressão: use `return` ou `match` |
| `E_UNRESOLVED_IMPORT` | Erro interno: o checker recebeu um import não resolvido | Rode pelo `liafc`, que resolve os imports antes |

## `result`, `match` e enums

| Código | Causa | Correção |
|---|---|---|
| `E_UNHANDLED_RESULT` | `try` sem destino: sem `on-err` e numa função que não devolve `result` com o mesmo erro (inclusive dentro de `test`) | Acrescente `on-err var ... end`, faça a função devolver `(result T str)`, ou use `match`/`unwrap-or` |
| `E_RESULT_CONTEXT` | `ok(...)`/`err(...)` onde o tipo do `result` não é conhecido | Use-os como retorno de uma função `(result T E)` ou num `let` tipado |
| `E_NONEXHAUSTIVE_MATCH` | `match` num enum sem todas as variantes | Acrescente os `case Enum.Variante` que a mensagem lista |
| `E_DUPLICATE_CASE` | A mesma variante duas vezes no `match` | Apague a repetida |
| `E_UNKNOWN_VARIANT` | Variante que o enum não tem | Confira a declaração do enum |
| `E_INVALID_VARIANT_CALL` | Variante escrita sem o enum | `Status.Pago`, não `Pago` |
| `E_LOOP_CONTROL` | `break`/`continue` fora de laço | Só dentro de `while`, `for-range`, `for-each` |
| `E_CHANNEL_TYPE_MISMATCH` | `recv` de algo que não é canal | `recv` recebe um `(chan T)` |

## Efeitos

| Código | Causa | Correção |
|---|---|---|
| `E_UNDECLARED_EFFECT` | A função usa um efeito (dela ou de quem ela chama) que não declarou | Acrescente-o: `effects(io, fs)` |
| `E_UNUSED_EFFECT` | Efeito declarado que o corpo não usa | Remova-o da lista |
| `E_UNKNOWN_EFFECT` | Efeito que não existe | `io`, `fs`, `net`, `clock`, `spawn`, `db`, `rand`, `env` |
| `E_DUPLICATE_EFFECT` | O mesmo efeito duas vezes | Deixe um só |

## Web, banco e criptografia

| Código | Causa | Correção |
|---|---|---|
| `E_ROUTE_PARAM` | `{nome}` no caminho sem parâmetro de mesmo nome, ou parâmetro de caminho que não é `int`/`str` | `route GET "/x/{id}" (id int)`. O `{}` tem de ser um segmento inteiro: `/x/{id}.pdf` não funciona, use `/x/{id}/pdf` |
| `E_WS_PARAM` | `ws-route` sem exatamente um `WSConn`, com dois `Request`, ou parâmetro fora do caminho | `(sala str, req Request, conn WSConn)` |
| `E_MIDDLEWARE_SIGNATURE` | `middleware` que não é `(req Request) (result Request Response)` | Ajuste a assinatura |
| `E_UNKNOWN_MIDDLEWARE` | Nome no `with` não declarado, ou declarado com `fn` | Declare com `middleware` |
| `E_DUPLICATE_MIDDLEWARE` | O mesmo middleware duas vezes no `with` | Deixe um só |
| `W_MANUAL_JSON_FMT` | JSON montado com `fmt` (`fmt("{{\"id\":{}}}", x)`): um valor com `"` injeta campos | Struct + `json-encode`; para erro, `json-error(status, msg)` |
| `E_SQL_INTERPOLATION` | SQL que não é literal (`concat`, `fmt`, variável) | Escreva o SQL inteiro no fonte e passe os valores como argumentos |
| `E_SQL_PARAM_COUNT` | Número de marcadores diferente do de argumentos, ou `$1` e `?` misturados | Um argumento por marcador, no estilo do driver |
| `E_UNKNOWN_DRIVER` | Driver fora da lista ou não literal | `"sqlite"`, `"postgres"`, `"mysql"`, `"redis"` |
| `E_UNKNOWN_ENCODING` | Codificação de `sha256`/`hmac-sha256` não literal ou inválida | `"hex"` ou `"base64url"` |
| `E_UNKNOWN_METHOD` | Método inválido em `http-fetch` | `"GET"`, `"POST"`, `"PUT"`, `"PATCH"`, `"DELETE"`... |

## Módulos e arquivos (loader)

| Código | Causa | Correção |
|---|---|---|
| `E_FILE_NOT_FOUND`, `E_INVALID_PATH`, `E_FILE_READ` | Arquivo passado ao `liafc` não existe ou não pôde ser lido | Confira o caminho |
| `E_NO_LIAF_FILES`, `E_DIR_READ` | Pasta sem `.liaf`, ou ilegível | Aponte para a pasta certa |
| `E_IMPORT_NOT_FOUND` | Import inexistente | Módulos da std: `auth`, `cli`, `cookie`, `cors`, `csv`, `fs`, `html`, `jwt`, `regex`, `seo`, `storage`, `time`. Arquivos seus: `"./arquivo.liaf"` |
| `E_CIRCULAR_IMPORT` | A importa B que importa A | Mova o que é comum para um terceiro arquivo |
| `E_STD_RELATIVE_IMPORT` | Módulo da std importando arquivo do usuário | Só acontece editando a std |
| `E_DUPLICATE_DECL` | O mesmo nome em dois arquivos, ou um nome que já é da std | Renomeie o seu |
| `E_DUPLICATE_ROUTE` | Mesmo método + caminho registrado duas vezes | Mude o caminho ou o método de uma das rotas |

## Documentos (`liafc doc check`)

Saem de `liafc doc check doc.typ --dados d.json --json`; detalhes na skill `liaf-pdf`.

| Código | Causa | Correção |
|---|---|---|
| `E_TYPST` | Erro de compilação do Typst (com linha e coluna do `.typ`) | Corrija o `.typ` na posição indicada |
| `W_TYPST` | Aviso do Typst (fonte inexistente, por exemplo) | Use as fontes embutidas: `"Inter"`, `"Libertinus Serif"`, `"DejaVu Sans Mono"` |
| `W_LAYOUT_OUT_OF_PAGE` | Conteúdo saindo da página | Aplique o `fix` sugerido (margem, tamanho) |
| `W_LAYOUT_BLANK_PAGE` | Página em branco no meio | Remova a quebra de página sobrando |
| `W_LAYOUT_EMPTY_TAIL` | Última página quase vazia | Compacte para caber na anterior |
| `W_LAYOUT_LOW_CONTRAST` | Texto sem contraste WCAG AA | Use a cor que o `fix` sugere |

## Conteúdo e postagens (`liafc post`)

| Código | Causa | Correção |
|---|---|---|
| `E_POST_PARSE_ERROR` | Frontmatter malformado ou delimitador `---` ausente | Comece o arquivo com `---`, feche com `---` e use sintaxe chave: valor |
| `E_CONTENT_MISSING_TITLE` | Campo `title` ausente no frontmatter | Adicione `title: "Meu Post"` dentro do cabeçalho `---` |
| `E_CONTENT_MISSING_SLUG` | Campo `slug` ausente e título em branco | Adicione `slug: "meu-artigo"` ou preencha o `title` |
| `E_CONTENT_INVALID_SLUG` | Slug com caracteres inválidos ou maiúsculas | Use apenas letras minúsculas, números e hífens: `slug: "meu-post"` |
| `E_CONTENT_IMAGE_NOT_FOUND` | Imagem de capa declarada em `image:` não existe localmente | Verifique se o arquivo da imagem está presente na pasta |
| `E_CONTENT_EMBED_IMAGE_NOT_FOUND` | Imagem no corpo markdown `![...](img.png)` não existe | Coloque o arquivo de imagem na pasta indicada ou corrija o caminho |
