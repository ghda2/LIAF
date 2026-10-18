package builtins_test

import (
	"testing"

	"liaf/pkg/ast"
	"liaf/pkg/builtins"
)

func TestBuiltinsRegistry(t *testing.T) {
	all := builtins.All()
	if len(all) < 50 {
		t.Fatalf("esperava ao menos 50 builtins registrados, obteve %d", len(all))
	}

	for _, name := range []string{"println", "print", "concat", "not", "make-list", "fs-read-file", "db-connect", "ws-send"} {
		b := builtins.Lookup(name)
		if b == nil {
			t.Fatalf("builtin esperado %q nao encontrado", name)
		}
		// Teste de normalização com underscore
		underscored := builtins.Lookup(name)
		if underscored != b {
			t.Fatalf("normalizacao falhou para %q", name)
		}
	}
}

func TestParseType(t *testing.T) {
	cases := []struct {
		spec string
		want string
	}{
		{"int", "int"},
		{"float", "float"},
		{"str", "str"},
		{"bool", "bool"},
		{"void", "void"},
		{"Response", "Response"},
		{"(result int str)", "(result int str)"},
		{"(list str)", "(list str)"},
		{"(result void str)", "(result void str)"},
	}

	for _, tc := range cases {
		typ := builtins.ParseType(tc.spec)
		if typ == nil {
			t.Fatalf("ParseType(%q) retornou nil", tc.spec)
		}
		if got := typ.String(); got != tc.want {
			t.Errorf("ParseType(%q) = %s; want %s", tc.spec, got, tc.want)
		}
	}
}

func TestAddSimpleBuiltinInOneFile(t *testing.T) {
	// Demonstra que um builtin simples novo pode ser registrado diretamente
	builtins.Register(&builtins.Builtin{
		Name:       "test-dummy-sum",
		Category:   "math",
		Arity:      builtins.ExactArity(2),
		Params:     []string{"int", "int"},
		Return:     "int",
		GoTemplate: "(%s + %s)",
		CTemplate:  "vadd(%s, %s)",
	})

	b := builtins.Lookup("test_dummy_sum")
	if b == nil {
		t.Fatal("builtin dummy nao registrado")
	}
	if b.Return != "int" || len(b.Params) != 2 {
		t.Fatal("parametros ou retorno incorretos")
	}
	typ := builtins.ParseType(b.Return)
	if _, ok := typ.(*ast.PrimitiveType); !ok {
		t.Fatal("tipo de retorno deve ser primitivo int")
	}
}
