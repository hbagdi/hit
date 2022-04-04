package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
	"github.com/hbagdi/hit/pkg/version"
	"github.com/hokaccha/go-prettyjson"
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
		fmt.Printf("%s (commit: %s)\n", version.Version, version.CommitHash)
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
	err = printResponse(resp)
	if err != nil {
		return err
	}

	// save cached response
	err = cache.Save(req, resp)
	if err != nil {
		return fmt.Errorf("saving response: %v", err)
	}
	return nil
}

var (
	cyan    = color.New(color.FgCyan)
	white   = color.New(color.FgWhite)
	red     = color.New(color.FgRed)
	green   = color.New(color.FgGreen)
	yellow  = color.New(color.FgYellow)
	magenta = color.New(color.FgMagenta)
)

func printResponse(resp *http.Response) error {
	fmt.Printf("%s ", resp.Proto)
	var fn func(format string, a ...interface{}) (int, error)
	switch {
	case resp.StatusCode < 300: //nolint:gomnd
		fn = green.Printf
	case resp.StatusCode < 400: //nolint:gomnd
		fn = yellow.Printf
	case resp.StatusCode < 500: //nolint:gomnd
		fn = magenta.Printf
	case resp.StatusCode < 600: //nolint:gomnd
		fn = red.Printf
	default:
		fn = white.Printf
	}
	_, err := fn("%s\n", resp.Status)
	if err != nil {
		return err
	}

	for k, values := range resp.Header {
		for _, v := range values {
			cyan.Printf("%s", k)
			fmt.Printf(": ")
			white.Printf("%s\n", v)
		}
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	if resp.Header.Get("content-type") == "application/json" {
		js, err := prettyjson.Format(body)
		if err != nil {
			return err
		}
		fmt.Println(string(js))
	} else {
		white.Printf(string(body))
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
