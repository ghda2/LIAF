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

// Assinar JWT (segredo deve ter no mínimo 32 bytes; payload JSON deve conter "exp" em segundos unix)
// Assinatura: (jwt-sign segredo payload-json)
let token = try jwt-sign("chave-secreta-militar-32-bytes-ok!", "{\"sub\":\"123\",\"role\":\"admin\",\"exp\":4102444800}")

// Validar assinatura e expiração (efeito: clock)
// Assinatura: (jwt-verify segredo token)
let payload = try jwt-verify("chave-secreta-militar-32-bytes-ok!", token)
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

---

## 7. Rate Limiting e Client IP
Proteção contra ataques de força bruta (ex: login) e extração de IP ciente de proxy reverso (`X-Forwarded-For`):

```liaf
// 1. Extrair IP do cliente
let ip = req-client-ip(req)

// 2. Limitar requisições (chave, max_requisicoes, janela_em_segundos)
// Retorna false se o limite foi excedido
if not rate-limit(concat("login:", ip), 5, 60)
  return json-error(429, "Muitas tentativas de login. Tente novamente em 1 minuto.")
end
```

---

## 8. Transações Atômicas de Banco de Dados (`db-transaction`)
Bloco atômico com `BEGIN`, `COMMIT` e rollback automático caso ocorra qualquer erro:

```liaf
let resultado = db-transaction conn
  db-exec(conn, "UPDATE contas SET saldo = saldo - 100 WHERE id = ?1", origem_id)
  db-exec(conn, "UPDATE contas SET saldo = saldo + 100 WHERE id = ?1", destino_id)
  "Transferência concluída"
end
```

---

## 9. Guardas Declarativas de Autenticação
Retorno imediato de 401 Unauthorized via `try`:

```liaf
import "std/auth"

route GET "/api/perfil" (req Request) Response
  // Se o token for ausente, inválido ou expirado, retorna 401 JSON automaticamente
  let payload = try auth-guard-jwt(req, "chave-secreta-militar-32-bytes-ok!")
  let usuario = try json-decode(payload, UsuarioClaims)
  json-response(200, try json-encode(usuario))
end
```

---

## 10. `std/cli` — Argumentos e Linha de Comando
Criação de utilitários de terminal e processamento de flags:

```liaf
import "std/cli"

// Listar argumentos passados ao processo (excluindo o executável)
let args = cli-args-list()
let total = cli-args-len()

// Checar flag booleana (ex: "--verbose" ou "-v")
if (cli-has-flag "--verbose")
  println "Modo detalhado ativo"

// Obter valor de flag com fallback (ex: "--porta=8080" ou "--porta" "8080")
let porta = cli-get-flag "--porta" "8080"

// Sair do processo com código de status
cli-exit 0
```

---

## 11. `std/fs` — Sistema de Arquivos e Pastas
Manipulação de diretórios e arquivos locais:

```liaf
import "std/fs"

// Criar árvore de diretórios (mkdir -p)
try fs-make-dir "dados/backups"

// Listar arquivos dentro de uma pasta
let arquivos = try fs-read-dir "dados"

// Checar se caminho é diretório
if (fs-is-directory "dados/backups")
  println "É uma pasta"

// Tamanho de arquivo em bytes
let bytes = try fs-size "dados/relatorio.pdf"

// Remover arquivo ou pasta recursivamente (rm -rf)
try fs-delete "dados/backups"
```

---

## 12. `std/csv` — Manipulação de Dados CSV
Parser e gerador nativo de CSV:

```liaf
import "std/csv"

// Ler e decodificar CSV em lista de linhas e colunas
let linhas = try csv-decode "nome,preco\nCafe,10\nLeite,5"

// Serializar lista de linhas para texto CSV
let texto = csv-serialize linhas
```

---

## 13. `std/html` — Server-Driven UI e HTML Seguro
Construção declarativa de HTML, fragmentos para swap (HTMX), proteção contra XSS e integração com formulários:

```liaf
import "std/html"

// 1. Criar resposta HTML completa ou parcial
route GET "/componente/hora" () Response
  let seguro = html-safe "<script>alert('xss')</script>"
  let card = html-elem "div" "class=\"card\"" (concat "Horário: " (str-from-int now-ms()))
  html-render(200, card)
end

// 2. Manipulação de formulários (Server-Driven UI com HTMX)
route POST "/tarefas/nova" (req Request) Response
  let titulo = match request-form(req, "titulo")
    case ok(t)
      t
    case err(_)
      "Sem título"
  end

  let fragmento = fmt("<div class=\"item\">✓ {}</div>", html-safe(titulo))
  html-render(200, fragmento)
end
```

### Arquitetura Recomendada com Arquivos Separados:
* `app.liaf`: Backend com rotas de API e endpoints que devolvem fragmentos HTML renderizados.
* `public/index.html`: Casca inicial da aplicação, carregando HTMX (`htmx.min.js`) e disparando requisições com `hx-get`, `hx-post`, `hx-target` e `hx-swap`.
* `public/style.css`: Estilização pura e desacoplada.
* Inicialização com `serve-hybrid("./public", porta)` para servir arquivos estáticos em alta velocidade via RAM cache e despachar rotas dinâmicas.
* Veja exemplos completos e funcionais em `examples/demo_app` (básico) e `examples/demo_pro` (avançado com SQLite, WebSockets e SVG).



