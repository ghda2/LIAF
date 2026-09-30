# Typst essencial para documentos LIAF

O suficiente para escrever recibos, relatórios, currículos e cobranças. A referência completa é
<https://typst.app/docs>, e tudo o que está lá funciona no motor embutido.

## 1. Dois modos

- **Marcação** (o padrão): texto comum, `= Título`, `== Subtítulo`, `*negrito*`, `_itálico_`,
  `- item de lista`, `+ item numerado`, `\` para quebra de linha forçada.
- **Código**: começa com `#`. `#let x = 1`, `#d.nome`, `#if cond [...]`, `#for i in lista [...]`.
  Dentro de `[...]` volta a ser marcação; dentro de `{...}` é código.

## 2. Ler os dados

```typst
#let d = json("/dados.json")

Cliente: *#d.cliente*                        // campo obrigatório
#d.at("observacao", default: "")             // campo opcional
#for item in d.itens [ - #item.nome \ ]      // lista
```

- `d.campo` inexistente é erro com linha e coluna ("dictionary does not contain key").
- Número vira texto com `str(n)`: `"Pedido " + str(d.numero)`.

## 3. Configuração do documento (no topo)

```typst
#set document(title: "Recibo " + str(d.numero))
#set page(paper: "a4", margin: 20mm)              // "a5", "us-letter", (x: 15mm, y: 20mm)
#set text(font: "Inter", size: 10pt, lang: "pt", region: "BR")
#set par(justify: true, leading: 0.65em)
#set heading(numbering: "1.")                     // títulos numerados
```

Rodapé com numeração de página:

```typst
#set page(footer: context align(right, counter(page).display("1 de 1", both: true)))
```

## 4. Layout

```typst
#grid(columns: (1fr, auto), gutter: 8pt, [esquerda], [direita])   // colunas
#align(right)[Total]                                              // alinhamento
#v(12pt)  #h(1fr)                                                 // espaço vertical / horizontal elástico
#block(fill: rgb("#f3f4f6"), inset: 10pt, radius: 6pt, width: 100%)[caixa]
#box(fill: blue, inset: 4pt)[em linha]
#place(top + right, dx: -10pt)[carimbo]                           // posição absoluta
#pagebreak()
#columns(2)[texto em duas colunas]
```

`1fr` divide o espaço que sobra; `auto` ocupa o necessário. Prefira larguras relativas (`100%`, `1fr`)
a larguras fixas grandes, que podem passar da página.

## 5. Tabelas

```typst
#table(
  columns: (1fr, auto, auto),
  align: (left, center, right),
  stroke: none,
  fill: (_, y) => if y == 0 { rgb("#1e3a8a") } else if calc.odd(y) { rgb("#eef2ff") },
  table.header(..("Item", "Qtd", "Preço").map(h => text(fill: white, weight: 600, h))),
  ..d.itens.map(i => (i.nome, str(i.qtd), reais(i.preco_centavos))).flatten(),
)
```

O `table.header` se repete sozinho quando a tabela continua na página seguinte.

## 6. Cores, imagens e formas

```typst
#let destaque = rgb(d.at("cor", default: "#2563eb"))
destaque.lighten(90%)   destaque.darken(20%)
#text(fill: destaque, weight: 700)[texto]
#image("/img/logo.png", width: 3cm)                     // arquivo entregue por pdf-typst-files
#image(bytes("<svg xmlns='http://www.w3.org/2000/svg' width='20' height='20'><circle cx='10' cy='10' r='10' fill='teal'/></svg>"))  // SVG no próprio documento
#box(width: 60pt, height: 60pt, radius: 50%, clip: true, image("/foto.jpg", width: 100%, height: 100%, fit: "cover"))
#rect(width: 100%, height: 2pt, fill: destaque)
#line(length: 100%, stroke: 0.5pt + gray)
```

## 7. Funções próprias

```typst
#let secao(titulo) = block(above: 16pt, below: 8pt, text(weight: 700, upper(titulo)))

// Dinheiro em centavos (int) -> "R$ 1.234,56"
#let reais(c) = {
  let inteiro = str(calc.quo(c, 100))
  let cent = str(calc.rem(c, 100))
  let partes = ()
  let resto = inteiro
  while resto.len() > 3 {
    partes.insert(0, resto.slice(resto.len() - 3))
    resto = resto.slice(0, resto.len() - 3)
  }
  partes.insert(0, resto)
  "R$ " + partes.join(".") + "," + (if cent.len() == 1 { "0" + cent } else { cent })
}

#let total = d.itens.map(i => i.qtd * i.preco_centavos).sum(default: 0)
```

Uma função precisa ser definida **antes** de ser usada, senão dá "unknown variable".

## 8. Fórmulas, código e pacotes

```typst
$ sum_(i=1)^n i = (n(n+1))/2 $                  // fórmula em destaque (fonte matemática embutida)
O juro é $j = C dot i dot t$.                   // fórmula na linha
#raw("func main() {}", lang: "go", block: true) // bloco de código (DejaVu Sans Mono)

#import "@preview/tiaoma:0.3.0": qrcode         // pacote do Typst Universe
#qrcode(d.pix, width: 40mm)
```

Pacotes são baixados e embutidos pelo `liafc build`/`run` (ver [comandos.md](comandos.md)). Use sempre
a versão exata (`nome:1.2.3`).
