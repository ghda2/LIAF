---
title: "LIAF SSG: Superando o Hugo em RAM e Velocidade"
date: "2026-09-15"
author: "LIAF Core Team"
layout: "base.html"
description: "Como a LIAF Web Engine une gerador estático e servidor web em milissegundos."
---
# LIAF SSG: Superando o Hugo em RAM e Velocidade

Bem-vindo ao **LIAF SSG**, a nova geração de motores web para sistemas estáticos e híbridos!

## Por que LIAF é diferente?

Diferente de geradores tradicionais como **Hugo** ou **Astro**, que dependem de etapas de build demoradas gravando arquivos em disco e necessitando de um servidor Nginx ou Caddy externo:

- **100% em Memória RAM:** Todo o processamento de Markdown, Frontmatter e layouts ocorre no boot em RAM.
- **Zero I/O de Disco:** Nenhum arquivo HTML intermediário precisa ser gravado em disco durante a execução.
- **Fingerprinting Automático:** Hashes SHA-1 nos assets com cache busting imutável (*zero Ctrl+F5*).
- **Single-Binary Deploy:** Com `liafc build --embed`, você tem um único binário com todo o site dentro!

### Exemplo de Servidor Web em LIAF

```liaf
[fn main () -> (void)
  (serve_site "./public" "" "7070" false)
/fn main]
```

> "Simplicidade, consumo mínimo de RAM (~2.4 MB) e ausência total de dependências externas."
