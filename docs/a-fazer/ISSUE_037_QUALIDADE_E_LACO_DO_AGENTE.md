# Issue #037 — Qualidade de documento e laço de correção do agente

**Estado:** Proposta. **Criada em:** 29/09/2026. **Depende de:** [#034](ISSUE_034_DOCUMENTOS_PDF_NATIVOS.md);
aproveita a [#035](ISSUE_035_DECLARACAO_DOC.md) quando ela existir.
**Componentes:** `cmd/liafc` (`liafc doc`), `engines/typst` (introspecção do layout), `pkg/typst`,
`.agents/skills/liaf`.

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
