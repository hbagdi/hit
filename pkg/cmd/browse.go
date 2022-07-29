package cmd

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gdamore/tcell/v2"
	"github.com/hbagdi/hit/pkg/db"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/hbagdi/hit/pkg/model"
	json "github.com/nwidger/jsoncolor"
	"github.com/rivo/tview"
)

var (
	tBlack = tcell.NewRGBColor(0, 0, 0)
	tGreen = tcell.NewRGBColor(0, 154, 23) //nolint:gomnd
)

type browser struct {
	app         *tview.Application
	hitTextArea *tview.TextView
	hitListView *tview.List
	hits        []model.Hit
	pages       *tview.Pages
}

func (b *browser) Run() error {
	return b.app.Run()
}

func (b *browser) listHandler() {
	hitTextArea := b.hitTextArea
	hitsList := b.hitListView
	hitTextArea.Clear()
	hitTextArea.SetDynamicColors(true)
	i := hitsList.GetCurrentItem()
	hit := b.hits[i]

	requestLine := fmt.Sprintf("%s %s", hit.Request.Method, hit.Request.Path)
	if hit.Request.QueryString != "" {
		requestLine += fmt.Sprintf("?%s", hit.Request.QueryString)
	}
	requestLine += "\n"
	fprintf(hitTextArea, "%s", requestLine)

	prettyPrintHeaders(hitTextArea, hit.Request.Header)

	jsBody, err := prettyPrint(hit.Request.Body)
	if err != nil {
		fmt.Println("request body", err)
	}
	fprintf(hitTextArea, "\n%s\n\n", jsBody)

	fprintf(hitTextArea, "HTTP/1.1 %s\n", hit.Response.Status)
	prettyPrintHeaders(hitTextArea, hit.Response.Header)

	js, err := prettyPrint(hit.Response.Body)
	if err != nil {
		fmt.Println("response body", err)
	}
	fprintf(hitTextArea, "\n%s\n", string(js))
}

type fi struct {
	color string
}

func (f fi) SprintfFunc() func(format string, a ...interface{}) string {
	return func(format string, a ...interface{}) string {
		return fmt.Sprintf("["+f.color+"]"+format+"[-:-:-]", a...)
	}
}

var formatter *json.Formatter

func init() {
	// create custom formatter
	f := json.NewFormatter()
	// set custom colors
	white := fi{color: "white"}
	blue := fi{color: "blue"}
	yellow := fi{color: "yellow"}
	green := fi{color: "green"}

	f.ObjectColor = white
	f.ArrayColor = white
	f.FieldQuoteColor = white
	f.CommaColor = white
	f.StringQuoteColor = white
	f.ColonColor = white
	f.SpaceColor = white

	f.FieldColor = blue

	f.NullColor = fi{color: "#656565"}

	f.StringColor = green

	f.TrueColor = yellow
	f.FalseColor = yellow

	f.NumberColor = blue
	formatter = f
}

func prettyPrintHeaders(w io.Writer, header http.Header) {
	for key, values := range header {
		for _, value := range values {
			fprintf(w, "[teal]%s[white]: %s\n", key, value)
		}
	}
}

func prettyPrint(js []byte) ([]byte, error) {
	if len(js) == 0 {
		return js, nil
	}

	var m interface{}
	err := stdjson.Unmarshal(js, &m)
	if err != nil {
		// probably not a valid json
		// TODO(hbagdi): could there be other failure modes?
		return js, nil
	}

	dst, err := json.MarshalIndentWithFormatter(m, "", "  ", formatter)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (b *browser) keyHandler(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune {
		r := event.Rune()
		if r == 'q' {
			b.app.Stop()
		}
		if r == 's' {
			b.shareModal()
		}
	}
	return event
}

func (b *browser) shareModal() {
	// i := hitsList.GetCurrentItem()
	const shareModalPageName = "share-modal"
	const sharedModalPageName = "shared-modal"
	modal := newModal()
	modal.SetText("Share request with others," +
		"this will upload the request to hit.yolo42.com")
	modal.AddButtons([]string{"share", "back"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case -1:
			b.pages.RemovePage(shareModalPageName)
		case 1:
			b.pages.RemovePage(shareModalPageName)
		case 0:
			b.pages.RemovePage(shareModalPageName)
			modal := newModal()
			modal.SetText("Sharing not yet implemented!")
			modal.AddButtons([]string{"back"})
			modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				b.pages.RemovePage(sharedModalPageName)
			})
			b.pages.AddPage(sharedModalPageName, modal, true, true)
		}
	})
	b.pages.AddPage(shareModalPageName, modal, true, true)
}

