# Biblioteca padrão da LIAF (`std/`)

As 84 funções públicas dos 13 módulos que vêm embutidos no `liafc`. Diferente dos
[builtins](builtins.md), cada módulo precisa de `import` no topo do arquivo:

```text
import "std/auth"
import "std/jwt"
```

Os dois programas no fim da página usam **todas** as funções, e o teste `TestSkillExamples` do
repositório da LIAF os compila e roda os blocos `test`.

- **Nomes reservados:** não declare funções com o prefixo de um módulo importado
  (`auth-...`, `cors-...`). O nome já pertence à std e dá `E_DUPLICATE_DECL`.
- **Imports da própria std:** um módulo da std só importa outros da std
  (`E_STD_RELATIVE_IMPORT`).

## `std/auth` — autenticação

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `auth-basic(req)` | `(result BasicAuth str)` | — | Lê `Authorization: Basic ...`. `BasicAuth` tem os campos `user` e `password` |
| `auth-validate-basic(creds, usuario, senha)` | `bool` | — | Compara em tempo constante |
| `auth-bearer-token(req)` | `(result str str)` | — | O token de `Authorization: Bearer ...` (aceita `bearer` minúsculo) |
| `auth-api-key(req, header)` | `(result str str)` | — | Chave de um header próprio, ex.: `"X-API-Key"` |
| `auth-validate-api-key(recebida, esperada)` | `bool` | — | Tempo constante |
| `auth-cookie(req, nome)` | `(result str str)` | — | Valor do cookie |
| `auth-format-cookie(c)` | `str` | — | Monta o `Set-Cookie` de uma struct `Cookie(name, value, max_age, path, domain, secure, http_only, same_site)` |
| `auth-hash-password(senha)` | `(result str str)` | `rand` | Argon2id; senha com menos de 8 caracteres é `err` |
| `auth-verify-password(senha, hash)` | `bool` | — | Senha ou hash vazios são `false` |
| `auth-jwt-bearer(req, segredo)` | `(result str str)` | `clock` | Bearer + `jwt-verify`; devolve o payload JSON |
| `auth-guard-jwt(req, segredo)` | `(result str Response)` | `clock` | Igual, mas o erro já é uma `Response` 401 pronta para devolver |

## `std/jwt` — JSON Web Token (HS256)

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `jwt-sign(segredo, payload)` | `(result str str)` | — | `payload` é JSON (monte com `json-encode` de uma struct) e **precisa** ter `exp` (segundos desde 1970) |
| `jwt-verify(segredo, token)` | `(result str str)` | `clock` | Confere assinatura e expiração; devolve o payload para `json-decode` |

O segredo precisa ter pelo menos 32 bytes; os dois recusam um segredo curto. Leia-o de
`env-get`, nunca do código.

## `std/cookie` — cookies

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `cookie-make(nome, valor, max_age, path, secure, http_only, same_site)` | `str` | — | Linha de `Set-Cookie`; `same_site` é `"Lax"`, `"Strict"` ou `"None"` |
| `cookie-expire(nome, path)` | `str` | — | `Set-Cookie` que apaga o cookie (`Max-Age=0`) |
| `response-set-cookie(res, cookie)` | `Response` | — | Acrescenta o cookie sem apagar os outros |
| `response-delete-cookie(res, nome, path)` | `Response` | — | Atalho de `response-set-cookie(res, cookie-expire(...))` |

## `std/cors` — CORS com lista de origens

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `cors-origin(req, permitidas)` | `(result str str)` | — | A origem da requisição, se estiver na `(list str)`; nunca há `"*"` |
| `cors-headers(res, origem)` | `Response` | — | `Access-Control-Allow-Origin` + `Vary: Origin` |
| `cors-preflight(origem, metodos, headers)` | `Response` | — | Resposta 204 ao `OPTIONS`; listas separadas por vírgula |
| `cors-headers-credentials(res, origem)` | `Response` | — | Igual, com `Allow-Credentials` (navegador mandando cookie) |
| `cors-preflight-credentials(origem, metodos, headers)` | `Response` | — | Preflight com `Allow-Credentials` |

O preflight precisa de uma `route OPTIONS` para cada caminho.

## `std/html` — Server-Driven UI

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `html-render(status, html)` | `Response` | — | O mesmo que `html-response` |
| `html-safe(texto)` | `str` | — | O mesmo que `html-escape` |
| `html-elem(tag, atributos, conteudo)` | `str` | — | `<tag atributos>conteudo</tag>`. **Não escapa** `conteudo` nem `atributos`: passe dado do usuário por `html-safe` |
| `html-fragment(id, conteudo)` | `str` | — | `<div id="id">conteudo</div>`, para trocar um pedaço da página (HTMX) |

