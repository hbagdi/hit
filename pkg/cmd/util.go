package cmd

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/hbagdi/hit/pkg/parser"
)

func fetchGlobal(files []parser.File) (parser.Global, error) {
	var res parser.Global
	for _, file := range files {
		if file.Global.Version != 0 && file.Global.Version != 1 {
			return parser.Global{},
				fmt.Errorf("invalid hit file version '%v'", file.Global.Version)
		}
		if file.Global.Version == 1 {
			res.Version = 1
		}
		if res.BaseURL == "" && file.Global.BaseURL != "" {
			res.BaseURL = file.Global.BaseURL
		}
	}
	if res.Version != 1 {
		return parser.Global{}, fmt.Errorf("no global.version")
	}
	if res.BaseURL == "" {
		return parser.Global{}, fmt.Errorf("no global.base_url provided")
	}
	if _, err := url.Parse(res.BaseURL); err != nil {
		return parser.Global{},
			fmt.Errorf("invalid base_url '%v': %v", res.BaseURL, err)
	}
	return res, nil
}

func loadFiles() ([]parser.File, error) {
	filenames, err := filepath.Glob("*.hit")
	if err != nil {
		return nil, fmt.Errorf("list hit files: %v", err)
	}

	res := make([]parser.File, 0, len(filenames))
	for _, filename := range filenames {
		parsedFile, err := parser.Parse(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%v': %v", filenames, err)
		}
		res = append(res, parsedFile)
	}
	return res, nil
}

func fetchRequest(id string, files []parser.File) (parser.Request, error) {
	for _, file := range files {
		for _, r := range file.Requests {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return parser.Request{}, fmt.Errorf("not found")
}

func requestIDs() ([]string, error) {
	files, err := loadFiles()
	if err != nil {
		return nil, fmt.Errorf("read hit files: %v", err)
	}
	var requestIDs []string
	for _, f := range files {
		for _, r := range f.Requests {
			requestIDs = append(requestIDs, "@"+r.ID)
		}
	}
	return requestIDs, nil
}
