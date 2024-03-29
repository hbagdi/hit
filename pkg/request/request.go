package request

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ghodss/yaml"
	cachePkg "github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/model"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/version"
)

const (
	encodingY2J = "y2j"
)

type Options struct {
	GlobalContext parser.Global
	Cache         cachePkg.Cache
	Args          []string
}

func Generate(request parser.Request, opts Options) (model.Request, error) {
	resolver := newCacheResolver(opts.Cache, opts.Args)

	urlComponents, err := genURL(request, opts.GlobalContext, resolver)
	if err != nil {
		return model.Request{}, err
	}

	body, cType, err := resolveBody(request, resolver)
	if err != nil {
		return model.Request{}, err
	}
	headers := http.Header{}
	for key, values := range request.Headers {
		for _, value := range values {
			headers.Add(key, value)
		}
	}

	for k, v := range opts.GlobalContext.Headers {
		if headers.Get(k) == "" {
			headers.Add(k, v)
		}
	}
	if headers.Get("host") == "" {
		// TODO(hbagdi): attempt to clean host or error out if host is not
		// valid
		headers.Set("host", urlComponents.host)
	}
	if headers.Get("user-agent") == "" {
		headers.Add("user-agent", "hit/"+version.Version)
	}

	switch cType {
	case contentTypeNone:
	// no body encoding, proceed without any implicit content-type header
	case contentTypeJSON:
		headers.Set("content-type", "application/json")
	case contentTypeInvalid:
		return model.Request{}, fmt.Errorf("invalid content-type")
	}

	return model.Request{
		Method:      request.Method,
		Scheme:      urlComponents.scheme,
		Host:        urlComponents.host,
		Path:        urlComponents.path,
		QueryString: urlComponents.query,
		Header:      headers,
		Body:        body,
	}, nil
}

type urlComponents struct {
	scheme, host, path, query string
}

func genURL(request parser.Request, global parser.Global, resolver resolver) (urlComponents, error) {
	res, err := url.Parse(global.BaseURL + request.Path)
	if err != nil {
		return urlComponents{}, err
	}

	res.Path, err = resolvePath(res.Path, resolver)
	if err != nil {
		return urlComponents{}, err
	}

	resolvedQueryParams, err := resolveQueryParams(res.Query(), resolver)
	if err != nil {
		return urlComponents{}, err
	}
	return urlComponents{
		scheme: res.Scheme,
		host:   res.Host,
		path:   res.EscapedPath(),
		query:  resolvedQueryParams.Encode(),
	}, nil
}

func resolvePath(path string, resolver resolver) (string, error) {
	if !strings.Contains(path, "@") {
		return path, nil
	}
	resolvedPath := ""
	fragments := strings.Split(path, "/")
	for _, fragment := range fragments {
		if fragment == "" {
			continue
		}
		if len(fragment) > 0 && fragment[0] == '@' {
			str, err := resolveValue(fragment, resolver)
			if err != nil {
				return "", err
			}
			resolvedPath += "/" + str
		} else {
			resolvedPath += "/" + fragment
		}
	}
	return resolvedPath, nil
}

func resolveQueryParams(qp url.Values, resolver resolver) (url.Values, error) {
	res := url.Values{}
	for k, v := range qp {
		for _, value := range v {
			if len(value) > 0 && value[0] == '@' {
				str, err := resolveValue(value, resolver)
				if err != nil {
					return nil, err
				}
				res.Add(k, str)
			} else {
				res.Add(k, value)
			}
		}
	}
	return res, nil
}

func resolveValue(key string, resolver resolver) (string, error) {
	resolvedValue, err := resolver.Resolve(key)
	if err != nil {
		return "", err
	}
	str, err := getStringOrErr(resolvedValue)
	if err != nil {
		return "", err
	}
	return str, nil
}

func getStringOrErr(value interface{}) (string, error) {
	switch value.(type) {
	case int:
		return fmt.Sprintf("%v", value), nil
	case bool:
		return fmt.Sprintf("%v", value), nil
	case float64:
		return fmt.Sprintf("%v", value), nil
	case string:
		return fmt.Sprintf("%v", value), nil
	default:
		return "", fmt.Errorf("invalid type %T for key %s", value, value)
	}
}

type contentType int

const (
	contentTypeNone = iota
	contentTypeInvalid
	contentTypeJSON
)

func resolveBody(request parser.Request, resolver resolver) ([]byte, contentType, error) {
	if len(request.Body) == 0 {
		return nil, contentTypeNone, nil
	}

	parsedBody := []byte(strings.Join(request.Body, "\n"))

	switch request.BodyEncoding {
	case encodingY2J:
		jsonBytes, err := yaml.YAMLToJSON(parsedBody)
		if err != nil {
			return nil, contentTypeNone, err
		}
		r := &BodyResolver{resolver: resolver}
		preparedBody, err := r.Resolve(jsonBytes)
		if err != nil {
			return nil, contentTypeNone, err
		}
		return preparedBody, contentTypeJSON, nil
	default:
		return parsedBody, contentTypeNone, nil
	}
}