func newModal() *tview.Modal {
	modal := tview.NewModal()
	modal.SetBackgroundColor(tGreen)
	return modal
}

func (b *browser) setupMainPage() {
	sidebarFrame := tview.NewFrame(b.hitListView)
	sidebarFrame.SetBackgroundColor(tBlack)
	sidebarFrame.AddText("Requests", true, tview.AlignCenter, tcell.ColorAntiqueWhite)
	sidebarFrame.SetBorders(0, 0, 0, 0, 0, 0)

	mainFlexbox := tview.NewFlex()
	mainFlexbox.SetBackgroundColor(tBlack)
	mainFlexbox.AddItem(sidebarFrame, 0, 1, true).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(b.hitTextArea, 0, 4, false), 0, 4, false) //nolint:gomnd

	mainFlexbox.SetInputCapture(b.keyHandler)

	mainFrame := tview.NewFrame(mainFlexbox)
	mainFrame.AddText("[::b]hit browser[::-]", true, tview.AlignCenter,
		tcell.ColorWhite).
		SetBorders(0, 0, 0, 0, 0, 0).
		AddText("[s[] Share request [q[] Quit", false, tview.AlignCenter, tcell.ColorWhite)

	mainFrame.SetTitle("hit browser").
		SetBorder(false).
		SetBorderPadding(0, 0, 0, 0).
		SetBackgroundColor(tBlack)

	b.pages.AddPage("main-page", mainFrame, true, true)
}

func (b *browser) setupHitTextArea() {
	b.hitTextArea = tview.NewTextView()
	b.hitTextArea.
		SetBorderPadding(1, 1, 1, 1).
		SetBorder(true).
		SetBackgroundColor(tBlack)
}

func (b *browser) setupApp() {
	b.app = tview.NewApplication().
		SetRoot(b.pages, true).
		EnableMouse(true)
}

func (b *browser) setupPages() {
	b.pages = tview.NewPages()
}

func (b *browser) setupHitsList() {
	b.hitListView = tview.NewList()
	b.hitListView.
		SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
			b.listHandler()
		}).
		SetFocusFunc(func() {
			b.listHandler()
		})
	b.hitListView.ShowSecondaryText(false).
		SetBorder(false).
		SetBackgroundColor(tBlack)
	b.refreshListView()
}

func (b *browser) refreshListView() {
	b.hitListView.Clear()
	for i := 0; i < len(b.hits); i++ {
		hit := b.hits[i]
		title := fmt.Sprintf("[%s][%d][-] %s %s", colorForCode(hit.Response.Code), hit.Response.Code,
			hit.Request.Method, hit.Request.Path)
		b.hitListView.AddItem(title, "", 0, func() {
		})
	}
}

func newBrowser(store *db.Store) (*browser, error) {
	hits, err := store.List(context.Background(), db.PageOpts{})
	if err != nil {
		return nil, fmt.Errorf("hitsList requests: %v", err)
	}

	b := &browser{
		hits: hits,
	}

	b.setupHitTextArea()
	b.setupHitsList()
	b.setupPages()
	b.setupApp()
	b.setupMainPage()

	return b, nil
}

func executeBrowse() error {
	store, err := db.NewStore(db.StoreOpts{Logger: log.Logger})
	if err != nil {
		return fmt.Errorf("set up DB: %v", err)
	}
	defer func() {
		err := store.Close()
		if err != nil {
			log.Logger.Sugar().Errorf("failed to close store: %v", err)
		}
	}()
	b, err := newBrowser(store)
	if err != nil {
		return fmt.Errorf("set up browser :%v", err)
	}

	if err := b.Run(); err != nil {
		return fmt.Errorf("run browser: %v", err)
	}
	return nil
}

//nolint:gomnd
func colorForCode(code int) string {
	switch {
	case code < 200:
		return "white"
	case code < 300:
		return "green"
	case code < 400:
		return "yellow"
	case code < 500:
		return "yellow"
	case code < 600:
		return "red"
	default:
		return "white"
	}
}

func fprintf(w io.Writer, format string, a ...any) {
	_, err := fmt.Fprintf(w, format, a...)
	if err != nil {
		panic(fmt.Sprintf("fmt.Fprintf failed: %v", err))
	}
}
