package request

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

type BodyResolver struct {
	resolver resolver
	res      interface{}
	err      error
}

func (r *BodyResolver) Resolve(input []byte) ([]byte, error) {
	g := gjson.ParseBytes(input)
	r.res, r.err = r.deRefJSON(g)
	if r.err != nil {
		return nil, r.err
	}
	return json.Marshal(r.res)
}

func (r *BodyResolver) deRefJSON(j gjson.Result) (interface{}, error) {
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
		return r.resolver.Resolve(v)
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
