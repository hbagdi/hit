package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

	clonedRequest, err := cloneHTTPRequest(httpRequest)
	if err != nil {
		return nil, err
	}

	resp, err := e.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("do request: %v", err)
	}

	clonedResponse, err := cloneHTTPResponse(resp)
	if err != nil {
		return nil, err
	}

	hit, err := getHit(req.parserRequest, clonedRequest, clonedResponse)
	if err != nil {
		return nil, fmt.Errorf("render hit: %v", err)
	}
	err = e.cache.Save(hit)
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

const cloneCount = 2

func cloneHTTPResponse(resp *http.Response) (*http.Response, error) {
	bodies, err := cloneReadCloser(resp.Body, cloneCount)
	if err != nil {
		return nil, fmt.Errorf("clone response body: %v", err)
	}
	// restore the original body
	resp.Body = bodies[0]

	// clone the response

	clonedResponse := *resp
	clonedResponse.Body = bodies[1]

	return &clonedResponse, nil
}

func cloneHTTPRequest(req *http.Request) (*http.Request, error) {
	bodies, err := cloneReadCloser(req.Body, cloneCount)
	if err != nil {
		return nil, fmt.Errorf("clone request body: %v", err)
	}
	// restore the original body
	req.Body = bodies[0]

	// clone the response
	clonedRequest := req.Clone(context.Background())
	clonedRequest.Body = bodies[1]
	return clonedRequest, nil
}

func readBody(r io.ReadCloser) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func getHit(parserRequest parser.Request, httpRequest *http.Request, httpResponse *http.Response) (model.Hit, error) {
	requestBody, err := readBody(httpRequest.Body)
	if err != nil {
		return model.Hit{}, err
	}
	responseBody, err := readBody(httpResponse.Body)
	if err != nil {
		return model.Hit{}, err
	}

	qp, err := url.QueryUnescape(httpRequest.URL.RawQuery)
	if err != nil {
		return model.Hit{}, fmt.Errorf("unescape query params: %v", err)
	}
	return model.Hit{
		HitRequestID: parserRequest.ID,
		Request: model.Request{
			Method:      httpRequest.Method,
			Host:        httpRequest.URL.Hostname(),
			QueryString: qp,
			Path:        httpRequest.URL.Path,
			Header:      httpRequest.Header,
			Body:        requestBody,
		},
		Response: model.Response{
			Code:   httpResponse.StatusCode,
			Status: httpResponse.Status,
			Header: httpResponse.Header,
			Body:   responseBody,
		},
	}, nil
}

func cloneReadCloser(r io.ReadCloser, count int) ([]io.ReadCloser, error) {
	if count < 1 {
		panic("count < 1")
	}
	if r == nil {
		res := make([]io.ReadCloser, count)
		for i := 0; i < count; i++ {
			res[i] = nil
		}
	}
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	var res []io.ReadCloser
	for i := 0; i < count; i++ {
		res = append(res, ioutil.NopCloser(bytes.NewReader(content)))
	}
	return res, nil
}