## `std/seo` — SEO, sitemap e RSS

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `seo-meta-tags(meta)` | `str` | — | `<title>`, description, canonical, OpenGraph e Twitter Card. `MetaTags(title, description, canonical, image, site_name, author, twitter_handle, article_date, locale)` |
| `seo-jsonld-article(meta, editora, logo)` | `str` | — | JSON-LD `Article` do Schema.org |
| `seo-jsonld-website(nome, url)` | `str` | — | JSON-LD `WebSite` |
| `seo-robots-txt(url_sitemap, permitir)` | `str` | — | Conteúdo do `robots.txt` |
| `seo-sitemap-start()`, `seo-sitemap-url(loc, lastmod, changefreq, priority)`, `seo-sitemap-end()` | `str` | — | Junte com `concat` para montar o `sitemap.xml` |
| `seo-rss-start(titulo, descricao, link)`, `seo-rss-item(titulo, link, descricao, data)`, `seo-rss-end()` | `str` | — | Feed RSS 2.0; `data` no formato RFC 1123 |

## `std/storage` — upload e entrega de imagens

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `storage-init(dir)` | `(result bool str)` | `fs` | Cria a pasta |
| `storage-upload-image(req, campo, dir)` | `(result str str)` | `fs` | Lê o arquivo do campo, converte para WebP sem perdas e grava; devolve o nome |
| `storage-upload-image-opt(req, campo, dir, max_l, max_a, max_bytes, nivel)` | `(result str str)` | `fs` | Com redimensionamento, limite de bytes e nível de compressão (0–9); `0` = sem limite |
| `storage-serve-image(dir, nome)` | `Response` | `fs` | `image/webp` com cache longo; ausente responde 404 |
| `storage-delete-image(dir, nome)` | `(result bool str)` | `fs` | |
| `storage-image-exists(dir, nome)` | `bool` | `fs` | |

O nome gravado é o nome original com `.webp`: **dois uploads com o mesmo nome se
sobrescrevem**. Os nomes são saneados contra `../`.

## `std/cli` — linha de comando

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `cli-args-list()` | `(list str)` | — | Argumentos, sem o executável |
| `cli-args-len()` | `int` | — | |
| `cli-get-arg(i, padrao)` | `str` | — | Argumento posicional ou o padrão |
| `cli-has-flag(flag)` | `bool` | — | Aceita `--flag` e `--flag=valor` |
| `cli-get-flag(flag, padrao)` | `str` | — | Aceita `--flag=valor` e `--flag valor` |
| `cli-exit(codigo)` | `void` | `io` | |

## `std/csv`

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `csv-decode(texto)` | `(result (list (list str)) str)` | — | O mesmo que o builtin `csv-parse` |
| `csv-serialize(linhas)` | `str` | — | O mesmo que `csv-encode` |

## `std/fs` — nomes alternativos

| Função | Retorno | Efeito | Equivale a |
|---|---|---|---|
| `fs-make-dir(dir)` | `(result bool str)` | `fs` | `fs-mkdir-all` |
| `fs-delete(caminho)` | `(result bool str)` | `fs` | `fs-remove-all` |
| `fs-read-dir(caminho)` | `(result (list str) str)` | `fs` | `fs-list-dir` |
| `fs-is-directory(caminho)` | `bool` | `fs` | `fs-is-dir` |
| `fs-size(caminho)` | `(result int str)` | `fs` | `fs-file-size` |

## `std/regex` — validação sem `result`

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `regex-is-match(padrao, texto)` | `bool` | — | Padrão inválido vira `false` |
| `regex-first(padrao, texto)` | `(option str)` | — | |
| `regex-extract-all(padrao, texto)` | `(list str)` | — | |
| `regex-sub(padrao, texto, troca)` | `str` | — | Padrão inválido devolve o texto intacto |
| `is-email(s)`, `is-slug(s)`, `is-numeric(s)`, `is-uuid(s)` | `bool` | — | |
| `is-cpf(s)`, `is-cnpj(s)` | `bool` | — | **Só o formato** (com ou sem pontuação); não confere os dígitos verificadores |

## `std/time` — datas

