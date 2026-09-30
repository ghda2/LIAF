# LIAF — Language for AI First

Linguagem experimental com AST textual em S-expressions, tipos e efeitos explícitos, diagnósticos JSON e compilação via Go. A hipótese de reduzir o custo de programação por modelos menores deve ser medida com benchmarks.

```liaf
(module hello
  (fn main (params) (returns void) (effects io)
    (body (do (println "Hello, LIAF")))))
```

```powershell
go build -o liafc.exe ./cmd/liafc
./liafc.exe check pkg/codegen/testdata/loops.liaf --json
./liafc.exe run pkg/codegen/testdata/collections.liaf
./liafc.exe build app.liaf -o app.exe --embed=public
go test ./...
```

Recursos: loops, listas e mapas tipados, `Result` com `match` e `try`/`on-err`, rotas HTTP declarativas, arquivos com escrita atômica e JSON, concorrência por canais, SSG Markdown, layouts, includes, gzip, ETags, streaming de mídias, publicação autenticada e geração de PDF com o Typst embutido. A compilação pelo backend Go requer Go e este repositório; o executável gerado funciona sem Go instalado.

## Documentos e PDF

O layout do documento é escrito em [Typst](https://typst.app/docs) (arquivo `.typ`) e o programa LIAF
entrega os dados tipados. Não há modelos prontos: a liberdade criativa vem do código. O motor Typst vai
embutido no executável, então em execução não há Rust, binário extra nem rede.

```liaf
let layout = try fs-read-file("recibo.typ")        // o .typ lê os dados com json("/dados.json")
let pdf = try pdf-typst(layout, try json-encode(recibo))
try fs-write-file("recibo.pdf", pdf)                 // numa rota: pdf-response(pdf, "recibo.pdf")
```

Exemplos: `curriculo.liaf` + `curriculo.typ` (raiz) e `examples/cobranca_pix_pdf/` (QR Code via pacote
do Typst Universe). Para conferir um documento enquanto edita:
`liafc doc watch meu.typ --dados dados.json -o previa.png`. Referência completa: SPEC §26.

## Documentação

- [Primeiros passos](docs/linguagem/GETTING_STARTED.md) — compilar, rodar, formatar
- [Guia de agentes](docs/linguagem/AI_GUIDE.md) — o que um modelo precisa saber para escrever LIAF correta
- [Especificação da linguagem](docs/linguagem/SPEC.md) — núcleo v0.2; a seção 18 cobre as adições da v0.3
- [Arquitetura](docs/linguagem/ARCHITECTURE.md) — pipeline do compilador e pacotes
- [Mapa do código](docs/linguagem/SOURCEMAP.md) — onde cada coisa mora no repositório

Exemplo completo em v0.3: [`pkg/codegen/testdata/task_api_v03.liaf`](pkg/codegen/testdata/task_api_v03.liaf) — API REST com
rotas declarativas, `try`/`on-err` e escrita atômica de estado.

O estado do projeto está em [docs/STATUS.md](docs/STATUS.md): o que falta em [docs/a-fazer/](docs/a-fazer/)
e o que já foi entregue em [docs/feito/](docs/feito/). Índice completo: [docs/README.md](docs/README.md).

Publish/reload são desabilitados sem token configurado. Defina `LIAF_DEPLOY_TOKEN` e use POST com `Authorization: Bearer ...`. Assets embutidos são imutáveis; publicação dinâmica requer diretório em disco.
