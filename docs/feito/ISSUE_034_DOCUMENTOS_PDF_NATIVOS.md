# Issue #034 — Motor de documentos: Typst embutido via WebAssembly

**Estado:** Concluída em 30/09/2026. **Criada em:** 29/09/2026.
**Tipo:** runtime.
**Componentes:** `engines/typst` (Rust → WASM), `pkg/typst` (motor em Go), `pkg/typst/fonts`,
`pkg/typst/packages`, `pkg/docrt`, `pkg/builtins/table_doc.go`, `pkg/codegen` (import condicional),
`cmd/liafc` (pacotes, `--doc-fonts`), `runtime_embed.go`, `scripts/build-typst-wasm.sh`,
`.github/workflows/typst-wasm.yml`, `THIRD_PARTY_NOTICES.md`.
**Relacionadas:** [#035](../arquivo/ISSUE_035_DECLARACAO_DOC.md) (descartada),
[#037](ISSUE_037_QUALIDADE_E_LACO_DO_AGENTE.md) (`liafc doc`, laço de correção).
A #036 (modelos prontos) foi descartada: ver [arquivo/](../arquivo/ISSUE_036_BIBLIOTECA_DE_DOCUMENTOS.md).

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
  LIAF vai para o que o Typst não oferece (dados tipados, diagnóstico JSON, integração com rotas,
  pacotes embutidos no build, laço de correção do agente).

**Sem modelos prontos (30/09/2026):** o documento é código Typst comum, escrito pelo programa. A
liberdade criativa vem do código; a LIAF não traz layouts prontos nem uma biblioteca de design
montada por baixo. Os exemplos (`curriculo.typ`, `examples/cobranca_pix_pdf/cobranca.typ`) são
autocontidos e servem de ponto de partida, não de dependência.

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

### 4.3 Builtins

| Builtin | Assinatura | O que faz |
|---|---|---|
| `pdf-typst` | `(fonte str, dados_json str) (result str str)` | Documento Typst; dados em `/dados.json` |
| `pdf-typst-files` | `(fonte str, dados_json str, arquivos (map str str)) (result str str)` | Idem, com arquivos extras (imagens, SVG, CSV) |
| `png-typst` / `png-typst-files` | mesmas assinaturas | Prévia PNG da 1ª página (144 ppi) |
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
| Imagens e arquivos | `pdf-typst-files` recebe `(map str str)` nome → bytes. Os nomes são validados: nada de `..`, `\`, `@`, nem sobrescrever `/main.typ` ou `/dados.json`. Limite de 64 MB |
| Pacotes do Typst Universe | O `liafc build`/`run` procura `@ns/nome:versão` no código gerado, no `.liaf` e nos `.typ` da pasta, baixa com cache (`<UserCacheDir>/liaf/typst-packages`), resolve as dependências entre pacotes, confere o **SHA-256 em `liaf-typst.lock`** e embute em `typstpkgs/` com um `init()` que registra no runtime. O motor aceita arquivos `@ns/nome:versão/...`; em execução nada acessa a rede. Pacotes com plugin WASM (ex.: `tiaoma`, QR/código de barras via zint) funcionam |

Exemplo: `examples/cobranca_pix_pdf/`, uma cobrança Pix com QR Code via `@preview/tiaoma:0.3.0`.
Lição registrada no `.typ`: texto para copiar (Pix copia e cola) precisa de `hyphenate: false` e
`lang: "en"`. Em pt, o Typst repete o hífen na linha seguinte e corrompe o código colado.

### 4.6 Rodada 3: sem modelos, tamanho, licenças e CI (30/09/2026)

| Item | Entrega |
|---|---|
| Sem modelos prontos | Removidos `pkg/typst/lib` e os builtins `pdf-template*`/`png-template`. O currículo virou `curriculo.typ` (código comum, autocontido) e gera o mesmo PDF byte a byte |
| Tamanho | Fontes comprimidas (7,7 → 4,8 MB) e cada família com build tag (`liaf_doc_sem_<família>`). `liafc build --doc-fonts=inter,math` escolhe as famílias. App com documentos: **30,4 MB** com todas as fontes, **26,8 MB** só com a Inter (antes 33,3 MB; sem documentos, 12 MB) |
| Primeira carga | O runtime começa a carregar o motor em segundo plano quando o programa sobe (`LIAF_DOC_WARMUP=0` desliga): a primeira requisição não espera a compilação |
| Licenças | `scripts/build-typst-wasm.sh` gera `pkg/typst/wasm/THIRD_PARTY.txt` (279 crates, todas permissivas: MIT, Apache-2.0, BSD, Zlib, Unicode). `typst.Notices()` junta motor, wazero e as licenças das fontes (embutidas em todo app com documentos). `THIRD_PARTY_NOTICES.md` na raiz vai no pacote npm, e um teste falha se ficar desatualizado |
| Build reprodutível | Rust fixado (`engines/typst/rust-toolchain.toml`), caminhos da máquina removidos do binário (`--remap-path-prefix`), `--locked`, gzip sem data. `--check` recompila e compara; `--docker` usa o mesmo contêiner do CI |
| CI | `.github/workflows/typst-wasm.yml` recompila no contêiner `rust:1.98.1-bookworm` e falha se o artefato versionado não bater, publicando o artefato gerado |

### 4.7 Exemplos

- `curriculo.liaf` + `curriculo.typ` na raiz: structs tipadas → `json-encode` → `pdf-typst` →
  `curriculo.pdf` + `curriculo.png`.
- `examples/cobranca_pix_pdf/`: cobrança Pix com QR Code via `@preview/tiaoma:0.3.0`.

## 5. Testes

- `pkg/typst`: PDF, determinismo, diagnóstico com posição, arquivo de dados, PNG, pacote
  indisponível e concorrência.
- `pkg/docrt`: o `curriculo.typ` da raiz renderiza, é determinístico, aceita dados mínimos e foto;
  erro com posição; headers da resposta. Com `LIAF_DOC_OUT=<pasta>`, grava PDF e PNG para inspeção
  visual.
- `pkg/codegen`: `TestDocImportOnlyWhenUsed`.
- `pkg/typst/packages`: citações, dependências entre pacotes, cache sem rede, trava divergente e tar com `../` (servidor falso).
- `pkg/docrt`: fontes (fórmula/serifa/mono sem aviso), imagem e SVG, nomes de arquivo proibidos, pacote registrado e pacote ausente.
- `pkg/typst/fonts`: as 4 famílias descomprimem para fontes válidas e trazem licença; `cmd/liafc`: `--doc-fonts`.
- `pkg/typst`: `THIRD_PARTY_NOTICES.md` em dia com o que está embutido.

## 6. Fora do escopo desta issue

- Sintaxe própria de documento na LIAF → [#035](ISSUE_035_DECLARACAO_DOC.md).
- Modelos prontos e biblioteca de design: descartados por decisão (§2).
- `liafc doc`, score de qualidade e laço de correção → [#037](ISSUE_037_QUALIDADE_E_LACO_DO_AGENTE.md).
- Ler, editar ou assinar PDFs existentes; formulários AcroForm; assinatura ICP-Brasil.

## 7. Pendências e riscos

- **Primeira execução do CI:** o `.wasm.gz` versionado foi gerado no Windows. É provável que o
  build no Linux saia com bytes diferentes (o Rust grava caminhos com separador do sistema). Se o
  job falhar, baixe o artefato que ele publica, ou rode `scripts/build-typst-wasm.sh --docker` com o
  Docker ligado, e versione o resultado. Depois disso, o CI passa a ser a referência.
- **Tamanho:** o motor em si (8,9 MB comprimido, ~25 MB em memória) é o piso. A medir: `wasm-opt`
  (tirado do script por ora, para manter o build reprodutível) e remover do motor o que o backend não
  usa (realce de sintaxe, exportação HTML/SVG).
- **Artefato no git:** `pkg/typst/wasm/liaf_typst.wasm.gz` (8,9 MB) é versionado a cada mudança do
  motor. Se o histórico pesar, mover para Git LFS ou para download no build com checksum fixo.
- **Primeira carga sem cache** (7 s) continua existindo num contêiner novo. O aquecimento em segundo
  plano esconde isso da primeira requisição, mas não do tempo de partida em si.

## 8. Critério de conclusão

Concluída quando o job `typst-wasm` rodar verde no GitHub com o artefato versionado. O resto
(§4, §5, SPEC §26, `AI_GUIDE.md`, skill) está entregue.
