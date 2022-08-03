package executor

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/model"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/request"
)

const timeout = 10 * time.Second

type Executor struct {
	files      []parser.File
	global     parser.Global
	cache      cache.Cache
	httpClient *http.Client
}

type Opts struct {
	Cache cache.Cache
}

func NewExecutor(opts *Opts) (*Executor, error) {
	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	e := &Executor{
		httpClient: client,
	}
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

func validateGlobal(g parser.Global) error {
	if g.Version != 0 && g.Version != 1 {
		return fmt.Errorf("invalid hit file version '%v'", g.Version)
	}
	if g.BaseURL != "" {
		u, err := url.Parse(g.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base_url '%v': %v", g.BaseURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("invalid scheme '%v': only 'http' "+
				"or 'https' is supported", u.Scheme)
		}
	}
	return nil
}

func fetchGlobal(files []parser.File) (parser.Global, error) {
	var res parser.Global
	for _, file := range files {
		if err := validateGlobal(file.Global); err != nil {
			return parser.Global{}, err
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

func (e *Executor) BuildRequest(id string, opts *RequestOpts) (model.Request, error) {
	parserRequest, err := e.fetchRequest(id)
	if err != nil {
		return model.Request{}, err
	}
	if opts == nil {
		opts = &RequestOpts{}
	}
	request, err := request.Generate(parserRequest, request.Options{
		GlobalContext: e.global,
		Cache:         e.cache,
		Args:          opts.Params,
	})
	if err != nil {
		return model.Request{}, fmt.Errorf("failed to build request: %v", err)
	}
	return request, nil
}

func (e *Executor) Close() error {
	return nil
}

func (e *Executor) Execute(ctx context.Context, requestID string, req model.Request) (model.Hit, error) {
	var err error

	httpRequest, err := httpRequestFromHitRequest(req)
	if err != nil {
		return model.Hit{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	httpRequest = httpRequest.WithContext(ctx)

	resp, err := e.httpClient.Do(httpRequest)
	if err != nil {
		return model.Hit{}, fmt.Errorf("do request: %v", err)
	}
	updatedRequest := req
	updatedRequest.Proto = httpRequest.Proto

	hitResponse, err := hitResponseFromHitRequest(resp)
	if err != nil {
		return model.Hit{}, err
	}

	hit := model.Hit{
		HitRequestID: requestID,
		Request:      updatedRequest,
		Response:     hitResponse,
	}

	err = e.cache.Save(hit)
	if err != nil {
		return model.Hit{}, fmt.Errorf("save response: %v", err)
	}

	return hit, nil
}

func httpRequestFromHitRequest(req model.Request) (*http.Request, error) {
	body := bytes.NewReader(req.Body)

	httpRequest, err := http.NewRequest(req.Method, req.URL(), body)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}
	for key, values := range req.Header {
		if httpRequest.Header.Get(key) == "" {
			for _, value := range values {
				httpRequest.Header.Add(key, value)
			}
		}
	}
	return httpRequest, nil
}

func hitResponseFromHitRequest(resp *http.Response) (model.Response, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return model.Response{}, err
	}

	return model.Response{
		Proto:  resp.Proto,
		Code:   resp.StatusCode,
		Status: resp.Status,
		Header: resp.Header.Clone(),
		Body:   body,
	}, nil
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
