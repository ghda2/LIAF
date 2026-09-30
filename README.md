# LIAF — Language for AI First

LIAF é uma linguagem de programação moderna desenvolvida especificamente para modelos de inteligência artificial (LLMs) e agentes autônomos escreverem código correto, conciso e de alta performance no menor custo de tokens possível.

O código compila para binários autônomos ultrarrápidos via Go, trazendo um runtime seguro, biblioteca padrão integrada e motor nativo de renderização de documentos (Typst WASM).

```liaf
module api_tarefas

struct Tarefa
  id int
  titulo str
  concluida bool
end

route GET "/tarefas/{id}" (id int) Response effects(io)
  let tarefa = new Tarefa(id, "Aprender LIAF", true)
  json-response(200, tarefa)
end

fn main() void effects(io)
  println("Servidor LIAF rodando em http://127.0.0.1:8080")
end
```

---

## Principais Pilares

- **AI-First & Eficiência de Tokens:** Sintaxe linear concisa (redução de até 69% em tokens BPE frente a formatos tradicionais) e diagnósticos com saída em JSON (`--json`) pensados para auto-cura mecânica imediata por agentes de IA.
- **Motor de Documentos & PDF Nativo:** Renderizador [Typst](https://typst.app) embutido no binário via WebAssembly (Wazero). Gera PDFs e prévias PNG sem necessidade de Chromium, NodeJS ou binários externos no sistema.
- **Auditoria Automática de Layout (WCAG AA):** Com o comando `liafc doc check`, o motor detecta overflow, páginas em branco e mede o contraste das cores, sugerindo a correção exata em hexadecimais para modelos de linguagem.
- **Web & Dados Integrados:** Rotas HTTP declarativas, WebSockets bidirecionais, drivers nativos para PostgreSQL, MySQL e Redis, e suporte a escrita atômica em arquivos e JSON.
- **Compilação Autônoma:** Binário gerado (`liafc build`) é 100% estático e autocontido, pronto para deploy sem requerer a instalação do compilador na máquina de produção.

---

## Instalação e Compilação

Para compilar a toolchain `liafc` a partir do repositório:

```bash
# Compilar o compilador (liafc)
go build -o liafc.exe ./cmd/liafc

# Executar a suíte de testes
go test ./...
```

---

## Comandos da CLI (`liafc`)

| Comando | Descrição |
|---|---|
| `liafc init [caminho]` | Inicializa um novo projeto LIAF no diretório |
| `liafc check <arquivo.liaf> [--json]` | Valida sintaxe, contratos de efeitos e tipos com retorno amigável para IA |
| `liafc run <arquivo.liaf>` | Compila em memória e executa imediatamente |
| `liafc build <arquivo.liaf> [-o binario]` | Gera executável nativo autônomo |
| `liafc doc check <doc.typ> --dados d.json` | Valida layout, páginas e contraste de acessibilidade do documento |
| `liafc doc preview <doc.typ> -o p.png` | Gera prévia em imagem da 1ª página do documento |
| `liafc doc watch <doc.typ> -o p.png` | Observa alterações em tempo real e regrava a prévia |
| `liafc fmt <arquivo.liaf> -w` | Formata o código fonte no padrão canônico |

---

## Geração de Documentos em PDF

Em LIAF, o layout visual é escrito em **Typst** (`.typ`) e os dados de negócio são fornecidos de forma tipada pelo programa via `/dados.json`.

```liaf
fn emitir_recibo() void effects(fs, io)
  on-err e
    println(fmt("erro ao gerar: {}", e))
    exit(1)
  end

  let layout = try fs-read-file("recibo.typ")
  let dados_json = try json-encode(new Recibo(1042, "30/09/2026", "Café Central", 2500))
  let pdf = try pdf-typst(layout, dados_json)

  try fs-write-file("recibo.pdf", pdf)
  println(fmt("recibo.pdf gerado com sucesso ({} bytes)", str-byte-len(pdf)))
end
```

Exemplos inclusos no repositório:
- [curriculo.liaf](curriculo.liaf) + [curriculo.typ](curriculo.typ): Currículo executivo diagramado.
- [relatorio.liaf](examples/relatorio_lambari/relatorio.liaf) + [relatorio.typ](examples/relatorio_lambari/relatorio.typ): Boletim climatológico semanal de Lambari/MG.
- [cobranca.liaf](examples/cobranca_pix_pdf/cobranca.liaf) + [cobranca.typ](examples/cobranca_pix_pdf/cobranca.typ): Boleto e cobrança com QR Code Pix gerado dinamicamente.

---

## Documentação

- [Guia de Primeiros Passos](docs/linguagem/GETTING_STARTED.md) — Como configurar e rodar seu primeiro programa.
- [Guia para Agentes de IA](docs/linguagem/AI_GUIDE.md) — Referência técnica e regras de ouro para LLMs escreverem LIAF.
- [Especificação da Linguagem (SPEC)](docs/linguagem/SPEC.md) — Gramática completa, tipos primitivos, efeitos e builtins.
- [Arquitetura do Compilador](docs/linguagem/ARCHITECTURE.md) — Pipeline interno do frontend, checker e emissores de código.
- [Status do Projeto](docs/STATUS.md) — Roadmap, histórico de releases e issues concluídas/arquivadas.
