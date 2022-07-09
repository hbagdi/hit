package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/hbagdi/hit/pkg/db"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/hbagdi/hit/pkg/model"
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
	i := hitsList.GetCurrentItem()
	hit := b.hits[i]
	fprintf(hitTextArea, "%s %s?%s\n", hit.Request.Method,
		hit.Request.Path, hit.Request.QueryString)
	for key, values := range hit.Request.Header {
		for _, value := range values {
			fprintf(hitTextArea, "%s: %s\n", key, value)
		}
	}
	fprintf(hitTextArea, "\n%s\n\n", hit.Request.Body)
	fprintf(hitTextArea, "HTTP/1.1 %s\n", hit.Response.Status)
	for key, values := range hit.Response.Header {
		for _, value := range values {
			fprintf(hitTextArea, "%s: %s\n", key, value)
		}
	}
	fprintf(hitTextArea, "\n%s\n", hit.Response.Body)
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
