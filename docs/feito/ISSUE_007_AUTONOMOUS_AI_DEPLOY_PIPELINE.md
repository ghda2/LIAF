# Issue #007: Pipeline de Deploy Autônomo e Otimizado para IA

**Status:** Concluída  
**Componente:** `pkg/web`, `pkg/deploy`, `cmd/liafc`  
**Data:** 17 de outubro de 2026  

---

## 1. Contexto e Objetivo

O ciclo de deploy automatizado para agentes autônomos de IA foi estruturado em quatro pilares para eliminar passos manuais (SSH, SCP, edição manual de arquivos de configuração de servidores web):
1. **Single-Binary Embed (`liafc build --embed`):** Assets e arquivos estáticos embutidos no executável eliminando a necessidade de transferir pastas de assets separadamente.
2. **HTTP Publish API (`POST /_liaf/publish`):** Envio atômico de novas páginas e updates diretamente via HTTP autenticado (Bearer Token) com recálculo imediato de VFS e cache em RAM.
3. **Auto-Instalação de Serviço (`liafc service install`):** Binário registra e inicia seu próprio serviço systemd no host com 1 flag (`--dry-run` para inspeção ou execução direta no Linux).
4. **Integração com Caddy Admin API (`liafc deploy caddy-bind`):** Comando para registrar rotas de proxy reverso no Caddy dinamicamente via API HTTP sem reiniciar o daemon ou editar arquivos de texto.

---

## 2. O que foi entregue e validado

- Endpoint de publicação `POST /_liaf/publish` com validação de token `LIAF_DEPLOY_TOKEN`, proteção estrita contra Directory Traversal e atualização atômica de cache (`pkg/web/admin.go`).
- Endpoint de recarga a quente `POST /_liaf/reload` sem downtime.
- Gerador de units systemd seguras (`pkg/deploy/deploy.go`).
- Conector REST para a API de Administração do Caddy com controle de concorrência via ETags (`pkg/deploy/deploy.go`).
- Comandos CLI integrados em `cmd/liafc`:
  - `liafc service install --name <n> --bin <b> [--dry-run]`
  - `liafc deploy caddy-bind --domain <d> --upstream <u>`
  - `liafc publish <arquivo> --url <endpoint> [--token <t>]`

---

## 3. Evidências de Testes

- `TestAdminAuthenticationAndLimits`: validação de segurança de rotas admin, autenticação Bearer, rejeição de path traversal e limite de tamanho.
- `TestConcurrentPublishAndThreshold`: validação de publicação simultânea com concorrência sob carga, ranges parciais (206) e recarga atômica em RAM.
- `TestUnitValidation`: validação de escape e geração de units systemd para Linux.
- `TestCaddyBinding`: validação de fluxo HTTP PATCH na Caddy Admin API com ETag If-Match e roteamento reverso.
