# Próximos passos da LIAF

**Atualizado em 26/09/2026.** O estado de cada issue está em [STATUS.md](../STATUS.md). O raciocínio
completo da ordem abaixo está na [#031](ISSUE_031_GUIA_PROXIMAS_ISSUES.md).

## A pergunta que decide o rumo

*Um modelo de IA escreve programas corretos mais barato em LIAF do que em Python?* Ainda não há
resposta. O único dado comparável (`workspace_ws`, 26/09) é desfavorável: **LIAF +64% de tokens**.
Crescer a linguagem antes de responder arrisca investir numa forma que depois vai mudar (#028 ou #033).

## Ordem

### 1. Commitar o que está pendente

Revisão da std (#027), reorganização dos docs e o trabalho em `std/storage` / `pkg/runtime/image.go`,
cada frente no seu commit. O `storage` ainda não tem issue: criar a #029 para ele.

### 2. Fechar a #028 (Concluído)

- [x] `liafc fmt` imprime a forma compacta (sem `fields`, `params`, `returns`, `body`, com retorno implícito).
- [x] Documentar a forma compacta, `fmt`, `unwrap-or` e `match` como valor na `docs/linguagem/SPEC.md` e no `.agents/skills/liaf/SKILL.md`.
- [x] Sintaxe compacta canônica integrada e validada em todos os testes.

### 3. Rodar o benchmark — #009 (Concluído)

- [x] Comparação LIAF vs Python em consumo de tokens realizada e registrada.

### 4. Depois do resultado

**Se a LIAF ganhar:** #030 (modularizar `parser`, `checker` e `codegen` antes de mexer neles) →
closures → middleware → tipos soma, funções privadas, testes em LIAF → ~~#032 (compilar fora do
repositório)~~ (feito em 27/09) → bytes, upload, data/hora, regex.

**Se perder:** atacar tokens primeiro — ~~resto da #012 (`s.campo`)~~ (feito em 27/09), `on-err` padrão por módulo, e decidir
se a #033 (sintaxe sem parênteses) vale a troca. Medir de novo antes de crescer.

### A qualquer momento (não dependem do benchmark)

Itens de biblioteca e runtime, sem mudar a linguagem:

- ~~Pendências de autenticação da [#027](../feito/ISSUE_027_SEGURANCA_WEB_E_STD.md): `request-cookie`,
  `std/cookie`, `response-add-header` para o `Vary`~~ — Concluídas em 27/09.
- ~~Validar a #015 contra bancos reais (PostgreSQL, MySQL com `caching_sha2_password`, Redis)~~ — Concluída em 27/09 via Docker.
- #030, item 1: `web/cache.go` tem 88% de código duplicado — é risco, não só desconforto.

## Cuidados com o ambiente

- Windows/PowerShell, Go 1.26, GCC MinGW. Python é `py -3`; o comando `python` é o alias da Microsoft
  Store e não funciona.
- `go build ./...` não atualiza o `liafc.exe`. Use `go build -o liafc.exe ./cmd/liafc`.
- Python no Windows grava CRLF: abra arquivos com `newline=""`.
- PowerShell 5.1 remove aspas duplas de argumentos para programas nativos: escape como `'{\"a\":1}'`.
- Servidores de teste escutam em `127.0.0.1` (`LIAF_HOST`), para o firewall não abrir janela.
- `.env` nunca vai para commit nem para saída de ferramenta.
- Não marcar issue como concluída só porque o código existe: registrar os testes e o que ficou sem
  verificar.
