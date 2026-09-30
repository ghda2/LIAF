// Relatório de Temperaturas - Lambari/MG
// Layout Typst com dados consumidos de /dados.json

#let d = json("/dados.json")
#let primario = rgb(d.at("cor_tema", default: "#0369a1"))
#let cinza_texto = rgb("#475569")
#let borda = rgb("#e2e8f0")
#let fundo_card = rgb("#f8fafc")

#let min_absoluta = d.registros.map(r => r.temp_min).fold(100, (acc, v) => calc.min(acc, v))
#let max_absoluta = d.registros.map(r => r.temp_max).fold(-100, (acc, v) => calc.max(acc, v))

#set document(title: "Relatório Climatológico - " + d.cidade)
#set page(
  paper: "a4",
  margin: (top: 20mm, bottom: 20mm, left: 20mm, right: 20mm),
  footer: context align(center, text(size: 8pt, fill: cinza_texto)[
    #d.estacao · Emitido em #d.gerado_em · Página #counter(page).display("1 de 1", both: true)
  ])
)
#set text(font: "Inter", size: 10pt, lang: "pt", region: "BR")
#set par(leading: 0.6em)

// Cabeçalho Principal
#grid(
  columns: (1fr, auto),
  align: (left + horizon, right + horizon),
  [
    #text(size: 8.5pt, weight: 700, fill: primario, tracking: 0.1em)[BOLETIM CLIMATOLÓGICO SEMANAL]
    #v(3pt)
    #text(size: 22pt, weight: 700, fill: rgb("#0f172a"))[#d.cidade - #d.estado]
    #v(2pt)
    #text(size: 9.5pt, fill: cinza_texto)[#d.estacao]
  ],
  [
    #block(
      fill: primario.lighten(92%),
      stroke: 0.5pt + primario.lighten(60%),
      inset: (x: 12pt, y: 8pt),
      radius: 6pt,
      align(center)[
        #text(size: 8pt, weight: 600, fill: primario)[PERÍODO] \
        #text(size: 10pt, weight: 700, fill: rgb("#0f172a"))[#d.periodo]
      ]
    )
  ]
)

#v(14pt)
#line(length: 100%, stroke: 0.8pt + borda)
#v(14pt)

// Resumo executivo em cartões de métricas (Cards)
#grid(
  columns: (1fr, 1fr, 1fr, 1fr),
  gutter: 10pt,
  [
    #block(fill: fundo_card, stroke: 0.5pt + borda, inset: 10pt, radius: 6pt, width: 100%)[
      #text(size: 8pt, weight: 600, fill: cinza_texto)[MÉDIA SEMANAL] \
      #v(2pt)
      #text(size: 16pt, weight: 700, fill: primario)[#d.temp_media]
    ]
  ],
  [
    #block(fill: fundo_card, stroke: 0.5pt + borda, inset: 10pt, radius: 6pt, width: 100%)[
      #text(size: 8pt, weight: 600, fill: cinza_texto)[MÍNIMA REGISTRADA] \
      #v(2pt)
      #text(size: 16pt, weight: 700, fill: rgb("#0274af"))[#str(min_absoluta)°C]
    ]
  ],
  [
    #block(fill: fundo_card, stroke: 0.5pt + borda, inset: 10pt, radius: 6pt, width: 100%)[
      #text(size: 8pt, weight: 600, fill: cinza_texto)[MÁXIMA REGISTRADA] \
      #v(2pt)
      #text(size: 16pt, weight: 700, fill: rgb("#b45309"))[#str(max_absoluta)°C]
    ]
  ],
  [
    #block(fill: fundo_card, stroke: 0.5pt + borda, inset: 10pt, radius: 6pt, width: 100%)[
      #text(size: 8pt, weight: 600, fill: cinza_texto)[PRECIPITAÇÃO TOTAL] \
      #v(2pt)
      #text(size: 16pt, weight: 700, fill: rgb("#047857"))[#d.precipitacao_total]
    ]
  ]
)

#v(12pt)

// Bloco com nota de síntese meteorológica
#block(
  fill: rgb("#f1f5f9"),
  stroke: (left: 3pt + primario),
  inset: (x: 12pt, y: 10pt),
  radius: (right: 4pt),
  [
    #text(size: 9.5pt, fill: rgb("#1e293b"))[*Análise Sinótica:* #d.resumo]
  ]
)

#v(18pt)
#text(size: 12pt, weight: 700, fill: rgb("#0f172a"))[Histórico Diário de Medições]
#v(6pt)

// Tabela de medições diárias
#table(
  columns: (1fr, 0.8fr, 1fr, 1fr, 1fr, 1.6fr),
  align: (center, center, center, center, center, left),
  stroke: none,
  inset: (x: 6pt, y: 8pt),
  fill: (_, y) => if y == 0 { primario } else if calc.odd(y) { primario.lighten(94%) } else { white },
  table.header(
    ..("Data", "Dia", "Mínima", "Máxima", "Umidade Rel.", "Condição").map(h => text(fill: white, weight: 600, size: 9pt, h)),
  ),
  ..d.registros.map(r => (
    text(weight: 500, size: 9pt, r.data),
    text(size: 9pt, r.dia_semana),
    text(size: 9pt, weight: 600, fill: rgb("#0274af"), str(r.temp_min) + "°C"),
    text(size: 9pt, weight: 600, fill: rgb("#b45309"), str(r.temp_max) + "°C"),
    text(size: 9pt, str(r.umidade) + "%"),
    text(size: 9pt, r.condicao),
  )).flatten(),
)

#v(16pt)
#line(length: 100%, stroke: 0.5pt + borda)
#v(8pt)

#text(size: 8.5pt, fill: cinza_texto)[
  *Observações Técnicas:* Os dados foram aferidos por sensores automáticos calibrados em conformidade com as diretrizes do INMET. As temperaturas refletem o microclima característico das estâncias minerais da Mantiqueira, com amplitude térmica diária acentuada.
]
