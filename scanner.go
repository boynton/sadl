package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"io/ioutil"
	"strings"
	"path"
	"path/filepath"
)

//TODO remove this
func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: scanner file.sadl")
		os.Exit(1)
	}
	path := os.Args[1]
	fi, err := os.Open(path)
	if err != nil {
		panic("Can't open file")
	}
	defer fi.Close()	
	reader := bufio.NewReader(fi)
	scanner := NewScanner(path, reader)
	for {
		tok := scanner.Scan()
		if tok.Type == EOF {
			break
		}
		if tok.Type != BLOCK_COMMENT { //ignore those
			msg := tok.Type.String()
			fmt.Println(scanner.formattedAnnotation("", msg, tok, RED, -1))
		}
	}
}


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
	OPEN_BRACE
	CLOSE_BRACE
	OPEN_BRACKET
	CLOSE_BRACKET
	OPEN_PAREN
	CLOSE_PAREN
	OPEN_ANGLE
	CLOSE_ANGLE
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
	case DOT: return "DOT"
	case EQUALS: return "EQUALS"
	case DOLLAR: return "DOLLAR"
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
			} else if isDigit(ch) {
				return s.scanNumber(ch)
			} else if ch == '/' {
				return s.scanComment()
			} else if ch == '"' || ch == '\'' {
				return s.scanString(ch)
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
	return tok.illegal("/" + string(ch) + "...")
}

func (s *Scanner) scanString(matchingQuote rune) Token {
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
			case matchingQuote:
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
		case matchingQuote:
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
	case '=':
		tok.Type = EQUALS
	case '$':
		tok.Type = DOLLAR
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

/*
func (s *Scanner) error(msg string) {
	s := s.formattedAnnotation(msg)
	p.err = fmt.Errorf(s)
}
*/

const BLACK = "\033[0;0m"
const RED = "\033[0;31m"
const YELLOW = "\033[0;33m"
const BLUE = "\033[94m"
const GREEN = "\033[92m"

func (s *Scanner) formattedAnnotation(prefix string, msg string, tok Token, color string, contextSize int) string {
	if len(s.filename) > 0 {
		data, err := ioutil.ReadFile(s.filename)
		if err == nil && contextSize >= 0 {
			lines := strings.Split(string(data), "\n")
			line := tok.Line - 1
			begin := max(0, line-contextSize)
				end := min(len(lines), line+contextSize+1)
			context := lines[begin:end]
			tmp := ""
			for i, l := range context {
				if i+begin == line {
					toklen := len(tok.Text)
					if tok.Type == STRING {
						toklen = len(fmt.Sprintf("%q", tok.Text))
					} else if tok.Type == LINE_COMMENT {
						toklen = toklen + 2
					}
					left := l[:tok.Start-1]
					mid := l[tok.Start-1:tok.Start-1+toklen]
					right := l[tok.Start-1+toklen:]
					tmp += fmt.Sprintf("%3d\t%v", i+begin+1, left)
					tmp += fmt.Sprintf("%s%v%s", color, mid, BLACK)
					tmp += fmt.Sprintf("%v\n", right)
				} else {
					tmp += fmt.Sprintf("%3d\t%v\n", i+begin+1, l)
				}
			}
			return fmt.Sprintf("%s%s:%d:%d: %s%s%s\n%s", prefix, path.Base(s.filename), tok.Line, tok.Start, color, msg, BLACK, tmp)
		}
		return fmt.Sprintf("%s%s:%d:%d: %s", prefix, filepath.Base(s.filename), tok.Line, tok.Start, msg)
	}
	return fmt.Sprintf("%s%d:%d: %s", prefix, tok.Line, tok.Start, msg)
}

func max(n1 int, n2 int) int {
	if n1 > n2 {
		return n1
	}
	return n2
}

func min(n1 int, n2 int) int {
	if n1 < n2 {
		return n1
	}
	return n2
}

