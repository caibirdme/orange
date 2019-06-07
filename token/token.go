package token

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Token struct {
	Type TokenType
	Text string
	Pos  Position
}

func (t *Token) String() string {
	return fmt.Sprintf("Text: %s, near: [%s]", t.Text, t.Pos.String())
}

type Position struct {
	Line   int
	Column int
}

func (pos *Position) String() string {
	return fmt.Sprintf("line: %d, col: %d", pos.Line, pos.Column)
}

type TokenType int

const (
	Ident TokenType = iota
	StringLit
	Semicolon
	Comment
	LBracket
	RBracket
	EOF
)

/*
State machine:

StringLit:
	` [^`]* `
	" [^"]* "

Semicolon: ;
Ident:	letter (letter|digit|_)*
Comment:
	/* .* \*\/
	// .* \n
LBracket: {
RBracket: }
*/

type Lexer struct {
	col  int
	line int
	buf  *bufio.Reader
}

func (l *Lexer) Peek1() (byte, error) {
	s, err := l.buf.Peek(1)
	if err != nil {
		return 0, err
	}
	return s[0], nil
}

func (l *Lexer) PeekStr(str string) bool {
	n := len(str)
	peeked, err := l.buf.Peek(n)
	if err != nil {
		return false
	}
	return string(peeked) == str
}

func (l *Lexer) PeekByte(c byte) bool {
	p, err := l.Peek1()
	if err != nil {
		return false
	}
	return p == c
}

func (l *Lexer) ReadUntilWhiteSpace() (string, error) {
	buf := bytes.NewBuffer(nil)
	for {
		b, err := l.buf.ReadByte()
		if err != nil {
			return buf.String(), err
		}
		if b == ';' {
			err = l.buf.UnreadByte()
			if err != nil {
				return buf.String(), fmt.Errorf("[Bug] ReadUntilWhiteSpace UnreadByte err: %s", err)
			}
			break
		}
		l.col++
		if b == ' ' || b == '\t' {
			break
		}
		if b == '\r' {
			if c,err := l.Peek1(); err == nil && c == '\n' {
				l.line++
				l.col = 0
				break
			}
		} else if b == '\n' {
			l.line++
			l.col = 0
			break
		}
		buf.WriteByte(b)
	}
	return buf.String(), nil
}

func (l *Lexer) EatWhiteSpace() {
	for {
		b, err := l.Peek1()
		if err != nil {
			return
		}
		if b == ' ' || b == '\t' {
			l.col += 1
			l.Discard(1)
		} else if b == '\r' {
			if c, err := l.Peek1(); err == nil && c == '\n' {
				l.line += 1
				l.col = 0
				l.Discard(2)
			} else {
				l.col += 1
				l.Discard(1)
			}
		} else if b == '\n' {
			l.line += 1
			l.col = 0
			l.Discard(1)
		} else {
			return
		}
	}
}

func (l *Lexer) ReadUntil(str string, include bool) (string, error) {
	buf := bytes.NewBuffer(nil)
	delim := str[0]
	n := len(str)
	for {
		b, err := l.buf.ReadByte()
		if err != nil {
			return buf.String(), err
		}
		l.col++
		if b == '\n' {
			l.line += 1
			l.col = 0
		}
		if b == delim {
			if n == 1 {
				if include {
					buf.WriteByte(b)
				}
				break
			}
			peeked, err := l.buf.Peek(n - 1)
			if err == nil && string(peeked) == str[1:] {
				_, _ = l.buf.Discard(n - 1)
				l.col+=n-1
				if include {
					buf.WriteByte(b)
					buf.Write(peeked)
				}
				break
			}
		}
		buf.WriteByte(b)
	}
	return buf.String(), nil
}

// Discard the caller should make sure the n is valid or it may panic
func (l *Lexer) Discard(n int) {
	_, err := l.buf.Discard(n)
	if err != nil {
		panic(err)
	}
}

func NewToken(text string, pos Position, t TokenType) Token {
	return Token{
		Type: t,
		Text: text,
		Pos:  pos,
	}
}

