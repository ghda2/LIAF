package lexer

import (
	"testing"

	"liaf/pkg/token"
)

func TestLexerBasicTokens(t *testing.T) {
	input := `
; Comentário de exemplo
(module math-lib
  (fn sum
    (params
      (a int)
      (b int))
    (returns int)
    (effects)
    (body
      (return (add a b)))))
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.LPAREN, "("},
		{token.MODULE, "module"},
		{token.IDENT, "math-lib"},
		{token.LPAREN, "("},
		{token.FN, "fn"},
		{token.IDENT, "sum"},
		{token.LPAREN, "("},
		{token.PARAMS, "params"},
		{token.LPAREN, "("},
		{token.IDENT, "a"},
		{token.IDENT, "int"},
		{token.RPAREN, ")"},
		{token.LPAREN, "("},
		{token.IDENT, "b"},
		{token.IDENT, "int"},
		{token.RPAREN, ")"},
		{token.RPAREN, ")"},
		{token.LPAREN, "("},
		{token.RETURNS, "returns"},
		{token.IDENT, "int"},
		{token.RPAREN, ")"},
		{token.LPAREN, "("},
		{token.EFFECTS, "effects"},
		{token.RPAREN, ")"},
		{token.LPAREN, "("},
		{token.BODY, "body"},
		{token.LPAREN, "("},
		{token.RETURN, "return"},
		{token.LPAREN, "("},
		{token.ADD, "add"},
		{token.IDENT, "a"},
		{token.IDENT, "b"},
		{token.RPAREN, ")"},
		{token.RPAREN, ")"},
		{token.RPAREN, ")"},
		{token.RPAREN, ")"},
		{token.RPAREN, ")"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - token type errado. esperado=%q, obteve=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("test[%d] - literal errado. esperado=%q, obteve=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}

	if len(l.Errors) > 0 {
		t.Fatalf("erros inesperados no lexer: %v", l.Errors)
	}
}

func TestLexerLiteralsAndEscapes(t *testing.T) {
	input := `42 -100 3.14 -0.05 "hello\nworld" "tab\t\"escaped\"" true false`

	expected := []struct {
		tokType token.TokenType
		lit     string
	}{
		{token.INT, "42"},
		{token.INT, "-100"},
		{token.FLOAT, "3.14"},
		{token.FLOAT, "-0.05"},
		{token.STRING, "hello\nworld"},
		{token.STRING, "tab\t\"escaped\""},
		{token.BOOL, "true"},
		{token.BOOL, "false"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.tokType {
			t.Fatalf("[%d] tipo errado: esperado %v, obteve %v", i, exp.tokType, tok.Type)
		}
		if tok.Literal != exp.lit {
			t.Fatalf("[%d] literal errado: esperado %q, obteve %q", i, exp.lit, tok.Literal)
		}
	}
}

func TestLexerErrors(t *testing.T) {
	l1 := New(`"string não fechada`)
	tok1 := l1.NextToken()
	if tok1.Type != token.ILLEGAL {
		t.Errorf("esperava token ILLEGAL para string não fechada, obteve %v", tok1.Type)
	}
	if len(l1.Errors) == 0 {
		t.Errorf("esperava erro registrado no lexer")
	}

	l2 := New(`"teste \x inválido"`)
	tok2 := l2.NextToken()
	if tok2.Type != token.ILLEGAL {
		t.Errorf("esperava token ILLEGAL para escape inválido, obteve %v", tok2.Type)
	}
}
