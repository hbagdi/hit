package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fatih/color"
	"github.com/hbagdi/hit/pkg/model"
	"github.com/hokaccha/go-prettyjson"
)

func printRequest(r model.Request) error {
	path := r.Path
	if r.QueryString != "" {
		q, err := url.QueryUnescape(r.QueryString)
		if err != nil {
			return fmt.Errorf("unescape query params: %v", err)
		}
		path += "&" + q
	}

	fmt.Println(r.Method + " " + path + " " + r.Proto)
	printHeaders(r.Header)
	fmt.Println()

	if err := printBody(r.Body); err != nil {
		return err
	}
	fmt.Println()
	return nil
}

var (
	cyan    = color.New(color.FgCyan)
	white   = color.New(color.FgWhite)
	red     = color.New(color.FgRed)
	green   = color.New(color.FgGreen)
	yellow  = color.New(color.FgYellow)
	magenta = color.New(color.FgMagenta)
)

func printResponse(resp model.Response) error {
	fmt.Printf("%s ", resp.Proto)
	code := resp.Code
	var fn func(format string, a ...interface{}) (int, error)
	switch {
	case code < 300: //nolint:gomnd
		fn = green.Printf
	case code < 400: //nolint:gomnd
		fn = yellow.Printf
	case code < 500: //nolint:gomnd
		fn = magenta.Printf
	case code < 600: //nolint:gomnd
		fn = red.Printf
	default:
		fn = white.Printf
	}
	if _, err := fn("%s\n", resp.Status); err != nil {
		return err
	}

	printHeaders(resp.Header)

	return printBody(resp.Body)
}

func isJSON(b []byte) bool {
	var r interface{}
	err := json.Unmarshal(b, &r)
	return err == nil
}

func printBody(body []byte) error {
	if isJSON(body) {
		js, err := prettyjson.Format(body)
		if err != nil {
			return err
		}
		fmt.Println(string(js))
	} else {
		white.Printf(string(body))
	}
	return nil
}

func printHeaders(header http.Header) {
	for k, values := range header {
		for _, v := range values {
			cyan.Printf("%s", k)
			fmt.Printf(": ")
			white.Printf("%s\n", v)
		}
	}
}
