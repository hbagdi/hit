package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/hbagdi/hit/pkg/cache"
	"github.com/hbagdi/hit/pkg/parser"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func Generate(global parser.Global, request parser.Request) (*http.Request, error) {
	url, err := url.Parse(global.BaseURL)
	if err != nil {
		return nil, err
	}
	path := path.Join(url.Path, request.Path)
	url.Path = path

	bodyS := strings.Join(request.Body, "\n")
	var body []byte
	if request.BodyEncoding == "y2j" {
		var i interface{}
		err := yaml.Unmarshal([]byte(bodyS), &i)
		if err != nil {
			return nil, err
		}
		body, err = json.Marshal(i)
		if err != nil {
			return nil, err
		}
	}
	if request.BodyEncoding == "hy2j" {
		c, err := cache.Load()
		if err != nil {
			return nil, err
		}
		body, err = deRef(bodyS, func(key string) ([]byte, error) {
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
			return []byte(fmt.Sprintf("%v", r)), nil
		})
		var i interface{}
		err = yaml.Unmarshal(body, &i)
		if err != nil {
			return nil, err
		}
		body, err = json.Marshal(i)
		if err != nil {
			return nil, err
		}
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
	return httpReq, nil
}

func deRef(input string, m func(string) ([]byte, error)) ([]byte, error) {
	var res []byte
	j := 0
	i := 0
	for i < len(input) {
		if input[i] != '@' {
			i++
			continue
		}
		// '@' found, copy until '@'
		res = append(res, input[j:i]...)
		key := keyName(input[i:])
		r, err := m(key)
		if err != nil {
			return nil, err
		}
		res = append(res, r...)

		i = i + len(key)
		j = i
	}
	res = append(res, input[j:]...)
	return res, nil
}

var nameRE = regexp.MustCompile("^@[a-zA-Z0-9-.]+")

func keyName(input string) string {
	return string(nameRE.FindString(input))
}
