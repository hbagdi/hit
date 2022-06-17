package executor

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
)

const timeout = 10 * time.Second

type Executor struct {
	files  []parser.File
	global parser.Global
	cache  cache.Cache
}

type Opts struct {
	Cache cache.Cache
}

func NewExecutor(opts *Opts) (*Executor, error) {
	e := &Executor{}
	if opts != nil {
		e.cache = opts.Cache
	}

	return e, nil
}

func (e *Executor) LoadFiles() error {
	files, err := loadFiles()
	if err != nil {
		return err
	}
	e.files = files

	global, err := fetchGlobal(e.files)
	if err != nil {
		return err
	}
	e.global = global
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

func (e *Executor) fetchRequest(id string) (parser.Request, error) {
	for _, file := range e.files {
		for _, r := range file.Requests {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return parser.Request{}, fmt.Errorf("request '%v' not found", id)
}

type RequestOpts struct {
	Params []string
}

func (e *Executor) BuildRequest(id string, opts *RequestOpts) (*Request, error) {
	parserRequest, err := e.fetchRequest(id)
	if err != nil {
		return nil, err
	}
	if opts == nil {
		opts = &RequestOpts{}
	}
	httpReq, err := request.Generate(context.Background(), parserRequest, request.Options{
		GlobalContext: e.global,
		Cache:         e.cache,
		Args:          opts.Params,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}
	return &Request{
		parserRequest: parserRequest,
		HTTPRequest:   httpReq,
	}, nil
}

func (e *Executor) Close() error {
	return nil
}

type Request struct {
	parserRequest parser.Request
	HTTPRequest   *http.Request
}

func (e *Executor) Execute(ctx context.Context, req *Request) (*http.Response, error) {
	var (
		err         error
		httpRequest = req.HTTPRequest
	)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpRequest = httpRequest.WithContext(ctx)
	resp, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("do request: %v", err)
	}

	// save cached response
	// TODO(hbagdi): does this clone the body? Doesn't seem like it

	clonedRequest := req.HTTPRequest.Clone(context.Background())
	err = e.cache.Save(*clonedRequest, resp)
	if err != nil {
		return nil, fmt.Errorf("save response: %v", err)
	}
	return resp, nil
}

func (e *Executor) AllRequestIDs() ([]string, error) {
	var requestIDs []string
	for _, f := range e.files {
		for _, r := range f.Requests {
			requestIDs = append(requestIDs, "@"+r.ID)
		}
	}
	return requestIDs, nil
}
