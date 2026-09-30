# Issue #036 — Biblioteca de documentos: temas, componentes e modelos

**Estado:** Em andamento. **Criada em:** 29/09/2026. **Depende de:** [#034](ISSUE_034_DOCUMENTOS_PDF_NATIVOS.md).
**Componentes:** `pkg/typst/lib` (Typst), `pkg/typst/fonts`, `pkg/docrt/testdata`.

---

## 1. Contexto

Um motor excelente com padrões ruins gera documento feio. O que decide se o PDF sai bonito sem
esforço do autor (humano ou modelo) é a biblioteca: tokens de design, componentes prontos e modelos
completos. Como não há segunda linguagem para aprender, o modelo de IA só escolhe o modelo e preenche
os dados.

## 2. Arquitetura (camadas, cada uma usa só as de baixo)

```text
modelos/*.typ   cv-moderno, recibo, cobranca-pix, relatorio ...   (documento completo)
componentes.typ titulo-secao, avatar, barra-nivel, etiqueta,        (peças reutilizáveis)
                linha-do-tempo, destaques ...
tema.typ        escala tipográfica, grade de 4 pt, paleta derivada  (tokens)
fonts/          pacotes de fonte OFL, embutidos à parte
```

- Um modelo **nunca** usa tamanho ou cor solta: só os tokens. Trocar uma cor (`"cor": "#0f766e"`)
  recolore o documento inteiro, com contraste garantido pelos neutros fixos.
- Todo campo de dados é opcional (`tem`, `campo`, `lista`). Dados mínimos (`{"nome": "Ana"}`)
  precisam renderizar.
- Cada modelo tem dados de exemplo em `pkg/docrt/testdata/`. O teste
  `TestEveryTemplateRendersSample` renderiza todos e confere o determinismo, então modelo novo sem
  exemplo reprova.

## 3. Estado

| Item | Estado |
|---|---|
| `tema.typ` (escala, espaçamento, paleta de uma cor) | Feito |
| `componentes.typ` (título de seção, avatar de iniciais, barra de nível, etiqueta(s), rótulo/valor, linha do tempo, destaques) | Feito |
| Fonte Inter (6 faces) | Feito |
| **`cv-moderno`**: duas colunas, lateral tingida, linha do tempo, barras de habilidade, rodapé a partir da 2ª página | Feito, 2 rodadas estéticas (§5) |
| `recibo` / `nota-de-pedido` (tabela paginada, totais, `moeda-br`) | A fazer |
| `cobranca-pix`: **QR Code EMV** | Protótipo em `examples/cobranca_pix_pdf` via `@preview/tiaoma` (pacote embutido no build). Falta virar modelo e gerar o BR Code com CRC16 válido a partir dos dados |
| `boleto`: código de barras ITF-25 + linha digitável | A fazer; o `tiaoma` (zint) já cobre o ITF-25 |
| `relatorio`: capa, sumário, cabeçalho/rodapé, gráficos | A fazer |
| `certificado` e `etiqueta-envio` (camada de desenho) | A fazer |
| Temas alternativos (`classico` serifado, `compacto`) | A fazer; a Libertinus Serif já está embutida |
| Foto no currículo | Feito: `pdf-template-files` + campo `foto` + componente `avatar-foto` |
| Formatação BR (`moeda-br`, `data-br`, CPF/CNPJ/CEP) como funções Typst | A fazer |

## 4. Fora do escopo

Pacotes do Typst Universe baixados **em tempo de execução**: o motor não tem rede por decisão. Um
documento pode citar pacotes, e o `liafc build` os embute com trava SHA-256 (§4.5 da #034). Um modelo
da biblioteca que dependa de pacote deve vendorizá-lo em `lib/`, com a licença, para funcionar também
nos testes do repositório.

## 5. Rodadas estéticas

Cada modelo passa por rodadas: gerar o PNG (`LIAF_DOC_OUT`), inspecionar e corrigir.

**`cv-moderno`**
1. Base: duas colunas, avatar, linha do tempo e barras.
2. Cargo encostava no nome (descendente do "q"). Títulos, subtítulos e corpo estavam apertados. A
   linha do tempo era interrompida entre itens. Texto base de 9,5 → 10 pt. Todos corrigidos.

A partir da #037, as rodadas passam a ter métricas objetivas além da inspeção visual.

## 6. Critério de conclusão

Pelo menos `cv-moderno`, `recibo`, `cobranca-pix` e `relatorio` entregues, cada um com dados de
exemplo, teste e PNG de referência aprovado. Um segundo tema funcionando com os mesmos modelos.
