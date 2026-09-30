# Issue #034 — Motor de documentos: Typst embutido via WebAssembly

**Estado:** Implementada (base). **Criada em:** 29/09/2026. **Tipo:** runtime + biblioteca.
**Componentes:** `engines/typst` (Rust → WASM), `pkg/typst` (motor em Go), `pkg/typst/fonts`,
`pkg/typst/lib`, `pkg/docrt`, `pkg/builtins/table_doc.go`, `pkg/codegen` (import condicional),
`runtime_embed.go`, `scripts/build-typst-wasm.sh`.
**Issues filhas:** [#035](ISSUE_035_DECLARACAO_DOC.md) (sintaxe `doc`),
[#036](ISSUE_036_BIBLIOTECA_DE_DOCUMENTOS.md) (temas, componentes e modelos),
[#037](ISSUE_037_QUALIDADE_E_LACO_DO_AGENTE.md) (diagnóstico, prévia e score).

---

## 1. Contexto

Todo backend de negócio precisa emitir documento: recibo, cobrança Pix, relatório, contrato,
currículo. Antes desta issue um programa LIAF não tinha como gerar PDF. As alternativas eram ruins:

| Caminho | Problema |
|---|---|
| HTML + Chrome headless | ~150 MB de dependência, centenas de ms por PDF, quebra de página imprevisível |
| `typst` por subprocesso | Segundo binário no deploy (contra a #032), dados sem tipo, erro em texto livre |
| Motor próprio do zero | Meses para chegar à qualidade do Typst |

## 2. Decisão: reaproveitar o Typst, não reescrever

O Typst é código aberto (**Apache-2.0**), escrito em Rust, com qualidade tipográfica de nível
LaTeX. Ele foi compilado para **`wasm32-wasip1`** e roda dentro do Go pelo **wazero** (runtime WASM
em Go puro). Resultado:

- quem usa a LIAF **não precisa de Rust**: o `.wasm` vai pronto dentro do `liafc`;
- continua **sem CGO e sem binário externo**: o executável gerado carrega o motor sozinho;
- o layout, a tipografia, as fórmulas, as grades e o desenho vêm prontos do Typst. O esforço da
  LIAF vai para o que o Typst não oferece (tipos, diagnóstico JSON, integração com rotas, modelos
  brasileiros, laço de correção do agente).

Rust só é necessário para **recompilar o motor** (`scripts/build-typst-wasm.sh`), ao atualizar o
Typst ou mudar `engines/typst`.

## 3. Fase 0 — prova de conceito (medida em 29/09/2026)

Typst 0.15.1, Rust 1.98.1, wazero 1.12.0, Windows 11, Go 1.26.

| Medida | Resultado |
|---|---|
| Tamanho do motor | 25,0 MB de WASM; **8,9 MB** com gzip (embutido assim) |
| Fontes | Fora do WASM, em pacotes Go separados. Inter com 6 faces: ~2,5 MB |
| Primeira carga num processo novo | 7,2 s sem cache; **0,46 s** com o cache de compilação do wazero (`<UserCacheDir>/liaf/wazero`) |
| PDF de uma página simples | 3–4 ms; 1 ms repetido (memoização do Typst entre chamadas) |
| Currículo completo (`curriculo.liaf`) | Executável inteiro, da partida ao PDF + PNG gravados: **0,79 s** |
| Tamanho do PDF do currículo | 65 KB (fontes com subconjunto) |
| Binário de um app LIAF | 12 MB sem documentos; 28 MB com só a Inter; **33 MB com os 4 pacotes de fonte** (+21 MB, só em quem usa `pdf-*`) |
| Currículo com dados novos a cada chamada | ~30–48 ms (o CLI oficial do Typst leva ~40 ms no mesmo documento: empate) |
| Determinismo | Mesma entrada → mesmo SHA-256 (testado) |
| Concorrência | 6 renderizações simultâneas em pool de instâncias (testado) |

O critério da fase 0 (viável para servidor) foi atingido. O custo real é o tamanho do binário;
ver §7.

## 4. O que foi entregue

### 4.1 `engines/typst` (Rust)

Biblioteca `cdylib` com uma ABI mínima: `liaf_alloc/free`, `liaf_add_font`, `liaf_set_file`,
`liaf_clear_files`, `liaf_render` (pedido JSON), `liaf_result_*`, `liaf_output_*` e `liaf_evict`.
O `World` do Typst **não enxerga disco nem rede**: tudo o que o documento lê é entregue pelo host.
Pacotes `@preview` retornam erro claro. As saídas são PDF, PNG por página ou `check` (só
diagnósticos). O PDF sai marcado (acessível) por padrão, aceita padrões (`a-2b`, `ua-1`...) e tem o
`/ID` estável quando recebe `ident`.

### 4.2 `pkg/typst` (Go)

- `Engine`: compila o motor uma vez e mantém um **pool de instâncias** (padrão: 2). Uma instância
  que falha é descartada e recriada. A memoização antiga é descartada a cada 32 usos, para a memória
  de um servidor não crescer sem limite.
- `Job` / `Result`: arquivos virtuais, formato, PPI, padrões, `ident`, data fixa para
  `datetime.today()`. Os diagnósticos vêm com arquivo, linha, coluna e dicas.
- `typst.Default()`: motor compartilhado pelo processo, usado pelos builtins.
- `fonts`: pacotes de fonte embutidos à parte (hoje: Inter, OFL).
- `lib`: biblioteca Typst da LIAF montada em `/liaf/` (ver #036).

### 4.3 Builtins

| Builtin | Assinatura | O que faz |
|---|---|---|
| `pdf-template` | `(modelo str, dados_json str) (result str str)` | Aplica um modelo da biblioteca aos dados |
| `pdf-typst` | `(fonte str, dados_json str) (result str str)` | Documento Typst livre; dados em `/dados.json`, biblioteca em `/liaf/` |
| `png-template` | `(modelo str, dados_json str) (result str str)` | Prévia PNG da 1ª página (144 ppi) |
| `pdf-response` | `(pdf str, nome str) Response` | Resposta HTTP `application/pdf`, `inline`, nome sanitizado |

Os bytes trafegam como `str`, como já acontece em `image-*` e `storage-*`. O `str-len` conta
caracteres, não bytes: use `str-byte-len`. Isso reforça a necessidade do tipo `bytes` (#031, seção C).

### 4.4 Import condicional no codegen

Um builtin agora pode declarar `GoImport`. O codegen só importa `liaf/pkg/docrt` se o programa
usar `pdf-*`, e é isso que mantém os +16 MB fora dos apps que não geram documento
(`TestDocImportOnlyWhenUsed`). Os fontes do motor foram adicionados ao `runtime_embed.go` para que
`liafc build` funcione fora do repositório (`TestRuntimeFSCoversRuntimeDeps` cobre `pkg/docrt`).

### 4.5 Rodada 2: "qualquer documento Typst" dentro do binário (29/09/2026)

Comparado o motor embutido com o compilador oficial do Typst (subprocesso). A velocidade empata e o
oficial traria três atritos: executável gravado em disco, um binário por sistema e menos isolamento.
Decisão: **manter o WASM** e fechar as três lacunas de recursos.

| Lacuna | Solução |
|---|---|
| Fontes | Pacotes Go separados: Inter, **Libertinus Serif**, **New Computer Modern Math** (fórmulas) e **DejaVu Sans Mono** (código). `TestSourceFontsCoverTypst` exige compilar fórmula, serifa e código **sem aviso** |
| Imagens e arquivos | `pdf-template-files` e `pdf-typst-files` recebem `(map str str)` nome → bytes. Os nomes são validados: nada de `..`, `\`, nem sobrescrever `/main.typ`, `/dados.json` ou `/liaf/`. Limite de 64 MB. O `cv-moderno` ganhou `foto` |
| Pacotes do Typst Universe | O `liafc build`/`run` procura `@ns/nome:versão` no código gerado, no `.liaf` e nos `.typ` da pasta, baixa com cache (`<UserCacheDir>/liaf/typst-packages`), resolve as dependências entre pacotes, confere o **SHA-256 em `liaf-typst.lock`** e embute em `typstpkgs/` com um `init()` que registra no runtime. O motor aceita arquivos `@ns/nome:versão/...`; em execução nada acessa a rede. Pacotes com plugin WASM (ex.: `tiaoma`, QR/código de barras via zint) funcionam |

Exemplo: `examples/cobranca_pix_pdf/`, uma cobrança Pix com QR Code via `@preview/tiaoma:0.3.0`.
Lição registrada no `.typ`: texto para copiar (Pix copia e cola) precisa de `hyphenate: false` e
`lang: "en"`. Em pt, o Typst repete o hífen na linha seguinte e corrompe o código colado.

### 4.6 Exemplo

`curriculo.liaf` na raiz: structs tipadas → `json-encode` → `pdf-template("cv-moderno", ...)` →
`curriculo.pdf` + `curriculo.png`.

## 5. Testes

- `pkg/typst`: PDF, determinismo, diagnóstico com posição, arquivo de dados, PNG, pacote
  indisponível e concorrência.
- `pkg/docrt`: todo modelo renderiza os dados de exemplo e é determinístico; dados mínimos; modelo
  desconhecido; *path traversal* no nome do modelo; fonte livre com a biblioteca; erro com posição;
  headers da resposta. Com `LIAF_DOC_OUT=<pasta>`, grava PDF e PNG para inspeção visual.
- `pkg/codegen`: `TestDocImportOnlyWhenUsed`.
- `pkg/typst/packages`: citações, dependências entre pacotes, cache sem rede, trava divergente e tar com `../` (servidor falso).
- `pkg/docrt`: fontes (fórmula/serifa/mono sem aviso), imagem e SVG, foto no currículo, nomes de arquivo proibidos, pacote registrado e pacote ausente.

## 6. Fora do escopo desta issue

- Sintaxe própria de documento na LIAF → [#035](ISSUE_035_DECLARACAO_DOC.md).
- Novos modelos, QR Pix, código de barras, temas → [#036](ISSUE_036_BIBLIOTECA_DE_DOCUMENTOS.md).
- `liafc doc`, score de qualidade e laço de correção → [#037](ISSUE_037_QUALIDADE_E_LACO_DO_AGENTE.md).
- Ler, editar ou assinar PDFs existentes; formulários AcroForm; assinatura ICP-Brasil.

## 7. Pendências e riscos

- **Tamanho:** +21 MB por app com documentos (motor + 4 pacotes de fonte) e ~+16 MB no `liafc`. Próximo passo óbvio: embutir só os pacotes de fonte que o documento usa. Opções a medir: `wasm-opt -Oz`
  (o script já usa quando instalado) e tirar do motor o que o backend não usa (realce de sintaxe,
  exportação HTML/SVG).
- **Artefato no git:** `pkg/typst/wasm/liaf_typst.wasm.gz` (8,9 MB) é versionado. Se o histórico
  pesar, mover para Git LFS ou para download no build com checksum fixo.
- **CI:** falta um job que recompile o WASM e confira se ele bate com o versionado
  (build reprodutível).
- **Primeira carga sem cache** (7 s): num contêiner novo, pré-aquecer o cache no build da imagem
  ou medir o compilador `interpreter` do wazero como alternativa.
- **Licenças:** incluir o NOTICE do Typst (Apache-2.0) e o OFL da Inter na distribuição do `liafc`.

## 8. Critério de conclusão

Base entregue (§4, §5). Fica concluída quando os itens de §7 sobre CI, licenças e tamanho tiverem
sido resolvidos ou registrados como decisão, e a SPEC, o `AI_GUIDE.md` e a skill documentarem os
builtins `pdf-*`.
