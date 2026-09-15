# Aprendizados e Validação em Produção (LIAF Web Engine)

**Data do Teste**: 15/09/2026  
**Domínio Testado**: `tt.webdrop.bio` (com Cloudflare Proxy)  
**Host de Teste**: VPS Linux Ubuntu x86_64 (`srv1961313`)

---

## 1. Eficiência de Recursos e Execução Estática

- **Binário Único Puro**: Cross-compilação com `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` gerou um executável ELF estático de ~10.6 MB sem dependência de bibliotecas C dinâmicas.
- **Consumo Real de Memória**: O serviço rodou em produção consumindo apenas **3.4 MB de RAM**.
- **Densidade Extrema**: Esse footprint permite hospedar centenas de sites/microserviços LIAF independentes em uma VPS modesta de 1GB de RAM.

---

## 2. Pipeline de Performance em RAM (Zero Disk I/O)

- **Minificação no Boot**: HTML, CSS e JS foram minificados no momento do carregamento, eliminando 100% dos comentários e espaços desnecessários.
- **Cache e Gzip**: Os assets foram pré-comprimidos com Gzip na memória, entregando o CSS em apenas 276 bytes.
- **Headers de Borda**:
  - `x-powered-by: LIAF Web Engine`
  - `Cache-Control: public, max-age=31536000, immutable`
  - `ETag` com suporte nativo a respostas `304 Not Modified`.
  - Integração instantânea com cache e proxy do Cloudflare.

---

## 3. Modelo de Arquitetura Multi-Porta / Micro-Sites

- A abordagem ideal para produção é **multi-tenant por portas**: cada site LIAF roda como um microserviço autônomo em sua própria porta local (`7070`, `7071`, `7072`...).
- Um reverse proxy de borda (como o Caddy ou Nginx da VPS) recebe o domínio público e repassa para a porta interna correspondente.
- Isso evita disputas de portas e permite reiniciar, atualizar ou substituir um site sem afetar os outros.

---

## 4. Nuances de Rede em Ambientes Docker + Host

- Em VPS com redes customizadas do Docker Compose, o gateway padrão para o host não é necessariamente `172.17.0.1` (rede padrão `docker0`), mas sim a subnet da rede do compose (no teste, `172.18.0.1` na rede `inoxlink_default`).
- O firewall UFW bloqueia por padrão conexões vindas da bridge do Docker para portas locais do host, exigindo a regra explícita `ufw allow from 172.18.0.0/16 to any port <PORTA>`.
