package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/mattn/go-isatty"
	tb "github.com/nsf/termbox-go"

	"git.sr.ht/~sircmpwn/aerc2/config"
	libui "git.sr.ht/~sircmpwn/aerc2/lib/ui"
	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

type fill rune

func (f fill) Draw(ctx *libui.Context) {
	for x := 0; x < ctx.Width(); x += 1 {
		for y := 0; y < ctx.Height(); y += 1 {
			ctx.SetCell(x, y, rune(f), tb.ColorDefault, tb.ColorDefault)
		}
	}
}

func (f fill) OnInvalidate(callback func(d libui.Drawable)) {
	// no-op
}

func (f fill) Invalidate() {
	// no-op
}

func main() {
	var logOut io.Writer
	var logger *log.Logger
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		logOut = os.Stdout
	} else {
		logOut = ioutil.Discard
	}
	logger = log.New(logOut, "", log.LstdFlags)
	logger.Println("Starting up aerc")

	conf, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}

	tabs := libui.NewTabs()
	tabs.Add(fill('★'), "白い星")
	tabs.Add(fill('☆'), "empty stars")

	grid := libui.NewGrid().Rows([]libui.GridSpec{
		libui.GridSpec{libui.SIZE_EXACT, 1},
		libui.GridSpec{libui.SIZE_WEIGHT, 1},
		libui.GridSpec{libui.SIZE_EXACT, 1},
	}).Columns([]libui.GridSpec{
		libui.GridSpec{libui.SIZE_EXACT, 20},
		libui.GridSpec{libui.SIZE_WEIGHT, 1},
	})

	// TODO: move sidebar into tab content, probably
	grid.AddChild(libui.NewText("aerc").
		Strategy(libui.TEXT_CENTER).
		Color(tb.ColorBlack, tb.ColorWhite))
	// sidebar placeholder:
	grid.AddChild(libui.NewBordered(
		fill('.'), libui.BORDER_RIGHT)).At(1, 0).Span(2, 1)
	grid.AddChild(tabs.TabStrip).At(0, 1)
	grid.AddChild(tabs.TabContent).At(1, 1)
	exline := widgets.NewExLine()
	grid.AddChild(exline).At(2, 1)

	ui, err := libui.Initialize(conf, grid)
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	// TODO: this should be a stack
	ui.AddInteractive(exline)

	go (func() {
		for {
			time.Sleep(1 * time.Second)
			tabs.Select((tabs.Selected + 1) % 2)
		}
	})()

	for !ui.Exit {
		if !ui.Tick() {
			// ~60 FPS
			time.Sleep(16 * time.Millisecond)
		}
	}
}
