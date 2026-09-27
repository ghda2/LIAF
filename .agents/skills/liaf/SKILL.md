---
name: liaf
description: Guia e workflow para agentes de IA escreverem, validarem com auto-cura, compilarem e realizarem deploy de aplicações na linguagem LIAF (Language for AI First).
---

# LIAF — AI Agent Skill

Este guia capacita agentes de IA a interagir, gerar código, auto-curar erros e fazer deploy de aplicações utilizando a linguagem **LIAF (Language for AI First)**.

---

## 1. Sintaxe Canônica — LIAF v0.5 (Compacta e Redução de Tokens)

A fonte oficial é em S-expressions compactas. A sintaxe de tags redundantes (`fields`, `params`, `returns`, `body` e `return` final obrigatório) pertence a versões antigas; utilize sempre a **forma canônica v0.5**:

```liaf
(module exemplo
  (struct Task (id int) (title str) (done bool))

  (fn soma ((a int) (b int)) int (effects)
    (add a b)))
```

Regras:

1. **Assinatura compacta sem tags redundantes:**
   - Funções: `(fn nome (params) retorno (effects...) stmts...)`
   - Rotas: `(route METODO path (params) retorno (effects...) stmts...)`
   - Parâmetros vazios são expressos por `()`: `(fn main () void (effects io) (println "Olá"))`
2. **Retorno implícito:** Em funções e rotas não-void, a última expressão avaliada é o retorno implícito:
   ```liaf
   (fn dobro ((n int)) int (effects)
     (mul n 2))
   ```
3. **Struct compacta:** Sem tag `fields`: `(struct User (id int) (name str) (email str))`.
4. **`match` como expressão de valor:** Atribui diretamente a variáveis sem necessidade de mutação imperativa com `set`:
   ```liaf
   (let token str
     (match (request-header req "Authorization")
       (ok t t)
       (err _ "")))
   ```
5. **Combinador `unwrap-or`:** Desempacota `(result T E)` ou `(option T)` com fallback em 1 linha:
   ```liaf
   (let status str (unwrap-or (request-query req "status") "todos"))
   ```
6. **Interpolação com `(fmt ...)`:** Substitui chamadas recursivas de `concat`:
   ```liaf
   (println (fmt "Usuário {} conectado na sala {}" user room))
   ```
7. **Tratamento de erro com `try` e `on-err`:**
   ```liaf
   (fn salvar ((estado Estado)) Response (effects fs io)
     (on-err message (json-response 500 message))
     (let texto str (try (json-encode estado)))
     (try (fs-write-atomic "estado.json" texto))
     (json-response 200 "{\"ok\":true}"))
   ```
8. **Efeitos obrigatórios:** `io`, `fs`, `net`, `clock`, `spawn`, `db`. São transitivos. Pura é `(effects)`.

### Rotas HTTP declarativas

```liaf
(route PUT "/tasks/{id}" ((id int) (input UpdateInput)) Response (effects fs io)
  (on-err message (json-response 500 message))
  (let estado Estado (try (carregar-estado)))
  (atualizar estado id input))
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

Exemplo completo e executável: `pkg/codegen/testdata/task_api_v03.liaf`.

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
liafc fmt -l *.liaf  # lista o que está fora do formato
```
O formatador ainda não preserva comentários, então `-w` recusa gravar em arquivo comentado sem
`--drop-comments`.

### Compilar localmente (Windows)
```bash
liafc build ./app.liaf -o server.exe
```

### Cross-compilar para Linux (Produção x86_64)
No PowerShell:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; liafc build ./app.liaf -o liaf_server_linux; Remove-Item Env:\GOOS; Remove-Item Env:\GOARCH
```

### Deploy Remoto via SSH
1. Enviar binário e pasta pública:
   ```bash
   scp -B -C -r liaf_server_linux public/ HOST:/opt/app/
   ```
2. Configurar permissão e serviço systemd (`/etc/systemd/system/app.service`).
3. Se houver proxy reverso (Caddy / Nginx), apontar para a porta configurada no script LIAF.
