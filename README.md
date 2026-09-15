# LIAF — Language for AI First

> Uma linguagem de programação estruturada, estática e de alta concorrência desenhada desde o primeiro byte para a mecânica de Transformers, Tokenizadores e Auto-Cura de IAs.

---

## 🚀 Destaques

- **Otimizada para Transformers**:
  - Tags de fechamento nomeadas (`[fn nome ... /fn nome]`) eliminam o problema de contagem de parênteses/chaves (`}}}` ou `)))`).
  - Precedência prefixada explícita (`(add a b)`).
  - Diagnósticos estruturados em **JSON** prontos para auto-cura por LLM (`liafc check --json`).
- **Concorrência de Alta Performance (CSP/Atores)**:
  - Green threads nativas (`spawn`) e canais fortemente tipados (`chan[T]`, `send`, `recv`).
- **Web Engine Autônomo Embutido**:
  - Auto-minificação de HTML, CSS e JS em memória (zero disk I/O).
  - Compressão Gzip automática e validação de cache HTTP (`ETag`, `304 Not Modified`).
  - Auto-TLS / ACME (Let's Encrypt estilo Caddy) de fábrica.
  - Testado em produção com footprint de apenas **3.4 MB de RAM**.
- **Binário Único Estático**:
  - Compila para executáveis nativos e independentes (Windows, Linux, macOS).

---

## 🛠️ Como Usar

### 1. Compilar o compilador LIAF (`liafc`)
```bash
go build -o liafc.exe ./cmd/liafc
```

### 2. Verificar Sintaxe e Auto-Cura de IA (JSON)
```bash
./liafc check ./examples/broken.liaf --json
```

### 3. Subir um Site com 1 Linha de Código
```liaf
# examples/web_engine.liaf
[fn main () -> (void)
  (serve_site "./public" "" "8080" false)
/fn main]
```

Compilar e gerar binário estático:
```bash
./liafc build ./examples/web_engine.liaf -o meu_site.exe
./meu_site.exe
```

---

## 📚 Documentação

- [`.docs/SPEC.md`](.docs/SPEC.md) — Especificação formal da gramática e tipos.
- [`.docs/ARCHITECTURE.md`](.docs/ARCHITECTURE.md) — Pipeline do compilador e componentes modulares.
- [`.docs/ROADMAP.md`](.docs/ROADMAP.md) — Fases de maturidade da linguagem.
- [`.docs/LEARNINGS.md`](.docs/LEARNINGS.md) — Relatório e métricas do teste em produção.
