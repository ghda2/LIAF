# Laço de correção

O `liafc doc check --json` é o compilador do documento: entregue só quando ele estiver limpo.

## Formato da saída

```json
{
  "status": "success",
  "errors": [],
  "warnings": [
    {
      "code": "W_LAYOUT_LOW_CONTRAST",
      "severity": "warning",
      "message": "9 texto(s) #6b7280 com contraste 4.47:1 sobre #f3f7f6 (mínimo 4.5:1): \"E-MAIL\", ...",
      "page": 1,
      "fix": "troque a cor do texto #6b7280 por #686f7c (contraste 4.5:1 sobre #f3f7f6)"
    }
  ],
  "metrics": {
    "pages": 1, "overflow": 0, "low_contrast": 9, "blank_pages": 0,
    "last_page_usage": 0.62, "warnings": 1
  }
}
```

`status` é `"error"` quando `errors` não está vazio. `file`, `line` e `column` aparecem nos erros do
Typst.

## Códigos

| Código | Significa | O que fazer |
|---|---|---|
| `E_TYPST` | Erro de sintaxe, campo inexistente, arquivo ou pacote ausente | Corrigir na linha/coluna indicada. Leia também `hints` |
| `W_TYPST` | Aviso do Typst (ex.: fonte desconhecida) | Usar uma fonte embutida ou corrigir o que ele indica |
| `W_LAYOUT_OUT_OF_PAGE` | Elemento visível passa da borda (vai ser cortado) | Largura relativa (`100%`, `1fr`) em vez de fixa; fonte ou imagem menor |
| `W_LAYOUT_BLANK_PAGE` | Página sem nenhum conteúdo | Remover `pagebreak()` sobrando |
| `W_LAYOUT_EMPTY_TAIL` | A última página usa menos de 15% da altura | Reduzir espaçamentos ou fonte no percentual sugerido |
| `W_LAYOUT_LOW_CONTRAST` | Texto abaixo do WCAG AA contra o fundo | Trocar pela cor do `fix` |

## Algoritmo

```text
rodada = 1
repita:
    r = liafc doc check doc.typ --dados dados-exemplo.json --json
    se r.errors: corrija TODOS os erros; volte ao início (erro não conta rodada)
    se r.warnings vazio: pare, documento pronto
    aplique o "fix" de cada aviso
    se rodada == 3 ou r.metrics.warnings não diminuiu: pare e explique o que ficou
    rodada += 1
```

- Corrija **todos** os erros antes de olhar os avisos: o layout só é medido quando compila.
- Um aviso só pode ficar se for intencional, e isso precisa ser dito na entrega. Exemplo: última página
  curta num documento que precisa começar em página nova.
- Se você enxerga imagens, faça uma rodada visual no fim:
  `liafc doc preview doc.typ --dados dados-exemplo.json -o p.png` e confira alinhamento, hierarquia e
  espaçamento.
- Teste também com **dados extremos**: lista vazia, lista longa (confere a quebra de página e o
  cabeçalho repetido da tabela) e nome muito comprido (confere se estoura a coluna).
