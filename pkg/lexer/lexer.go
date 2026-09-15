package lexer

import (
	"strings"
	"unicode"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	col          int
}

func New(input string) *Lexer {
	l := &Lexer{
		input: input,
		line:  1,
		col:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	tokLine := l.line
	tokCol := l.col

	var tok Token

	switch l.ch {
	case '[':
		tok = Token{Type: TOKEN_LBRACKET, Literal: "[", Line: tokLine, Col: tokCol}
	case ']':
		tok = Token{Type: TOKEN_RBRACKET, Literal: "]", Line: tokLine, Col: tokCol}
	case '(':
		tok = Token{Type: TOKEN_LPAREN, Literal: "(", Line: tokLine, Col: tokCol}
	case ')':
		tok = Token{Type: TOKEN_RPAREN, Literal: ")", Line: tokLine, Col: tokCol}
	case ':':
		tok = Token{Type: TOKEN_COLON, Literal: ":", Line: tokLine, Col: tokCol}
	case '/':
		tok = Token{Type: TOKEN_SLASH, Literal: "/", Line: tokLine, Col: tokCol}
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: TOKEN_ARROW, Literal: "->", Line: tokLine, Col: tokCol}
		} else {
			tok = Token{Type: TOKEN_ILLEGAL, Literal: string(l.ch), Line: tokLine, Col: tokCol}
		}
	case '"':
		strVal := l.readString()
		return Token{Type: TOKEN_STRING, Literal: strVal, Line: tokLine, Col: tokCol}
	case 0:
		tok = Token{Type: TOKEN_EOF, Literal: "", Line: tokLine, Col: tokCol}
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			tokType := lookupIdent(ident)
			return Token{Type: tokType, Literal: ident, Line: tokLine, Col: tokCol}
		} else if isDigit(l.ch) {
			num, isFloat := l.readNumber()
			tokType := TOKEN_INT
			if isFloat {
				tokType = TOKEN_FLOAT
			}
			return Token{Type: tokType, Literal: num, Line: tokLine, Col: tokCol}
		} else {
			tok = Token{Type: TOKEN_ILLEGAL, Literal: string(l.ch), Line: tokLine, Col: tokCol}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
			if l.ch == '\n' {
				l.line++
				l.col = 0
			}
			l.readChar()
		}
		// Comment starting with '#' or ';'
		if l.ch == '#' || l.ch == ';' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}
		break
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readNumber() (string, bool) {
	start := l.position
	isFloat := false
	for isDigit(l.ch) || l.ch == '.' {
		if l.ch == '.' {
			isFloat = true
		}
		l.readChar()
	}
	return l.input[start:l.position], isFloat
}

func (l *Lexer) readString() string {
	l.readChar() // skip initial '"'
	var sb strings.Builder
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			default:
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}
	l.readChar() // skip closing '"'
	return sb.String()
}

func lookupIdent(ident string) TokenType {
	switch ident {
	case "fn":
		return TOKEN_FN
	case "struct":
		return TOKEN_STRUCT
	case "let":
		return TOKEN_LET
	case "return":
		return TOKEN_RETURN
	case "spawn":
		return TOKEN_SPAWN
	case "send":
		return TOKEN_SEND
	case "recv":
		return TOKEN_RECV
	case "if":
		return TOKEN_IF
	case "else":
		return TOKEN_ELSE
	case "true", "false":
		return TOKEN_BOOL
	default:
		return TOKEN_IDENT
	}
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return unicode.IsDigit(rune(ch))
}
