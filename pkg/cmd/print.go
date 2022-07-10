package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
)

func printRequest(r *http.Request) error {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		q, err := url.QueryUnescape(r.URL.RawQuery)
		if err != nil {
			return fmt.Errorf("unescape query params: %v", err)
		}
		path += "&" + q
	}

	fmt.Println(r.Method + " " + path + " " + r.Proto)
	printHeaders(r.Header)
	fmt.Println()
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	err = printBody(body)
	if err != nil {
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

func printResponse(resp *http.Response) error {
	fmt.Printf("%s ", resp.Proto)
	var fn func(format string, a ...interface{}) (int, error)
	switch {
	case resp.StatusCode < 300: //nolint:gomnd
		fn = green.Printf
	case resp.StatusCode < 400: //nolint:gomnd
		fn = yellow.Printf
	case resp.StatusCode < 500: //nolint:gomnd
		fn = magenta.Printf
	case resp.StatusCode < 600: //nolint:gomnd
		fn = red.Printf
	default:
		fn = white.Printf
	}
	if _, err := fn("%s\n", resp.Status); err != nil {
		return err
	}

	printHeaders(resp.Header)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return printBody(body)
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
