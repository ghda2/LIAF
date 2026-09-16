package lexer

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"liaf/pkg/token"
)

type Lexer struct {
	input        string
	position     int  // posição atual do caractere (índice em bytes)
	readPosition int  // próxima posição de leitura
	ch           rune // caractere atual
	line         int  // linha atual (1-based)
	col          int  // coluna atual (1-based)
	Errors       []string
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
		l.position = l.readPosition
		l.readPosition++
		l.col++
	} else {
		r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.position = l.readPosition
		l.readPosition += size
		l.col++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
		} else if l.ch == '\n' {
			l.line++
			l.col = 0
			l.readChar()
		} else if l.ch == ';' {
			// Comentário até o fim da linha
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
		} else {
			break
		}
	}
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()

	tokLine := l.line
	tokCol := l.col

	var tok token.Token

	switch l.ch {
	case 0:
		if l.position < len(l.input) {
			l.readChar()
			return token.Token{Type: token.ILLEGAL, Literal: "NUL", Line: tokLine, Col: tokCol}
		}
		tok = token.Token{Type: token.EOF, Literal: "", Line: tokLine, Col: tokCol}
	case '(':
		tok = token.Token{Type: token.LPAREN, Literal: "(", Line: tokLine, Col: tokCol}
		l.readChar()
	case ')':
		tok = token.Token{Type: token.RPAREN, Literal: ")", Line: tokLine, Col: tokCol}
		l.readChar()
	case '"':
		str, ok := l.readString()
		if !ok {
			tok = token.Token{Type: token.ILLEGAL, Literal: str, Line: tokLine, Col: tokCol}
		} else {
			tok = token.Token{Type: token.STRING, Literal: str, Line: tokLine, Col: tokCol}
		}
	default:
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			tokType := token.LookupIdent(lit)
			return token.Token{Type: tokType, Literal: lit, Line: tokLine, Col: tokCol}
		} else if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			lit, isFloat, ok := l.readNumber()
			if !ok {
				tok = token.Token{Type: token.ILLEGAL, Literal: lit, Line: tokLine, Col: tokCol}
			} else if isFloat {
				tok = token.Token{Type: token.FLOAT, Literal: lit, Line: tokLine, Col: tokCol}
			} else {
				tok = token.Token{Type: token.INT, Literal: lit, Line: tokLine, Col: tokCol}
			}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Line: tokLine, Col: tokCol}
			l.addError(fmt.Sprintf("Caractere inválido '%c' em %d:%d", l.ch, tokLine, tokCol))
			l.readChar()
		}
	}

	return tok
}

func (l *Lexer) readIdentifier() string {
	startPos := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '-' || l.ch == '_' {
		l.readChar()
	}
	return l.input[startPos:l.position]
}

func (l *Lexer) readNumber() (string, bool, bool) {
	startPos := l.position
	isFloat := false

	if l.ch == '-' {
		l.readChar()
	}

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' {
		if !isDigit(l.peekChar()) {
			l.addError(fmt.Sprintf("Número float malformado terminado em ponto em %d:%d", l.line, l.col))
			return l.input[startPos:l.position], false, false
		}
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	if isLetter(l.ch) {
		l.addError(fmt.Sprintf("Identificador ou número malformado em %d:%d", l.line, l.col))
		for isLetter(l.ch) || isDigit(l.ch) {
			l.readChar()
		}
		return l.input[startPos:l.position], false, false
	}

	return l.input[startPos:l.position], isFloat, true
}

func (l *Lexer) readString() (string, bool) {
	tokLine := l.line
	tokCol := l.col
	l.readChar()

	var sb strings.Builder

	for {
		if l.ch == 0 || l.ch == '\n' {
			l.addError(fmt.Sprintf("String não terminada iniciada em %d:%d", tokLine, tokCol))
			return sb.String(), false
		}

		if l.ch == '"' {
			l.readChar()
			break
		}

		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'u':
				var hexStr strings.Builder
				for i := 0; i < 4; i++ {
					l.readChar()
					if !isHex(l.ch) {
						l.addError(fmt.Sprintf("Escape unicode inválido \\u em %d:%d", l.line, l.col))
						return sb.String(), false
					}
					hexStr.WriteRune(l.ch)
				}
				val, err := strconv.ParseInt(hexStr.String(), 16, 32)
				if err != nil {
					l.addError(fmt.Sprintf("Erro ao decodificar \\u%s em %d:%d", hexStr.String(), l.line, l.col))
					return sb.String(), false
				}
				sb.WriteRune(rune(val))
			default:
				l.addError(fmt.Sprintf("Escape inválido '\\%c' em %d:%d", l.ch, l.line, l.col))
				return sb.String(), false
			}
			l.readChar()
		} else {
			sb.WriteRune(l.ch)
			l.readChar()
		}
	}

	return sb.String(), true
}

func (l *Lexer) addError(msg string) {
	l.Errors = append(l.Errors, msg)
}

func isLetter(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isHex(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}
