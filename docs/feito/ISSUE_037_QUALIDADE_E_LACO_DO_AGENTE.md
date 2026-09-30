# Issue #037 — Qualidade de documento e laço de correção do agente

**Estado:** Concluída em 30/09/2026. **Criada em:** 29/09/2026. **Depende de:** [#034](ISSUE_034_DOCUMENTOS_PDF_NATIVOS.md).
**Componentes:** `cmd/liafc` (`liafc doc`), `engines/typst` (introspecção do layout), `pkg/typst`,
`.agents/skills/liaf`.

> **Progresso (30/09/2026):** entregue o núcleo, sobre documentos Typst (não depende da #035):
>
> - `liafc doc check doc.typ [--dados d.json] [--arquivo nome=caminho]... [--json] [--layout]` e
>   `liafc doc preview doc.typ [-o p.png] [--pagina N] [--ppi N]` (`cmd/liafc/doc.go`). Os pacotes
>   `@preview` citados são resolvidos como no build (cache + `liaf-typst.lock`).
> - Relatório de layout calculado no motor (`engines/typst/src/layout.rs`): por página, a caixa do
>   conteúdo, as contagens de texto, imagem e forma, e os elementos fora da página. Conteúdo recortado
>   de propósito (`clip`) e fundos de página inteira não contam.
> - `typst.Analyze` (`pkg/typst/quality.go`) gera `W_LAYOUT_OUT_OF_PAGE`, `W_LAYOUT_BLANK_PAGE` e
>   `W_LAYOUT_EMPTY_TAIL` com `fix`, além das métricas `pages`, `overflow`, `blank_pages`,
>   `last_page_usage` e `warnings`. Erros e avisos do Typst saem como `E_TYPST`/`W_TYPST`, com
>   linha e coluna.
> - Testes: `pkg/typst/quality_test.go` (documento limpo, estouro, página em branco, última página
>   quase vazia, espaço invisível e recorte) e `cmd/liafc/doc_test.go`.
>
> **Rodada 2 (30/09/2026):**
>
> - `W_LAYOUT_LOW_CONTRAST`: o motor guarda a cor de cada fundo sólido na ordem de pintura e mede o
>   contraste WCAG de cada texto contra o fundo desenhado atrás dele (4,5:1, ou 3:1 para texto
>   ≥ 18 pt ou ≥ 14 pt em negrito). O `fix` traz uma cor do mesmo tom que passa: escurecida sobre
>   fundo claro e clareada sobre fundo escuro. Testes: `TestAnalyzeLowContrast` e
>   `TestFixContrastReachesMinimum`.
> - `liafc doc watch`: reconfere quando o documento, os dados ou um arquivo extra mudam; com `-o`,
>   regrava a prévia.
> - Regressão visual: `pkg/docrt/golden_test.go` compara a prévia do `curriculo.typ` com
>   `testdata/curriculo.png`, com tolerância de 0,1% dos pixels. Para aprovar uma mudança
>   intencional: `LIAF_UPDATE_GOLDEN=1 go test ./pkg/docrt`.
>
> **Decidido não fazer:** `W_LAYOUT_ORPHAN`. O Typst já prende títulos ao conteúdo seguinte
> (`sticky`), então um título sozinho no pé da página só acontece se o autor desligar isso de
> propósito, e o aviso geraria mais ruído que ajuda. Glifo sem fonte já aparece como `W_TYPST`
> ("unknown font family").

---

## 1. Contexto

"Ficou bonito" não é critério, e um modelo só de texto não enxerga o PDF. Esta issue dá ao
documento o que a LIAF já dá ao código: **diagnóstico em JSON e correção em rodadas**.

## 2. Proposta

### 2.1 `liafc doc`

```text
liafc doc check  arquivo.liaf [--dados x.json] --json   # diagnósticos + métricas
liafc doc preview arquivo.liaf [--pagina N] -o p.png     # prévia para modelos multimodais
liafc doc watch  arquivo.liaf                            # recompila ao salvar
```

### 2.2 Relatório de layout (para modelos só de texto)

O motor devolve, por página, as caixas de cada elemento (título, parágrafo, tabela, imagem) com
posição e tamanho, extraídas do *frame* do Typst. O modelo confere a quebra de página e o
alinhamento sem ver a imagem.

### 2.3 Avisos de layout com correção pronta

| Código | Quando | `fix` sugerido |
|---|---|---|
| `W_LAYOUT_OVERFLOW` | Conteúdo mais largo que a coluna | nova largura de coluna ou `hyphenate` |
| `W_LAYOUT_ORPHAN` | Título sozinho no pé da página | `sticky`/`breakable: false` |
| `W_LAYOUT_EMPTY_TAIL` | Última página com < 15% de uso | reduzir espaçamento ou fonte |
| `W_GLYPH_MISSING` | Caractere sem glifo na fonte | fonte de reserva |
| `W_CONTRAST` | Texto com contraste abaixo de WCAG AA | cor do token mais próximo que passa |

### 2.4 Score

`--score` devolve as métricas `overfull`, `orphans`, `hyphen_runs`, `tail_usage`, `pages`,
`ms_per_page` e `bytes_per_page`, com metas por nível de qualidade:

- **N1 funcional**;
- **N2 profissional**, a meta dos modelos da #036;
- **N3 editorial**.

### 2.5 Rodadas

```text
escreve ─► doc check --score --json ─► aplica os "fix" ─► repete
          (rodada 3 opcional: preview PNG para modelo multimodal)
```

Limite de 3 rodadas, ou parar quando o score melhorar menos de 5%. As regras ficam na skill.

### 2.6 Regressão visual

PNG de referência aprovado por modelo (`pkg/docrt/testdata/*.png`), comparado por diferença de
pixel com tolerância. Uma mudança de tema ou de versão do Typst que altere o visual reprova até a
nova referência ser aprovada.

## 3. Casos de aceitação

Um documento com estouro de coluna recebe `W_LAYOUT_OVERFLOW` com `fix`, e aplicar o `fix` zera o
aviso. O mesmo para órfão e para página final quase vazia. A regressão visual reprova uma mudança
proposital de cor.

## 4. Critério de conclusão

`liafc doc check/preview` funcionando, com os avisos de §2.3 testados, o score documentado e a
skill instruindo o laço de rodadas. A #009 deve registrar quantas rodadas um modelo pequeno leva até
o N2.
