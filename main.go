package main

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/repr"
)

var hitLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"comment", `[#][^\n]*`},
	{"Ident", `[a-zA-Z_]\w*`},
	{"Path", `/.*`},
	{"Punct", `[[!@#~$%^&*()+_={}\|:;"'<,>.?/]|]`},
	{"Whitespace", `[\t]+`},
	{"NL", `\n`},
	{"SingleSpace", `[ ]{1}`},
})

type File struct {
	Requests []*Requests `@@*`
}

type Requests struct {
	Id     string `"@"@Ident`
	NL     string `| @NL`
	Method string `@Ident`
	Space  string `@SingleSpace`
	Path   string `@Path`
	NL1    string `@NL`
	Start  string `"~"`
	// Body includes new lines
	Body string `(@Ident@NL@SingleSpace)*`
	End  string `"~"`
	NL2  string `| @NL+`
}

var parser = participle.MustBuild(&File{},
	participle.Lexer(hitLexer),
)

var input = `# comment
@c1
GET /foo
~
foobar
body
~
`

func main() {
	hitFile := &File{}
	err := parser.ParseString("", input, hitFile)
	if err != nil {
		panic(err)
	}
	repr.Println(hitFile, repr.Indent("  "), repr.OmitEmpty(true))
}
