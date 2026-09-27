# Issue #007: Pipeline de Deploy Autônomo e Otimizado para IA

**Status:** Implementada e testada localmente; validacao remota pendente
**Componente:** `pkg/web`, `cmd/liafc`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
O fluxo de deploy atual exige múltiplos passos manuais da IA (cross-compilação, envio de pasta via SCP, criação manual de serviço systemd e edição de Caddyfile).

Esta issue simplifica o ciclo para um processo autônomo de alta velocidade:
1. **Single-Binary Embed (`liafc pack`):** Assets embutidos no executável eliminando a necessidade de transferir a pasta `public/` via SCP.
2. **HTTP Publish API (`POST /_liaf/publish`):** Envio atômico de novas páginas e updates diretamente via HTTP autenticado (sem SSH/SCP).
3. **Auto-Instalação de Serviço (`liafc service install`):** Binário registra e inicia seu próprio serviço systemd no host com 1 flag.
4. **Integração com Caddy Admin API:** Comando para registrar rotas de proxy no Caddy sem editar arquivos de configuração.

---

## 2. Especificações Técnicas

### 2.1. HTTP Publish API em RAM
- Endpoint seguro: `POST /_liaf/publish`
- Headers:
  - `Authorization: Bearer <LIAF_DEPLOY_TOKEN>`
  - `X-LIAF-Path: /sobre.html` (ou caminho relativo de asset)
- Comportamento:
  - Salva o arquivo no VFS em RAM e sincroniza no disco.
  - Recalcula hashes de fingerprinting e revalida templates em ~3ms.
  - Retorna JSON: `{"status": "published", "path": "/sobre", "reload_ms": 2.8}`.

### 2.2. Self-Install Service (systemd)
- Comando CLI: `./liaf_server --install-service [nome] [porta]`
- Gera automaticamente `/etc/systemd/system/[nome].service` com restart automático e executa `systemctl enable --now [nome]`.

### 2.3. Caddy Admin API Helper
- Helper na CLI para registrar proxy reverso via `http://localhost:2019/config/`:
  - `liafc deploy caddy-bind --domain tw.webdrop.bio --upstream 172.17.0.1:7070`

---

## 3. Tarefas de Implementação
- [ ] Implementar endpoint seguro `/_liaf/publish` em `pkg/web/server.go` com validação de token e hot-reload atômico.
- [ ] Adicionar suporte a token de deploy configurável via variável de ambiente `LIAF_DEPLOY_TOKEN`.
- [ ] Implementar comando CLI `--install-service` para systemd em Linux.
- [ ] Adicionar testes automatizados para a Publish API em `pkg/web/server_test.go`.
- [ ] Validar deploy no servidor de produção com `tw.webdrop.bio`.


Contrato atual e ajustes de sintaxe: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
