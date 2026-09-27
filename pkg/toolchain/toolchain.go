// Package toolchain compila o Go gerado pelo liafc sem depender da arvore do
// repositorio (issue #032).
//
// O codigo gerado importa liaf/pkg/web e liaf/pkg/runtime. Em vez de exigir um
// go.mod com "replace liaf => ../liaf" na pasta do usuario, o liafc carrega os
// fontes desses pacotes embutidos (liaf.RuntimeFS) e os grava uma vez num
// cache, numa pasta cujo nome e o hash do conteudo. Cada programa vira um
// pacote main temporario dentro desse modulo, onde os imports liaf/... resolvem
// sem configuracao.
//
// O que continua necessario na maquina: o comando go e, na primeira
// compilacao, acesso ao proxy de modulos para baixar as dependencias de
// terceiros (x/crypto, sqlite). Depois disso o cache de modulos do Go basta.
package toolchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"liaf"
)

// CacheEnv sobrescreve a pasta onde o runtime e materializado.
const CacheEnv = "LIAF_CACHE"

// Module garante que o modulo do runtime existe no cache e devolve a pasta.
// A escrita e feita numa pasta temporaria renomeada no fim, entao dois liafc
// rodando ao mesmo tempo nunca veem um modulo pela metade.
func Module() (string, error) {
	files, sum, err := runtimeFiles()
	if err != nil {
		return "", err
	}
	root, err := cacheRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "runtime-"+sum[:16])
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("toolchain: criar cache %s: %w", root, err)
	}
	tmp, err := os.MkdirTemp(root, "tmp-runtime-*")
	if err != nil {
		return "", fmt.Errorf("toolchain: %w", err)
	}
	for _, name := range files {
		data, err := fs.ReadFile(liaf.RuntimeFS, name)
		if err != nil {
			os.RemoveAll(tmp)
			return "", err
		}
		target := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			os.RemoveAll(tmp)
			return "", err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			os.RemoveAll(tmp)
			return "", err
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		os.RemoveAll(tmp)
		// Outro liafc pode ter terminado primeiro: o conteudo e o mesmo.
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		return "", fmt.Errorf("toolchain: publicar runtime em %s: %w", dir, err)
	}
	return dir, nil
}

// App e um programa gerado pronto para o go build.
type App struct {
	Module string // raiz do modulo do runtime
	Dir    string // pasta do pacote main, dentro de Module
}

// NewApp grava main.go num pacote novo dentro do modulo do runtime. Quem chama
// pode acrescentar arquivos em Dir (a pasta public do --embed) e deve chamar
// Remove no fim.
func NewApp(mainGo string) (*App, error) {
	mod, err := Module()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(mod, "app-*")
	if err != nil {
		return nil, fmt.Errorf("toolchain: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return &App{Module: mod, Dir: dir}, nil
}

// Remove apaga o pacote temporario; o modulo do runtime fica para a proxima.
func (a *App) Remove() { os.RemoveAll(a.Dir) }

// Build compila o programa em out (caminho absoluto).
func (a *App) Build(out string) *exec.Cmd {
	cmd := exec.Command("go", "build", "-o", out, "./"+filepath.Base(a.Dir))
	cmd.Dir = a.Module
	// Um go.work do usuario acima do cache mudaria como liaf/... resolve.
	cmd.Env = append(os.Environ(), "GOWORK=off")
	// O runtime e Go puro (sqlite incluso), entao o binario sai estatico e a
	// cross-compilacao funciona mesmo com CGO_ENABLED=1 no go env da maquina.
	// Quem precisar de cgo define CGO_ENABLED no ambiente.
	if _, set := os.LookupEnv("CGO_ENABLED"); !set {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	}
	return cmd
}

// runtimeFiles lista os arquivos embutidos, sem testes, e o hash do conjunto.
func runtimeFiles() ([]string, string, error) {
	var files []string
	err := fs.WalkDir(liaf.RuntimeFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && !strings.HasSuffix(p, "_test.go") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, name := range files {
		data, err := fs.ReadFile(liaf.RuntimeFS, name)
		if err != nil {
			return nil, "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(data))
		h.Write(data)
	}
	return files, hex.EncodeToString(h.Sum(nil)), nil
}

func cacheRoot() (string, error) {
	if dir := os.Getenv(CacheEnv); dir != "" {
		return filepath.Abs(dir)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("toolchain: sem pasta de cache (defina %s): %w", CacheEnv, err)
	}
	return filepath.Join(base, "liaf"), nil
}