| Função | Retorno | Efeito | Notas |
|---|---|---|---|
| `time-now()`, `time-now-unix()`, `time-now-ms()` | `str` / `int` / `int` | `clock` | Os mesmos dos builtins |
| `time-parse(iso)` | `(result int str)` | — | O mesmo que `time-parse-iso` |
| `time-format(ts, layout)` | `str` | — | Layout do Go (`"2006-01-02"`) |
| `time-add(ts, s)`, `time-add-minutes`, `time-add-hours`, `time-add-days` | `int` | — | |
| `time-diff(a, b)` | `int` | — | `a - b` em segundos |
| `time-is-past(ts)`, `time-is-future(ts)` | `bool` | `clock` | |
| `time-year`, `time-month`, `time-day`, `time-hour`, `time-minute`, `time-second`, `time-weekday(ts)` | `int` | — | UTC; `time-weekday` 0 = domingo |

## Exemplo: módulos web (`auth`, `jwt`, `cookie`, `cors`, `html`, `seo`, `storage`)

```liaf
module std_web

import "std/auth"
import "std/cookie"
import "std/cors"
import "std/html"
import "std/jwt"
import "std/seo"
import "std/storage"

// Nao ha constantes de topo: um segredo fixo vira uma fn. Em producao, leia de env-get.
fn segredo() str
  "segredo-de-exemplo-com-32-bytes-ou-mais!"
end

// Corpos de resposta e claims: sempre struct + json-encode, nunca JSON com fmt.
struct Valor
  valor str
end

struct Contagem
  total int
end

struct Claims
  sub str
  exp int
end

// --- std/auth ---------------------------------------------------------------

route GET "/basic" (req Request) Response
  on-err motivo
    json-error(401, motivo)
  end
  let creds = try auth-basic(req)
  if !auth-validate-basic(creds, "admin", "senha-forte")
    return json-error(401, "credenciais invalidas")
  end
  json-response(200, try json-encode(new Valor(creds.user)))
end

route GET "/api-key" (req Request) Response
  on-err motivo
    json-error(401, motivo)
  end
  let chave = try auth-api-key(req, "X-API-Key")
  if !auth-validate-api-key(chave, "chave-esperada")
    return json-error(403, "chave invalida")
  end
  json-response(200, "{}")
end

route GET "/bearer" (req Request) Response
  on-err motivo
    json-error(401, motivo)
  end
  let token = try auth-bearer-token(req)
  json-response(200, try json-encode(new Contagem(str-len(token))))
end

route GET "/sessao" (req Request) Response
  on-err motivo
    json-error(401, motivo)
  end
  let sid = try auth-cookie(req, "sid")
  json-response(200, try json-encode(new Valor(sid)))
end

// JWT: auth-jwt-bearer devolve o payload ou um erro str;
// auth-guard-jwt devolve o payload ou ja uma Response 401 pronta.
route GET "/perfil" (req Request) Response effects(clock)
  on-err motivo
    json-error(401, motivo)
  end
  let payload = try auth-jwt-bearer(req, segredo())
  json-response(200, payload)
end

route GET "/admin" (req Request) Response effects(clock)
  match auth-guard-jwt(req, segredo())
    case ok(payload)
      json-response(200, payload)
    case err(negado)
      negado
  end
end

// --- std/cookie --------------------------------------------------------------

route POST "/login" () Response effects(rand)
  let sid = random-token()
  let res = json-response(200, "{\"ok\":true}")
  response-set-cookie(res, cookie-make("sid", sid, 86400, "/", true, true, "Lax"))
end

route POST "/logout" () Response
  response-delete-cookie(json-response(200, "{}"), "sid", "/")
end

// --- std/cors ----------------------------------------------------------------

fn origens() (list str)
  list("https://painel.exemplo.com")
end

route GET "/api/dados" (req Request) Response
  let res = json-response(200, "[]")
  match cors-origin(req, origens())
    case ok(origem)
      cors-headers(res, origem)
    case err(_)
      res
  end
end

route OPTIONS "/api/dados" (req Request) Response
  match cors-origin(req, origens())
    case ok(origem)
      cors-preflight(origem, "GET, POST", "Authorization, Content-Type")
    case err(_)
      json-error(403, "origem nao permitida")
  end
end

// Com cookie (fetch com credentials: "include").
route OPTIONS "/api/conta" (req Request) Response
  match cors-origin(req, origens())
    case ok(origem)
      cors-preflight-credentials(origem, "GET", "Content-Type")
    case err(_)
      json-error(403, "origem nao permitida")
  end
end

route GET "/api/conta" (req Request) Response
  let res = json-response(200, "{}")
  match cors-origin(req, origens())
    case ok(origem)
      cors-headers-credentials(res, origem)
    case err(_)
      res
  end
end

// --- std/html ----------------------------------------------------------------

route GET "/saudacao" (req Request) Response
  let nome = unwrap-or(request-query(req, "nome"), "visitante")
  let item = html-elem("li", "class=\"novo\"", html-safe(nome))
  html-render(200, html-fragment("lista", item))
end

// --- std/storage -------------------------------------------------------------

route POST "/fotos" (req Request) Response effects(fs)
  on-err motivo
    json-error(400, motivo)
  end
  let nome = try storage-upload-image(req, "foto", "uploads")
  json-response(201, try json-encode(new Valor(nome)))
end

route POST "/fotos/miniatura" (req Request) Response effects(fs)
  on-err motivo
    json-error(400, motivo)
  end
  let nome = try storage-upload-image-opt(req, "foto", "uploads", 320, 320, 5000000, 6)
  json-response(201, try json-encode(new Valor(nome)))
end

route GET "/fotos/{nome}" (nome str) Response effects(fs)
  storage-serve-image("uploads", nome)
end

route DELETE "/fotos/{nome}" (nome str) Response effects(fs)
  if !storage-image-exists("uploads", nome)
    return json-error(404, "nao encontrada")
  end
  let _ = storage-delete-image("uploads", nome)
  json-response(200, "{}")
end

// --- testes das funcoes puras ------------------------------------------------

test "senhas, cookies e chaves"
  let hash = unwrap-or(auth-hash-password("senha-forte-123"), "")
  assert-true(auth-verify-password("senha-forte-123", hash))
  assert-false(auth-verify-password("outra", hash))
  assert-false(str-len(unwrap-or(auth-hash-password("curta"), "")) > 0)
  assert-true(auth-validate-api-key("abc", "abc"))
  assert-true(auth-validate-basic(new BasicAuth("ana", "x"), "ana", "x"))
  let c = new Cookie("sid", "v", 3600, "/", "", true, true, "Strict")
  assert-true(str-contains(auth-format-cookie(c), "HttpOnly"))
  assert-eq(cookie-make("a", "1", 60, "/", true, true, "Lax"), "a=1; Path=/; Max-Age=60; HttpOnly; Secure; SameSite=Lax", "cookie-make")
  assert-eq(cookie-expire("a", "/"), "a=; Path=/; Max-Age=0; HttpOnly; SameSite=Lax", "cookie-expire")
end

test "jwt"
  let exp = time-now-unix() + 3600
  let claims = unwrap-or(json-encode(new Claims("ana", exp)), "")
  let token = unwrap-or(jwt-sign(segredo(), claims), "")
  assert-true(str-contains(unwrap-or(jwt-verify(segredo(), token), ""), "ana"))
  assert-false(str-len(unwrap-or(jwt-verify("outro-segredo-com-32-bytes-ou-mais!!", token), "")) > 0)
end

test "html e seo"
  assert-eq(html-safe("<x>"), "&lt;x&gt;", "html-safe")
  assert-eq(html-elem("p", "", "oi"), "<p>oi</p>", "html-elem sem atributos")
  assert-true(str-contains(html-fragment("caixa", "oi"), "id=\"caixa\""))
  let meta = new MetaTags("Titulo", "Descricao", "https://ex.com/a", "https://ex.com/a.webp", "Site", "Ana", "@site", "2026-10-17", "pt_BR")
  assert-true(str-contains(seo-meta-tags(meta), "<title>Titulo</title>"))
  assert-true(str-contains(seo-jsonld-article(meta, "Site", "https://ex.com/logo.png"), "\"@type\""))
  assert-true(str-contains(seo-jsonld-website("Site", "https://ex.com"), "WebSite"))
  assert-true(str-contains(seo-robots-txt("https://ex.com/sitemap.xml", true), "Sitemap:"))
  let sitemap = concat(seo-sitemap-start(), seo-sitemap-url("https://ex.com/a", "2026-10-17", "weekly", "0.8"), seo-sitemap-end())
  assert-true(str-contains(sitemap, "<loc>https://ex.com/a</loc>"))
  let rss = concat(seo-rss-start("Blog", "Novidades", "https://ex.com"), seo-rss-item("Post", "https://ex.com/p", "Resumo", "Sat, 17 Oct 2026 10:00:00 GMT"), seo-rss-end())
  assert-true(str-contains(rss, "<item>"))
end

test "storage em disco"
  assert-true(unwrap-or(storage-init("tmp_std_storage"), false))
  assert-false(storage-image-exists("tmp_std_storage", "nada.webp"))
  let _ = fs-remove-all("tmp_std_storage")
end

fn main() void effects(fs, net)
  unwrap-or(serve-hybrid("./public", "8080"), false)
end
```

