package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/fatih/color"
	"github.com/hbagdi/hit/pkg/model"
	"github.com/nwidger/jsoncolor"
)

type Printer struct {
	writer io.Writer
	mode   Mode
}

type Mode int

const (
	ModeColorConsole = iota
	ModeBrowser
	ModeNoColor
)

type Opts struct {
	Writer io.Writer
	Mode   Mode
}

func NewPrinter(opts Opts) Printer {
	return Printer{
		writer: opts.Writer,
		mode:   opts.Mode,
	}
}

func (p Printer) Print(hit model.Hit) error {
	var err error
	err = p.printRequest(hit.Request)
	if err != nil {
		return err
	}
	err = p.printResponse(hit.Response)
	if err != nil {
		return err
	}
	return nil
}

type colorPrinter interface {
	SprintfFunc() func(format string, a ...interface{}) string
}

type noColor struct {
}

func (n noColor) SprintfFunc() func(format string, a ...interface{}) string {
	return fmt.Sprintf
}

type tvColor struct {
	color string
}

func (c tvColor) SprintfFunc() func(format string, a ...interface{}) string {
	return func(format string, a ...interface{}) string {
		return fmt.Sprintf("["+c.color+"]"+format+"[-:-:-]", a...)
	}
}

type colorName int

const (
	white colorName = iota
	cyan
	yellow
	grey
	blue
	green
)

var (
	consoleColors = map[colorName]colorPrinter{}
	browserColors = map[colorName]colorPrinter{}
)

func init() {
	consoleColors[white] = color.New(color.FgWhite)
	browserColors[white] = tvColor{color: "white"}

	consoleColors[cyan] = color.New(color.FgCyan)
	browserColors[cyan] = tvColor{color: "darkcyan"}

	consoleColors[yellow] = color.New(color.FgYellow)
	browserColors[yellow] = tvColor{color: "yellow"}

	consoleColors[grey] = color.New(color.FgBlack, color.Bold)
	browserColors[grey] = tvColor{color: "#656565"}

	consoleColors[blue] = color.New(color.FgBlue)
	browserColors[blue] = tvColor{color: "blue"}

	consoleColors[green] = color.New(color.FgGreen)
	browserColors[green] = tvColor{color: "green"}
}

func (p Printer) colorPrinterFor(name colorName) colorPrinter {
	switch p.mode {
	case ModeColorConsole:
		return consoleColors[name]
	case ModeBrowser:
		return browserColors[name]
	case ModeNoColor:
		return noColor{}
	default:
		panic(fmt.Sprintf("invalid mode: %v", p.mode))
	}
}

func (p Printer) printRequest(r model.Request) error {
	path := r.Path
	if r.QueryString != "" {
		q, err := url.QueryUnescape(r.QueryString)
		if err != nil {
			return fmt.Errorf("unescape query params: %v", err)
		}
		path += "?" + q
	}

	requestLine := p.colorPrinterFor(white).SprintfFunc()("%s %s %s\n",
		r.Method, path, r.Proto)
	_, err := fmt.Fprintf(p.writer, "%s", requestLine)
	if err != nil {
		return err
	}
	p.printHeaders(r.Header)
	fmt.Fprintln(p.writer)

	if err := p.printBody(r.Body); err != nil {
		return err
	}
	fmt.Fprintln(p.writer)
	return nil
}

func (p Printer) printResponse(resp model.Response) error {
	res := p.colorPrinterFor(white).SprintfFunc()("%s %s\n", resp.Proto, resp.Status)
	fmt.Fprintf(p.writer, "%s", res)

	p.printHeaders(resp.Header)

	return p.printBody(resp.Body)
}

func isJSON(b []byte) bool {
	var r interface{}
	err := json.Unmarshal(b, &r)
	return err == nil
}

func (p Printer) printBody(body []byte) error {
	if isJSON(body) {
		js, err := p.prettyJSON(body)
		if err != nil {
			return err
		}
		fmt.Fprintln(p.writer, string(js))
	} else {
		res := p.colorPrinterFor(white).SprintfFunc()("%s", body)
		fmt.Fprintf(p.writer, "%s", res)
	}
	return nil
}

func (p Printer) prettyJSON(js []byte) ([]byte, error) {
	// create custom formatter
	formatter := p.formatter()

	if len(js) == 0 {
		return js, nil
	}

	var jsMap interface{}
	if err := json.Unmarshal(js, &jsMap); err != nil {
		return nil, err
	}

	dst, err := jsoncolor.MarshalIndentWithFormatter(jsMap, "", "  ", formatter)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (p Printer) printHeaders(header http.Header) {
	headerKeySprintf := p.colorPrinterFor(cyan).SprintfFunc()
	headerValueSprintf := p.colorPrinterFor(white).SprintfFunc()
	var res string
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		values := header[k]
		for _, v := range values {
			res += headerKeySprintf("%s", k)
			res += headerValueSprintf(": %s\n", v)
		}
	}
	fmt.Fprintf(p.writer, "%s", res)
}

func (p Printer) formatter() *jsoncolor.Formatter {
	// create custom formatter
	f := jsoncolor.NewFormatter()
	// set custom colors
	whiteP := p.colorPrinterFor(white)
	blueP := p.colorPrinterFor(blue)
	greenP := p.colorPrinterFor(green)
	greyP := p.colorPrinterFor(grey)
	yellowP := p.colorPrinterFor(yellow)

	f.ObjectColor = whiteP
	f.ArrayColor = whiteP
	f.FieldQuoteColor = whiteP
	f.CommaColor = whiteP
	f.StringQuoteColor = whiteP
	f.ColonColor = whiteP
	f.SpaceColor = whiteP

	f.FieldColor = blueP

	f.NullColor = greyP

	f.StringColor = greenP

	f.TrueColor = yellowP
	f.FalseColor = yellowP

	f.NumberColor = blueP
	return f
}
