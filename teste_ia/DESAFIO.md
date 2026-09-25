# Desafio: Gerenciador de Clientes com Banco de Dados PostgreSQL

## Objetivo:
Construa um módulo em LIAF v0.4 chamado `solucao` que implemente operações seguras de banco de dados relacional usando o efeito nativo `db`.

## Requisitos Técnicos Estritos:

1. **Structs:**
   - `Cliente`: campos `id` (int), `nome` (str), `email` (str), `ativo` (bool).
   - `NovoCliente`: campos `id` (int), `nome` (str), `email` (str).

2. **Operações de Banco de Dados (com efeito db):**
   - Função `inicializar-banco`: recebe `(db DBConnection)`, retorna `(result bool str)`, declara `(effects db)`. Executa via `db-exec` o SQL literal:
     `"CREATE TABLE IF NOT EXISTS clientes (id int PRIMARY KEY, nome text NOT NULL, email text NOT NULL, ativo bool NOT NULL)"` e retorna `(ok true)`.
   - Função `inserir-cliente`: recebe `(db DBConnection) (cliente NovoCliente)`, retorna `(result bool str)`, declara `(effects db)`.
     Lê os campos de `cliente` com `(field ...)` e executa via `db-exec`:
     `"INSERT INTO clientes (id, nome, email, ativo) VALUES ($1, $2, $3, $4)"` passando o id, nome, email e `true`. Retorna `(ok true)`.
   - Função `listar-clientes`: recebe `(db DBConnection)`, retorna `(result (list Cliente) str)`, declara `(effects db)`.
     Executa via `db-query`:
     `"SELECT id, nome, email, ativo FROM clientes ORDER BY id"` mapeando para a struct `Cliente`. Retorna a lista obtida.
   - Função `desativar-cliente`: recebe `(db DBConnection) (id int)`, retorna `(result bool str)`, declara `(effects db)`.
     Executa via `db-exec`:
     `"UPDATE clientes SET ativo = $1 WHERE id = $2"` passando `false` e o `id`. Retorna `(ok true)`.

3. **Função Principal:**
   - `fn main`: declara `(effects io db)`, conecta ao banco `"postgres"` com a string de conexão `"postgres://user:pass@localhost:5432/db"`. Se conectar com sucesso, imprime `"Conectado ao banco"`. Se falhar, imprime o erro.

4. **Atenção Máxima às Regras:**
   - Lembre-se: `(let nome tipo valor)` SEMPRE possui o tipo explícito.
   - O SQL DEVE ser uma string literal, nunca use concatenação para montar a consulta.
   - Onde acessar banco declare `(effects db)`. Onde imprimir no terminal declare `(effects io)`.
