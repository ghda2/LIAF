// Cobrança Pix em PDF, escrita como código Typst comum. O QR Code vem do pacote tiaoma
// (Typst Universe), que o liafc build embute no binário. Dados em /dados.json.
#import "@preview/tiaoma:0.3.0": qrcode

// Tokens de design: tamanhos, espaçamento e paleta derivada de uma cor.
#let escala = (xs: 7.5pt, sm: 8.5pt, base: 10pt, xl: 20pt, xxl: 28pt)
#let esp = (xs: 2pt, sm: 4pt, md: 8pt, lg: 12pt, xl: 16pt)
#let paleta(destaque) = (
  destaque: destaque,
  destaque-forte: destaque.darken(25%),
  destaque-suave: destaque.lighten(88%),
  texto: rgb("#1f2937"),
  suave: rgb("#6b7280"),
  linha: rgb("#e5e7eb"),
)

#let titulo-secao(p, texto) = block(above: esp.xl, below: esp.md, sticky: true, grid(
  columns: (auto, 1fr),
  column-gutter: esp.md,
  align: horizon,
  text(size: escala.sm, weight: 700, tracking: 1.4pt, fill: p.destaque, upper(texto)),
  line(length: 100%, stroke: 0.6pt + p.linha),
))

#let etiqueta(p, texto) = box(
  inset: (x: 6pt, y: 3pt),
  radius: 8pt,
  fill: p.destaque-suave,
  text(size: escala.xs, weight: 500, fill: p.destaque-forte, texto),
)

#let d = json("/dados.json")
#let p = paleta(rgb(d.at("cor", default: "#2563eb")))
#let reais(centavos) = {
  let s = str(calc.rem(centavos, 100))
  "R$ " + str(calc.quo(centavos, 100)) + "," + (if s.len() == 1 { "0" + s } else { s })
}

#set document(title: "Cobrança " + d.txid)
#set page(paper: "a5", margin: 14mm)
#set text(font: "Inter", size: escala.base, fill: p.texto, lang: "pt")

#text(size: escala.sm, weight: 700, tracking: 1.4pt, fill: p.destaque, upper(d.recebedor))
#v(esp.sm)
#text(size: escala.xl, weight: 700)[Pague com Pix]

#v(esp.lg)
#grid(
  columns: (auto, 1fr),
  column-gutter: esp.xl,
  align: horizon,
  box(inset: esp.md, radius: 8pt, stroke: 1pt + p.linha, qrcode(d.copia_e_cola, width: 42mm)),
  {
    text(size: escala.sm, fill: p.suave)[Valor]
    linebreak()
    text(size: escala.xxl, weight: 700, fill: p.destaque, reais(d.valor_centavos))
    v(esp.md)
    etiqueta(p, "Vence em " + d.vencimento)
  },
)

#titulo-secao(p, "Pix copia e cola")
#block(
  width: 100%,
  inset: esp.md,
  radius: 6pt,
  fill: p.destaque-suave,
  // Código para copiar: sem hifenização e sem a regra do português de repetir o hífen na
  // linha seguinte, que corromperia o texto colado no app do banco.
  text(font: "DejaVu Sans Mono", size: escala.xs, fill: p.destaque-forte, lang: "en", hyphenate: false, d.copia_e_cola),
)

#titulo-secao(p, "Detalhes")
#grid(
  columns: (auto, 1fr),
  row-gutter: esp.md,
  column-gutter: esp.xl,
  text(fill: p.suave)[Descrição], d.descricao,
  text(fill: p.suave)[Identificador], raw(d.txid),
)
