package token

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestLexer_Parse(t *testing.T) {
	type fields struct {
		col  int
		line int
		buf  *bufio.Reader
	}
	type args struct {
		data io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Token
		wantErr bool
	}{
		{
			name: "multi lines comment",
			args:args{
				data:bytes.NewBuffer(mustOpenFile("./testdata/multi_comment.txt")),
			},
			want: []Token{
				{Type:Ident,Text:"port", Pos:Position{Line:0,Column:0,}},
				{Type:Ident,Text:`123`, Pos:Position{Line:0,Column:5,}},
				{Type:Comment,Text:"\nthis\nis\na\nmulti-lines\ncomment\n", Pos:Position{Line:1, Column:2,}},
				{Type:EOF, Pos:Position{Line:7,Column:2,}},
			},
			wantErr:false,
		},
		{
			name: "one line with comment",
			args:args{
				data:bytes.NewBufferString(`port 456 // this is comment`),
			},
			want: []Token{
				{Type:Ident,Text:"port", Pos:Position{Line:0,Column:0,}},
				{Type:Ident,Text:`456`, Pos:Position{Line:0,Column:5,}},
				{Type:Comment,Text:" this is comment", Pos:Position{Line:0, Column:11,}},
				{Type:EOF, Pos:Position{Line:0,Column:27,}},
			},
			wantErr:false,
		},
		{
			name: "one line with stringLit",
			args:args{
				data:bytes.NewBuffer(mustOpenFile("./testdata/multi_strlit.txt")),
			},
			want: []Token{
				{Type:Ident,Text:"body", Pos:Position{Line:0,Column:0,}},
				{Type:StringLit,Text:`{"name":"caibirdme", "age": 25}`, Pos:Position{Line:0,Column:6,}},
				{Type:EOF, Pos:Position{Line:0,Column:38,}},
			},
			wantErr:false,
		},
		{
			name: "multi lines with stringLit",
			args:args{
				data:bytes.NewBuffer([]byte(
					`
process 4
http { 
	server {
		server_name "www.example.com"
		port 8080

		location / {
			proxy_pass "www.foo.com"
		}
	}
}
`)),
			},
			want: []Token{
				{Type:Ident,Text:"process", Pos:Position{Line:1,Column:0,}},
				{Type:Ident,Text:"4", Pos:Position{Line:1,Column:8,}},
				{Type:Ident,Text:"http", Pos:Position{Line:2,Column:0,}},
				{Type:LBracket,Text:"{", Pos:Position{Line:2,Column:5,}},
				{Type: Ident, Text:"server", Pos:Position{Line:3, Column:1,},},
				{Type: LBracket, Text:"{", Pos:Position{Line:3, Column:8,},},
				{Type: Ident, Text:"server_name", Pos:Position{Line:4, Column:2,},},
				{Type: StringLit, Text:"www.example.com", Pos:Position{Line:4, Column:15,},},
				{Type: Ident, Text:"port", Pos:Position{Line:5, Column:2,},},
				{Type: Ident, Text:"8080", Pos:Position{Line:5, Column:7,},},
				{Type: Ident, Text:"location", Pos:Position{Line:7, Column:2,},},
				{Type: Ident, Text:"/", Pos:Position{Line:7, Column:11,},},
				{Type: LBracket, Text:"{", Pos:Position{Line:7, Column:13,},},
				{Type: Ident, Text:"proxy_pass", Pos:Position{Line:8, Column:3,},},
				{Type: StringLit, Text:"www.foo.com", Pos:Position{Line:8, Column:15,},},
				{Type: RBracket, Text:"}", Pos:Position{Line:9, Column:2,},},
				{Type: RBracket, Text:"}", Pos:Position{Line:10, Column:1,},},
				{Type: RBracket, Text:"}", Pos:Position{Line:11, Column:0,},},
				{Type:EOF, Pos:Position{Line:12,}},
			},
			wantErr:false,
		},
		{
			name: "multi lines",
			args:args{
				data:bytes.NewBuffer([]byte(
`
process 4
http { 
	server {
		server_name www.example.com
		port 8080

		location / {
			proxy_pass www.foo.com
		}
	}
}
`)),
			},
			want: []Token{
				{Type:Ident,Text:"process", Pos:Position{Line:1,Column:0,}},
				{Type:Ident,Text:"4", Pos:Position{Line:1,Column:8,}},
				{Type:Ident,Text:"http", Pos:Position{Line:2,Column:0,}},
				{Type:LBracket,Text:"{", Pos:Position{Line:2,Column:5,}},
				{Type: Ident, Text:"server", Pos:Position{Line:3, Column:1,},},
				{Type: LBracket, Text:"{", Pos:Position{Line:3, Column:8,},},
				{Type: Ident, Text:"server_name", Pos:Position{Line:4, Column:2,},},
				{Type: Ident, Text:"www.example.com", Pos:Position{Line:4, Column:14,},},
				{Type: Ident, Text:"port", Pos:Position{Line:5, Column:2,},},
				{Type: Ident, Text:"8080", Pos:Position{Line:5, Column:7,},},
				{Type: Ident, Text:"location", Pos:Position{Line:7, Column:2,},},
				{Type: Ident, Text:"/", Pos:Position{Line:7, Column:11,},},
				{Type: LBracket, Text:"{", Pos:Position{Line:7, Column:13,},},
				{Type: Ident, Text:"proxy_pass", Pos:Position{Line:8, Column:3,},},
				{Type: Ident, Text:"www.foo.com", Pos:Position{Line:8, Column:14,},},
				{Type: RBracket, Text:"}", Pos:Position{Line:9, Column:2,},},
				{Type: RBracket, Text:"}", Pos:Position{Line:10, Column:1,},},
				{Type: RBracket, Text:"}", Pos:Position{Line:11, Column:0,},},
				{Type:EOF, Pos:Position{Line:12,}},
			},
			wantErr:false,
		},
		{
			name: "one line but with multi ws",
			args:args{
				data:bytes.NewBuffer([]byte(`

					foo bar baz

				`)),
			},
			want: []Token{
				{Type:Ident,Text:"foo", Pos:Position{Line:2,Column:5,}},
				{Type:Ident,Text:"bar", Pos:Position{Line:2,Column:9,}},
				{Type:Ident,Text:"baz", Pos:Position{Line:2,Column:13,}},
				{Type:EOF,Text:"", Pos:Position{Line:4,Column:4,}},
			},
			wantErr:false,
		},
		{
			name: "one line",
			args:args{
				data:bytes.NewBuffer([]byte(`foo bar baz`)),
			},
			want: []Token{
				{Type:Ident,Text:"foo", Pos:Position{Line:0,Column:0,}},
				{Type:Ident,Text:"bar", Pos:Position{Line:0,Column:4,}},
				{Type:Ident,Text:"baz", Pos:Position{Line:0,Column:8,}},
				{Type:EOF,Text:"", Pos:Position{Line:0,Column:11,}},
			},
			wantErr:false,
		},
		{
			name: "one line with semicolon",
			args:args{
				data:bytes.NewBuffer([]byte(`foo bar baz; a b;`)),
			},
			want: []Token{
				{Type:Ident,Text:"foo", Pos:Position{Line:0,Column:0,}},
				{Type:Ident,Text:"bar", Pos:Position{Line:0,Column:4,}},
				{Type:Ident,Text:"baz", Pos:Position{Line:0,Column:8,}},
				{Type:Semicolon,Text:";", Pos:Position{Line:0,Column:11,}},
				{Type:Ident,Text:"a", Pos:Position{Line:0,Column:13,}},
				{Type:Ident,Text:"b", Pos:Position{Line:0,Column:15,}},
				{Type:Semicolon,Text:";", Pos:Position{Line:0,Column:16,}},
				{Type:EOF,Text:"", Pos:Position{Line:0,Column:17,}},
			},
			wantErr:false,
		},
		{
			name: "semicolon without whitespace",
			args:args{
				data:bytes.NewBuffer([]byte(`foo bar baz;ab`)),
			},
			want: []Token{
				{Type:Ident,Text:"foo", Pos:Position{Line:0,Column:0,}},
				{Type:Ident,Text:"bar", Pos:Position{Line:0,Column:4,}},
				{Type:Ident,Text:"baz", Pos:Position{Line:0,Column:8,}},
				{Type:Semicolon,Text:";", Pos:Position{Line:0,Column:11,}},
				{Type:Ident,Text:"ab", Pos:Position{Line:0,Column:12,}},
				{Type:EOF,Text:"", Pos:Position{Line:0,Column:14,}},
			},
			wantErr:false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Lexer{
				col:  tt.fields.col,
				line: tt.fields.line,
				buf:  tt.fields.buf,
			}
			got, err := l.parse(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lexer.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Lexer.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustOpenFile(name string) []byte {
	f,err := os.Open(name)
	if err != nil {
		panic(err)
	}
	data,err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return data
}