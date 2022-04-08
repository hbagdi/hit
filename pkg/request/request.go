package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"github.com/hbagdi/hit/pkg/version"
	"github.com/tidwall/gjson"
)

const (
	encodingY2J  = "y2j"
	encodingHY2J = "hy2j"
)

func Generate(global parser.Global,
	request parser.Request) (*http.Request, error) {
	url, err := url.Parse(global.BaseURL + request.Path)
	if err != nil {
		return nil, err
	}

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
		c, err := cache.Load()
		if err != nil {
			return nil, err
		}
		fn := func(key string) (interface{}, error) {
			key = key[1:]
			n, err := strconv.Atoi(key)
			if err == nil && n < len(os.Args) {
				v := os.Args[n]
				if v[0] != '@' {
					return []byte(v), nil
				}
				key = v[1:]
			}
			pathElements := strings.Split(key, ".")
			var r interface{} = c
			for _, element := range pathElements {
				m, ok := r.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("failed to index key: %v", key)
				}
				r, ok = m[element]
				if !ok {
					return nil, fmt.Errorf("key not found: %v", key)
				}
			}
			return r, nil
		}

		jsonBytes, err := yaml.YAMLToJSON([]byte(bodyS))
		if err != nil {
			return nil, err
		}
		r := &Resolver{Fn: fn}
		body, err = r.Resolve(jsonBytes)
		if err != nil {
			return nil, err
		}

	case "":
	default:
		return nil, fmt.Errorf("invalid encoding: %v", request.BodyEncoding)
	}

	httpReq, err := http.NewRequestWithContext(context.Background(),
		request.Method,
		url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if request.Headers != nil {
		httpReq.Header = request.Headers
	}
	httpReq.Header.Add("user-agent", "hit/"+version.Version)
	return httpReq, nil
}

type SubFn func(string) (interface{}, error)

type Resolver struct {
	Fn  SubFn
	res interface{}
	err error
}

func (r *Resolver) Resolve(input []byte) ([]byte, error) {
	g := gjson.ParseBytes(input)
	r.res, r.err = r.deRefJSON(g)
	if r.err != nil {
		return nil, r.err
	}
	return json.Marshal(r.res)
}

func (r *Resolver) deRefJSON(j gjson.Result) (interface{}, error) {
	if j.IsArray() {
		var res []interface{}
		var iteratorErr error
		j.ForEach(func(key, value gjson.Result) bool {
			r, err := r.deRefJSON(value)
			if err != nil {
				iteratorErr = err
				return false
			}
			res = append(res, r)
			return true
		})
		if iteratorErr != nil {
			return nil, iteratorErr
		}
		return res, nil
	}
	if j.IsObject() {
		res := map[string]interface{}{}
		var iteratorErr error
		j.ForEach(func(key, value gjson.Result) bool {
			r, err := r.deRefJSON(value)
			if err != nil {
				iteratorErr = err
				return false
			}
			res[key.String()] = r
			return true
		})
		if iteratorErr != nil {
			return nil, iteratorErr
		}
		return res, nil
	}
	if j.IsBool() {
		return j.Value(), nil
	}
	switch j.Type {
	case gjson.String:
		v := j.String()
		if v[0] != '@' {
			return v, nil
		}
		return r.Fn(v)
	case gjson.Number:
		fallthrough
	case gjson.Null:
		return j.Value(), nil
	case gjson.JSON:
		fallthrough
	case gjson.True:
		fallthrough
	case gjson.False:
		fallthrough
	default:
		panic(fmt.Sprintf("unhandled type: %v", j.Type.String()))
	}
}
