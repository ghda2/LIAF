// Modelo "cv-moderno": currículo de duas colunas, com coluna lateral tingida pela cor de
// destaque, linha do tempo nas experiências e barras de nível nas habilidades.
//
// Dados (JSON; campos ausentes ou vazios são omitidos):
//   nome, cargo, resumo, email, telefone, local, site, linkedin, github, cor ("#rrggbb")
//   foto: nome de um arquivo extra (pdf-template-files), ex.: "foto.jpg"; sem ela, iniciais
//   experiencias: [{cargo, empresa, periodo, local, descricao, destaques: [str], tecnologias: [str]}]
//   formacao:     [{curso, instituicao, periodo, descricao}]
//   projetos:     [{nome, descricao, link, tecnologias: [str]}]
//   habilidades:  [{nome, nivel (1 a 5)}]
//   idiomas:      [{nome, nivel}]
//   certificacoes: [str]
//   interesses:   [str]
#import "../tema.typ": *
#import "../componentes.typ": *

#let largura-lateral = 190pt

#let coluna-lateral(p, d) = {
  let foto = campo(d, "foto")
  align(center, if tem(foto) {
    avatar-foto(p, "/" + foto.trim("/", at: start), tamanho: 84pt)
  } else {
    avatar-iniciais(p, d.nome, tamanho: 76pt)
  })
  v(esp.xl)

  titulo-secao(p, "Contato")
  if tem(campo(d, "email")) { rotulo-valor(p, "E-mail", d.email, destino: "mailto:" + d.email) }
  if tem(campo(d, "telefone")) { rotulo-valor(p, "Telefone", d.telefone) }
  if tem(campo(d, "local")) { rotulo-valor(p, "Local", d.local) }
  if tem(campo(d, "site")) { rotulo-valor(p, "Site", d.site, destino: "https://" + d.site.trim("https://", at: start)) }
  if tem(campo(d, "linkedin")) { rotulo-valor(p, "LinkedIn", d.linkedin, destino: "https://" + d.linkedin.trim("https://", at: start)) }
  if tem(campo(d, "github")) { rotulo-valor(p, "GitHub", d.github, destino: "https://" + d.github.trim("https://", at: start)) }

  let habilidades = lista(d, "habilidades")
  if habilidades.len() > 0 {
    titulo-secao(p, "Habilidades")
    for h in habilidades {
      block(below: esp.md + 1pt, breakable: false, {
        text(size: escala.sm, weight: 500, h.nome)
        v(esp.sm, weak: true)
        barra-nivel(p, campo(h, "nivel", padrao: 3))
      })
    }
  }

  let idiomas = lista(d, "idiomas")
  if idiomas.len() > 0 {
    titulo-secao(p, "Idiomas")
    for i in idiomas {
      block(below: esp.md, grid(
        columns: (1fr, auto),
        text(size: escala.sm, weight: 500, i.nome),
        text(size: escala.xs, fill: p.suave, campo(i, "nivel", padrao: "")),
      ))
    }
  }

  let certificacoes = lista(d, "certificacoes")
  if certificacoes.len() > 0 {
    titulo-secao(p, "Certificações")
    for c in certificacoes {
      block(below: esp.sm, text(size: escala.sm, c))
    }
  }

  let interesses = lista(d, "interesses")
  if interesses.len() > 0 {
    titulo-secao(p, "Interesses")
    set par(leading: esp.md)
    interesses.map(i => etiqueta(p, i)).join(h(esp.sm))
  }
}

#let coluna-principal(p, d) = {
  text(size: escala.xxl, weight: 700, tracking: -0.6pt, fill: p.texto, d.nome)
  if tem(campo(d, "cargo")) {
    block(above: esp.lg, below: 0pt, text(size: escala.lg, weight: 500, fill: p.destaque, d.cargo))
  }
  if tem(campo(d, "resumo")) {
    block(
      above: esp.xxl,
      width: 100%,
      inset: (left: esp.lg, y: esp.sm),
      stroke: (left: 2.5pt + p.destaque),
      par(leading: 0.7em, text(fill: p.texto.lighten(15%), d.resumo)),
    )
  }

  let experiencias = lista(d, "experiencias")
  if experiencias.len() > 0 {
    titulo-secao(p, "Experiência")
    for e in experiencias {
      item-linha-do-tempo(
        p,
        e.cargo,
        subtitulo: campo(e, "empresa"),
        quando: campo(e, "periodo"),
        onde: campo(e, "local"),
        corpo: {
          if tem(campo(e, "descricao")) { par(leading: 0.65em, e.descricao) }
          destaques(p, lista(e, "destaques"))
          etiquetas(p, lista(e, "tecnologias"))
        },
      )
    }
  }

  let projetos = lista(d, "projetos")
  if projetos.len() > 0 {
    titulo-secao(p, "Projetos")
    for pr in projetos {
      block(below: esp.xl, breakable: false, {
        grid(
          columns: (1fr, auto),
          text(weight: 600, size: escala.md, pr.nome),
          if tem(campo(pr, "link")) {
            link("https://" + pr.link.trim("https://", at: start), text(size: escala.sm, fill: p.destaque, pr.link))
          },
        )
        if tem(campo(pr, "descricao")) {
          block(above: esp.sm, below: 0pt, par(leading: 0.65em, pr.descricao))
        }
        etiquetas(p, lista(pr, "tecnologias"))
      })
    }
  }

  let formacao = lista(d, "formacao")
  if formacao.len() > 0 {
    titulo-secao(p, "Formação")
    for f in formacao {
      item-linha-do-tempo(
        p,
        f.curso,
        subtitulo: campo(f, "instituicao"),
        quando: campo(f, "periodo"),
        corpo: if tem(campo(f, "descricao")) { f.descricao },
      )
    }
  }
}

#let modelo(d) = {
  let p = paleta(cor-de(campo(d, "cor")))
  set document(title: d.nome + " — Currículo", author: d.nome)
  set page(
    paper: "a4",
    margin: (x: 0pt, y: esp.xxxl),
    background: place(top + left, rect(width: largura-lateral, height: 100%, fill: p.lateral)),
    footer: context {
      if counter(page).final().first() > 1 {
        set text(size: escala.xs, fill: p.suave)
        pad(left: largura-lateral + 28pt, right: 32pt, grid(
          columns: (1fr, auto),
          d.nome,
          counter(page).display("1 / 1", both: true),
        ))
      }
    },
  )
  set text(font: "Inter", size: escala.base, fill: p.texto, lang: "pt", region: "BR", hyphenate: false)
  set par(leading: 0.62em, spacing: esp.md)

  grid(
    columns: (largura-lateral, 1fr),
    pad(x: 22pt, coluna-lateral(p, d)),
    pad(left: 28pt, right: 32pt, coluna-principal(p, d)),
  )
}
