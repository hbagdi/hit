package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
)

var VERSION = "dev"
var COMMIT_HASH = "dev"

const (
	minArgs = 2
	timeout = 10 * time.Second
)

func Run(ctx context.Context) error {
	args := os.Args
	if len(args) < minArgs {
		return fmt.Errorf("need a request to execute")
	}
	id := args[1]

	if id == "version" {
		fmt.Printf("%s (commit: %s)\n", VERSION, COMMIT_HASH)
		return nil
	}
	if id[0] != '@' {
		return fmt.Errorf("request must begin with '@' character")
	}
	id = id[1:]

	fileName := "test.hit"
	file, err := parser.Parse(fileName)
	if err != nil {
		return fmt.Errorf("failed to parse file '%v': %v", fileName, err)
	}
	var req parser.Request
	for _, r := range file.Requests {
		if r.ID == id {
			req = r
		}
	}
	if req.ID == "" {
		return fmt.Errorf("no such request: %v", id)
	}
	httpReq, err := request.Generate(file.Global, req)
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}
	// execute
	o, err := httputil.DumpRequestOut(httpReq, true)
	fmt.Println(string(o))
	if err != nil {
		return fmt.Errorf("failed to dump request: %v", err)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	o, err = httputil.DumpResponse(resp, true)
	if err != nil {
		return fmt.Errorf("failed to dump response: %v", err)
	}
	fmt.Println(string(o))

	// save cached response
	err = cache.Save(req, resp)
	if err != nil {
		return fmt.Errorf("saving response: %v", err)
	}
	return nil
}
