# LIAF — Language for AI First

Linguagem experimental com AST textual em S-expressions, tipos e efeitos explícitos, diagnósticos JSON e compilação via Go. A hipótese de reduzir o custo de programação por modelos menores deve ser medida com benchmarks.

```liaf
(module hello
  (fn main (params) (returns void) (effects io)
    (body (do (call println "Hello, LIAF")))))
```

```powershell
go build -o liafc.exe ./cmd/liafc
./liafc.exe check examples/loops.liaf --json
./liafc.exe run examples/collections.liaf
./liafc.exe build examples/web_engine.liaf -o site.exe --embed=public
go test ./...
```

Recursos: loops, listas e mapas tipados, `Result`/`match`, arquivos e JSON, concorrência por canais, rotas HTTP, SSG Markdown, layouts, includes, gzip, ETags, streaming de mídias e publicação autenticada. A compilação pelo backend Go requer Go e este repositório; o executável gerado funciona sem Go instalado.

- [Contrato implementado](.docs/conceitos/IMPLEMENTATION.md)
- [Status das issues](.docs/STATUS.md)
- [Próximos passos e checklist](.docs/PROXIMOS_PASSOS.md)
- [Registro para retomar o trabalho](.docs/RETOMADA.md)
- [Especificação v0.2](.docs/conceitos/SPEC_V2.md)
- [Guia de agentes](.docs/regras/AI_AGENT_GUIDE.md)
- [Arquitetura](.docs/conceitos/ARCHITECTURE.md)
- [Roadmap](.docs/conceitos/ROADMAP.md)
- [Logs históricos](.docs/logs/LOG_2026-09-16_PIPELINE_JORNALISMO_IA.md)

Publish/reload são desabilitados sem token configurado. Defina `LIAF_DEPLOY_TOKEN` e use POST com `Authorization: Bearer ...`. Assets embutidos são imutáveis; publicação dinâmica requer diretório em disco.
