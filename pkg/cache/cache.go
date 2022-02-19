package cache

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/hbagdi/hit/pkg/parser"
)

func Load() (map[string]interface{}, error) {
	content, err := ioutil.ReadFile(".hit.cache")
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(content, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func Save(req parser.Request, resp *http.Response) error {
	m, err := Load()
	if err != nil {
		return err
	}
	if resp.Header.Get("content-type") != "application/json" {
		return nil
	}
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var i interface{}
	err = json.Unmarshal(res, &i)
	if err != nil {
		return err
	}
	m[req.ID] = i

	f, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(".hit.cache", f, 0)
}
