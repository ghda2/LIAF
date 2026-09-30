# Issue #035 — Declaração `doc`: documentos escritos em LIAF

**Estado:** Proposta. **Criada em:** 29/09/2026. **Depende de:** [#034](ISSUE_034_DOCUMENTOS_PDF_NATIVOS.md).
**Componentes:** `pkg/lexer`, `pkg/parser` (modo marcação), `pkg/checker`, `pkg/codegen` (emissor Typst),
`pkg/diagnostic`.

---

## 1. Contexto

Com a #034, um programa LIAF gera PDF de duas formas:

- `pdf-template("cv-moderno", json)`: escolhe um modelo pronto. Não dá para mudar o layout;
- `pdf-typst(fonte, json)`: escreve Typst. Liberdade total, mas é **uma segunda linguagem**, os
  dados chegam sem tipo (`json("/dados.json")`) e um campo errado só aparece quando o documento roda.

Falta o meio-termo: escrever o documento **na própria LIAF**, com os dados tipados e o checker
conferindo tudo, e deixar o Typst só como motor por baixo.

## 2. Proposta

`doc` é uma declaração que o codegen **traduz para Typst**. Quem escreve LIAF não precisa saber
Typst. Quem sabe continua com o `pdf-typst` como válvula de escape.

```liaf
doc recibo(p Pedido)
  @page A4 margin 18mm
  @set text font "Inter" size 10pt

  # Recibo nº {p.id}
  Cliente: **{p.cliente}** · {p.criado_em}

  @table cols(1fr, auto, auto) header("Item", "Qtd", "Preço")
    @for-each i p.itens
      {i.nome} | {i.qtd} | {moeda-br(i.preco)}
    @end
  @end
end

route GET "/pedidos/{id}/recibo.pdf" (id int) Response effects(db)
  on-err e
    json-response(500, e)
  end
  let p = try buscar_pedido(id)
  pdf-response(try pdf-render(recibo(p)), "recibo.pdf")
end
```

### 2.1 Três camadas, para dar liberdade total

| Camada | Diretivas | Para quê |
|---|---|---|
| **Fluxo** | marcação Markdown, `{expr}`, `@for-each`, `@if`, `@table`, `@list` | Texto que quebra página sozinho |
| **Layout** | `@grid`, `@columns`, `@stack`, `@box`, `@align`, `@place`, `@page` | Colunas, cartões, cabeçalhos, sobreposição |
| **Desenho** | `@canvas` com `line`, `rect`, `circle`, `path`, `@svg`, `@image` | Certificado, etiqueta, crachá, capa |

E `@typst ... @end` injeta Typst cru para o que ainda não tiver diretiva.

### 2.2 Regras

- `doc nome(params)` é **pura** e devolve `Doc`. Só quem renderiza tem efeito. `pdf-render(d)` e
  `png-render(d)` devolvem `(result str str)`.
- `{expr}` aceita `str`, `int`, `float` e `Doc` (componente), e o checker confere o tipo.
  Interpolação vira texto escapado no Typst: um dado nunca é interpretado como código.
- Os componentes da biblioteca (#036) ficam disponíveis como diretivas (`@etiqueta`, `@linha-do-tempo`).
- Marcação Markdown, e não Typst, porque os modelos erram menos nela. Isso precisa ser medido (§4).

### 2.3 Diagnósticos novos

| Código | Quando |
|---|---|
| `E_DOC_UNKNOWN_DIRECTIVE` | `@tabel`, com sugestão por distância de edição |
| `E_DOC_UNCLOSED_BLOCK` | `@table` sem `@end` |
| `E_DOC_TABLE_ARITY` | A linha tem 2 células e `cols` declara 3 |
| `E_DOC_INTERP_TYPE` | `{p.itens}`: lista interpolada como texto |
| `E_DOC_TYPST` | Erro do motor, **remapeado para a linha do `.liaf`** que gerou o trecho |

O remapeamento exige que o emissor guarde um mapa linha-Typst → linha-LIAF (*source map*).

## 3. Fora do escopo

Fórmulas matemáticas com sintaxe própria (usar `@typst`), `@show` (regras de transformação) e
saída HTML do mesmo `doc` (depois, via `typst-html`).

## 4. Impacto em tokens (regra 4 da #031)

Medir o recibo de três jeitos: `doc` da LIAF, `pdf-typst` com Typst escrito à mão e HTML/CSS.
Medir também marcação Markdown contra marcação Typst dentro do `doc`. Fica a forma mais barata até
o PDF correto.

## 5. Casos de aceitação

`conformance/basics/035_*.liaf`: `035_recibo`, `035_componente` (`doc` dentro de `doc`),
`035_tabela_paginada` (120 linhas, cabeçalho repetido), `035_escape` (dado com `#` e `*` sai
literal), `035_campo_errado` (`E_UNKNOWN_FIELD`), `035_aridade_tabela` (`E_DOC_TABLE_ARITY`) e
`035_erro_typst` (`E_DOC_TYPST` com a linha do `.liaf`).

## 6. Critério de conclusão

Casos de §5 compilados e executados; rota servindo o PDF de um `doc` ponta a ponta; SPEC, `AI_GUIDE.md` e
skill (`references/doc.md`) atualizados; impacto em tokens registrado.
