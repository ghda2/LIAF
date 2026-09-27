# Issue #031 — Guia das próximas issues (depois da v0.4)

**Estado:** Aberta. **Criada em:** 26/09/2026. **Tipo:** planejamento.

> **Progresso (26/09/2026, noite):** item 0.2 feito (commits `c7d5ff6` e `64cf5d4`). Item 0.1 feito:
> [#027](../feito/ISSUE_027_SEGURANCA_WEB_E_STD.md) criada e casos renomeados para `027_*`. A linha
> "Compilar fora do repositório" da seção B virou a [#032](ISSUE_032_COMPILADOR_AUTONOMO_RUNTIME_EMBUTIDO.md).
> O harness do benchmark (A) saiu do repositório — ver o aviso na [#009](ISSUE_009_AI_FIRST_BENCHMARK_AND_SHOWCASE.md).

Esta issue não implementa nada. Ela lista o que falta, diz **como** cada lacuna deve virar uma issue
própria e em que ordem, e define a decisão que vem antes de todas: medir se a tese da LIAF se
sustenta. Contexto completo em `docs/logs/LOG_2026-09-26.md`.

---

## 1. Onde estamos

No nicho escolhido (**API de backend**), a LIAF saiu do básico em 26/09/2026:

| Capacidade | Antes | Agora |
|---|---|---|
| Saber quem está chamando | não lia headers | Bearer, cookie, query, `std/auth` |
| Login com senha | — | argon2id com parâmetros da OWASP |
| Sessão / token | — | `random-token`, `std/jwt` |
| Isolamento entre clientes | — | testado ponta a ponta (`pedidos_api`) |
| Frontend em outro domínio | — | `std/cors` com lista de origens |
| Tempo real autenticado | sem auth | WebSocket com token no handshake |
| Chamar APIs externas | — | `http-fetch` |
| Webhooks assinados | — | HMAC + `secure-eq` |
| Banco sob carga | vazava e travava | 0 erros, memória estável |
| Reaproveitar código | um arquivo só | `import` + `std/` embutida |

Como **linguagem de propósito geral**, ainda não (SPEC §10.2). E a pergunta que decide o projeto —
*um modelo de IA acerta mais em LIAF do que em Python?* — continua sem resposta.

---

## 2. Regras para criar cada issue nova

1. **Número:** o próximo livre na tabela de `docs/STATUS.md`. Confira também os prefixos de
   `conformance/basics/`: **o prefixo dos casos de aceitação é o número da issue**, e não pode ser
   reaproveitado. (Em 26/09 isso foi violado: `028_jwt_*` e `029_campo_reservado` ocupam números que
   não são deles — ver item 0.1.)
2. **Registrar** a issue na tabela do `STATUS.md` no mesmo commit em que ela é criada.
3. **Seções mínimas**, no formato das issues existentes:
   - Estado, data de criação, componentes afetados;
   - Contexto: o problema, com um exemplo real (de preferência de `examples/` ou do delivery);
   - Proposta: a forma na linguagem (sintaxe, tipos, efeitos, códigos de diagnóstico novos);
   - Fora do escopo;
   - Casos de aceitação: `conformance/basics/<número>_*.liaf`;
   - Critério de conclusão.
4. **Todo recurso novo de linguagem declara o impacto em tokens.** Reescreva um exemplo existente
   com o recurso e meça antes/depois (script de contagem em `docs/logs/LOG_2026-09-26.md`, §3.2). A tese
   da LIAF é custo por programa correto; um recurso que aumenta o custo precisa se justificar.
5. **Testes exigidos para dar como concluída:** unitários no runtime; o checker recusando o uso
   errado com o código de diagnóstico certo; casos de conformidade compilados e executados; teste
   ponta a ponta quando envolver rede ou banco. Servidores de teste escutam em `127.0.0.1`
   (`LIAF_HOST`), para não abrir o firewall do Windows.
6. **Documentar** na SPEC (seção nova numerada) e no `docs/linguagem/AI_GUIDE.md` na mesma entrega. Todo
   exemplo de código da documentação precisa passar no `liafc check`.

---

## 3. Issues a criar

Prioridade: **P0** decide o rumo; **P1** bloqueia casos reais; **P2** importante; **P3** quando houver
tempo.

### 0. Arrumação (antes de tudo)

| # | Issue a criar | Por quê |
|---|---|---|
| 0.1 | **#027 — Segurança web e biblioteca padrão** (retroativa) | Registrar o que foi entregue em 26/09 (headers/query, crypto, `std/` embutida, `std/auth`/`jwt`/`cors`, `http-fetch`, palavras reservadas como campo) e **renomear** `028_jwt_*` e `029_campo_reservado` para `027_*`, liberando os prefixos da #028 e de uma futura #029 |
| 0.2 | — (sem issue) | **Commitar** o trabalho de 26/09 separado por frente, antes de qualquer outra mudança |

### A. Decisão — P0

Não criar issue nova: **executar a #009** (benchmark) com o recorte abaixo.

- **Tarefa:** a mesma API (pedidos com login, isolamento por restaurante) pedida em LIAF e em Python.
  O par `examples/workspace_ws.liaf` / `.py` já serve de referência.
- **Modelos:** um pequeno e um grande.
- **Medir:** pass@1, tentativas até passar, tokens e custo até o sucesso.
- **Regra de decisão**, registrada **antes** de rodar (SPEC §15.1):
  - LIAF com custo até o sucesso menor → seguir a ordem da seção 4 (linguagem geral).
  - LIAF pior → priorizar #028/#012 (redução de tokens) e rodar de novo antes de crescer a linguagem.

### B. Linguagem de propósito geral

| Issue a criar | Prio | Impacto | Depende de | Observação |
|---|---|---|---|---|
| **Funções como valor / closures** | P1 | `map`/`filter` com função, callbacks, e é a base do middleware | — | Definir se a closure captura por valor ou referência e como os efeitos da função passada entram na assinatura de quem a recebe |
| **Genéricos do usuário** | P2 | Bibliotecas tipadas; hoje `jwt-verify` devolve JSON cru | closures (idealmente) | Começar por funções genéricas com restrição explícita; structs genéricos depois |
| **Tipos soma (enums)** | P1 | `match` sobre estados de domínio (status do pedido) | — | Com checagem de exaustividade no `match`; hoje só `result` e `option` |
| **Funções privadas em módulos** | P1 | Helpers da `std/` e de módulos vazam e ocupam nomes | — | Hoje há só a convenção de prefixo (`auth-`); ver loader (`E_DUPLICATE_DECL`) |
| **Testes escritos em LIAF** | P1 | Quem usa a linguagem não consegue testar o próprio código | — | Ex.: `(test "nome" ...)` + `liafc test`; asserções com diagnóstico JSON |
| **Compilar fora do repositório** | P1 | `liafc build/run` exigem o código-fonte do projeto | — | **Estende a #010**: embutir o runtime Go no `liafc` (como a `std/`) e gerar um módulo temporário. Não exige o backend C |
| **Ferramentas: LSP e `fmt` com comentários** | P2 | Erros e autocompletar no editor; hoje o `fmt` apaga comentários | — | Duas issues separadas; o `fmt` antes (é o que impede exemplos na forma canônica) |

### C. Backend completo

| Issue a criar | Prio | Caso real | Depende de | Observação |
|---|---|---|---|---|
| **Tipo de bytes** | P1 | Imagem, PDF, `base64url-decode` de binário | — | Hoje `str` é texto UTF-8 e o `http-fetch` recusa corpo binário |
| **Upload multipart** | P1 | Foto do prato | bytes | Limite de tamanho fixo e tipo verificado pelo conteúdo, não pela extensão |
| **Data e hora** | P1 | "Pedido às 19:30", horário de funcionamento | — | Formatar, somar, fuso horário; efeito `clock` só para ler o relógio |
| **Middleware** | P1 | CORS, log e autenticação repetidos em cada rota | closures | Resolve também o preflight de CORS (hoje uma rota `OPTIONS` por caminho) |
| **Regex** | P2 | CEP, telefone, e-mail | — | Padrão literal compilado e validado pelo checker; motor sem backtracking (RE2) |
| **Tarefas em segundo plano / agendamento** | P2 | Cancelar pedido não pago em 15 min | — | Relacionar com a #017 (tempo real e resiliência); definir o que acontece num restart |
| **Rodar em mais de uma instância** | P3 | Redundância do delivery | — | SQLite e pub/sub em memória prendem a uma máquina; ver #017 e o driver PostgreSQL |

### D. Já existentes que continuam valendo

#009 (benchmark), #010 (backend nativo), #012/#028 (tokens), #017 (tempo real),
#030 (modularização — fazer antes de mexer em `parser`, `checker` e `codegen`).

---

## 4. Ordem sugerida

```
0.2 commits ─► 0.1 issue #027 e renumeração ─► A. benchmark (#009)
                                                   │
                        ┌──────────────────────────┴───────────────────────┐
                 LIAF ganha                                          LIAF perde
                        │                                                  │
   #030 modularização                                         #028/#012 tokens
   closures ─► middleware                                     benchmark de novo
   tipos soma, funções privadas, testes em LIAF
   compilar fora do repositório (#010)
   bytes ─► upload; data/hora; regex
   genéricos; LSP; agendamento; várias instâncias
```

A #030 entra antes das mudanças de linguagem porque closures, tipos soma e genéricos mexem justamente
em `parser.go`, `checker.go` e `codegen.go`, os três maiores monólitos.

---

**Concluída quando:** o item 0.1 estiver resolvido, o benchmark (A) tiver rodado com a regra de decisão
registrada antes, e cada linha das seções B e C tiver virado uma issue numerada no `STATUS.md` — ou
tiver sido descartada com o motivo anotado aqui.
