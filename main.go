package main

import (
	"fmt"
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/alecthomas/repr"
	"log"
	"net/http"
	"os"
)

var hitLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"comment", `[#][^\n]*`},
	{"Ident", `[a-zA-Z_]\w*`},
	{"Path", `/.*`},
	{"Punct", `[[!@#~$%^&*()+_={}\|:;"'<,>.?/]|]`},
	{"whitespace", `[\s]+`},
	{"jump", `[\n]+`},
	{"Marker", `[\-]{3}`},
})

type File struct {
	Requests []*Requests `@@*`
}

type Requests struct {
	Id           string `"---" "@"@Ident`
	Method       string `@Ident?`
	Path         string `@Path `
	BodyEncoding string `("~"@Ident)?`
	Body         string `@!Marker* Marker`
}

var parser = participle.MustBuild(&File{},
	participle.Lexer(hitLexer),
)

func main() {
	hitFile := &File{}
	err := parser.Parse("", os.Stdin, hitFile)
	if err != nil {
		panic(err)
	}
	repr.Println(hitFile, repr.Indent("  "), repr.OmitEmpty(true))
	for _, req := range hitFile.Requests {
		fmt.Println(req.Body)
		if req.Id == "c1" {
			req, err := http.NewRequest(req.Method, "http://httpbin.org"+req.Path, nil)
			if err != nil {
				log.Println(err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Println(err)
			}
			repr.Println(res.StatusCode)
		}
	}
}
