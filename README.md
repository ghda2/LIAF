# LIAF — Language for AI First

<p align="center">
  <img src="https://raw.githubusercontent.com/ghda2/LIAF/main/logo.png" alt="LIAF Logo" width="180" onerror="this.style.display='none'"/>
</p>

<p align="center">
  <b>A primeira linguagem de programação projetada do zero para agentes autônomos e LLMs.</b><br>
  <i>Sintaxe concisa para economia de tokens, checagem estrita de efeitos, binários autônomos sem runtime externo, motor Typst/PDF embutido e Language Server Protocol nativo.</i>
</p>

<p align="center">
  <a href="#principais-pilares">Pilares</a> •
  <a href="#exemplo-completo">Exemplos</a> •
  <a href="#comandos-da-cli-liafc">CLI Toolchain</a> •
  <a href="#motor-de-documentos-pdf-typst">Motor Typst/PDF</a> •
  <a href="#motor-de-conteúdo-ia-first">Motor de Conteúdo</a> •
  <a href="#integração-com-agentes-de-ia">Agentes & Skills</a> •
  <a href="#documentação">Documentação</a>
</p>

---

## O que é a LIAF?

**LIAF** (*Language for AI First*) é uma linguagem compilada, fortemente tipada e orientada a efeitos, desenhada para resolver o maior gargalo da engenharia de software com Inteligência Artificial: **alucinações de sintaxe, consumo excessivo de tokens BPE e complexidade desnecessária de ambientes de execução.**

Com a LIAF, um modelo de linguagem ou agente autônomo (como Claude Code, Antigravity, Cursor ou agentes customizados) escreve microsserviços, rotas HTTP, bancos de dados, integrações de tempo real e relatórios executivos em PDF com **redução de até 69% no gasto de tokens** em relação a stacks tradicionais (TypeScript/Node, Python, Go puro).

O compilador oficial (`liafc`) emite binários estáticos nativos e ultrarrápidos, sem depender de Node.js, Chromium, Python, LaTeX ou da instalação do compilador no servidor de destino.

---

## Principais Pilares da Versão 1.0.0

- 🤖 **AI-First & Eficiência de Tokens:** Gramática linear sem chaves (`{}`) nem parênteses supérfluos. Reduz o consumo de contexto de LLMs em até 69% e padroniza diagnósticos de erro em JSON legível por máquina (`--json`) para auto-correção determinística em 1 ciclo.
- ⚡ **Sistema Estrito de Efeitos (`effects`):** Funções e rotas declaram exatamente o que acessam (`effects(net, fs, io, db, env)`). Erros não tratados resultam em falha de compilação (`E_UNDECLARED_EFFECT`, `E_UNHANDLED_RESULT`), eliminando surpresas e exceções silenciosas em produção.
- 📄 **Motor de Documentos & PDF Nativo (Typst WASM):** Compilador Typst embutido diretamente no executável via WebAssembly. Renderiza PDFs e prévias PNG de alta fidelidade sem Chromium, headless browser ou dependências do sistema operacional.
- ♿ **Auditoria de Layout & Acessibilidade WCAG AA:** O comando `liafc doc check` audita quebras de linha, páginas em branco e contraste de cores, fornecendo sugestões exatas de ajuste em hexadecimais para o agente de IA.
- 🌐 **Web, Middlewares & Tempo Real:** Suporte nativo de primeira classe para rotas HTTP tipadas, pipeline extensível de middlewares, WebSockets bidirecionais e Server-Sent Events.
- 🗄️ **Banco de Dados & Storage Híbrido:** Drivers nativos para PostgreSQL, MySQL, SQLite e Redis, além de utilitários atômicos para arquivos, processamento de imagens WebP e upload de mídia.
- 📰 **Motor de Conteúdo IA-First (`std/content` + `liafc post`):** Pipeline nativo para sites, blogs e portais jornalísticos gerados por IA, com validação de taxonomia, frontmatter, slugificação automática e upload atômico para servidores LIAF.
- 🛠️ **Language Server Protocol Nativo (`liafc lsp`):** LSP completo integrado via stdin/stdout, garantindo autocompletion, hover tipado, diagnósticos em tempo real e formatação em qualquer editor (VS Code, Cursor, NeoVim).
- 🧪 **Testes Integrados no Compilador:** Suíte de testes nativa com blocos `test "descricao" ... end` e runner `liafc test` com relatórios em formato texto ou JSON.
- 🚀 **Deploy Autônomo para Agentes:** Comandos prontos para orquestração automática: `liafc build --embed`, `liafc publish`, `liafc deploy caddy-bind` e `liafc service install` (systemd).

---

## Exemplo Completo

### API REST com Middlewares, Banco de Dados e Efeitos

```liaf
module api_v1

struct Usuario
  id int
  nome str
  email str
  ativo bool
end

// Middleware de autenticação
fn auth_guard(req Request) Result[Request, Response] effects(net)
  let token = req-header(req, "Authorization")
  if str-len(token) == 0
    return err(json-error(401, "Token de autorização ausente"))
  end
  ok(req)
end

// Rota protegida com injeção de parâmetros e query SQL segura
route GET "/usuarios/{id}" (id int) Response effects(db, net)
  use auth_guard
  on-err e
    json-error(500, e)
  end

  let db = try db-open("postgres", env("DATABASE_URL"))
  let rows = try db-query(db, "SELECT id, nome, email, ativo FROM usuarios WHERE id = $1", [id])
  
  if len(rows) == 0
    return json-error(404, "Usuário não encontrado")
  end

  let row = rows[0]
  let usuario = new Usuario(row.id, row.nome, row.email, row.ativo)
  json-response(200, try json-encode(usuario))
end

// Ponto de entrada do serviço
fn main() void effects(fs, io, net)
  println("Servidor LIAF v1.0.0 escutando em http://0.0.0.0:8080")
  unwrap-or(serve-hybrid("./public", "8080"), false)
end
```

