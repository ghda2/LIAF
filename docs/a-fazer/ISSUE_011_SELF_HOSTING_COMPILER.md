# Issue #011: Compilador Auto-Hospedado (Self-Hosting LIAF)

**Status:** Pendente: bootstrap ainda nao implementado
**Componente:** `compiler/`, `std/compiler/`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
O ápice da maturidade de qualquer linguagem de programação é o **self-hosting**: a capacidade do compilador ser escrito na própria linguagem que ele compila (`liafc.liaf`).

Ao reescrever o compilador LIAF em LIAF, alcançamos:
1. **Validação Definitiva da Linguagem:** Prova prática de que a LIAF é robusta o suficiente para implementar sistemas complexos e de baixo nível.
2. **Ciclo Fechado de IA:** Agentes de IA que programam em LIAF passam a conseguir evoluir, refatorar e consertar o próprio compilador autonomamente.

---

## 2. Etapas de Migração

### 2.1. Requisitos Prévios
- Conclusão das Issues #002 (Loops), #003 (Listas/Mapas), #005 (FS/JSON) e #006 (Result/Error Handling).
- Type Checker e Effect System estáveis.

### 2.2. Arquitetura do Compilador em LIAF
1. **`compiler/token.liaf`**: Definição dos tipos de tokens e posições.
2. **`compiler/lexer.liaf`**: Tokenizador orientado a streaming de bytes.
3. **`compiler/parser.liaf`**: Parser recursivo descendente produzindo AST tipada.
4. **`compiler/check.liaf`**: Verificador semântico de tipos e efeitos.
5. **`compiler/codegen.liaf`**: Emissor de código nativo.

### 2.3. Processo de Bootstrapping
- **Estágio 0:** O compilador Go atual compila o código fonte `compiler/*.liaf` e gera o binário `liafc-stage1`.
- **Estágio 1:** O binário `liafc-stage1` compila `compiler/*.liaf` novamente e gera o `liafc-stage2`.
- **Verificação:** `hash(liafc-stage1) == hash(liafc-stage2)` comprova compilação determinística e self-hosting completo.

---

## 3. Tarefas de Implementação
- [ ] Portar lexer de Go para sintaxe canônica LIAF em `compiler/lexer.liaf`.
- [ ] Portar parser e estruturas de AST para LIAF em `compiler/parser.liaf`.
- [ ] Implementar pipeline de compilação de 2 estágios (bootstrap test).
- [ ] Arquivar compilador Go original como legado/referência bootstrap.


Contrato atual e ajustes de sintaxe: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
