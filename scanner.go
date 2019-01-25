package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)


type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	LINE_COMMENT
	BLOCK_COMMENT
	SYMBOL
	NUMBER
	STRING
	COLON
	SEMICOLON
	COMMA
	AT
	DOT
	EQUALS
	DOLLAR
	QUOTE
	SLASH
	QUESTION
	OPEN_BRACE
	CLOSE_BRACE
	OPEN_BRACKET
	CLOSE_BRACKET
	OPEN_PAREN
	CLOSE_PAREN
	OPEN_ANGLE
	CLOSE_ANGLE
//not sure we need to model more than that
	HASH
	AMPERSAND
	STAR
	BACKQUOTE
	TILDE
	BANG
)

type Token struct {
	Type TokenType
	Text string
	Line int
	Start int
}

func (tokenType TokenType) String() string {
	switch tokenType {
	case ILLEGAL: return "ILLEGAL"
	case EOF: return "EOF"
	case LINE_COMMENT: return "LINE_COMMENT"
	case BLOCK_COMMENT: return "BLOCK_COMMENT"
	case SYMBOL: return "SYMBOL"
	case NUMBER: return "NUMBER"
	case STRING: return "STRING"
	case COMMA: return "COMMA"
	case COLON: return "COLON"
	case SEMICOLON: return "SEMICOLON"
	case AT: return "AT"
	case AMPERSAND: return "AMPERSAND"
	case DOT: return "DOT"
	case EQUALS: return "EQUALS"
	case STAR: return "STAR"
	case DOLLAR: return "DOLLAR"
	case QUOTE: return "QUOTE"
	case BACKQUOTE: return "BACKQUOTE"
	case TILDE: return "TILDE"
	case BANG: return "BANG"
	case SLASH: return "SLASH"
	case HASH: return "HASH"
	case QUESTION: return "QUESTION"
	case OPEN_BRACE: return "OPEN_BRACE"
	case CLOSE_BRACE: return "CLOSE_BRACE"
	case OPEN_BRACKET: return "OPEN_BRACKET"
	case CLOSE_BRACKET: return "CLOSE_BRACKET"
	case OPEN_PAREN: return "OPEN_PAREN"
	case CLOSE_PAREN: return "CLOSE_PAREN"
	case OPEN_ANGLE: return "OPEN_ANGLE"
	case CLOSE_ANGLE: return "CLOSE_ANGLE"
	}
	return "?"
}