---

## Motor de Documentos (PDF Nativo com Typst)

A LIAF separa rigorosamente **layout visual** (desenhado em sintaxe Typst) de **dados de negócio** (estruturas tipadas em LIAF). O motor compila os dois em PDFs de qualidade editorial sem navegadores headless.

```liaf
fn emitir_cobranca(fatura Fatura) str effects(fs, io)
  on-err e
    println(fmt("Falha na geração do documento: {}", e))
    return ""
  end

  let template = try fs-read-file("templates/cobranca.typ")
  let payload_json = try json-encode(fatura)
  
  // Renderização nativa via Typst WASM embutido
  let pdf_bytes = try pdf-typst(template, payload_json)
  let destino = fmt("build/fatura_{}.pdf", fatura.id)
  
  try fs-write-file(destino, pdf_bytes)
  println(fmt("Fatura salva em {} ({} bytes)", destino, str-byte-len(pdf_bytes)))
  destino
end
```

---

## Motor de Conteúdo IA-First (`std/content`)

Criado especialmente para portais de conteúdo, jornais automatizados e blogs orientados a IA:

```bash
# 1. Validação estrutural do post em Markdown com frontmatter YAML
liafc post conteudo/artigo.md --check

# 2. Publicação instantânea com upload atômico de imagens associadas
liafc post conteudo/artigo.md --url https://meusite.com --token $LIAF_AUTH_TOKEN
```

---

## Comandos da CLI (`liafc`)

| Comando | Descrição |
|---|---|
| `liafc init [dir]` | Inicializa a estrutura de um novo projeto LIAF |
| `liafc new <nome>` | Cria um projeto pronto em um novo diretório |
| `liafc check <arquivo.liaf> [--json]` | Executa análise sintática e tipagem estrita com saída amigável para IA |
| `liafc test [caminho] [--json]` | Executa os testes unitários e de integração nativos da linguagem |
| `liafc run <arquivo.liaf>` | Compila em memória e executa imediatamente |
| `liafc build <arquivo.liaf> [-o bin] [--embed]` | Gera binário estático autocontido pronto para produção |
| `liafc emit <arquivo.liaf>` | Exibe o código Go otimizado gerado internamente |
| `liafc doc check <doc.typ> [--dados d.json]` | Audita páginas, overflow e contraste de acessibilidade WCAG |
| `liafc doc preview <doc.typ> -o p.png` | Gera imagem PNG da primeira página para inspeção visual do agente |
| `liafc post <post.md> [--check] [--url <url>]` | Valida metadados e publica artigos via pipeline de conteúdo |
| `liafc lsp` | Inicia o Language Server Protocol (stdin/stdout) |
| `liafc fmt <arquivo.liaf> [-w]` | Formata o código fonte segundo o padrão canônico |
| `liafc agents [caminho] [--force]` | Exporta as skills completas da LIAF para `.agents/skills` |
| `liafc publish <arquivo> --url <endpoint>` | Envia arquivos e assets para o endpoint de storage do servidor |
| `liafc deploy caddy-bind [flags]` | Gera e recarrega configurações de proxy reverso Caddy com SSL |
| `liafc service install --name <n> --bin <b>` | Cria e habilita serviço daemon no systemd do Linux |
| `liafc version` | Exibe a versão atual da toolchain (`v1.0.0`) |

---

## Compilação e Instalação

### Pré-requisitos
- [Go 1.22+](https://go.dev/) instalado

### Compilar a Toolchain Localmente
```bash
# Clonar o repositório
git clone https://github.com/ghda2/LIAF.git
cd LIAF

# Compilar o binário liafc
go build -o liafc ./cmd/liafc

# Executar a suíte completa de testes
go test ./...
```

---

## Integração com Agentes de IA

A LIAF fornece pacotes de habilidades (*skills*) pré-formatados para agentes inteligentes (Antigravity, Claude Code, Cursor, Windsurf, Roo Code):

```bash
# Exportar as skills diretamente para o seu projeto
liafc agents
```

Isso instala:
- `.agents/skills/liaf/SKILL.md`: Guia definitivo de desenvolvimento, sintaxe e correção de erros.
- `.agents/skills/liaf-pdf/SKILL.md`: Padrões de design para Typst, diagramação de documentos e auditoria WCAG.

---

## Suíte de Testes Nativos

Escreva testes unitários diretamente nos seus arquivos ou em módulos de teste dedicados:

```liaf
test "calculo de frete prioritario"
  let valor = calcular_frete(1500, true)
  assert-eq(valor, 300)
end
```

Execute todos os testes com saída em texto legível ou JSON:
```bash
liafc test
liafc test --json
```

---

## Documentação Completa

| Documento | Descrição |
|---|---|
| [Primeiros Passos](docs/linguagem/GETTING_STARTED.md) | Tutorial prático para escrever sua primeira aplicação em LIAF |
| [Guia para Agentes de IA](docs/linguagem/AI_GUIDE.md) | Regras de ouro, padrões sintáticos e diretrizes para modelos de linguagem |
| [Especificação da Linguagem (SPEC)](docs/linguagem/SPEC.md) | Gramática formal, tipos de dados, sistema de efeitos e semântica |
| [Arquitetura do Compilador](docs/linguagem/ARCHITECTURE.md) | Detalhes do frontend, checker, geração de código e embutimento Typst |
| [Status & Histórico de Versões](docs/STATUS.md) | Registro de todas as 48 issues implementadas até a v1.0.0 |

---

## Licença

Distribuído sob licença aberta. Consulte o arquivo [LICENSE](LICENSE) para mais detalhes.
