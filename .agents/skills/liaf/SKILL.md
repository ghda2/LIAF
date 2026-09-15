---
name: liaf
description: Guia e workflow para agentes de IA escreverem, validarem com auto-cura, compilarem e realizarem deploy de aplicações na linguagem LIAF (Language for AI First).
---

# LIAF — AI Agent Skill

Este guia capacita agentes de IA a interagir, gerar código, auto-curar erros e fazer deploy de aplicações utilizando a linguagem **LIAF (Language for AI First)**.

---

## 1. Princípios Sintáticos Inegociáveis

Ao gerar código `.liaf`, obedeça estritamente:

1. **Tags de Fechamento Nomeadas Obrigatórias:**
   Toda função ou struct aberta com `[fn nome ...]` ou `[struct nome ...]` DEVE ser fechada com a mesma tag e o mesmo identificador:
   ```liaf
   [fn soma (a: int b: int) -> (int)
     [return (add a b)]
   /fn soma]
   ```
2. **Notação Prefixada para Operações:**
   Operadores binários usam notação de prefixo:
   - Adição: `(add a b)`
   - Subtração: `(sub a b)`
   - Multiplicação: `(mul a b)`
   - Divisão: `(div a b)`
   - Comparação: `(eq a b)`, `(neq a b)`, `(gt a b)`, `(lt a b)`
3. **Declarações em Colchetes:**
   - Variáveis: `[let nome: tipo valor]`
   - Retorno: `[return valor]` ou `[return]`
   - Concorrência: `[spawn (funcao args)]`, `[send canal valor]`, `[recv canal]`
   - Condicionais: `[if condicao ... [else ... /else] /if]`

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
[fn main () -> (void)
  (serve_site "./public" "" "7070" false)
/fn main]
```
- Argumentos de `serve_site`:
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
