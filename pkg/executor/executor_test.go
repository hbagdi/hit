package executor

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/hbagdi/hit/pkg/model"
	"github.com/hbagdi/hit/pkg/parser"
)

func Test_getHit(t *testing.T) {
	type args struct {
		parserRequest parser.Request
		httpRequest   *http.Request
		httpResponse  *http.Response
	}
	tests := []struct {
		name    string
		args    args
		want    model.Hit
		wantErr bool
	}{
		{
			name: "a get request and a response",
			args: args{
				parserRequest: parser.Request{
					ID: "foo",
				},
				httpRequest: &http.Request{
					Method: http.MethodGet,
					URL:    mustParse("http://foo.com/bar/baz"),
					Header: map[string][]string{
						"foo-headr": {"foo-value"},
					},
				},
				httpResponse: &http.Response{
					StatusCode: 201,
					Header: map[string][]string{
						"resp-foo": {"resp-foo-value"},
					},
				},
			},
			want: model.Hit{
				ID:           0,
				HitRequestID: "foo",
				CreatedAt:    0,
				Request: model.Request{
					Method:      "GET",
					Host:        "foo.com",
					Path:        "/bar/baz",
					QueryString: "",
					Header: map[string][]string{
						"foo-headr": {"foo-value"},
					},
				},
				Response: model.Response{
					Code: 201,
					Header: map[string][]string{
						"resp-foo": {"resp-foo-value"},
					},
					Body: nil,
				},
				Latency: model.Latency{},
				Network: model.Network{},
			},
			wantErr: false,
		},
		{
			name: "with bodies",
			args: args{
				parserRequest: parser.Request{
					ID: "foo",
				},
				httpRequest: &http.Request{
					Method: http.MethodGet,
					URL:    mustParse("http://foo.com/bar/baz"),
					Header: map[string][]string{
						"foo-headr": {"foo-value"},
					},
					Body: readCloserFromString("foobar"),
				},
				httpResponse: &http.Response{
					StatusCode: 201,
					Header: map[string][]string{
						"resp-foo": {"resp-foo-value"},
					},
					Body: readCloserFromString(`{"s":"foobar"}`),
				},
			},
			want: model.Hit{
				ID:           0,
				HitRequestID: "foo",
				CreatedAt:    0,
				Request: model.Request{
					Method:      "GET",
					Host:        "foo.com",
					Path:        "/bar/baz",
					QueryString: "",
					Header: map[string][]string{
						"foo-headr": {"foo-value"},
					},
					Body: []byte("foobar"),
				},
				Response: model.Response{
					Code: 201,
					Header: map[string][]string{
						"resp-foo": {"resp-foo-value"},
					},
					Body: []byte(`{"s":"foobar"}`),
				},
				Latency: model.Latency{},
				Network: model.Network{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getHit(tt.args.parserRequest, tt.args.httpRequest, tt.args.httpResponse)
			if (err != nil) != tt.wantErr {
				t.Errorf("getHit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHit() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustParse(u string) *url.URL {
	res, err := url.Parse(u)
	if err != nil {
		panic(fmt.Sprintf("parse url '%v': %v", u, err))
	}
	return res
}

func readCloserFromString(s string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader([]byte(s)))
}
