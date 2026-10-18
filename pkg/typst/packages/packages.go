// Package packages resolve os pacotes do Typst Universe (@preview/nome:versão) que um programa
// usa, no momento do build. O motor não acessa a rede: o liafc baixa os pacotes citados, confere
// o SHA-256 contra o arquivo de trava (como o go.sum) e os embute no binário.
package packages

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Spec identifica um pacote: @namespace/nome:versão.
type Spec struct {
	Namespace, Name, Version string
}

func (s Spec) String() string { return "@" + s.Namespace + "/" + s.Name + ":" + s.Version }

// Dir é o caminho relativo do pacote dentro de um diretório de pacotes: namespace/nome/versão.
func (s Spec) Dir() string { return path.Join(s.Namespace, s.Name, s.Version) }

var specRe = regexp.MustCompile(`@([a-z][a-z0-9-]*)/([a-zA-Z0-9][a-zA-Z0-9_-]*):(\d+\.\d+\.\d+)`)

// Scan encontra as citações de pacotes em textos (código gerado, fontes .typ), sem repetir.
func Scan(texts ...string) []Spec {
	seen := map[Spec]bool{}
	var out []Spec
	for _, t := range texts {
		for _, m := range specRe.FindAllStringSubmatch(t, -1) {
			s := Spec{m[1], m[2], m[3]}
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

// Registry é de onde os pacotes vêm. Só o namespace "preview" é público.
var Registry = "https://packages.typst.org"

// MaxArchive limita o tamanho de um pacote baixado (compactado e extraído).
const MaxArchive = 64 << 20

// Resolver baixa e guarda pacotes em Cache, conferindo o SHA-256 em Lock.
type Resolver struct {
	Cache string          // pasta de cache: <Cache>/<namespace>/<nome>/<versão>/
	Lock  map[Spec]string // SHA-256 do .tar.gz de cada pacote; novos são acrescentados
	HTTP  *http.Client
}

// Resolve devolve os pacotes pedidos e todas as dependências entre eles, já no cache.
func (r *Resolver) Resolve(specs []Spec) ([]Spec, error) {
	pending := append([]Spec(nil), specs...)
	done := map[Spec]bool{}
	var out []Spec
	for len(pending) > 0 {
		s := pending[0]
		pending = pending[1:]
		if done[s] {
			continue
		}
		done[s] = true
		dir, err := r.fetch(s)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
		deps, err := scanDir(dir)
		if err != nil {
			return nil, err
		}
		pending = append(pending, deps...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out, nil
}

// Path é a pasta do pacote no cache.
func (r *Resolver) Path(s Spec) string { return filepath.Join(r.Cache, filepath.FromSlash(s.Dir())) }

func (r *Resolver) fetch(s Spec) (string, error) {
	dir := r.Path(s)
	if _, err := os.Stat(filepath.Join(dir, "typst.toml")); err == nil {
		return dir, nil
	}
	if s.Namespace != "preview" {
		return "", fmt.Errorf("pacote %s: só o namespace @preview é baixado (pacotes locais: copie para %s)", s, dir)
	}
	url := fmt.Sprintf("%s/%s/%s-%s.tar.gz", Registry, s.Namespace, s.Name, s.Version)
	client := r.HTTP
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("pacote %s: baixar %s: %w", s, url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pacote %s: %s respondeu %s", s, url, resp.Status)
	}
	archive, err := io.ReadAll(io.LimitReader(resp.Body, MaxArchive+1))
	if err != nil {
		return "", fmt.Errorf("pacote %s: %w", s, err)
	}
	if len(archive) > MaxArchive {
		return "", fmt.Errorf("pacote %s: maior que %d MB", s, MaxArchive>>20)
	}
	sum := sha256.Sum256(archive)
	got := hex.EncodeToString(sum[:])
	if want, ok := r.Lock[s]; ok && want != got {
		return "", fmt.Errorf("pacote %s: SHA-256 %s difere do arquivo de trava (%s); o pacote mudou no servidor", s, got, want)
	}
	if r.Lock != nil {
		r.Lock[s] = got
	}

	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dir), "tmp-"+s.Version+"-*")
	if err != nil {
		return "", err
	}
	if err := extract(archive, tmp); err != nil {
		os.RemoveAll(tmp)
		return "", fmt.Errorf("pacote %s: %w", s, err)
	}
	if err := os.Rename(tmp, dir); err != nil {
		os.RemoveAll(tmp)
		if _, statErr := os.Stat(filepath.Join(dir, "typst.toml")); statErr == nil {
			return dir, nil // outro liafc terminou primeiro
		}
		return "", err
	}
	return dir, nil
}

func extract(archive []byte, dst string) error {
	gz, err := gzip.NewReader(strings.NewReader(string(archive)))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	total := int64(0)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(strings.TrimPrefix(h.Name, "./"))
		if name == "." || strings.HasPrefix(name, "../") || path.IsAbs(name) || strings.Contains(name, `\`) {
			if name == "." {
				continue
			}
			return fmt.Errorf("caminho inválido no pacote: %q", h.Name)
		}
		target := filepath.Join(dst, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			total += h.Size
			if total > MaxArchive {
				return fmt.Errorf("conteúdo extraído passa de %d MB", MaxArchive>>20)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, io.LimitReader(tr, h.Size))
			f.Close()
			if err != nil {
				return err
			}
		default:
			// links e dispositivos não entram: o motor só lê arquivos regulares.
		}
	}
}

func scanDir(dir string) ([]Spec, error) {
	var texts []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(p, ".typ") || strings.HasSuffix(p, ".toml") {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			texts = append(texts, string(b))
		}
		return nil
	})
	return Scan(texts...), err
}

// ReadLock lê um arquivo de trava ("@preview/nome:1.0.0 <sha256>" por linha). Arquivo
// inexistente é uma trava vazia.
func ReadLock(file string) (map[Spec]string, error) {
	lock := map[Spec]string{}
	b, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return lock, nil
	}
	if err != nil {
		return nil, err
	}
	for n, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		specs := Scan(fields[0])
		if len(fields) != 2 || len(specs) != 1 {
			return nil, fmt.Errorf("%s:%d: linha inválida: %q", file, n+1, line)
		}
		lock[specs[0]] = fields[1]
	}
	return lock, nil
}

// WriteLock grava a trava em ordem estável.
func WriteLock(file string, lock map[Spec]string) error {
	specs := make([]Spec, 0, len(lock))
	for s := range lock {
		specs = append(specs, s)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].String() < specs[j].String() })
	var b strings.Builder
	b.WriteString("# Pacotes Typst embutidos pelo liafc build: SHA-256 do .tar.gz. Versione este arquivo.\n")
	for _, s := range specs {
		b.WriteString(s.String() + " " + lock[s] + "\n")
	}
	return os.WriteFile(file, []byte(b.String()), 0o644)
}

// Files lê os pacotes do cache no formato que o motor aceita: "@ns/nome:versão/caminho".
func (r *Resolver) Files(specs []Spec) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, s := range specs {
		dir := r.Path(s)
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			out[s.String()+"/"+filepath.ToSlash(rel)] = data
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("pacote %s: %w", s, err)
		}
	}
	return out, nil
}
