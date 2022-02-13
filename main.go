package main

import (
	"fmt"
	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalln("need a request to execute")
	}
	id := args[1][1:]

	file, err := parser.Parse("test.hit")
	if err != nil {
		log.Fatalln("failed to parse", err)
	}
	var req parser.Request
	for _, r := range file.Requests {
		if r.ID == id {
			req = r
		}
	}
	if req.ID == "" {
		log.Fatalln("no such request: ", id)
	}
	httpReq, err := request.Generate(file.Global, req)
	if err != nil {
		log.Fatalln("failed to build request:", err)
	}
	// execute
	o, err := httputil.DumpRequestOut(httpReq, true)
	fmt.Println(string(o))
	if err != nil {
		log.Fatalln("failed to dump request", err)
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Fatalln("failed to perform request", err)
	}

	defer resp.Body.Close()
	o, err = httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatalln("failed to dump response", err)
	}
	fmt.Println(string(o))

	// save cached response
	err = cache.Save(req, resp)
	if err != nil {
		log.Fatalln("saving response", err)
	}
}
