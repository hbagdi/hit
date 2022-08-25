package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	client2 "github.com/hbagdi/hit/pkg/client"
	"github.com/hbagdi/hit/pkg/db"
	"github.com/hbagdi/hit/pkg/log"
	"github.com/hbagdi/hit/pkg/model"
	"github.com/hbagdi/hit/pkg/printer"
	"github.com/hbagdi/hit/pkg/version"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
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

	p := printer.NewPrinter(printer.Opts{
		Mode:   printer.ModeBrowser,
		Writer: hitTextArea,
	})
	_ = p.Print(hit) // TODO(hbagdi): error handling
	hitTextArea.ScrollToBeginning()
}

func (b *browser) keyHandler(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune { //nolint:nestif
		r := event.Rune()
		if r == 'q' {
			b.app.Stop()
		}
		if r == 's' {
			b.shareModal()
		}
		if r == 'w' {
			p := b.app.GetFocus()
			if p == b.hitListView {
				b.app.SetFocus(b.hitTextArea)
			}
			if p == b.hitTextArea {
				b.app.SetFocus(b.hitListView)
			}
		}
	}
	return event
}

var hitAppClient *client2.HitClient

func init() {
	var err error
	hitAppClient, err = client2.NewHitClient(client2.HitClientOpts{
		Logger:        log.Logger,
		HitCLIVersion: version.Version,
	})
	if err != nil {
		panic(err)
	}
}

func (b *browser) shareModal() {
	// i := hitsList.GetCurrentItem()
	const shareModalPageName = "share-modal"
	const sharedModalPageName = "shared-modal"
	shareModal := newModal()
	shareModal.SetText("Share request with others," +
		"this will upload the request to hit-app.yolo42.com")
	shareModal.AddButtons([]string{"share", "back"})
	shareModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case -1:
			b.pages.RemovePage(shareModalPageName)
		case 1:
			b.pages.RemovePage(shareModalPageName)
		case 0:
			b.pages.RemovePage(shareModalPageName)
			modal := newModal()
			modal.SetText("Uploading...")
			b.pages.AddPage(sharedModalPageName, modal, true, true)

			go func() {
				currentRequest := b.hitListView.GetCurrentItem()
				hit := b.hits[currentRequest]
				encodedRequestBody := base64.StdEncoding.EncodeToString(hit.Request.Body)
				encodedResponseBody := base64.StdEncoding.EncodeToString(hit.Response.Body)
				data := client2.ShareData{
					Request: client2.ShareRequest{
						Proto:       hit.Request.Proto,
						Scheme:      hit.Request.Scheme,
						Method:      hit.Request.Method,
						Host:        hit.Request.Host,
						Path:        hit.Request.Path,
						QueryString: hit.Request.QueryString,
						Header:      hit.Request.Header,
						Body:        encodedRequestBody,
					},
					Response: client2.ShareResponse{
						Proto:  hit.Response.Proto,
						Code:   hit.Response.Code,
						Status: hit.Response.Status,
						Header: hit.Response.Header,
						Body:   encodedResponseBody,
					},
				}
				var (
					url           string
					responseModal = newModal()
					resp, err     = doShare(data)
				)
				if err != nil {
					message := fmt.Sprintf("failed to upload request: %v", err)
					responseModal.SetText(message)
				} else {
					url = shareURL(resp)
					responseModal.SetText(fmt.Sprintf("Upload successful:\n%v", url))
					responseModal.AddButtons([]string{"open", "copy"})
				}

				responseModal.AddButtons([]string{"back"})
				responseModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonLabel {
					case "back":
						b.pages.RemovePage(sharedModalPageName)
					case "open":
						err := open.Run(url)
						if err != nil {
							panic(err)
						}
					case "copy":
						err := clipboard.WriteAll(url)
						if err != nil {
							panic(err)
						}
						responseModal.SetText("Copied!")
					}
				})
				b.pages.RemovePage(shareModalPageName)
				b.pages.AddPage(sharedModalPageName, responseModal, true, true)
				b.app.Draw()
			}()
		}
	})
	b.pages.AddPage(shareModalPageName, shareModal, true, true)
}

func doShare(data client2.ShareData) (string, error) {
	const requestTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	response, err := hitAppClient.ShareHit(ctx, client2.ShareAPIRequest{
		Data: data,
	})
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

func shareURL(id string) string {
	return fmt.Sprintf("https://hit-app.yolo42.com/browse/hits/%s", id)
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
		AddText("[w[] Window focus [s[] Share request [q[] Quit", false,
			tview.AlignCenter, tcell.ColorWhite)

	mainFrame.SetTitle("hit browser").
		SetBorder(false).
		SetBorderPadding(0, 0, 0, 0).
		SetBackgroundColor(tBlack)

	b.pages.AddPage("main-page", mainFrame, true, true)
}

func (b *browser) setupHitTextArea() {
	b.hitTextArea = tview.NewTextView()
	b.hitTextArea.SetWrap(false)
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
	b.hitListView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		r := event.Rune()
		if r == 'j' {
			current := b.hitListView.GetCurrentItem()
			newItem := current + 1
			if newItem == b.hitListView.GetItemCount() {
				newItem = 0
			}
			b.hitListView.SetCurrentItem(newItem)
		}
		if r == 'k' {
			current := b.hitListView.GetCurrentItem()
			newItem := current - 1
			b.hitListView.SetCurrentItem(newItem)
		}
		return event
	})
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

func newBrowser(ctx context.Context, store *db.Store) (*browser, error) {
	hits, err := store.List(ctx, db.PageOpts{})
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

func executeBrowse(ctx context.Context) error {
	store, err := db.NewStore(ctx, db.StoreOpts{Logger: log.Logger})
	if err != nil {
		return fmt.Errorf("set up DB: %v", err)
	}
	defer func() {
		err := store.Close()
		if err != nil {
			log.Logger.Sugar().Errorf("failed to close store: %v", err)
		}
	}()
	b, err := newBrowser(ctx, store)
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
