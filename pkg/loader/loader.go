package loader

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"liaf/pkg/ast"
	"liaf/pkg/diagnostic"
	"liaf/pkg/lexer"
	"liaf/pkg/parser"
	"liaf/std"
)

// stdPrefix marca um import da biblioteca padrao embutida. "std/..." nunca e
// lido do disco: uma pasta local chamada std se importa com "./std/...".
const stdPrefix = "std/"

type Loader struct {
	// Std e a biblioteca padrao. O padrao e a embutida no compilador; os
	// testes trocam por um fs.FS em memoria.
	Std fs.FS

	visited map[string]*ast.Module
	stack   []string
	diags   []diagnostic.Diagnostic

	// origin guarda de qual arquivo veio cada declaracao. Depois da mescla
	// e o unico jeito de dizer onde esta a outra metade de uma colisao.
	origin map[ast.TopLevel]string
}

func New() *Loader {
	return &Loader{
		Std:     std.FS,
		visited: make(map[string]*ast.Module),
		origin:  make(map[ast.TopLevel]string),
	}
}

// Load resolve e carrega um arquivo ou diretório LIAF, processando recursivamente
// todos os (import ...), detectando ciclos e mesclando as declarações em um módulo unificado.
func Load(path string) (*ast.Module, []diagnostic.Diagnostic) {
	l := New()
	return l.Load(path)
}

func (l *Loader) Load(path string) (*ast.Module, []diagnostic.Diagnostic) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		l.addError(path, 1, 1, fmt.Sprintf("caminho inválido: %v", err), "E_INVALID_PATH")
		return nil, l.diags
	}

	info, err := os.Stat(absPath)
	if err != nil {
		l.addError(path, 1, 1, fmt.Sprintf("arquivo ou diretório não encontrado: %s", path), "E_FILE_NOT_FOUND")
		return nil, l.diags
	}

	var mod *ast.Module
	if info.IsDir() {
		mod, _ = l.loadDirectory(absPath)
	} else {
		mod, _ = l.loadFile(absPath)
	}
	if mod == nil || len(l.diags) > 0 {
		return nil, l.diags
	}

	l.checkCollisions(mod.Decls)
	if len(l.diags) > 0 {
		return nil, l.diags
	}
	return mod, nil
}

func (l *Loader) loadFile(absPath string) (*ast.Module, []diagnostic.Diagnostic) {
	rootMod, err := l.parseFile(absPath)
	if err != nil {
		return nil, l.diags
	}

	l.visited[absPath] = rootMod
	l.stack = append(l.stack, absPath)
	mergedDecls := l.expand(rootMod, absPath, filepath.Dir(absPath))
	l.stack = l.stack[:len(l.stack)-1]

	if len(l.diags) > 0 {
		return nil, l.diags
	}

	rootMod.Decls = mergedDecls
	return rootMod, nil
}

// expand devolve as declaracoes do modulo com cada (import ...) trocado pelo
// conteudo importado, e anota a origem de cada declaracao. dir e a base dos
// imports relativos; vazio num modulo da std, que so importa outros da std.
func (l *Loader) expand(mod *ast.Module, file, dir string) []ast.TopLevel {
	var decls []ast.TopLevel
	for _, decl := range mod.Decls {
		imp, ok := decl.(*ast.ImportDecl)
		if !ok {
			l.origin[decl] = file
			decls = append(decls, decl)
			continue
		}
		decls = append(decls, l.resolveImport(imp, file, dir)...)
	}
	return decls
}

func (l *Loader) resolveImport(imp *ast.ImportDecl, fromFile, fromDir string) []ast.TopLevel {
	if strings.HasPrefix(imp.Path, stdPrefix) {
		return l.resolveStd(imp, fromFile)
	}
	if fromDir == "" {
		l.addError(fromFile, imp.Line, imp.Col, fmt.Sprintf("um módulo da biblioteca padrão só importa outros da std, não %q", imp.Path), "E_STD_RELATIVE_IMPORT")
		return nil
	}

	rawPath := imp.Path
	if !filepath.IsAbs(rawPath) {
		rawPath = filepath.Join(fromDir, rawPath)
	}
	targetPath := filepath.Clean(rawPath)

	// Se não tem extensão, tenta .liaf
	if filepath.Ext(targetPath) == "" {
		if _, err := os.Stat(targetPath + ".liaf"); err == nil {
			targetPath = targetPath + ".liaf"
		}
	}

	// Verifica se arquivo existe
	info, err := os.Stat(targetPath)
	if err != nil {
		l.addError(fromFile, imp.Line, imp.Col, fmt.Sprintf("arquivo importado não encontrado: %s", imp.Path), "E_IMPORT_NOT_FOUND")
		return nil
	}

	if info.IsDir() {
		return l.resolveImportDir(targetPath, imp, fromFile)
	}

	if !l.enter(targetPath, imp, fromFile) {
		return nil
	}
	defer l.leave()

	mod, err := l.parseFile(targetPath)
	if err != nil {
		return nil
	}
	l.visited[targetPath] = mod
	return l.expand(mod, targetPath, filepath.Dir(targetPath))
}

