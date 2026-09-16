---
name: liaf
description: Guia e workflow para agentes de IA escreverem, validarem com auto-cura, compilarem e realizarem deploy de aplicações na linguagem LIAF (Language for AI First).
---

# LIAF — AI Agent Skill

Este guia capacita agentes de IA a interagir, gerar código, auto-curar erros e fazer deploy de aplicações utilizando a linguagem **LIAF (Language for AI First)**.

---

## 1. Sintaxe — LIAF v0.3

A fonte é uma AST textual em S-expressions. **A sintaxe de tags `[fn nome ...] /fn nome]` é da v0.1
e o compilador a rejeita.** Não a use.

```liaf
(module exemplo
  (struct Task (fields (id int) (title str) (done bool)))

  (fn soma (params (a int) (b int)) (returns int) (effects)
    (body (return (add a b)))))
```

Regras:

1. **Chamada direta:** `(println "oi")`, `(json-encode task)`, `(add a b)`. A forma `(call ...)` da
   v0.2 ainda é aceita, mas não é a canônica.
2. **Toda função declara `params`, `returns` e `effects`.** Efeitos são `io`, `fs`, `net`, `clock`,
   `spawn`, e são transitivos: se você chama uma função com `fs`, você declara `fs`. Declarar um
   efeito que o corpo não usa é erro (`E_UNUSED_EFFECT`). Função pura é `(effects)`, nunca
   `(effects none)`.
3. **Operações falíveis devolvem `(result T E)` e não podem ser ignoradas.**

### Tratamento de erro: `try` e `on-err`

`(try expr)` desempacota um `(result T E)` e, no erro, sai da função na hora. O destino é o bloco
`(on-err var ...)`, que vem **antes** de `(body ...)`:

```liaf
(fn salvar (params (estado Estado)) (returns Response) (effects fs io)
  (on-err message (return (json-response 500 message)))
  (body
    (let texto str (try (json-encode estado)))
    (try (fs-write-atomic "estado.json" texto))
    (return (json-response 200 "{\"ok\":true}"))))
```

Use `try` quando o erro só precisa ser propagado — é a maioria dos casos, e evita a pirâmide de
`match` aninhado. Use `(match expr (ok v ...) (err m ...))` quando os dois lados fazem coisas
diferentes.

### Rotas HTTP declarativas

`route` é declaração de topo, irmã de `fn`, e se registra sozinha:

```liaf
(route PUT "/tasks/{id}" (params (id int) (input UpdateInput)) (returns Response) (effects fs io)
  (on-err message (return (json-response 500 message)))
  (body
    (let estado Estado (try (carregar-estado)))
    (return (atualizar estado id input))))
```

- Cada `{nome}` no caminho precisa de um parâmetro homônimo, `int` ou `str`. Ele chega convertido.
- Parâmetro de struct recebe o corpo JSON já desserializado; corpo inválido vira `400` sozinho.
- Não chame `http-get` para uma rota declarativa.

### Armadilhas comuns

- `fs-write-file`, `fs-rename` e `fs-write-atomic` retornam `(result void str)`. `ok` já significa
  que gravou; não existe `ok false` para testar.
- `json-encode` de `(list T)` produz array JSON nativo `[...]`. Não monte colchetes com `concat`.
- Structs são imutáveis: não há `set-field`, construa uma nova com `(new Tipo ...)`.
- Leitura de campo é `(field objeto campo)`.

Exemplo completo e executável: `examples/task_api_v03.liaf`.

---

## 2. Loop de Auto-Cura (Auto-Healing Loop)

Sempre que gerar ou editar um arquivo `.liaf`, execute o ciclo de validação autônoma:

### Passo 1: Executar o checker com saída JSON
```bash
liafc check <caminho/arquivo.liaf> --json
```

### Passo 2: Analisar a resposta JSON
- Se `"status": "success"`, prossiga para a compilação.
- Se `"status": "error"`, examine a lista `errors`:
  - `code`: Tipo do erro (ex: `TAG_NAME_MISMATCH`, `TYPE_MISMATCH`, `CHANNEL_TYPE_MISMATCH`).
  - `line` e `col`: Coordenadas exatas no arquivo.
  - `suggested_patch`: Correção pontual sugerida pelo compilador.

### Passo 3: Aplicar o patch e revalidar
Substitua o trecho defeituoso com base em `suggested_patch` e execute o `liafc check` novamente até obter `status: success`.

---

## 3. Web Engine Autônomo

Para criar um servidor web estático de alta performance (RAM < 4 MB):

### Código LIAF (`web_engine.liaf`)
```liaf
(module web-engine
  (fn main (params) (returns void) (effects fs io net)
    (body
      (do (println "Iniciando LIAF Web Engine na porta 7070..."))
      (do (serve-site "./public" "" "7070" false)))))
```
- Argumentos de `serve-site`:
  1. Diretório público (ex: `"./public"` com index.html, style.css, js).
  2. Domínio para Auto-TLS (ou `""` para desativar).
  3. Porta (ex: `"7070"`).
  4. Auto-TLS booleano (`false` ou `true`).

### Componentização e Layouts em RAM
O LIAF Web Engine suporta montagem de templates diretamente na inicialização em RAM:
- **Layout Base:** Use `<!-- content -->` ou `<!-- slot -->` no arquivo base (ex: `base.html`).
- **Páginas Filhas:** No topo do arquivo HTML da página, declare `<!-- layout "base.html" -->`.
- **Inclusão de Parciais:** Use `<!-- include "menu.html" -->` ou `<!-- include "footer.html" -->`.
- **URLs Limpas:** Rotas como `/sobre` ou `/servicos` servem automaticamente `sobre.html` e `servicos.html`.
- **Versionamento Automático (Zero Ctrl+F5):** O runtime calcula o hash SHA1 dos arquivos CSS/JS e injeta automaticamente `?v=<hash>` nas tags `<link>` e `<script>` do HTML.
- **Hot-Reload VFS:** Chame `POST /_liaf/reload` para recarregar todos os arquivos na RAM em ~3ms sem reiniciar o processo.

---

## 4. Pipeline de Compilação e Deploy

### Formatar na forma canônica
```bash
liafc fmt arquivo.liaf        # imprime
liafc fmt -l examples/*.liaf  # lista o que está fora do formato
```
O formatador ainda não preserva comentários, então `-w` recusa gravar em arquivo comentado sem
`--drop-comments`.

### Compilar localmente (Windows)
```bash
liafc build ./examples/web_engine.liaf -o server.exe
```

### Cross-compilar para Linux (Produção x86_64)
No PowerShell:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; liafc build ./examples/web_engine.liaf -o liaf_server_linux; Remove-Item Env:\GOOS; Remove-Item Env:\GOARCH
```

### Deploy Remoto via SSH
1. Enviar binário e pasta pública:
   ```bash
   scp -B -C liaf_server_linux public/ HOST:/opt/app/
   ```
2. Configurar permissão e serviço systemd (`/etc/systemd/system/app.service`).
3. Se houver proxy reverso (Caddy / Nginx), apontar para a porta configurada no script LIAF.
