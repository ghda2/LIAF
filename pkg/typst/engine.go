// Package typst roda o motor de documentos da LIAF: o Typst compilado para WebAssembly
// (engines/typst), executado em Go puro pelo wazero. Não depende de Rust, de CGO nem de
// binário externo em tempo de execução.
//
//	eng, _ := typst.New(ctx, typst.Options{})
//	res, _ := eng.Render(ctx, typst.Job{Files: map[string][]byte{"/main.typ": src}})
//	pdf := res.Outputs[0]
package typst

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"liaf/pkg/typst/fonts"
	"liaf/pkg/typst/wasm"
)

// Options configura um Engine. O valor zero é utilizável.
type Options struct {
	// Fonts são os pacotes carregados em cada instância. Vazio = fonts.Default().
	Fonts []fonts.Pack
	// Instances é o máximo de renderizações simultâneas (cada instância custa memória).
	// Padrão: min(GOMAXPROCS, 2).
	Instances int
	// CacheDir guarda o código de máquina gerado a partir do WASM. Sem ele, cada processo
	// recompila o motor ao iniciar. Padrão: <UserCacheDir>/liaf/wazero; "-" desliga.
	CacheDir string
	// EvictEvery descarta a memoização antiga do Typst a cada N renderizações por instância,
	// para a memória de um servidor não crescer sem limite. Padrão 32.
	EvictEvery int
}

// Engine é um conjunto de instâncias do motor, seguro para uso concorrente.
type Engine struct {
	rt       wazero.Runtime
	compiled wazero.CompiledModule
	fonts    [][]byte
	evict    int

	slots chan struct{}
	mu    sync.Mutex
	idle  []*instance
}

// New compila o motor e prepara o pool. A primeira instância é criada já aqui, para que um
// erro de carga (fonte inválida, wasm corrompido) apareça cedo.
func New(ctx context.Context, opts Options) (*Engine, error) {
	if opts.Instances <= 0 {
		opts.Instances = min(runtime.GOMAXPROCS(0), 2)
	}
	if opts.EvictEvery <= 0 {
		opts.EvictEvery = 32
	}
	packs := opts.Fonts
	if len(packs) == 0 {
		packs = fonts.Default()
	}
	if len(packs) == 0 {
		return nil, fmt.Errorf("typst: nenhuma família de fonte no binário (liafc build --doc-fonts precisa de pelo menos uma)")
	}

	cfg := wazero.NewRuntimeConfig()
	if dir := cacheDir(opts.CacheDir); dir != "" {
		if cache, err := wazero.NewCompilationCacheWithDir(dir); err == nil {
			cfg = cfg.WithCompilationCache(cache)
		}
	}
	rt := wazero.NewRuntimeWithConfig(ctx, cfg)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("typst: wasi: %w", err)
	}
	bin, err := wasm.Module()
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("typst: descomprimir motor: %w", err)
	}
	compiled, err := rt.CompileModule(ctx, bin)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("typst: compilar motor: %w", err)
	}

	e := &Engine{
		rt: rt, compiled: compiled, evict: opts.EvictEvery,
		slots: make(chan struct{}, opts.Instances),
	}
	for _, p := range packs {
		e.fonts = append(e.fonts, p.Files...)
	}
	first, err := newInstance(ctx, rt, compiled, e.fonts)
	if err != nil {
		rt.Close(ctx)
		return nil, err
	}
	e.idle = append(e.idle, first)
	return e, nil
}

// Render compila um documento. Erros do documento (sintaxe, fonte ausente, campo inexistente)
// vêm em Result.Diagnostics com OK=false; o error só é usado para falhas do próprio motor.
func (e *Engine) Render(ctx context.Context, job Job) (Result, error) {
	select {
	case e.slots <- struct{}{}:
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
	defer func() { <-e.slots }()

	in, err := e.take(ctx)
	if err != nil {
		return Result{}, err
	}
	res, err := in.render(ctx, job)
	if err != nil {
		// Uma armadilha (panic no Rust, falta de memória) deixa a instância inconsistente:
		// descarta e deixa o pool criar outra na próxima chamada.
		in.close(ctx)
		return Result{}, err
	}
	if in.uses%e.evict == 0 {
		in.evict(ctx, 8)
	}
	e.give(in)
	return res, nil
}

func (e *Engine) take(ctx context.Context) (*instance, error) {
	e.mu.Lock()
	if n := len(e.idle); n > 0 {
		in := e.idle[n-1]
		e.idle = e.idle[:n-1]
		e.mu.Unlock()
		return in, nil
	}
	e.mu.Unlock()
	return newInstance(ctx, e.rt, e.compiled, e.fonts)
}

func (e *Engine) give(in *instance) {
	e.mu.Lock()
	e.idle = append(e.idle, in)
	e.mu.Unlock()
}

// Close libera o runtime e todas as instâncias.
func (e *Engine) Close(ctx context.Context) error {
	e.mu.Lock()
	e.idle = nil
	e.mu.Unlock()
	return e.rt.Close(ctx)
}

func cacheDir(opt string) string {
	switch opt {
	case "-":
		return ""
	case "":
		base, err := os.UserCacheDir()
		if err != nil {
			return ""
		}
		return filepath.Join(base, "liaf", "wazero")
	default:
		return opt
	}
}

var (
	defaultOnce sync.Once
	defaultEng  *Engine
	defaultErr  error
)

// Default devolve um Engine compartilhado pelo processo, criado na primeira chamada. É o que
// os builtins de documento usam.
func Default() (*Engine, error) {
	defaultOnce.Do(func() {
		defaultEng, defaultErr = New(context.Background(), Options{})
	})
	return defaultEng, defaultErr
}