// resolveStd carrega std/<nome> da biblioteca embutida.
func (l *Loader) resolveStd(imp *ast.ImportDecl, fromFile string) []ast.TopLevel {
	name := strings.TrimSuffix(strings.TrimPrefix(imp.Path, stdPrefix), ".liaf")
	key := stdPrefix + name + ".liaf"
	// path.Clean pega "std/../segredo" e afins: a std e plana.
	if name == "" || strings.ContainsAny(name, `/\`) || path.Clean(name) != name {
		l.addError(fromFile, imp.Line, imp.Col, fmt.Sprintf("módulo da biblioteca padrão inválido: %q", imp.Path), "E_IMPORT_NOT_FOUND")
		return nil
	}

	content, err := fs.ReadFile(l.Std, name+".liaf")
	if err != nil {
		l.addError(fromFile, imp.Line, imp.Col, fmt.Sprintf("módulo %q não existe na biblioteca padrão; disponíveis: %s", imp.Path, strings.Join(l.stdModules(), ", ")), "E_IMPORT_NOT_FOUND")
		return nil
	}

	if !l.enter(key, imp, fromFile) {
		return nil
	}
	defer l.leave()

	mod, err := l.parseSource(key, string(content))
	if err != nil {
		return nil
	}
	l.visited[key] = mod
	return l.expand(mod, key, "")
}

func (l *Loader) stdModules() []string {
	names, _ := fs.Glob(l.Std, "*.liaf")
	for i, n := range names {
		names[i] = stdPrefix + strings.TrimSuffix(n, ".liaf")
	}
	sort.Strings(names)
	return names
}

// enter empilha o arquivo antes de expandi-lo. Devolve false se ele ja esta
// na pilha (ciclo, que e erro) ou ja foi carregado por outro ramo (diamante,
// que nao e: as declaracoes ja estao na mescla).
func (l *Loader) enter(key string, imp *ast.ImportDecl, fromFile string) bool {
	for _, s := range l.stack {
		if s == key {
			cycle := append(append([]string{}, l.stack...), key)
			l.addError(fromFile, imp.Line, imp.Col, fmt.Sprintf("import circular detectado: %s", strings.Join(cycle, " -> ")), "E_CIRCULAR_IMPORT")
			return false
		}
	}
	if _, ok := l.visited[key]; ok {
		return false
	}
	l.stack = append(l.stack, key)
	return true
}

func (l *Loader) leave() { l.stack = l.stack[:len(l.stack)-1] }

func (l *Loader) resolveImportDir(dirPath string, imp *ast.ImportDecl, fromFile string) []ast.TopLevel {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		l.addError(fromFile, imp.Line, imp.Col, fmt.Sprintf("erro ao ler diretório importado: %v", err), "E_DIR_READ")
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".liaf") {
			files = append(files, filepath.Join(dirPath, e.Name()))
		}
	}
	sort.Strings(files)

	var decls []ast.TopLevel
	for _, f := range files {
		if _, ok := l.visited[f]; ok {
			continue
		}
		subMod, err := l.parseFile(f)
		if err != nil {
			continue
		}
		l.visited[f] = subMod
		decls = append(decls, l.expand(subMod, f, dirPath)...)
	}
	return decls
}

func (l *Loader) loadDirectory(dirPath string) (*ast.Module, []diagnostic.Diagnostic) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		l.addError(dirPath, 1, 1, fmt.Sprintf("erro ao ler diretório: %v", err), "E_DIR_READ")
		return nil, l.diags
	}

	var liafFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".liaf") {
			liafFiles = append(liafFiles, filepath.Join(dirPath, e.Name()))
		}
	}

	if len(liafFiles) == 0 {
		l.addError(dirPath, 1, 1, fmt.Sprintf("nenhum arquivo .liaf encontrado em %s", dirPath), "E_NO_LIAF_FILES")
		return nil, l.diags
	}

	sort.Strings(liafFiles)

	var rootMod *ast.Module
	var allDecls []ast.TopLevel

	for _, file := range liafFiles {
		if _, ok := l.visited[file]; ok {
			continue
		}
		mod, diags := l.loadFile(file)
		if len(diags) > 0 {
			return nil, l.diags
		}
		if rootMod == nil {
			rootMod = mod
		}
		allDecls = append(allDecls, mod.Decls...)
	}

	if rootMod == nil {
		rootMod = &ast.Module{Name: filepath.Base(dirPath)}
	}
	rootMod.Decls = allDecls
	return rootMod, l.diags
}

// checkCollisions procura, na mescla final, nomes declarados em dois
// arquivos e rotas registradas duas vezes. Funcoes e structs dividem um
// espaco de nomes, como no checker; uma ws-route ocupa o GET do seu caminho.
//
// Nomes repetidos no mesmo arquivo ficam para o checker (E_DUPLICATE_SYMBOL).
// Rotas repetidas sao pegas aqui mesmo no mesmo arquivo, porque ninguem mais
// as pega: o ServeMux do Go daria panic na subida do binario.
func (l *Loader) checkCollisions(decls []ast.TopLevel) {
	symbols := map[string]ast.TopLevel{}
	routes := map[string]ast.TopLevel{}
	for _, d := range decls {
		switch v := d.(type) {
		case *ast.FuncDecl:
			l.collide(symbols, v.Name, "fn "+v.Name, d, false)
		case *ast.StructDecl:
			l.collide(symbols, v.Name, "struct "+v.Name, d, false)
		case *ast.RouteDecl:
			pattern := strings.ToUpper(v.Method) + " " + v.Path
			l.collide(routes, pattern, "route "+pattern, d, true)
		case *ast.WSRouteDecl:
			l.collide(routes, "GET "+v.Path, "ws-route "+v.Path, d, true)
		}
	}
}

func (l *Loader) collide(seen map[string]ast.TopLevel, key, what string, d ast.TopLevel, sameFileToo bool) {
	first, ok := seen[key]
	if !ok {
		seen[key] = d
		return
	}
	firstFile, file := l.origin[first], l.origin[d]
	if firstFile == file && !sameFileToo {
		return
	}
	fl, fc := first.Pos()
	line, col := d.Pos()
	code, hint := "E_DUPLICATE_DECL", "renomeie uma das duas"
	if _, isFn := d.(*ast.FuncDecl); !isFn {
		if _, isStruct := d.(*ast.StructDecl); !isStruct {
			code, hint = "E_DUPLICATE_ROUTE", "o servidor não sobe com o mesmo caminho registrado duas vezes"
		}
	}
	if strings.HasPrefix(firstFile, stdPrefix) {
		hint = "o nome já pertence à biblioteca padrão; renomeie a sua"
	}
	l.addError(file, line, col, fmt.Sprintf("%s já declarado em %s:%d:%d; %s", what, firstFile, fl, fc, hint), code)
}

func (l *Loader) parseFile(absPath string) (*ast.Module, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		l.addError(absPath, 1, 1, fmt.Sprintf("erro ao ler arquivo: %v", err), "E_FILE_READ")
		return nil, err
	}
	return l.parseSource(absPath, string(content))
}

func (l *Loader) parseSource(name, content string) (*ast.Module, error) {
	lex := lexer.New(content)
	p := parser.New(lex, name)
	mod := p.ParseModule()

	for _, d := range p.Diagnostics {
		l.diags = append(l.diags, diagnostic.Diagnostic{
			Code:    d.Code,
			File:    d.File,
			Line:    d.Line,
			Col:     d.Col,
			Message: d.Message,
		})
	}

	if len(p.Diagnostics) > 0 || mod == nil {
		return nil, fmt.Errorf("erros de sintaxe em %s", name)
	}

	return mod, nil
}

func (l *Loader) addError(file string, line, col int, message, code string) {
	l.diags = append(l.diags, diagnostic.Diagnostic{
		Code:    code,
		File:    file,
		Line:    line,
		Col:     col,
		Message: message,
	})
}
