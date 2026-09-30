# Biblioteca Padrão (`std/`)

A biblioteca padrão da LIAF é embutida diretamente no executável do compilador e pode ser importada via `(import "std/<modulo>")` ou `import "std/<modulo>"`.

---

## 1. `std/cookie` — Cookies HTTP Seguros
Gerencia emissão e exclusão de cookies HTTP com proteção contra XSS e CSRF:

```liaf
import "std/cookie"

// Criar cookie seguro (HttpOnly, Secure, SameSite=Lax)
let c = cookie-make("sid", "token123", 86400, "/", true, true, "Lax")

// Adicionar à resposta sem sobrescrever outros cookies
let resp = response-set-cookie(json-response(200, "{\"ok\":true}"), c)

// Ler cookie da requisição
let sid = try request-cookie(req, "sid")

// Remover cookie do cliente (Max-Age=0)
let logout_resp = response-delete-cookie(resp, "sid", "/")
```

---

## 2. `std/auth` — Autenticação Web e Senhas
```liaf
import "std/auth"

// Extrair token Bearer
let token = try auth-bearer-token(req)

// Extrair credenciais Basic Auth
let creds = try auth-basic-credentials(req) // retorna struct BasicCredentials

// Hash e verificação de senha com Argon2id
let hash = try auth-hash-password("minha-senha")
let ok = auth-verify-password("minha-senha", hash)
```

---

## 3. `std/jwt` — Tokens JWT (HMAC-SHA256)
```liaf
import "std/jwt"

// Assinar JWT (segredo deve ter no mínimo 32 bytes)
let token = try jwt-sign("{\"sub\":\"123\",\"role\":\"admin\"}", "chave-secreta-militar-32-bytes-ok!", 3600)

// Validar assinatura e expiração
let payload = try jwt-verify(token, "chave-secreta-militar-32-bytes-ok!")
```

---

## 4. `std/cors` — Proteção de Origem e Preflight
```liaf
import "std/cors"

let origens = (list "https://painel.exemplo.com")

// Rota comum com CORS
let origem = try cors-origin(req, origens)
let resp = cors-headers(json-response(200, "dados"), origem)

// Com suporte a credenciais (cookies)
let resp_cookie = cors-headers-credentials(json-response(200, "dados"), origem)
```

---

## 5. `std/storage` — Uploads e Otimização WebP
```liaf
import "std/storage"

// Salvar imagem convertendo para WebP sem perdas e limitando a 1200px
let meta = try storage-save-image(req, "foto", "./uploads", 1200, 1200)

// Upload de arquivo binário genérico
let arquivo = try storage-save-file(req, "documento", "./uploads", "contrato.pdf")
```

---

## 6. `std/seo` — Otimização para Motores de Busca e Metatags
Gera meta tags completas (OpenGraph, Twitter Cards, Canonical), Schema.org JSON-LD e Sitemaps:

```liaf
import "std/seo"

// 1. Criar metadados da página / artigo
let meta = new MetaTags(
  "Guia Completo da LIAF",
  "Aprenda a construir APIs e blogs ultra-rápidos.",
  "https://meusite.com/artigo",
  "https://meusite.com/capa.webp",
  "Meu Blog",
  "Gabriel Santos",
  "@gabriel",
  "2026-09-27",
  "pt_BR"
)

// 2. Renderizar bloco de tags HTML <head> (OpenGraph + Twitter Cards + Canonical)
let head_tags = seo-meta-tags(meta)

// 3. Renderizar Schema.org JSON-LD estruturado para o Google
let schema_json = seo-jsonld-article(meta, "Meu Blog", "https://meusite.com/logo.webp")

// 4. Gerar robots.txt e sitemaps dinâmicos
let robots = seo-robots-txt("https://meusite.com/sitemap.xml", true)
```

> **Dica SEO Automático:** Servidores `serve-hybrid` geram `/sitemap.xml` e `/robots.txt` automaticamente a partir dos arquivos HTML e posts Markdown caso não sejam definidos manualmente em `public/`.

