package runtime

import (
	"testing"
)

func TestRegexMatch(t *testing.T) {
	// E-mail basico
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	res := RegexMatch(emailPattern, "dev@liaf.org")
	if !res.OK || !res.Value {
		t.Fatalf("dev@liaf.org deveria ser valido: %v", res)
	}

	resInvalid := RegexMatch(emailPattern, "nao-eh-email")
	if !resInvalid.OK || resInvalid.Value {
		t.Fatalf("nao-eh-email deveria ser falso")
	}

	// Padrao invalido
	resErr := RegexMatch("[a-z", "abc")
	if resErr.OK {
		t.Fatalf("padrao invalido deveria retornar erro")
	}
}

func TestRegexFindAndFindAll(t *testing.T) {
	numPattern := `\d+`
	res := RegexFind(numPattern, "temos 42 pedidos e 100 clientes")
	if !res.OK || !res.Value.Some || res.Value.Value != "42" {
		t.Fatalf("primeiro numero deveria ser 42: %v", res)
	}

	all := RegexFindAll(numPattern, "temos 42 pedidos e 100 clientes")
	if !all.OK || len(all.Value.Items) != 2 || all.Value.Items[0] != "42" || all.Value.Items[1] != "100" {
		t.Fatalf("todos os numeros deveriam ser [42, 100]: %v", all)
	}
}

func TestRegexReplace(t *testing.T) {
	res := RegexReplace(`\s+`, "um   dois    tres", " ")
	if !res.OK || res.Value != "um dois tres" {
		t.Fatalf("replace falhou: %v", res)
	}
}
