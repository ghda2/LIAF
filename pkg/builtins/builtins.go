package builtins

import (
	"fmt"
	"strings"

	"liaf/pkg/ast"
)

// Arity define a quantidade de argumentos aceita por um builtin.
type Arity struct {
	Min int
	Max int // -1 para variádico sem limite superior
}

func ExactArity(n int) Arity {
	return Arity{Min: n, Max: n}
}

func MinArity(min int) Arity {
	return Arity{Min: min, Max: -1}
}

func RangeArity(min, max int) Arity {
	return Arity{Min: min, Max: max}
}

// CheckContext é a interface oferecida pelo checker para os ganchos dos builtins.
type CheckContext interface {
	Expr(e ast.Expr, want ast.Type) ast.Type
	Require(n ast.Node, actual, want ast.Type)
	Error(n ast.Node, code, msg string)
	Arity(v *ast.CallExpr, n int) bool
	Effect(n ast.Node, eff string)
	TypeArg(e ast.Expr) ast.Type
	FindStruct(name string) *ast.StructDecl
	FindFunc(name string) *ast.FuncDecl
	TypeAt(e ast.Expr) ast.Type
}

// GoContext é a interface oferecida pelo gerador Go.
type GoContext interface {
	GenExpr(e ast.Expr) string
	MapType(t ast.Type) string
	TypeOf(e ast.Expr) ast.Type
	TypeName(e ast.Expr) string
	FieldIdent(f string) string
}

// CContext é a interface oferecida pelo gerador C.
type CContext interface {
	Expr(e ast.Expr) string
	Value(code string) string
	Quote(s string) string
	Array(items []string) string
	FindStruct(name string) *ast.StructDecl
	SetError(err error)
}

// Builtin representa um builtin registrado na tabela única da linguagem.
type Builtin struct {
	Name        string
	Category    string
	Arity       Arity
	Effects     []string // Efeitos exigidos: "io", "fs", "net", "clock", "db", etc.
	Params      []string // Tipos esperados para cada parâmetro ("str", "int", "float", "bool", etc.)
	Return      string   // Tipo de retorno ("void", "int", "str", "(result void str)", etc.)
	UnevaluatedArgs bool // Se true, os argumentos não são expressões comuns (ex: tipos ou identificadores literais)
	
	// SpecialCheck é chamado para validações com tipos dinâmicos/genéricos ou sintaxe especial.
	// Se for nil, o checker usa a checagem padrão baseada em Arity, Params, Effects e Return.
	SpecialCheck func(ctx CheckContext, call *ast.CallExpr, want ast.Type) ast.Type

	// Emissão Go
	GoCall     string // Nome da função no runtime Go (ex: "rt.ReadFile" gera "rt.ReadFile(arg1, arg2)")
	GoTemplate string // Formato Go (ex: "fmt.Println(%s)" ou "(%s == %s)")
	GoEmit     func(ctx GoContext, call *ast.CallExpr, args []string) (string, bool)

	// Emissão C
	UnsupportedC       bool
	UnsupportedCReason string
	CCall              string // Função C (ex: "vread" gera "vread(arg1, arg2)")
	CTemplate          string // Formato C (ex: "vb(!%s.b)")
	CEmit              func(ctx CContext, call *ast.CallExpr, args []string) (string, bool)
}

var registry = map[string]*Builtin{}
var orderedBuiltins []*Builtin

// Register adiciona um builtin na tabela única.
func Register(b *Builtin) {
	name := normalize(b.Name)
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("builtin %q ja registrado", b.Name))
	}
	registry[name] = b
	orderedBuiltins = append(orderedBuiltins, b)
}

// Lookup busca um builtin pelo nome (aceita tanto hífen quanto underscore).
func Lookup(name string) *Builtin {
	return registry[normalize(name)]
}

// All retorna todos os builtins registrados na ordem de inserção.
func All() []*Builtin {
	return orderedBuiltins
}

func normalize(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

// Helper para construir tipos AST a partir de string de especificação
func ParseType(spec string) ast.Type {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	if strings.HasPrefix(spec, "(") && strings.HasSuffix(spec, ")") {
		inner := strings.TrimSpace(spec[1 : len(spec)-1])
		parts := strings.Fields(inner)
		if len(parts) == 0 {
			return nil
		}
		args := make([]ast.Type, len(parts)-1)
		for i, p := range parts[1:] {
			args[i] = ParseType(p)
		}
		return &ast.AppliedType{Constructor: parts[0], Args: args}
	}
	switch spec {
	case "int", "float", "str", "bool", "void":
		return &ast.PrimitiveType{Name: spec}
	default:
		return &ast.NamedType{Name: spec}
	}
}