## Exemplo: utilitários (`cli`, `csv`, `fs`, `regex`, `time`)

```liaf
module std_util

import "std/cli"
import "std/content"
import "std/csv"
import "std/fs"
import "std/regex"
import "std/time"

test "std/cli"
  assert-true(cli-args-len() >= 0)
  assert-eq(list-len(cli-args-list()), cli-args-len(), "cli-args-list")
  assert-eq(cli-get-arg(99, "padrao"), "padrao", "cli-get-arg com indice ausente")
  assert-false(cli-has-flag("--nunca-passada"))
  assert-eq(cli-get-flag("--saida", "out.txt"), "out.txt", "cli-get-flag: --saida=valor ou o padrao")
end

test "std/csv"
  match csv-decode("a,b\n1,2\n")
    case ok(linhas)
      assert-eq(list-len(linhas), 2, "csv-decode")
      assert-eq(csv-serialize(linhas), "a,b\n1,2\n", "csv-serialize")
    case err(e)
      assert(false, e)
  end
end

test "std/fs"
  assert-true(unwrap-or(fs-make-dir("tmp_std_fs/sub"), false))
  assert-true(fs-is-directory("tmp_std_fs/sub"))
  let _ = fs-write-file("tmp_std_fs/a.txt", "1234")
  assert-eq(unwrap-or(fs-size("tmp_std_fs/a.txt"), 0), 4, "fs-size")
  assert-eq(list-len(unwrap-or(fs-read-dir("tmp_std_fs"), make-list(str))), 2, "fs-read-dir")
  assert-true(unwrap-or(fs-delete("tmp_std_fs"), false))
  assert-false(fs-exists("tmp_std_fs"))
end

test "std/regex"
  assert-true(regex-is-match("^[a-z]+$", "liaf"))
  assert-eq(unwrap-or(regex-first("[0-9]+", "v1.25"), ""), "1", "regex-first")
  assert-eq(list-len(regex-extract-all("[0-9]+", "1-22-333")), 3, "regex-extract-all")
  assert-eq(regex-sub("\\s+", "a   b", " "), "a b", "regex-sub")
  assert-true(is-email("ana@exemplo.com"))
  assert-false(is-email("ana@"))
  assert-true(is-slug("meu-post-2026"))
  assert-true(is-numeric("12345"))
  assert-true(is-uuid("123e4567-e89b-12d3-a456-426614174000"))
  assert-true(is-cpf("529.982.247-25"))
  assert-true(is-cnpj("11.222.333/0001-81"))
end

test "std/time"
  let ts = unwrap-or(time-parse("2026-01-31T10:00:00Z"), 0)
  assert-eq(time-format(time-add-days(ts, 1), "2006-01-02"), "2026-02-01", "time-add-days")
  assert-eq(time-diff(time-add-hours(ts, 2), ts), 7200, "time-add-hours e time-diff")
  assert-eq(time-minute(time-add-minutes(ts, 5)), 5, "time-add-minutes")
  assert-eq(time-second(time-add(ts, 9)), 9, "time-add soma segundos")
  assert-eq(time-year(ts), 2026, "time-year")
  assert-eq(time-month(ts), 1, "time-month")
  assert-eq(time-day(ts), 31, "time-day")
  assert-eq(time-hour(ts), 10, "time-hour")
  assert-eq(time-weekday(ts), 6, "time-weekday")
  assert-true(time-is-past(ts))
  assert-true(time-is-future(time-now-unix() + 3600))
  assert-true(time-now-ms() > 0)
  assert-true(str-len(time-now()) > 0)
end

test "std/content"
  let slug = slugify("Deploy Instantâneo")
  assert-eq(slug, "deploy-instantaneo", "slugify")
  let html = render-markdown("**texto em negrito**")
  assert-true(str-contains(html, "<strong>texto em negrito</strong>"))
  let raw = "---\ntitle: \"Meu Artigo\"\n---\nConteudo"
  match parse-post(raw)
    case ok(p)
      assert-eq(p.title, "Meu Artigo", "parse-post title")
      assert-eq(p.slug, "meu-artigo", "parse-post slug")
    case err(e)
      assert(false, e)
  end
end

fn main() void effects(io)
  if cli-has-flag("--ajuda")
    println("uso: app --saida=arquivo")
    cli-exit(0)
  end
end
```
