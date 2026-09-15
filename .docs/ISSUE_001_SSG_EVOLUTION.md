# Issue #001: Evolução da LIAF Web Engine — Superando SSGs Tradicionais (Hugo/Astro)

**Status:** Aprovado para Planejamento  
**Componente:** `pkg/web`, `pkg/markdown`, `cmd/liafc`  
**Data:** 15 de setembro de 2026  

---

## 1. Contexto e Objetivo
A LIAF Web Engine já provou sua eficiência com servidor em RAM (< 3 MB de RAM), layouts modulares em memória, fingerprinting automático de assets e hot-reload atômico em 3 ms. 

O objetivo desta issue é evoluir o motor para que funcione como uma plataforma estática e híbrida superior a geradores como Hugo e Astro:
- Zero dependências externas de runtime (gerador + servidor em binário único).
- Suporte a conteúdo Markdown com Frontmatter em RAM.
- Coleções dinâmicas e loops de templates (`<!-- for item in posts -->`).
- Opção de empacotamento completo em binário único (`liafc pack` / `embed.FS`).
- Auto-TLS / Let's Encrypt nativo no `serve_site`.
- Capacidade de rotas de API híbridas.

---

## 2. Especificações Técnicas

### 2.1. Markdown + Frontmatter em RAM
- Leitura de arquivos `.md` na pasta do site (`public/` ou `content/`).
- Extração de Frontmatter YAML/JSON entre delimitadores `---`:
  ```markdown
  ---
  title: "Meu Primeiro Post"
  date: "2026-09-15"
  layout: "base.html"
  ---
  # Título
  Texto do artigo...
  ```
- Conversão de Markdown para HTML no boot do VFS.
- Injeção transparente no slot `<!-- content -->` do layout correspondente.

### 2.2. Sistema de Coleções e Loops em Templates
- Suporte a fontes de dados em `public/data/*.json` ou metadados de posts `.md`.
- Sintaxe declarativa nos templates HTML:
  ```html
  <!-- for post in posts -->
    <article>
      <h2><a href="{{ post.url }}">{{ post.title }}</a></h2>
      <time>{{ post.date }}</time>
    </article>
  <!-- endfor -->
  ```
- Resolução e interpolação em memória RAM no momento da compilação de assets.

### 2.3. Single-Binary Deploy (`liafc pack`)
- Adição da flag/comando `liafc build --embed` ou `liafc pack`.
- Incorporação de todos os assets (`public/`) no executável Go via `embed.FS`.
- No runtime, o `serve_site` detecta se os arquivos estão embutidos ou em disco, eliminando a necessidade de transferir a pasta `public/` via SCP.

### 2.4. Auto-TLS Nativo
- Integração de `golang.org/x/crypto/acme/autocert` no `pkg/web/autotls.go`.
- Quando `serve_site(dir, "meudominio.com", "443", true)` for chamado, o servidor obtém e renova certificados SSL automaticamente.

### 2.5. Suporte Híbrido (SSG + Endpoints de API)
- Suporte a manipuladores de rotas customizadas em LIAF ou Go dentro do mesmo processo.

---

## 3. Critérios de Aceite
1. Um site com páginas `.md` com frontmatter renderiza corretamente com layout e URLs limpas.
2. Template com `<!-- for item in ... -->` renderiza listagens a partir de dados JSON ou artigos Markdown.
3. Testes unitários para Markdown, Frontmatter e Template Loops cobrem 100% dos novos componentes.
4. Consumo de memória RAM do binário final permanece inferior a 5 MB.
5. Hot-reload via `POST /_liaf/reload` continua funcionando em < 5 ms com Markdown e dados recarregados.
