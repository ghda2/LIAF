# Comandos e opções

## liafc doc — conferir o documento sem compilar o programa

```text
liafc doc check   doc.typ [--dados d.json] [--arquivo nome=caminho]... [--json] [--layout]
liafc doc preview doc.typ [--dados d.json] [--arquivo nome=caminho]... [-o p.png] [--pagina N] [--ppi N]
liafc doc watch   doc.typ [mesmas opções]
```

| Opção | Efeito |
|---|---|
| `--dados d.json` | Entrega o arquivo como `/dados.json`. Sem ela, os dados são `{}` |
| `--arquivo nome=caminho` | Entrega um arquivo extra em `/nome` (repita para vários). Ex.: `--arquivo img/logo.png=./logo.png` |
| `--json` | Saída em JSON para o agente (ver [laco-de-correcao.md](laco-de-correcao.md)) |
| `--layout` | Inclui no JSON a caixa do conteúdo de cada página e os elementos fora dela |
| `-o p.png` | Onde gravar a prévia (padrão: `doc-p1.png`) |
| `--pagina N` | Qual página a prévia mostra (padrão 1) |
| `--ppi N` | Resolução da prévia (padrão 144) |

- `check` sai com código **1** se houver erro e **0** caso contrário (avisos não mudam o código).
- `watch` reconfere a cada vez que o documento, os dados ou um arquivo extra mudam. Com `-o`, regrava a
  prévia. Encerra com Ctrl+C.
- Os pacotes `@preview` citados no `.typ` são resolvidos como no build.

## liafc check / run / build — o programa

```text
liafc check prog.liaf [--json]      # tipos e sintaxe do programa LIAF
liafc run   prog.liaf               # compila e executa na pasta atual
liafc build prog.liaf -o app [--doc-fonts=inter,serif,math,mono]
```

- O `run` e o `build` detectam as citações `@ns/nome:versão` no programa, no `.liaf` e nos `.typ` da
  mesma pasta, baixam na primeira vez (precisa de internet), conferem o SHA-256 em `liaf-typst.lock`
  (ao lado do `.liaf`; **versione esse arquivo**) e embutem os pacotes no binário.
- `--doc-fonts` escolhe quais famílias entram no binário. Sem a opção, entram todas. Exclua só as que o
  documento comprovadamente não usa: `serif` é a fonte padrão do Typst quando o documento não escolhe
  nenhuma, `math` é exigida por qualquer fórmula e `mono` por blocos de código.
- Um programa que não usa `pdf-*`/`png-*` não recebe o motor: o binário não cresce.

## Variáveis de ambiente

| Variável | Efeito |
|---|---|
| `LIAF_DOC_WARMUP=0` | Não carregar o motor em segundo plano ao iniciar o programa (padrão: carrega) |
| `LIAF_TYPST_PACKAGES` | Pasta de cache dos pacotes baixados (padrão: `<cache do usuário>/liaf/typst-packages`) |

## Limites e comportamento do motor

- Até 64 MB de arquivos extras por documento; 30 s por renderização.
- A mesma entrada gera o mesmo PDF byte a byte.
- O PDF sai marcado (acessível) por padrão.
- A primeira carga do motor num computador leva alguns segundos (compilação), e depois cerca de 0,5 s
  graças ao cache. Um PDF comum leva de 30 a 50 ms.
