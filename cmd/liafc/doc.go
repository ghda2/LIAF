package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"liaf/pkg/typst"
	"liaf/pkg/typst/packages"
)

// liafc doc (issue #037): confere e visualiza um documento Typst sem escrever o programa LIAF
// que o renderiza. É o laço de correção do agente: check devolve erros e avisos de layout com
// a correção pronta; preview devolve a página em PNG para quem enxerga imagem.
func runDoc(args []string) {
	if len(args) < 2 || (args[0] != "check" && args[0] != "preview" && args[0] != "watch") {
		fmt.Fprintln(os.Stderr, "Uso:")
		fmt.Fprintln(os.Stderr, "  liafc doc check   <doc.typ> [--dados d.json] [--arquivo nome=caminho]... [--json] [--layout]")
		fmt.Fprintln(os.Stderr, "  liafc doc preview <doc.typ> [--dados d.json] [--arquivo nome=caminho]... [-o saida.png] [--pagina N] [--ppi N]")
		fmt.Fprintln(os.Stderr, "  liafc doc watch   <doc.typ> [mesmas opções; com -o também grava a prévia a cada mudança]")
		os.Exit(1)
	}
	opts, err := parseDocArgs(args[1:])
	if err != nil {
		fatal(err)
	}
	if args[0] == "watch" {
		docWatch(opts)
		return
	}
	job, err := opts.job()
	if err != nil {
		fatal(err)
	}
	eng, err := typst.New(context.Background(), typst.Options{})
	if err != nil {
		fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if args[0] == "check" {
		os.Exit(docCheck(ctx, eng, job, opts))
	}
	os.Exit(docPreview(ctx, eng, job, opts))
}

type docOptions struct {
	file     string
	data     string
	extra    map[string]string // nome virtual -> caminho no disco
	json     bool
	layout   bool
	out      string
	page     int
	ppi      float64
	resolver *packages.Resolver
}

func parseDocArgs(args []string) (*docOptions, error) {
	o := &docOptions{extra: map[string]string{}, page: 1, ppi: 144}
	next := func(i *int, flag string) (string, error) {
		if *i+1 >= len(args) {
			return "", fmt.Errorf("%s precisa de um valor", flag)
		}
		*i++
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		var err error
		switch {
		case a == "--json":
			o.json = true
		case a == "--layout":
			o.layout = true
		case a == "--dados":
			o.data, err = next(&i, a)
		case a == "-o":
			o.out, err = next(&i, a)
		case a == "--pagina" || a == "--ppi":
			var v string
			if v, err = next(&i, a); err == nil {
				n, convErr := strconv.ParseFloat(v, 64)
				if convErr != nil || n <= 0 {
					return nil, fmt.Errorf("%s: número inválido %q", a, v)
				}
				if a == "--pagina" {
					o.page = int(n)
				} else {
					o.ppi = n
				}
			}
		case a == "--arquivo":
			var v string
			if v, err = next(&i, a); err == nil {
				name, path, ok := strings.Cut(v, "=")
				if !ok || name == "" || path == "" {
					return nil, fmt.Errorf("--arquivo espera nome=caminho, veio %q", v)
				}
				o.extra[name] = path
			}
		case strings.HasPrefix(a, "-"):
			return nil, fmt.Errorf("opção desconhecida: %s", a)
		case o.file == "":
			o.file = a
		default:
			return nil, fmt.Errorf("argumento sobrando: %s", a)
		}
		if err != nil {
			return nil, err
		}
	}
	if o.file == "" {
		return nil, fmt.Errorf("informe o documento .typ")
	}
	return o, nil
}

// job monta os arquivos virtuais como o runtime faz (/main.typ, /dados.json, extras) e resolve
// os pacotes citados (com cache e trava ao lado do documento, como no liafc build).
func (o *docOptions) job() (typst.Job, error) {
	src, err := os.ReadFile(o.file)
	if err != nil {
		return typst.Job{}, err
	}
	data := []byte("{}")
	if o.data != "" {
		if data, err = os.ReadFile(o.data); err != nil {
			return typst.Job{}, err
		}
	}
	files := map[string][]byte{"/main.typ": src, "/dados.json": data}
	for name, path := range o.extra {
		b, err := os.ReadFile(path)
		if err != nil {
			return typst.Job{}, err
		}
		files["/"+strings.TrimPrefix(filepath.ToSlash(name), "/")] = b
	}
	if specs := packages.Scan(string(src)); len(specs) > 0 {
		cache, err := typstPackageCache()
		if err != nil {
			return typst.Job{}, err
		}
		lockFile := filepath.Join(filepath.Dir(o.file), typstLockName)
		lock, err := packages.ReadLock(lockFile)
		if err != nil {
			return typst.Job{}, err
		}
		before := len(lock)
		r := &packages.Resolver{Cache: cache, Lock: lock}
		all, err := r.Resolve(specs)
		if err != nil {
			return typst.Job{}, err
		}
		if len(lock) != before {
			if err := packages.WriteLock(lockFile, lock); err != nil {
				return typst.Job{}, err
			}
		}
		pkgFiles, err := r.Files(all)
		if err != nil {
			return typst.Job{}, err
		}
		for k, v := range pkgFiles {
			files[k] = v
		}
	}
	return typst.Job{Files: files}, nil
}

// Relatório do doc check, no mesmo espírito do liafc check --json.
type docReport struct {
	Status   string             `json:"status"`
	Errors   []typst.Diagnostic `json:"errors"`
	Warnings []typst.Diagnostic `json:"warnings"`
	Metrics  typst.Metrics      `json:"metrics"`
	Layout   []typst.PageLayout `json:"layout,omitempty"`
}

func docCheck(ctx context.Context, eng *typst.Engine, job typst.Job, o *docOptions) int {
	job.Format = typst.FormatCheck
	job.Layout = true
	res, err := eng.Render(ctx, job)
	if err != nil {
		fatal(err)
	}
	rep := docReport{Status: "success", Errors: []typst.Diagnostic{}, Warnings: []typst.Diagnostic{}}
	for _, d := range res.Diagnostics {
		if d.Severity == "error" {
			d.Code = "E_TYPST"
			rep.Errors = append(rep.Errors, d)
		} else {
			d.Code = "W_TYPST"
			rep.Warnings = append(rep.Warnings, d)
		}
	}
	if res.OK {
		layoutWarns, m := typst.Analyze(res)
		rep.Warnings = append(rep.Warnings, layoutWarns...)
		m.Warnings = len(rep.Warnings)
		rep.Metrics = m
		if o.layout {
			rep.Layout = res.Layout
		}
	}
	if len(rep.Errors) > 0 {
		rep.Status = "error"
	}

	if o.json {
		b, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(b))
	} else {
		for _, d := range append(rep.Errors, rep.Warnings...) {
			line := d.String()
			if d.Code != "" {
				line = "[" + d.Code + "] " + line
			}
			if d.Page > 0 {
				line += fmt.Sprintf(" (página %d)", d.Page)
			}
			if d.Fix != "" {
				line += "\n  correção: " + d.Fix
			}
			fmt.Println(line)
		}
		if rep.Status == "success" {
			fmt.Printf("✓ %d página(s), %d aviso(s)\n", rep.Metrics.Pages, len(rep.Warnings))
		}
	}
	if rep.Status == "error" {
		return 1
	}
	return 0
}