func (tok Token) String() string {
	return fmt.Sprintf("<%v %q %d:%d>", tok.Type, tok.Text, tok.Line, tok.Start)
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

var eof = rune(0)

type Scanner struct {
	filename string
	r *bufio.Reader
	line int
	column int
	atEOL bool
}

func NewScanner(filename string, r io.Reader) *Scanner {
	return &Scanner{filename: filename, r: bufio.NewReader(r), line: 1, column: 0}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	if ch == '\n' {
		s.line = s.line + 1
		s.column = 0
	} else {
		s.column = s.column + 1
	}
	return ch
}

func (s *Scanner) unread() {
	s.column = s.column - 1
	s.r.UnreadRune()
}

func (s *Scanner) startToken(tokenType TokenType) Token {
	return Token{Type: tokenType, Text: "", Line: s.line, Start: s.column}
}

func (tok Token) finish(text string) Token {
	tok.Text = text
	return tok
}

func (tok Token) illegal(text string) Token {
	tok.Type = ILLEGAL
	tok.Text = text
	return tok
}

func (s *Scanner) Scan() Token {
	for {
		ch := s.read()
		if !isWhitespace(ch) {
			if isLetter(ch) {
				return s.scanSymbol(ch)
			} else if isDigit(ch) || ch == '-' {
				return s.scanNumber(ch)
			} else if ch == '/' {
				return s.scanComment()
			} else if ch == '"' {
				return s.scanString()
			} else {
				return s.scanPunct(ch)
			}
		}
	}
}

func (s *Scanner) scanSymbol(firstChar rune) Token {
	var buf bytes.Buffer
	buf.WriteRune(firstChar)
	tok := s.startToken(SYMBOL)
	
	for {
		ch := s.read()
		if ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			if ch != '\n' {
				s.unread()
			}
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	tok.Text = buf.String()
	return tok
}

func (s *Scanner) scanNumber(firstDigit rune) Token {
	var buf bytes.Buffer
	buf.WriteRune(firstDigit)
	tok := s.startToken(NUMBER)
	gotDecimal := false
	for {
		ch := s.read()
		if ch == eof {
			break
		} else if !isDigit(ch) {
			if ch == '.' {
				buf.WriteRune(ch)
				if gotDecimal {
					return tok.illegal(buf.String())
				}
				gotDecimal = true
			} else {
				if ch != '\n' {
					s.unread()
				}
				break
			}
		} else {
			buf.WriteRune(ch)
		}
	}
	tok.Text = buf.String()	
	return tok	
}

func (s *Scanner) scanComment() Token {
	tok := s.startToken(LINE_COMMENT)
	ch := s.read();
	if ch != eof {
		if ch == '/' {
			var buf bytes.Buffer
			for {
				ch = s.read()
				if ch == eof || ch == '\n' {
					break
				}
				buf.WriteRune(ch)
			}
			return tok.finish(buf.String())
		}
		if ch == '*' {
			var nextToLast bool
			tok.Type = BLOCK_COMMENT
			var buf bytes.Buffer
			for {
				if ch = s.read(); ch == eof {
					return tok.illegal("Unterminated block comment")
				}
				if nextToLast {
					if ch == '/' {
						tok.Text = buf.String()
						return tok.finish(buf.String())
					}
					buf.WriteRune('*')
					buf.WriteRune(ch)
					nextToLast = false
				} else {
					if ch == '*' {
						nextToLast = true
					} else {
						buf.WriteRune(ch)
					}
				}
			}
		}
	}
	tok.Type = SLASH
	return tok.finish("/")
}

func (s *Scanner) scanString() Token {
	escape := false
	var buf bytes.Buffer
	tok := s.startToken(STRING)
	for {
		ch := s.read();
		if ch == eof {
			return tok.illegal("unterminated string")
		}
		if escape {
			switch ch {
			case 'n':
				buf.WriteRune('\n')
				ch = '\n'
			case 't':
				buf.WriteRune('\t')
			case '"':
				buf.WriteRune(ch)
			case '\\':
				buf.WriteRune(ch)
			default:
				buf.WriteRune(ch)
				return tok.illegal("Bad escape char in string: \\" + string(ch))
			}
			escape = false
			continue
		}
		switch ch {
		case '"':
			return tok.finish(buf.String())
		case '\\':
			escape = true
		default:
			buf.WriteRune(ch)
			escape = false
		}
	}
}

func (s *Scanner) scanPunct(ch rune) Token {
	tok := s.startToken(ILLEGAL)
	tok.Text = string(ch)
	switch ch {
	case eof:
		tok.Type = EOF
		tok.Text = ""
	case ';':
		tok.Type = SEMICOLON
	case ':':
		tok.Type = COLON
	case ',':
		tok.Type = COMMA
	case '.':
		tok.Type = DOT
	case '@':
		tok.Type = AT
	case '&':
		tok.Type = AMPERSAND
	case '=':
		tok.Type = EQUALS
	case '*':
		tok.Type = STAR
	case '$':
		tok.Type = DOLLAR
	case '\'':
		tok.Type = QUOTE
	case '`':
		tok.Type = BACKQUOTE
	case '~':
		tok.Type = TILDE
	case '!':
		tok.Type = BANG
	case '/':
		tok.Type = SLASH
	case '#':
		tok.Type = HASH
	case '?':
		tok.Type = QUESTION
	case '{':
		tok.Type = OPEN_BRACE
	case '}':
		tok.Type = CLOSE_BRACE
	case '[':
		tok.Type = OPEN_BRACKET
	case ']':
		tok.Type = CLOSE_BRACKET
	case '(':
		tok.Type = OPEN_PAREN
	case ')':
		tok.Type = CLOSE_PAREN
	case '<':
		tok.Type = OPEN_ANGLE
	case '>':
		tok.Type = CLOSE_ANGLE
	}
	return tok
}