func (l *Lexer) parse(data io.Reader) ([]Token, error) {
	l.buf = bufio.NewReader(data)
	var tokens []Token
	for {
		l.EatWhiteSpace()
		switch {
		case l.PeekStr("//"):
			l.Discard(2)
			l.col+=2
			pos := l.curPos()
			commentText, err := l.ReadUntil("\n", false)
			if err != nil && err != io.EOF {
				return nil, l.ErrPos(err)
			}
			n := len(commentText)
			if n > 0 && commentText[n-1] == '\r' {
				commentText = commentText[:n-1]
			}
			tokens = append(tokens, NewToken(commentText, pos, Comment))
		case l.PeekStr("/*"):
			l.Discard(2)
			l.col+=2
			pos := l.curPos()
			commentText, err := l.ReadUntil("*/", false)
			if err != nil {
				return nil, l.ErrPos(err)
			}
			tokens = append(tokens, NewToken(commentText, pos, Comment))
		case l.PeekByte('`'):
			l.Discard(1)
			l.col++
			pos := l.curPos()
			strLit, err := l.ReadUntil("`", false)
			if err != nil {
				return nil, l.ErrPos(err)
			}
			tokens = append(tokens, NewToken(strLit, pos, StringLit))
		case l.PeekByte('"'):
			l.Discard(1)
			l.col++
			pos := l.curPos()
			strLit, err := l.ReadUntil("\"", false)
			if err != nil {
				return nil, l.ErrPos(err)
			}
			tokens = append(tokens, NewToken(strLit, pos, StringLit))
		case l.PeekByte('{'):
			l.Discard(1)
			pos := l.curPos()
			l.col++
			tokens = append(tokens, NewToken("{", pos, LBracket))
		case l.PeekByte('}'):
			l.Discard(1)
			pos := l.curPos()
			l.col++
			tokens = append(tokens, NewToken("}", pos, RBracket))
		case l.PeekByte(';'):
			l.Discard(1)
			pos := l.curPos()
			l.col++
			tokens = append(tokens, NewToken(";", pos, Semicolon))
		default:
			// EOF
			if _, err := l.Peek1(); err == io.EOF {
				tokens = append(tokens, NewToken("", l.curPos(), EOF))
				return tokens, nil
			}

			// Ident
			pos := l.curPos()
			identText, err := l.ReadUntilWhiteSpace()
			if err != nil && err != io.EOF {
				return nil, l.ErrPos(err)
			}
			tokens = append(tokens, NewToken(identText, pos, Ident))
		}
	}
	// unreachable!
}

func (l *Lexer) Parse(data io.Reader) (TokenStream, error) {
	tokens,err := l.parse(data)
	if err != nil {
		return nil, err
	}
	return NewTokenStream(tokens), nil
}

func (l *Lexer) ErrPos(err error) error {
	return fmt.Errorf("err occurs near [line: %d, col: %d], %s", l.line, l.col, err)
}

func (l *Lexer) curPos() Position {
	return Position{
		Line:   l.line,
		Column: l.col,
	}
}

func NewTokenStream(tokens []Token) TokenStream {
	tokensWithoutComments := make([]Token, 0, len(tokens))
	for _,t := range tokens {
		if t.Type != Comment {
			tokensWithoutComments = append(tokensWithoutComments, t)
		}
	}
	return &tokenStream{
		tokens:tokensWithoutComments,
	}
}

type TokenStream interface {
	Next() (Token, bool)
	Peek() (Token, bool)
	BreakPoint()
	Rollback()
}

type tokenStream struct {
	tokens []Token
	idx int
	bps []int
}

func (ts *tokenStream) BreakPoint() {
	ts.bps = append(ts.bps, ts.idx)
}

func (ts *tokenStream) Rollback() {
	n := len(ts.bps)
	if n > 0 {
		ts.idx = ts.bps[n-1]
		ts.bps = ts.bps[:n-1]
	}
}

func (ts *tokenStream) Next() (Token, bool) {
	t,ok := ts.Peek()
	if ok {
		ts.idx++
	}
	return t,ok
}

func (ts *tokenStream) Peek() (Token, bool) {
	if ts.idx < len(ts.tokens) {
		return ts.tokens[ts.idx], true
	}
	return Token{}, false
}