func docPreview(ctx context.Context, eng *typst.Engine, job typst.Job, o *docOptions) int {
	job.Format = typst.FormatPNG
	job.PPI = o.ppi
	res, err := eng.Render(ctx, job)
	if err != nil {
		fatal(err)
	}
	if err := res.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if o.page > len(res.Outputs) {
		fmt.Fprintf(os.Stderr, "--pagina %d: o documento tem %d página(s)\n", o.page, len(res.Outputs))
		return 1
	}
	out := o.out
	if out == "" {
		out = strings.TrimSuffix(o.file, filepath.Ext(o.file)) + fmt.Sprintf("-p%d.png", o.page)
	}
	if err := os.WriteFile(out, res.Outputs[o.page-1], 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("✓ %s (página %d de %d)\n", out, o.page, len(res.Outputs))
	return 0
}

// docWatch reconfere o documento sempre que ele, os dados ou um arquivo extra mudam. Com -o,
// regrava a prévia também. Termina com Ctrl+C.
func docWatch(o *docOptions) {
	eng, err := typst.New(context.Background(), typst.Options{})
	if err != nil {
		fatal(err)
	}
	files := o.watched()
	var last map[string]time.Time
	fmt.Printf("observando %s (Ctrl+C para sair)\n", strings.Join(files, ", "))
	for {
		now := mtimes(files)
		if changed(last, now) {
			last = now
			fmt.Printf("\n── %s ──\n", time.Now().Format("15:04:05"))
			if job, err := o.job(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				if docCheck(ctx, eng, job, o) == 0 && o.out != "" {
					docPreview(ctx, eng, job, o)
				}
				cancel()
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func (o *docOptions) watched() []string {
	files := []string{o.file}
	if o.data != "" {
		files = append(files, o.data)
	}
	for _, p := range o.extra {
		files = append(files, p)
	}
	return files
}

func mtimes(files []string) map[string]time.Time {
	out := make(map[string]time.Time, len(files))
	for _, f := range files {
		if st, err := os.Stat(f); err == nil {
			out[f] = st.ModTime()
		}
	}
	return out
}

func changed(before, after map[string]time.Time) bool {
	if before == nil || len(before) != len(after) {
		return true
	}
	for f, t := range after {
		if !before[f].Equal(t) {
			return true
		}
	}
	return false
}
