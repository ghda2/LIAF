# Guia para agentes — LIAF v0.2

Use S-expressions. As tags `[fn ... /fn ...]` pertencem à v0.1 e não são aceitas pelo compilador atual. Exemplos antigos na skill local são históricos; consulte este guia e [IMPLEMENTATION.md](../conceitos/IMPLEMENTATION.md).

```liaf
(module exemplo
  (fn main (params) (returns void) (effects io)
    (body (do (call println "Olá")))))
```

1. Execute `liafc check arquivo.liaf --json` após cada alteração.
2. Leia `status` e `errors`. Corrija a causa e revalide. O protocolo atual fornece código, arquivo, linha, coluna e mensagem; não oferece ainda todos os patches/intervalos propostos na SPEC_V2.
3. Execute testes de comportamento; passar no checker não prova que o algoritmo está correto.
4. Compile com `liafc build arquivo.liaf -o programa`. Use `--embed=public` para assets imutáveis no executável.

Declare efeitos transitivos (`io`, `fs`, `net`, `clock`, `spawn`). Trate operações falíveis com `match` ou retorne `Result`. Exemplos executáveis: `examples/loops.liaf`, `collections.liaf`, `fs_json.liaf`, `result.liaf` e `api_server.liaf`.

Para sites, `serve-site` recebe pasta, domínio, porta e auto-TLS. `serve-hybrid` oferece rotas dinâmicas junto aos arquivos estáticos e retorna um resultado tratável.

Publicação: `liafc publish pagina.md --url https://HOST/_liaf/publish --path blog/pagina.md`. Defina `LIAF_DEPLOY_TOKEN` no cliente e no servidor. Reload também exige POST autenticado. Não coloque credenciais em código ou documentação.

Deploy: inspecione a unidade com `liafc service install --name site --bin /opt/site/server --dry-run`. Para Caddy: `liafc deploy caddy-bind --domain HOST --upstream HOST:PORT --server srv0`.
