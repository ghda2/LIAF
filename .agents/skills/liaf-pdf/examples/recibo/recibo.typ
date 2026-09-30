// Recibo de pedido: layout em Typst, dados vindos do programa LIAF em /dados.json.
// Campos: numero, emitido_em, loja, cliente, itens [{nome, qtd, preco_centavos}], cor.

#let d = json("/dados.json")
#let destaque = rgb(d.at("cor", default: "#2563eb"))
#let suave = rgb("#5b6472") // contraste >= 4,5:1 sobre branco

// Dinheiro em centavos (int) -> "R$ 1.234,56". Nunca use float para dinheiro.
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

#set document(title: "Recibo " + str(d.numero))
#set page(paper: "a4", margin: 20mm, footer: context align(right, text(size: 8pt, fill: suave,
  counter(page).display("1 de 1", both: true))))
#set text(font: "Inter", size: 10pt, lang: "pt", region: "BR")

#grid(
  columns: (1fr, auto),
  align: (left + horizon, right + horizon),
  text(size: 20pt, weight: 700)[Recibo nº #d.numero],
  text(fill: suave)[#d.emitido_em],
)
#v(4pt)
#text(weight: 600, fill: destaque, d.loja) · Cliente: *#d.cliente*

#v(12pt)
#table(
  columns: (1fr, auto, auto, auto),
  align: (left, center, right, right),
  stroke: none,
  inset: (x: 6pt, y: 7pt),
  fill: (_, y) => if y == 0 { destaque } else if calc.odd(y) { destaque.lighten(92%) },
  table.header(
    ..("Item", "Qtd", "Preço", "Subtotal").map(h => text(fill: white, weight: 600, h)),
  ),
  ..d.itens.map(i => (
    i.nome,
    str(i.qtd),
    reais(i.preco_centavos),
    reais(i.qtd * i.preco_centavos),
  )).flatten(),
)

#align(right, block(inset: (top: 8pt), text(size: 14pt, weight: 700)[Total: #reais(total)]))
