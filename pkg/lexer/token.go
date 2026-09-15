package lexer

type TokenType string

const (
	TOKEN_EOF      TokenType = "EOF"
	TOKEN_ILLEGAL  TokenType = "ILLEGAL"

	// Delimiters
	TOKEN_LBRACKET TokenType = "["
	TOKEN_RBRACKET TokenType = "]"
	TOKEN_LPAREN   TokenType = "("
	TOKEN_RPAREN   TokenType = ")"
	TOKEN_COLON    TokenType = ":"
	TOKEN_ARROW    TokenType = "->"
	TOKEN_SLASH    TokenType = "/"

	// Literals & Identifiers
	TOKEN_IDENT  TokenType = "IDENT"
	TOKEN_INT    TokenType = "INT"
	TOKEN_FLOAT  TokenType = "FLOAT"
	TOKEN_STRING TokenType = "STRING"
	TOKEN_BOOL   TokenType = "BOOL"

	// Keywords
	TOKEN_FN     TokenType = "fn"
	TOKEN_STRUCT TokenType = "struct"
	TOKEN_LET    TokenType = "let"
	TOKEN_RETURN TokenType = "return"
	TOKEN_SPAWN  TokenType = "spawn"
	TOKEN_SEND   TokenType = "send"
	TOKEN_RECV   TokenType = "recv"
	TOKEN_IF     TokenType = "if"
	TOKEN_ELSE   TokenType = "else"
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}
