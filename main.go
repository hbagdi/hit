package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type File struct {
	Global   Global
	Requests []Request
}

type Global struct {
	BaseURL string
	Version int
}

type Request struct {
	ID           string
	Method       string
	Headers      map[string][]string
	Path         string
	BodyEncoding string
	Body         []string
}

func main() {
	f, err := os.Open("test.hit")
	if err != nil {
		log.Fatalln(err)
	}
	var file File
	r := bufio.NewReader(f)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "#"):
			// skip comments
			continue
		case line == "@_global":
			err := global(sc, &file.Global)
			if err != nil {
				log.Fatalln(err)
			}
		case strings.HasPrefix(line, "@"):
			id := line[1:]
			req, err := request(id, sc)
			if err != nil {
				log.Fatalln(err)
			}
			file.Requests = append(file.Requests, req)
		default:
			log.Fatalln("unexpected line")
		}
	}
	args := os.Args
	if len(args) < 2 {
		log.Fatalln("need a request to execute")
	}
	id := args[1][1:]

	var req Request
	for _, r := range file.Requests {
		if r.ID == id {
			req = r
		}
	}
	if req.ID == "" {
		log.Fatalln("no such request: ", id)
	}
	// execute
	url, err := url.Parse(file.Global.BaseURL)
	if err != nil {
		log.Fatalln("invalid url", err)
	}
	path := path.Join(url.Path, req.Path)
	url.Path = path

	bodyS := strings.Join(req.Body, "\n")
	var body []byte
	if req.BodyEncoding == "y2j" {
		var i interface{}
		err := yaml.Unmarshal([]byte(bodyS), &i)
		if err != nil {
			log.Fatalln("invalid body", err)
		}
		body, err = json.Marshal(i)
		if err != nil {
			log.Fatalln("invalid body, failed to marshal to json", err)
		}
	}
	if req.BodyEncoding == "hy2j" {
		c, err := loadCache()
		if err != nil {
			log.Fatalln("failed to load cache", err)
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
			log.Fatalln("invalid body", err)
		}
		body, err = json.Marshal(i)
		if err != nil {
			log.Fatalln("invalid body, failed to marshal to json", err)
		}
	}

	httpReq, err := http.NewRequestWithContext(context.Background(),
		req.Method,
		url.String(), bytes.NewReader(body))
	if err != nil {
		log.Fatalln("failed to construct request")
	}
	httpReq.Header = req.Headers
	o, err := httputil.DumpRequestOut(httpReq, true)
	fmt.Println(string(o))
	if err != nil {
		log.Fatalln("failed to dump request", err)
	}
	if err != nil {
		log.Fatalln("failed to construct request")
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		log.Fatalln("failed to perform request", err)
	}
	defer resp.Body.Close()
	o, err = httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatalln("failed to dump response", err)
	}
	fmt.Println(string(o))
	// save cached response
	err = save(req, resp)
	if err != nil {
		log.Fatalln("saving response", err)
	}
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

func loadCache() (map[string]interface{}, error) {
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

func save(req Request, resp *http.Response) error {
	m, err := loadCache()
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

func request(id string, sc *bufio.Scanner) (Request, error) {
	var res Request
	res.ID = id
	var lines []string
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			break
		}
		lines = append(lines, line)
	}
	l := len(lines)
	if l == 0 {
		return Request{}, fmt.Errorf("no request data")
	}
	i := 0
	var err error
	res.Method, err = method(lines[i])
	if err != nil {
		return Request{}, err
	}
	i++
	if i == l {
		return Request{}, fmt.Errorf("expected at least a method and path in" + " request")
	}
	res.Path, err = getPath(lines[i])
	if i == l {
		return Request{}, err
	}
	i++
	if i == l {
		return res, nil
	}
	// headers?
	var headerLines []string
	for i < l {
		line := lines[i]
		if strings.HasPrefix(line, "~") {
			break
		}
		headerLines = append(headerLines, line)
		i++
	}
	if len(headerLines) > 0 {
		headers, err := parseHeaders(headerLines)
		if err != nil {
			return Request{}, nil
		}
		res.Headers = headers
	}
	if i == l {
		return res, nil
	}

	// has body
	res.BodyEncoding, err = bodyEncoding(lines[i])
	if err != nil {
		return Request{}, err
	}
	i++
	if i == l {
		return res, nil
	}
	// remaining body
	res.Body = lines[i:]

	return res, nil
}

func parseHeaders(lines []string) (map[string][]string, error) {
	res := map[string][]string{}
	for _, line := range lines {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid header line: '%v'", line)
		}
		res[kv[0]] = []string{kv[1]}
	}
	return res, nil
}

func bodyEncoding(s string) (string, error) {
	if !strings.HasPrefix(s, "~") {
		return "", fmt.Errorf("invalid body encoding")
	}
	return s[1:], nil
}

func getPath(s string) (string, error) {
	if !strings.HasPrefix(s, "/") {
		return "", fmt.Errorf("expected path to begin with /")
	}
	return s, nil
}

var methodRegex = regexp.MustCompile("^[a-zA-Z]+$")

func method(s string) (string, error) {
	if !methodRegex.MatchString(s) {
		return "", fmt.Errorf("invalid method: %v", s)
	}
	return strings.ToUpper(s), nil
}

func global(sc *bufio.Scanner, g *Global) error {
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			return nil
		default:
			kv := strings.Split(line, "=")
			if len(kv) != 2 {
				return fmt.Errorf("failed to parse line '%v'", line)
			}
			if kv[0] == "base_url" {
				g.BaseURL = kv[1]
			}
			if kv[0] == "version" {
				v, err := strconv.Atoi(kv[1])
				if err != nil {
					return fmt.Errorf("invalid version '%d'", kv[1])
				}
				g.Version = v
			}
		}
	}
	return nil
}
