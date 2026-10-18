package checker

import (
	"testing"

	"liaf/pkg/lexer"
	"liaf/pkg/parser"
)

func checkCode(code string) (diagnosticsCount int, hasErrorCode func(code string) bool) {
	l := lexer.New(code)
	p := parser.New(l, "test.liaf")
	mod := p.ParseModule()
	_, diags := Check(mod, "test.liaf")
	return len(diags), func(expectedCode string) bool {
		for _, d := range diags {
			if d.Code == expectedCode {
				return true
			}
		}
		return false
	}
}

func TestCheckerEnumSuccess(t *testing.T) {
	code := `
	enum Status
		Pendente
		Pago
		Cancelado
	end

	fn checar(s Status) str
		match s
			case Status.Pendente
				"Aguardando"
			case Status.Pago
				"Confirmado"
			case Status.Cancelado
				"Cancelado"
		end
	end

	fn main() void
	end
	`
	l := lexer.New(code)
	p := parser.New(l, "test.liaf")
	mod := p.ParseModule()
	_, diags := Check(mod, "test.liaf")
	if len(diags) > 0 {
		t.Fatalf("Esperado 0 erros, obteve %d: %+v", len(diags), diags)
	}
}

func TestCheckerEnumWildcard(t *testing.T) {
	code := `
	enum Status
		Pendente
		Pago
		Cancelado
	end

	fn checar(s Status) str
		match s
			case Status.Pago
				"Confirmado"
			case _
				"Outro"
		end
	end

	fn main() void
	end
	`
	count, _ := checkCode(code)
	if count > 0 {
		t.Fatalf("Esperado 0 erros com wildcard, obteve %d", count)
	}
}

func TestCheckerEnumNonExhaustive(t *testing.T) {
	code := `
	enum Status
		Pendente
		Pago
		Cancelado
	end

	fn checar(s Status) str
		match s
			case Status.Pago
				"Confirmado"
		end
	end
	`
	_, hasCode := checkCode(code)
	if !hasCode("E_NONEXHAUSTIVE_MATCH") {
		t.Fatalf("Esperado erro E_NONEXHAUSTIVE_MATCH quando faltam variantes")
	}
}

func TestCheckerEnumUnknownVariant(t *testing.T) {
	code := `
	enum Status
		Pendente
		Pago
	end

	fn checar(s Status) str
		match s
			case Status.Inexistente
				"Erro"
			case _
				"Ok"
		end
	end
	`
	_, hasCode := checkCode(code)
	if !hasCode("E_UNKNOWN_VARIANT") {
		t.Fatalf("Esperado erro E_UNKNOWN_VARIANT para variante inexistente")
	}
}

func TestCheckerEnumDuplicateCase(t *testing.T) {
	code := `
	enum Status
		Pendente
		Pago
	end

	fn checar(s Status) str
		match s
			case Status.Pago
				"1"
			case Status.Pago
				"2"
			case Status.Pendente
				"3"
		end
	end
	`
	_, hasCode := checkCode(code)
	if !hasCode("E_DUPLICATE_CASE") {
		t.Fatalf("Esperado erro E_DUPLICATE_CASE para caso duplicado")
	}
}
