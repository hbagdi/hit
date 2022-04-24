package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ghodss/yaml"
	cachePkg "github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/version"
)

const (
	encodingY2J  = "y2j"
	encodingHY2J = "hy2j"
)

type Options struct {
	GlobalContext parser.Global
	Cache         cachePkg.Cache
	Args          []string
}

func Generate(ctx context.Context, request parser.Request, opts Options) (*http.Request, error) {
	resolver := newCacheResolver(opts.Cache, opts.Args)

	u, err := genURL(request, opts.GlobalContext, resolver)
	if err != nil {
		return nil, err
	}

	body, err := resolveBody(request, resolver)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, request.Method,
		u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if request.Headers != nil {
		httpReq.Header = request.Headers
	}
	httpReq.Header.Add("user-agent", "hit/"+version.Version)
	return httpReq, nil
}

func genURL(request parser.Request, global parser.Global, resolver resolver) (*url.URL, error) {
	res, err := url.Parse(global.BaseURL + request.Path)
	if err != nil {
		return nil, err
	}

	res.Path, err = resolvePath(res.Path, resolver)
	if err != nil {
		return nil, err
	}

	resolvedQueryParams, err := resolveQueryParams(res.Query(), resolver)
	if err != nil {
		return nil, err
	}

	res.RawQuery = resolvedQueryParams.Encode()
	return res, err
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
			resolvedValue, err := resolver.Resolve(fragment)
			if err != nil {
				return "", err
			}
			if resolvedFragment, ok := resolvedValue.(string); ok {
				resolvedPath += "/" + resolvedFragment
			} else {
				return "", fmt.Errorf("invalid type %T for key %s",
					resolvedValue, fragment)
			}
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
				resolvedValue, err := resolver.Resolve(value)
				if err != nil {
					return nil, err
				}
				if resolvedString, ok := resolvedValue.(string); ok {
					res.Add(k, resolvedString)
				} else {
					return nil, fmt.Errorf("invalid type %T for key %s",
						resolvedValue, value)
				}
			} else {
				res.Add(k, value)
			}
		}
	}
	return res, nil
}

func resolveBody(request parser.Request, resolver resolver) ([]byte, error) {
	bodyS := strings.Join(request.Body, "\n")
	var body []byte

	switch request.BodyEncoding {
	case encodingY2J:
		var i interface{}
		err := yaml.Unmarshal([]byte(bodyS), &i)
		if err != nil {
			return nil, err
		}
		body, err = json.Marshal(i)
		if err != nil {
			return nil, err
		}
	case encodingHY2J:
		jsonBytes, err := yaml.YAMLToJSON([]byte(bodyS))
		if err != nil {
			return nil, err
		}
		r := &BodyResolver{resolver: resolver}
		body, err = r.Resolve(jsonBytes)
		if err != nil {
			return nil, err
		}
	case "":
	default:
		return nil, fmt.Errorf("invalid encoding: %v", request.BodyEncoding)
	}
	return body, nil
}
