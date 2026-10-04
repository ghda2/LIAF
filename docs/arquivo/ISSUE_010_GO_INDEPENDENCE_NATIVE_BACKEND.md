# Issue #010: Emancipação do Go — Backend Nativo e Autonomia do Compilador

**Status:** Parcial. Conferido em 26/09/2026: `go test ./pkg/codegen/c` passa nos quatro cenários (a colisão do Windows SDK foi resolvida com o prefixo `LIAF_`), mas a CLI não tem `--backend` e nada foi testado sem Go no PATH.
**Componente:** `pkg/codegen`, `cmd/liafc`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Objetivo
Atualmente, o `liafc run` e `liafc build` geram código Go intermediário e invocam o compilador Go do sistema (`exec.Command("go", "build", ...)`). 

Para tornar a LIAF uma linguagem 100% autônoma, ela **não pode exigir o Go instalado na máquina do usuário**. O usuário deve baixar um único executável `liafc` e conseguir compilar seus programas diretamente para binários nativos de máquina (Linux ELF, Windows PE, macOS Mach-O).

---

## 2. Estratégia Técnica de Desacoplamento

### Fase 2.1: Backend C com Compilador Embutido (Abordagem TCC)
- O codegen emite código C ANSI limpo e determinístico.
- O executável `liafc` embutirá a biblioteca do **Tiny C Compiler (TCC)** ou distribuirá um toolchain estático pré-compilado de ~1 MB.
- Compilação direta em memória (in-memory execution) em menos de 15ms, sem qualquer dependência externa no host.

### Fase 2.2: Backend Nativo Direto (Cranelift / LLVM)
- Emissão de IR (Intermediate Representation) do **Cranelift** (utilizado pelo Wasmtime e Rust para codegen ultrarrápido).
- Geração de binários nativos stand-alone sem intermediários de texto.

---

## 3. Tarefas de Implementação
- [ ] Desenhar o gerador intermediário de código C puro em `pkg/codegen/c/`.
- [ ] Integrar runtime minimalista de gerenciamento de memória e concorrência LIAF em C (green threads ou libuv light).
- [ ] Adicionar flag `liafc build --backend=native|c|go`.
- [ ] Testar geração de binário estático no Linux e Windows sem chamar a toolchain Go.
- [ ] Validar tempo de compilação inferior a 50ms para projetos médios.


Contrato atual e ajustes de sintaxe: [docs/linguagem/SPEC.md](../linguagem/SPEC.md). Evidencias consolidadas: [STATUS.md](../STATUS.md).
