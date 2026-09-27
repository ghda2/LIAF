# LIAF: uma linguagem experimental para agentes

LIAF representa programas como uma AST textual com formas explícitas, tipos, efeitos e diagnósticos JSON. O compilador atual gera Go e inclui um motor web com SSG, publicação autenticada e streaming de mídia.

Experimente os exemplos em `examples/` e execute `go test ./...`. A demonstração historicamente publicada está em https://tw.webdrop.bio; sua disponibilidade não é monitorada por este documento.

A hipótese central é reduzir o custo total até software correto usando modelos menores. O runner em `benchmarks/` registra tentativas, tokens e validação comportamental. Ainda não há resultado comparativo multi-modelo publicado nesta revisão. Não alegamos pass@1 de 90%, economia de 10x ou RAM universal de 2 MB.

Para publicar o lançamento: anexar commit/release reproduzível, binários com hashes, ambiente de medição e dados brutos do benchmark antes de acrescentar números comparativos.
