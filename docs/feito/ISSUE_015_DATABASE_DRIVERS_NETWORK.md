# Issue #015: Drivers Nativos para Bancos de Dados Externos (PostgreSQL, MySQL, Redis)

**Status:** Concluída e Validada em Servidores Reais (27/09/2026)  
**Componente:** `pkg/dbdrv`, `pkg/runtime`, `pkg/checker`, `pkg/codegen`, `pkg/parser`, `pkg/ast`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Motivação
Atualmente, o LIAF suporta persistência orientada a filesystem (`fs-read-file`, `fs-write-atomic`, serialização JSON). Para aplicações corporativas e microsserviços com escala horizontal, agentes de IA necessitam de acesso direto e tipado a bancos relacionais e chave-valor externos através de conexões de rede seguras.

Sem suporte a banco de dados:
1. Aplicações dependem de arquivos locais que não escalam com múltiplos nós.
2. Não há suporte a transações ACID distribuídas.
3. Não há cache distribuído (Redis).

---

## 2. Proposta Técnica

### 2.1. Novo Efeito `net` / `db`
- Operações de banco exigem o efeito `db` (ou subefeito de `net`).
- Exemplo: `(effects net db)`.

### 2.2. Interface Declarativa de Conexão e Query
- Builtin `(db-connect driver dsn)` retornando `(result DBConnection str)`.
  - Drivers iniciais: `"postgres"`, `"mysql"`, `"redis"`.
- Execução de queries parametrizadas (anti SQL Injection compulsório para IAs):
  - `(db-query conn "SELECT id, name, email FROM users WHERE id = $1" (list id) User)` -> retorna `(result (list User) str)`.
  - `(db-exec conn "UPDATE users SET email = $1 WHERE id = $2" (list email id))` -> retorna `(result int str)` (linhas afetadas).
- Builtins específicos para Redis (chave-valor):
  - `(redis-get conn key)` -> `(result str str)`.
  - `(redis-set conn key value ttl-seconds)` -> `(result void str)`.

### 2.3. Transações Atômicas
- Bloco `(db-transaction conn ...)` que realiza commit ou rollback automático caso ocorra erro.

---

## 3. Tarefas de Implementação
- [x] Definir tipo opaco `DBConnection` no sistema de tipos (`pkg/checker`).
- [x] Adicionar validação estrita de SQL parametrizado no checker (rejeitar interpolação de strings em queries).
- [x] Implementar bindings de conexão e pool no runtime Go — **com protocolo nativo, não com drivers externos** (ver §4).
- [x] Mapear conversão automática de rows para structs LIAF.
- [ ] Adicionar suporte no backend C via `libpq` / sockets nativos. **Não feito.** O backend C recusa os builtins de banco com "unsupported operation"; integrá-los depende de a toolchain C da issue #010 estar completa.
- [x] Criar exemplo prático em `examples/db_postgres.liaf` (mais `examples/db_redis.liaf`).

---

## 4. Decisões tomadas na implementação

Três pontos onde o que foi entregue difere do que a proposta acima descrevia. Registrados aqui para que a divergência seja deliberada, e não esquecimento.

### 4.1 Protocolo nativo em vez de `pgx`/`go-sql-driver`/`go-redis`

A §3 nomeava os drivers externos; a §1 e o `PROXIMOS_PASSOS.md` pediam "driver de rede nativo". Prevaleceu o segundo, por decisão explícita durante a implementação.

`pkg/dbdrv` implementa sobre TCP puro o protocolo de mensagens do PostgreSQL (wire v3, com extended query e autenticação cleartext/MD5/SCRAM-SHA-256), o protocolo 41 do MySQL (prepared statements binários, `mysql_native_password` e `caching_sha2_password` incluindo o caminho lento com RSA) e o RESP2 do Redis.

Motivo: os drivers externos acrescentariam cerca de 18 módulos transitivos a `go.mod`, e `pkg/runtime` os importaria incondicionalmente — todo binário LIAF, inclusive um "olá mundo", passaria a linkar os três. Isso contradiz a tese do binário único e a issue #010. O custo é o volume de código de protocolo, coberto por testes contra servidores falsos e pelos vetores da RFC 7677.

### 4.2 Parâmetros variádicos em vez de `(list ...)`

A proposta escrevia `(db-query conn sql (list id) User)`. Listas na LIAF são homogêneas, e uma consulta com `id` inteiro e `email` texto não caberia em `(list T)` nenhum — a forma exigiria inventar um tipo de tupla só para isso.

A forma entregue é `(db-query conn "SQL" TipoLinha arg...)` e `(db-exec conn "SQL" arg...)`, com os valores como argumentos posicionais escalares. Mesma intenção, sem tipo novo, e mais curta.

### 4.3 O tipo de linha tem de ser struct

`db-query` recusa tipo de linha escalar (`E_INVALID_TYPE`). A conversão casa nome de coluna com nome de campo, e um escalar não tem nome de campo. Para `SELECT count(*)`, declare uma struct de um campo — é explícito e não custa nada em execução.

---

## 5. O que foi verificado

- `pkg/dbdrv`: servidor PostgreSQL falso exercitando SSLRequest, autenticação MD5 e o ciclo Parse/Bind/Describe/Execute/Sync; servidor Redis falso exercitando AUTH, SELECT, GET, SET e TTL; SCRAM-SHA-256 conferido contra o vetor de teste da RFC 7677; enquadramento do MySQL testado peça a peça (lenenc, greeting, provas de senha, `COM_STMT_EXECUTE`, decodificação binária).
- Anti-injeção provado dos dois lados: o checker recusa `(concat ...)` em posição de SQL, e o teste de protocolo confirma que um valor hostil (`x'; DROP TABLE users; --`) chega ao servidor como parâmetro do `Bind`, com o texto do comando intacto.
- `pkg/runtime`: conversão de linha em struct (incluindo `NULL`, resultado vazio e o apelido `snake_case`/`kebab-case`), pool (conexão de transporte quebrado é descartada, erro de SQL não invalida a sessão), transação (commit, rollback idempotente, conexão fixada, recusa de aninhamento).
- `examples/db_redis.liaf` compilado e **executado** contra um servidor RESP falso: `db-connect`, `redis-set`, `redis-get`, `json-decode` e o erro de chave ausente, tudo pelo binário gerado.
- `examples/db_postgres.liaf` compilado e executado; falha de conexão chega ao usuário como mensagem, sem panic.

### 5.1 Validação com Servidores Reais (27/09/2026)

Ambiente Docker configurado com `docker-compose.test.yml`:
- **PostgreSQL 16 (Alpine):** Testado handshake, SSLRequest, DDL, DML parametrizado, transações ACID e Rollback.
- **MySQL 8.0:** Validado caminho completo de autenticação moderna `caching_sha2_password` com troca de chaves RSA em texto puro (`tls=false`), enquadramento binário, prepared statements e transações.
- **Redis 7 (Alpine):** Validado protocolo RESP2 com autenticação via senha, comandos `SET` (com TTL), `GET` e tratamento de chaves inexistentes.
- **Testes Go automatizados:** `TestRealDatabases` em `pkg/dbdrv/dbdrv_real_test.go` passou 100%.
- **Execução ponta a ponta nativa:** Executados `db_postgres.liaf`, `db_mysql.liaf` e `db_redis.liaf` diretamente pelo compilador `liafc run`.

---

Contrato atual e especificações: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Sintaxe: seção 19 de [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidências consolidadas: [STATUS.md](../STATUS.md).
