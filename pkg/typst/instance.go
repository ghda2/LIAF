package typst

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// instance é uma cópia viva do motor. Não é segura para uso concorrente: o Engine entrega
// cada instância a um chamador por vez.
type instance struct {
	mod  api.Module
	mem  api.Memory
	fns  map[string]api.Function
	uses int
}

var abi = []string{
	"liaf_alloc", "liaf_free", "liaf_add_font", "liaf_set_file", "liaf_clear_files",
	"liaf_render", "liaf_result_ptr", "liaf_result_len", "liaf_output_count",
	"liaf_output_ptr", "liaf_output_len", "liaf_evict",
}

func newInstance(ctx context.Context, rt wazero.Runtime, compiled wazero.CompiledModule, fontFiles [][]byte) (*instance, error) {
	cfg := wazero.NewModuleConfig().WithName("")
	if _, ok := compiled.ExportedFunctions()["_initialize"]; ok {
		cfg = cfg.WithStartFunctions("_initialize")
	} else {
		cfg = cfg.WithStartFunctions()
	}
	mod, err := rt.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("typst: instanciar motor: %w", err)
	}
	in := &instance{mod: mod, mem: mod.Memory(), fns: map[string]api.Function{}}
	for _, name := range abi {
		fn := mod.ExportedFunction(name)
		if fn == nil {
			mod.Close(ctx)
			return nil, fmt.Errorf("typst: motor sem a função %s (wasm desatualizado?)", name)
		}
		in.fns[name] = fn
	}
	for _, f := range fontFiles {
		if _, err := in.withBuffer(ctx, f, func(ptr, n uint64) ([]uint64, error) {
			return in.fns["liaf_add_font"].Call(ctx, ptr, n)
		}); err != nil {
			mod.Close(ctx)
			return nil, fmt.Errorf("typst: carregar fonte: %w", err)
		}
	}
	return in, nil
}

// withBuffer copia data para a memória do motor, chama fn(ptr, len) e libera o buffer.
func (in *instance) withBuffer(ctx context.Context, data []byte, fn func(ptr, n uint64) ([]uint64, error)) ([]uint64, error) {
	n := uint64(len(data))
	res, err := in.fns["liaf_alloc"].Call(ctx, n)
	if err != nil {
		return nil, err
	}
	ptr := res[0]
	if n > 0 && !in.mem.Write(uint32(ptr), data) {
		return nil, fmt.Errorf("escrita fora da memória do motor")
	}
	out, callErr := fn(ptr, n)
	if _, err := in.fns["liaf_free"].Call(ctx, ptr, n); err != nil && callErr == nil {
		callErr = err
	}
	return out, callErr
}

func (in *instance) read(ptr, n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	b, ok := in.mem.Read(uint32(ptr), uint32(n))
	if !ok {
		return nil, fmt.Errorf("leitura fora da memória do motor")
	}
	return append([]byte(nil), b...), nil // copia: a memória muda na próxima chamada
}

func (in *instance) call(ctx context.Context, name string, args ...uint64) (uint64, error) {
	res, err := in.fns[name].Call(ctx, args...)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	return res[0], nil
}

func (in *instance) setFile(ctx context.Context, path string, data []byte) error {
	_, err := in.withBuffer(ctx, []byte(path), func(pp, pn uint64) ([]uint64, error) {
		return in.withBuffer(ctx, data, func(dp, dn uint64) ([]uint64, error) {
			ok, err := in.call(ctx, "liaf_set_file", pp, pn, dp, dn)
			if err == nil && ok == 0 {
				err = fmt.Errorf("caminho inválido: %q (use caminhos como \"/dados.json\")", path)
			}
			return nil, err
		})
	})
	return err
}

func (in *instance) render(ctx context.Context, job Job) (Result, error) {
	in.uses++
	if _, err := in.call(ctx, "liaf_clear_files"); err != nil {
		return Result{}, err
	}
	paths := make([]string, 0, len(job.Files))
	for p := range job.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if err := in.setFile(ctx, p, job.Files[p]); err != nil {
			return Result{}, fmt.Errorf("typst: enviar %s: %w", p, err)
		}
	}

	req, _ := json.Marshal(job.request())
	if _, err := in.withBuffer(ctx, req, func(ptr, n uint64) ([]uint64, error) {
		return in.fns["liaf_render"].Call(ctx, ptr, n)
	}); err != nil {
		return Result{}, fmt.Errorf("typst: renderizar: %w", err)
	}

	ptr, err := in.call(ctx, "liaf_result_ptr")
	if err != nil {
		return Result{}, err
	}
	n, err := in.call(ctx, "liaf_result_len")
	if err != nil {
		return Result{}, err
	}
	raw, err := in.read(ptr, n)
	if err != nil {
		return Result{}, err
	}
	var res Result
	if err := json.Unmarshal(raw, &res); err != nil {
		return Result{}, fmt.Errorf("typst: resposta inválida do motor: %w", err)
	}

	count, err := in.call(ctx, "liaf_output_count")
	if err != nil {
		return Result{}, err
	}
	for i := uint64(0); i < count; i++ {
		p, err := in.call(ctx, "liaf_output_ptr", i)
		if err != nil {
			return Result{}, err
		}
		l, err := in.call(ctx, "liaf_output_len", i)
		if err != nil {
			return Result{}, err
		}
		b, err := in.read(p, l)
		if err != nil {
			return Result{}, err
		}
		res.Outputs = append(res.Outputs, b)
	}
	return res, nil
}

func (in *instance) evict(ctx context.Context, maxAge uint64) {
	_, _ = in.call(ctx, "liaf_evict", maxAge)
}

func (in *instance) close(ctx context.Context) { _ = in.mod.Close(ctx) }
