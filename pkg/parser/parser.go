package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var idRegex = regexp.MustCompile(`^@[a-zA-Z][a-z-A-Z0-9-_]+$`)

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

func Parse(filename string) (File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return File{}, err
	}
	defer f.Close()

	var res File
	r := bufio.NewReader(f)
	sc := &scanner{bufio.NewScanner(r)}
	for {
		scanned, line := sc.Line()
		if !scanned {
			break
		}

		switch {
		case line == "":
			continue
		case line == "@_global":
			err := global(sc, &res.Global)
			if err != nil {
				return File{}, err
			}
		case strings.HasPrefix(line, "@"):
			if !idRegex.MatchString(line) {
				return File{}, fmt.Errorf("invalid id: '%v'", line)
			}
			id := line[1:]
			req, err := request(id, sc)
			if err != nil {
				return File{}, err
			}
			res.Requests = append(res.Requests, req)
		default:
			return File{}, err
		}
	}
	return res, nil
}

func request(id string, sc *scanner) (Request, error) {
	var res Request
	res.ID = id
	var lines []string
	for {
		scanned, line := sc.Line()
		if !scanned {
			break
		}
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
	res.Method, res.Path, err = getMethodAndPath(lines[i])
	if err != nil {
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
			return Request{}, err
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

const kvSplitCount = 2

func parseHeaders(lines []string) (map[string][]string, error) {
	res := map[string][]string{}
	for _, line := range lines {
		kv := strings.SplitN(line, ":", kvSplitCount)
		if len(kv) != kvSplitCount {
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

var requestLineRegex = regexp.MustCompile(`^([a-zA-Z]+) (\/.*)$`)

func getMethodAndPath(s string) (string, string, error) {
	matches := requestLineRegex.FindStringSubmatch(s)
	if len(matches) != 3 { //nolint:gomnd
		return "", "", fmt.Errorf("invalid request line")
	}
	return matches[1], matches[2], nil
}

func global(sc *scanner, g *Global) error {
	for {
		scanned, line := sc.Line()
		if !scanned {
			break
		}
		if line == "" {
			return nil
		}
		kv := strings.Split(line, "=")
		if len(kv) != kvSplitCount {
			return fmt.Errorf("failed to parse line '%v'", line)
		}
		if kv[0] == "base_url" {
			g.BaseURL = kv[1]
		}
		if kv[0] == "version" {
			v, err := strconv.Atoi(kv[1])
			if err != nil {
				return fmt.Errorf("invalid version '%v'", kv[1])
			}
			g.Version = v
		}
	}
	return nil
}

type scanner struct {
	sc *bufio.Scanner
}

func (s *scanner) Line() (bool, string) {
	if scanned := s.sc.Scan(); !scanned {
		return false, ""
	}
	line := s.sc.Text()
	// eat comments
	if strings.HasPrefix(line, "#") {
		return s.Line()
	}
	return true, line
}
