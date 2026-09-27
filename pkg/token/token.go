package token

type TokenType string

const (
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	// Delimiters
	LPAREN TokenType = "("
	RPAREN TokenType = ")"

	// Literals & Identifiers
	IDENT  TokenType = "IDENT"
	INT    TokenType = "INT"
	FLOAT  TokenType = "FLOAT"
	BOOL   TokenType = "BOOL"
	STRING TokenType = "STRING"

	// Keywords
	MODULE  TokenType = "module"
	IMPORT  TokenType = "import"
	STRUCT  TokenType = "struct"
	FIELDS  TokenType = "fields"
	FN      TokenType = "fn"
	PARAMS  TokenType = "params"
	RETURNS TokenType = "returns"
	EFFECTS TokenType = "effects"
	BODY    TokenType = "body"

	// Statements
	LET       TokenType = "let"
	SET       TokenType = "set"
	RETURN    TokenType = "return"
	IF        TokenType = "if"
	THEN      TokenType = "then"
	ELSE      TokenType = "else"
	SPAWN     TokenType = "spawn"
	SEND      TokenType = "send"
	DO        TokenType = "do"
	WHILE     TokenType = "while"
	FOR_RANGE TokenType = "for-range"
	FOR_EACH  TokenType = "for-each"
	BREAK     TokenType = "break"
	CONTINUE  TokenType = "continue"
	MATCH     TokenType = "match"
	ON_ERR    TokenType = "on-err"
	ROUTE     TokenType = "route"

	// WebSocket (issue #016)
	WS_ROUTE   TokenType = "ws-route"
	ON_OPEN    TokenType = "on-open"
	ON_MESSAGE TokenType = "on-message"
	ON_CLOSE   TokenType = "on-close"

	// Banco de dados (issue #015)
	DB_TRANSACTION TokenType = "db-transaction"

	// Expressions
	CALL TokenType = "call"
	TRY  TokenType = "try"
	RECV TokenType = "recv"

	// Operators
	ADD TokenType = "add"
	SUB TokenType = "sub"
	MUL TokenType = "mul"
	DIV TokenType = "div"
	EQ  TokenType = "eq"
	NEQ TokenType = "neq"
	GT  TokenType = "gt"
	LT  TokenType = "lt"
	GTE TokenType = "gte"
	LTE TokenType = "lte"
	AND TokenType = "and"
	OR  TokenType = "or"
)

var keywords = map[string]TokenType{
	"while": WHILE, "for-range": FOR_RANGE, "for-each": FOR_EACH,
	"break": BREAK, "continue": CONTINUE, "match": MATCH,
	"on-err": ON_ERR, "route": ROUTE, "try": TRY,
	"ws-route": WS_ROUTE, "on-open": ON_OPEN, "on-message": ON_MESSAGE, "on-close": ON_CLOSE,
	"db-transaction": DB_TRANSACTION,
	"module":  MODULE,
	"import":  IMPORT,
	"struct":  STRUCT,
	"fields":  FIELDS,
	"fn":      FN,
	"params":  PARAMS,
	"returns": RETURNS,
	"effects": EFFECTS,
	"body":    BODY,
	"let":     LET,
	"set":     SET,
	"return":  RETURN,
	"if":      IF,
	"then":    THEN,
	"else":    ELSE,
	"spawn":   SPAWN,
	"send":    SEND,
	"recv":    RECV,
	"do":      DO,
	"call":    CALL,
	"add":     ADD,
	"sub":     SUB,
	"mul":     MUL,
	"div":     DIV,
	"eq":      EQ,
	"neq":     NEQ,
	"gt":      GT,
	"lt":      LT,
	"gte":     GTE,
	"lte":     LTE,
	"and":     AND,
	"or":      OR,
	"true":    BOOL,
	"false":   BOOL,
}

// LookupIdent retorna o tipo de token para um identificador ou palavra-chave
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// IsName diz se o token pode ocupar uma posicao de nome de campo. Alem de
// identificadores, aceita as palavras reservadas: o nome de um campo e so um
// rotulo (e a chave no JSON), e formatos externos usam nomes como "sub" e
// "return" que a LIAF reserva. true e false ficam de fora porque tem valor.
func IsName(tok Token) bool {
	if tok.Type == IDENT {
		return true
	}
	kw, ok := keywords[tok.Literal]
	return ok && kw == tok.Type && tok.Type != BOOL
}

// IsOperator verifica se o tipo de token é um operador do núcleo
func IsOperator(tok TokenType) bool {
	switch tok {
	case ADD, SUB, MUL, DIV, EQ, NEQ, GT, LT, GTE, LTE, AND, OR:
		return true
	default:
		return false
	}
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}
