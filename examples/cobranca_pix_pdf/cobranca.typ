// Cobrança Pix em PDF: QR Code gerado pelo pacote tiaoma (Typst Universe), embutido no binário
// pelo liafc build. Dados em /dados.json; biblioteca de design da LIAF em /liaf/.
#import "@preview/tiaoma:0.3.0": qrcode
#import "/liaf/tema.typ": *
#import "/liaf/componentes.typ": titulo-secao, etiqueta

#let d = json("/dados.json")
#let p = paleta(cor-de(campo(d, "cor")))
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
