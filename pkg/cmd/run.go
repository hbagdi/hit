package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
)

var (
	Version    = "dev"
	CommitHash = "dev"
)

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

	switch {
	case id == "version":
		fmt.Printf("%s (commit: %s)\n", Version, CommitHash)
		return nil
	case id[0] == '@':
	default:
		return fmt.Errorf("request must begin with '@' character")
	}
	id = id[1:]

	files, err := loadFiles()
	if err != nil {
		return fmt.Errorf("read hit files: %v", err)
	}
	req, err := fetchRequest(id, files)
	if err != nil {
		return fmt.Errorf("request '@%s' not found", id)
	}
	global, err := fetchGlobal(files)
	if err != nil {
		return err
	}
	httpReq, err := request.Generate(global, req)
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

func fetchGlobal(files []parser.File) (parser.Global, error) {
	var res parser.Global
	for _, file := range files {
		if file.Global.Version != 0 && file.Global.Version != 1 {
			return parser.Global{},
				fmt.Errorf("invalid hit file version '%v'", file.Global.Version)
		}
		if file.Global.Version == 1 {
			res.Version = 1
		}
		if res.BaseURL == "" && file.Global.BaseURL != "" {
			res.BaseURL = file.Global.BaseURL
		}
	}
	if res.Version != 1 {
		return parser.Global{}, fmt.Errorf("no global.version")
	}
	if res.BaseURL == "" {
		return parser.Global{}, fmt.Errorf("no global.base_url provided")
	}
	if _, err := url.Parse(res.BaseURL); err != nil {
		return parser.Global{},
			fmt.Errorf("invalid base_url '%v': %v", res.BaseURL, err)
	}
	return res, nil
}

func loadFiles() ([]parser.File, error) {
	filenames, err := filepath.Glob("*.hit")
	if err != nil {
		return nil, fmt.Errorf("list hit files: %v", err)
	}

	res := make([]parser.File, 0, len(filenames))
	for _, filename := range filenames {
		parsedFile, err := parser.Parse(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%v': %v", filenames, err)
		}
		res = append(res, parsedFile)
	}
	return res, nil
}

func fetchRequest(id string, files []parser.File) (parser.Request, error) {
	for _, file := range files {
		for _, r := range file.Requests {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return parser.Request{}, fmt.Errorf("not found")
}
